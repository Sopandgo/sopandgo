package anchor

import (
	"context"
	"crypto/x509"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/sopandgo/sopandgo/backend/internal/audit"
)

var (
	// ErrNotEnabled is returned by AnchorNow when anchoring is off.
	ErrNotEnabled = errors.New("audit anchoring is off")
	// ErrInvalidSettings wraps every validation failure.
	ErrInvalidSettings = errors.New("invalid audit anchoring settings")
)

// NeedsConfirmation is returned by Save when the admin must first confirm
// what a probe found: the TSA certificate to pin and any chain rows that
// already fail verification.
type NeedsConfirmation struct {
	Probe ProbeResult
}

func (e *NeedsConfirmation) Error() string {
	return "confirm the TSA certificate fingerprint and any already-broken audit rows"
}

func invalid(format string, args ...any) error {
	return fmt.Errorf("%w: %s", ErrInvalidSettings, fmt.Sprintf(format, args...))
}

// Notice is sent to the onChange callback when anchoring is switched on, off,
// or reconfigured, so someone outside the server hears about it.
type Notice struct {
	Event   string // audit event type
	Message string
}

// Service anchors the audit chain tip with a TSA on a schedule, copies each
// receipt to S3, and runs the enable, change and disable flows.
type Service struct {
	store      *Store
	logger     *audit.Logger
	openStore  OpenObjectStore
	httpClient *http.Client

	onFailure func(ctx context.Context, err error)
	onChange  func(ctx context.Context, n Notice)

	saveMu sync.Mutex // one settings change at a time
	runMu  sync.Mutex // one anchor run at a time
	wake   chan struct{}

	mu          sync.Mutex
	settings    Settings
	nextRun     time.Time
	failures    int
	lastAttempt time.Time
	lastError   string
	objectLock  *bool
}

// NewService returns a service that is off until Load finds enabled settings.
// httpClient may be nil.
func NewService(store *Store, logger *audit.Logger, openStore OpenObjectStore, httpClient *http.Client) *Service {
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 30 * time.Second}
	}
	return &Service{store: store, logger: logger, openStore: openStore, httpClient: httpClient, wake: make(chan struct{}, 1)}
}

// SetOnFailure registers a callback for failed anchor runs. Safe before Start.
func (s *Service) SetOnFailure(fn func(ctx context.Context, err error)) { s.onFailure = fn }

// SetOnChange registers a callback for anchoring being switched on, off, or
// reconfigured. Safe before Start.
func (s *Service) SetOnChange(fn func(ctx context.Context, n Notice)) { s.onChange = fn }

// Load reads the saved settings and schedules the next run.
func (s *Service) Load() error {
	st, err := s.store.Load()
	if err != nil {
		return err
	}
	s.apply(st, time.Now().UTC().Add(st.Interval))
	return nil
}

func (s *Service) apply(st Settings, next time.Time) {
	s.mu.Lock()
	s.settings = st
	s.failures = 0
	if st.Enabled {
		s.nextRun = next
	} else {
		s.nextRun = time.Time{}
	}
	s.mu.Unlock()
	s.signal()
}

func (s *Service) signal() {
	select {
	case s.wake <- struct{}{}:
	default:
	}
}

func (s *Service) snapshot() Settings {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.settings
}

// Start runs scheduled anchors until ctx is done.
func (s *Service) Start(ctx context.Context) {
	go func() {
		for {
			s.mu.Lock()
			next := s.nextRun
			s.mu.Unlock()

			var due <-chan time.Time
			var timer *time.Timer
			if !next.IsZero() {
				timer = time.NewTimer(time.Until(next))
				due = timer.C
			}
			select {
			case <-ctx.Done():
				if timer != nil {
					timer.Stop()
				}
				return
			case <-s.wake:
				if timer != nil {
					timer.Stop()
				}
			case <-due:
				_, _ = s.run(ctx)
			}
		}
	}()
}

