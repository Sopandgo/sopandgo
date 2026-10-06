<script lang="ts">
  import IdBadge from '$lib/components/IdBadge.svelte';
  import ListSopVersions from '$lib/components/ListSopVersions.svelte';
  import SopTrainingCoverage from '$lib/components/SopTrainingCoverage.svelte';
  import { HouseIcon, NotebookIcon, PlusIcon, TagIcon, XIcon, PlusCircleIcon, StarIcon } from 'lucide-svelte';
  import ListAssociatedAssets from '$lib/components/ListAssociatedAssets.svelte';
  import Breadcrumbs from '$lib/components/Breadcrumbs.svelte';
  import Card from '$lib/components/Card.svelte';
  import CardPageHeading from '$lib/components/CardPageHeading.svelte';
  import { enhance } from '$app/forms';
  import * as m from '$lib/paraglide/messages.js';

  let { data } = $props();

  let newTagTitle = $state('');

  // filter out tags already attached (and optionally inactive ones)
  let attachedIds = $derived(new Set((data.sop.tags ?? []).map((t) => t.id)));
  let addableTags = $derived(
    (data.allTags ?? []).filter((t) => !attachedIds.has(t.id) && t.is_active !== false)
  );
</script>

<svelte:head>
  <title>{m.page_sop()}</title>
</svelte:head>

<div class="flex flex-col gap-6">
    <Breadcrumbs items={[
        { label: m.page_dashboard(), href: '/sops', icon: HouseIcon},
        { label: m.page_sops(), href: '/sops', icon: NotebookIcon},
        { label: data.sop.title, icon: NotebookIcon},
    ]}/>

    <Card>
        <div class="card-body">
            <CardPageHeading>
                <NotebookIcon class="w-8 h-8" />
                {data.sop.title}
            </CardPageHeading>

            <div class="flex items-center gap-2 text-base-content/70">
                <span class="text-sm font-medium">ID:</span> 
                <IdBadge id={data.sop.id} />
            </div>

            <div class="flex flex-wrap items-center gap-2 pt-2">
              {#if data.sop.is_favorite}
                <form
                  method="POST"
                  action="?/unfavorite"
                  use:enhance={() =>
                    async ({ update }) => {
                      await update({ invalidateAll: true });
                    }}
                >
                  <button
                    type="submit"
                    class="btn btn-outline btn-warning btn-sm gap-1"
                    aria-label={m.sops_remove_favorites_aria()}
                  >
                    <StarIcon class="w-4 h-4 fill-current" />
                    {m.sops_favorited()}
                  </button>
                </form>
              {:else}
                <form
                  method="POST"
                  action="?/favorite"
                  use:enhance={() =>
                    async ({ update }) => {
                      await update({ invalidateAll: true });
                    }}
                >
                  <button type="submit" class="btn btn-ghost btn-sm gap-1" aria-label={m.sops_add_favorites_aria()}>
                    <StarIcon class="w-4 h-4" />
                    {m.sops_add_to_favorites()}
                  </button>
                </form>
              {/if}
            </div>

            {#if data.user!.role === 'admin' || data.user!.role === 'approver' || data.user!.role === 'editor'}
                <div class="card-actions justify-end pt-4 border-t border-base-200">
                    <a href={`/sops/${data.sop.id}/new`}  class="btn btn-primary">
                        <PlusIcon class="w-5 h-5" />
                        {m.common_new_version()}
                    </a>
                </div>
            {/if}
        </div>
    </Card>    

<Card>
  <div class="card-body">
    <span class="text-xs opacity-60 tracking-widest uppercase font-bold">
      {m.common_tags()}
    </span>

    <div class="pt-4 border-t border-base-200 flex flex-col gap-4">
      <!-- current tags -->
      <div class="flex flex-wrap gap-2">
        {#if (data.sop.tags ?? []).length === 0}
          <span class="text-sm opacity-60">{m.sops_no_tags_assigned()}</span>
        {:else}
          {#each data.sop.tags as tag (tag.id)}
            <form method="POST" action="?/detach_tag">
              <input type="hidden" name="tag_id" value={tag.id} />
              <button
                type="submit"
                class="badge badge-primary gap-1 cursor-pointer"
                aria-label={m.sops_remove_tag({ title: tag.title })}
                title={m.sops_remove_tag_title()}
              >
                <TagIcon class="w-3 h-3" />
                {tag.title}
                <XIcon class="w-3 h-3" />
              </button>
            </form>
          {/each}
        {/if}
      </div>

      <!-- attach existing -->
      <form method="POST" action="?/attach_tag" class="flex flex-col md:flex-row gap-2 md:items-center">
        <select class="select select-bordered flex-1" name="tag_id" aria-label={m.sops_select_tag()}>
          {#each addableTags as t (t.id)}
            <option value={t.id}>{t.title}</option>
          {/each}
        </select>
        <button class="btn btn-primary" type="submit" aria-label={m.sops_attach_tag()}>
          <PlusCircleIcon class="w-5 h-5" />
          {m.common_add()}
        </button>
      </form>      

      <!-- create + attach -->
      <form method="POST" action="?/create_and_attach_tag" class="flex flex-col md:flex-row gap-2 md:items-center">
        <input
          class="input input-bordered flex-1"
          name="title"
          placeholder={m.sops_new_tag_placeholder()}
          bind:value={newTagTitle}
        />
        <button class="btn btn-accent" type="submit" aria-label={m.sops_create_add_aria()}>
          <PlusIcon class="w-5 h-5" />
          {m.sops_create_add()}
        </button>
      </form>
    </div>
  </div>
</Card>

    {#if data.user?.role === 'admin' || data.user?.role === 'approver'}
      <Card>
        <div class="border-b border-base-200 p-4 sm:p-5">
          <h2 class="text-lg font-semibold">{m.dashboard_training_title()}</h2>
          <p class="mt-1 text-sm text-base-content/70">
            {m.sops_training_help()}
          </p>
        </div>
        <SopTrainingCoverage items={data.trainingCoverage ?? []} />
      </Card>
    {/if}

    <ListSopVersions 
        sopId={data.sop.id} 
        items={data.versions ?? []}
    />

    <ListAssociatedAssets items={data.assets} />
</div>

