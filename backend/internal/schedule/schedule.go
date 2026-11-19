// Package schedule runs a periodic background job: whether it is on, its
// interval, the next run, one run at a time, backoff after failures, the
// status snapshot, and failure reporting. A job supplies only its own work.
package schedule

import (
	"context"
	"errors"
	"log"
	"sync"
	"time"

	"github.com/sopandgo/sopandgo/backend/internal/audit"
)

// ErrNotEnabled is returned by RunNow when the job is off.
var ErrNotEnabled = errors.New("scheduled job is off")

// auditLogger is satisfied by *audit.Logger.
type auditLogger interface {
	Log(executor audit.DBTX, eventType, entityType, entityID string, actorUserID *string, payload any) error
}

// Job describes one periodic job. Run does the job's own work, including its
// success audit event, and returns what RunNow hands back to its caller.
type Job[T any] struct {
	// Name is used in log lines.
	Name string
	Run  func(ctx context.Context) (T, error)
	// FailureEvent and FailureEntityID are the audit event written, on the
	// system entity, for every failed run.
	FailureEvent    string
	FailureEntityID string
	// OnFailure, when set, is sent each failure after the run is over. It runs
	// on its own goroutine; nothing waits for it.
	OnFailure func(ctx context.Context, err error)
}

// Snapshot is the job's state. Times are UTC; a zero time means "never" or,
// for NextRun, "nothing scheduled".
type Snapshot struct {
	Enabled             bool
	Interval            time.Duration
	NextRun             time.Time
	LastAttempt         time.Time
	LastSuccess         time.Time
	LastError           string
	ConsecutiveFailures int
}

// Scheduled runs one Job. It is off until Configure switches it on.
type Scheduled[T any] struct {
	job    Job[T]
	logger auditLogger

	runMu sync.Mutex // one run at a time
	wake  chan struct{}

	// Times keep Go's monotonic reading, so a due check right after the
	// timer fires is exact even if the wall clock moves.
	mu          sync.Mutex
	enabled     bool
	interval    time.Duration
	nextRun     time.Time
	lastAttempt time.Time
	lastSuccess time.Time
	lastError   string
	failures    int
}

// New returns a job that is off. logger may be nil (no failure audit events).
func New[T any](job Job[T], logger auditLogger) *Scheduled[T] {
	return &Scheduled[T]{job: job, logger: logger, wake: make(chan struct{}, 1)}
}

// Configure switches the job on or off and sets its interval. Switching off
// cancels the next run. Staying on with the same interval keeps the next run;
// switching on or a new interval puts it one interval from now. It resets the
// failure count.
func (s *Scheduled[T]) Configure(enabled bool, interval time.Duration) {
	s.mu.Lock()
	switch {
	case !enabled:
		s.nextRun = time.Time{}
	case s.enabled && s.interval == interval && !s.nextRun.IsZero():
	default:
		s.nextRun = time.Now().Add(interval)
	}
	s.enabled, s.interval = enabled, interval
	s.failures = 0
	s.mu.Unlock()
	s.signal()
}

func (s *Scheduled[T]) signal() {
	select {
	case s.wake <- struct{}{}:
	default:
	}
}

// Start runs the job whenever it is due, until ctx is done.
func (s *Scheduled[T]) Start(ctx context.Context) {
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
			_, _ = s.run(ctx, true)
		}
	}
}

// RunNow runs the job immediately and returns its result, or ErrNotEnabled
// when the job is off. It waits for a run already in progress.
func (s *Scheduled[T]) RunNow(ctx context.Context) (T, error) {
	out, err := s.run(ctx, false)
	s.signal()
	return out, err
}

// RunSoon makes the next run due now, when the job is on.
func (s *Scheduled[T]) RunSoon() {
	s.mu.Lock()
	if s.enabled {
		s.nextRun = time.Now()
	}
	s.mu.Unlock()
	s.signal()
}

// run runs the job once, then sends the failure notice, if any, on its own
// goroutine after the run lock is released. The notice's context is not
// cancelled with the caller's.
func (s *Scheduled[T]) run(ctx context.Context, onlyIfDue bool) (T, error) {
	out, ran, err := s.runLocked(ctx, onlyIfDue)
	if ran && err != nil && s.job.OnFailure != nil {
		go s.job.OnFailure(context.WithoutCancel(ctx), err)
	}
	return out, err
}

// runLocked runs the job under the run lock and records the outcome. A
// scheduled run (onlyIfDue) is skipped when a RunNow that finished while it
// waited for the lock moved the next run.
func (s *Scheduled[T]) runLocked(ctx context.Context, onlyIfDue bool) (out T, ran bool, err error) {
	s.runMu.Lock()
	defer s.runMu.Unlock()

	s.mu.Lock()
	enabled, interval, next := s.enabled, s.interval, s.nextRun
	s.mu.Unlock()
	if !enabled {
		return out, false, ErrNotEnabled
	}
	if onlyIfDue && (next.IsZero() || time.Now().Before(next)) {
		return out, false, nil
	}

	start := time.Now()
	out, err = s.job.Run(ctx)
	end := time.Now()

	s.mu.Lock()
	s.lastAttempt = start
	after := interval
	if err != nil {
		s.failures++
		s.lastError = err.Error()
		after = backoff(s.failures, interval)
	} else {
		s.failures = 0
		s.lastError = ""
		s.lastSuccess = start
	}
	if s.enabled {
		s.nextRun = end.Add(after)
	} else {
		s.nextRun = time.Time{}
	}
	s.mu.Unlock()

	if err != nil {
		s.reportFailure(err)
	}
	return out, true, err
}

// reportFailure logs the failure and writes the job's audit event. A failed
// audit write is only logged.
func (s *Scheduled[T]) reportFailure(err error) {
	log.Printf("%s failed: %v", s.job.Name, err)
	if s.logger == nil || s.job.FailureEvent == "" {
		return
	}
	if auditErr := s.logger.Log(nil, s.job.FailureEvent, audit.EntitySystem, s.job.FailureEntityID, nil, map[string]any{
		"error": err.Error(),
	}); auditErr != nil {
		log.Printf("audit write failed for %s failure: %v", s.job.Name, auditErr)
	}
}

// backoff is how long to wait after the n-th consecutive failure: 1 minute,
// doubled for each failure after the first (at most 10 doublings), never
// more than the interval.
func backoff(n int, interval time.Duration) time.Duration {
	d := time.Minute << min(n-1, 10)
	return min(d, interval)
}

// Status returns a snapshot of the job's state.
func (s *Scheduled[T]) Status() Snapshot {
	s.mu.Lock()
	defer s.mu.Unlock()
	return Snapshot{
		Enabled:             s.enabled,
		Interval:            s.interval,
		NextRun:             utc(s.nextRun),
		LastAttempt:         utc(s.lastAttempt),
		LastSuccess:         utc(s.lastSuccess),
		LastError:           s.lastError,
		ConsecutiveFailures: s.failures,
	}
}

func utc(t time.Time) time.Time {
	if t.IsZero() {
		return t
	}
	return t.UTC()
}
