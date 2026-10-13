<script lang="ts">
  import { page } from '$app/state';
  import BrandLogo from '#lib/components/BrandLogo.svelte';
  import { brandName, DEFAULT_BRAND_NAME, isCustomBranding } from '#lib/branding.js';
  import * as m from '#lib/paraglide/messages.js';
  import { PRODUCT_URL, REPO_URL, versionLabel, versionUrl } from '#lib/product.js';

  const version = versionLabel(__APP_VERSION__);
  const versionHref = versionUrl(__APP_VERSION__);

  const branded = $derived(isCustomBranding(page.data.branding));
</script>

<footer class="footer sm:footer-horizontal items-center border-t border-base-300 bg-base-100 p-4">
  <aside class="grid-flow-col items-center">
    <BrandLogo
      branding={page.data.branding}
      alt={m.footer_logo_alt({ name: brandName(page.data.branding) })}
      class="h-12 w-12"
    />
    <p>
      {#if branded}
        <span class="font-semibold">{brandName(page.data.branding)}</span>
      {:else}
        <a href={PRODUCT_URL} target="_blank" rel="noopener noreferrer" class="font-semibold underline"
          >{DEFAULT_BRAND_NAME}</a
        >
      {/if}
      <br />
      {#if branded}
        <!-- Attribution stays when a lab brands the app. -->
        <a href={PRODUCT_URL} target="_blank" rel="noopener noreferrer" class="underline"
          >{m.footer_powered_by({ product: DEFAULT_BRAND_NAME })}</a
        > ·
      {/if}
      © {new Date().getFullYear()} Leopold Dürrauer ·
      <a href={versionHref} target="_blank" rel="noopener noreferrer" class="underline">{version}</a>
      · {m.footer_license()}
    </p>
  </aside>
  <nav class="grid-flow-col gap-4 md:place-self-center md:justify-self-end">
    <a
      href="https://duerrauer.com"
      target="_blank"
      rel="noopener noreferrer"
      aria-label={m.footer_author_site()}
    >
       <img
        src="/duerrauer.svg"
        alt={m.footer_duerrauer_alt()}
        class="h-8 w-8"
      />
    </a>
    <a
      href={REPO_URL}
      target="_blank"
      rel="noopener noreferrer"
      aria-label={m.footer_github()}
    >
       <img
        src="/github.svg"
        alt={m.footer_github_alt()}
        class="h-8 w-8"
      />
    </a>    
  </nav>
</footer>
