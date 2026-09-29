<script lang="ts">
	let { children } = $props();
</script>

<div class="flex min-h-0 w-full flex-1 flex-col items-center justify-center py-8">
	{@render children()}
</div>
