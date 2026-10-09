<script lang="ts">
    import type { Snippet } from 'svelte';
    import { CircleCheckIcon, CircleXIcon, InfoIcon, TriangleAlertIcon } from 'lucide-svelte';

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

    const typeClasses: Record<AlertType, string> = {
        info: 'alert-info',
        success: 'alert-success',
        warning: 'alert-warning',
        error: 'alert-error'
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
        {#if children}
            {@render children()}
        {:else if message}
            <span class="whitespace-pre-wrap">{message}</span>
        {/if}
    </div>
</div>
