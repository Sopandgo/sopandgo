<script lang="ts">
  import {
    ChevronRightIcon,
    NotebookIcon,
    ChevronLeftIcon,
    StarIcon
  } from 'lucide-svelte';
  import type { SOPListResponse } from '$lib/sdk/types';
  import Card from './Card.svelte';
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
    onTagClick = () => {}
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
  <ul class="list">
    <li class="p-4 pb-2 flex justify-between items-center border-b border-base-200">
      <span class="text-xs opacity-60 tracking-widest uppercase font-bold">
        {m.sops_results({ shown: String(sops.length), total: String(total) })}
      </span>
    </li>

    {#each sops as sop (sop.id)}
      <li
        class="list-row items-center hover:bg-base-200/50 transition-colors group relative flex flex-nowrap gap-2"
      >
        <a
          href={rowHref(sop.id)}
          class="flex flex-1 items-center gap-3 lg:gap-4 min-w-0"
          aria-label={m.sops_view_latest({ title: sop.title })}
        >
          <div class="z-10 pointer-events-none shrink-0" aria-hidden="true">
            <div class="avatar avatar-placeholder">
              <div
                class="bg-secondary text-secondary-content w-12 lg:w-16 rounded-field flex items-center justify-center"
              >
                <NotebookIcon size={24} />
              </div>
            </div>
          </div>

          <div class="flex-1 z-10 min-w-0">
            <div class="font-bold text-sm lg:text-base group-hover:text-primary transition-colors">
              {sop.title}
            </div>

            <div class="flex flex-wrap items-center gap-2 mt-1">
              <div class="text-[10px] opacity-50 font-mono uppercase tracking-tighter">
                {m.sops_id_date({ id: sop.id, date: new Date(sop.created_at).toLocaleDateString(getLocale()) })}
              </div>

              {#each sop.tags as tag (tag.id)}
                <button
                  type="button"
                  onclick={(e) => {
                    e.preventDefault();
                    e.stopPropagation();
                    onTagClick(tag.id);
                  }}
                  class="badge badge-sm cursor-pointer hover:badge-outline transition-all {activeFilters.tag_id === tag.id
                    ? 'badge-primary'
                    : 'badge-ghost opacity-70'}"
                  aria-label={m.sops_filter_tag({ title: tag.title })}
                >
                  {tag.title}
                </button>
              {/each}
            </div>
          </div>
        </a>

        <div class="flex gap-1 z-10 shrink-0 items-center">
          {#if sop.is_favorite}
            <form method="POST" action="?/unfavorite" use:enhance>
              <input type="hidden" name="sop_id" value={sop.id} />
              <button
                type="submit"
                class="btn btn-square btn-ghost btn-sm lg:btn-md text-warning"
                aria-label={m.aria_remove_favorite({ title: sop.title })}
              >
                <StarIcon size={20} class="fill-current" />
              </button>
            </form>
          {:else}
            <form method="POST" action="?/favorite" use:enhance>
              <input type="hidden" name="sop_id" value={sop.id} />
              <button
                type="submit"
                class="btn btn-square btn-ghost btn-sm lg:btn-md opacity-70 hover:opacity-100"
                aria-label={m.aria_add_favorite({ title: sop.title })}
              >
                <StarIcon size={20} />
              </button>
            </form>
          {/if}

          <a
            href={rowHref(sop.id)}
            class="btn btn-square btn-ghost btn-sm lg:btn-md"
            aria-label={m.sops_open({ title: sop.title })}
          >
            <ChevronRightIcon />
          </a>
        </div>
      </li>
    {:else}
      <li class="p-12 text-center">
        <div class="text-sm opacity-40">{m.sops_empty()}</div>
      </li>
    {/each}
  </ul>

  {#if totalPages > 1}
    <nav class="p-4 flex justify-center border-t border-base-200 bg-base-100/50" aria-label={m.sops_pagination()}>
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
