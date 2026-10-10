<script lang="ts">
    import { FileTextIcon } from '@lucide/svelte';
    import type { SOPVersion } from '$lib/sdk/types';
    import Card from './Card.svelte';
    import ListRow from './ListRow.svelte';
    import SopVersionStatusBadge from './SopVersionStatusBadge.svelte';
    import * as m from '$lib/paraglide/messages.js';
    import { getLocale } from '$lib/paraglide/runtime';

    interface Props {
        sopId: string;
        items?: SOPVersion[] | null;
    }

    let { sopId, items = [] }: Props = $props();

    let versions = $derived(items ?? []);
</script>

<Card title={m.versions_count({ count: String(versions.length) })}>
    <ul class="list">
        {#each versions as version (version.id)}
            <ListRow
                href={`/sops/${sopId}/v/${version.id}`}
                title={m.common_version({ version: String(version.version) })}
                icon={FileTextIcon}
            >
                {#if version.change_summary}{version.change_summary}{/if}
                {#snippet meta()}
                    <span class="font-mono" title={version.id}>{version.id.slice(0, 8)}</span>
                    · {new Date(version.created_at).toLocaleDateString(getLocale())}
                {/snippet}
                {#snippet trailing()}
                    <SopVersionStatusBadge status={version.status} />
                {/snippet}
            </ListRow>
        {:else}
            <li class="p-6 text-sm text-base-content/70">
                <p>{m.versions_empty()}</p>
                <p>{m.versions_empty_help()}</p>
            </li>
        {/each}
    </ul>
</Card>
