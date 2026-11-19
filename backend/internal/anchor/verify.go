package anchor

import (
	"bytes"
	"context"
	"crypto/x509"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/sopandgo/sopandgo/backend/internal/audit"
	"github.com/sopandgo/sopandgo/backend/internal/backup"
	"github.com/sopandgo/sopandgo/backend/internal/objectstore"
)

// Status is a snapshot for the admin UI. It holds no secrets.
type Status struct {
	Enabled        bool    `json:"enabled"`
	TSAURL         string  `json:"tsa_url"`
	AuthType       string  `json:"auth_type"`
	Username       string  `json:"username"`
	SecretSet      bool    `json:"secret_set"`
	Interval       string  `json:"interval"`
	Fingerprint    string  `json:"fingerprint,omitempty"`
	Subject        string  `json:"subject,omitempty"`
	InstanceID     string  `json:"instance_id,omitempty"`
	ObjectLock     *bool   `json:"object_lock,omitempty"`
	LastAnchor     *Anchor `json:"last_anchor,omitempty"`
	Anchors        int     `json:"anchors"`
	PendingUploads int     `json:"pending_uploads"`
	LastAttemptUTC string  `json:"last_attempt_utc,omitempty"`
	LastError      string  `json:"last_error,omitempty"`
	NextRunUTC     string  `json:"next_run_utc,omitempty"`
}

// Status returns the current state.
func (s *Service) Status() (Status, error) {
	job := s.job.Status()
	s.mu.Lock()
	st := s.settings
	out := Status{
		Enabled:   st.Enabled,
		TSAURL:    st.TSAURL,
		AuthType:  st.AuthType,
		Username:  st.Username,
		SecretSet: st.Secret != "",
		Interval:  st.Interval.String(),
		LastError: job.LastError,
	}
	if st.InstanceID != "" && st.Enabled {
		out.InstanceID = st.InstanceID
	}
	if s.objectLock != nil {
		v := *s.objectLock
		out.ObjectLock = &v
	}
	s.mu.Unlock()
	if !job.LastAttempt.IsZero() {
		out.LastAttemptUTC = job.LastAttempt.Format(time.RFC3339)
	}
	if !job.NextRun.IsZero() {
		out.NextRunUTC = job.NextRun.Format(time.RFC3339)
	}

	if certs, err := ParsePEMCerts(st.PinnedCAPEM); err == nil {
		out.Fingerprint = Fingerprint(certs)
		out.Subject = certs[0].Subject.String()
	}
	last, err := s.store.Latest()
	if err != nil {
		return out, err
	}
	out.LastAnchor = last
	out.Anchors, out.PendingUploads, err = s.store.Count()
	return out, err
}

// RefreshObjectLock re-reads the bucket's Object Lock setting for Status.
func (s *Service) RefreshObjectLock(ctx context.Context) {
	if !s.snapshot().Enabled {
		return
	}
	objs, err := s.open(ctx)
	if err != nil {
		return
	}
	if lock, err := objs.ObjectLockEnabled(ctx); err == nil {
		s.setObjectLock(lock)
	}
}

// Finding statuses. Problems fail the integrity scan; explained findings are
// listed for a person to review; warnings mean the protection is weaker than
// it should be.
const (
	FindingTampered          = "tampered"
	FindingBadReceipt        = "bad_receipt"
	FindingMissingFromServer = "missing_from_server"
	FindingReceiptDiffers    = "receipt_differs"
	FindingNotUploaded       = "not_uploaded"
	FindingReplacedByRestore = "replaced_by_restore"

	SeverityProblem   = "problem"
	SeverityExplained = "explained"
	SeverityWarning   = "warning"
)

// Finding is one anchor that did not check out cleanly.
type Finding struct {
	Source   string `json:"source"` // "server" or "s3"
	EventID  int64  `json:"event_id"`
	TipHash  string `json:"tip_hash"`
	GenTime  string `json:"gen_time,omitempty"`
	S3Key    string `json:"s3_key,omitempty"`
	Status   string `json:"status"`
	Severity string `json:"severity"`
	Detail   string `json:"detail"`
}

// CheckReport is the anchoring part of the integrity scan.
type CheckReport struct {
	Enabled     bool                 `json:"enabled"`
	Anchors     int                  `json:"anchors"`
	S3Receipts  int                  `json:"s3_receipts"`
	OK          int                  `json:"ok"`
	Problems    int                  `json:"problems"`
	Explained   int                  `json:"explained"`
	Warnings    int                  `json:"warnings"`
	Findings    []Finding            `json:"findings"`
	KnownBroken []audit.ChainFailure `json:"known_broken"`
	ObjectLock  *bool                `json:"object_lock,omitempty"`
	S3Error     string               `json:"s3_error,omitempty"`
}

