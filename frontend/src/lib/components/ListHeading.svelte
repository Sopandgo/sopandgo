<script lang="ts">
    import type { Snippet } from 'svelte';

    /* Group heading above rows in a `<ul class="list">`: label style, muted, sentence case. */
    let { children }: { children: Snippet } = $props();
</script>

<li class="border-b border-base-300 px-4 py-3 text-sm font-medium text-base-content/70">
    {@render children()}
</li>
