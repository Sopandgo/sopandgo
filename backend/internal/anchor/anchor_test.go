package anchor

import (
	"archive/zip"
	"bytes"
	"context"
	"crypto"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/asn1"
	"errors"
	"io"
	"math/big"
	"net/http"
	"net/http/httptest"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/digitorus/timestamp"

	"github.com/sopandgo/sopandgo/backend/internal/audit"
	"github.com/sopandgo/sopandgo/backend/internal/backup"
	"github.com/sopandgo/sopandgo/backend/internal/storage"
)

// --- test TSA ---------------------------------------------------------------

type testCA struct {
	cert    *x509.Certificate
	leaf    *x509.Certificate
	leafKey *ecdsa.PrivateKey
}

func newTestCA(t *testing.T, name, aiaURL string) *testCA {
	t.Helper()
	caKey, _ := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	caTmpl := &x509.Certificate{
		SerialNumber:          big.NewInt(1),
		Subject:               pkix.Name{CommonName: name + " Root"},
		NotBefore:             time.Now().Add(-time.Hour),
		NotAfter:              time.Now().Add(24 * time.Hour),
		IsCA:                  true,
		BasicConstraintsValid: true,
		KeyUsage:              x509.KeyUsageCertSign,
	}
	caDER, err := x509.CreateCertificate(rand.Reader, caTmpl, caTmpl, &caKey.PublicKey, caKey)
	if err != nil {
		t.Fatal(err)
	}
	ca, _ := x509.ParseCertificate(caDER)

	leafKey, _ := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	// RFC 3161: a critical extended key usage of time-stamping only. Go's
	// ExtKeyUsage field would write it non-critical.
	eku, _ := asn1.Marshal([]asn1.ObjectIdentifier{{1, 3, 6, 1, 5, 5, 7, 3, 8}})
	leafTmpl := &x509.Certificate{
		SerialNumber:          big.NewInt(2),
		Subject:               pkix.Name{CommonName: name + " TSA"},
		NotBefore:             time.Now().Add(-time.Hour),
		NotAfter:              time.Now().Add(24 * time.Hour),
		KeyUsage:              x509.KeyUsageDigitalSignature,
		ExtraExtensions:       []pkix.Extension{{Id: oidExtKeyUsage, Critical: true, Value: eku}},
		IssuingCertificateURL: []string{aiaURL},
	}
	leafDER, err := x509.CreateCertificate(rand.Reader, leafTmpl, ca, &leafKey.PublicKey, caKey)
	if err != nil {
		t.Fatal(err)
	}
	leaf, _ := x509.ParseCertificate(leafDER)
	return &testCA{cert: ca, leaf: leaf, leafKey: leafKey}
}

type testTSA struct {
	srv   *httptest.Server
	mu    sync.Mutex
	ca    *testCA
	fail  atomic.Bool
	calls atomic.Int32
}

func newTestTSA(t *testing.T) *testTSA {
	t.Helper()
	tsa := &testTSA{}
	mux := http.NewServeMux()
	mux.HandleFunc("/ca.crt", func(w http.ResponseWriter, r *http.Request) {
		tsa.mu.Lock()
		defer tsa.mu.Unlock()
		_, _ = w.Write(tsa.ca.cert.Raw)
	})
	mux.HandleFunc("/tsa", func(w http.ResponseWriter, r *http.Request) {
		tsa.calls.Add(1)
		if tsa.fail.Load() {
			http.Error(w, "down", http.StatusServiceUnavailable)
			return
		}
		body, _ := io.ReadAll(r.Body)
		req, err := timestamp.ParseRequest(body)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		tsa.mu.Lock()
		ca := tsa.ca
		tsa.mu.Unlock()
		ts := timestamp.Timestamp{
			HashAlgorithm: req.HashAlgorithm,
			HashedMessage: req.HashedMessage,
			Time:          time.Now().UTC(),
			Nonce:         req.Nonce,
			Policy:        asn1.ObjectIdentifier{1, 2, 3, 4, 1},
			// Real TSAs include their certificate when the request sets certReq.
			AddTSACertificate: req.Certificates,
		}
		resp, err := ts.CreateResponseWithOpts(ca.leaf, ca.leafKey, crypto.SHA256)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/timestamp-reply")
		_, _ = w.Write(resp)
	})
	tsa.srv = httptest.NewServer(mux)
	t.Cleanup(tsa.srv.Close)
	tsa.ca = newTestCA(t, "Test", tsa.srv.URL+"/ca.crt")
	return tsa
}

func (tsa *testTSA) url() string { return tsa.srv.URL + "/tsa" }

