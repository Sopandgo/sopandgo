<script lang="ts">
    import { enhance, type SubmitFunction } from '$app/forms';
    import { resolve } from '$app/paths';
    import Alert from '#lib/components/Alert.svelte';
    import Card from '#lib/components/Card.svelte';
    import CardPageHeading from '#lib/components/CardPageHeading.svelte';
    import ConfirmDialog from '#lib/components/ConfirmDialog.svelte';
    import { AnchorIcon, CalendarClock, StampIcon } from '@lucide/svelte';
    import * as m from '#lib/paraglide/messages.js';
    import { getLocale } from '#lib/paraglide/runtime.js';
    import type { AuditAnchoringAuth, AuditAnchoringProbe, AuditAnchoringStatus } from '#lib/sdk/types.js';

    let { data, form }: { data: any; form: any } = $props();

    const status = $derived(data.status as AuditAnchoringStatus | null);
    const s3Configured = $derived(Boolean(data.s3Configured));
    const enabled = $derived(status?.enabled ?? false);

    type Result = { error?: string; ok?: boolean } | undefined;
    const saveResult = $derived(form?.save as (Result & { confirm?: AuditAnchoringProbe; wasEnabled?: boolean }) | undefined);
    const disableResult = $derived(form?.disable as Result);
    const runResult = $derived(form?.run as (Result & { skipped?: boolean; row?: number; warning?: string }) | undefined);
    const probe = $derived(saveResult?.confirm);

    // The auth fields follow the selected type before saving.
    const savedAuth = () => (status?.auth_type ?? 'none') as AuditAnchoringAuth;
    let auth = $state<AuditAnchoringAuth>(savedAuth());
    $effect(() => {
        auth = savedAuth();
    });

    // The admin confirms each already-broken row before switching on.
    let acceptBroken = $state(false);

    const intervals = $derived([
        { value: '15m0s', label: m.backup_every_minutes({ n: '15' }) },
        { value: '1h0m0s', label: m.backup_every_hour() },
        { value: '6h0m0s', label: m.backup_every_hours({ n: '6' }) },
        { value: '24h0m0s', label: m.backup_every_day() }
    ]);
    const intervalOptions = $derived(
        status?.interval && !intervals.some((i) => i.value === status.interval)
            ? [...intervals, { value: status.interval, label: status.interval }]
            : intervals
    );

    let pending = $state<'save' | 'confirm' | 'run' | 'disable' | null>(null);
    let disableOpen = $state(false);

    const saveSubmit: SubmitFunction = ({ submitter }) => {
        pending = submitter?.getAttribute('name') === 'confirm_fingerprint' ? 'confirm' : 'save';
        return async ({ update }) => {
            // Keep typed values (including a new secret) for the confirm step.
            await update({ reset: false });
            pending = null;
            acceptBroken = false;
        };
    };

    function submitting(which: 'run' | 'disable'): SubmitFunction {
        return () => {
            pending = which;
            return async ({ update }) => {
                await update();
                pending = null;
                disableOpen = false;
            };
        };
    }

    function formatUtc(iso?: string): string {
        if (!iso) return '—';
        const d = new Date(iso);
        if (Number.isNaN(d.getTime())) return iso;
        return `${d.toLocaleString(getLocale(), { dateStyle: 'medium', timeStyle: 'medium', timeZone: 'UTC' })} UTC`;
    }

    function shortHash(h?: string): string {
        return h ? h.slice(0, 8) : '';
    }
</script>

