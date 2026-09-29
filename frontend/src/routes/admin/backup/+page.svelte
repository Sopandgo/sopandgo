<script lang="ts">
    import { enhance } from '$app/forms';
    import Alert from '$lib/components/Alert.svelte';
    import Card from '$lib/components/Card.svelte';
    import CardPageHeading from '$lib/components/CardPageHeading.svelte';
    import { ArchiveIcon, CalendarClock, History } from 'lucide-svelte';

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

    function formatUtc(iso?: string): string {
        if (!iso) return '—';
        const d = new Date(iso);
        if (Number.isNaN(d.getTime())) return iso;
        return `${d.toLocaleString(undefined, {
            dateStyle: 'medium',
            timeStyle: 'medium',
            timeZone: 'UTC'
        })} UTC`;
    }

    /** Human label for Go duration strings from the API (e.g. `2m0s`, `24h0m0s`). */
    function intervalSummary(interval?: string): string {
        if (!interval) return '—';
        const h = interval.match(/^(\d+)h/);
        if (h) {
            const n = parseInt(h[1], 10);
            return `Every ${n} hour${n === 1 ? '' : 's'}`;
        }
        const m = interval.match(/^(\d+)m/);
        if (m) {
            const n = parseInt(m[1], 10);
            return `Every ${n} minute${n === 1 ? '' : 's'}`;
        }
        const s = interval.match(/^(\d+)s/);
        if (s) {
            const n = parseInt(s[1], 10);
            return `Every ${n} second${n === 1 ? '' : 's'}`;
        }
        return interval;
    }

</script>

