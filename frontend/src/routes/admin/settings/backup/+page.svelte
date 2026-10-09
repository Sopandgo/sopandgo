<script lang="ts">
    import { enhance } from '$app/forms';
    import { resolve } from '$app/paths';
    import type { SubmitFunction } from '@sveltejs/kit';
    import Alert from '$lib/components/Alert.svelte';
    import Card from '$lib/components/Card.svelte';
    import CardPageHeading from '$lib/components/CardPageHeading.svelte';
    import CollapsibleCard from '$lib/components/CollapsibleCard.svelte';
    import ConfirmDialog from '$lib/components/ConfirmDialog.svelte';
    import { ArchiveIcon, CalendarClock, History } from 'lucide-svelte';
    import * as m from '$lib/paraglide/messages.js';
    import { getLocale } from '$lib/paraglide/runtime';
    import type { BackupS3Settings, S3ScheduledBackupStatus } from '$lib/sdk/types';

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

    type ActionResult = { error?: string; ok?: boolean } | undefined;
    const s3Settings = $derived(data.s3Settings as BackupS3Settings | null);
    const s3Status = $derived(backupStatus?.s3_scheduled as S3ScheduledBackupStatus | undefined);
    const saveS3Result = $derived(form?.saveS3 as ActionResult);
    const setS3EnabledResult = $derived(form?.setS3Enabled as ActionResult);
    const testS3Result = $derived(form?.testS3 as ActionResult);
    const runS3Result = $derived(form?.runS3 as (ActionResult & { objectKey?: string }) | undefined);

    // The header toggle flips at once and follows the saved value after each reload.
    const savedS3Enabled = () => s3Settings?.enabled ?? false;
    let s3Enabled = $state(savedS3Enabled());
    $effect(() => {
        s3Enabled = savedS3Enabled();
    });

    // Schedule presets as Go durations (the API's format); a saved custom value stays selectable.
    const intervalPresets = ['1h0m0s', '6h0m0s', '12h0m0s', '24h0m0s', '168h0m0s'];
    const intervalOptions = $derived(
        s3Settings?.interval && !intervalPresets.includes(s3Settings.interval)
            ? [...intervalPresets, s3Settings.interval]
            : intervalPresets
    );

    // Which form is in flight, for its button's spinner.
    let pending = $state<'validate' | 'apply' | 's3save' | 's3test' | 's3run' | null>(null);

    function submitting(which: NonNullable<typeof pending>): SubmitFunction {
        return () => {
            pending = which;
            return async ({ update }) => {
                await update();
                pending = null;
            };
        };
    }

    // Apply: the file is chosen in the card, the typed confirmation happens in the dialog.
    let applyDialogOpen = $state(false);
    let applyFileName = $state('');
    let applyConfirmation = $state('');

    const applySubmit: SubmitFunction = () => {
        pending = 'apply';
        return async ({ update }) => {
            await update();
            pending = null;
            applyDialogOpen = false;
            applyConfirmation = '';
            applyFileName = '';
        };
    };

    // One S3 form, three buttons: Save, Test connection and Back up now (both use the saved settings).
    const s3Submit: SubmitFunction = ({ submitter }) => {
        const action = submitter?.getAttribute('formaction');
        pending = action === '?/testS3' ? 's3test' : action === '?/runS3' ? 's3run' : 's3save';
        return async ({ update }) => {
            // Testing or running must not wipe edits the admin has not saved yet.
            await update({ reset: pending === 's3save' });
            pending = null;
        };
    };

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
            const wholeHours = /^\d+h(0m0s)?$/.test(interval);
            if (wholeHours && n === 24) return m.backup_every_day();
            if (wholeHours && n === 168) return m.backup_every_week();
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

    {#snippet s3ToggleError()}
        <Alert type="error" message={setS3EnabledResult?.error} />
    {/snippet}

    <CollapsibleCard
        title={m.backup_s3_title()}
        meta={s3Settings?.configured ? s3Meta(intervalSummary(s3Settings.interval), s3Settings.bucket, s3Settings.key_prefix) : undefined}
        openWhen={Boolean(saveS3Result || testS3Result || runS3Result || s3Status?.last_error)}
        notice={setS3EnabledResult?.error ? s3ToggleError : undefined}
    >
        {#snippet trailing()}
            {#if s3Status?.last_error}
                <span class="badge badge-soft badge-error">{m.backup_upload_failed()}</span>
            {/if}
            {#if s3Settings?.configured}
                <form
                    method="POST"
                    action="?/setS3Enabled"
                    use:enhance={() => {
                        return async ({ result, update }) => {
                            await update({ reset: false });
                            // On failure the data does not change, so put the toggle back by hand.
                            if (result.type === 'failure' || result.type === 'error') {
                                s3Enabled = s3Settings?.enabled ?? false;
                            }
                        };
                    }}
                >
                    <label class="flex cursor-pointer items-center gap-2 text-sm">
                        <!-- The word repeats the toggle's state for sighted users; the checkbox conveys it to assistive tech. -->
                        <span class="text-base-content/70" aria-hidden="true">
                            {s3Enabled ? m.settings_integration_on() : m.settings_integration_off()}
                        </span>
                        <input
                            type="checkbox"
                            name="enabled"
                            class="toggle toggle-success"
                            aria-label={m.backup_s3_enable()}
                            bind:checked={s3Enabled}
                            onchange={(e) => e.currentTarget.form?.requestSubmit()}
                        />
                    </label>
                </form>
            {:else}
                <span class="badge badge-outline">{m.common_not_configured()}</span>
            {/if}
        {/snippet}

        <p class="max-w-prose text-sm text-base-content/70">{m.backup_s3_help()}</p>

        {#if s3Settings && !s3Settings.encryption_key_set}
            <Alert type="info" message={m.backup_s3_key_missing()} />
        {/if}
        {#if saveS3Result?.error}
            <Alert type="error" message={saveS3Result.error} />
        {:else if saveS3Result?.ok}
            <Alert type="success" message={m.backup_s3_saved()} />
        {/if}
        {#if testS3Result?.error}
            <Alert type="error" message={testS3Result.error} />
        {:else if testS3Result?.ok}
            <Alert type="success" message={m.backup_s3_test_ok()} />
        {/if}
        {#if runS3Result?.error}
            <Alert type="error" message={runS3Result.error} />
        {:else if runS3Result?.ok}
            <Alert type="success" message={m.backup_s3_run_ok({ key: runS3Result.objectKey ?? '' })} />
        {/if}
        {#if s3Status?.last_error && !runS3Result}
            <Alert type="error" title={m.backup_last_error()} message={s3Status.last_error} />
        {/if}

        {#if s3Status?.enabled}
            <div class="grid gap-4 lg:grid-cols-2">
                <div class="rounded-box border border-base-300 bg-base-100 p-4">
                    <div class="mb-2 flex items-center gap-2 text-sm font-medium text-base-content/70">
                        <History class="size-4" aria-hidden="true" />
                        {m.backup_last_success()}
                    </div>
                    {#if s3Status.last_success_utc}
                        <p class="text-lg font-medium leading-snug">{formatUtc(s3Status.last_success_utc)}</p>
                        {#if s3Status.last_object_key}
                            <div class="mt-3">
                                <div class="mb-1 text-xs text-base-content/70">{m.backup_object_key()}</div>
                                <code class="block w-full overflow-x-auto rounded-field border border-base-300 bg-base-200 px-3 py-2 font-mono text-xs leading-relaxed">
                                    {s3Status.last_object_key}
                                </code>
                            </div>
                        {/if}
                    {:else if s3Status.last_run_utc}
                        <p class="text-sm text-base-content/70">
                            {m.backup_last_run({ when: formatUtc(s3Status.last_run_utc) })}
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
                    {#if s3Status.next_run_utc}
                        <p class="text-lg font-medium leading-snug">{formatUtc(s3Status.next_run_utc)}</p>
                        <p class="mt-2 text-xs text-base-content/70">{m.backup_next_help()}</p>
                    {:else}
                        <p class="text-sm text-base-content/70">—</p>
                    {/if}
                </div>
            </div>
        {/if}

        <form
            method="POST"
            action="?/saveS3"
            use:enhance={s3Submit}
            class="grid grid-cols-1 gap-5 md:grid-cols-2"
        >
            {#if s3Settings?.configured}
                <!-- The header toggle owns `enabled`; saving the fields keeps it as it is. -->
                {#if s3Enabled}
                    <input type="hidden" name="enabled" value="on" />
                {/if}
            {:else}
                <label class="flex w-fit cursor-pointer items-center gap-3 text-sm md:col-span-2">
                    <input
                        type="checkbox"
                        name="enabled"
                        class="toggle toggle-success peer"
                        checked={s3Settings?.enabled ?? false}
                    />
                    <span class="font-medium">{m.backup_s3_enable()}</span>
                    <!-- The word repeats the toggle's state for sighted users; the checkbox conveys it to assistive tech. -->
                    <span class="text-base-content/70 peer-checked:hidden" aria-hidden="true">
                        {m.settings_integration_off()}
                    </span>
                    <span class="hidden text-base-content/70 peer-checked:inline" aria-hidden="true">
                        {m.settings_integration_on()}
                    </span>
                </label>
            {/if}

            <label class="flex flex-col gap-1.5">
                <span class="text-sm font-medium">{m.backup_bucket()}</span>
                <input
                    type="text"
                    name="bucket"
                    class="input w-full font-mono"
                    autocomplete="off"
                    placeholder="lab-backups"
                    value={s3Settings?.bucket ?? ''}
                />
            </label>

            <label class="flex flex-col gap-1.5">
                <span class="text-sm font-medium">{m.backup_s3_region()}</span>
                <input
                    type="text"
                    name="region"
                    class="input w-full font-mono"
                    autocomplete="off"
                    placeholder="eu-central-1"
                    value={s3Settings?.region ?? ''}
                />
            </label>

            <label class="flex flex-col gap-1.5">
                <span class="text-sm font-medium">{m.backup_prefix()}</span>
                <input
                    type="text"
                    name="key_prefix"
                    class="input w-full font-mono"
                    autocomplete="off"
                    placeholder="myorg/prod"
                    value={s3Settings?.key_prefix ?? ''}
                />
                <span class="text-xs text-base-content/70">{m.backup_s3_prefix_help()}</span>
            </label>

            <label class="flex flex-col gap-1.5">
                <span class="text-sm font-medium">{m.backup_s3_endpoint()}</span>
                <input
                    type="url"
                    name="endpoint"
                    class="input w-full font-mono"
                    autocomplete="off"
                    placeholder="https://minio.example.com"
                    value={s3Settings?.endpoint ?? ''}
                />
                <span class="text-xs text-base-content/70">{m.backup_s3_endpoint_help()}</span>
            </label>

            <label class="flex w-fit cursor-pointer items-center gap-3 text-sm md:col-span-2">
                <input
                    type="checkbox"
                    name="use_path_style"
                    class="checkbox checkbox-sm checkbox-primary"
                    checked={s3Settings?.use_path_style ?? false}
                />
                <span>{m.backup_s3_path_style()}</span>
            </label>

            <fieldset class="grid grid-cols-1 gap-5 md:col-span-2 md:grid-cols-2">
                <legend class="mb-1 text-sm font-medium md:col-span-2">{m.backup_s3_credentials()}</legend>
                <p class="-mt-3 text-xs text-base-content/70 md:col-span-2">{m.backup_s3_credentials_help()}</p>
                <label class="flex flex-col gap-1.5">
                    <span class="text-sm font-medium">{m.backup_s3_access_key()}</span>
                    <input
                        type="text"
                        name="access_key_id"
                        class="input w-full font-mono"
                        autocomplete="off"
                        value={s3Settings?.access_key_id ?? ''}
                    />
                </label>
                <label class="flex flex-col gap-1.5">
                    <span class="text-sm font-medium">{m.backup_s3_secret_key()}</span>
                    <input
                        type="password"
                        name="secret_access_key"
                        class="input w-full"
                        autocomplete="new-password"
                        placeholder={s3Settings?.secret_configured ? m.settings_keep_key() : m.common_optional()}
                    />
                </label>
            </fieldset>

            <fieldset class="grid grid-cols-1 gap-5 md:col-span-2 md:grid-cols-3">
                <legend class="mb-1 text-sm font-medium md:col-span-3">{m.backup_retention_schedule()}</legend>
                <label class="flex flex-col gap-1.5">
                    <span class="text-sm font-medium">{m.backup_schedule()}</span>
                    <select name="interval" class="select w-full" value={s3Settings?.interval ?? '24h0m0s'}>
                        {#each intervalOptions as option (option)}
                            <option value={option}>{intervalSummary(option)}</option>
                        {/each}
                    </select>
                </label>
                <label class="flex flex-col gap-1.5">
                    <span class="text-sm font-medium">{m.backup_s3_retention_max()}</span>
                    <input
                        type="number"
                        name="retention_max"
                        class="input w-full"
                        min="0"
                        step="1"
                        value={s3Settings?.retention_max ?? 14}
                    />
                    <span class="text-xs text-base-content/70">{m.backup_s3_retention_max_help()}</span>
                </label>
                <label class="flex flex-col gap-1.5">
                    <span class="text-sm font-medium">{m.backup_s3_retention_days()}</span>
                    <input
                        type="number"
                        name="retention_days"
                        class="input w-full"
                        min="0"
                        step="1"
                        value={s3Settings?.retention_days ?? 30}
                    />
                    <span class="text-xs text-base-content/70">{m.backup_s3_retention_days_help()}</span>
                </label>
            </fieldset>

            <div
                class="flex flex-col-reverse gap-3 border-t border-base-300 pt-4 sm:flex-row sm:items-center sm:justify-between md:col-span-2"
            >
                <p class="text-xs text-base-content/70">{m.backup_s3_test_hint()}</p>
                <!-- Save comes first in the DOM so Enter in a field saves rather than tests. -->
                <div class="flex flex-col gap-2 sm:flex-row-reverse">
                    <button type="submit" class="btn w-full sm:w-auto" disabled={pending !== null}>
                        {@render buttonLabel(pending === 's3save', m.backup_s3_save(), m.common_saving())}
                    </button>
                    {#if s3Settings?.configured}
                        <button
                            type="submit"
                            formaction="?/testS3"
                            class="btn btn-ghost w-full sm:w-auto"
                            disabled={pending !== null}
                        >
                            {@render buttonLabel(pending === 's3test', m.backup_s3_test(), m.backup_s3_testing())}
                        </button>
                    {/if}
                    {#if s3Status?.enabled}
                        <button
                            type="submit"
                            formaction="?/runS3"
                            class="btn btn-ghost w-full sm:w-auto"
                            disabled={pending !== null}
                        >
                            {@render buttonLabel(pending === 's3run', m.backup_s3_run(), m.backup_s3_running())}
                        </button>
                    {/if}
                </div>
            </div>
        </form>
    </CollapsibleCard>

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

        <p class="text-sm text-base-content/70">{m.backup_apply_help()}</p>
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
            use:enhance={applySubmit}
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
                    required
                    disabled={uploadsLocked}
                    onchange={(e) => (applyFileName = e.currentTarget.files?.[0]?.name ?? '')}
                />
            </label>
            <div class="flex flex-col border-t border-base-300 pt-4 sm:flex-row sm:justify-end">
                <!-- Opens the confirmation; only the dialog's button is the destructive one. -->
                <button
                    type="button"
                    class="btn text-error w-full sm:w-auto"
                    disabled={uploadsLocked || pending !== null}
                    onclick={(e) => {
                        if (e.currentTarget.form?.reportValidity()) applyDialogOpen = true;
                    }}
                >
                    {m.backup_apply_open()}
                </button>
            </div>

            <ConfirmDialog title={m.backup_apply_dialog_title()} bind:open={applyDialogOpen} busy={pending === 'apply'}>
                <Alert type="warning" message={m.backup_apply_warning()} />
                {#if applyFileName}
                    <p>{m.backup_apply_dialog_file()} <span class="font-mono text-xs">{applyFileName}</span></p>
                {/if}
                <label class="flex flex-col gap-1.5">
                    <span class="font-medium">{m.backup_confirm_label()}</span>
                    <input
                        type="text"
                        name="confirmation"
                        class="input w-full font-mono"
                        placeholder="APPLY BACKUP"
                        autocomplete="off"
                        bind:value={applyConfirmation}
                    />
                </label>
                {#snippet confirm()}
                    <button
                        type="submit"
                        class="btn btn-error"
                        disabled={applyConfirmation.trim() !== 'APPLY BACKUP' || pending !== null}
                    >
                        {@render buttonLabel(pending === 'apply', m.backup_stage_button(), m.backup_staging())}
                    </button>
                {/snippet}
            </ConfirmDialog>
        </form>
    </CollapsibleCard>
</div>
