<script lang="ts">
    import type { Snippet } from 'svelte';
    import { CircleCheckIcon, CircleXIcon, InfoIcon, TriangleAlertIcon } from '@lucide/svelte';

    /*
     * Inline message about the page or a form (docs/design/style-guide.md → Alert).
     * Always soft: status text on its soft tint, never a solid fill.
     */
    type AlertType = 'info' | 'success' | 'warning' | 'error';

    interface Props {
        type?: AlertType;
        /** Short bold lead line. */
        title?: string;
        /** Plain text body; use `children` for markup (links, code). */
        message?: string;
        children?: Snippet;
        /** Defaults to `alert` for errors and warnings, `status` otherwise. */
        role?: 'alert' | 'status' | 'note';
        /** Smaller padding for alerts inside sidebars and lists. */
        compact?: boolean;
        class?: string;
    }

    let {
        type = 'info',
        title,
        message,
        children,
        role,
        compact = false,
        class: extraClass = ''
    }: Props = $props();

    // daisyUI's soft tint alone nearly matches the base-200 page, so every alert adds a
    // stronger tint and a status border; it reads the same in a card and on the page.
    const typeClasses: Record<AlertType, string> = {
        info: 'alert-info border-info/40 bg-info/10',
        success: 'alert-success border-success/40 bg-success/10',
        warning: 'alert-warning border-warning/40 bg-warning/10',
        error: 'alert-error border-error/40 bg-error/10'
    };

    const icons = {
        info: InfoIcon,
        success: CircleCheckIcon,
        warning: TriangleAlertIcon,
        error: CircleXIcon
    };

    const Icon = $derived(icons[type]);
    const resolvedRole = $derived(role ?? (type === 'error' || type === 'warning' ? 'alert' : 'status'));
</script>

<div
    role={resolvedRole}
    class="alert alert-soft {typeClasses[type]} items-start text-sm {compact ? 'py-2' : ''} {extraClass}"
>
    <Icon class="mt-0.5 size-4 shrink-0" aria-hidden="true" />
    <div class="min-w-0">
        {#if title}
            <div class="font-medium">{title}</div>
        {/if}
        <!-- Icon and title carry the status colour; the body reads in the text colour -->
        <div class="text-base-content">
            {#if children}
                {@render children()}
            {:else if message}
                <span class="whitespace-pre-wrap">{message}</span>
            {/if}
        </div>
    </div>
</div>
