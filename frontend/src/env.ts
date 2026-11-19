import { defineEnvVars } from '@sveltejs/kit/env';

export const variables = defineEnvVars({
	// Baked at frontend build time (Dockerfile sets http://127.0.0.1:8080).
	// Schema default keeps vitest/CI working when BACKEND_URL is unset (no .env in CI).
	BACKEND_URL: {
		static: true,
		schema: (input) => input ?? 'http://127.0.0.1:8080'
	},
	// Read-only public demo. Only "true" (any case) turns it on; see $lib/demo.
	DEMO_MODE: {
		schema: (input) => input ?? ''
	},
	// Runtime toggle; empty/unset means enabled (same as previous $env/dynamic/private default).
	PDF_EXPORT_ENABLED: {
		schema: (input) => (input === undefined || input === '' ? 'true' : input)
	}
});
