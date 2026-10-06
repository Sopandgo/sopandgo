<script lang="ts">
  import { ChevronRightIcon, AlertTriangleIcon, CheckCircleIcon, FileQuestionIcon } from 'lucide-svelte';
  import type { UserSignatureStatus } from '$lib/sdk/types';
  import { partitionSignatureStatus } from '$lib/signatureBuckets';
  import Card from './Card.svelte';
  import * as m from '$lib/paraglide/messages.js';

  interface Props {
    status?: UserSignatureStatus[];
  }

  let { status = [] }: Props = $props();

  let { actionRequired, notStarted, upToDate } = $derived(partitionSignatureStatus(status));
</script>

<Card>
  <ul class="list">
    <li class="p-4 pb-2 text-xs opacity-60 tracking-widest uppercase font-bold border-b border-base-200">
      {m.signatures_heading()}
    </li>

    {#if actionRequired.length > 0}
      <li class="p-4 bg-warning/10 border-b border-warning/20">
        <h3 class="text-sm font-semibold text-warning flex items-center gap-2">
          <AlertTriangleIcon size={16} />
          {m.signatures_action()}
        </h3>
      </li>
      {#each actionRequired as sop (sop.sop_id)}
        <a href={`/sops/${sop.sop_id}`} class="block">
          <li class="list-row items-center hover:bg-base-200/50 transition-colors group relative bg-warning/5 border-b border-base-200/50">
            <div class="z-10 pointer-events-none">
              <div class="avatar avatar-placeholder">
                <div class="bg-warning/20 text-warning w-10 lg:w-12 rounded-field flex items-center justify-center">
                  <AlertTriangleIcon size={20} />
                </div>
              </div>
            </div>
            <div class="flex-1 z-10">
              <div class="font-bold text-sm lg:text-base group-hover:text-primary transition-colors">
                {sop.title}
              </div>
              <div class="text-[10px] opacity-70 mt-1 uppercase font-mono tracking-tighter">
                {m.common_version_published({ version: String(sop.latest_version) })}
              </div>
            </div>
            <div class="flex gap-1 z-10 items-center">
              <span class="badge badge-warning badge-sm mr-2 hidden sm:inline-flex">{m.dashboard_review()}</span>
              <div class="btn btn-square btn-ghost btn-sm lg:btn-md" aria-hidden="true">
                <ChevronRightIcon />
              </div>
            </div>
          </li>
        </a>
      {/each}
    {/if}

    {#if notStarted.length > 0}
      <li class="p-4 bg-base-200/30 border-b border-base-200">
        <h3 class="text-sm font-semibold flex items-center gap-2">
          <FileQuestionIcon size={16} />
          {m.signatures_not_started()}
        </h3>
      </li>
      {#each notStarted as sop (sop.sop_id)}
        <a href={`/sops/${sop.sop_id}`} class="block">
          <li class="list-row items-center hover:bg-base-200/50 transition-colors group relative border-b border-base-200/50">
            <div class="z-10 pointer-events-none">
              <div class="avatar avatar-placeholder">
                <div class="bg-base-300 text-base-content w-10 lg:w-12 rounded-field flex items-center justify-center">
                  <FileQuestionIcon size={20} />
                </div>
              </div>
            </div>
            <div class="flex-1 z-10">
              <div class="font-bold text-sm lg:text-base group-hover:text-primary transition-colors">
                {sop.title}
              </div>
              <div class="text-[10px] opacity-50 mt-1 uppercase font-mono tracking-tighter">
                {m.common_version({ version: String(sop.latest_version) })}
              </div>
            </div>
            <div class="flex gap-1 z-10 items-center">
              <span class="badge badge-ghost badge-sm mr-2 hidden sm:inline-flex">{m.signatures_read()}</span>
              <div class="btn btn-square btn-ghost btn-sm lg:btn-md" aria-hidden="true">
                <ChevronRightIcon />
              </div>
            </div>
          </li>
        </a>
      {/each}
    {/if}

    {#if upToDate.length > 0}
      <li class="p-4 bg-success/5 border-b border-success/10">
        <h3 class="text-sm font-semibold text-success flex items-center gap-2">
          <CheckCircleIcon size={16} />
          {m.signatures_up_to_date()}
        </h3>
      </li>
      {#each upToDate as sop (sop.sop_id)}
        <li class="list-row items-center cursor-default bg-success/5 border-b border-base-200/50">
          <div class="z-10 pointer-events-none">
            <div class="avatar avatar-placeholder">
              <div class="bg-success/20 text-success w-10 lg:w-12 rounded-field flex items-center justify-center">
                <CheckCircleIcon size={20} />
              </div>
            </div>
          </div>
          <div class="flex-1 z-10">
            <div class="font-medium text-sm lg:text-base opacity-90">{sop.title}</div>
            <div class="text-[10px] opacity-50 mt-1 uppercase font-mono tracking-tighter">
              {m.signatures_signed({ version: String(sop.latest_version) })}
            </div>
          </div>
        </li>
      {/each}
    {/if}

    {#if !actionRequired.length && !notStarted.length && !upToDate.length}
      <li class="p-12 text-center">
        <div class="text-sm opacity-40 italic">{m.signatures_empty()}</div>
      </li>
    {/if}
  </ul>
</Card>
