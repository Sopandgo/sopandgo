<script lang="ts">
    import type { Snippet } from 'svelte';

    /*
     * The page title (style guide: page-title, one per page): 24px semibold in
     * body colour. A leading icon is sized and muted here so pages need not style it.
     */
    let { children }: { children: Snippet } = $props();
</script>

<h1 class="flex items-center gap-2 text-2xl font-semibold [&>svg]:size-6 [&>svg]:shrink-0 [&>svg]:text-base-content/70">
    {@render children()}
</h1>
