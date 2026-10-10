<script lang="ts">
    import type { Snippet } from 'svelte';
    import type { ChevronRightIcon } from 'lucide-svelte';

    /*
     * Centred placeholder for a list or card with nothing to show
     * (docs/design/style-guide.md → EmptyState): a large icon tile, a heading,
     * one line of help and the next step. `role="status"` announces it when a
     * filter empties the list, so screen-reader users hear that nothing matched.
     */
    interface Props {
        icon: typeof ChevronRightIcon;
        title: string;
        description?: string;
        /** `primary` for an invitation to create; `neutral` for "nothing here". */
        tone?: 'neutral' | 'primary';
        /** Next steps: at most one `btn-primary`, otherwise default or ghost buttons. */
        actions?: Snippet;
    }

    let { icon: Icon, title, description, tone = 'neutral', actions }: Props = $props();
</script>

<div role="status" class="flex flex-col items-center px-6 py-12 text-center sm:py-16">
    <div
        class="mb-5 flex size-16 items-center justify-center rounded-box ring-8 {tone === 'primary'
            ? 'bg-primary/10 text-primary ring-primary/5'
            : 'bg-base-200 text-base-content/70 ring-base-200/50'}"
        aria-hidden="true"
    >
        <Icon class="size-8" />
    </div>

    <h3 class="text-base font-semibold">{title}</h3>

    {#if description}
        <p class="mt-1 max-w-sm text-sm text-base-content/70">{description}</p>
    {/if}

    {#if actions}
        <div class="mt-6 flex flex-wrap justify-center gap-2">{@render actions()}</div>
    {/if}
</div>
