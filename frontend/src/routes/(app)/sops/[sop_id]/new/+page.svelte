<script lang="ts">
    import type { PageData, ActionData } from './$types';
    import SopVersionEditor from '$lib/components/editor/SopVersionEditor.svelte';
    import SopEditorAssetSidebar from '$lib/components/editor/SopEditorAssetSidebar.svelte';
    import { HouseIcon, NotebookIcon, NotebookPenIcon, PlusIcon } from 'lucide-svelte';
    import IdBadge from '$lib/components/IdBadge.svelte';
  import Breadcrumbs from '$lib/components/Breadcrumbs.svelte';
  import Card from '$lib/components/Card.svelte';
  import CardPageHeading from '$lib/components/CardPageHeading.svelte';
  import * as m from '$lib/paraglide/messages.js';

    let { data, form } = $props<{ data: PageData, form: ActionData }>();

    /** Shared with editor and sidebar so Word import updates the draft textarea */
    let documentContent = $state('');
</script>

<svelte:head>
    <title>{m.page_new_version()}</title>
</svelte:head>

<div class="flex flex-col gap-6">
    <Breadcrumbs items={[
        { label: m.page_dashboard(), href: '/dashboard', icon: HouseIcon},
        { label: m.page_sops(), href: '/sops', icon: NotebookIcon},
        { label: data.sop.title, href: `/sops/${data.sop.id}`, icon: NotebookIcon},
        { label: m.common_new_version(), icon: PlusIcon},
    ]}/>

    <Card>
        <div class="card-body">
            <CardPageHeading color="accent">
                <NotebookPenIcon class="w-8 h-8" />
                {data.sop.title}
            </CardPageHeading>

            <div class="flex items-center gap-2 text-base-content/70">
                <span class="text-sm font-medium">{m.sops_new_draft()}</span> 
                <IdBadge id={data.sop.id} />
            </div>
        </div>
    </Card>

    <div class="grid grid-cols-1 lg:grid-cols-3 gap-8">
        <div class="lg:col-span-2">
            <SopVersionEditor 
                sopId={data.sop.id} 
                assets={data.assets ?? []}
                bind:content={documentContent}
                {form} 
            />
        </div>

        <div>
            <SopEditorAssetSidebar 
                assets={data.assets ?? []} 
                bind:content={documentContent}
                {form} 
            />
        </div>
    </div>
</div>