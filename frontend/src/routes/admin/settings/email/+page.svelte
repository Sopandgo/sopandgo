<script lang="ts">
    import { enhance } from '$app/forms';
    import type { SubmitFunction } from '@sveltejs/kit';
    import Alert from '$lib/components/Alert.svelte';
    import Card from '$lib/components/Card.svelte';
    import CardPageHeading from '$lib/components/CardPageHeading.svelte';
    import CollapsibleCard from '$lib/components/CollapsibleCard.svelte';
    import { MailIcon } from 'lucide-svelte';
    import * as m from '$lib/paraglide/messages.js';

    type Transport = 'smtp' | 'resend';

    let { data, form }: { data: any; form: any } = $props();

    const smtp = $derived(data.smtp);
    const saveModeResult = $derived(form?.saveMode as { error?: string; ok?: boolean; mode?: 'smtp' | 'manual_links' } | undefined);
    const saveTransportResult = $derived(
        form?.saveTransport as { error?: string; ok?: boolean; transport?: Transport } | undefined
    );
    const saveResendResult = $derived(form?.saveResend as { error?: string; ok?: boolean } | undefined);
    const saveResult = $derived(form?.save as { error?: string; ok?: boolean } | undefined);
    const testResult = $derived(form?.test as { error?: string; ok?: boolean } | undefined);

    const manualLinks = $derived(smtp?.mail_mode === 'manual_links');
    const smtpMeta = $derived(
        smtp?.configured ? `${smtp.host}:${smtp.port} · ${smtp.from_address}` : undefined
    );
    const resendMeta = $derived(smtp?.resend_configured ? smtp.resend_from_address : undefined);

    function transportError(kind: Transport) {
        return saveTransportResult?.error && saveTransportResult.transport === kind
            ? saveTransportResult.error
            : undefined;
    }

    // Which form is in flight, for its button's spinner.
    let pending = $state<'mode' | 'transport' | 'smtp' | 'resend' | 'test' | null>(null);

    function submitting(which: NonNullable<typeof pending>, reset = true): SubmitFunction {
        return () => {
            pending = which;
            return async ({ update }) => {
                await update({ reset });
                pending = null;
            };
        };
    }
</script>

<svelte:head>
    <title>{m.settings_email()}</title>
</svelte:head>

