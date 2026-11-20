# Deployment

This document describes how to deploy SOPandGO in a typical self-hosted environment. It is intended for administrators and technical staff responsible for running the system in a lab, university, or small organization.

Trying the product locally with sample data first? See [Evaluating sopandgo](./evaluating.md). Version tags and GitHub Releases: [Releasing](./releasing.md).

## Deployment model

SOPandGO is designed to run as a **single self-hosted service**.

Key properties:
- **App container**: One image (**`sopandgo`**) runs Caddy, the Go API, and the SvelteKit Node server together.
- **PDF (optional)**: Compose also runs **Gotenberg** by default when PDF export is enabled; see root `docker-compose.yml` and `.env.example`.
- **Persistent Storage**: One data volume for all database and file assets.
- **Configuration**: Sane defaults in compose; override with a root **`.env`** (from **`.env.example`**) for `ORIGIN`, mail encryption key, ports, and PDF-related variables.

This model is intentional and optimized for simplicity, portability, and reliability.

## Data directory

All persistent data lives under **`DATA_DIR`** inside the process (default **`/app/data`** in Docker). On the default compose setup, that is **`./backend/data` on the host** bind-mounted to **`/app/data`** in the **`sopandgo`** container.

The directory contains:

- `app.db`: The SQLite database.
- `sops/`: SOP content — versioned Markdown and per-SOP assets (images, attachments) on disk.
- `users/`: Profile pictures — per-user directories with `avatar-96.jpg`, `avatar-256.jpg`, `avatar-512.jpg`, and `avatar-1024.jpg`.

**Important:** This directory **must** be on persistent storage so data survives image updates and container recreation.

## Basic deployment (recommended)

### Using docker compose
A **`docker-compose.yml`** is in the repository root ( **`sopandgo`** app + **`gotenberg`** for PDF by default). Copy **`.env.example`** to **`.env`** and set variables there; Compose substitutes `${VAR}` from `.env` automatically.

Start the stack. The compose file uses the published image `ghcr.io/sopandgo/sopandgo:1`. You do not choose a version number.

```sh
docker compose pull
docker compose up -d
```

Initial Credentials:

- User: `admin`
- Password: `admin`
- **Forced change:** the bootstrap admin must set a new password on first login before other routes unlock. The form asks for the current password, which is `admin`.

A fresh install has no demo accounts. Demo users and sample SOPs are inserted on first boot only when `SEED_DEMO_DATA=true` is set, for a local trial (see [Evaluating](./evaluating.md)). They use weak passwords such as `12345`. Changing the flag later does not remove data that was already seeded.

### First-week checklist

