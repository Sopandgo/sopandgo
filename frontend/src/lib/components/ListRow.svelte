<script lang="ts">
    import type { Snippet } from 'svelte';
    import { ChevronRightIcon } from 'lucide-svelte';

    /*
     * One item in a `<ul class="list">` inside a Card (docs/design/style-guide.md → ListRow).
     * With `href`, the title is a stretched link: the whole row is clickable, while
     * `trailing` controls and anything marked `relative z-10` keep their own clicks
     * (no interactive elements nested inside <a>).
     */
    interface Props {
        title: string;
        href?: string;
        /** Accessible name for the link when the title alone is ambiguous. */
        linkLabel?: string;
        /** Muted line under the title; a snippet when parts need `font-mono` (IDs, versions). */
        meta?: string | Snippet;
        icon?: typeof ChevronRightIcon;
        /** Needs the reader's action: warning-soft row, warning icon, reason in `meta`. */
        attention?: boolean;
        /** Replaces the icon tile entirely (e.g. an Avatar). */
        leading?: Snippet;
        /** Body content between the title and the meta line. */
        children?: Snippet;
        /** Inline content after the meta, e.g. tag buttons (mark them `relative z-10`). */
        metaExtra?: Snippet;
        /** Status badges and small actions at the end of the row. */
        trailing?: Snippet;
    }

    let {
        title,
        href,
        linkLabel,
        meta,
        icon: Icon,
        attention = false,
        leading,
        children,
        metaExtra,
        trailing
    }: Props = $props();
</script>

<li
    class="list-row relative flex items-center gap-3 rounded-none [ul:last-child>&:last-child]:rounded-b-box border-b border-base-300 transition-colors duration-150 after:hidden last:border-b-0 {attention ? 'bg-warning/10' : ''} {href ? 'hover:bg-base-200' : ''}"
>
    {#if leading}
        <div class="shrink-0">{@render leading()}</div>
    {:else if Icon}
        <div
            class="flex size-10 shrink-0 items-center justify-center rounded-field bg-base-200 {attention ? 'text-warning' : 'text-base-content/70'}"
            aria-hidden="true"
        >
            <Icon class="size-5" />
        </div>
    {/if}

    <div class="min-w-0 flex-1">
        {#if href}
            <!-- Stretched link; its focus ring is drawn inset on the row so the card does not clip it. -->
            <a
                {href}
                aria-label={linkLabel}
                class="block truncate text-sm font-medium after:absolute after:inset-0 after:rounded-field hover:underline focus-visible:outline-none focus-visible:after:outline-2 focus-visible:after:-outline-offset-2 focus-visible:after:outline-primary"
            >
                {title}
            </a>
        {:else}
            <div class="truncate text-sm font-medium">{title}</div>
        {/if}

        {#if children}
            <div class="mt-0.5 text-sm">{@render children()}</div>
        {/if}

        {#if meta || metaExtra}
            <div class="mt-0.5 flex flex-wrap items-center gap-2 text-xs text-base-content/70">
                {#if typeof meta === 'function'}
                    <span>{@render meta()}</span>
                {:else if meta}
                    <span>{meta}</span>
                {/if}
                {#if metaExtra}{@render metaExtra()}{/if}
            </div>
        {/if}
    </div>

    {#if trailing}
        <!-- Badges let clicks through to the row link; controls stay clickable. -->
        <div
            class="pointer-events-none relative z-10 flex shrink-0 items-center gap-2 [&_a]:pointer-events-auto [&_button]:pointer-events-auto [&_input]:pointer-events-auto"
        >
            {@render trailing()}
        </div>
    {/if}

    {#if href}
        <ChevronRightIcon class="size-5 shrink-0 text-base-content/70" aria-hidden="true" />
    {/if}
</li>