func (tsa *testTSA) rotateCA(t *testing.T) {
	ca := newTestCA(t, "Other", tsa.srv.URL+"/ca.crt")
	tsa.mu.Lock()
	tsa.ca = ca
	tsa.mu.Unlock()
}

// --- fake S3 ----------------------------------------------------------------

type memStore struct {
	mu      sync.Mutex
	objects map[string][]byte
	lock    bool
	failPut bool
}

func newMemStore() *memStore { return &memStore{objects: map[string][]byte{}} }

func (m *memStore) Prefix(id string) string { return "backups/anchors/" + id + "/" }

func (m *memStore) Put(_ context.Context, key string, body []byte, _ string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.failPut {
		return errors.New("s3 unavailable")
	}
	m.objects[key] = append([]byte(nil), body...)
	return nil
}

func (m *memStore) Get(_ context.Context, key string) ([]byte, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	b, ok := m.objects[key]
	if !ok {
		return nil, errors.New("not found")
	}
	return b, nil
}

func (m *memStore) List(_ context.Context, prefix string) ([]string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var keys []string
	for k := range m.objects {
		if strings.HasPrefix(k, prefix) {
			keys = append(keys, k)
		}
	}
	sort.Strings(keys)
	return keys, nil
}

func (m *memStore) ObjectLockEnabled(context.Context) (bool, error) { return m.lock, nil }

// --- harness ----------------------------------------------------------------

type harness struct {
	t       *testing.T
	svc     *Service
	logger  *audit.Logger
	store   *Store
	s3      *memStore
	tsa     *testTSA
	notices []Notice
	fails   []error
}

func newHarness(t *testing.T) *harness {
	t.Helper()
	st, err := storage.OpenInMemory()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { st.DB.Close() })
	logger, err := audit.New(st.DB)
	if err != nil {
		t.Fatal(err)
	}
	h := &harness{t: t, logger: logger, s3: newMemStore(), tsa: newTestTSA(t)}
	h.store = NewStore(st.DB, []byte("01234567890123456789012345678901"))
	h.svc = NewService(h.store, logger, func(context.Context) (ObjectStore, error) { return h.s3, nil }, h.tsa.srv.Client())
	h.svc.SetOnChange(func(_ context.Context, n Notice) { h.notices = append(h.notices, n) })
	h.svc.SetOnFailure(func(_ context.Context, err error) { h.fails = append(h.fails, err) })
	if err := h.svc.Load(); err != nil {
		t.Fatal(err)
	}
	return h
}

func (h *harness) log(n int) {
	h.t.Helper()
	for i := 0; i < n; i++ {
		if err := h.logger.Log(nil, "login", "user", "u1", nil, map[string]int{"n": i}); err != nil {
			h.t.Fatal(err)
		}
	}
}

func (h *harness) input() SettingsInput {
	return SettingsInput{Enabled: true, TSAURL: h.tsa.url(), Interval: "1h"}
}

// enable runs the two-step confirmation the admin UI does.
func (h *harness) enable() ProbeResult {
	h.t.Helper()
	in := h.input()
	err := h.svc.Save(context.Background(), in, nil)
	var nc *NeedsConfirmation
	if !errors.As(err, &nc) {
		h.t.Fatalf("first Save should ask for confirmation, got %v", err)
	}
	in.ConfirmFingerprint = nc.Probe.Fingerprint
	for _, f := range nc.Probe.BrokenRows {
		in.AcceptBroken = append(in.AcceptBroken, f.EventID)
	}
	if err := h.svc.Save(context.Background(), in, nil); err != nil {
		h.t.Fatalf("confirmed Save: %v", err)
	}
	return nc.Probe
}

func (h *harness) check() CheckReport {
	h.t.Helper()
	chain, err := h.logger.VerifyEntireChain()
	if err != nil {
		h.t.Fatal(err)
	}
	rep, err := h.svc.Check(context.Background(), chain)
	if err != nil {
		h.t.Fatal(err)
	}
	return rep
}

// --- tests ------------------------------------------------------------------

