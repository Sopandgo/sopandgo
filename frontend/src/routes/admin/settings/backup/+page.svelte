<script lang="ts">
    import { enhance } from '$app/forms';
    import { resolve } from '$app/paths';
    import type { SubmitFunction } from '@sveltejs/kit';
    import Alert from '$lib/components/Alert.svelte';
    import Card from '$lib/components/Card.svelte';
    import CardPageHeading from '$lib/components/CardPageHeading.svelte';
    import CollapsibleCard from '$lib/components/CollapsibleCard.svelte';
    import { ArchiveIcon, CalendarClock, History } from 'lucide-svelte';
    import * as m from '$lib/paraglide/messages.js';
    import { getLocale } from '$lib/paraglide/runtime';

    let { data, form }: { data: any; form: any } = $props();

    const backupStatus = $derived(data.backupStatus);
    const importBackupResult = $derived(
        form?.importBackup as
            | {
                    error?: string;
                    ok?: boolean;
                    note?: string;
                    manifest?: { app_version: string; db_schema_version: number; created_at_utc: string };
              }
            | undefined
    );
    const applyBackupResult = $derived(
        form?.applyBackup as
            | {
                    error?: string;
                    ok?: boolean;
                    note?: string;
                    requires_restart?: boolean;
                    pre_apply_backup?: string;
                    manifest?: { app_version: string; db_schema_version: number; created_at_utc: string };
              }
            | undefined
    );
    const pendingRestart = $derived(backupStatus?.pending_restore ?? false);
    const uploadsLocked = $derived((backupStatus?.locked ?? false) || pendingRestart);

    // Which upload is in flight, for its button's spinner.
    let pending = $state<'validate' | 'apply' | null>(null);

    function submitting(which: NonNullable<typeof pending>): SubmitFunction {
        return () => {
            pending = which;
            return async ({ update }) => {
                await update();
                pending = null;
            };
        };
    }

    function formatUtc(iso?: string): string {
        if (!iso) return '—';
        const d = new Date(iso);
        if (Number.isNaN(d.getTime())) return iso;
        return `${d.toLocaleString(getLocale(), {
            dateStyle: 'medium',
            timeStyle: 'medium',
            timeZone: 'UTC'
        })} UTC`;
    }

    /** Human label for Go duration strings from the API (e.g. `2m0s`, `24h0m0s`). */
    function every(n: number, one: () => string, many: (count: string) => string): string {
        return n === 1 ? one() : many(String(n));
    }

    function intervalSummary(interval?: string): string {
        if (!interval) return '—';
        const h = interval.match(/^(\d+)h/);
        if (h) {
            const n = parseInt(h[1], 10);
            return every(n, m.backup_every_hour, (count) => m.backup_every_hours({ n: count }));
        }
        const minutes = interval.match(/^(\d+)m/);
        if (minutes) {
            const n = parseInt(minutes[1], 10);
            return every(n, m.backup_every_minute, (count) => m.backup_every_minutes({ n: count }));
        }
        const seconds = interval.match(/^(\d+)s/);
        if (seconds) {
            const n = parseInt(seconds[1], 10);
            return every(n, m.backup_every_second, (count) => m.backup_every_seconds({ n: count }));
        }
        return interval;
    }

    /** Collapsed-row summary: schedule and destination, e.g. "Every 24 hours · my-bucket/lab". */
    function s3Meta(intervalLabel: string, bucket?: string, prefix?: string): string {
        const dest = [bucket, prefix?.trim()].filter(Boolean).join('/');
        return dest ? `${intervalLabel} · ${dest}` : intervalLabel;
    }
</script>

<svelte:head>
    <title>{m.page_backup()}</title>
</svelte:head>

