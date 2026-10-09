<script lang="ts">
  import {
    ChevronRightIcon,
    FileTextIcon,
    ChevronLeftIcon,
    StarIcon
  } from 'lucide-svelte';
  import type { Snippet } from 'svelte';
  import type { SOPListResponse } from '$lib/sdk/types';
  import Card from './Card.svelte';
  import ListRow from './ListRow.svelte';
  import { goto } from '$app/navigation';
  import { enhance } from '$app/forms';
  import * as m from '$lib/paraglide/messages.js';
  import { getLocale } from '$lib/paraglide/runtime';

  interface Props {
    items?: SOPListResponse | null;
    activeFilters?: {
      limit: number;
      offset: number;
      tag_id: string;
      q: string;
      favorites_only: boolean;
      favorites_first: boolean;
    };
    onTagClick?: (tagId: string) => void; // parent owns filter logic
    /** Search and filter controls, shown above the rows in the same card. */
    toolbar?: Snippet;
  }

  let {
    items = { sops: [], total: 0 },
    activeFilters = {
      limit: 10,
      offset: 0,
      tag_id: '',
      q: '',
      favorites_only: false,
      favorites_first: false
    },
    onTagClick = () => {},
    toolbar
  }: Props = $props();

  let sops = $derived(items?.sops ?? []);
  let total = $derived(items?.total ?? 0);

  let currentPage = $derived(Math.floor(activeFilters.offset / activeFilters.limit) + 1);
  let totalPages = $derived(Math.ceil(total / activeFilters.limit));

  function changePage(newOffset: number) {
    const url = new URL(window.location.href);
    url.searchParams.set('offset', newOffset.toString());
    goto(url.toString());
  }

  function rowHref(sopId: string) {
    return `/sops/${sopId}/v/latest`;
  }
</script>

<Card>
  <!-- One-row header (count on the right at every width) to leave room for the rows -->
  <div class="flex items-baseline justify-between gap-3 border-b border-base-300 p-4 sm:p-6">
    <h2 class="text-lg font-semibold">{m.sops_list_title()}</h2>
    <span class="text-sm text-base-content/70">
      {m.sops_showing({ shown: String(sops.length), total: String(total) })}
    </span>
  </div>

  {#if toolbar}
    <div class="flex flex-col gap-4 border-b border-base-300 p-4 sm:p-6">
      {@render toolbar()}
    </div>
  {/if}

  <ul class="list">
    {#each sops as sop (sop.id)}
      <ListRow
        href={rowHref(sop.id)}
        title={sop.title}
        linkLabel={m.sops_view_latest({ title: sop.title })}
        icon={FileTextIcon}
      >
        {#snippet meta()}
          <span class="font-mono" title={sop.id}>{sop.id.slice(0, 8)}</span>
          · {new Date(sop.created_at).toLocaleDateString(getLocale())}
        {/snippet}
        {#snippet metaExtra()}
          {#each sop.tags as tag (tag.id)}
            <button
              type="button"
              onclick={() => onTagClick(tag.id)}
              class="badge badge-sm relative z-10 cursor-pointer {activeFilters.tag_id === tag.id ? 'badge-soft badge-primary' : 'badge-outline'}"
              aria-pressed={activeFilters.tag_id === tag.id}
              aria-label={m.sops_filter_tag({ title: tag.title })}
            >
              {tag.title}
            </button>
          {/each}
        {/snippet}

        {#snippet trailing()}
          {#if sop.is_favorite}
            <form method="POST" action="?/unfavorite" use:enhance>
              <input type="hidden" name="sop_id" value={sop.id} />
              <button
                type="submit"
                class="btn btn-square btn-ghost btn-sm"
                aria-label={m.aria_remove_favorite({ title: sop.title })}
                title={m.aria_remove_favorite({ title: sop.title })}
              >
                <StarIcon class="size-4 fill-accent" aria-hidden="true" />
              </button>
            </form>
          {:else}
            <form method="POST" action="?/favorite" use:enhance>
              <input type="hidden" name="sop_id" value={sop.id} />
              <button
                type="submit"
                class="btn btn-square btn-ghost btn-sm"
                aria-label={m.aria_add_favorite({ title: sop.title })}
                title={m.aria_add_favorite({ title: sop.title })}
              >
                <StarIcon class="size-4" aria-hidden="true" />
              </button>
            </form>
          {/if}
        {/snippet}
      </ListRow>
    {:else}
      <li class="p-6 text-sm text-base-content/70">{m.sops_empty()}</li>
    {/each}
  </ul>

  {#if totalPages > 1}
    <nav class="flex justify-center border-t border-base-300 p-4" aria-label={m.sops_pagination()}>
      <div class="join">
        <button
          class="join-item btn btn-sm"
          disabled={currentPage === 1}
          onclick={() => changePage(activeFilters.offset - activeFilters.limit)}
          aria-label={m.common_previous_page()}
        >
          <ChevronLeftIcon size={16} />
        </button>
        <span class="join-item btn btn-sm no-animation pointer-events-none">
          {m.common_page_of({ page: String(currentPage), total: String(totalPages) })}
        </span>
        <button
          class="join-item btn btn-sm"
          disabled={currentPage === totalPages}
          onclick={() => changePage(activeFilters.offset + activeFilters.limit)}
          aria-label={m.common_next_page()}
        >
          <ChevronRightIcon size={16} />
        </button>
      </div>
    </nav>
  {/if}
</Card>
