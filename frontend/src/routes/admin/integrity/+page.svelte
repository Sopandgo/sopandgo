<script lang="ts">
    import { enhance } from '$app/forms';
    import { resolve } from '$app/paths';
    import Alert from '#lib/components/Alert.svelte';
    import Card from '#lib/components/Card.svelte';
    import CardPageHeading from '#lib/components/CardPageHeading.svelte';
    import { DownloadIcon, LoaderCircleIcon, ShieldCheckIcon } from '@lucide/svelte';
    import * as m from '#lib/paraglide/messages.js';
    import { getLocale } from '#lib/paraglide/runtime.js';
    import type { AdminIntegrity, AnchorFinding } from '#lib/sdk/types.js';

    let { form } = $props();

    let running = $state(false);

    const report = $derived((form?.report ?? null) as AdminIntegrity | null);

    const sections = $derived(
        report
            ? [
                  { label: m.integrity_admin_versions(), ...report.versions },
                  { label: m.integrity_admin_assets(), ...report.assets },
                  { label: m.integrity_admin_audit(), ...report.audit }
              ]
            : []
    );

    const anchoring = $derived(report?.anchoring ?? null);

    const findingLabels: Record<AnchorFinding['status'], () => string> = {
        tampered: m.integrity_anchor_tampered,
        bad_receipt: m.integrity_anchor_bad_receipt,
        missing_from_server: m.integrity_anchor_missing_from_server,
        receipt_differs: m.integrity_anchor_receipt_differs,
        not_uploaded: m.integrity_anchor_not_uploaded,
        replaced_by_restore: m.integrity_anchor_replaced_by_restore
    };

    // A problem fails the scan (error); explained breaks and weaker protection need a look (warning).
    function findingBadge(f: AnchorFinding): string {
        return f.severity === 'problem' ? 'badge-error' : 'badge-warning';
    }

    type FailureRow = { kind: string; href: string; item: string; error: string };

    const failures = $derived<FailureRow[]>(
        report
            ? [
                  ...(report.versions.failed ?? []).map((f) => ({
                      kind: m.integrity_admin_versions(),
                      href: `/sops/${f.sop_id}`,
                      item: m.details_version({ version: String(f.version ?? '?') }),
                      error: f.error
                  })),
                  ...(report.assets.failed ?? []).map((f) => ({
                      kind: m.integrity_admin_assets(),
                      href: `/sops/${f.sop_id}`,
                      item: f.asset_id ?? '',
                      error: f.error
                  })),
                  ...(report.audit.failed ?? []).map((f) => ({
                      kind: m.integrity_admin_audit(),
                      href: '/audit-logs',
                      item: m.integrity_admin_audit_event({ id: String(f.event_id) }),
                      error: f.error
                  }))
              ]
            : []
    );
</script>

<svelte:head>
    <title>{m.page_integrity()}</title>
</svelte:head>

