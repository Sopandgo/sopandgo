<script lang="ts">
    interface Props {
        children: import('svelte').Snippet;
        color?: 'primary' | 'secondary' | 'accent' | 'success' | 'warning' | 'error';
    }

    let { children, color = 'primary' }: Props = $props();

    // Mapping ensures Tailwind's scanner sees the full class names
    const colorClasses = {
        primary: 'text-primary',
        secondary: 'text-secondary',
        accent: 'text-accent',
        success: 'text-success',
        warning: 'text-warning',
        error: 'text-error'
    };
</script>

<h1 class="card-title text-3xl font-bold flex gap-x-2 mb-2 {colorClasses[color]}">
    {@render children()}
</h1>