func TestEnable_BaselineAndFirstAnchor(t *testing.T) {
	h := newHarness(t)
	h.log(3)

	probe := h.enable()
	if probe.Fingerprint != Fingerprint([]*x509.Certificate{h.tsa.ca.cert}) {
		t.Fatalf("pinned %s, want the CA fetched over AIA", probe.Subject)
	}

	var baselines int
	h.store.DB().QueryRow(`SELECT COUNT(*) FROM audit_events WHERE event_type = 'integrity_baseline'`).Scan(&baselines)
	if baselines != 1 {
		t.Fatalf("baselines = %d", baselines)
	}

	last, err := h.store.Latest()
	if err != nil || last == nil {
		t.Fatalf("no first anchor: %v", err)
	}
	if last.S3Key == "" {
		t.Fatal("first anchor was not copied to S3")
	}
	keys, _ := h.s3.List(context.Background(), "")
	if len(keys) != 2 {
		t.Fatalf("S3 objects = %v", keys)
	}
	if len(h.notices) != 1 || h.notices[0].Event != audit.EventAuditAnchoringEnabled {
		t.Fatalf("notices = %+v", h.notices)
	}

	rep := h.check()
	if rep.Problems != 0 || rep.OK != 1 || rep.S3Receipts != 1 {
		t.Fatalf("check = %+v", rep)
	}
}

func TestAnchorNow_SkipsWhenIdleAndAnchorsNewEvents(t *testing.T) {
	h := newHarness(t)
	h.log(1)
	h.enable()
	calls := h.tsa.calls.Load()

	// Only our own audit_anchored event since the first anchor: nothing to do.
	a, err := h.svc.AnchorNow(context.Background())
	if err != nil || a != nil {
		t.Fatalf("idle AnchorNow = %+v, %v", a, err)
	}
	if h.tsa.calls.Load() != calls {
		t.Fatal("idle run should not call the TSA")
	}

	h.log(2)
	a, err = h.svc.AnchorNow(context.Background())
	if err != nil || a == nil {
		t.Fatalf("AnchorNow = %+v, %v", a, err)
	}
	tipID, _, _ := h.logger.Tip()
	if a.EventID >= tipID || a.HashVersion != audit.HashV2 {
		t.Fatalf("anchor %+v, tip now %d", a, tipID)
	}
}

func TestTamperAfterAnchor_DetectedByRunAndScan(t *testing.T) {
	h := newHarness(t)
	h.log(3)
	h.enable()
	anchored, _ := h.store.Latest()

	// Rewrite row 2 and recompute every later hash, the attack anchoring is for.
	db := h.store.DB()
	if _, err := db.Exec(`UPDATE audit_events SET entity_type = 'forged' WHERE id = 2`); err != nil {
		t.Fatal(err)
	}
	events, _ := h.logger.ChainEvents()
	prev := audit.GenesisHash
	for _, e := range events {
		hash, _ := audit.ComputeHash(e.HashVersion, audit.HashInput{
			PrevHash: prev, EventType: e.EventType, EntityType: e.EntityType, EntityID: e.EntityID,
			ActorUserID: e.ActorID, Payload: e.Payload, CreatedAt: e.CreatedAtRaw(),
		})
		if _, err := db.Exec(`UPDATE audit_events SET prev_hash = ?, hash = ? WHERE id = ?`, prev, hash, e.ID); err != nil {
			t.Fatal(err)
		}
		prev = hash
	}
	if r, _ := h.logger.VerifyEntireChain(); r.Invalid != 0 {
		t.Fatalf("recomputed chain should verify on its own, got %+v", r)
	}

	h.log(1)
	if _, err := h.svc.AnchorNow(context.Background()); err == nil || !strings.Contains(err.Error(), "changed after it was anchored") {
		t.Fatalf("AnchorNow after tamper: %v", err)
	}
	if len(h.fails) == 0 {
		t.Fatal("failure callback not called")
	}

	rep := h.check()
	if rep.Problems == 0 {
		t.Fatalf("scan missed the rewrite: %+v", rep)
	}
	found := false
	for _, f := range rep.Findings {
		if f.Status == FindingTampered && f.EventID == anchored.EventID {
			found = true
		}
	}
	if !found {
		t.Fatalf("findings = %+v", rep.Findings)
	}
}

func TestDeletedServerAnchor_FoundInS3(t *testing.T) {
	h := newHarness(t)
	h.log(1)
	h.enable()
	if _, err := h.store.DB().Exec(`DELETE FROM audit_anchors`); err != nil {
		t.Fatal(err)
	}
	rep := h.check()
	if rep.Problems != 1 || rep.Findings[0].Status != FindingMissingFromServer {
		t.Fatalf("check = %+v", rep)
	}
}

