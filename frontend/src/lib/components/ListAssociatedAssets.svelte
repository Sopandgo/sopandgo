<script lang="ts">
    import { ExternalLinkIcon, PaperclipIcon } from 'lucide-svelte';
    import IntegrityCheck from './IntegrityCheck.svelte';
    import ListRow from './ListRow.svelte';
    import type { SOPAsset } from '$lib/sdk/types';
  import Card from './Card.svelte';
  import * as m from '$lib/paraglide/messages.js';
  import { getLocale } from '$lib/paraglide/runtime';

    // Define props with Svelte 5 runes
    let { items = [] }: { items?: SOPAsset[] | null } = $props();
    
    // Safely derive assets
    let assets = $derived(items ?? []);
</script>

<Card>
    <ul class="list">
        <li class="p-4 pb-2 text-xs opacity-60 tracking-widest uppercase font-bold border-b border-base-200/50">
            {m.assets_count({ count: String(assets.length) })}
        </li>

        {#each assets as asset (asset.id)}
            <ListRow
                title={asset.file_name}
                meta={m.assets_uploaded({ when: new Date(asset.created_at).toLocaleString(getLocale()) })}
                icon={PaperclipIcon}
                iconTone="plain"
            >
                {#snippet trailing()}
                    <IntegrityCheck kind="asset" id={asset.id} />
                    <a
                        href={`/api/assets/download?id=${encodeURIComponent(asset.id)}`}
                        target="_blank"
                        rel="noopener noreferrer"
                        class="btn btn-square btn-ghost btn-sm"
                        title={m.assets_open_title()}
                        aria-label={m.assets_open_aria()}
                    >
                        <ExternalLinkIcon size={18} class="opacity-70"/>
                    </a>
                {/snippet}
            </ListRow>
        {:else}
            <li class="p-12 text-center">
                <div class="text-sm opacity-40">{m.assets_empty()}</div>
            </li>
        {/each}
    </ul>
</Card>