<section class="space-y-4" aria-labelledby="landing-fit-heading">
	<h2 id="landing-fit-heading" class="text-2xl font-bold">Built for how labs actually work</h2>
	<p class="text-base-content/80 max-w-3xl text-sm leading-relaxed md:text-base">
		sopandgo treats SOPs as structured, versioned documents with explicit acknowledgment,
		auditability, and cryptographic integrity checks — while staying lightweight, transparent, and
		entirely self-hosted. It is for teams that need traceability without taking on a full quality
		management suite.
	</p>
</section>

<section class="grid gap-6 md:grid-cols-2 lg:grid-cols-4" aria-label="Product pillars">
	<div class="card bg-base-100 border-base-300 border shadow-sm">
		<div class="card-body">
			<h3 class="card-title text-base">Simple SOPs</h3>
			<p class="text-base-content/70 text-sm">
				All standard operating procedures in one place. Versions are immutable and searchable.
				Acknowledgments are tied to exact versions — not moving targets after someone edits a file
				share.
			</p>
		</div>
	</div>
	<div class="card bg-base-100 border-base-300 border shadow-sm">
		<div class="card-body">
			<h3 class="card-title text-base">Complete control</h3>
			<p class="text-base-content/70 text-sm">
				No cloud dependency or proprietary format lock-in. Content lives on disk as Markdown and
				assets you can read without the app. Style and structure procedures with Markdown, then
				export, back up, or migrate on your schedule.
			</p>
		</div>
	</div>
	<div class="card bg-base-100 border-base-300 border shadow-sm">
		<div class="card-body">
			<h3 class="card-title text-base">Built for traceability</h3>
			<p class="text-base-content/70 text-sm">
				Tamper-evident, hash-chained audit logs; SHA-256 verification for version and asset files;
				explicit acknowledgment tracking per version with role-based access control enforced on the
				server.
			</p>
		</div>
	</div>
	<div class="card bg-base-100 border-base-300 border shadow-sm">
		<div class="card-body">
			<h3 class="card-title text-base">Simple deployment</h3>
			<p class="text-base-content/70 text-sm">
				Run with Docker Compose: one application image (Caddy, Go API, SvelteKit), optional Gotenberg
				for PDF, one published port, and one data directory. See the deployment guide for the full
				environment reference.
			</p>
		</div>
	</div>
</section>

<section class="card bg-base-100 border-base-300 border shadow-sm" aria-labelledby="landing-security-heading">
	<div class="card-body gap-4">
		<h2 id="landing-security-heading" class="card-title text-xl">Security-first architecture</h2>
		<p class="text-base-content/80 text-sm md:text-base">
			The stack combines fast, stateless verification for routine API calls with server-tracked
			sessions so administrators can revoke access on realistic timelines. Identity stays local —
			invite-based onboarding, bcrypt password hashes, strict rate limits on auth endpoints, and
			role-based permissions evaluated only on the server.
		</p>
		<ul class="list-inside list-disc space-y-1 text-sm text-base-content/80 md:text-base">
			<li>Short-lived access tokens plus refresh tokens tied to SQLite session rows.</li>
			<li>Administrative revocation within roughly five minutes via the refresh boundary.</li>
			<li>Optional SMTP delivery or manual invite links when email is not available.</li>
		</ul>
		<p>
			<a
				class="link link-primary font-medium"
				href="https://github.com/Sopandgo/sopandgo/blob/main/docs/concepts/security-model.md"
				rel="noreferrer"
				target="_blank"
				>Read the security model</a
			>
			for assumptions, guarantees, and limits.
		</p>
	</div>
</section>

<section class="card bg-base-200/80 border-base-300 border shadow-inner" aria-labelledby="landing-data-heading">
	<div class="card-body gap-3">
		<h2 id="landing-data-heading" class="card-title text-xl">Your data stays yours</h2>
		<p class="text-base-content/80 text-sm md:text-base">
			If the application is removed, SOP Markdown and assets remain ordinary files on disk. SQLite
			holds metadata, sessions, audit entries, and acknowledgments — all covered by the built-in
			backup export so you can validate archives and stage restores when you are ready for a
			controlled migration.
		</p>
		<p>
			<a
				class="link link-primary font-medium"
				href="https://github.com/Sopandgo/sopandgo/blob/main/docs/ops/backup-and-restore.md"
				rel="noreferrer"
				target="_blank"
				>Backup and restore documentation</a
			>
		</p>
	</div>
</section>

<section class="space-y-4" aria-labelledby="landing-features-heading">
	<h2 id="landing-features-heading" class="text-2xl font-bold">Capabilities today</h2>
	<p class="text-base-content/70 max-w-3xl text-sm md:text-base">
			The project ships a broad slice of day-two operations work already: from authoring and PDF
			export to admin mail modes and staged restores. Highlights from the current README status:
		</p>
	<div class="grid gap-3 sm:grid-cols-2">
		<ul class="list-inside list-disc space-y-1 text-sm text-base-content/80">
			<li>Authentication, sessions, and password reset flows</li>
			<li>Role-based access control for editors, readers, approvers, and admins</li>
			<li>SOP lifecycle, Markdown editor, and asset uploads</li>
			<li>Version immutability with content hashes</li>
			<li>Signatures and acknowledgment tracking</li>
			<li>Tags, favorites, and list filters</li>
		</ul>
		<ul class="list-inside list-disc space-y-1 text-sm text-base-content/80">
			<li>Audit dashboard with tamper-evident hash chains</li>
			<li>Mail service with SMTP or manual invite links</li>
			<li>Backup export, validation, and staged restore</li>
			<li>PDF export via optional Gotenberg sidecar</li>
			<li>Word import (.docx) into the draft workflow</li>
		</ul>
	</div>
</section>

<section
	class="rounded-box border-base-300 bg-base-100 flex flex-col gap-3 border p-6 shadow-sm sm:flex-row sm:flex-wrap sm:items-center sm:justify-between"
	aria-label="Documentation links"
>
	<p class="text-base-content/80 m-0 max-w-xl text-sm">
		Architecture, deployment variables, and API notes live alongside the code in the repository
		docs.
	</p>
	<div class="flex flex-wrap gap-2">
		<a
			class="btn btn-sm btn-outline"
			href="https://github.com/Sopandgo/sopandgo/blob/main/docs/ops/evaluating.md"
			rel="noreferrer"
			target="_blank"
			>Evaluating</a
		>
		<a
			class="btn btn-sm btn-outline"
			href="https://github.com/Sopandgo/sopandgo/blob/main/docs/dev/architecture.md"
			rel="noreferrer"
			target="_blank"
			>Architecture</a
		>
		<a
			class="btn btn-sm btn-outline"
			href="https://github.com/Sopandgo/sopandgo/blob/main/docs/ops/deployment.md"
			rel="noreferrer"
			target="_blank"
			>Deployment</a
		>
		<a
			class="btn btn-sm btn-outline"
			href="https://github.com/Sopandgo/sopandgo/blob/main/README.md"
			rel="noreferrer"
			target="_blank"
			>README</a
		>
	</div>
</section>
