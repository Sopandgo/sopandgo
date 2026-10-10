<script lang="ts">
  import {
    ChevronRightIcon,
    FileTextIcon,
    ChevronLeftIcon,
    StarIcon
  } from 'lucide-svelte';
  import type { Snippet } from 'svelte';
  import { SvelteSet } from 'svelte/reactivity';
  import type { SOPListItem, SOPListResponse, Tag } from '$lib/sdk/types';
  import Card from './Card.svelte';
  import ListRow from './ListRow.svelte';
  import SopVersionStatusBadge from './SopVersionStatusBadge.svelte';
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

  // Tags shown per row before the rest collapse into a "+N" badge
  const MAX_ROW_TAGS = 3;
  const expandedTags = new SvelteSet<string>();

  // The filtered tag always stays visible, so the reader sees why the row matched
  function orderTags(tags: Tag[]) {
    const i = tags.findIndex((t) => t.id === activeFilters.tag_id);
    if (i < MAX_ROW_TAGS) return tags;
    return [tags[i], ...tags.slice(0, i), ...tags.slice(i + 1)];
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
          {@const tags = orderTags(sop.tags)}
          {@const expanded = expandedTags.has(sop.id)}
          {@const hidden = tags.slice(MAX_ROW_TAGS)}
          {#each expanded ? tags : tags.slice(0, MAX_ROW_TAGS) as tag (tag.id)}
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
          {#if hidden.length > 0}
            {#if expanded}
              <button
                type="button"
                onclick={() => expandedTags.delete(sop.id)}
                class="badge badge-sm badge-ghost relative z-10 cursor-pointer"
                aria-expanded="true"
              >
                {m.sops_fewer_tags()}
              </button>
            {:else}
              <button
                type="button"
                onclick={() => expandedTags.add(sop.id)}
                class="badge badge-sm badge-ghost relative z-10 cursor-pointer"
                aria-expanded="false"
                aria-label={m.sops_more_tags_aria({ count: String(hidden.length) })}
                title={hidden.map((t) => t.title).join(', ')}
              >
                +{hidden.length}
              </button>
            {/if}
          {/if}
        {/snippet}

        {#snippet trailing()}
          <!-- A newer version than the published one (draft, in review, rejected) -->
          {#if sop.latest_version && sop.latest_version.status !== 'published'}
            <SopVersionStatusBadge status={sop.latest_version.status} />
          {/if}
          {#if sop.is_favorite}
            <form method="POST" action="?/unfavorite" use:enhance>
              <input type="hidden" name="sop_id" value={sop.id} />
              <button
                type="submit"
                class="btn btn-square btn-ghost btn-sm"
                aria-label={m.aria_remove_favorite({ title: sop.title })}
                title={m.aria_remove_favorite({ title: sop.title })}
              >
                <StarIcon class="size-4 fill-current" aria-hidden="true" />
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