{#snippet buttonLabel(busy: boolean, idle: string, busyLabel: string)}
    {#if busy}
        <span class="loading loading-spinner loading-xs"></span>
        {busyLabel}
    {:else}
        {idle}
    {/if}
{/snippet}

<!-- Header of each transport card: which one sends, or a button to switch to it. -->
{#snippet transportState(kind: Transport, name: string, configured: boolean)}
    {#if smtp?.mail_transport === kind}
        {#if manualLinks}
            <span class="badge badge-outline">{m.settings_transport_selected()}</span>
        {:else}
            <span class="badge badge-soft badge-success">{m.settings_transport_in_use()}</span>
        {/if}
    {:else if configured}
        <form method="POST" action="?/saveTransport" use:enhance={submitting('transport', false)}>
            <input type="hidden" name="mail_transport" value={kind} />
            <button type="submit" class="btn btn-sm" disabled={pending !== null}>
                {@render buttonLabel(pending === 'transport', m.settings_transport_use({ name }), m.common_saving())}
            </button>
        </form>
    {:else}
        <span class="badge badge-outline">{m.common_not_configured()}</span>
    {/if}
{/snippet}

{#snippet smtpTransportError()}
    <Alert type="error" message={transportError('smtp')} />
{/snippet}

{#snippet resendTransportError()}
    <Alert type="error" message={transportError('resend')} />
{/snippet}

<div class="flex flex-col gap-6">
    <Card>
        <div class="card-body space-y-4">
            <CardPageHeading>
                <MailIcon class="w-8 h-8" />
                {m.settings_email()}
            </CardPageHeading>

            <p class="text-sm text-base-content/70">
                {m.settings_email_lead()} <strong class="font-medium">Resend</strong> {m.settings_email_mid()} <strong class="font-medium">SMTP</strong>.
                {m.settings_email_secrets_before()}
                <code class="text-xs bg-base-200 px-1 rounded">SECRET_ENCRYPTION_KEY</code>
                {m.settings_email_secrets_after()}
            </p>

            {#if smtp && !smtp.encryption_key_set}
                <Alert type="error" message={m.settings_encryption_missing()} />
            {/if}
            {#if manualLinks}
                <Alert type="info" message={m.settings_manual_active()} />
            {/if}
            {#if saveModeResult?.error}
                <Alert type="error" message={saveModeResult.error} />
            {:else if saveModeResult?.ok}
                <Alert type="success" message={m.settings_mail_mode_saved()} />
            {/if}
            {#if saveTransportResult?.ok}
                <Alert type="success" message={m.settings_transport_saved()} />
            {/if}

            <form
                method="POST"
                action="?/saveMode"
                use:enhance={submitting('mode', false)}
                class="flex flex-col gap-1.5"
            >
                <label for="mail_mode" class="text-sm font-medium">{m.settings_mail_mode()}</label>
                <div class="flex flex-col gap-2 sm:flex-row">
                    <select
                        id="mail_mode"
                        name="mail_mode"
                        class="select w-full sm:flex-1"
                        value={saveModeResult?.mode ?? smtp?.mail_mode ?? 'smtp'}
                    >
                        <option value="smtp">{m.settings_mode_smtp()}</option>
                        <option value="manual_links">{m.settings_mode_manual()}</option>
                    </select>
                    <button type="submit" class="btn w-full sm:w-auto" disabled={pending !== null}>
                        {@render buttonLabel(pending === 'mode', m.settings_save_mode(), m.common_saving())}
                    </button>
                </div>
            </form>
        </div>
    </Card>

    <CollapsibleCard
        title={m.settings_smtp()}
        meta={smtpMeta}
        openWhen={Boolean(saveResult)}
        notice={transportError('smtp') ? smtpTransportError : undefined}
    >
        {#snippet trailing()}
            {@render transportState('smtp', 'SMTP', smtp?.configured ?? false)}
        {/snippet}

        {#if saveResult?.error}
            <Alert type="error" message={saveResult.error} />
        {:else if saveResult?.ok}
            <Alert type="success" message={m.settings_smtp_saved()} />
        {/if}

        <form
            method="POST"
            action="?/save"
            use:enhance={submitting('smtp')}
            class="grid grid-cols-1 gap-5 md:grid-cols-2"
        >
            <label class="flex flex-col gap-1.5 md:col-span-2">
                <span class="text-sm font-medium">{m.settings_smtp_host()}</span>
                <input
                    type="text"
                    name="host"
                    class="input w-full"
                    required
                    autocomplete="off"
                    value={form?.values?.host ?? smtp?.host ?? ''}
                />
            </label>

            <label class="flex flex-col gap-1.5">
                <span class="text-sm font-medium">{m.common_port()}</span>
                <input
                    type="text"
                    name="port"
                    class="input w-full"
                    required
                    inputmode="numeric"
                    placeholder="587"
                    value={form?.values?.port ?? smtp?.port ?? '587'}
                />
            </label>

            <label class="flex flex-col gap-1.5">
                <span class="text-sm font-medium">{m.common_from_address()}</span>
                <input
                    type="text"
                    name="from_address"
                    class="input w-full"
                    required
                    autocomplete="off"
                    placeholder="no-reply@yourcompany.com"
                    value={form?.values?.from_address ?? smtp?.from_address ?? ''}
                />
            </label>

            <label class="flex flex-col gap-1.5 md:col-span-2">
                <span class="text-sm font-medium">{m.common_username()}</span>
                <input
                    type="text"
                    name="username"
                    class="input w-full"
                    required
                    autocomplete="username"
                    value={form?.values?.username ?? smtp?.username ?? ''}
                />
            </label>

            <label class="flex flex-col gap-1.5 md:col-span-2">
                <span class="text-sm font-medium">{m.common_password()}</span>
                <input
                    type="password"
                    name="password"
                    class="input w-full"
                    autocomplete="new-password"
                    placeholder={smtp?.password_configured ? m.settings_keep_password() : m.common_required()}
                />
            </label>

            <div class="flex flex-col border-t border-base-300 pt-4 sm:flex-row sm:justify-end md:col-span-2">
                <button type="submit" class="btn w-full sm:w-auto" disabled={pending !== null}>
                    {@render buttonLabel(pending === 'smtp', m.common_save(), m.common_saving())}
                </button>
            </div>
        </form>
    </CollapsibleCard>

    <CollapsibleCard
        title={m.settings_resend()}
        meta={resendMeta}
        openWhen={Boolean(saveResendResult)}
        notice={transportError('resend') ? resendTransportError : undefined}
    >
        {#snippet trailing()}
            {@render transportState('resend', 'Resend', smtp?.resend_configured ?? false)}
        {/snippet}

        <p class="text-sm text-base-content/70">{m.settings_resend_help()}</p>

        {#if saveResendResult?.error}
            <Alert type="error" message={saveResendResult.error} />
        {:else if saveResendResult?.ok}
            <Alert type="success" message={m.settings_resend_saved()} />
        {/if}

        <form
            method="POST"
            action="?/saveResend"
            use:enhance={submitting('resend')}
            class="grid grid-cols-1 gap-5"
        >
            <label class="flex flex-col gap-1.5">
                <span class="text-sm font-medium">{m.common_from_address()}</span>
                <input
                    type="text"
                    name="resend_from_address"
                    class="input w-full"
                    required
                    autocomplete="off"
                    placeholder="onboarding@resend.dev"
                    value={form?.resendValues?.from_address ?? smtp?.resend_from_address ?? ''}
                />
            </label>

            <label class="flex flex-col gap-1.5">
                <span class="text-sm font-medium">{m.settings_api_key()}</span>
                <input
                    type="password"
                    name="resend_api_key"
                    class="input w-full"
                    autocomplete="new-password"
                    placeholder={smtp?.resend_api_key_configured ? m.settings_keep_key() : m.common_required()}
                />
            </label>

            <div class="flex flex-col border-t border-base-300 pt-4 sm:flex-row sm:justify-end">
                <button type="submit" class="btn w-full sm:w-auto" disabled={pending !== null}>
                    {@render buttonLabel(pending === 'resend', m.settings_save_resend(), m.common_saving())}
                </button>
            </div>
        </form>
    </CollapsibleCard>

    <Card title={m.settings_test_delivery()}>
        <div class="card-body space-y-4">
            {#if testResult?.error}
                <Alert type="error" message={testResult.error} />
            {:else if testResult?.ok}
                <Alert type="success" message={m.settings_test_sent()} />
            {/if}

            <form
                method="POST"
                action="?/test"
                use:enhance={submitting('test', false)}
                class="flex flex-col gap-1.5"
            >
                <label for="test_to" class="text-sm font-medium">{m.settings_test_to()}</label>
                <div class="flex flex-col gap-2 sm:flex-row">
                    <input
                        id="test_to"
                        type="email"
                        name="test_to"
                        class="input w-full sm:flex-1"
                        autocomplete="email"
                        placeholder="you@example.com"
                    />
                    <button type="submit" class="btn w-full sm:w-auto" disabled={pending !== null}>
                        {@render buttonLabel(pending === 'test', m.common_send_test(), m.common_sending())}
                    </button>
                </div>
            </form>
        </div>
    </Card>
</div>
