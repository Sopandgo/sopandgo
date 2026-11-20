# 3. Page analytics run only on the public demo

Status: accepted (2026-11-20)

## Context

The public demo should show how visitors use it: which pages they open, whether they switch to the auditor view, whether they follow the install link. Lab instances are a different matter. Page analytics there would record what employees read and when, which is a works-council and data-protection question that the lab, not SOPandGO, has to answer. A lab admin could also turn it on by copying a demo's environment by mistake.

The demo cannot be configured in the admin UI: demo mode closes the admin area and wipes the database at every start.

## Decision

Umami is configured with three environment variables (`UMAMI_SCRIPT_URL`, `UMAMI_WEBSITE_ID`, `UMAMI_DOMAINS`) that the SvelteKit server reads. The tracker renders only when `DEMO_MODE` is on and the values are valid. The rule lives in code (`frontend/src/lib/analytics.ts`), so on any other instance the variables have no effect; the server logs why at startup.

The tracker ignores query strings and fragments and honours Do Not Track.

## Consequences

- A lab instance never loads analytics, whatever its environment says. Supporting analytics on lab instances would need a new decision, not a configuration change.
- The demo needs no admin UI or database row for analytics, so the nightly reset does not affect it.
- Umami sees what any visitor of the demo sees. The demo holds only seeded, public data, so page paths reveal nothing private.
