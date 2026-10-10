<script lang="ts">
    import type { Snippet } from 'svelte';
    import { ChevronRightIcon } from 'lucide-svelte';
    import Card from '$lib/components/Card.svelte';

    /*
     * A Card whose body collapses behind a header row (docs/design/style-guide.md →
     * Card → Collapsible). The title is a disclosure button stretched over the row;
     * `trailing` sits above it, so a status badge or a control (toggle, small
     * button) stays usable while collapsed. Collapsed by default.
     */
    interface Props {
        title: string;
        /** One line under the title, in muted text. */
        meta?: string;
        /** Right side of the header: a status badge or a control. */
        trailing?: Snippet;
        /** Shown under the header even while collapsed, e.g. a failed header action. */
        notice?: Snippet;
        /** Opens the card when it becomes true, e.g. when a form inside returned a result. */
        openWhen?: boolean;
        children: Snippet;
    }

    let { title, meta, trailing, notice, openWhen = false, children }: Props = $props();

    const uid = $props.id();
    const shouldOpen = () => openWhen;

    let open = $state(shouldOpen());
    $effect(() => {
        if (shouldOpen()) open = true;
    });
</script>

<Card>
    <div
        class="relative flex items-center gap-3 rounded-box p-4 transition-colors duration-150 hover:bg-base-200 sm:p-6 {open
            ? 'rounded-b-none border-b border-base-300'
            : ''}"
    >
        <ChevronRightIcon
            class="size-5 shrink-0 text-base-content/70 transition-transform duration-150 motion-reduce:transition-none {open
                ? 'rotate-90'
                : ''}"
            aria-hidden="true"
        />
        <div class="min-w-0 flex-1">
            <h2 class="text-lg font-semibold">
                <!-- Stretched over the row; its focus ring is drawn inset so the card does not clip it. -->
                <button
                    type="button"
                    id="{uid}-header"
                    class="cursor-pointer text-left after:absolute after:inset-0 after:rounded-box focus-visible:outline-none focus-visible:after:outline-2 focus-visible:after:-outline-offset-2 focus-visible:after:outline-primary"
                    aria-expanded={open}
                    aria-controls="{uid}-body"
                    onclick={() => (open = !open)}
                >
                    {title}
                </button>
            </h2>
            {#if meta}
                <p class="mt-1 truncate text-sm text-base-content/70">{meta}</p>
            {/if}
        </div>
        {#if trailing}
            <div class="relative z-10 flex shrink-0 items-center gap-2">{@render trailing()}</div>
        {/if}
    </div>

    {#if notice}
        <div class="px-4 pt-4 sm:px-6 {open ? '' : 'pb-4 sm:pb-6'}">{@render notice()}</div>
    {/if}

    <!-- A plain wrapper takes `hidden`: card-body sets display:flex, which would override it. -->
    <div id="{uid}-body" role="region" aria-labelledby="{uid}-header" hidden={!open}>
        <div class="card-body space-y-4">
            {@render children()}
        </div>
    </div>
</Card>
