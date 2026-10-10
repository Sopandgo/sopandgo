<script lang="ts">
  import { goto } from '$app/navigation';
  import Card from '#lib/components/Card.svelte';
  import CardPageHeading from '#lib/components/CardPageHeading.svelte';
  import Combobox from '#lib/components/Combobox.svelte';
  import ListSops from '#lib/components/ListSops.svelte';
  import EmptyState from '#lib/components/EmptyState.svelte';
  import {
    NotebookIcon,
    NotebookPenIcon,
    PlusIcon,
    SearchIcon,
    SearchXIcon,
    StarIcon,
    XIcon
  } from '@lucide/svelte';
  import * as m from '#lib/paraglide/messages.js';
  import type { PageData } from './$types';

  let { data }: { data: PageData } = $props();

  const SEARCH_DELAY_MS = 300;

  let canCreate = $derived(['admin', 'approver', 'editor'].includes(data.user!.role));

  // Local UI state mirrors URL-backed filters (avoid capturing only initial `data`)
  let searchTerm = $state('');
  // The query this page last navigated with; not reactive, so typing ahead is never overwritten
  let sentQuery: string | null = null;
  $effect(() => {
    const q = data.filters.q;
    if (q !== sentQuery) searchTerm = q;
    sentQuery = null;
  });

  // Helper: update query params and reset offset when filters change
  function updateQuery(updates: Record<string, string | null>, resetOffset = true) {
    const url = new URL(window.location.href);

    for (const [key, value] of Object.entries(updates)) {
      if (value && value.length > 0) url.searchParams.set(key, value);
      else url.searchParams.delete(key);
    }

    if (resetOffset) url.searchParams.set('offset', '0');
    // Filters change the list, not the page: keep focus and scroll where the user is
    goto(url.toString(), { reset: false });
  }

  let searchTimer: ReturnType<typeof setTimeout> | undefined;

  function handleSearch() {
    clearTimeout(searchTimer);
    const q = searchTerm.trim();
    if (q === data.filters.q) return;
    sentQuery = q;
    updateQuery({ q }, true);
  }

  function queueSearch() {
    clearTimeout(searchTimer);
    searchTimer = setTimeout(handleSearch, SEARCH_DELAY_MS);
  }

  function setTag(tagId: string) {
    updateQuery({ tag_id: tagId || null }, true);
  }

  function toggleTag(tagId: string) {
    setTag(data.filters.tag_id === tagId ? '' : tagId);
  }

  type FavoritesMode = 'all' | 'first' | 'only';

  const favoritesOptions: { mode: FavoritesMode; label: () => string }[] = [
    { mode: 'all', label: m.sops_favorites_all },
    { mode: 'first', label: m.sops_favorites_first },
    { mode: 'only', label: m.sops_favorites_only }
  ];

  let favoritesMode = $derived<FavoritesMode>(
    data.filters.favorites_only ? 'only' : data.filters.favorites_first ? 'first' : 'all'
  );

  function setFavoritesMode(mode: FavoritesMode) {
    updateQuery(
      {
        favorites_only: mode === 'only' ? 'true' : null,
        favorites_first: mode === 'first' ? 'true' : null
      },
      true
    );
  }

  function clearFilters() {
    clearTimeout(searchTimer);
    searchTerm = '';
    // Keep the user on the same route but strip filter params (and reset offset)
    updateQuery(
      { q: null, tag_id: null, favorites_only: null, favorites_first: null },
      true
    );
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

  // "Favorites first" only reorders; these are the filters that can hide SOPs
  let isNarrowed = $derived(
    Boolean(data.filters.q || data.filters.tag_id || data.filters.favorites_only)
  );
  // Nothing to search yet: the empty state replaces the toolbar and the header action
  let isLibraryEmpty = $derived(data.total === 0 && !isNarrowed);
  let isFavoritesEmpty = $derived(
    data.filters.favorites_only && !data.filters.q && !data.filters.tag_id
  );

  let tagOptions = $derived(data.tags.map((t) => ({ value: t.id, label: t.title })));
</script>

<svelte:head>
  <title>{m.page_sops()}</title>
</svelte:head>

<div class="flex flex-col gap-6">
  <Card>
    <div class="card-body">
      <div class="flex items-start justify-between gap-4">
        <div class="flex min-w-0 flex-col gap-1">
          <CardPageHeading>
            <NotebookIcon class="w-8 h-8" />
            {m.sops_heading()}
          </CardPageHeading>
          <span class="text-sm font-medium text-base-content/70">
            {m.sops_intro()}
          </span>
        </div>

        {#if canCreate && !isLibraryEmpty}
          <a
            href="/sops/new"
            class="btn btn-primary shrink-0"
            aria-label={m.common_new_sop()}
            title={m.common_new_sop()}
          >
            <PlusIcon class="w-5 h-5" />
            <span class="hidden sm:inline">{m.common_new_sop()}</span>
          </a>
        {/if}
      </div>
    </div>
  </Card>

  <ListSops
    items={{ sops: data.sops, total: data.total }}
    activeFilters={data.filters}
    onTagClick={toggleTag}
    toolbar={isLibraryEmpty ? undefined : toolbar}
    {empty}
  />
</div>

{#snippet toolbar()}
  <div class="flex flex-wrap items-center gap-3">
    <form
      role="search"
      class="relative w-full sm:w-72"
      onsubmit={(e) => {
        e.preventDefault();
        handleSearch();
      }}
    >
      <SearchIcon
        class="pointer-events-none absolute left-3 top-1/2 z-10 size-4 -translate-y-1/2 text-base-content/70"
        aria-hidden="true"
      />
      <input
        type="search"
        placeholder={m.sops_search_placeholder()}
        aria-label={m.sops_search_placeholder()}
        class="input w-full pl-9"
        bind:value={searchTerm}
        oninput={queueSearch}
      />
    </form>

    {#if tagOptions.length > 0}
      <Combobox
        class="w-full sm:w-56"
        label={m.common_tags()}
        hideLabel
        placeholder={m.sops_tag_placeholder()}
        options={tagOptions}
        bind:value={() => data.filters.tag_id, setTag}
        allLabel={m.sops_all_tags()}
        emptyLabel={m.sops_no_matching_tags()}
        clearLabel={m.sops_clear_tag_aria()}
      />
    {/if}

    <div class="join w-full sm:w-auto" role="group" aria-label={m.common_favorites()}>
      {#each favoritesOptions as option (option.mode)}
        <button
          type="button"
          class="btn join-item btn-sm flex-1 sm:btn-md sm:flex-none {favoritesMode === option.mode ? 'btn-active' : ''}"
          aria-pressed={favoritesMode === option.mode}
          onclick={() => setFavoritesMode(option.mode)}
        >
          {option.label()}
        </button>
      {/each}
    </div>

    {#if hasActiveFilters}
      <button
        type="button"
        class="btn btn-ghost sm:ml-auto"
        onclick={clearFilters}
        aria-label={m.sops_clear_filters_aria()}
      >
        <XIcon class="size-4" />
        {m.sops_clear_filters()}
      </button>
    {/if}
  </div>
{/snippet}

{#snippet empty()}
  {#if isLibraryEmpty}
    {#if canCreate}
      <EmptyState
        icon={NotebookPenIcon}
        tone="primary"
        title={m.sops_empty_new_title()}
        description={m.sops_empty_new_body()}
      >
        {#snippet actions()}
          <a href="/sops/new" class="btn btn-primary">
            <PlusIcon class="size-4" />
            {m.common_new_sop()}
          </a>
        {/snippet}
      </EmptyState>
    {:else}
      <EmptyState
        icon={NotebookIcon}
        title={m.sops_empty_none_title()}
        description={m.sops_empty_none_body()}
      />
    {/if}
  {:else if isFavoritesEmpty}
    <EmptyState
      icon={StarIcon}
      title={m.sops_empty_fav_title()}
      description={m.sops_empty_fav_body()}
    >
      {#snippet actions()}
        <button type="button" class="btn" onclick={() => setFavoritesMode('all')}>
          {m.common_all_sops()}
        </button>
      {/snippet}
    </EmptyState>
  {:else}
    <EmptyState
      icon={SearchXIcon}
      title={m.sops_empty_match_title()}
      description={data.filters.q
        ? m.sops_empty_match_query({ query: data.filters.q })
        : m.sops_empty_match_filters()}
    >
      {#snippet actions()}
        <button type="button" class="btn" onclick={clearFilters}>
          <XIcon class="size-4" />
          {m.sops_clear_filters()}
        </button>
      {/snippet}
    </EmptyState>
  {/if}
{/snippet}
