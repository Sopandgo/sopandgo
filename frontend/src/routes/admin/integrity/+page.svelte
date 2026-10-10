<script lang="ts">
    import { enhance } from '$app/forms';
    import Alert from '$lib/components/Alert.svelte';
    import Card from '$lib/components/Card.svelte';
    import CardPageHeading from '$lib/components/CardPageHeading.svelte';
    import { LoaderCircleIcon, ShieldCheckIcon } from '@lucide/svelte';
    import * as m from '$lib/paraglide/messages.js';
    import { getLocale } from '$lib/paraglide/runtime';
    import type { AdminIntegrity } from '$lib/sdk/types';

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
                      href: '/admin/audit-logs',
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
                class="card-actions justify-end pt-4"
                use:enhance={() => {
                    running = true;
                    return async ({ update }) => {
                        await update();
                        running = false;
                    };
                }}
            >
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