func (r *CheckReport) add(f Finding) {
	r.Findings = append(r.Findings, f)
	switch f.Severity {
	case SeverityProblem:
		r.Problems++
	case SeverityExplained:
		r.Explained++
	default:
		r.Warnings++
	}
}

type restoreEvent struct {
	backup.RestoreRecord
	EventID int64
}

// chainView is what the checks need to know about the current chain.
type chainView struct {
	hashes   map[int64]string // event id -> stored hash
	failures []audit.ChainFailure
	known    map[int64]string
	restores []restoreEvent
}

func (s *Service) loadChainView(chain audit.ChainIntegrityReport) (chainView, error) {
	v := chainView{hashes: map[int64]string{}}
	events, err := s.logger.ChainEvents()
	if err != nil {
		return v, err
	}
	for _, e := range events {
		v.hashes[e.ID] = e.Hash
		if e.EventType == audit.EventBackupRestored {
			var r backup.RestoreRecord
			if json.Unmarshal([]byte(e.Payload), &r) == nil {
				v.restores = append(v.restores, restoreEvent{RestoreRecord: r, EventID: e.ID})
			}
		}
	}
	v.known, err = knownBroken(s.store.DB())
	if err != nil {
		return v, err
	}
	v.failures = unexpectedFailures(chain.Failed, v.known)
	return v, nil
}

// holds reports why an anchor at (eventID, tipHash) no longer holds, or "".
func (v chainView) holds(eventID int64, tipHash string) string {
	stored, ok := v.hashes[eventID]
	if !ok {
		return fmt.Sprintf("audit row %d no longer exists", eventID)
	}
	if stored != tipHash {
		return fmt.Sprintf("audit row %d now has a different hash than the one anchored", eventID)
	}
	for _, f := range v.failures {
		if f.EventID <= eventID {
			return fmt.Sprintf("audit row %d, covered by this anchor, fails verification (%s)", f.EventID, f.Error)
		}
	}
	return ""
}

// restoreExplains returns the restore that replaced the history an anchor
// covered: the anchor points past the archive's tip, at or before the tip that
// was overwritten, and was taken before the restore.
func (v chainView) restoreExplains(eventID int64, genTime time.Time) *restoreEvent {
	for i := range v.restores {
		r := &v.restores[i]
		applied, err := time.Parse(time.RFC3339Nano, r.AppliedAt)
		if err != nil {
			continue
		}
		if eventID > r.ArchiveTipID && eventID <= r.OldTipID && !genTime.After(applied) {
			return r
		}
	}
	return nil
}

func pinsFor(current []*x509.Certificate, recorded string) []*x509.Certificate {
	out := append([]*x509.Certificate(nil), current...)
	if certs, err := ParsePEMCerts(recorded); err == nil {
		out = append(out, certs...)
	}
	return out
}

func verifyTip(raw []byte, tipHash string, pins []*x509.Certificate) (time.Time, error) {
	digest, err := hex.DecodeString(tipHash)
	if err != nil {
		return time.Time{}, errors.New("anchored hash is not hex")
	}
	ts, err := VerifyReceipt(raw, digest, pins)
	if err != nil {
		return time.Time{}, err
	}
	return ts.Time, nil
}

