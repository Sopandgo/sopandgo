<script lang="ts">
    import { getInitials } from '$lib/utils/getInitials';

    type Color = 'neutral' | 'primary' | 'secondary' | 'accent' | 'info' | 'error' | 'warning';
    type Size = 'sm' | 'md' | 'lg';

    interface Props {
        displayName?: string;
        color?: Color;
        size?: Size;
    }

    let { displayName = '', color = 'neutral', size = 'md' }: Props = $props();

    const sizeClasses: Record<Size, string> = {
        sm: 'w-8',
        md: 'w-10',
        lg: 'w-12'
    };

    const colorClasses: Record<Color, string> = {
        neutral: 'bg-neutral text-neutral-content',
        primary: 'bg-primary text-primary-content',
        secondary: 'bg-secondary text-secondary-content',
        accent: 'bg-accent text-accent-content',
        info: 'bg-info text-info-content',
        error: 'bg-error text-error-content',
        warning: 'bg-warning text-warning-content'
    };
    
    const selectedColorClass = $derived(colorClasses[color]);
    const selectedSizeClass = $derived(sizeClasses[size]);
</script>

<div class="avatar avatar-placeholder">
    <div class={`rounded-full ${selectedColorClass} ${selectedSizeClass}`}>
        <span>{getInitials(displayName)}</span>
    </div>
</div>