<script lang="ts">
  import {
    ChevronRightIcon,
    FileTextIcon,
    ChevronLeftIcon,
    GitBranchIcon
  } from '@lucide/svelte';
  import type { Snippet } from 'svelte';
  import type { SOPListItem, SOPListResponse } from '$lib/sdk/types';
  import Card from './Card.svelte';
  import ListRow from './ListRow.svelte';
  import FavoriteToggle from './FavoriteToggle.svelte';
  import SopVersionStatusBadge from './SopVersionStatusBadge.svelte';
  import TagList from './TagList.svelte';
  import { goto } from '$app/navigation';
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
    /** Shown instead of the rows when there are none (the parent knows why). */
    empty?: Snippet;
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
    toolbar,
    empty
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

  // What readers see, and when it last changed
  function rowMeta(sop: SOPListItem) {
    const published =
      sop.published_version != null
        ? m.common_version({ version: String(sop.published_version) })
        : m.sops_not_published();
    const changed = sop.latest_version?.created_at ?? sop.created_at;
    const date = new Date(changed).toLocaleDateString(getLocale());
    return `${published} · ${m.sops_updated({ date })}`;
  }

  // Work waiting on a newer version (draft, in review), or the state of an SOP nothing
  // is published for yet. A rejected version after a published one changes nothing for
  // readers, so it shows only on the SOP page.
  function pendingStatus(sop: SOPListItem) {
    const status = sop.latest_version?.status;
    if (!status || status === 'published') return undefined;
    if (status === 'rejected' && sop.published_version != null) return undefined;
    return status;
  }

  function rowHref(sopId: string) {
    return `/sops/${sopId}/v/latest`;
  }
</script>

<Card>
  <!-- One-row header (count on the right at every width) to leave room for the rows -->
  <div class="flex items-baseline justify-between gap-3 border-b border-base-300 p-4 sm:p-6">
    <h2 class="text-lg font-semibold">{m.sops_list_title()}</h2>
    {#if total > 0}
      <span class="text-sm text-base-content/70">
        {m.sops_showing({ shown: String(sops.length), total: String(total) })}
      </span>
    {/if}
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
        meta={rowMeta(sop)}
      >
        {#snippet metaExtra()}
          <TagList tags={sop.tags} activeTagId={activeFilters.tag_id} {onTagClick} />
        {/snippet}

        {#snippet trailing()}
          {@const pending = pendingStatus(sop)}
          {#if pending}
            <SopVersionStatusBadge status={pending} />
          {/if}
          <!-- The row opens the latest version; this opens the SOP page with all versions -->
          <a
            href={`/sops/${sop.id}`}
            class="btn btn-square btn-ghost btn-sm"
            aria-label={m.sops_view_versions({ title: sop.title })}
            title={m.sops_view_versions({ title: sop.title })}
          >
            <GitBranchIcon class="size-4" aria-hidden="true" />
          </a>
          <FavoriteToggle sopId={sop.id} title={sop.title} isFavorite={!!sop.is_favorite} />
        {/snippet}
      </ListRow>
    {:else}
      {#if empty}
        <li>{@render empty()}</li>
      {:else}
        <li class="p-6 text-sm text-base-content/70">{m.sops_empty()}</li>
      {/if}
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