// AnchorNow takes an anchor immediately. It returns (nil, nil) when nothing
// was written to the audit log since the last anchor.
func (s *Service) AnchorNow(ctx context.Context) (*Anchor, error) {
	a, err := s.run(ctx)
	s.signal()
	return a, err
}

// TriggerSoon schedules a run right away, for example after a restore.
func (s *Service) TriggerSoon() {
	s.mu.Lock()
	if s.settings.Enabled {
		s.nextRun = time.Now().UTC()
	}
	s.mu.Unlock()
	s.signal()
}

func (s *Service) tsa(st Settings) *TSA {
	return &TSA{URL: st.TSAURL, AuthType: st.AuthType, Username: st.Username, Secret: st.Secret, Client: s.httpClient}
}

// run takes one anchor with the current settings.
func (s *Service) run(ctx context.Context) (*Anchor, error) {
	s.runMu.Lock()
	defer s.runMu.Unlock()

	st := s.snapshot()
	if !st.Enabled {
		return nil, ErrNotEnabled
	}
	a, err := s.anchorOnce(ctx, st)
	s.finishRun(ctx, st, a, err)
	return a, err
}

// finishRun records the outcome, schedules the next run (backing off after
// failures), and audits and reports failures.
func (s *Service) finishRun(ctx context.Context, st Settings, a *Anchor, err error) {
	now := time.Now().UTC()
	s.mu.Lock()
	s.lastAttempt = now
	if err != nil {
		s.failures++
		s.lastError = err.Error()
		backoff := time.Minute << min(s.failures-1, 10)
		if backoff > st.Interval {
			backoff = st.Interval
		}
		if s.settings.Enabled {
			s.nextRun = now.Add(backoff)
		}
	} else {
		s.failures = 0
		s.lastError = ""
		if s.settings.Enabled {
			s.nextRun = now.Add(st.Interval)
		}
	}
	s.mu.Unlock()

	if err == nil {
		return
	}
	log.Printf("audit anchoring failed: %v", err)
	if auditErr := s.logger.Log(nil, audit.EventAuditAnchorFailed, audit.EntitySystem, "audit_anchoring", nil, map[string]any{
		"error": err.Error(),
	}); auditErr != nil {
		log.Printf("audit write failed for anchoring failure: %v", auditErr)
	}
	if s.onFailure != nil {
		s.onFailure(ctx, err)
	}
}

