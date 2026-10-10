<script lang="ts">
    import type { Snippet } from 'svelte';
    import type { HTMLAttributes } from 'svelte/elements';

    /*
     * The bordered base-100 section that holds every block of content
     * (docs/design/style-guide.md → Card). Flat: border, no shadow.
     * daisyUI's card-border draws in base-200, which vanishes on the base-200
     * page, so the border is set here. `class` is for layout only.
     */
    interface Props extends HTMLAttributes<HTMLElement> {
        as?: 'div' | 'section' | 'article' | 'figure' | 'li';
        /** Optional header: section-title heading. */
        title?: string;
        /** One line under the title, in muted text. */
        description?: string;
        /** Header id, so a wrapping region can use aria-labelledby. */
        headingId?: string;
        /** Header actions, aligned right. */
        actions?: Snippet;
        class?: string;
        children: Snippet;
    }

    let {
        as = 'div',
        title,
        description,
        headingId,
        actions,
        class: extraClass = '',
        children,
        ...rest
    }: Props = $props();
</script>

<svelte:element this={as} class="card border border-base-300 bg-base-100 {extraClass}" {...rest}>
    {#if title || actions}
        <div
            class="flex flex-col gap-3 border-b border-base-300 p-4 sm:flex-row sm:items-start sm:justify-between sm:p-6"
        >
            <div class="min-w-0">
                {#if title}
                    <h2 id={headingId} class="text-lg font-semibold">{title}</h2>
                {/if}
                {#if description}
                    <p class="mt-1 text-sm text-base-content/70">{description}</p>
                {/if}
            </div>
            {#if actions}
                <div class="flex shrink-0 flex-wrap gap-2">{@render actions()}</div>
            {/if}
        </div>
    {/if}
    {@render children()}
</svelte:element>
