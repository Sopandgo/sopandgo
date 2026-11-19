package backup

import (
	"context"
	"encoding/json"
	"errors"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/sopandgo/sopandgo/backend/internal/audit"
	"github.com/sopandgo/sopandgo/backend/internal/objectstore"
)

// s3Harness is the backup job against the in-memory object store, a real
// SQLite database and the real audit logger, with failure notices recorded.
type s3Harness struct {
	t        *testing.T
	svc      *Service
	settings *S3SettingsStore
	logger   *audit.Logger
	objects  *objectstore.Memory
	sch      *S3Scheduler
	notices  chan error
}

func newS3Harness(t *testing.T) *s3Harness {
	t.Helper()
	svc, store, _ := newTestService(t)
	logger, err := audit.New(store.DB)
	if err != nil {
		t.Fatal(err)
	}
	h := &s3Harness{
		t:        t,
		svc:      svc,
		settings: NewS3SettingsStore(store.DB, testS3Key),
		logger:   logger,
		objects:  objectstore.NewMemory("lab"),
		notices:  make(chan error, 16),
	}
	h.sch = NewS3Scheduler(svc, h.settings, h.objects.Opener(), logger)
	h.sch.SetOnFailure(func(_ context.Context, err error) { h.notices <- err })
	return h
}

// enable saves valid settings, which switches scheduled backups on.
func (h *s3Harness) enable(edit func(*SaveS3Input)) {
	h.t.Helper()
	in := validS3Input()
	if edit != nil {
		edit(&in)
	}
	if err := h.sch.SaveSettings(in, nil); err != nil {
		h.t.Fatal(err)
	}
}

// events returns the payloads of the audit events of type eventType.
func (h *s3Harness) events(eventType string) []map[string]any {
	h.t.Helper()
	all, err := h.logger.ChainEvents()
	if err != nil {
		h.t.Fatal(err)
	}
	var out []map[string]any
	for _, e := range all {
		if e.EventType != eventType {
			continue
		}
		var p map[string]any
		if err := json.Unmarshal([]byte(e.Payload), &p); err != nil {
			h.t.Fatal(err)
		}
		out = append(out, p)
	}
	return out
}

// waitNotice returns the next failure notice.
func (h *s3Harness) waitNotice() error {
	h.t.Helper()
	select {
	case err := <-h.notices:
		return err
	case <-time.After(time.Second):
		h.t.Fatal("no failure notice")
		return nil
	}
}

// seed puts an object last modified age ago.
func (h *s3Harness) seed(key string, age time.Duration) {
	h.t.Helper()
	if err := h.objects.Put(context.Background(), key, strings.NewReader("old"), "application/zip"); err != nil {
		h.t.Fatal(err)
	}
	if err := h.objects.SetLastModified(key, time.Now().UTC().Add(-age)); err != nil {
		h.t.Fatal(err)
	}
}

var automatedKey = regexp.MustCompile(`^lab/automated/sopandgo-automated-\d{8}T\d{6}\.zip$`)

