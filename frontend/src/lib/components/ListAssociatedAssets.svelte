<script lang="ts">
    import { ExternalLinkIcon, PaperclipIcon } from 'lucide-svelte';
    import IntegrityButtonAssets from './IntegrityButtonAssets.svelte';
    import type { SOPAsset } from '$lib/sdk/types';
  import Card from './Card.svelte';

    // Define props with Svelte 5 runes
    let { items = [] } = $props<{ items?: SOPAsset[] | null }>();
    
    // Safely derive assets
    let assets = $derived(items ?? []);
</script>

<Card>
    <ul class="list">
        <li class="p-4 pb-2 text-xs opacity-60 tracking-widest uppercase font-bold border-b border-base-200/50">
            Associated Assets ({assets.length})
        </li>

        {#each assets as asset (asset.id)}
            <li class="list-row items-center hover:bg-base-200/50 transition-colors">
                <div>
                    <PaperclipIcon class="p-1 opacity-40 " size={20}/>
                </div>

                <div class="flex-1">
                    <div class="font-bold text-sm lg:text-base">{asset.file_name}</div>
                    <div class="text-[10px] opacity-40 font-mono uppercase tracking-tighter">
                        Uploaded: {new Date(asset.created_at).toLocaleString()}
                    </div>
                </div>

                <div class="flex gap-2 items-center">
                    <IntegrityButtonAssets 
                        assetId={asset.id} 
                    />
                    
                    <a
                        href={`/api/assets/download?id=${encodeURIComponent(asset.id)}`}
                        target="_blank"
                        rel="noopener noreferrer"
                        class="btn btn-square btn-ghost btn-sm"
                        title="Open/Download Asset"
                        aria-label="Open asset"
                    >
                        <ExternalLinkIcon size={18} class="opacity-70"/>
                    </a>
                </div>
            </li>
        {:else}
            <li class="p-12 text-center">
                <div class="text-sm opacity-40">No associated assets found.</div>
            </li>
        {/each}
    </ul>
</Card>