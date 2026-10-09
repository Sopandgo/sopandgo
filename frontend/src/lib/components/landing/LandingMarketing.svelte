<script lang="ts">
	import Card from '$lib/components/Card.svelte';
	import * as m from '$lib/paraglide/messages.js';
</script>

<section class="space-y-4" aria-labelledby="landing-fit-heading">
	<h2 id="landing-fit-heading" class="text-2xl font-bold">{m.landing_fit()}</h2>
	<p class="text-base-content/80 max-w-3xl text-sm leading-relaxed md:text-base">
		{m.landing_fit_body()}
	</p>
</section>

<section class="grid gap-6 md:grid-cols-2 lg:grid-cols-4" aria-label={m.landing_pillars()}>
	<Card variant="flat" class="border-base-300">
		<div class="card-body">
			<h3 class="card-title text-base">{m.landing_simple()}</h3>
			<p class="text-base-content/70 text-sm">
				{m.landing_simple_body()}
			</p>
		</div>
	</Card>
	<Card variant="flat" class="border-base-300">
		<div class="card-body">
			<h3 class="card-title text-base">{m.landing_control()}</h3>
			<p class="text-base-content/70 text-sm">
				{m.landing_control_body()}
			</p>
		</div>
	</Card>
	<Card variant="flat" class="border-base-300">
		<div class="card-body">
			<h3 class="card-title text-base">{m.landing_trace()}</h3>
			<p class="text-base-content/70 text-sm">
				{m.landing_trace_body()}
			</p>
		</div>
	</Card>
	<Card variant="flat" class="border-base-300">
		<div class="card-body">
			<h3 class="card-title text-base">{m.landing_deploy()}</h3>
			<p class="text-base-content/70 text-sm">
				{m.landing_deploy_body()}
			</p>
		</div>
	</Card>
</section>

<Card as="section" variant="flat" class="border-base-300" aria-labelledby="landing-security-heading">
	<div class="card-body gap-4">
		<h2 id="landing-security-heading" class="card-title text-xl">{m.landing_security()}</h2>
		<p class="text-base-content/80 text-sm md:text-base">
			{m.landing_security_body()}
		</p>
		<ul class="list-inside list-disc space-y-1 text-sm text-base-content/80 md:text-base">
			<li>{m.landing_security_1()}</li>
			<li>{m.landing_security_2()}</li>
			<li>{m.landing_security_3()}</li>
		</ul>
		<p>
			<a
				class="link link-primary font-medium"
				href="https://github.com/Sopandgo/sopandgo/blob/main/docs/concepts/security-model.md"
				rel="noreferrer"
				target="_blank"
				>{m.landing_security_link()}</a
			>
			{m.landing_security_after()}
		</p>
	</div>
</Card>

<Card as="section" variant="inset" class="border-base-300" aria-labelledby="landing-data-heading">
	<div class="card-body gap-3">
		<h2 id="landing-data-heading" class="card-title text-xl">{m.landing_data()}</h2>
		<p class="text-base-content/80 text-sm md:text-base">
			{m.landing_data_body()}
		</p>
		<p>
			<a
				class="link link-primary font-medium"
				href="https://github.com/Sopandgo/sopandgo/blob/main/docs/ops/backup-and-restore.md"
				rel="noreferrer"
				target="_blank"
				>{m.landing_backup_link()}</a
			>
		</p>
	</div>
</Card>

<section class="space-y-4" aria-labelledby="landing-features-heading">
	<h2 id="landing-features-heading" class="text-2xl font-bold">{m.landing_capabilities()}</h2>
	<p class="text-base-content/70 max-w-3xl text-sm md:text-base">
			{m.landing_capabilities_body()}
		</p>
	<div class="grid gap-3 sm:grid-cols-2">
		<ul class="list-inside list-disc space-y-1 text-sm text-base-content/80">
			<li>{m.landing_cap_auth()}</li>
			<li>{m.landing_cap_roles()}</li>
			<li>{m.landing_cap_lifecycle()}</li>
			<li>{m.landing_cap_hashes()}</li>
			<li>{m.landing_cap_signatures()}</li>
			<li>{m.landing_cap_tags()}</li>
		</ul>
		<ul class="list-inside list-disc space-y-1 text-sm text-base-content/80">
			<li>{m.landing_cap_audit()}</li>
			<li>{m.landing_cap_mail()}</li>
			<li>{m.landing_cap_backup()}</li>
			<li>{m.landing_cap_pdf()}</li>
			<li>{m.landing_cap_word()}</li>
		</ul>
	</div>
</section>

<section
	class="rounded-box border-base-300 bg-base-100 flex flex-col gap-3 border p-6 shadow-sm sm:flex-row sm:flex-wrap sm:items-center sm:justify-between"
	aria-label={m.landing_docs_aria()}
>
	<p class="text-base-content/80 m-0 max-w-xl text-sm">
		{m.landing_docs_body()}
	</p>
	<div class="flex flex-wrap gap-2">
		<a
			class="btn btn-sm btn-outline"
			href="https://github.com/Sopandgo/sopandgo/blob/main/docs/ops/evaluating.md"
			rel="noreferrer"
			target="_blank"
			>{m.landing_docs_evaluating()}</a
		>
		<a
			class="btn btn-sm btn-outline"
			href="https://github.com/Sopandgo/sopandgo/blob/main/docs/dev/architecture.md"
			rel="noreferrer"
			target="_blank"
			>{m.landing_docs_architecture()}</a
		>
		<a
			class="btn btn-sm btn-outline"
			href="https://github.com/Sopandgo/sopandgo/blob/main/docs/ops/deployment.md"
			rel="noreferrer"
			target="_blank"
			>{m.landing_docs_deployment()}</a
		>
		<a
			class="btn btn-sm btn-outline"
			href="https://github.com/Sopandgo/sopandgo/blob/main/README.md"
			rel="noreferrer"
			target="_blank"
			>{m.landing_readme()}</a
		>
	</div>
</section>