<div class="flex flex-col gap-6">
    <Card>
        <div class="card-body">
            <CardPageHeading>
                <ShieldCheckIcon class="w-8 h-8" />
                {m.integrity_admin_heading()}
            </CardPageHeading>

            <p class="text-sm text-base-content/70">{m.integrity_admin_intro()}</p>
            <p class="text-xs text-base-content/70">{m.integrity_admin_note()}</p>

            <form
                method="POST"
                action="?/run"
                class="card-actions justify-end gap-2 pt-4"
                use:enhance={() => {
                    running = true;
                    return async ({ update }) => {
                        await update();
                        running = false;
                    };
                }}
            >
                <a href={resolve('admin/integrity/evidence')} class="btn" title={m.integrity_evidence_help()} data-sveltekit-reload>
                    <DownloadIcon class="w-4 h-4" />
                    {m.integrity_evidence_download()}
                </a>
                <button type="submit" class="btn btn-primary" disabled={running}>
                    {#if running}
                        <LoaderCircleIcon class="w-5 h-5 animate-spin" />
                        {m.integrity_checking()}
                    {:else}
                        <ShieldCheckIcon class="w-5 h-5" />
                        {m.integrity_admin_run()}
                    {/if}
                </button>
            </form>
        </div>
    </Card>

    {#if form?.error}
        <Alert type="warning" message={form.error} />
    {/if}

    {#if report}
        <div aria-live="polite" class="flex flex-col gap-6">
            <Alert
                type={report.ok ? 'success' : 'error'}
                message={`${report.ok ? m.integrity_admin_ok() : m.integrity_admin_failed()} ${m.integrity_admin_checked_at({
                    when: new Date(report.checked_at).toLocaleString(getLocale())
                })}`}
            />

            <div class="stats stats-vertical sm:stats-horizontal border border-base-300 bg-base-100 w-full">
                {#each sections as section (section.label)}
                    <div class="stat">
                        <div class="stat-title">{section.label}</div>
                        <div class="stat-value {section.invalid > 0 ? 'text-error' : 'text-success'}">
                            {section.valid}/{section.checked}
                        </div>
                        <div class="stat-desc">
                            {m.integrity_admin_invalid({ count: String(section.invalid) })}
                        </div>
                    </div>
                {/each}
            </div>

            {#if anchoring}
                <Card title={m.integrity_anchor_title()} description={m.integrity_anchor_description()}>
                    <div class="card-body space-y-4">
                        {#if anchoring.object_lock === false}
                            <Alert type="warning" message={m.anchoring_object_lock_off()} />
                        {/if}
                        {#if anchoring.s3_error}
                            <Alert type="warning" message={m.integrity_anchor_s3_error({ error: anchoring.s3_error })} />
                        {/if}

                        <div class="stats stats-vertical sm:stats-horizontal border border-base-300 bg-base-100 w-full">
                            <div class="stat">
                                <div class="stat-title">{m.integrity_anchor_server()}</div>
                                <div class="stat-value {anchoring.problems > 0 ? 'text-error' : 'text-success'}">
                                    {anchoring.ok}/{anchoring.anchors}
                                </div>
                                <div class="stat-desc">{m.integrity_anchor_problems({ count: String(anchoring.problems) })}</div>
                            </div>
                            <div class="stat">
                                <div class="stat-title">{m.integrity_anchor_s3()}</div>
                                <div class="stat-value">{anchoring.s3_receipts}</div>
                                <div class="stat-desc">{m.integrity_anchor_review({ count: String(anchoring.explained) })}</div>
                            </div>
                        </div>

                        {#if anchoring.known_broken.length > 0}
                            <p class="text-sm text-base-content/70">
                                {m.integrity_anchor_known_broken({ count: String(anchoring.known_broken.length) })}
                            </p>
                        {/if}

                        {#if anchoring.findings.length > 0}
                            <div class="overflow-x-auto">
                                <table class="table table-sm">
                                    <thead>
                                        <tr>
                                            <th>{m.integrity_anchor_col_status()}</th>
                                            <th>{m.integrity_anchor_col_row()}</th>
                                            <th>{m.integrity_anchor_col_source()}</th>
                                            <th>{m.integrity_anchor_col_detail()}</th>
                                        </tr>
                                    </thead>
                                    <tbody>
                                        {#each anchoring.findings as f, i (i)}
                                            <tr>
                                                <td>
                                                    <span class="badge badge-soft {findingBadge(f)} whitespace-nowrap">{findingLabels[f.status]()}</span>
                                                </td>
                                                <td class="font-mono text-xs">
                                                    {f.event_id}
                                                    <span title={f.tip_hash}>{f.tip_hash.slice(0, 8)}</span>
                                                </td>
                                                <td>{f.source === 's3' ? m.integrity_anchor_source_s3() : m.integrity_anchor_source_server()}</td>
                                                <td class="text-sm">{f.detail}</td>
                                            </tr>
                                        {/each}
                                    </tbody>
                                </table>
                            </div>
                        {/if}
                    </div>
                </Card>
            {/if}

            {#if failures.length > 0}
                <Card title={m.integrity_admin_failures()}>
                    <div class="overflow-x-auto">
                        <table class="table table-sm">
                            <thead>
                                <tr>
                                    <th>{m.integrity_admin_col_area()}</th>
                                    <th>{m.integrity_admin_col_item()}</th>
                                    <th>{m.integrity_admin_col_error()}</th>
                                </tr>
                            </thead>
                            <tbody>
                                {#each failures as failure, i (i)}
                                    <tr>
                                        <td>{failure.kind}</td>
                                        <td>
                                            <a href={failure.href} class="link font-mono text-xs">{failure.item}</a>
                                        </td>
                                        <td class="font-mono text-xs text-error">{failure.error}</td>
                                    </tr>
                                {/each}
                            </tbody>
                        </table>
                    </div>
                </Card>
            {/if}
        </div>
    {/if}
</div>
