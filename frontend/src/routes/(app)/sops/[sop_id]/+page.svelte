<script lang="ts">
  import IdBadge from '#lib/components/IdBadge.svelte';
  import ListSopVersions from '#lib/components/ListSopVersions.svelte';
  import SopVersionStatusBadge from '#lib/components/SopVersionStatusBadge.svelte';
  import FavoriteToggle from '#lib/components/FavoriteToggle.svelte';
  import {
    ArchiveIcon,
    ArchiveRestoreIcon,
    BookOpenIcon,
    NotebookIcon,
    PlusIcon,
    XIcon
  } from '@lucide/svelte';
  import ListAssociatedAssets from '#lib/components/ListAssociatedAssets.svelte';
  import Alert from '#lib/components/Alert.svelte';
  import Card from '#lib/components/Card.svelte';
  import CardPageHeading from '#lib/components/CardPageHeading.svelte';
  import SopArchivedNotice from '#lib/components/SopArchivedNotice.svelte';
  import { enhance } from '$app/forms';
  import * as m from '#lib/paraglide/messages.js';
  import { getLocale } from '#lib/paraglide/runtime.js';

  let { data, form } = $props();

  const tagListId = $props.id();

  // Same roles the backend gives sop:write (new versions, tag changes)
  let canEdit = $derived(['admin', 'approver', 'editor'].includes(data.user!.role));
  // Approvers decide what readers use, so they archive and unarchive (sop:sign:approver)
  let canArchive = $derived(['admin', 'approver'].includes(data.user!.role));
  // Set while the SOP is archived: read-only, so no new versions
  let archive = $derived(data.sop.archive);

  // The archive form opens in place; a failed submit keeps it open with the reason
  let archiveOpen = $state(false);
  let archiving = $state(false);
  let unarchiving = $state(false);
  let archiveFailed = $derived(form?.action === 'archive');
  $effect(() => {
    if (archiveFailed) archiveOpen = true;
  });

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
  // Only one release candidate exists at a time, and it is always the newest version
  let reviewPending = $derived(latest?.status === 'rc');
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

        {#if published || (canEdit && !archive) || (canArchive && !archive)}
          <div class="flex shrink-0 flex-wrap gap-2">
            {#if canArchive && !archive}
              <button
                type="button"
                class="btn btn-ghost"
                aria-expanded={archiveOpen}
                aria-controls={`${tagListId}-archive`}
                onclick={() => (archiveOpen = !archiveOpen)}
              >
                <ArchiveIcon class="size-4" />
                {m.sop_archive_button()}
              </button>
            {/if}
            {#if canEdit && !archive}
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

      {#if archive}
        <SopArchivedNotice {archive}>
          {#snippet actions()}
            {#if canArchive}
              <form
                method="POST"
                action="?/unarchive"
                use:enhance={() => {
                  unarchiving = true;
                  return async ({ update }) => {
                    await update();
                    unarchiving = false;
                  };
                }}
              >
                <button class="btn btn-sm" disabled={unarchiving}>
                  {#if unarchiving}
                    <span class="loading loading-spinner loading-xs"></span>
                    {m.sop_unarchiving()}
                  {:else}
                    <ArchiveRestoreIcon class="size-4" />
                    {m.sop_unarchive_button()}
                  {/if}
                </button>
              </form>
            {/if}
          {/snippet}
        </SopArchivedNotice>
      {:else if canArchive && archiveOpen}
        <form
          id={`${tagListId}-archive`}
          method="POST"
          action="?/archive"
          class="flex flex-col gap-3 border-t border-base-300 pt-4"
          use:enhance={() => {
            archiving = true;
            return async ({ update, result }) => {
              await update();
              archiving = false;
              if (result.type === 'success') archiveOpen = false;
            };
          }}
        >
          <div>
            <h2 class="font-semibold">{m.sop_archive_title()}</h2>
            <p class="mt-1 text-sm text-base-content/70">{m.sop_archive_body()}</p>
          </div>

          {#if reviewPending}
            <Alert type="info" compact message={m.sop_archive_rc_hint()} />
          {/if}

          <div>
            <label class="label" for={`${tagListId}-reason`}>
              <span class="label-text">{m.sop_archive_reason_label()}</span>
            </label>
            <textarea
              id={`${tagListId}-reason`}
              name="reason"
              rows="2"
              maxlength="500"
              class="textarea mt-2 w-full {archiveFailed ? 'textarea-error' : ''}"
              placeholder={m.sop_archive_reason_placeholder()}
              value={form && 'reason' in form ? String(form.reason ?? '') : ''}
              required
              autocomplete="off"
              disabled={reviewPending}
            ></textarea>
            <p class="mt-1 text-xs text-base-content/70">{m.sop_archive_reason_help()}</p>
          </div>

          <div class="flex justify-end gap-2">
            <button type="button" class="btn btn-ghost" onclick={() => (archiveOpen = false)}>
              {m.common_cancel()}
            </button>
            <!-- Archiving is undone with one click, so it is not styled as destructive -->
            <button class="btn" disabled={archiving || reviewPending}>
              {#if archiving}
                <span class="loading loading-spinner loading-xs"></span>
                {m.sop_archiving()}
              {:else}
                <ArchiveIcon class="size-4" />
                {m.sop_archive_confirm()}
              {/if}
            </button>
          </div>
        </form>
      {/if}

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