func TestEnable_WithAlreadyBrokenRows(t *testing.T) {
	h := newHarness(t)
	h.log(3)
	if _, err := h.store.DB().Exec(`UPDATE audit_events SET payload = '{"n":99}' WHERE id = 2`); err != nil {
		t.Fatal(err)
	}

	// Confirming the fingerprint but not the broken row is not enough.
	in := h.input()
	err := h.svc.Save(context.Background(), in, nil)
	var nc *NeedsConfirmation
	if !errors.As(err, &nc) || len(nc.Probe.BrokenRows) != 1 || nc.Probe.BrokenRows[0].EventID != 2 {
		t.Fatalf("probe = %+v, %v", nc, err)
	}
	in.ConfirmFingerprint = nc.Probe.Fingerprint
	if err := h.svc.Save(context.Background(), in, nil); !errors.As(err, &nc) {
		t.Fatalf("unconfirmed broken row should block, got %v", err)
	}
	in.AcceptBroken = []int64{2}
	if err := h.svc.Save(context.Background(), in, nil); err != nil {
		t.Fatal(err)
	}
	if last, _ := h.store.Latest(); last == nil {
		t.Fatal("first anchor should cover the chain with the known break")
	}
	rep := h.check()
	if rep.Problems != 0 || len(rep.KnownBroken) != 1 {
		t.Fatalf("check = %+v", rep)
	}
}

func TestCAChange_FailsUntilRepinned(t *testing.T) {
	h := newHarness(t)
	h.log(1)
	h.enable()

	h.tsa.rotateCA(t)
	h.log(1)
	if _, err := h.svc.AnchorNow(context.Background()); err == nil || !strings.Contains(err.Error(), "re-pin") {
		t.Fatalf("expected re-pin error, got %v", err)
	}

	in := h.input()
	in.Repin = true
	err := h.svc.Save(context.Background(), in, nil)
	var nc *NeedsConfirmation
	if !errors.As(err, &nc) {
		t.Fatalf("re-pin should ask for confirmation, got %v", err)
	}
	in.ConfirmFingerprint = nc.Probe.Fingerprint
	if err := h.svc.Save(context.Background(), in, nil); err != nil {
		t.Fatal(err)
	}
	if _, err := h.svc.AnchorNow(context.Background()); err != nil {
		t.Fatalf("after re-pin: %v", err)
	}
	// Old receipts still verify against the CA recorded with them.
	if rep := h.check(); rep.Problems != 0 {
		t.Fatalf("check = %+v", rep)
	}
}

func TestTSAOutage_BacksOffAndAlerts(t *testing.T) {
	h := newHarness(t)
	h.log(1)
	h.enable()
	h.tsa.fail.Store(true)
	h.log(1)
	if _, err := h.svc.AnchorNow(context.Background()); err == nil {
		t.Fatal("expected failure")
	}
	st, _ := h.svc.Status()
	next, _ := time.Parse(time.RFC3339, st.NextRunUTC)
	if st.LastError == "" || time.Until(next) > 2*time.Minute {
		t.Fatalf("status = %+v", st)
	}
	var failed int
	h.store.DB().QueryRow(`SELECT COUNT(*) FROM audit_events WHERE event_type = 'audit_anchor_failed'`).Scan(&failed)
	if failed != 1 || len(h.fails) != 1 {
		t.Fatalf("failed events = %d, callbacks = %d", failed, len(h.fails))
	}
}

func TestS3Outage_KeepsReceiptAndRetries(t *testing.T) {
	h := newHarness(t)
	h.log(1)
	h.enable()
	h.s3.failPut = true
	h.log(1)
	a, err := h.svc.AnchorNow(context.Background())
	if err == nil || a == nil || a.S3Key != "" {
		t.Fatalf("AnchorNow = %+v, %v", a, err)
	}
	if rep := h.check(); rep.Warnings == 0 {
		t.Fatalf("not-uploaded receipt should warn: %+v", rep)
	}
	h.s3.failPut = false
	if _, err := h.svc.AnchorNow(context.Background()); err != nil {
		t.Fatal(err)
	}
	if _, pending, _ := h.store.Count(); pending != 0 {
		t.Fatalf("pending uploads = %d", pending)
	}
}

func TestDisable_TakesFinalAnchor(t *testing.T) {
	h := newHarness(t)
	h.log(1)
	h.enable()
	h.log(2)
	before, _, _ := h.store.Count()

	in := h.input()
	in.Enabled = false
	if err := h.svc.Save(context.Background(), in, nil); err != nil {
		t.Fatal(err)
	}
	after, _, _ := h.store.Count()
	if after != before+1 {
		t.Fatalf("anchors %d -> %d; want a final anchor", before, after)
	}
	if st, _ := h.svc.Status(); st.Enabled {
		t.Fatal("still enabled")
	}
	last := h.notices[len(h.notices)-1]
	if last.Event != audit.EventAuditAnchoringDisabled || !strings.Contains(last.Message, "final anchor") {
		t.Fatalf("notice = %+v", last)
	}
	if _, err := h.svc.AnchorNow(context.Background()); !errors.Is(err, ErrNotEnabled) {
		t.Fatalf("AnchorNow when off: %v", err)
	}
}

