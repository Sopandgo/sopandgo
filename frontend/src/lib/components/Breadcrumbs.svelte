<script lang="ts">
  import { ChevronRightIcon } from 'lucide-svelte';
  import type { Crumb } from '$lib/breadcrumbs';
  import * as m from '$lib/paraglide/messages.js';

  let { items = [] }: { items: Crumb[] } = $props();
</script>

<!-- Phones keep the last two crumbs: the page and the way back to its parent. -->
{#if items.length}
  <nav aria-label={m.nav_breadcrumb()} class="min-w-0">
    <ol class="flex min-w-0 items-center text-sm">
      {#each items as { label, href }, i}
        {@const last = i === items.length - 1}
        <li class="flex min-w-0 items-center {i < items.length - 2 ? 'max-sm:hidden' : ''}">
          {#if i > 0}
            <ChevronRightIcon
              size={16}
              aria-hidden="true"
              class="mx-1 shrink-0 text-base-content/70 rtl:rotate-180 {i === items.length - 2 ? 'max-sm:hidden' : ''}"
            />
          {/if}
          {#if href && !last}
            <a {href} title={label} class="truncate text-base-content/70 underline">{label}</a>
          {:else if last}
            <span title={label} class="truncate font-medium" aria-current="page">{label}</span>
          {:else}
            <span title={label} class="truncate text-base-content/70">{label}</span>
          {/if}
        </li>
      {/each}
    </ol>
  </nav>
{/if}
