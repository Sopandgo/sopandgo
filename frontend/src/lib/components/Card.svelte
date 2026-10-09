<script lang="ts">
    import type { Snippet } from 'svelte';
    import type { HTMLAttributes } from 'svelte/elements';

    /*
     * Variants own background, shadow and border so that `class` only adds
     * layout (width, margin, height) and never fights over the same property.
     * - raised: default page card
     * - flat: secondary panels and marketing blocks
     * - inset: recessed panel on a base-200 surface
     * - subtle: tile inside another card
     */
    type Variant = 'raised' | 'flat' | 'inset' | 'subtle';

    interface Props extends HTMLAttributes<HTMLElement> {
        as?: 'div' | 'section' | 'article' | 'figure' | 'li' | 'form';
        variant?: Variant;
        bordered?: boolean;
        class?: string;
        children: Snippet;
    }

    let {
        as = 'div',
        variant = 'raised',
        bordered = true,
        class: extraClass = '',
        children,
        ...rest
    }: Props = $props();

    const variantClasses: Record<Variant, string> = {
        raised: 'bg-base-100 shadow-md',
        flat: 'bg-base-100 shadow-sm',
        inset: 'bg-base-200 shadow-inner',
        subtle: 'bg-base-200/30 shadow-none transition-colors hover:bg-base-200/50'
    };
</script>

<svelte:element
    this={as}
    class="card {bordered ? 'card-border' : ''} {variantClasses[variant]} {extraClass}"
    {...rest}
>
    {@render children()}
</svelte:element>