1. Set `ORIGIN` and `SECRET_ENCRYPTION_KEY` in `.env`. Leave `SEED_DEMO_DATA` unset or `false` so no demo users or sample SOPs are inserted. Restart if you changed `ORIGIN` or the encryption key after first boot.
2. Sign in as `admin` / `admin` and set a strong password. The form asks for the current password, which is `admin`.
3. Behind a reverse proxy (Coolify, Traefik, nginx): open the **Audit log** and check that your sign-in shows your own IP address, not a private one such as `10.…` or `172.…`. If it shows a private address, set `TRUSTED_PROXIES`; see [Behind a reverse proxy](#behind-a-reverse-proxy).
4. Optionally save SMTP or Resend under **Settings → Email**, switch it on, and send a test email. Without a transport switched on, invites use manual links.
5. Turn on two-factor authentication for the admin account in **Account settings**, store the recovery codes, and choose the organization's required second factor under **Settings** (General → Security). If an admin is locked out later, see [Locked-out admin](#locked-out-admin-break-glass).
6. Create a real user invite. Keep demo data only for a trial.
7. Open **Settings → Backup**, export a `.zip`, and validate it.
8. Practice restore once on a non-production copy: stage apply, restart the container, and confirm SOPs and acknowledgments. See [Backup and restore](./backup-and-restore.md).

## Advanced Configuration

### Environment Variables

| Variable | Default | Description |
| --- | --- | --- |
| `DATA_DIR` | `/app/data` | The internal path for SQLite and assets. |
| `SEED_DEMO_DATA` | `false` | When `true` (any case), the first start with no `app.db` inserts demo users and sample SOPs for a local trial. Any other value inserts nothing. Has no effect once `app.db` exists. Ignored when `DEMO_MODE` is on. |
| `DEMO_MODE` | `false` | When `true` (any case), the instance becomes a read-only public demo that is wiped and reseeded at every start. Only for a public demo; see [Demo mode](#demo-mode). |
| `UMAMI_SCRIPT_URL` | - | Umami tracker script for the public demo, for example `https://cloud.umami.is/script.js`. Must be `https` (`http` only for `localhost`). Ignored unless `DEMO_MODE` is on; see [Demo mode](#demo-mode). |
| `UMAMI_WEBSITE_ID` | - | The Umami website ID (a UUID). Ignored unless `DEMO_MODE` is on. |
| `UMAMI_DOMAINS` | - | Optional comma-separated host names Umami counts, for example `demo.sopandgo.com`. Ignored unless `DEMO_MODE` is on. |
| `ORIGIN` | `http://localhost:8087` | **CRITICAL for the Go API.** The full public URL of your instance (invite/reset links, CORS-facing URLs). If it does not match the browser URL, login/invites will fail. SvelteKit 3 / adapter-node no longer reads `ORIGIN`; CSRF and request origin use `HOST_HEADER` / `PROTOCOL_HEADER` instead (see below). On a non-default port with no outer proxy, Caddy forwards the browser `Host` value, port included. An outer proxy's `X-Forwarded-Host` is forwarded as sent. |
| `HOST_WEB_PORT` | `8087` | Host port mapped to Caddy `:80` inside the `sopandgo` container (compose). |
| `BACKEND_URL` | `http://127.0.0.1:8080` | Internal URL the SvelteKit Node server uses to reach the Go API (loopback inside the container). |
| `HOST_HEADER` | `x-forwarded-host` | Forwarded host header name for SvelteKit behind Caddy. |
| `PROTOCOL_HEADER` | `x-forwarded-proto` | Forwarded protocol header name for SvelteKit behind Caddy. |
| `TRUSTED_PROXIES` | - | Addresses of the outer reverse proxy in front of the container, as CIDR ranges separated by spaces, or `private_ranges` (the private IPv4 and IPv6 ranges and loopback). Caddy then takes the visitor's address from that proxy's `X-Forwarded-For`, so the API's rate limits (sign-in, password reset, general and demo sessions) apply per visitor. Empty trusts no proxy: behind Coolify or Traefik, every visitor then shares the proxy's limits. Set it only to addresses that nothing but your proxy can connect from; any peer in these ranges can choose the address it is limited as. See [Behind a reverse proxy](#behind-a-reverse-proxy). |
| `BODY_SIZE_LIMIT` | `52428800` | Max request size (bytes). Default is ~50MB. |
| `AUDIT_STRICT_TYPES` | `false` | If `true`, audit writes reject unknown `event_type`/`entity_type` values (recommended once all extensions use canonical types). |
| `DB_BUSY_TIMEOUT` | `5s` | How long any database write waits for another writer before failing, as a Go duration of at least `1ms` (for example `500ms`, `10s`). An invalid value stops startup with an error. A timed-out write returns an error and is reported in the application log; increase this only if legitimate write transactions regularly exceed five seconds. |
| `AUDIT_BUSY_TIMEOUT` | - | **Deprecated**; rename it to `DB_BUSY_TIMEOUT`. Still honoured when `DB_BUSY_TIMEOUT` is unset, now for every database write, and logs a deprecation warning at startup. When both are set, `DB_BUSY_TIMEOUT` wins. A `docker-compose.yml` copied from an older release sets `AUDIT_BUSY_TIMEOUT=5s` itself, so the warning appears until you update that line. |
| `SECRET_ENCRYPTION_KEY` | - | **Required to save outbound secrets in the UI** (SMTP password, Resend API key, Slack webhook URL, Gotify token, optional webhook bearer, S3 backup secret access key), **to send mail or integration notifications that need those secrets, and to use two-factor authentication** (it seals each user's authenticator secret). A 32-byte AES-256 key, provided as **base64** or **hex** (64 hex chars, optional `0x` prefix). Used only to encrypt secrets stored in SQLite—not for signing JWTs. Generate: `openssl rand -base64 32`. If unset, the server starts, but the admin UI cannot persist those credentials until this is set and the process is restarted. |
| `APP_VERSION` | baked into the image | Label on the public footer, in the signed-in sidebar, and in admin **export** `manifest.json`. Release images set this at build time (for example `1.0.1`). Labs do not set it. Contributor source builds and `go run` use `dev`. Local `npm run dev` ignores this and shows `git describe` instead. The root `package.json` version is not used. |
| `PDF_EXPORT_ENABLED` | `true` | When `true`, PDF artifact generation/download is enabled (requires a renderer when not `none`). A renderer failure never fails a version create or lifecycle change; it is logged, and the startup backfill generates the missing PDFs. Set `false` for local `go run` / `air` without Gotenberg. |
| `PDF_RENDERER` | `gotenberg` | `gotenberg` or `none`. |
| `GOTENBERG_URL` | `http://gotenberg:3000` | Base URL of the Gotenberg service (default compose service name: `gotenberg`). |
| `PDF_GENERATOR_VERSION` | `1` | Integer version for PDF pipeline/backfill; bump when output format changes. |

The Go backend does **not** read a `.env` file on its own. With **Docker Compose**, put values in a root **`.env`** (substituted into `docker-compose.yml`) and/or under `environment:` in the compose file. For local **`go run`**, export variables in your shell or IDE run configuration.

### Email (SMTP, Resend, or manual links)

Outgoing mail (user invites, password resets) is **not** configured via `SMTP_HOST` / `SMTP_USER` style environment variables. Credentials live in SQLite and are encrypted with `SECRET_ENCRYPTION_KEY`.

**How mail is delivered** (`/admin/settings/email`): the **SMTP** and **Resend** cards each get an on/off toggle in their header once their settings are saved, like the integration channels. At most one is on; switching one on switches the other off.

- **A transport is on:** invite/reset links (and publish notices) are sent by email through it. **Send test email** uses it too.
- **Neither is on, or none is saved (the default on a fresh install):** invite/reset links are returned to admins in the UI/API for manual sharing over a trusted channel (**manual links**), and the Email page says so. Nothing tries to send through an empty configuration.

1. Set `SECRET_ENCRYPTION_KEY` as above (required to save any stored secret and to send).
2. Sign in as **admin**, open **Settings → Email** (`/admin/settings/email`).
3. For **SMTP**: enter host, port, username, password, and from-address, then save.
4. For **Resend**: create an API key in the Resend dashboard, enter from-address and API key, then save. Use a verified domain (or Resend’s test sender for trials).
5. Switch the transport on in its card header, then use **Send test email** to verify connectivity.

To go back to manual links, switch the transport off; its saved settings stay. The API still exposes `mail_mode` and `mail_transport`; `effective_mail_mode` in `GET /api/admin/settings/email` is what invites and resets actually use.

Mail sends invites, password resets, a test email, and optional notices when a version is published, to other active users who can reader-sign. Publish notices are skipped in `manual_links` mode. A failed send leaves the version published. Reminder digests for pending acknowledgments are not sent. Emails for a release candidate, a rejection, or an archive are not sent. The home dashboard lists each person's own pending signatures.

### Outbound integrations (Slack, Gotify, webhook)

Channel credentials are **not** set via env vars (no `SLACK_WEBHOOK_URL` / `GOTIFY_*` feature flags). Configure them under **Settings → Integrations** (`/admin/settings/integrations`), encrypted with the same `SECRET_ENCRYPTION_KEY`.

| Channel | What you configure | Delivery |
| --- | --- | --- |
| Slack | Incoming Webhook URL (HTTPS), enable toggle, event checkboxes | POST `{"text":"..."}` |
| Gotify | Server base URL + application token, enable, events | POST `{url}/message?token=...` |
| Generic webhook | Destination URL, optional bearer token (enter `-` to clear), enable, events | POST JSON envelope (`event`, `occurred_at`, `title`, `message`, `url`, `sop_id`, …) |

**Events:** `sop_published`, `sop_rc`, `sop_rejected`, `backup_s3_failed`, `integrity_check_failed`, `break_glass_used` (an admin was reset with `recover-admin`), `audit_anchor_failed`, `audit_anchoring_changed` (anchoring switched on, off, or reconfigured), and `backup_restored` (a restore replaced the audit log). New channels subscribe to all of them. Channels saved before these events existed keep their saved choice, so tick the new events on each channel. Use **Send test** per channel to verify. Delivery failures are audited and never roll back SOP publish/promote/reject.

If you **rotate** `SECRET_ENCRYPTION_KEY`, existing ciphertext becomes undecryptable; sign in as admin and **re-enter the SMTP password and/or Resend API key**, plus any Slack/Gotify/webhook secrets, then save. Authenticator apps stop working too: admins sign in with a recovery code, users with an emailed code if the policy allows email, and then set up their app again; reset normal users from **Admin → Users** and admins with [`recover-admin`](#locked-out-admin-break-glass).

### Error reporting (Sentry, GlitchTip)

SOPandGO can send unexpected errors to [Sentry](https://sentry.io) or to a self-hosted [GlitchTip](https://glitchtip.com), which speaks the same protocol. It is off until an admin turns it on under **Settings → Error reporting** (`/admin/settings/error-reporting`). There are no environment variables for it, and it does not need `SECRET_ENCRYPTION_KEY`.

1. In Sentry or GlitchTip, create a project and copy its DSN from **Client Keys (DSN)**. It looks like `https://<key>@<host>/<project>`. `http` works for a GlitchTip on the lab network; the page warns that events then travel unencrypted.
2. Paste it, set an **Environment** (default `production`), switch **Report errors** on and save. The change applies at once in the Go API, and within a minute in the SvelteKit server.
3. Click **Send test**. The page shows the event ID, or says the event was not delivered. The container must reach the DSN's host.
4. Optionally tick **Also report errors in users' browsers**. Browsers then load a small Sentry script and report to the DSN themselves, so the DSN host must be reachable from users' machines too.

What is and is not sent is listed in the [security model](../concepts/security-model.md#error-reporting). In short: error type and message, stack trace, route pattern, method and status, version and environment; never users, IP addresses, cookies, headers, bodies or query strings. Error messages are sent as they are.

Panics in the Go API are always recovered and logged with a stack trace, whether reporting is on or off. The API reports 5xx responses except 502 (a service you configured failed, such as SMTP, S3, Slack or a test button) and 503 (maintenance lock). Restoring a backup keeps this instance's error reporting settings.

### Audit anchoring

The audit log is a hash chain, but someone with direct access to the database can rewrite it and recompute every hash. Audit anchoring closes that gap. A Time Stamping Authority (TSA) regularly signs the newest audit hash, and each receipt is copied to your S3 bucket. An auditor can check the receipts without SOPandGO; see [Verifying audit anchors](verifying-anchors.md).

Anchoring is off by default. To switch it on:

1. Configure an S3 bucket under **Settings → Backup**. Anchoring uses that bucket and its credentials, even when scheduled backups are off. Receipts go under `<prefix>/anchors/<instance_id>/`.
2. Protect the receipts from this server. Anyone with a shell on the host can read `SECRET_ENCRYPTION_KEY` and so the S3 credentials. Do one of the following:
   - Turn on **S3 Object Lock** for the bucket, in compliance mode with a default retention period. The bucket must be created with Object Lock enabled.
   - Give SOPandGO a key that can add objects under `anchors/` but not delete or overwrite them:

   ```json
   {
     "Version": "2012-10-17",
     "Statement": [
       { "Effect": "Allow", "Action": ["s3:PutObject", "s3:GetObject"], "Resource": "arn:aws:s3:::lab-backups/anchors/*" },
       { "Effect": "Allow", "Action": ["s3:ListBucket", "s3:GetBucketObjectLockConfiguration"], "Resource": "arn:aws:s3:::lab-backups" }
     ]
   }
   ```

   With versioning on, an overwrite keeps the old version, but a key that can delete can remove it too. Scheduled backups need their own write and delete rights for their own prefix, so a put-only key means separate credentials for backups. In that case Object Lock is simpler. SOPandGO shows a warning until the bucket reports Object Lock.
3. Open **Settings → Audit anchoring**. Enter a TSA URL and choose how often to anchor (default every hour). Any RFC 3161 service works:
   - free public TSAs such as `http://timestamp.digicert.com`, `http://timestamp.sectigo.com`, or `https://freetsa.org/tsr`;
   - a qualified TSA from the EU Trusted List if your timestamps need eIDAS legal standing (these usually need an account; enter its username and password or token).
4. Choose **Test the TSA and switch on**. SOPandGO sends one test request and shows the TSA's CA certificate and its SHA-256 fingerprint. Compare it with what the TSA publishes, then confirm. If audit rows already fail verification (for example rows rewritten by an old migration), they are listed. You must confirm that you reviewed them before anchoring starts. They are then recorded as known in an `integrity_baseline` event.

What it needs and does:

- **Outbound access** from the container to the TSA URL (HTTP or HTTPS) and to S3. `HTTPS_PROXY`, `HTTP_PROXY`, and `NO_PROXY` are honoured. Only the 32-byte hash, the server's IP address, and the request time reach the TSA.
- **Runs** on the chosen interval, and only when something new was logged. **Admin → Integrity** also has **Anchor now**. A failed run never blocks the app: it retries with backoff, writes `audit_anchor_failed`, and notifies integrations. A receipt that could not be copied to S3 stays on the server and is uploaded on the next run.
- **Certificate pinning:** receipts must chain to the CA pinned at setup. When the TSA changes CA, anchoring fails with an alert until an admin ticks **Pin the TSA's current CA again** and confirms the new fingerprint.
- **Changes are loud:** switching anchoring off first takes a final anchor. Every switch-on, switch-off, and settings change is audited and sent to integrations as `audit_anchoring_changed`.
- **Restores:** a restore keeps the running instance's anchoring settings, like its S3 settings. On the first start after a restore, SOPandGO writes `backup_restored` with the old and the restored chain tips, sends `backup_restored` to integrations, and anchors right away. Receipts taken after the restored archive was made show up in **Admin → Integrity** as *replaced by a restore*, for someone to review. They are not reported as a silent pass.
- **The integrity scan** (**Admin → Integrity**) checks every receipt on the server against the chain and the pinned CA. It lists the bucket and flags receipts that are missing from the server or differ from it. Any problem fails the scan and sends `integrity_check_failed`.

### Audit Taxonomy Hardening

Audit taxonomy is now normalized to lowercase (`event_type`, `entity_type`) at write time and via migration for existing rows.

- Keep `AUDIT_STRICT_TYPES=false` during transition if custom integrations still emit ad-hoc values.
- Set `AUDIT_STRICT_TYPES=true` to enforce known canonical values and prevent taxonomy drift.

### Network Access & Security

**SOPandGO** includes an internal Caddy proxy to handle request routing.

* **Direct API Access:** You can directly access the Go backend via `/sopandgo/api/*` (Caddy forwards to Go on **:8080** inside the container).
* **Frontend:** All other traffic is handled by the SvelteKit Node.js adapter on **:3000** inside the container (behind Caddy on **:80**).

**Security Recommendations:**

The application does not enforce network-level restrictions. Operators should:

- **Restrict Access**: Use a firewall or institutional network access control.

- **VPN Deployment**: Deploy behind a VPN (like Tailscale or WireGuard) rather than opening ports to the public internet.

- **HTTPS/TLS:** If exposing to a network, terminate TLS at an external reverse proxy and forward **HTTP** to the **published app port** (compose default **`HOST_WEB_PORT` 8087** → Caddy **:80** inside the **`sopandgo`** container). Do **not** point the outer proxy at the Go process port **8080** unless you intentionally bypass Caddy (unsupported for normal deployments).

#### Behind a reverse proxy

Most labs that put SOPandGO on the internet run it behind a reverse proxy, such as Coolify's Traefik or an nginx that terminates HTTPS. If yours does, set `TRUSTED_PROXIES`. Without it, SOPandGO cannot tell your users apart.

**Why it matters.** The API limits sign-in attempts, password resets, demo sessions and general requests per visitor address, and the audit log records that address for every sign-in. Each hop only sees the hop before it:

```
browser 203.0.113.7 ──► your proxy 10.0.1.5 ──► SOPandGO (Caddy)
```

SOPandGO sees `10.0.1.5` for everyone. Then every user shares one set of limits, so a few wrong passwords from anyone can block sign-in for the whole lab, and the audit log shows the proxy's address instead of the user's. The proxy writes the browser's address into an `X-Forwarded-For` header, but anyone can send that header, so SOPandGO only believes it from addresses you list in `TRUSTED_PROXIES`.

**What it does.** For each request, Caddy checks who connected:

- If the address is in `TRUSTED_PROXIES`, Caddy reads `X-Forwarded-For` from the right, skips addresses it trusts, and takes the first one it does not trust as the visitor. Entries further left came from outside and are ignored, so a visitor cannot pass a fake address.
- Otherwise, Caddy ignores the header and the connecting address is the visitor.

The value is a list of addresses or CIDR ranges separated by spaces, for example `10.0.1.5/32 192.0.2.0/24`. The keyword `private_ranges` stands for every private network address: `10.0.0.0/8`, `172.16.0.0/12`, `192.168.0.0/16`, `127.0.0.1/8`, `fd00::/8` and `::1`. No public internet address is in it. Empty, the default, trusts nothing.

**What to set:**

| How browsers reach SOPandGO | `TRUSTED_PROXIES` |
| --- | --- |
| Directly, no proxy in front | Leave empty |
| Coolify or Traefik on a server of its own (for example a VPS) | `private_ranges` |
| A proxy on another machine, such as an institute's nginx | That proxy's address, for example `10.20.30.40/32` |
| A proxy on a server inside a LAN, where the container's port is also open to that LAN | Only the Docker network's subnet (`docker network inspect <network>` shows it), for example `172.18.0.0/16` |
| Cloudflare's proxy in front of Coolify | See [Cloudflare in front of Coolify](#cloudflare-in-front-of-coolify) |

**Trust only your proxy.** Any machine connecting from a listed address can choose the address it is limited and logged as. `private_ranges` is exact on a VPS, where the proxy is the only machine with a private address. On a server inside a university or office network, colleagues' computers have private addresses too, so list only the proxy there. A list that is too narrow is safe: users share limits, as without the variable. A list that is too broad lets machines on it get around the sign-in limits.

**Close the published port.** The bundled `docker-compose.yml` publishes `HOST_WEB_PORT` (default `8087`) on the host. Behind a proxy, that is a second way in, over plain HTTP, that skips the proxy and its HTTPS. It does not get around the rate limits, because those connections come from public addresses that are not trusted. Remove the `ports:` lines from your compose file, or block the port in the server's firewall (on a VPS, allow only 22, 80 and 443).

**Check it.** Sign in and open the **Audit log**. Your `login` entry shows the address SOPandGO used. It should be your own public address. A private address (`10.…`, `172.16.…` to `172.31.…`, `192.168.…`) means SOPandGO still sees the proxy. On Docker Desktop (Windows, macOS), every request arrives from Docker's own gateway address, whatever `TRUSTED_PROXIES` says; that is fine for a local trial.

##### Cloudflare in front of Coolify

With the DNS record set to **DNS only** (grey cloud), visitors connect to Traefik directly. Use `TRUSTED_PROXIES=private_ranges` and nothing else changes.

With Cloudflare's proxy on (orange cloud), there is one more hop, and both Traefik and SOPandGO must trust Cloudflare. Otherwise Traefik replaces the visitor with a Cloudflare server, and everyone arriving through the same Cloudflare data centre shares one limit.

1. Copy Cloudflare's IPv4 and IPv6 ranges from [cloudflare.com/ips](https://www.cloudflare.com/ips/).
2. In Coolify, open **Servers →** your server **→ Proxy → Configuration**. Add the ranges to the Traefik command, comma-separated, for both entry points, then restart the proxy:
   ```
   - '--entrypoints.http.forwardedHeaders.trustedIPs=<Cloudflare ranges>'
   - '--entrypoints.https.forwardedHeaders.trustedIPs=<Cloudflare ranges>'
   ```
3. Set `TRUSTED_PROXIES=private_ranges <Cloudflare ranges>`, separated by spaces.
4. In Cloudflare, set **SSL/TLS** to **Full (strict)**. Coolify serves a valid certificate; **Flexible** causes redirect loops.

Someone who bypasses Cloudflare and connects to the server's IP directly gains nothing: Traefik does not trust their address and replaces their `X-Forwarded-For`. Cloudflare changes its ranges rarely; when it does, update both places.

### Demo mode

`DEMO_MODE=true` turns an instance into a read-only public demo, like the one at demo.sopandgo.com. It is meant only for that. Never set it on an instance that holds a lab's data.

With demo mode on:

- **Visitors are signed in without a password,** as the seeded researcher. A banner on every page switches between the researcher and the auditor, and links to the install instructions. Demo sessions last 4 hours, and opening one is rate-limited per visitor address.
- **The API refuses every write itself,** including requests sent straight to `/sopandgo/api/*`, with `403` and the error code `demo_read_only`. Password sign-in, the 2FA steps and password reset are refused too, so the seeded passwords and `admin`/`admin` cannot be used. The only writes allowed are opening a demo session, refreshing it and signing out.
- **The admin area is closed** for every role, except the audit log, which the auditor reads. Audit log responses show IP addresses, browsers and email addresses as `redacted`.
- **Nothing is sent:** mail (SMTP and Resend) and notifications (Slack, Gotify, webhooks) are switched off, whatever the stored settings say.
- **Restart = reset.** At every start, the seeder wipes the data directory (database, uploads and generated files) and seeds the demo again, including a Demo Auditor account. Visitors' sessions and audit events last at most until the next restart. Schedule a nightly container restart on the demo host to reset it nightly; see [Resetting the demo nightly](#resetting-the-demo-nightly).

The wipe is guarded by a `.demo-instance` marker the seeder writes into the data directory:

- An empty data directory is seeded and gets the marker.
- A data directory with the marker is wiped and reseeded.
- A data directory that holds data but no marker is refused: the seeder exits with an error and the container does not start. A mistaken `DEMO_MODE=true` cannot wipe a lab's SOPs.

Demo mode always seeds and ignores `SEED_DEMO_DATA`, so you only set `DEMO_MODE=true`. The shipped `SEED_DEMO_DATA=false` can stay as it is.

#### Resetting the demo nightly

Restart the demo container, and only that one, from cron on the host. Do not match containers by part of a name: Coolify names every SOPandGO container `sopandgo-<uuid>`, so a pattern such as `name=sopandgo-` also restarts any lab instance on the same server.

1. Find the demo container's full name. This prints every running container started with `DEMO_MODE=true`:

   ```bash
   for c in $(docker ps --format '{{.Names}}'); do
     docker inspect -f '{{range .Config.Env}}{{println .}}{{end}}' "$c" | grep -qx 'DEMO_MODE=true' && echo "$c"
   done
   ```

2. Add a cron entry (`crontab -e` as root) with that exact name. It restarts the container only if it was started with `DEMO_MODE=true`, so a wrong name never restarts a lab's instance:

   ```cron
   0 4 * * * docker inspect -f '{{range .Config.Env}}{{println .}}{{end}}' sopandgo-<uuid> | grep -qx 'DEMO_MODE=true' && docker restart sopandgo-<uuid> >/dev/null 2>&1
   ```

Coolify keeps the name across redeploys. If the resource is deleted and created again, the name changes, the entry does nothing, and the demo stops resetting until you update it.

#### Analytics on the demo (Umami)

The demo can count visits with [Umami](https://umami.is). Set `UMAMI_SCRIPT_URL` and `UMAMI_WEBSITE_ID`, and optionally `UMAMI_DOMAINS`, next to `DEMO_MODE=true`. Every page then loads the tracker. It does not count query strings or `#` fragments, so SOP searches and filters stay out of Umami, and it skips browsers that send Do Not Track. It also counts two demo events: `demo-switch-role` (with the role switched to) and `demo-install-click`.

The variables work only in demo mode. On any other instance the tracker never loads, and the SvelteKit server logs `Umami analytics is off: demo mode is off` at startup. An invalid value is logged the same way, with the reason, and nothing loads.

Each visitor address can open about ten demo sessions a minute. Returning visitors keep their session and do not count against it. Behind Coolify or Traefik, set `TRUSTED_PROXIES` (see [Behind a reverse proxy](#behind-a-reverse-proxy)); without it, every visitor shares the proxy's address, and so one limit of about ten new sessions a minute for the whole demo, and one general API limit of 20 requests a second. The audit log check described there does not work on the demo, which shows addresses as `redacted`: check the setting on a deployment with `DEMO_MODE=false` and its own volume first, then switch to demo mode with a new, empty volume.

### Two-factor authentication

Users add an authenticator app in **Account settings**. Under **Settings → General → Security**, an admin chooses the organization's required second factor:

- **None** (default): password only, unless someone adds an app on their own.
- **Email code or app**: every sign-in needs an app code or a code mailed to the account's address. Admins must use an app. Needs a switched-on email transport; while this level is active, switching email off stops users without an app from signing in (Settings → Email warns about this).
- **Authenticator app**: every sign-in needs an app code; email codes are not accepted.

Levels above none need `SECRET_ENCRYPTION_KEY` and an admin who already uses an app. Admins get recovery codes when they set up an app. A normal user who lost their phone is reset from **Admin → Users**.

### Locked-out admin (break-glass)

If an admin lost their password, authenticator, and recovery codes, the server operator resets that admin from a shell in the container. There is no web route for this.

**Coolify:** open the application, choose **Terminal**, select the `sopandgo` container, then run:

```bash
cd /app
./sopandgo recover-admin --list
./sopandgo recover-admin --email admin@lab.example.org
```

**Docker Compose:**

```bash
docker compose exec sopandgo ./sopandgo recover-admin --list
docker compose exec sopandgo ./sopandgo recover-admin --email admin@lab.example.org
```

The command asks you to type the email again (`--yes` skips this). It then removes the admin's 2FA and recovery codes, sets a temporary password, reactivates the account, and signs out every session. It prints the temporary password once. Sign in with it, choose a new password, and set up 2FA again.

- It runs safely while the server is up and uses `DATA_DIR` (`/app/data` in the image). It refuses to start when there is no `app.db` there, so a wrong path cannot create an empty database.
- Each reset writes `admin_break_glass_reset` to the audit log in the same transaction, with the host name and OS user. Refused or cancelled runs write `admin_break_glass_failed`. Integrations subscribed to `break_glass_used` get an alert. Review these with the other admin events.
- It only resets admin accounts. Anyone who can open this terminal can already read the database, so protect host and Coolify access accordingly (see `docs/concepts/security-model.md`).

## Maintenance

### Backup

**Recommended:** Sign in as **admin** and use **Settings → Backup** (`/admin/settings/backup`) to **export** a `.zip` (consistent DB snapshot plus `sops/`, `users/` when present, and `manifest.json`). You can **validate** archives and **stage** a restore; staged restores apply on the **next backend restart**. See `docs/ops/backup-and-restore.md`.

**Manual:** Stop the container and copy the entire mounted data directory, or at minimum `app.db`, the `sops/` tree, and `users/` together (SQLite may use `app.db-wal` / `app.db-shm` while running—stopping the container avoids inconsistent copies).

During backup export, mutating API requests may return `503` briefly while a maintenance lock is active; reads generally continue.

### Upgrading

From **1.0.0** onward, the image tag in `docker-compose.yml` is `ghcr.io/sopandgo/sopandgo:1`. That name moves to each new 1.x release. A normal update does not edit the tag.

1. Export a backup (admin UI) or stop the container and copy the data directory.
2. `docker compose pull` and `docker compose up -d`.
3. Append-only migrations in `backend/internal/storage/migrations.go` apply automatically on startup.
4. Verify login and a published SOP. After sign-in, the sidebar shows the exact release, such as `v1.0.1`. Public pages still show it in the footer.

### Rollback and a future major version

These are not part of a normal install or update.

- **Roll back** to an exact release by changing the image tag in `docker-compose.yml` from `1` to that release, for example `ghcr.io/sopandgo/sopandgo:1.0.1`, then pull and start. If the database was already migrated by a newer release, restore the backup instead. Do not point an older binary at a database that has already been migrated.
- **Move to 2.x** only when a release note says to. Change the image tag from `1` to `2`. Until then, leave it at `1`. The name `latest` can move to 2.0.0; the lab tag `1` does not.
