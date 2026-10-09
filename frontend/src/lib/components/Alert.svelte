<script lang="ts">
    import type { Snippet } from 'svelte';
    import { CircleCheckIcon, CircleXIcon, InfoIcon, TriangleAlertIcon } from 'lucide-svelte';

    type Variant = 'info' | 'success' | 'warning' | 'error' | 'neutral';

    interface Props {
        variant?: Variant;
        /** Plain text body; use `children` for markup (links, code). */
        message?: string;
        children?: Snippet;
        /** Defaults to `alert` for errors and warnings, `status` otherwise. */
        role?: 'alert' | 'status' | 'note';
        /** Tinted background instead of solid. */
        soft?: boolean;
        /** Smaller text and icon for alerts inside sidebars and lists. */
        compact?: boolean;
        class?: string;
    }

    let {
        variant = 'info',
        message,
        children,
        role,
        soft = false,
        compact = false,
        class: extraClass = ''
    }: Props = $props();

    const variantClasses: Record<Variant, string> = {
        info: 'alert-info',
        success: 'alert-success',
        warning: 'alert-warning',
        error: 'alert-error',
        neutral: ''
    };

    const icons = {
        info: InfoIcon,
        success: CircleCheckIcon,
        warning: TriangleAlertIcon,
        error: CircleXIcon,
        neutral: null
    };

    const Icon = $derived(icons[variant]);
    const resolvedRole = $derived(role ?? (variant === 'error' || variant === 'warning' ? 'alert' : 'status'));
</script>

<div
    role={resolvedRole}
    class="alert {variantClasses[variant]} {soft ? 'alert-soft' : ''} {compact ? 'py-2 text-xs' : ''} {extraClass}"
>
    {#if Icon}
        <Icon class="{compact ? 'h-4 w-4' : 'h-6 w-6'} shrink-0" aria-hidden="true" />
    {/if}

    {#if children}
        <div class="min-w-0">{@render children()}</div>
    {:else}
        <span class="whitespace-pre-wrap">{message}</span>
    {/if}
</div>
