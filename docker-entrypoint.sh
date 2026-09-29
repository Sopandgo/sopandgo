#!/bin/sh
# Strip trailing slashes from ORIGIN. If set to e.g. https://app.example.com/, SvelteKit
# can build URLs whose pathname is //login; enhanced forms POST there and get 404.
if [ -n "${ORIGIN:-}" ]; then
	while [ "${ORIGIN%/}" != "$ORIGIN" ]; do
		ORIGIN="${ORIGIN%/}"
	done
	export ORIGIN
fi

# Only "false" skips demo data. Empty or any other value seeds when app.db
# is absent. The seeder binary applies the same rule if invoked directly.
seed_demo=$(printf '%s' "${SEED_DEMO_DATA:-}" | tr '[:upper:]' '[:lower:]' | sed 's/^[[:space:]]*//;s/[[:space:]]*$//')
if [ "$seed_demo" = "false" ]; then
	echo "--- Step 1: Skipping demo data (SEED_DEMO_DATA=false) ---"
else
	echo "--- Step 1: Seeding Data ---"
	./seeder
fi

echo "--- Step 2: Starting Go Backend ---"
./sopandgo &

echo "--- Step 3: Starting SvelteKit Frontend ---"
PORT=3000 node frontend-build/index.js &

echo "--- Step 4: Starting Caddy Proxy ---"
exec caddy run --config /etc/caddy/Caddyfile --adapter caddyfile
