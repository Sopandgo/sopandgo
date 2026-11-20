# 2. The demo wipe is guarded by a marker the seeder writes

Status: accepted (2026-11-19), amended (2026-11-20): demo mode ignores `SEED_DEMO_DATA`

## Context

A public demo must not accumulate visitors' sessions, audit events and files. The simplest reset is to wipe and reseed the data directory at every container start; a nightly restart then resets the demo. But a wipe at start-up is dangerous: if an operator sets `DEMO_MODE=true` on a lab's instance by mistake, every SOP, signature and audit record would be gone at the next restart.

## Decision

The seeder owns the wipe decision, and only wipes a directory it marked itself. With demo mode on, at every start:

- An empty data directory is seeded and gets a `.demo-instance` marker.
- A directory with the marker is wiped (its contents, not the mount point) and reseeded, and the marker is written again.
- A directory that holds data but no marker is refused. The seeder exits with an error, and the entrypoint does not start the API.

The marker is written before seeding, so a seed that fails half-way is wiped at the next start instead of blocking it.

## Consequences

- Restart is the whole reset procedure. The schedule lives on the demo host (cron or Coolify), not in this repository.
- A mistaken `DEMO_MODE=true` stops the container instead of deleting data. Turning a real instance into a demo needs a new, empty data directory.
- Demo mode always seeds and ignores `SEED_DEMO_DATA`. As first accepted, `DEMO_MODE=true` with `SEED_DEMO_DATA=false` was refused as contradictory. But `docker-compose.yml` and `.env.example` both ship `SEED_DEMO_DATA=false`, so every operator who only added `DEMO_MODE=true` got a container that would not start. The refusal protected nothing: the marker is what keeps a lab's data from being wiped.
- Anyone who can write a `.demo-instance` file into a lab's data directory and set `DEMO_MODE` can wipe it. Both need host access, which already allows deleting the data directly.
