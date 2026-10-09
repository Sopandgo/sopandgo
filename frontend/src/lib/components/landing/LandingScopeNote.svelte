<script lang="ts">
	import Alert from '$lib/components/Alert.svelte';
	import * as m from '$lib/paraglide/messages.js';
</script>

<Alert variant="info" role="note" class="shadow-sm">
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
</Alert>
