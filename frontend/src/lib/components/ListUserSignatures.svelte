<script lang="ts">
  import { FileTextIcon } from '@lucide/svelte';
  import type { UserSignatureStatus } from '$lib/sdk/types';
  import { partitionSignatureStatus } from '$lib/signatureBuckets';
  import Card from './Card.svelte';
  import ListHeading from './ListHeading.svelte';
  import ListRow from './ListRow.svelte';
  import * as m from '$lib/paraglide/messages.js';

  interface Props {
    status?: UserSignatureStatus[];
  }

  let { status = [] }: Props = $props();

  let { actionRequired, notStarted, upToDate } = $derived(partitionSignatureStatus(status));
</script>

<Card title={m.signatures_heading()}>
  <ul class="list">
    {#if actionRequired.length > 0}
      <ListHeading>{m.dashboard_outdated()}</ListHeading>
      {#each actionRequired as sop (sop.sop_id)}
        <ListRow
          href={`/sops/${sop.sop_id}/v/latest`}
          title={sop.title}
          meta={m.common_version_published({ version: String(sop.latest_version) })}
          icon={FileTextIcon}
          attention
        >
          {#snippet trailing()}
            <span class="badge badge-soft badge-warning badge-sm hidden sm:inline-flex">{m.dashboard_update()}</span>
          {/snippet}
        </ListRow>
      {/each}
    {/if}

    {#if notStarted.length > 0}
      <ListHeading>{m.dashboard_new_unsigned()}</ListHeading>
      {#each notStarted as sop (sop.sop_id)}
        <ListRow href={`/sops/${sop.sop_id}/v/latest`} title={sop.title} icon={FileTextIcon}>
          {#snippet meta()}<span class="font-mono">{m.common_version({ version: String(sop.latest_version) })}</span>{/snippet}
          {#snippet trailing()}
            <span class="badge badge-outline badge-sm hidden sm:inline-flex">{m.dashboard_review()}</span>
          {/snippet}
        </ListRow>
      {/each}
    {/if}

    {#if upToDate.length > 0}
      <ListHeading>{m.signatures_up_to_date()}</ListHeading>
      {#each upToDate as sop (sop.sop_id)}
        <ListRow title={sop.title} icon={FileTextIcon}>
          {#snippet meta()}<span class="font-mono">{m.signatures_signed({ version: String(sop.latest_version) })}</span>{/snippet}
          {#snippet trailing()}
            <span class="badge badge-soft badge-success badge-sm">{m.signatures_up_to_date()}</span>
          {/snippet}
        </ListRow>
      {/each}
    {/if}

    {#if !actionRequired.length && !notStarted.length && !upToDate.length}
      <li class="p-6 text-sm text-base-content/70">{m.signatures_empty()}</li>
    {/if}
  </ul>
</Card>