// anchorOnce verifies the chain since the last anchor, timestamps the tip,
// stores the receipt, and copies it to S3. A receipt that could not be copied
// is kept and retried on the next run, and the run reports the upload error.
func (s *Service) anchorOnce(ctx context.Context, st Settings) (*Anchor, error) {
	pinned, err := ParsePEMCerts(st.PinnedCAPEM)
	if err != nil {
		return nil, fmt.Errorf("pinned TSA certificate: %w", err)
	}

	uploadErr := s.uploadPending(ctx, st)

	last, err := s.store.Latest()
	if err != nil {
		return nil, err
	}
	fromID, fromHash := int64(0), audit.GenesisHash
	if last != nil {
		fromID, fromHash = last.EventID, last.TipHash
	}

	report, err := s.logger.VerifySince(fromID, fromHash)
	if errors.Is(err, audit.ErrAnchorMoved) {
		return nil, fmt.Errorf("audit row %d no longer has the hash anchored at %s: the chain was changed after it was anchored",
			fromID, last.GenTime.UTC().Format(time.RFC3339))
	}
	if err != nil {
		return nil, err
	}
	known, err := knownBroken(s.store.DB())
	if err != nil {
		return nil, err
	}
	if bad := unexpectedFailures(report.Failed, known); len(bad) > 0 {
		return nil, fmt.Errorf("audit chain fails verification after the last anchor (first bad row %d: %s); not anchoring it",
			bad[0].EventID, bad[0].Error)
	}

	// Our own audit_anchored events would otherwise make every run look like
	// new activity.
	var newEvents int
	if err := s.store.DB().QueryRow(
		`SELECT COUNT(*) FROM audit_events WHERE id > ? AND id <= ? AND event_type <> ?`,
		fromID, report.TipID, audit.EventAuditAnchored,
	).Scan(&newEvents); err != nil {
		return nil, err
	}
	if report.TipID == 0 || (last != nil && newEvents == 0) {
		return nil, uploadErr
	}

	var hashVersion int
	if err := s.store.DB().QueryRow(`SELECT hash_version FROM audit_events WHERE id = ?`, report.TipID).Scan(&hashVersion); err != nil {
		return nil, err
	}
	digest, err := hex.DecodeString(report.TipHash)
	if err != nil {
		return nil, fmt.Errorf("tip hash is not hex: %w", err)
	}

	raw, ts, err := s.tsa(st).Stamp(ctx, digest)
	if err != nil {
		return nil, err
	}
	if _, err := VerifyReceipt(raw, digest, pinned); err != nil {
		return nil, fmt.Errorf("%w; if the TSA changed its CA, re-pin it under Settings → Backup", err)
	}

	a := Anchor{
		EventID:     report.TipID,
		TipHash:     report.TipHash,
		HashVersion: hashVersion,
		TSR:         raw,
		GenTime:     ts.Time,
		TSAURL:      st.TSAURL,
		TSACAPEM:    st.PinnedCAPEM,
	}
	a.ID, err = s.store.insert(a)
	if err != nil {
		return nil, err
	}

	if key, err := s.upload(ctx, st, a); err != nil {
		uploadErr = err
	} else {
		a.S3Key = key
	}

	if err := s.logger.Log(nil, audit.EventAuditAnchored, audit.EntitySystem, "audit_anchoring", nil, map[string]any{
		"event_id": a.EventID,
		"tip_hash": a.TipHash,
		"gen_time": a.GenTime.UTC().Format(time.RFC3339Nano),
		"tsa_url":  a.TSAURL,
		"s3_key":   a.S3Key,
	}); err != nil {
		log.Printf("audit write failed for anchor %d: %v", a.ID, err)
	}

	if uploadErr != nil {
		return &a, fmt.Errorf("receipt saved on this server but not copied to S3: %w", uploadErr)
	}
	return &a, nil
}

// Sidecar is the JSON written next to each .tsr in S3 and in evidence bundles.
type Sidecar struct {
	Format      string `json:"format"`
	InstanceID  string `json:"instance_id"`
	EventID     int64  `json:"event_id"`
	TipHash     string `json:"tip_hash"`
	HashVersion int    `json:"hash_version"`
	GenTime     string `json:"gen_time"`
	TSAURL      string `json:"tsa_url"`
}

const sidecarFormat = "sopandgo-anchor-v1"

func sidecarFor(instanceID string, a Anchor) Sidecar {
	return Sidecar{
		Format:      sidecarFormat,
		InstanceID:  instanceID,
		EventID:     a.EventID,
		TipHash:     a.TipHash,
		HashVersion: a.HashVersion,
		GenTime:     a.GenTime.UTC().Format(time.RFC3339Nano),
		TSAURL:      a.TSAURL,
	}
}

// receiptBaseName is the object name without extension: zero-padded event id
// so a listing sorts in chain order, plus the start of the hash.
func receiptBaseName(a Anchor) string {
	return fmt.Sprintf("%012d-%s", a.EventID, a.TipHash[:16])
}

func (s *Service) upload(ctx context.Context, st Settings, a Anchor) (string, error) {
	if s.openStore == nil {
		return "", ErrS3NotConfigured
	}
	objs, err := s.openStore(ctx)
	if err != nil {
		return "", err
	}
	base := objs.Prefix(st.InstanceID) + receiptBaseName(a)
	side, err := json.MarshalIndent(sidecarFor(st.InstanceID, a), "", "  ")
	if err != nil {
		return "", err
	}
	if err := objs.Put(ctx, base+".tsr", a.TSR, "application/timestamp-reply"); err != nil {
		return "", err
	}
	if err := objs.Put(ctx, base+".json", side, "application/json"); err != nil {
		return "", err
	}
	key := base + ".tsr"
	if err := s.store.markUploaded(a.ID, key); err != nil {
		return "", err
	}
	return key, nil
}

