<script lang="ts">
    import MarkdownRenderer from '$lib/components/MarkdownRenderer.svelte';
    import ListAssociatedAssets from '$lib/components/ListAssociatedAssets.svelte';
    import SopDetails from '$lib/components/SopDetails.svelte';
    import type { PageData, ActionData } from './$types';
    import AcknowledgmentContainerSOPVersion from '$lib/components/Acknowledgment/AcknowledgmentContainerSOPVersion.svelte';
    import { DownloadIcon, NotebookTextIcon, PlusIcon } from 'lucide-svelte';
    import FavoriteToggle from '$lib/components/FavoriteToggle.svelte';
    import Card from '$lib/components/Card.svelte';
    import CardPageHeading from '$lib/components/CardPageHeading.svelte';
    import SopVersionStatusBadge from '$lib/components/SopVersionStatusBadge.svelte';
    import { ArrowUpRightIcon } from 'lucide-svelte';
    import { enhance } from '$app/forms';
    import Alert from '$lib/components/Alert.svelte';
    import VersionDiff from '$lib/components/VersionDiff.svelte';
    import * as m from '$lib/paraglide/messages.js';
    let pdfError = $state<string | null>(null);

    // Receive data and form (ActionData) from the server
    let { data, form } = $props<{ data: PageData, form: ActionData }>();

    const downloadGeneratedPdf = async () => {
        if (!data.pdf.enabled) {
            pdfError = m.pdf_disabled();
            return;
        }
        pdfError = null;
        try {
            const res = await fetch(data.pdf.downloadUrl, { method: 'GET' });
            if (!res.ok) {
                pdfError = res.status === 404
                    ? m.pdf_not_ready()
                    : m.pdf_download_failed();
                return;
            }

            const blob = await res.blob();
            const fileName =
                res.headers.get('content-disposition')?.match(/filename="?([^"]+)"?/)?.[1] ??
                `sop_version_${data.versionSummary.id}.pdf`;
            const objectUrl = URL.createObjectURL(blob);
            const a = document.createElement('a');
            a.href = objectUrl;
            a.download = fileName;
            document.body.appendChild(a);
            a.click();
            document.body.removeChild(a);
            URL.revokeObjectURL(objectUrl);
        } catch {
            pdfError = m.pdf_download_failed();
        }
    };
</script>

<svelte:head>
    <title>{m.page_sop_version()}</title>
</svelte:head>


<div class="flex flex-col gap-6">
<Card>
    <div class="card-body">
        <div class="flex items-center gap-1">
            <CardPageHeading>
                <NotebookTextIcon class="w-8 h-8" />
                {data.sop.title}
            </CardPageHeading>
            <FavoriteToggle sopId={data.sop.id} title={data.sop.title} isFavorite={!!data.sop.is_favorite} />
        </div>

        <div class="flex flex-col gap-2 text-base-content/70">
            {#if data.versionSummary.status != 'published'}
                <SopVersionStatusBadge status={data.versionSummary.status} />
            {:else}
                <p>{m.version_active()}</p>
            {/if}
            {#if data.versionSummary.change_summary}
                <p class="text-base-content">{data.versionSummary.change_summary}</p>
            {/if}
        </div>

        <div class="card-actions justify-end pt-4 border-t border-base-300 mt-2">
            {#if data.pdf.enabled}
                <button type="button" class="btn" onclick={downloadGeneratedPdf}>
                    <DownloadIcon class="size-4" />
                    {m.version_download_pdf()}
                </button>
            {:else}
                <button type="button" class="btn" disabled>
                    {m.version_pdf_disabled()}
                </button>
            {/if}
            
            {#if data.versionSummary.status === 'draft' && (data.user.role === 'editor' || data.user.role === 'admin')}
                <form method="POST" action="?/promote" use:enhance>
                    <button class="btn btn-primary">
                        <ArrowUpRightIcon class="size-4" />
                        {m.version_promote()}
                    </button>
                </form>
            {/if}

            {#if (data.user.role === 'admin' || data.user.role === 'editor')}
                <a href={`/sops/${data.sop.id}/new`} class="btn">
                    <PlusIcon class="size-4" />
                    {m.common_new_version()}
                </a>
            {/if}

        </div>
        {#if pdfError}
            <div class="mt-2">
                <Alert type="warning" message={pdfError} />
            </div>
        {/if}
    </div>
</Card>

    <VersionDiff diff={data.versionDiff} />

    <SopDetails 
        sopId={data.sop.id} 
        versionId={data.versionSummary.id}
        contentHash={data.versionSummary.content_hash} 
        createdAt={data.versionSummary.created_at} 
        hashValid={data.versionSummary.hash_valid} 
        status={data.versionSummary.status}
        version={data.versionSummary.version} 
    />

    {#if data.versionSummary.status != 'published'}
        <Alert type='warning' message={m.version_inactive_warning()}/>
    {/if}
    
    <Card title={m.version_document()}>
        <div class="card-body">
            <MarkdownRenderer 
                content={data.versionSummary.content}
                assets={data.versionSummary.assets}
            />
        </div>
    </Card>

    {#if data.versionSummary.status != 'published'}
        <Alert type='warning' message={m.version_inactive()}/>
    {/if}

    <ListAssociatedAssets items={data.versionSummary.assets ?? []} />

    <AcknowledgmentContainerSOPVersion 
        acks={data.versionSummary.acknowledgments ?? []}
        user={data.user}
        form={form}
        status={data.versionSummary.status ?? ""}
    />
</div>