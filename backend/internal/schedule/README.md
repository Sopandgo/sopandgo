# Schedule Package

Runs a periodic background job. It owns everything a job needs besides its work: whether it is on, its interval, the next run, one run at a time, "run now", "run soon", backoff after failures, the status snapshot, and failure reporting. Automatic S3 backups (`internal/backup`) and audit anchoring (`internal/anchor`) are its two jobs.

## Interface

| Piece | Role |
| --- | --- |
| **`Job[T]`** | What a job supplies: `Name` for log lines, `Run` (its own work and its success audit event, returning what `RunNow` hands back), `FailureEvent` and `FailureEntityID` for the failure audit event, and an optional `OnFailure` notice |
| **`New(job, logger)`** | A job that is off. `logger` may be nil |
| **`Configure(enabled, interval)`** | Off: no next run. On with the same interval and a run scheduled: the next run stays. Otherwise: now plus the interval. Resets the failure count and wakes the loop |
| **`Start(ctx)`** | The loop. Blocks until `ctx` is done; callers run it with `go` |
| **`RunNow(ctx)`** | Runs now, after any run in progress, and returns its result, or `ErrNotEnabled` when off |
| **`RunSoon()`** | Makes the next run due now when on (anchoring uses it after a restore) |
| **`Status()`** | `Snapshot`: enabled, interval, next run, last attempt, last success, last error, consecutive failures |

## After a run

- **Success:** clear the error and the failure count, set the last success, next run one interval after the run ended.
- **Failure:** count it, keep the error, next run after the backoff: 1 minute, doubled for each consecutive failure after the first (at most 10 doublings), never more than the interval.
- **Both:** the last attempt is the run's start. A job switched off during the run has no next run. The run uses the interval it started with.

Failures are reported in this order: a log line, then the audit event (system entity, the job's entity id, payload `{"error": …}`; a failed write is only logged), then the notice. The notice runs on its own goroutine after the run lock is released, with a context that outlives the caller's request, so a slow webhook never holds up a run, `RunNow` or `Configure`.

Status lives in memory and starts empty on every boot. Tests use short real intervals; there is no clock seam.
