<script lang="ts">
    import { brandLogoUrl } from '#lib/branding.js';
    import * as m from '#lib/paraglide/messages.js';
    import type { Branding } from '#lib/sdk/types.js';

    /*
     * The organization's logo, or the SOPandGO gopher. The box is set by
     * `class` (e.g. size-8); a wide logo shrinks to fit instead of stretching.
     */
    let {
        branding,
        alt = '',
        class: className = ''
    }: { branding: Branding | null | undefined; alt?: string; class?: string } = $props();

    const src = $derived(brandLogoUrl(branding));
</script>

{#if src}
    <img {src} {alt} class="object-contain {className}" />
{:else}
    <img src="/favicon.svg" {alt} title={m.footer_gopher()} class={className} />
{/if}