// Check runs the anchoring checks of the integrity scan. chain is the result
// of the full chain verification the scan already ran.
func (s *Service) Check(ctx context.Context, chain audit.ChainIntegrityReport) (CheckReport, error) {
	st := s.snapshot()
	rep := CheckReport{Enabled: st.Enabled, Findings: []Finding{}, KnownBroken: []audit.ChainFailure{}}

	anchors, err := s.store.List()
	if err != nil {
		return rep, err
	}
	if !st.Enabled && len(anchors) == 0 {
		return rep, nil
	}
	view, err := s.loadChainView(chain)
	if err != nil {
		return rep, err
	}
	for id, msg := range view.known {
		rep.KnownBroken = append(rep.KnownBroken, audit.ChainFailure{EventID: id, Error: msg})
	}
	sort.Slice(rep.KnownBroken, func(i, j int) bool { return rep.KnownBroken[i].EventID < rep.KnownBroken[j].EventID })

	current, _ := ParsePEMCerts(st.PinnedCAPEM)

	// (a) + (b): every receipt on this server.
	type key struct {
		id   int64
		hash string
	}
	onServer := map[key]Anchor{}
	for _, a := range anchors {
		rep.Anchors++
		onServer[key{a.EventID, a.TipHash}] = a
		f := Finding{Source: "server", EventID: a.EventID, TipHash: a.TipHash, GenTime: a.GenTime.UTC().Format(time.RFC3339), S3Key: a.S3Key}
		if _, err := verifyTip(a.TSR, a.TipHash, pinsFor(current, a.TSACAPEM)); err != nil {
			f.Status, f.Severity, f.Detail = FindingBadReceipt, SeverityProblem, err.Error()
			rep.add(f)
			continue
		}
		if why := view.holds(a.EventID, a.TipHash); why != "" {
			f.Status, f.Severity, f.Detail = FindingTampered, SeverityProblem, why
			rep.add(f)
			continue
		}
		if a.S3Key == "" {
			f.Status, f.Severity, f.Detail = FindingNotUploaded, SeverityWarning, "receipt is only on this server; it will be copied to S3 on the next run"
			rep.add(f)
			continue
		}
		rep.OK++
	}

	// (c): every receipt in S3, which an attacker on this server cannot
	// remove from the bucket if Object Lock is on.
	objs, err := s.open(ctx)
	if err == nil {
		var lock bool
		if lock, err = objs.ObjectLockEnabled(ctx); err == nil {
			rep.ObjectLock = &lock
			s.setObjectLock(lock)
		}
	}
	var listed []objectstore.Object
	if err == nil {
		listed, err = objs.List(ctx, receiptPrefix(objs, st.InstanceID))
	}
	if err != nil {
		rep.S3Error = err.Error()
		rep.Warnings++
		return rep, nil
	}

	for _, o := range listed {
		k := o.Key
		if !strings.HasSuffix(k, ".json") {
			continue
		}
		rep.S3Receipts++
		base := strings.TrimSuffix(k, ".json")
		f := Finding{Source: "s3", S3Key: base + ".tsr"}

		sideRaw, err := objs.Get(ctx, k)
		var side Sidecar
		if err == nil {
			err = json.Unmarshal(sideRaw, &side)
		}
		if err != nil {
			f.Status, f.Severity, f.Detail = FindingBadReceipt, SeverityProblem, "sidecar unreadable: "+err.Error()
			rep.add(f)
			continue
		}
		f.EventID, f.TipHash, f.GenTime = side.EventID, side.TipHash, side.GenTime

		tsr, err := objs.Get(ctx, base+".tsr")
		if err != nil {
			f.Status, f.Severity, f.Detail = FindingBadReceipt, SeverityProblem, "receipt unreadable: "+err.Error()
			rep.add(f)
			continue
		}

		srv, known := onServer[key{side.EventID, side.TipHash}]
		if known {
			if !bytes.Equal(srv.TSR, tsr) {
				f.Status, f.Severity, f.Detail = FindingReceiptDiffers, SeverityProblem, "the receipt in S3 differs from the one on this server"
				rep.add(f)
			}
			continue // the server copy was checked above
		}

		pins := current
		if srvPEM := anyPinFor(anchors); srvPEM != "" {
			pins = pinsFor(current, srvPEM)
		}
		genTime, err := verifyTip(tsr, side.TipHash, pins)
		if err != nil {
			f.Status, f.Severity, f.Detail = FindingBadReceipt, SeverityProblem, err.Error()
			rep.add(f)
			continue
		}
		why := view.holds(side.EventID, side.TipHash)
		if r := view.restoreExplains(side.EventID, genTime); r != nil && why != "" {
			f.Status, f.Severity = FindingReplacedByRestore, SeverityExplained
			f.Detail = fmt.Sprintf("history after audit row %d was replaced by the backup restore recorded at row %d (%s)",
				r.ArchiveTipID, r.EventID, r.AppliedAt)
			rep.add(f)
			continue
		}
		if why != "" {
			f.Status, f.Severity, f.Detail = FindingTampered, SeverityProblem, why
			rep.add(f)
			continue
		}
		// The chain still matches, but this server has no record of the receipt.
		f.Status = FindingMissingFromServer
		if r := view.restoreAfter(genTime); r != nil {
			f.Severity = SeverityExplained
			f.Detail = fmt.Sprintf("taken before the backup restore recorded at row %d; the restored database does not list it", r.EventID)
		} else {
			f.Severity = SeverityProblem
			f.Detail = "this receipt is in S3 but its record was removed from this server"
		}
		rep.add(f)
	}
	return rep, nil
}

// restoreAfter returns a restore applied after t.
func (v chainView) restoreAfter(t time.Time) *restoreEvent {
	for i := range v.restores {
		applied, err := time.Parse(time.RFC3339Nano, v.restores[i].AppliedAt)
		if err == nil && applied.After(t) {
			return &v.restores[i]
		}
	}
	return nil
}

// anyPinFor returns every CA recorded with server anchors, so S3 receipts
// taken under an earlier pin still verify.
func anyPinFor(anchors []Anchor) string {
	seen := map[string]bool{}
	var b strings.Builder
	for _, a := range anchors {
		if a.TSACAPEM != "" && !seen[a.TSACAPEM] {
			seen[a.TSACAPEM] = true
			b.WriteString(a.TSACAPEM)
		}
	}
	return b.String()
}