func (s *Service) uploadPending(ctx context.Context, st Settings) error {
	pending, err := s.store.pendingUploads()
	if err != nil {
		return err
	}
	var firstErr error
	for _, a := range pending {
		if _, err := s.upload(ctx, st, a); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	return firstErr
}

// ProbeResult is what a test request to a TSA found.
type ProbeResult struct {
	Fingerprint  string               `json:"fingerprint"`
	Subject      string               `json:"subject"`
	PinnedCAPEM  string               `json:"pinned_ca_pem"`
	ObjectLock   bool                 `json:"object_lock"`
	BrokenRows   []audit.ChainFailure `json:"broken_rows"`
	AlreadyKnown int                  `json:"already_known"`
}

// SettingsInput is the admin form. An empty Secret keeps the stored one.
// PinnedCAPEM, when set, replaces what the probe would capture (for TSAs that
// send only their own certificate). ConfirmFingerprint and AcceptBroken carry
// the admin's confirmation of a previous probe.
type SettingsInput struct {
	Enabled            bool    `json:"enabled"`
	TSAURL             string  `json:"tsa_url"`
	AuthType           string  `json:"auth_type"`
	Username           string  `json:"username"`
	Secret             string  `json:"secret"`
	Interval           string  `json:"interval"`
	PinnedCAPEM        string  `json:"pinned_ca_pem"`
	Repin              bool    `json:"repin"`
	ConfirmFingerprint string  `json:"confirm_fingerprint"`
	AcceptBroken       []int64 `json:"accept_broken"`
}

func (s *Service) settingsFromInput(in SettingsInput, cur Settings) (Settings, error) {
	st := Settings{
		Enabled:     in.Enabled,
		TSAURL:      strings.TrimSpace(in.TSAURL),
		AuthType:    strings.TrimSpace(in.AuthType),
		Username:    strings.TrimSpace(in.Username),
		Secret:      in.Secret,
		PinnedCAPEM: cur.PinnedCAPEM,
		InstanceID:  cur.InstanceID,
	}
	if st.AuthType == "" {
		st.AuthType = AuthNone
	}
	switch st.AuthType {
	case AuthNone:
		st.Username, st.Secret = "", ""
	case AuthBasic:
		if st.Username == "" {
			return Settings{}, invalid("basic authentication needs a username")
		}
	case AuthBearer:
		st.Username = ""
	default:
		return Settings{}, invalid("auth_type must be none, basic or bearer")
	}
	if st.AuthType != AuthNone && st.Secret == "" && cur.AuthType == st.AuthType {
		st.Secret = cur.Secret
	}
	if st.AuthType != AuthNone && st.Secret == "" {
		return Settings{}, invalid("enter the TSA password or token")
	}

	st.Interval = DefaultInterval
	if raw := strings.TrimSpace(in.Interval); raw != "" {
		d, err := time.ParseDuration(raw)
		if err != nil || d < MinInterval || d > MaxInterval {
			return Settings{}, invalid("interval must be between %s and %s", MinInterval, MaxInterval)
		}
		st.Interval = d
	}
	if st.Enabled || st.TSAURL != "" {
		if err := ValidateTSAURL(st.TSAURL); err != nil {
			return Settings{}, invalid("%s", err.Error())
		}
	}
	if st.InstanceID == "" {
		st.InstanceID = newInstanceID()
	}
	return st, nil
}

// Probe sends one test request to the TSA in in, works out which certificate
// to pin, checks the S3 bucket, and runs a full chain check. Nothing is saved.
func (s *Service) Probe(ctx context.Context, in SettingsInput) (ProbeResult, error) {
	cur := s.snapshot()
	in.Enabled = true
	st, err := s.settingsFromInput(in, cur)
	if err != nil {
		return ProbeResult{}, err
	}
	return s.probe(ctx, st, in.PinnedCAPEM)
}

func (s *Service) probe(ctx context.Context, st Settings, pastedPEM string) (ProbeResult, error) {
	var res ProbeResult

	if s.openStore == nil {
		return res, ErrS3NotConfigured
	}
	store, err := s.openStore(ctx)
	if err != nil {
		return res, err
	}
	lock, err := store.ObjectLockEnabled(ctx)
	if err != nil {
		return res, fmt.Errorf("check S3 bucket: %w", err)
	}
	res.ObjectLock = lock

	digest := make([]byte, 32)
	copy(digest, []byte("sopandgo anchoring probe       "))
	raw, ts, err := s.tsa(st).Stamp(ctx, digest)
	if err != nil {
		return res, err
	}
	var pinned []*x509.Certificate
	if strings.TrimSpace(pastedPEM) != "" {
		pinned, err = ParsePEMCerts(pastedPEM)
		if err != nil {
			return res, invalid("pasted CA certificate: %s", err.Error())
		}
	} else {
		pinned, err = PinFromToken(ctx, ts, s.httpClient)
		if err != nil {
			return res, err
		}
	}
	if _, err := VerifyReceipt(raw, digest, pinned); err != nil {
		return res, err
	}
	res.PinnedCAPEM = EncodePEMCerts(pinned)
	res.Fingerprint = Fingerprint(pinned)
	res.Subject = pinned[0].Subject.String()

	report, err := s.logger.VerifyEntireChain()
	if err != nil {
		return res, err
	}
	known, err := knownBroken(s.store.DB())
	if err != nil {
		return res, err
	}
	res.BrokenRows = unexpectedFailures(report.Failed, known)
	if res.BrokenRows == nil {
		res.BrokenRows = []audit.ChainFailure{}
	}
	res.AlreadyKnown = len(report.Failed) - len(res.BrokenRows)
	return res, nil
}

// Save applies the admin form. Switching on, changing the TSA, or re-pinning
// needs a confirmed probe: the first call returns *NeedsConfirmation with what
// to show, and the admin repeats it with ConfirmFingerprint and AcceptBroken.
// Switching off takes one final anchor first.
func (s *Service) Save(ctx context.Context, in SettingsInput, actorUserID *string) error {
	s.saveMu.Lock()
	defer s.saveMu.Unlock()

	cur := s.snapshot()
	st, err := s.settingsFromInput(in, cur)
	if err != nil {
		return err
	}

	if !st.Enabled {
		return s.disable(ctx, cur, st, actorUserID)
	}

	tsaChanged := !cur.Enabled || in.Repin || strings.TrimSpace(in.PinnedCAPEM) != "" ||
		st.TSAURL != cur.TSAURL || st.AuthType != cur.AuthType || st.Username != cur.Username || st.Secret != cur.Secret ||
		cur.PinnedCAPEM == ""
	var probe ProbeResult
	if tsaChanged {
		probe, err = s.probe(ctx, st, in.PinnedCAPEM)
		if err != nil {
			return err
		}
		needsChainOK := !cur.Enabled
		if in.ConfirmFingerprint != probe.Fingerprint || (needsChainOK && !sameFailureIDs(probe.BrokenRows, in.AcceptBroken)) {
			return &NeedsConfirmation{Probe: probe}
		}
		st.PinnedCAPEM = probe.PinnedCAPEM
	}

	if !cur.Enabled {
		return s.enable(ctx, st, probe, actorUserID)
	}
	return s.update(ctx, cur, st, tsaChanged, probe, actorUserID)
}

func (s *Service) enable(ctx context.Context, st Settings, probe ProbeResult, actorUserID *string) error {
	tx, err := s.store.DB().Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if err := s.store.save(tx, st); err != nil {
		return err
	}
	baseline, err := buildBaseline(tx, st.InstanceID, probe.BrokenRows)
	if err != nil {
		return err
	}
	if err := s.logger.Log(tx, audit.EventIntegrityBaseline, audit.EntitySystem, "audit_anchoring", actorUserID, baseline); err != nil {
		return err
	}
	if err := s.logger.Log(tx, audit.EventAuditAnchoringEnabled, audit.EntitySystem, "audit_anchoring", actorUserID, map[string]any{
		"tsa_url":      st.TSAURL,
		"interval":     st.Interval.String(),
		"fingerprint":  probe.Fingerprint,
		"object_lock":  probe.ObjectLock,
		"known_broken": len(probe.BrokenRows),
		"instance_id":  st.InstanceID,
	}); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return err
	}

	s.setObjectLock(probe.ObjectLock)
	s.apply(st, time.Now().UTC().Add(st.Interval))
	s.notify(ctx, Notice{Event: audit.EventAuditAnchoringEnabled, Message: fmt.Sprintf("Audit anchoring was switched on with TSA %s.", st.TSAURL)})

	// First anchor right away. A failure is reported and retried by the
	// scheduler; anchoring stays on.
	_, _ = s.run(ctx)
	return nil
}

