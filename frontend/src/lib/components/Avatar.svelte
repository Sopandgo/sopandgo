<script>
    import { getInitials } from '$lib/utils/getInitials';

    /** @type {{ displayName?: string, color?: 'neutral' | 'primary' | 'secondary' | 'accent' | 'info' | 'error' | 'warning' , size?: 'sm' | 'md' | 'lg' }} */
    let { displayName = '', color = 'neutral', size = 'md' } = $props();

    const sizeClasses = {
        sm: 'w-8',
        md: 'w-10',
        lg: 'w-12'
    };

    const colorClasses = {
        neutral: 'bg-neutral text-neutral-content',
        primary: 'bg-primary text-primary-content',
        secondary: 'bg-secondary text-secondary-content',
        accent: 'bg-accent text-accent-content',
        info: 'bg-info text-info-content',
        error: 'bg-error text-error-content',
        warning: 'bg-warning text-warning-content'
    };
    
    const selectedColorClass = $derived(colorClasses[color] || colorClasses.neutral);
    const selectedSizeClass = $derived(sizeClasses[size] || sizeClasses.md);
</script>

<div class="avatar avatar-placeholder">
    <div class={`rounded-full ${selectedColorClass} ${selectedSizeClass}`}>
        <span>{getInitials(displayName)}</span>
    </div>
</div>