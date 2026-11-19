package schedule_test

import (
	"context"
	"errors"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/sopandgo/sopandgo/backend/internal/audit"
	"github.com/sopandgo/sopandgo/backend/internal/schedule"
	"github.com/sopandgo/sopandgo/backend/internal/storage"
)

func TestOff_RunNowRefusesAndNothingIsScheduled(t *testing.T) {
	ran := 0
	job := schedule.New(schedule.Job[int]{
		Name: "test job",
		Run:  func(context.Context) (int, error) { ran++; return 1, nil },
	}, nil)

	if _, err := job.RunNow(context.Background()); !errors.Is(err, schedule.ErrNotEnabled) {
		t.Fatalf("RunNow while off: want ErrNotEnabled, got %v", err)
	}
	if ran != 0 {
		t.Fatalf("job ran %d times while off", ran)
	}
	if st := job.Status(); st.Enabled || !st.NextRun.IsZero() {
		t.Fatalf("status while off: %+v", st)
	}
}

// counter is a stub job that counts its runs and can be told to fail.
type counter struct {
	runs atomic.Int32
	fail atomic.Bool
}

func (c *counter) run(context.Context) (int, error) {
	n := int(c.runs.Add(1))
	if c.fail.Load() {
		return n, errors.New("stub failure")
	}
	return n, nil
}

func newCounterJob(c *counter) *schedule.Scheduled[int] {
	return schedule.New(schedule.Job[int]{Name: "test job", Run: c.run}, nil)
}

// startLoop runs Start until the test ends.
func startLoop(t *testing.T, job *schedule.Scheduled[int]) {
	t.Helper()
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() { job.Start(ctx); close(done) }()
	t.Cleanup(func() { cancel(); <-done })
}

// eventually fails the test unless cond holds within a second.
func eventually(t *testing.T, what string, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(time.Second)
	for !cond() {
		if time.Now().After(deadline) {
			t.Fatalf("timed out waiting for %s", what)
		}
		time.Sleep(2 * time.Millisecond)
	}
}

func TestOn_StartRunsWhenDueAndEveryInterval(t *testing.T) {
	c := &counter{}
	job := newCounterJob(c)
	job.Configure(true, 20*time.Millisecond)
	if st := job.Status(); !st.Enabled || st.NextRun.IsZero() {
		t.Fatalf("status after switching on: %+v", st)
	}
	startLoop(t, job)

	eventually(t, "three scheduled runs", func() bool { return c.runs.Load() >= 3 })
	st := job.Status()
	if st.LastSuccess.IsZero() || st.LastAttempt.IsZero() || st.LastError != "" {
		t.Fatalf("status after scheduled runs: %+v", st)
	}
}

func TestConfigure_KeepsOrMovesTheNextRun(t *testing.T) {
	job := newCounterJob(&counter{})
	job.Configure(true, time.Hour)
	first := job.Status().NextRun

	time.Sleep(5 * time.Millisecond)
	job.Configure(true, time.Hour)
	if got := job.Status().NextRun; !got.Equal(first) {
		t.Fatalf("same interval should keep the next run: %v -> %v", first, got)
	}

	job.Configure(true, 2*time.Hour)
	moved := job.Status().NextRun
	if until := time.Until(moved); until < 119*time.Minute || until > 2*time.Hour {
		t.Fatalf("a new interval should put the next run one interval out, got %v", until)
	}

	job.Configure(false, 2*time.Hour)
	if st := job.Status(); st.Enabled || !st.NextRun.IsZero() {
		t.Fatalf("switching off should cancel the next run: %+v", st)
	}

	time.Sleep(5 * time.Millisecond)
	job.Configure(true, 2*time.Hour)
	if got := job.Status().NextRun; !got.After(moved) {
		t.Fatalf("switching on should start the interval again: %v, was %v", got, moved)
	}
}

func TestConfigure_OffDuringARunLeavesNothingScheduled(t *testing.T) {
	started, release := make(chan struct{}), make(chan struct{})
	job := schedule.New(schedule.Job[int]{
		Name: "slow job",
		Run: func(context.Context) (int, error) {
			close(started)
			<-release
			return 0, nil
		},
	}, nil)
	job.Configure(true, time.Hour)

	done := make(chan error)
	go func() { _, err := job.RunNow(context.Background()); done <- err }()
	<-started
	job.Configure(false, time.Hour)
	close(release)
	if err := <-done; err != nil {
		t.Fatal(err)
	}
	if st := job.Status(); st.Enabled || !st.NextRun.IsZero() || st.LastSuccess.IsZero() {
		t.Fatalf("status after switching off during the run: %+v", st)
	}
}

func TestRunNow_NeverRunsTwiceAtOnce(t *testing.T) {
	var running, overlaps, runs atomic.Int32
	job := schedule.New(schedule.Job[int]{
		Name: "exclusive job",
		Run: func(context.Context) (int, error) {
			if running.Add(1) > 1 {
				overlaps.Add(1)
			}
			time.Sleep(2 * time.Millisecond)
			running.Add(-1)
			runs.Add(1)
			return 0, nil
		},
	}, nil)
	job.Configure(true, time.Hour)

	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() { defer wg.Done(); _, _ = job.RunNow(context.Background()) }()
	}
	wg.Wait()
	if runs.Load() != 8 || overlaps.Load() != 0 {
		t.Fatalf("runs = %d, overlapping runs = %d", runs.Load(), overlaps.Load())
	}
}

