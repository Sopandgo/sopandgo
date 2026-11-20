#!/bin/sh
# Strip trailing slashes from ORIGIN (Go API invite/reset absolute URLs). A trailing
# slash can produce pathnames like //login in generated links.
if [ -n "${ORIGIN:-}" ]; then
	while [ "${ORIGIN%/}" != "$ORIGIN" ]; do
		ORIGIN="${ORIGIN%/}"
	done
	export ORIGIN
fi

# The seeder decides from SEED_DEMO_DATA whether to seed, and seeds only when
# app.db is absent.
echo "--- Step 1: Demo data ---"
./seeder || exit 1

echo "--- Step 2: Starting Go Backend ---"
./sopandgo &

echo "--- Step 3: Starting SvelteKit Frontend ---"
PORT=3000 node frontend-build/index.js &

echo "--- Step 4: Starting Caddy Proxy ---"
exec caddy run --config /etc/caddy/Caddyfile --adapter caddyfile