{#snippet buttonLabel(busy: boolean, idle: string, busyLabel: string)}
    {#if busy}
        <span class="loading loading-spinner loading-xs"></span>
        {busyLabel}
    {:else}
        {idle}
    {/if}
{/snippet}

<div class="flex flex-col gap-6">
    <Card>
        <div class="card-body space-y-4">
            <CardPageHeading>
                <ArchiveIcon class="w-8 h-8" />
                {m.page_backup()}
            </CardPageHeading>

            <p class="text-sm text-base-content/70">{m.backup_intro()}</p>

            {#if backupStatus?.locked}
                <Alert type="warning" message={m.backup_lock({ detail: backupStatus.message || m.backup_in_progress() })} />
            {/if}
            {#if pendingRestart}
                <Alert type="warning" message={m.backup_restart_wait()} />
            {/if}

            <div class="flex flex-col border-t border-base-300 pt-4 sm:flex-row sm:justify-end">
                <a href={resolve('/admin/settings/backup/export')} class="btn btn-primary w-full sm:w-auto">{m.backup_export()}</a>
            </div>
        </div>
    </Card>

    {#if backupStatus?.s3_scheduled}
        {@const s3 = backupStatus.s3_scheduled}
        {@const intervalLabel = intervalSummary(s3.interval)}
        <CollapsibleCard
            title={m.backup_s3_title()}
            meta={s3.enabled ? s3Meta(intervalLabel, s3.bucket, s3.key_prefix) : undefined}
            openWhen={Boolean(s3.last_error)}
        >
            {#snippet trailing()}
                {#if s3.last_error}
                    <span class="badge badge-soft badge-error">{m.backup_upload_failed()}</span>
                {/if}
                {#if s3.enabled}
                    <span class="badge badge-soft badge-success">{m.backup_enabled()}</span>
                {:else}
                    <span class="badge badge-outline">{m.backup_disabled_badge()}</span>
                {/if}
            {/snippet}

            <p class="max-w-prose text-sm text-base-content/70">
                {m.backup_s3_help_before()}
                <code class="rounded bg-base-200 px-1 py-0.5 text-xs">BACKUP_S3_*</code>
                {m.backup_s3_help_after()}
            </p>

            {#if s3.enabled}
                {#if s3.last_error}
                    <Alert type="error" title={m.backup_last_error()} message={s3.last_error} />
                {/if}

                <div class="stats stats-vertical border border-base-300 bg-base-100 w-full sm:stats-horizontal">
                    <div class="stat place-items-start border-base-300 py-4 sm:border-e">
                        <div class="stat-title text-sm font-medium text-base-content/70">{m.backup_bucket()}</div>
                        <div class="stat-value font-mono text-lg font-normal break-all text-start leading-snug">
                            {s3.bucket ?? '—'}
                        </div>
                    </div>
                    <div class="stat place-items-start border-base-300 py-4 sm:border-e">
                        <div class="stat-title text-sm font-medium text-base-content/70">{m.backup_prefix()}</div>
                        <div class="stat-value font-mono text-base font-normal break-all text-start leading-snug">
                            {#if s3.key_prefix?.trim()}
                                {s3.key_prefix}
                            {:else}
                                <span class="text-base-content/70">{m.backup_bucket_root()}</span>
                            {/if}
                        </div>
                    </div>
                    <div class="stat place-items-start py-4">
                        <div class="stat-title text-sm font-medium text-base-content/70">{m.backup_schedule()}</div>
                        <div class="stat-value text-lg font-normal text-start leading-snug">{intervalLabel}</div>
                        {#if s3.interval && s3.interval !== intervalLabel}
                            <div class="stat-desc font-mono text-xs text-base-content/70">{s3.interval}</div>
                        {/if}
                    </div>
                </div>

                {#if s3.retention_max_objects || s3.retention_days}
                    <div class="rounded-box border border-base-300 bg-base-200 p-4">
                        <div class="mb-3 text-sm font-medium text-base-content/70">{m.backup_retention()}</div>
                        <ul class="flex flex-col gap-3 sm:flex-row sm:flex-wrap sm:gap-4">
                            {#if s3.retention_max_objects}
                                <li class="flex items-start gap-2 text-sm">
                                    <span class="badge badge-outline shrink-0">{m.backup_count()}</span>
                                    <span>
                                        {Number(s3.retention_max_objects) === 1
                                            ? m.backup_keep_one()
                                            : m.backup_keep_many({ n: String(s3.retention_max_objects) })}
                                    </span>
                                </li>
                            {/if}
                            {#if s3.retention_days}
                                <li class="flex items-start gap-2 text-sm">
                                    <span class="badge badge-outline shrink-0">{m.backup_age()}</span>
                                    <span>
                                        {Number(s3.retention_days) === 1
                                            ? m.backup_age_one()
                                            : m.backup_age_many({ n: String(s3.retention_days) })}
                                    </span>
                                </li>
                            {/if}
                        </ul>
                    </div>
                {/if}

                <h3 class="text-sm font-medium text-base-content/70">{m.backup_recent()}</h3>

                <div class="grid gap-4 lg:grid-cols-2">
                    <div class="rounded-box border border-base-300 bg-base-100 p-4">
                        <div class="mb-2 flex items-center gap-2 text-sm font-medium text-base-content/70">
                            <History class="size-4" aria-hidden="true" />
                            {m.backup_last_success()}
                        </div>
                        {#if s3.last_success_utc}
                            <p class="text-lg font-medium leading-snug">{formatUtc(s3.last_success_utc)}</p>
                            {#if s3.last_object_key}
                                <div class="mt-3">
                                    <div class="mb-1 text-xs text-base-content/70">{m.backup_object_key()}</div>
                                    <code class="block w-full overflow-x-auto rounded-field border border-base-300 bg-base-200 px-3 py-2 font-mono text-xs leading-relaxed">
                                        {s3.last_object_key}
                                    </code>
                                </div>
                            {/if}
                        {:else if s3.last_run_utc}
                            <p class="text-sm text-base-content/70">
                                {m.backup_last_run({ when: formatUtc(s3.last_run_utc) })}
                            </p>
                        {:else}
                            <p class="text-sm text-base-content/70">{m.backup_no_upload()}</p>
                        {/if}
                    </div>

                    <div class="rounded-box border border-base-300 bg-base-100 p-4">
                        <div class="mb-2 flex items-center gap-2 text-sm font-medium text-base-content/70">
                            <CalendarClock class="size-4" aria-hidden="true" />
                            {m.backup_next()}
                        </div>
                        {#if s3.next_run_utc}
                            <p class="text-lg font-medium leading-snug">{formatUtc(s3.next_run_utc)}</p>
                            <p class="mt-2 text-xs text-base-content/70">{m.backup_next_help()}</p>
                        {:else}
                            <p class="text-sm text-base-content/70">—</p>
                        {/if}
                    </div>
                </div>
            {:else}
                <Alert type="info">
                    {m.backup_s3_off_before()}
                    <code class="mx-0.5 rounded bg-base-200 px-1 py-0.5 text-xs">BACKUP_S3_ENABLED</code>
                    {m.backup_s3_off_after()}
                </Alert>
            {/if}
        </CollapsibleCard>
    {/if}

    <CollapsibleCard
        title={m.backup_validate_title()}
        meta={m.backup_validate_help()}
        openWhen={Boolean(importBackupResult)}
    >
        {#if importBackupResult?.error}
            <Alert type="error" message={importBackupResult.error} />
        {/if}
        {#if importBackupResult?.ok}
            <Alert
                type="success"
                message={m.backup_validated({
                    version: importBackupResult.manifest?.app_version ?? '',
                    schema: String(importBackupResult.manifest?.db_schema_version ?? '')
                })}
            />
            {#if importBackupResult.note}
                <Alert type="info" message={importBackupResult.note} />
            {/if}
        {/if}

        <form
            method="POST"
            action="?/importBackup"
            use:enhance={submitting('validate')}
            enctype="multipart/form-data"
            class="grid grid-cols-1 gap-5"
        >
            <label class="flex flex-col gap-1.5">
                <span class="text-sm font-medium">{m.backup_archive_label()}</span>
                <input
                    type="file"
                    name="backup_file"
                    class="file-input w-full"
                    accept=".zip,application/zip"
                    disabled={uploadsLocked}
                />
            </label>
            <div class="flex flex-col border-t border-base-300 pt-4 sm:flex-row sm:justify-end">
                <button type="submit" class="btn w-full sm:w-auto" disabled={uploadsLocked || pending !== null}>
                    {@render buttonLabel(pending === 'validate', m.backup_validate_button(), m.backup_validating())}
                </button>
            </div>
        </form>
    </CollapsibleCard>

    <CollapsibleCard title={m.backup_apply_title()} openWhen={Boolean(applyBackupResult)}>
        {#snippet trailing()}
            {#if pendingRestart}
                <span class="badge badge-soft badge-warning">{m.backup_restart_pending()}</span>
            {/if}
        {/snippet}

        <Alert type="warning" message={m.backup_apply_warning()} />
        {#if applyBackupResult?.error}
            <Alert type="error" message={applyBackupResult.error} />
        {/if}
        {#if applyBackupResult?.ok}
            <Alert
                type="success"
                message={m.backup_apply_staged({
                    restart: applyBackupResult.requires_restart ? m.common_yes() : m.common_no(),
                    snapshot: applyBackupResult.pre_apply_backup ?? ''
                })}
            />
            {#if applyBackupResult.note}
                <Alert type="info" message={applyBackupResult.note} />
            {/if}
        {/if}

        <form
            method="POST"
            action="?/applyBackup"
            use:enhance={submitting('apply')}
            enctype="multipart/form-data"
            class="grid grid-cols-1 gap-5"
        >
            <label class="flex flex-col gap-1.5">
                <span class="text-sm font-medium">{m.backup_archive_label()}</span>
                <input
                    type="file"
                    name="backup_file"
                    class="file-input w-full"
                    accept=".zip,application/zip"
                    disabled={uploadsLocked}
                />
            </label>
            <label class="flex flex-col gap-1.5">
                <span class="text-sm font-medium">{m.backup_confirm_label()}</span>
                <input
                    type="text"
                    name="confirmation"
                    class="input w-full font-mono"
                    placeholder="APPLY BACKUP"
                    autocomplete="off"
                    disabled={uploadsLocked}
                />
            </label>
            <div class="flex flex-col border-t border-base-300 pt-4 sm:flex-row sm:justify-end">
                <button type="submit" class="btn btn-error w-full sm:w-auto" disabled={uploadsLocked || pending !== null}>
                    {@render buttonLabel(pending === 'apply', m.backup_stage_button(), m.backup_staging())}
                </button>
            </div>
        </form>
    </CollapsibleCard>
</div>
