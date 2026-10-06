<script lang="ts">
    import { 
        CalendarClockIcon, 
        FingerprintPatternIcon, 
        GitBranchIcon, 
        GitCompareIcon, 
        WaypointsIcon

    } from 'lucide-svelte';
  import Card from './Card.svelte';
    import SopVersionStatusBadge from './SopVersionStatusBadge.svelte';
    import * as m from '$lib/paraglide/messages.js';
    import { getLocale } from '$lib/paraglide/runtime';

    interface Props {
        sopId: string;
        contentHash: string;
        createdAt: string;
        hashValid: boolean;
        status: string;
        version: number | string;
    }

    let { 
        sopId = "", 
        contentHash = "", 
        createdAt = "", 
        hashValid = false, 
        status = "",
        version 
    }: Props = $props();

    let formattedDate = $derived(
        createdAt ? new Date(createdAt).toLocaleString(getLocale()) : m.common_na()
    );
</script>

<!-- Todo: refactor into snippets, add tags, categories, projects etc -->

<Card>
    <ul class="list">
        <li class="p-4 pb-2 text-xs opacity-60 tracking-widest uppercase font-bold border-b border-base-200/50">
            {m.details_technical()}
        </li>

        <a href={`/sops/${sopId}`} class="contents">
            <li class="list-row items-center hover:bg-base-200/50 transition-colors">
                <div>
                    <GitBranchIcon size={24} class="p-1 opacity-70"/>
                </div>

                <div class="flex-1">
                    <div class="text-sm font-medium">{m.details_version({ version: String(version) })}</div>
                    <div class="text-xs opacity-40 font-mono italic">
                        {m.details_version_number()}
                    </div>
                </div>

                <div class="flex gap-1">
                    <div class="btn btn-square btn-ghost btn-sm lg:btn-md">
                        <GitCompareIcon size={18}/>
                    </div>
                </div>
            </li>
        </a>

        <li class="list-row items-center">
            <div>
                <CalendarClockIcon size={24} class="p-1 opacity-70"/>
            </div>

            <div class="flex-1">
                <div class="text-sm font-medium">{m.common_created({ when: formattedDate })}</div>
                <div class="text-xs opacity-40 font-mono italic">
                    {m.profile_timestamp()}
                </div>
            </div>
        </li>
        
        <li class="list-row items-center">
            <div>
                <WaypointsIcon size={24} class="p-1 opacity-70"/>
            </div>

            <div class="flex-1">
                <div class="text-sm font-medium">{m.details_status()} <SopVersionStatusBadge status={status} /></div>
                <div class="text-xs opacity-40 font-mono italic">
                    {m.details_status_help()}
                </div>
            </div>
        </li>

        <li class="list-row items-center {hashValid ? 'bg-success/5' : 'bg-error/5'}">
            <div>
                <FingerprintPatternIcon 
                    size={24} 
                    class="p-1 {hashValid ? 'stroke-success' : 'stroke-error'}"
                />
            </div>

            <div class="flex-1 overflow-hidden">
                <div class="text-sm font-medium">{m.details_checksum()}</div>
                <div class="text-[10px] opacity-60 font-mono truncate" title={contentHash}>
                    {contentHash}
                </div>
                
                <div class="mt-1 flex items-center gap-1">
                    <span class="inline-block w-2 h-2 rounded-full {hashValid ? 'bg-success' : 'bg-error'}"></span>
                    <span class="text-[10px] font-bold uppercase tracking-wider {hashValid ? 'text-success' : 'text-error'}">
                        {hashValid ? m.details_verified() : m.details_violated()}
                    </span>
                </div>
            </div>
        </li>
    </ul>
</Card>