func TestRestore_ReportedAsExplainedBreak(t *testing.T) {
	h := newHarness(t)
	h.log(2)
	h.enable()
	first, _ := h.store.Latest()
	h.log(3)
	second, err := h.svc.AnchorNow(context.Background())
	if err != nil || second == nil {
		t.Fatal(err)
	}
	oldTipID, oldTipHash, _ := h.logger.Tip()

	// Simulate restoring an archive taken right after the first anchor: the
	// later rows and the second receipt's record are gone; S3 still has it.
	db := h.store.DB()
	if _, err := db.Exec(`DELETE FROM audit_events WHERE id > ?`, first.EventID); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`DELETE FROM audit_anchors WHERE id = ?`, second.ID); err != nil {
		t.Fatal(err)
	}
	h.log(2) // rows written after the restore reuse the replaced ids
	rec := backup.RestoreRecord{
		OldTipID: oldTipID, OldTipHash: oldTipHash,
		ArchiveTipID: first.EventID, ArchiveTipHash: first.TipHash,
		AppliedAt: time.Now().UTC().Format(time.RFC3339Nano),
	}
	if err := h.logger.Log(nil, audit.EventBackupRestored, audit.EntitySystem, "backup", nil, rec); err != nil {
		t.Fatal(err)
	}

	rep := h.check()
	if rep.Problems != 0 || rep.Explained != 1 || rep.Findings[0].Status != FindingReplacedByRestore {
		t.Fatalf("check = %+v", rep)
	}

	// Without the restore record the same state is tampering.
	if _, err := db.Exec(`DELETE FROM audit_events WHERE event_type = 'backup_restored'`); err != nil {
		t.Fatal(err)
	}
	if rep := h.check(); rep.Problems == 0 {
		t.Fatalf("unexplained truncation passed: %+v", rep)
	}
}

func TestWriteBundle(t *testing.T) {
	h := newHarness(t)
	h.log(2)
	h.enable()
	var buf bytes.Buffer
	if err := h.svc.WriteBundle(&buf); err != nil {
		t.Fatal(err)
	}
	zr, err := zip.NewReader(bytes.NewReader(buf.Bytes()), int64(buf.Len()))
	if err != nil {
		t.Fatal(err)
	}
	names := map[string]bool{}
	tsrs := 0
	for _, f := range zr.File {
		names[f.Name] = true
		if strings.HasSuffix(f.Name, ".tsr") {
			tsrs++
		}
	}
	for _, want := range []string{"README.txt", "audit_events.jsonl", "tsa-ca.pem"} {
		if !names[want] {
			t.Errorf("bundle lacks %s (has %v)", want, names)
		}
	}
	if tsrs != 1 {
		t.Errorf("receipts in bundle = %d", tsrs)
	}
}

func TestStamp_RejectsWrongDigest(t *testing.T) {
	tsa := newTestTSA(t)
	client := &TSA{URL: tsa.url(), Client: tsa.srv.Client()}
	digest := bytes.Repeat([]byte{1}, 32)
	raw, _, err := client.Stamp(context.Background(), digest)
	if err != nil {
		t.Fatal(err)
	}
	other := bytes.Repeat([]byte{2}, 32)
	if _, err := VerifyReceipt(raw, other, []*x509.Certificate{tsa.ca.cert}); err == nil {
		t.Fatal("receipt verified for a different hash")
	}
	stranger := newTestCA(t, "Stranger", "")
	if _, err := VerifyReceipt(raw, digest, []*x509.Certificate{stranger.cert}); err == nil {
		t.Fatal("receipt verified against an unrelated CA")
	}
	if _, err := VerifyReceipt(raw, digest, []*x509.Certificate{tsa.ca.cert}); err != nil {
		t.Fatalf("valid receipt: %v", err)
	}
}

func TestCheckTSAKeyUsage(t *testing.T) {
	ca := newTestCA(t, "Good", "")
	if err := checkTSAKeyUsage(ca.leaf); err != nil {
		t.Fatalf("conforming TSA cert rejected: %v", err)
	}
	// Go's ExtKeyUsage field writes the extension non-critical, which RFC 3161
	// and OpenSSL reject.
	key, _ := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	tmpl := &x509.Certificate{
		SerialNumber: big.NewInt(3),
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().Add(time.Hour),
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageTimeStamping},
	}
	der, _ := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &key.PublicKey, key)
	leaf, _ := x509.ParseCertificate(der)
	if err := checkTSAKeyUsage(leaf); err == nil {
		t.Fatal("non-critical extended key usage accepted")
	}
}
