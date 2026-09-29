<script lang="ts">
    import { ChevronRightIcon, NotebookTextIcon } from 'lucide-svelte';
    import type { SOPVersion } from '$lib/sdk/types'; 
  import Card from './Card.svelte';
    import SopVersionStatusBadge from './SopVersionStatusBadge.svelte';

    // Define the component props using TypeScript interface
    interface Props {
        sopId: string; // Required for routing
        items?: SOPVersion[] | null; // Versions can be null from API
    }

    // Initialize props with defaults
    let { sopId, items = [] }: Props = $props();
    
    // Safely derive versions to prevent .length errors on null
    let versions = $derived(items ?? []);
</script>

<Card>
    <ul class="list">
        <li class="p-4 pb-2 text-xs opacity-60 tracking-widest uppercase font-bold border-b border-base-200/50">
            Versions ({versions.length})
        </li>

        {#each versions as version (version.id)}
            <li>
                <a 
                    href={`/sops/${sopId}/v/${version.id}`} 
                    class={`list-row items-center ${version.status === 'published' ? 'bg-primary/20' : ''} hover:bg-base-200/50 transition-colors p-4 flex gap-4`}
                >
                    <div>
                        <div class="avatar avatar-placeholder">
                            <div class="w-16 rounded-field grid place-items-center"
                                class:bg-primary={version.status === 'published'}
                                class:text-primary-content={version.status === 'published'}
                                class:bg-secondary={version.status !== 'published'}
                                class:text-secondary-content={version.status !== 'published'}>
                                <NotebookTextIcon />
                            </div>
                        </div>
                    </div>

                    <div class="flex-1">
                        <div class="font-bold text-sm lg:text-base">Version {version.version}</div>
                        {#if version.change_summary}
                            <div class="text-sm text-base-content/80">{version.change_summary}</div>
                        {/if}
                        <div class="text-xs opacity-50 font-mono uppercase tracking-tighter">
                            ID: {version.id} • Created: {new Date(version.created_at).toLocaleDateString()}
                        </div>
                    </div>

                    <SopVersionStatusBadge status={version.status} />

                    <div class="flex gap-1 items-center">
                        <div class="btn btn-square btn-ghost btn-sm lg:btn-md" aria-label="Open">
                            <ChevronRightIcon />
                        </div>
                    </div>
                </a>
            </li>
        {:else}
            <li class="p-12 text-center">
                <div class="text-sm opacity-40 mb-2">No versions found.</div>
                <div class="text-xs opacity-30">Create a new version to get started.</div>
            </li>
        {/each}
    </ul>
</Card>