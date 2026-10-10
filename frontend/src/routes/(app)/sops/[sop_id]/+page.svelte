<script lang="ts">
  import IdBadge from '$lib/components/IdBadge.svelte';
  import ListSopVersions from '$lib/components/ListSopVersions.svelte';
  import SopVersionStatusBadge from '$lib/components/SopVersionStatusBadge.svelte';
  import FavoriteToggle from '$lib/components/FavoriteToggle.svelte';
  import { BookOpenIcon, NotebookIcon, PlusIcon, XIcon } from '@lucide/svelte';
  import ListAssociatedAssets from '$lib/components/ListAssociatedAssets.svelte';
  import Alert from '$lib/components/Alert.svelte';
  import Card from '$lib/components/Card.svelte';
  import CardPageHeading from '$lib/components/CardPageHeading.svelte';
  import { enhance } from '$app/forms';
  import * as m from '$lib/paraglide/messages.js';
  import { getLocale } from '$lib/paraglide/runtime';

  let { data, form } = $props();

  const tagListId = $props.id();

  // Same roles the backend gives sop:write (new versions, tag changes)
  let canEdit = $derived(['admin', 'approver', 'editor'].includes(data.user!.role));

  let versions = $derived(data.versions ?? []);
  // What readers see, and the newest version if it has not replaced that yet
  let published = $derived(versions.find((v) => v.status === 'published'));
  let latest = $derived(
    versions.reduce<(typeof versions)[number] | undefined>(
      (max, v) => (!max || v.version > max.version ? v : max),
      undefined
    )
  );
  let pending = $derived(latest && latest.status !== 'published' ? latest : undefined);
  let updated = $derived(
    new Date(latest?.created_at ?? data.sop.created_at).toLocaleDateString(getLocale())
  );

  let tags = $derived(data.sop.tags ?? []);
  // Suggestions for the add field: active tags not on this SOP yet
  let attachedIds = $derived(new Set(tags.map((t) => t.id)));
  let addableTags = $derived(
    (data.allTags ?? []).filter((t) => !attachedIds.has(t.id) && t.is_active !== false)
  );
</script>

<svelte:head>
  <title>{m.page_sop()}</title>
</svelte:head>

<div class="flex flex-col gap-6">
  <Card>
    <div class="card-body gap-4">
      <div class="flex flex-col gap-4 sm:flex-row sm:items-start sm:justify-between">
        <div class="flex min-w-0 flex-col gap-1">
          <div class="flex items-center gap-1">
            <CardPageHeading>
              <NotebookIcon class="w-8 h-8" />
              {data.sop.title}
            </CardPageHeading>
            <FavoriteToggle sopId={data.sop.id} title={data.sop.title} isFavorite={!!data.sop.is_favorite} />
          </div>

          <p class="flex flex-wrap items-center gap-x-2 gap-y-1 text-sm text-base-content/70">
            {#if published}
              <a href={`/sops/${data.sop.id}/v/${published.id}`} class="text-base-content underline">
                {m.common_version({ version: String(published.version) })}
              </a>
              <SopVersionStatusBadge status={published.status} />
            {:else}
              <span>{m.sops_not_published()}</span>
            {/if}
            {#if pending}
              <span aria-hidden="true">·</span>
              <a href={`/sops/${data.sop.id}/v/${pending.id}`} class="text-base-content underline">
                {m.common_version({ version: String(pending.version) })}
              </a>
              <SopVersionStatusBadge status={pending.status} />
            {/if}
            <span aria-hidden="true">·</span>
            <span>{m.sops_updated({ date: updated })}</span>
            <span aria-hidden="true">·</span>
            <IdBadge id={data.sop.id} />
          </p>
        </div>

        {#if published || canEdit}
          <div class="flex shrink-0 flex-wrap gap-2">
            {#if canEdit}
              <a href={`/sops/${data.sop.id}/new`} class="btn {published ? '' : 'btn-primary'}">
                <PlusIcon class="size-4" />
                {m.common_new_version()}
              </a>
            {/if}
            {#if published}
              <a href={`/sops/${data.sop.id}/v/latest`} class="btn btn-primary">
                <BookOpenIcon class="size-4" />
                {m.sop_read_current()}
              </a>
            {/if}
          </div>
        {/if}
      </div>

      {#if tags.length > 0 || canEdit}
        <div class="flex flex-wrap items-center gap-2">
          {#if tags.length > 0}
            <ul class="flex flex-wrap items-center gap-2" aria-label={m.common_tags()}>
              {#each tags as tag (tag.id)}
                <li class="badge badge-outline gap-1 {canEdit ? 'pr-0.5' : ''}">
                  {tag.title}
                  {#if canEdit}
                    <form method="POST" action="?/detach_tag" use:enhance class="flex">
                      <input type="hidden" name="tag_id" value={tag.id} />
                      <button
                        type="submit"
                        class="btn btn-ghost btn-xs btn-square size-5 min-h-0"
                        aria-label={m.sops_remove_tag({ title: tag.title })}
                        title={m.sops_remove_tag({ title: tag.title })}
                      >
                        <XIcon class="size-3" aria-hidden="true" />
                      </button>
                    </form>
                  {/if}
                </li>
              {/each}
            </ul>
          {/if}

          {#if canEdit}
            <!-- One field adds an existing tag or creates it; the list only suggests -->
            <form method="POST" action="?/add_tag" use:enhance class="flex items-center gap-2">
              <input
                class="input input-sm w-44"
                name="title"
                list={tagListId}
                autocomplete="off"
                required
                placeholder={m.sops_add_tag_placeholder()}
                aria-label={m.sops_add_tag_label()}
              />
              <datalist id={tagListId}>
                {#each addableTags as t (t.id)}
                  <option value={t.title}></option>
                {/each}
              </datalist>
              <button class="btn btn-sm" type="submit">
                <PlusIcon class="size-4" />
                {m.common_add()}
              </button>
            </form>
          {/if}
        </div>
      {/if}

      {#if form?.message}
        <Alert type="error" message={form.message} />
      {/if}
    </div>
  </Card>

  <ListSopVersions sopId={data.sop.id} items={versions} />

  <ListAssociatedAssets items={data.assets} />
</div>
