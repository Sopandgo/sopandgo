<script lang="ts">
    import MarkdownRenderer from '$lib/components/MarkdownRenderer.svelte';
    import ListAssociatedAssets from '$lib/components/ListAssociatedAssets.svelte';
    import ListAcknowledgmentLog from '$lib/components/ListAcknowledgmentLog.svelte';
    import SopDetails from '$lib/components/SopDetails.svelte';
    import type { PageData, ActionData } from './$types';
    import AcknowledgmentContainerSOPVersion from '$lib/components/Acknowledgment/AcknowledgmentContainerSOPVersion.svelte';
    import { ArrowUpRightIcon, DownloadIcon, FileDiffIcon, FileTextIcon, NotebookTextIcon, PlusIcon } from 'lucide-svelte';
    import FavoriteToggle from '$lib/components/FavoriteToggle.svelte';
    import Card from '$lib/components/Card.svelte';
    import CardPageHeading from '$lib/components/CardPageHeading.svelte';
    import SopVersionStatusBadge from '$lib/components/SopVersionStatusBadge.svelte';
    import { enhance } from '$app/forms';
    import { page } from '$app/state';
    import Alert from '$lib/components/Alert.svelte';
    import VersionDiff from '$lib/components/VersionDiff.svelte';
    import * as m from '$lib/paraglide/messages.js';
    import { getLocale } from '$lib/paraglide/runtime';
    let pdfError = $state<string | null>(null);

    // Receive data and form (ActionData) from the server
    let { data, form }: { data: PageData; form: ActionData } = $props();

    const uid = $props.id();

    let version = $derived(data.versionSummary);
    // The (app) layout only renders for a signed-in user
    let user = $derived(data.user!);
    let role = $derived(user.role);
    let canAuthor = $derived(role === 'editor' || role === 'admin');
    let canApprove = $derived(role === 'approver' || role === 'admin');

    let created = $derived(new Date(version.created_at).toLocaleString(getLocale()));

    // Where readers are pointed instead, when this is not the published version
    let publishedId = $derived(data.sop.published_version_id);
    let publishedNumber = $derived(data.sop.published_version);
    let isPublished = $derived(version.status === 'published');
    // A newer version on its way, for the people who move it along
    let pending = $derived(
        isPublished &&
            (canAuthor || canApprove) &&
            data.sop.latest_version &&
            data.sop.latest_version.id !== version.id &&
            (data.sop.latest_version.status === 'draft' || data.sop.latest_version.status === 'rc')
            ? data.sop.latest_version
            : null
    );
    // Draft and RC are on their way to readers; rejected and superseded are not, so those warn
    let noticeType = $derived<'info' | 'warning'>(
        version.status === 'draft' || version.status === 'rc' ? 'info' : 'warning'
    );

    // Document | Changes. The view lives in the URL so a link can open the diff;
    // an approver reviewing a release candidate starts on the changes.
    let comparable = $derived(!!data.versionDiff?.comparable);
    let added = $derived(data.versionDiff?.lines.filter((l) => l.kind === 'add').length ?? 0);
    let removed = $derived(data.versionDiff?.lines.filter((l) => l.kind === 'del').length ?? 0);
    let defaultView = $derived(comparable && version.status === 'rc' && canApprove ? 'changes' : 'document');
    let view = $derived.by(() => {
        const requested = page.url.searchParams.get('view');
        if (requested === 'changes' && comparable) return 'changes';
        if (requested === 'document') return 'document';
        return defaultView;
    });
    const viewHref = (v: string) => (v === defaultView ? page.url.pathname : `?view=${v}`);

    let promoteError = $derived(form?.action === 'promote' ? form.error : undefined);

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
    <div class="card-body gap-4">
        <div class="flex flex-col gap-4 sm:flex-row sm:items-start sm:justify-between">
            <div class="flex min-w-0 flex-col gap-1">
                <div class="flex items-center gap-1">
                    <CardPageHeading>
                        <NotebookTextIcon class="w-8 h-8" />
                        {data.sop.title}
                    </CardPageHeading>
                    <FavoriteToggle sopId={data.sop.id} title={data.sop.title} isFavorite={!!data.sop.is_favorite} />
                </div>

                <p class="flex flex-wrap items-center gap-x-2 gap-y-1 text-sm text-base-content/70">
                    <SopVersionStatusBadge status={version.status} />
                    <span class="font-mono text-base-content">{m.common_version({ version: String(version.version) })}</span>
                    <span aria-hidden="true">·</span>
                    <span>{created}</span>
                </p>
            </div>

            <div class="flex shrink-0 flex-wrap gap-2">
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

                {#if canAuthor}
                    <a href={`/sops/${data.sop.id}/new`} class="btn">
                        <PlusIcon class="size-4" />
                        {m.common_new_version()}
                    </a>
                {/if}

                {#if version.status === 'draft' && canAuthor}
                    <form method="POST" action="?/promote" use:enhance>
                        <button class="btn btn-primary">
                            <ArrowUpRightIcon class="size-4" />
                            {m.version_promote()}
                        </button>
                    </form>
                {/if}
            </div>
        </div>

        {#if version.change_summary}
            <p>{version.change_summary}</p>
        {/if}

        {#if !isPublished}
            <Alert
                type={noticeType}
                title={version.rejection
                    ? m.version_rejected_by({
                          name: version.rejection.actor_name || version.rejection.actor_user_id,
                          when: new Date(version.rejection.created_at).toLocaleString(getLocale())
                      })
                    : undefined}
            >
                {#if version.rejection}
                    <!-- The reason the approver gave; older rejections have none on record -->
                    {#if version.rejection.reason}
                        <p class="mb-2 whitespace-pre-wrap">{version.rejection.reason}</p>
                    {:else}
                        <p class="mb-2 italic">{m.version_rejected_no_reason()}</p>
                    {/if}
                {/if}
                {#if publishedId && publishedNumber != null}
                    {m.version_readers_follow({ version: String(publishedNumber) })}
                    <a href={`/sops/${data.sop.id}/v/${publishedId}`} class="underline">
                        {m.version_open({ version: String(publishedNumber) })}
                    </a>
                {:else}
                    {m.version_none_published()}
                {/if}
            </Alert>
        {:else if pending}
            <Alert type="info">
                {pending.status === 'rc'
                    ? m.version_pending_rc({ version: String(pending.version) })
                    : m.version_pending_draft({ version: String(pending.version) })}
                <a href={`/sops/${data.sop.id}/v/${pending.id}`} class="underline">
                    {m.version_open({ version: String(pending.version) })}
                </a>
            </Alert>
        {/if}

        {#if promoteError}
            <Alert type="error" message={promoteError} />
        {/if}
        {#if pdfError}
            <Alert type="warning" message={pdfError} />
        {/if}
    </div>
</Card>

<!-- One column until xl: with the app sidebar open, narrower screens leave no room for two -->
<div class="grid items-start gap-6 xl:grid-cols-[minmax(0,1fr)_22rem]">
    <div class="tabs tabs-lift min-w-0" role="tablist" aria-label={m.version_views()}>
        <a
            href={viewHref('document')}
            id="{uid}-tab-document"
            class="tab gap-2"
            class:tab-active={view === 'document'}
            role="tab"
            aria-selected={view === 'document'}
            aria-controls="{uid}-panel"
            data-sveltekit-noscroll
            data-sveltekit-replacestate
        >
            <FileTextIcon class="size-4" aria-hidden="true" />
            {m.version_tab_document()}
        </a>
        {#if comparable}
            <a
                href={viewHref('changes')}
                id="{uid}-tab-changes"
                class="tab gap-2"
                class:tab-active={view === 'changes'}
                role="tab"
                aria-selected={view === 'changes'}
                aria-controls="{uid}-panel"
                data-sveltekit-noscroll
                data-sveltekit-replacestate
            >
                <FileDiffIcon class="size-4" aria-hidden="true" />
                {m.version_tab_changes()}
                <span
                    class="font-mono text-xs"
                    title={m.diff_counts({ added: String(added), removed: String(removed) })}
                >
                    <!-- Signs carry the meaning too, so the colours are never the only cue -->
                    <span aria-hidden="true"><span class="text-success">+{added}</span> <span class="text-error">−{removed}</span></span>
                    <span class="sr-only">{m.diff_counts({ added: String(added), removed: String(removed) })}</span>
                </span>
            </a>
        {/if}
        <div
            id="{uid}-panel"
            class="tab-content block! min-w-0 border-base-300 bg-base-100 p-4 sm:p-6"
            role="tabpanel"
            aria-labelledby="{uid}-tab-{view}"
        >
            {#if view === 'changes' && data.versionDiff}
                <VersionDiff diff={data.versionDiff} assets={version.assets} />
            {:else}
                <!-- Kept to a readable line length when the column is wide -->
                <div class="max-w-3xl">
                    <MarkdownRenderer
                        content={version.content}
                        assets={version.assets}
                    />
                </div>
            {/if}
        </div>
    </div>

    <aside class="flex min-w-0 flex-col gap-6" aria-label={m.version_aside()}>
        <AcknowledgmentContainerSOPVersion
            acks={version.acknowledgments ?? []}
            {user}
            form={form}
            status={version.status ?? ""}
        />

        <ListAcknowledgmentLog items={version.acknowledgments ?? []} limit={5} />

        <ListAssociatedAssets items={version.assets ?? []} />

        <SopDetails
            sopId={data.sop.id}
            versionId={version.id}
            contentHash={version.content_hash}
            createdAt={version.created_at}
            hashValid={version.hash_valid}
            status={version.status}
            version={version.version}
        />
    </aside>
</div>
</div>