<svelte:head>
    <title>{m.settings_anchoring()}</title>
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
                <AnchorIcon class="w-8 h-8" />
                {m.settings_anchoring()}
            </CardPageHeading>

            <p class="text-sm text-base-content/70">{m.anchoring_intro()}</p>
            <p class="text-xs text-base-content/70">{m.anchoring_auditor_hint()}</p>

            {#if !status}
                <Alert type="error" message={m.error_anchoring_load()} />
            {:else if !s3Configured}
                <Alert type="warning" message={m.anchoring_needs_s3()} />
                <div class="flex justify-end">
                    <a href={resolve('admin/settings/backup')} class="btn">{m.anchoring_go_backup()}</a>
                </div>
            {/if}

            {#if enabled && status?.object_lock === false}
                <Alert type="warning" message={m.anchoring_object_lock_off()} />
            {/if}
            {#if disableResult?.error}
                <Alert type="error" message={disableResult.error} />
            {:else if disableResult?.ok}
                <Alert type="success" message={m.anchoring_disabled_ok()} />
            {/if}
        </div>
    </Card>

    {#if enabled && status}
        <Card title={m.anchoring_status_title()}>
            <div class="card-body space-y-4">
                {#if runResult?.error}
                    <Alert type="error" message={runResult.error} />
                {:else if runResult?.warning}
                    <Alert type="warning" message={runResult.warning} />
                {:else if runResult?.skipped}
                    <Alert type="info" message={m.anchoring_run_skipped()} />
                {:else if runResult?.ok}
                    <Alert type="success" message={m.anchoring_run_ok({ row: String(runResult.row ?? '') })} />
                {/if}
                {#if status.last_error && !runResult}
                    <Alert type="error" title={m.anchoring_last_error()} message={status.last_error} />
                {/if}

                <div class="grid gap-4 lg:grid-cols-2">
                    <div class="rounded-box border border-base-300 bg-base-100 p-4">
                        <div class="mb-2 flex items-center gap-2 text-sm font-medium text-base-content/70">
                            <StampIcon class="size-4" aria-hidden="true" />
                            {m.anchoring_last_anchor()}
                        </div>
                        {#if status.last_anchor}
                            <p class="text-lg font-medium leading-snug">{formatUtc(status.last_anchor.gen_time)}</p>
                            <p class="mt-2 text-sm">
                                {m.anchoring_last_anchor_row({ row: String(status.last_anchor.event_id) })}
                                <span class="font-mono text-xs" title={status.last_anchor.tip_hash}>{shortHash(status.last_anchor.tip_hash)}</span>
                            </p>
                            {#if status.pending_uploads > 0}
                                <p class="mt-2 text-xs text-warning">{m.anchoring_pending_uploads({ count: String(status.pending_uploads) })}</p>
                            {/if}
                        {:else}
                            <p class="text-sm text-base-content/70">{m.anchoring_none_yet()}</p>
                        {/if}
                    </div>

                    <div class="rounded-box border border-base-300 bg-base-100 p-4">
                        <div class="mb-2 flex items-center gap-2 text-sm font-medium text-base-content/70">
                            <CalendarClock class="size-4" aria-hidden="true" />
                            {m.anchoring_next()}
                        </div>
                        <p class="text-lg font-medium leading-snug">{formatUtc(status.next_run_utc)}</p>
                        <p class="mt-2 text-xs text-base-content/70">{m.anchoring_next_help()}</p>
                    </div>
                </div>

                <form method="POST" action="?/run" use:enhance={submitting('run')} class="flex justify-end">
                    <button type="submit" class="btn" disabled={pending !== null}>
                        {@render buttonLabel(pending === 'run', m.anchoring_run(), m.anchoring_running())}
                    </button>
                </form>
            </div>
        </Card>
    {/if}

    {#if status}
        <Card title={m.anchoring_tsa_title()}>
            <form method="POST" action="?/save" use:enhance={saveSubmit} class="card-body grid grid-cols-1 gap-5 md:grid-cols-2">
                {#if enabled}
                    <input type="hidden" name="was_enabled" value="on" />
                {/if}

                {#if saveResult?.error}
                    <div class="md:col-span-2"><Alert type="error" message={saveResult.error} /></div>
                {:else if saveResult?.ok}
                    <div class="md:col-span-2">
                        <Alert type="success" message={saveResult.wasEnabled ? m.anchoring_saved() : m.anchoring_enabled_ok()} />
                    </div>
                {/if}

                <label class="flex flex-col gap-1.5 md:col-span-2">
                    <span class="text-sm font-medium">{m.anchoring_tsa_url()}</span>
                    <input
                        type="url"
                        name="tsa_url"
                        class="input w-full"
                        autocomplete="off"
                        required
                        value={status.tsa_url}
                        aria-describedby="tsa-url-help"
                    />
                    <span id="tsa-url-help" class="text-xs text-base-content/70">{m.anchoring_tsa_url_help()}</span>
                </label>

                <label class="flex flex-col gap-1.5">
                    <span class="text-sm font-medium">{m.anchoring_auth()}</span>
                    <select name="auth_type" class="select w-full" bind:value={auth}>
                        <option value="none">{m.anchoring_auth_none()}</option>
                        <option value="basic">{m.anchoring_auth_basic()}</option>
                        <option value="bearer">{m.anchoring_auth_bearer()}</option>
                    </select>
                </label>

                <label class="flex flex-col gap-1.5">
                    <span class="text-sm font-medium">{m.anchoring_interval()}</span>
                    <select name="interval" class="select w-full" value={status.interval || '1h0m0s'}>
                        {#each intervalOptions as opt (opt.value)}
                            <option value={opt.value}>{opt.label}</option>
                        {/each}
                    </select>
                    <span class="text-xs text-base-content/70">{m.anchoring_interval_help()}</span>
                </label>

                {#if auth === 'basic'}
                    <label class="flex flex-col gap-1.5">
                        <span class="text-sm font-medium">{m.common_username()}</span>
                        <input type="text" name="username" class="input w-full" autocomplete="off" value={status.username} />
                    </label>
                {/if}
                {#if auth !== 'none'}
                    <label class="flex flex-col gap-1.5">
                        <span class="text-sm font-medium">{m.anchoring_secret()}</span>
                        <input type="password" name="secret" class="input w-full" autocomplete="new-password" />
                        {#if status.secret_set}
                            <span class="text-xs text-base-content/70">{m.anchoring_secret_keep()}</span>
                        {/if}
                    </label>
                {/if}

                <label class="flex flex-col gap-1.5 md:col-span-2">
                    <span class="text-sm font-medium">{m.anchoring_ca_pem()}</span>
                    <textarea name="pinned_ca_pem" class="textarea w-full font-mono text-xs" rows="3" spellcheck="false"></textarea>
                    <span class="text-xs text-base-content/70">{m.anchoring_ca_pem_help()}</span>
                </label>

                {#if enabled && status.fingerprint}
                    <div class="md:col-span-2 border-t border-base-300 pt-5">
                        <h3 class="text-sm font-medium text-base-content/70">{m.anchoring_pinned()}</h3>
                        <p class="mt-1 text-sm">{status.subject}</p>
                        <p class="mt-1 font-mono text-xs break-all">{status.fingerprint}</p>
                        <label class="mt-3 flex w-fit cursor-pointer items-center gap-3 text-sm">
                            <input type="checkbox" name="repin" class="checkbox checkbox-sm checkbox-primary" />
                            <span>{m.anchoring_repin()}</span>
                        </label>
                        <p class="mt-1 text-xs text-base-content/70">{m.anchoring_repin_help()}</p>
                    </div>
                {/if}

                {#if probe}
                    <div class="md:col-span-2 space-y-4 border-t border-base-300 pt-5" aria-live="polite">
                        <Alert type="info" title={m.anchoring_confirm_title()} message={m.anchoring_confirm_help()} />
                        <dl class="grid gap-2 text-sm">
                            <dt class="text-base-content/70">{m.anchoring_subject()}</dt>
                            <dd>{probe.subject}</dd>
                            <dt class="text-base-content/70">{m.anchoring_fingerprint()}</dt>
                            <dd class="font-mono text-xs break-all">{probe.fingerprint}</dd>
                        </dl>
                        {#if !probe.object_lock}
                            <Alert type="warning" message={m.anchoring_object_lock_off()} />
                        {/if}
                        {#if probe.broken_rows.length > 0}
                            <Alert type="warning" title={m.anchoring_broken_title()} message={m.anchoring_broken_help()} />
                            <ul class="list-disc pl-6 text-sm">
                                {#each probe.broken_rows as row (row.event_id)}
                                    <li>
                                        <a class="link font-mono text-xs" href={resolve('admin/audit-logs')}>{m.integrity_admin_audit_event({ id: String(row.event_id) })}</a>
                                        <span class="font-mono text-xs text-error">{row.error}</span>
                                        <input type="hidden" name="accept_broken" value={row.event_id} />
                                    </li>
                                {/each}
                            </ul>
                            <label class="flex w-fit cursor-pointer items-center gap-3 text-sm">
                                <input type="checkbox" class="checkbox checkbox-sm checkbox-primary" bind:checked={acceptBroken} />
                                <span>{m.anchoring_broken_accept({ count: String(probe.broken_rows.length) })}</span>
                            </label>
                        {/if}
                    </div>
                {/if}

                <div class="flex flex-col gap-2 border-t border-base-300 pt-5 md:col-span-2 sm:flex-row sm:justify-end">
                    {#if probe}
                        <button type="submit" class="btn" disabled={pending !== null}>
                            {@render buttonLabel(pending === 'save', m.anchoring_test_again(), m.anchoring_testing())}
                        </button>
                        <button
                            type="submit"
                            name="confirm_fingerprint"
                            value={probe.fingerprint}
                            class="btn btn-primary"
                            disabled={pending !== null || (probe.broken_rows.length > 0 && !acceptBroken)}
                        >
                            {@render buttonLabel(pending === 'confirm', enabled ? m.anchoring_confirm_change() : m.anchoring_confirm(), m.common_saving())}
                        </button>
                    {:else}
                        <button type="submit" class="btn btn-primary" disabled={pending !== null || !s3Configured}>
                            {@render buttonLabel(pending === 'save', enabled ? m.common_save() : m.anchoring_enable(), m.anchoring_testing())}
                        </button>
                    {/if}
                </div>
            </form>

            {#if enabled}
                <form method="POST" action="?/disable" use:enhance={submitting('disable')} class="card-body border-t border-base-300 pt-5">
                    <div class="flex flex-col gap-2 sm:flex-row sm:items-center sm:justify-between">
                        <p class="text-sm text-base-content/70">{m.anchoring_disable_help()}</p>
                        <button type="button" class="btn text-error" onclick={() => (disableOpen = true)} disabled={pending !== null}>
                            {m.anchoring_disable()}
                        </button>
                    </div>
                    <ConfirmDialog title={m.anchoring_disable_title()} bind:open={disableOpen} busy={pending === 'disable'}>
                        <Alert type="warning" message={m.anchoring_disable_warning()} />
                        {#snippet confirm()}
                            <button type="submit" class="btn btn-error" disabled={pending === 'disable'}>
                                {@render buttonLabel(pending === 'disable', m.anchoring_disable(), m.common_saving())}
                            </button>
                        {/snippet}
                    </ConfirmDialog>
                </form>
            {/if}
        </Card>
    {/if}
</div>