func TestFailures_BackOffAndSuccessResets(t *testing.T) {
	c := &counter{}
	job := newCounterJob(c)
	job.Configure(true, time.Hour)
	c.fail.Store(true)

	for i, want := range []time.Duration{time.Minute, 2 * time.Minute, 4 * time.Minute} {
		if _, err := job.RunNow(context.Background()); err == nil {
			t.Fatal("stub should fail")
		}
		st := job.Status()
		if got := time.Until(st.NextRun); got > want || got < want-5*time.Second {
			t.Fatalf("failure %d: next run in %v, want about %v", i+1, got, want)
		}
		if st.LastError != "stub failure" || st.ConsecutiveFailures != i+1 {
			t.Fatalf("failure %d: status %+v", i+1, st)
		}
	}

	c.fail.Store(false)
	if _, err := job.RunNow(context.Background()); err != nil {
		t.Fatal(err)
	}
	st := job.Status()
	if got := time.Until(st.NextRun); got < 59*time.Minute || st.LastError != "" || st.ConsecutiveFailures != 0 {
		t.Fatalf("after success: next run in %v, status %+v", got, st)
	}
}

func TestFailures_BackoffNeverExceedsTheInterval(t *testing.T) {
	c := &counter{}
	c.fail.Store(true)
	job := newCounterJob(c)
	job.Configure(true, 90*time.Second)
	for i := 0; i < 3; i++ {
		_, _ = job.RunNow(context.Background())
	}
	if got := time.Until(job.Status().NextRun); got > 90*time.Second || got < 85*time.Second {
		t.Fatalf("next run in %v, want the 90s interval", got)
	}
}

func newAuditLogger(t *testing.T) *audit.Logger {
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
	return logger
}

func TestFailure_WritesTheJobsAuditEvent(t *testing.T) {
	logger := newAuditLogger(t)
	c := &counter{}
	c.fail.Store(true)
	job := schedule.New(schedule.Job[int]{
		Name:            "test job",
		Run:             c.run,
		FailureEvent:    audit.EventBackupS3Failed,
		FailureEntityID: "backup",
	}, logger)
	job.Configure(true, time.Hour)
	_, _ = job.RunNow(context.Background())

	events, err := logger.ChainEvents()
	if err != nil {
		t.Fatal(err)
	}
	if len(events) != 1 {
		t.Fatalf("audit events = %+v", events)
	}
	e := events[0]
	if e.EventType != audit.EventBackupS3Failed || e.EntityType != audit.EntitySystem || e.EntityID != "backup" ||
		!strings.Contains(e.Payload, `"error":"stub failure"`) {
		t.Fatalf("audit event = %+v", e)
	}
}

func TestNotice_BlockingForeverDelaysNoRun(t *testing.T) {
	c := &counter{}
	c.fail.Store(true)
	notices := make(chan error, 16)
	block := make(chan struct{})
	t.Cleanup(func() { close(block) })
	job := schedule.New(schedule.Job[int]{
		Name: "test job",
		Run:  c.run,
		OnFailure: func(_ context.Context, err error) {
			notices <- err
			<-block // a webhook that never answers
		},
	}, nil)
	job.Configure(true, 30*time.Millisecond)

	finished := make(chan struct{})
	go func() {
		_, _ = job.RunNow(context.Background())
		_, _ = job.RunNow(context.Background())
		close(finished)
	}()
	select {
	case <-finished:
	case <-time.After(time.Second):
		t.Fatal("a blocked failure notice held up RunNow")
	}

	// Backoff puts the next scheduled run a minute out; switch it to "due now".
	job.RunSoon()
	startLoop(t, job)
	eventually(t, "a scheduled run while notices block", func() bool { return c.runs.Load() >= 3 })
	eventually(t, "a notice per failure", func() bool { return len(notices) >= 3 })
}

func TestNotice_OutlivesTheCallersContext(t *testing.T) {
	c := &counter{}
	c.fail.Store(true)
	got := make(chan error, 1)
	job := schedule.New(schedule.Job[int]{
		Name: "test job",
		Run:  c.run,
		OnFailure: func(ctx context.Context, err error) {
			time.Sleep(10 * time.Millisecond) // let the caller cancel first
			got <- ctx.Err()
		},
	}, nil)
	job.Configure(true, time.Hour)

	ctx, cancel := context.WithCancel(context.Background())
	_, _ = job.RunNow(ctx)
	cancel()
	select {
	case err := <-got:
		if err != nil {
			t.Fatalf("notice context was cancelled with the caller's: %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("notice never ran")
	}
}

func TestRunSoon_RunsPromptlyWhenOnAndNotWhenOff(t *testing.T) {
	c := &counter{}
	job := newCounterJob(c)
	startLoop(t, job)

	job.RunSoon()
	time.Sleep(30 * time.Millisecond)
	if n := c.runs.Load(); n != 0 {
		t.Fatalf("RunSoon while off ran the job %d times", n)
	}
	if !job.Status().NextRun.IsZero() {
		t.Fatal("RunSoon while off scheduled a run")
	}

	job.Configure(true, time.Hour)
	job.RunSoon()
	eventually(t, "the RunSoon run", func() bool { return c.runs.Load() == 1 })
}

func TestStart_ReturnsWhenItsContextIsDone(t *testing.T) {
	job := newCounterJob(&counter{})
	job.Configure(true, time.Hour)
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() { job.Start(ctx); close(done) }()
	cancel()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("Start did not return after its context was cancelled")
	}
}
