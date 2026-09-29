<script lang="ts">
  import { goto } from '$app/navigation';
  import { page } from '$app/state';
  import Breadcrumbs from '$lib/components/Breadcrumbs.svelte';
  import Card from '$lib/components/Card.svelte';
  import CardPageHeading from '$lib/components/CardPageHeading.svelte';
  import ListSops from '$lib/components/ListSops.svelte';
  import { HouseIcon, NotebookIcon, PlusIcon, SearchIcon, XIcon } from 'lucide-svelte';
  import type { PageData } from './$types';

  let { data }: { data: PageData } = $props();

  // Local UI state mirrors URL-backed filters (avoid capturing only initial `data`)
  let searchTerm = $state('');
  $effect(() => {
    searchTerm = data.filters.q;
  });

  // Helper: update query params and reset offset when filters change
  function updateQuery(updates: Record<string, string | null>, resetOffset = true) {
    const url = new URL(window.location.href);

    for (const [key, value] of Object.entries(updates)) {
      if (value && value.length > 0) url.searchParams.set(key, value);
      else url.searchParams.delete(key);
    }

    if (resetOffset) url.searchParams.set('offset', '0');
    goto(url.toString());
  }

  function handleSearch() {
    updateQuery({ q: searchTerm ?? '' }, true);
  }

  function toggleTag(tagId: string) {
    const current = page.url.searchParams.get('tag_id') ?? '';
    updateQuery({ tag_id: current === tagId ? null : tagId }, true);
  }

  function clearFilters() {
    // Keep the user on the same route but strip filter params (and reset offset)
    updateQuery(
      { q: null, tag_id: null, favorites_only: null, favorites_first: null },
      true
    );
  }

  function toggleFavoritesOnly() {
    const cur = page.url.searchParams.get('favorites_only') === 'true';
    updateQuery({ favorites_only: cur ? null : 'true' }, true);
  }

  function toggleFavoritesFirst() {
    const cur = page.url.searchParams.get('favorites_first') === 'true';
    updateQuery({ favorites_first: cur ? null : 'true' }, true);
  }

  // Derived for UI
  let hasActiveFilters = $derived(
    Boolean(
      data.filters.q ||
        data.filters.tag_id ||
        data.filters.favorites_only ||
        data.filters.favorites_first
    )
  );
  let activeTagId = $derived(data.filters.tag_id);
</script>

<div class="flex flex-col gap-6">
  <Breadcrumbs
    items={[
      { label: 'Dashboard', href: '/dashboard', icon: HouseIcon },
      { label: 'SOPs', icon: NotebookIcon }
    ]}
  />

  <Card>
    <div class="card-body">
      <div class="flex flex-col gap-1">
        <CardPageHeading>
          <NotebookIcon class="w-8 h-8" />
          Standard Operating Procedures
        </CardPageHeading>
        <span class="text-sm font-medium text-base-content/70">
          Manage and filter your organization's SOPs.
        </span>
      </div>

      <div class="card-actions justify-end pt-4 border-t border-base-200">
        {#if ['admin', 'approver', 'editor'].includes(data.user!.role)}
          <a href="/sops/new" class="btn btn-primary ml-2">
            <PlusIcon class="w-5 h-5" />
            <span class="hidden sm:inline">New SOP</span>
          </a>
        {/if}
      </div>
    </div>
  </Card>

  <Card>
    <div class="card-body">
      <span class="text-xs opacity-60 tracking-widest uppercase font-bold">
        Search & Filters
      </span>

      <div class="flex flex-col gap-3 pt-4 border-t border-base-200">
        <div class="flex flex-col md:flex-row gap-3 md:items-center md:justify-between">
          <div class="join w-full md:w-auto">
            <input
              type="text"
              placeholder="Search titles..."
              class="input input-bordered join-item flex-1 md:w-80"
              bind:value={searchTerm}
              onkeydown={(e) => e.key === 'Enter' && handleSearch()}
            />
            <button
              class="btn btn-primary join-item"
              onclick={handleSearch}
              aria-label="Submit search"
            >
              <SearchIcon class="w-5 h-5" />
            </button>
          </div>

          {#if hasActiveFilters}
            <button
              class="btn btn-error text-error-content md:self-auto"
              onclick={clearFilters}
              aria-label="Clear all filters"
            >
              <XIcon class="w-4 h-4" />
              Clear Filters
            </button>
          {/if}
        </div>

          <div class="flex flex-col gap-2 pt-2 border-t border-base-200/80">
            <span class="text-xs opacity-60 tracking-widest uppercase font-bold">
              Favorites
            </span>
            <div class="flex flex-wrap gap-3">
              <label class="label cursor-pointer gap-2 justify-start py-1">
                <input
                  type="checkbox"
                  class="checkbox checkbox-sm checkbox-primary"
                  checked={data.filters.favorites_only}
                  onchange={toggleFavoritesOnly}
                  aria-label="Show only favorite SOPs"
                />
                <span class="label-text text-sm">Favorites only</span>
              </label>
              <label class="label cursor-pointer gap-2 justify-start py-1">
                <input
                  type="checkbox"
                  class="checkbox checkbox-sm checkbox-primary"
                  checked={data.filters.favorites_first}
                  onchange={toggleFavoritesFirst}
                  aria-label="Sort favorites to the top"
                />
                <span class="label-text text-sm">Favorites first</span>
              </label>
            </div>
          </div>

          <div class="flex flex-col gap-2">
          <div class="flex items-center justify-between">
            <span class="text-xs opacity-60 tracking-widest uppercase font-bold">
              Tags
            </span>

            {#if activeTagId}
              <button
                type="button"
                class="btn btn-xs btn-ghost"
                onclick={() => toggleTag(activeTagId)}
                aria-label="Clear tag filter"
              >
                Clear tag
              </button>
            {/if}
          </div>

          <div class="flex flex-wrap gap-2">
            {#each data.tags as t (t.id)}
              <button
                type="button"
                onclick={() => toggleTag(t.id)}
                class="badge badge-sm cursor-pointer hover:badge-outline transition-all {activeTagId === t.id
                  ? 'badge-primary'
                  : 'badge-ghost opacity-70'}"
                aria-pressed={activeTagId === t.id}
                aria-label={`Filter by tag: ${t.title}`}
                title={activeTagId === t.id ? 'Click to remove filter' : 'Click to filter'}
              >
                {t.title}
              </button>
            {:else}
              <span class="text-sm opacity-60">No tags available.</span>
            {/each}
          </div>
        </div>
      </div>
    </div>
  </Card>

  <ListSops
    items={{ sops: data.sops, total: data.total }}
    activeFilters={data.filters}
    onTagClick={toggleTag}
  />
</div>