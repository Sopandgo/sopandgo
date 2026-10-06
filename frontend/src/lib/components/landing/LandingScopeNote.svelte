<script>
	import * as m from '$lib/paraglide/messages.js';
</script>

<div role="note" class="alert alert-info shadow-sm">
	<svg
		xmlns="http://www.w3.org/2000/svg"
		class="h-6 w-6 shrink-0 stroke-current"
		fill="none"
		viewBox="0 0 24 24"
		aria-hidden="true"
	>
		<path
			stroke-linecap="round"
			stroke-linejoin="round"
			stroke-width="2"
			d="M13 16h-1v-4h-1m1-4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z"
		/>
	</svg>
	<div class="text-sm">
		<strong class="font-semibold">{m.landing_scope_strong()}</strong>
		{m.landing_scope_body()}
		<a
			class="link font-medium"
			href="https://github.com/Sopandgo/sopandgo/blob/main/README.md"
			rel="noreferrer"
			target="_blank"
			>{m.landing_readme()}</a
		>
		{m.landing_scope_after()}
	</div>
</div>