func (s *Service) update(ctx context.Context, cur, st Settings, tsaChanged bool, probe ProbeResult, actorUserID *string) error {
	if err := s.store.save(s.store.DB(), st); err != nil {
		return err
	}
	payload := map[string]any{
		"tsa_url":  st.TSAURL,
		"interval": st.Interval.String(),
	}
	if tsaChanged {
		payload["fingerprint"] = probe.Fingerprint
		payload["previous_tsa_url"] = cur.TSAURL
		s.setObjectLock(probe.ObjectLock)
	}
	if err := s.logger.Log(nil, audit.EventAuditAnchoringUpdated, audit.EntitySystem, "audit_anchoring", actorUserID, payload); err != nil {
		return err
	}
	next := time.Now().UTC().Add(st.Interval)
	s.mu.Lock()
	if cur.Interval == st.Interval && !s.nextRun.IsZero() {
		next = s.nextRun
	}
	s.mu.Unlock()
	s.apply(st, next)
	s.notify(ctx, Notice{Event: audit.EventAuditAnchoringUpdated, Message: fmt.Sprintf("Audit anchoring settings were changed (TSA %s, every %s).", st.TSAURL, st.Interval)})
	return nil
}

func (s *Service) disable(ctx context.Context, cur, st Settings, actorUserID *string) error {
	finalNote := "no final anchor was needed"
	if cur.Enabled {
		// Final anchor: the last receipt in S3 proves the history up to the
		// moment anchoring was switched off.
		a, err := s.run(ctx)
		switch {
		case err != nil:
			finalNote = "the final anchor failed: " + err.Error()
		case a != nil:
			finalNote = fmt.Sprintf("final anchor at audit row %d", a.EventID)
		}
	}
	if err := s.store.save(s.store.DB(), st); err != nil {
		return err
	}
	if cur.Enabled {
		if err := s.logger.Log(nil, audit.EventAuditAnchoringDisabled, audit.EntitySystem, "audit_anchoring", actorUserID, map[string]any{
			"final_anchor": finalNote,
		}); err != nil {
			return err
		}
		s.notify(ctx, Notice{Event: audit.EventAuditAnchoringDisabled, Message: "Audit anchoring was switched off (" + finalNote + ")."})
	} else if err := s.logger.Log(nil, audit.EventAuditAnchoringUpdated, audit.EntitySystem, "audit_anchoring", actorUserID, map[string]any{
		"enabled": false,
		"tsa_url": st.TSAURL,
	}); err != nil {
		return err
	}
	s.apply(st, time.Time{})
	return nil
}

func (s *Service) notify(ctx context.Context, n Notice) {
	if s.onChange != nil {
		s.onChange(ctx, n)
	}
}

func (s *Service) setObjectLock(v bool) {
	s.mu.Lock()
	s.objectLock = &v
	s.mu.Unlock()
}