<div class="flex flex-col gap-6">
    <Card>
        <div class="card-body space-y-4">
            <CardPageHeading>
                <ArchiveIcon class="w-8 h-8" />
                Backup
            </CardPageHeading>

            <div class="flex items-center gap-2 text-base-content/70">
                Export creates a downloadable .zip containing app.db,
                sops/, and manifest.json.
            </div>

            {#if backupStatus?.locked}
                <Alert variant='warning' message='Maintenance lock active: {backupStatus.message || 'Backup operation in progress'}'/>
            {/if}

            <div class="card-actions pt-4 border-t border-base-200 mt-2 w-full flex-col sm:flex-row sm:justify-end">
                <a href="/admin/backup/export" class="btn btn-primary w-full sm:w-auto">Export backup (.zip)</a>
            </div>
        </div>
    </Card>

    {#if backupStatus?.s3_scheduled}
        {@const s3 = backupStatus.s3_scheduled}
        {@const intervalLabel = intervalSummary(s3.interval)}
        <Card>
            <div class="card-body space-y-4">
                <div class="flex flex-col gap-3 sm:flex-row sm:items-start sm:justify-between">
                    <div class="flex items-start gap-3">
                        <div>
                            <h2 class="card-title">Automatic S3 backups</h2>
                            <p class="text-base-content/60 mt-1 max-w-prose text-sm leading-relaxed">
                                The server uploads the same full archive as manual export on a timer. Configure with
                                <code class="rounded bg-base-200 px-1 py-0.5 text-xs">BACKUP_S3_*</code> environment variables
                                (see operator backup documentation).
                            </p>
                        </div>
                    </div>
                    {#if s3.enabled}
                        <span class="badge badge-success badge-lg self-start">Enabled</span>
                    {:else}
                        <span class="badge badge-warning badge-lg self-start">Disabled</span>
                    {/if}
                </div>

                {#if s3.enabled}
                    <div class="stats stats-vertical shadow-sm border border-base-300 bg-base-100 w-full sm:stats-horizontal">
                        <div class="stat place-items-start border-base-300 py-4 sm:border-e">
                            <div class="stat-title text-xs font-semibold uppercase tracking-wide text-base-content/50">
                                Bucket
                            </div>
                            <div class="stat-value font-mono text-lg font-normal break-all text-start leading-snug">
                                {s3.bucket ?? '—'}
                            </div>
                        </div>
                        <div class="stat place-items-start border-base-300 py-4 sm:border-e">
                            <div class="stat-title text-xs font-semibold uppercase tracking-wide text-base-content/50">
                                Key prefix
                            </div>
                            <div class="stat-value font-mono text-base font-normal break-all text-start leading-snug">
                                {#if s3.key_prefix?.trim()}
                                    {s3.key_prefix}
                                {:else}
                                    <span class="text-base-content/40">(bucket root)</span>
                                {/if}
                            </div>
                        </div>
                        <div class="stat place-items-start py-4">
                            <div class="stat-title text-xs font-semibold uppercase tracking-wide text-base-content/50">
                                Schedule
                            </div>
                            <div class="stat-value text-lg font-normal text-start leading-snug">{intervalLabel}</div>
                            {#if s3.interval && s3.interval !== intervalLabel}
                                <div class="stat-desc font-mono text-xs opacity-70">{s3.interval}</div>
                            {/if}
                        </div>
                    </div>

                    {#if s3.retention_max_objects || s3.retention_days}
                        <div class="rounded-box border border-base-300 bg-base-200/40 p-4">
                            <div class="mb-3 text-xs font-semibold uppercase tracking-wide text-base-content/50">
                                Retention after each upload
                            </div>
                            <ul class="flex flex-col gap-3 sm:flex-row sm:flex-wrap sm:gap-4">
                                {#if s3.retention_max_objects}
                                    <li class="flex items-start gap-2 text-sm">
                                        <span class="badge badge-primary badge-outline shrink-0">Count</span>
                                        <span>
                                            Keep at most <strong>{s3.retention_max_objects}</strong> recent automated
                                            archive{Number(s3.retention_max_objects) === 1 ? '' : 's'} (oldest eligible objects
                                            are removed first).
                                        </span>
                                    </li>
                                {/if}
                                {#if s3.retention_days}
                                    <li class="flex items-start gap-2 text-sm">
                                        <span class="badge badge-secondary badge-outline shrink-0">Age</span>
                                        <span>
                                            Remove automated archives older than <strong>{s3.retention_days}</strong> day{Number(s3.retention_days) === 1 ? '' : 's'}.
                                        </span>
                                    </li>
                                {/if}
                            </ul>
                        </div>
                    {/if}

                    <div class="divider my-0">Recent activity</div>

                    <div class="grid gap-4 lg:grid-cols-2">
                        <div class="rounded-box border border-base-300 bg-base-100 p-4 shadow-sm">
                            <div class="text-base-content/50 mb-2 flex items-center gap-2 text-xs font-semibold uppercase tracking-wide">
                                <History class="size-4" aria-hidden="true" />
                                Last successful upload
                            </div>
                            {#if s3.last_success_utc}
                                <p class="text-lg font-medium leading-snug">{formatUtc(s3.last_success_utc)}</p>
                                {#if s3.last_object_key}
                                    <div class="mt-3">
                                        <div class="text-base-content/50 mb-1 text-xs">Object key</div>
                                        <code class="bg-base-200 border-base-300 block w-full overflow-x-auto rounded-lg border px-3 py-2 font-mono text-xs leading-relaxed">
                                            {s3.last_object_key}
                                        </code>
                                    </div>
                                {/if}
                            {:else if s3.last_run_utc}
                                <p class="text-base-content/70 text-sm">
                                    Last run (no success recorded yet): {formatUtc(s3.last_run_utc)}
                                </p>
                            {:else}
                                <p class="text-base-content/60 text-sm">No upload has completed yet. Wait for the first scheduled run.</p>
                            {/if}
                        </div>

                        <div class="rounded-box border border-base-300 bg-base-100 p-4 shadow-sm">
                            <div class="text-base-content/50 mb-2 flex items-center gap-2 text-xs font-semibold uppercase tracking-wide">
                                <CalendarClock class="size-4" aria-hidden="true" />
                                Next run (approx.)
                            </div>
                            {#if s3.next_run_utc}
                                <p class="text-lg font-medium leading-snug">{formatUtc(s3.next_run_utc)}</p>
                                <p class="text-base-content/60 mt-2 text-xs">
                                    Timing depends on server load; use logs or refresh this page to confirm completion.
                                </p>
                            {:else}
                                <p class="text-base-content/60 text-sm">—</p>
                            {/if}
                        </div>
                    </div>

                    {#if s3.last_error}
                        <div role="alert" class="alert alert-error alert-soft flex-col items-start gap-1 text-start sm:flex-row sm:items-center">
                            <span class="font-semibold">Last error</span>
                            <span class="text-sm">{s3.last_error}</span>
                        </div>
                    {/if}
                {:else}
                    <div role="alert" class="alert alert-info alert-soft">
                        <span>
                            Automatic uploads are off for this deployment. Enable
                            <code class="mx-0.5 rounded bg-base-200 px-1 py-0.5 text-xs">BACKUP_S3_ENABLED</code>
                            and set bucket, region, and credentials as described in the operator backup guide.
                        </span>
                    </div>
                {/if}
            </div>
        </Card>
    {/if}

    <Card>
        <div class="card-body space-y-4">
            <h2 class="card-title">Validate backup</h2>
            {#if importBackupResult?.error}
                <Alert variant='error' message={importBackupResult.error}/>
            {/if}
            {#if importBackupResult?.ok}
                <Alert variant='success' message='Backup validated. Source app: {importBackupResult.manifest?.app_version}, schema: {importBackupResult.manifest?.db_schema_version}'/>
                {#if importBackupResult.note}
                    <Alert variant='info' message={importBackupResult.note}/>
                {/if}
            {/if}

            <form method="POST" action="?/importBackup" use:enhance enctype="multipart/form-data" class="grid grid-cols-1 gap-4">
                <span class="label-text-alt text-base-content/70">
                    This checks backup format and schema compatibility. It does not overwrite live data yet.
                </span>
                <label class="form-control w-full">
                    <!-- <span class="label-text">Validate backup import compatibility</span> -->
                    <input
                        type="file"
                        name="backup_file"
                        class="file-input file-input-bordered w-full"
                        accept=".zip,application/zip"
                        disabled={uploadsLocked}
                    />
                </label>
                <div class="card-actions w-full flex-col sm:flex-row sm:justify-end">
                    <button type="submit" class="btn btn-secondary w-full sm:w-auto" disabled={uploadsLocked}>
                        Validate backup
                    </button>
                </div>
            </form>
        </div>
    </Card>

    {#if uploadsLocked}
        <Alert variant='warning' message='
            A backup restore is staged and waiting for restart.
            Restart the backend/container to apply it.
            Validate/apply uploads stay disabled until restart.'
        />
    
    {:else}

        <Card>
            <div class="card-body space-y-4">
                <h2 class="card-title">Apply backup</h2>
                    <Alert variant='warning' message='
                        Destructive action: this stages replacement of live data. The restore is applied on next backend/container restart.
                    '/>
                {#if applyBackupResult?.error}
                    <Alert variant='error' message={applyBackupResult.error}/>
                {/if}
                {#if applyBackupResult?.ok}
                    <Alert variant='success' message='Backup apply staged. Restart required: {applyBackupResult.requires_restart ? 'yes' : 'no'}. Pre-apply snapshot: {applyBackupResult.pre_apply_backup}'/>
                    {#if applyBackupResult.note}
                        <Alert variant='info' message={applyBackupResult.note}/>
                    {/if}
                {/if}
                <form method="POST" action="?/applyBackup" use:enhance enctype="multipart/form-data" class="grid grid-cols-1 gap-4">
                    <label class="form-control w-full">
                        <!-- <span class="label-text">Backup archive (.zip)</span> -->
                        <input
                            type="file"
                            name="backup_file"
                            class="file-input file-input-bordered w-full"
                            accept=".zip,application/zip"
                            disabled={uploadsLocked}
                        />
                    </label>
                    <label class="form-control w-full">
                        <span class="label-text">Type to confirm destructive restore</span>
                        <input
                            type="text"
                            name="confirmation"
                            class="input input-bordered w-full"
                            placeholder="APPLY BACKUP"
                            autocomplete="off"
                            disabled={uploadsLocked}
                        />
                    </label>
                    <div class="card-actions w-full flex-col sm:flex-row sm:justify-end">
                        <button type="submit" class="btn btn-error w-full sm:w-auto" disabled={uploadsLocked}>
                            Stage apply backup (requires restart)
                        </button>
                    </div>
                </form>
            </div>
        </Card>
    {/if}
</div>