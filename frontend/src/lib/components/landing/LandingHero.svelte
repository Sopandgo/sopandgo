<script lang="ts">
	import Card from '$lib/components/Card.svelte';
	import * as m from '$lib/paraglide/messages.js';
</script>

<section class="hero rounded-box border border-base-300 bg-base-100">
	<div class="hero-content flex-col gap-10 py-14 text-center lg:flex-row lg:items-start lg:py-16 lg:text-left">
		<div class="max-w-xl space-y-4">
			<p class="text-sm font-semibold text-primary">SOP and GO</p>
			<h1 class="text-4xl font-semibold leading-tight sm:text-5xl">{m.landing_headline()}</h1>
			<p class="text-base-content/80 text-lg">
				{m.landing_lead()}
			</p>
			<div class="flex flex-col gap-3 sm:flex-row sm:flex-wrap sm:items-center">
				<a href="/login" class="btn btn-primary btn-lg">{m.page_login()}</a>
				<a
					href="https://github.com/Sopandgo/sopandgo/blob/main/docs/ops/evaluating.md"
					class="btn btn-lg"
					rel="noreferrer"
					target="_blank"
				>
					{m.landing_try()}
				</a>
				<a
					href="https://github.com/Sopandgo/sopandgo"
					class="btn btn-ghost btn-lg"
					rel="noreferrer"
					target="_blank"
				>
					{m.landing_github()}
				</a>
			</div>
			<p class="text-base-content/70 text-sm">
				{m.landing_eval_before()}
				<a
					class="link"
					href="https://github.com/Sopandgo/sopandgo/blob/main/docs/ops/evaluating.md"
					rel="noreferrer"
					target="_blank"
					>{m.landing_eval_link()}</a
				>{m.landing_eval_after()}
			</p>
		</div>
		<Card class="max-w-md">
			<div class="card-body gap-4 text-left">
				<h2 class="card-title text-lg">{m.landing_glance()}</h2>
				<ul class="space-y-3 text-sm">
					<li class="list-row">
						<span class="badge badge-outline badge-sm shrink-0">{m.page_sops()}</span>
						<span>{m.landing_glance_sops()}</span>
					</li>
					<li class="list-row">
						<span class="badge badge-outline badge-sm shrink-0">{m.landing_badge_proof()}</span>
						<span>{m.landing_glance_proof()}</span>
					</li>
					<li class="list-row">
						<span class="badge badge-outline badge-sm shrink-0">{m.landing_badge_host()}</span>
						<span>{m.landing_glance_host()}</span>
					</li>
				</ul>
			</div>
		</Card>
	</div>
</section>

<style>
	.list-row {
		display: flex;
		gap: 0.75rem;
		align-items: flex-start;
	}
</style>
