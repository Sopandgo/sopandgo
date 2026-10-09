<script lang="ts">
    import type { Snippet } from 'svelte';
    import { ChevronRightIcon } from 'lucide-svelte';

    /*
     * One row inside a `<ul class="list">`.
     * With `href`, the title is a stretched link: the whole row is clickable,
     * while `trailing` actions and anything marked `relative z-10` stay on top
     * and keep their own clicks (no interactive elements nested inside <a>).
     */
    type IconTone = 'neutral' | 'secondary' | 'warning' | 'plain';

    interface Props {
        title: string;
        href?: string;
        /** Accessible name for the link when the title alone is ambiguous. */
        linkLabel?: string;
        /** Small mono line under the title (id, version, date). */
        meta?: string;
        icon?: typeof ChevronRightIcon;
        iconTone?: IconTone;
        tone?: 'default' | 'warning';
        /** Show a chevron after the row content (only with `href`). */
        chevron?: boolean;
        /** Replaces the icon box entirely. */
        leading?: Snippet;
        /** Body content between the title and the meta line. */
        children?: Snippet;
        /** Inline content after the meta text, e.g. tag buttons. */
        metaExtra?: Snippet;
        /** Actions and badges at the end of the row. */
        trailing?: Snippet;
    }

    let {
        title,
        href,
        linkLabel,
        meta,
        icon: Icon,
        iconTone = 'neutral',
        tone = 'default',
        chevron = false,
        leading,
        children,
        metaExtra,
        trailing
    }: Props = $props();

    const iconToneClasses: Record<IconTone, string> = {
        neutral: 'bg-base-300 text-base-content',
        secondary: 'bg-secondary text-secondary-content',
        warning: 'bg-warning/20 text-warning',
        plain: ''
    };

    const rowTone = $derived(tone === 'warning' ? 'bg-warning/5 hover:bg-warning/10' : 'hover:bg-base-200/50');
</script>

<li class="list-row relative flex items-center gap-3 transition-colors group {rowTone}">
    {#if leading}
        <div class="shrink-0" aria-hidden="true">{@render leading()}</div>
    {:else if Icon && iconTone === 'plain'}
        <Icon class="size-5 shrink-0 opacity-40" aria-hidden="true" />
    {:else if Icon}
        <div class="avatar avatar-placeholder shrink-0" aria-hidden="true">
            <div class="flex h-10 w-10 items-center justify-center rounded-field sm:h-12 sm:w-12 {iconToneClasses[iconTone]}">
                <Icon class="h-5 w-5" />
            </div>
        </div>
    {/if}

    <div class="min-w-0 flex-1">
        {#if href}
            <a
                {href}
                aria-label={linkLabel}
                class="block truncate text-sm font-bold group-hover:text-primary sm:text-base after:absolute after:inset-0 focus-visible:outline-none focus-visible:after:outline-2 focus-visible:after:outline-primary focus-visible:after:rounded-box"
            >
                {title}
            </a>
        {:else}
            <div class="truncate text-sm font-bold sm:text-base">{title}</div>
        {/if}

        {#if children}
            <div class="mt-0.5 text-sm text-base-content/80">{@render children()}</div>
        {/if}

        {#if meta || metaExtra}
            <div class="mt-0.5 flex flex-wrap items-center gap-2">
                {#if meta}
                    <span class="font-mono text-[10px] uppercase tracking-tighter opacity-50">{meta}</span>
                {/if}
                {#if metaExtra}{@render metaExtra()}{/if}
            </div>
        {/if}
    </div>

    {#if trailing}
        <!-- Badges let clicks through to the row link; controls stay clickable. -->
        <div
            class="pointer-events-none relative z-10 flex shrink-0 items-center gap-1 [&_a]:pointer-events-auto [&_button]:pointer-events-auto [&_input]:pointer-events-auto"
        >
            {@render trailing()}
        </div>
    {/if}

    {#if href && chevron}
        <ChevronRightIcon class="h-5 w-5 shrink-0 opacity-40" aria-hidden="true" />
    {/if}
</li>