func TestS3Backup_RunNowUploadsOneArchive(t *testing.T) {
	h := newS3Harness(t)
	h.enable(nil)

	key, err := h.sch.RunNow(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if !automatedKey.MatchString(key) {
		t.Fatalf("object key %q does not follow the naming pattern", key)
	}
	if keys := h.objects.Keys(""); len(keys) != 1 || keys[0] != key {
		t.Fatalf("objects = %v", keys)
	}
	body, _ := h.objects.Get(context.Background(), key)
	if !strings.HasPrefix(string(body), "PK") {
		t.Fatal("uploaded object is not a zip archive")
	}

	up := h.events(audit.EventBackupS3Uploaded)
	if len(up) != 1 || up[0]["bucket"] != "lab-backups" || up[0]["object_key"] != key || up[0]["manifest"] == nil {
		t.Fatalf("backup_s3_uploaded = %+v", up)
	}

	st := h.sch.Status()
	if st.LastObjectKey != key || st.LastRunUTC == "" || st.LastSuccessUTC == "" || st.LastError != "" {
		t.Fatalf("status = %+v", st)
	}
	next, _ := time.Parse(time.RFC3339, st.NextRunUTC)
	if until := time.Until(next); until < 5*time.Hour {
		t.Fatalf("a success should put the next run one 6h interval out, got %v", until)
	}
}

func TestS3Backup_EmptyKeyPrefix(t *testing.T) {
	h := newS3Harness(t)
	h.objects = objectstore.NewMemory("")
	h.sch = NewS3Scheduler(h.svc, h.settings, h.objects.Opener(), h.logger)
	h.enable(func(in *SaveS3Input) { in.KeyPrefix = "" })

	key, err := h.sch.RunNow(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(key, "automated/sopandgo-automated-") {
		t.Fatalf("object key = %q", key)
	}
	if st := h.sch.Status(); st.KeyPrefix != "automated/" {
		t.Fatalf("status key prefix = %q", st.KeyPrefix)
	}
}

func TestS3Backup_Retention(t *testing.T) {
	const day = 24 * time.Hour
	untouchable := []string{
		"lab/automated/sopandgo-connection-test.txt",
		"lab/automated/notes.zip",
		"lab/automated/sopandgo-automated-old.txt",
		"lab/manual/sopandgo-automated-20200101T000000.zip",
	}
	cases := []struct {
		name      string
		max, days int
		seeded    map[string]time.Duration // automated archives and their age
		kept      []string
	}{
		{
			name: "by age", max: 0, days: 2,
			seeded: map[string]time.Duration{"a": 5 * day, "b": 3 * day, "c": day},
			kept:   []string{"c"},
		},
		{
			name: "by count", max: 2, days: 0,
			seeded: map[string]time.Duration{"a": 5 * day, "b": 3 * day, "c": day},
			kept:   []string{"c"}, // plus the new archive
		},
		{
			name: "by age then count", max: 2, days: 4,
			seeded: map[string]time.Duration{"a": 5 * day, "b": 3 * day, "c": 2 * day, "d": day},
			kept:   []string{"d"},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			h := newS3Harness(t)
			h.enable(func(in *SaveS3Input) { in.RetentionMax, in.RetentionDays = tc.max, tc.days })
			name := func(id string) string { return "lab/automated/sopandgo-automated-" + id + ".zip" }
			for id, age := range tc.seeded {
				h.seed(name(id), age)
			}
			for _, k := range untouchable {
				h.seed(k, 100*day)
			}

			key, err := h.sch.RunNow(context.Background())
			if err != nil {
				t.Fatal(err)
			}
			want := map[string]bool{key: true}
			for _, id := range tc.kept {
				want[name(id)] = true
			}
			for _, k := range untouchable {
				want[k] = true
			}
			got := h.objects.Keys("")
			if len(got) != len(want) {
				t.Fatalf("kept %v, want %v", got, want)
			}
			for _, k := range got {
				if !want[k] {
					t.Fatalf("kept %v, want %v", got, want)
				}
			}
		})
	}
}

// assertFailedRun checks what a failed run leaves behind: the error in the
// status, one backup_s3_failed, one notice, and a retry about a minute out.
func (h *s3Harness) assertFailedRun(err error, wantErr string) {
	h.t.Helper()
	if err == nil || !strings.Contains(err.Error(), wantErr) {
		h.t.Fatalf("RunNow error = %v, want it to mention %q", err, wantErr)
	}
	st := h.sch.Status()
	if !strings.Contains(st.LastError, wantErr) || st.LastRunUTC == "" || st.LastSuccessUTC != "" {
		h.t.Fatalf("status = %+v", st)
	}
	next, _ := time.Parse(time.RFC3339, st.NextRunUTC)
	if until := time.Until(next); until > time.Minute || until < 50*time.Second {
		h.t.Fatalf("retry in %v, want about a minute", until)
	}
	failed := h.events(audit.EventBackupS3Failed)
	if len(failed) != 1 || !strings.Contains(failed[0]["error"].(string), wantErr) {
		h.t.Fatalf("backup_s3_failed = %+v", failed)
	}
	if n := h.waitNotice(); !strings.Contains(n.Error(), wantErr) {
		h.t.Fatalf("notice = %v", n)
	}
}

func TestS3Backup_UploadFailureIsRetriedSoon(t *testing.T) {
	h := newS3Harness(t)
	h.enable(nil)
	h.objects.FailPut(errors.New("bucket unreachable"))

	_, err := h.sch.RunNow(context.Background())
	h.assertFailedRun(err, "bucket unreachable")
	if len(h.events(audit.EventBackupS3Uploaded)) != 0 {
		t.Fatal("a failed upload was audited as uploaded")
	}
}

func TestS3Backup_PruneFailureFailsTheRun(t *testing.T) {
	h := newS3Harness(t)
	h.enable(func(in *SaveS3Input) { in.RetentionMax = 1 })
	h.seed("lab/automated/sopandgo-automated-20200101T000000.zip", 48*time.Hour)
	h.objects.FailDelete(errors.New("delete denied"))

	_, err := h.sch.RunNow(context.Background())
	h.assertFailedRun(err, "delete denied")
}

func TestS3Backup_HeldMaintenanceLockFailsAndRetriesSoon(t *testing.T) {
	h := newS3Harness(t)
	h.enable(nil)
	if !h.svc.TryLock("restore in progress") {
		t.Fatal("could not take the maintenance lock")
	}
	defer h.svc.Unlock()

	_, err := h.sch.RunNow(context.Background())
	if !errors.Is(err, ErrBusy) {
		t.Fatalf("want ErrBusy, got %v", err)
	}
	h.assertFailedRun(err, ErrBusy.Error())
}

func TestS3Backup_RunNowWhileOff(t *testing.T) {
	h := newS3Harness(t)
	if _, err := h.sch.RunNow(context.Background()); !errors.Is(err, ErrS3NotEnabled) {
		t.Fatalf("want ErrS3NotEnabled, got %v", err)
	}
	h.enable(func(in *SaveS3Input) { in.Enabled = false })
	if _, err := h.sch.RunNow(context.Background()); !errors.Is(err, ErrS3NotEnabled) {
		t.Fatalf("after switching off: want ErrS3NotEnabled, got %v", err)
	}
	if st := h.sch.Status(); st.Enabled || st.NextRunUTC != "" {
		t.Fatalf("status while off = %+v", st)
	}
	if keys := h.objects.Keys(""); len(keys) != 0 {
		t.Fatalf("uploaded while off: %v", keys)
	}
}

func TestS3Backup_Configure(t *testing.T) {
	h := newS3Harness(t)
	cfg := S3SchedulerConfig{
		Enabled:    true,
		Bucket:     "lab-backups",
		RootPrefix: "lab",
		Interval:   6 * time.Hour,
	}
	if st := h.sch.Status(); st.Enabled || st.NextRunUTC != "" {
		t.Fatalf("new scheduler should be off, got %+v", st)
	}

	h.sch.Configure(cfg)
	st := h.sch.Status()
	if !st.Enabled || st.Bucket != "lab-backups" || st.KeyPrefix != "lab/automated/" || st.Interval != "6h0m0s" || st.NextRunUTC == "" {
		t.Fatalf("after enable: %+v", st)
	}
	firstNext := st.NextRunUTC

	cfg.RetentionMax = 3
	h.sch.Configure(cfg)
	if st := h.sch.Status(); st.NextRunUTC != firstNext || st.RetentionMax != 3 {
		t.Fatalf("retention change should keep the next run %q: %+v", firstNext, st)
	}

	cfg.Interval = 12 * time.Hour
	h.sch.Configure(cfg)
	if got := h.sch.Status().NextRunUTC; got == firstNext {
		t.Fatal("a new interval should move the next run")
	}

	h.sch.Configure(S3SchedulerConfig{})
	if st := h.sch.Status(); st.Enabled || st.NextRunUTC != "" || st.Bucket != "" {
		t.Fatalf("after disable: %+v", st)
	}
}

func TestS3Backup_SaveSettingsAuditsAndConfigures(t *testing.T) {
	h := newS3Harness(t)
	if err := h.sch.SaveSettings(validS3Input(), nil); err != nil {
		t.Fatal(err)
	}

	pub, err := h.settings.GetPublic()
	if err != nil || !pub.Enabled || pub.Bucket != "lab-backups" || pub.Interval != "6h0m0s" {
		t.Fatalf("saved settings = %+v, %v", pub, err)
	}
	ev := h.events(audit.EventBackupS3SettingsUpdated)
	want := map[string]any{
		"enabled": true, "bucket": "lab-backups", "region": "eu-central-1", "key_prefix": "lab",
		"endpoint": "", "interval": "6h0m0s", "retention_max": float64(7), "retention_days": float64(14),
		"static_credentials": true,
	}
	if len(ev) != 1 || len(ev[0]) != len(want) {
		t.Fatalf("backup_s3_settings_updated = %+v", ev)
	}
	for k, v := range want {
		if ev[0][k] != v {
			t.Fatalf("payload %s = %v, want %v", k, ev[0][k], v)
		}
	}
	if st := h.sch.Status(); !st.Enabled || st.Interval != "6h0m0s" || st.NextRunUTC == "" {
		t.Fatalf("status after save = %+v", st)
	}
}

func TestS3Backup_RefusedSaveWritesNothing(t *testing.T) {
	h := newS3Harness(t)
	bad := validS3Input()
	bad.Interval = "5s"
	if err := h.sch.SaveSettings(bad, nil); !errors.Is(err, ErrInvalidS3Settings) {
		t.Fatalf("want ErrInvalidS3Settings, got %v", err)
	}
	if pub, _ := h.settings.GetPublic(); pub.Configured {
		t.Fatalf("refused save stored settings: %+v", pub)
	}
	if ev := h.events(audit.EventBackupS3SettingsUpdated); len(ev) != 0 {
		t.Fatalf("refused save was audited: %+v", ev)
	}
	if st := h.sch.Status(); st.Enabled {
		t.Fatal("refused save switched backups on")
	}

	noKey := NewS3Scheduler(h.svc, NewS3SettingsStore(h.settings.db, nil), h.objects.Opener(), h.logger)
	if err := noKey.SaveSettings(validS3Input(), nil); !errors.Is(err, ErrS3KeyMissing) {
		t.Fatalf("want ErrS3KeyMissing, got %v", err)
	}
}

func TestS3Backup_TestConnection(t *testing.T) {
	h := newS3Harness(t)
	if err := h.sch.TestConnection(context.Background()); err != nil {
		t.Fatal(err)
	}
	if keys := h.objects.Keys(""); len(keys) != 0 {
		t.Fatalf("connection test left %v behind", keys)
	}

	h.objects.FailPut(errors.New("access denied"))
	if err := h.sch.TestConnection(context.Background()); err == nil || !strings.Contains(err.Error(), "access denied") {
		t.Fatalf("want the put failure, got %v", err)
	}
}

func TestS3Opener_NeedsBucket(t *testing.T) {
	_, store, _ := newTestService(t)
	settings := NewS3SettingsStore(store.DB, testS3Key)
	if _, err := settings.OpenObjectStore(context.Background()); !errors.Is(err, ErrS3NotConfigured) {
		t.Fatalf("want ErrS3NotConfigured, got %v", err)
	}
	sch := NewS3Scheduler(nil, settings, settings.OpenObjectStore, nil)
	if err := sch.TestConnection(context.Background()); !errors.Is(err, ErrS3NotConfigured) {
		t.Fatalf("test connection: want ErrS3NotConfigured, got %v", err)
	}
}
