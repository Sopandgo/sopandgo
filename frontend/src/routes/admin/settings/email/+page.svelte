<script lang="ts">
    import { enhance } from '$app/forms';
    import Alert from '$lib/components/Alert.svelte';
    import Card from '$lib/components/Card.svelte';
    import CardPageHeading from '$lib/components/CardPageHeading.svelte';
    import { MailIcon } from 'lucide-svelte';
    import * as m from '$lib/paraglide/messages.js';

    let { data, form }: { data: any; form: any } = $props();

    const smtp = $derived(data.smtp);
    const saveModeResult = $derived(form?.saveMode as { error?: string; ok?: boolean; mode?: 'smtp' | 'manual_links' } | undefined);
    const saveTransportResult = $derived(
        form?.saveTransport as { error?: string; ok?: boolean; transport?: 'smtp' | 'resend' } | undefined
    );
    const saveResendResult = $derived(form?.saveResend as { error?: string; ok?: boolean } | undefined);
    const saveResult = $derived(form?.save as { error?: string; ok?: boolean } | undefined);
    const testResult = $derived(form?.test as { error?: string; ok?: boolean } | undefined);
</script>

<svelte:head>
    <title>{m.settings_email()}</title>
</svelte:head>

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
                <code class="text-xs bg-base-200 px-1 rounded">SMTP_SECRET_ENCRYPTION_KEY</code>
                {m.settings_email_secrets_after()}
            </p>

            {#if saveModeResult?.error}
                <Alert type='error' message={saveModeResult.error}/>
            {/if}
            {#if saveModeResult?.ok}
                <Alert type='success' message={m.settings_mail_mode_saved()}/>
            {/if}

            {#if saveTransportResult?.error}
                <Alert type='error' message={saveTransportResult.error}/>
            {/if}
            {#if saveTransportResult?.ok}
                <Alert type='success' message={m.settings_transport_saved()}/>
            {/if}

            <form method="POST" action="?/saveTransport" use:enhance class="">
                <label class="form-control w-full mb-4">
                    <span class="label-text">{m.settings_transport()}</span>
                    <select
                        name="mail_transport"
                        class="select w-full"
                        value={saveTransportResult?.transport ?? smtp?.mail_transport ?? 'smtp'}
                    >
                        <option value="smtp">{m.settings_transport_smtp()}</option>
                        <option value="resend">{m.settings_transport_resend()}</option>
                    </select>
                </label>
                <div class="card-actions mt-4 w-full flex-col sm:flex-row sm:justify-end">
                    <button type="submit" class="btn w-full sm:w-auto">{m.settings_save_transport()}</button>
                </div>
            </form>

            <form method="POST" action="?/saveMode" use:enhance class="">
                <label class="form-control w-full mb-4">
                    <span class="label-text">{m.settings_mail_mode()}</span>
                    <select
                        name="mail_mode"
                        class="select w-full"
                        value={saveModeResult?.mode ?? smtp?.mail_mode ?? 'smtp'}
                    >
                        <option value="smtp">{m.settings_mode_smtp()}</option>
                        <option value="manual_links">{m.settings_mode_manual()}</option>
                    </select>
                </label>
                <div class="card-actions mt-4 w-full flex-col sm:flex-row sm:justify-end">
                    <button type="submit" class="btn w-full sm:w-auto">{m.settings_save_mode()}</button>
                </div>
            </form>
        </div>
    </Card>

    <Card>
        <div class="card-body space-y-4">
            <h2 class="card-title">{m.settings_resend()}</h2>
            <p class="text-sm text-base-content/70">
                {m.settings_resend_help()}
            </p>

            {#if saveResendResult?.error}
                <Alert type='error' message={saveResendResult.error}/>
            {/if}
            {#if saveResendResult?.ok}
                <Alert type='success' message={m.settings_resend_saved()}/>
            {/if}

            <form method="POST" action="?/saveResend" use:enhance class="grid grid-cols-1 gap-4">
                <label class="form-control w-full">
                    <span class="label-text">{m.common_from_address()}</span>
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

                <label class="form-control w-full">
                    <span class="label-text">{m.settings_api_key()}</span>
                    <input
                        type="password"
                        name="resend_api_key"
                        class="input w-full"
                        autocomplete="new-password"
                        placeholder={smtp?.resend_api_key_configured ? m.settings_keep_key() : m.common_required()}
                    />
                </label>

                <div class="card-actions w-full flex-col sm:flex-row sm:justify-end">
                    <button type="submit" class="btn w-full sm:w-auto">{m.settings_save_resend()}</button>
                </div>
            </form>
        </div>
    </Card>

    <Card>
        <div class="card-body space-y-4">
            <h2 class="card-title">{m.settings_smtp()}</h2>

            {#if smtp && !smtp.encryption_key_set}
                <Alert type='error' message={m.settings_encryption_missing()}/>
            {/if}

            {#if smtp?.mail_mode === 'manual_links'}
                <Alert type='info' message={m.settings_manual_active()}/>
            {/if}

            {#if saveResult?.error}
                <Alert type='error' message={saveResult.error}/>
            {/if}
            {#if saveResult?.ok}
                <Alert type='success' message={m.settings_smtp_saved()}/>
            {/if}

            <form
                method="POST"
                action="?/save"
                use:enhance
                class="grid grid-cols-1 md:grid-cols-2 gap-4"
            >
                <label class="form-control w-full md:col-span-2">
                    <span class="label-text">{m.settings_smtp_host()}</span>
                    <input
                        type="text"
                        name="host"
                        class="input w-full"
                        required
                        autocomplete="off"
                        value={form?.values?.host ?? smtp?.host ?? ''}
                    />
                </label>

                <label class="form-control w-full">
                    <span class="label-text">{m.common_port()}</span>
                    <input
                        type="text"
                        name="port"
                        class="input w-full"
                        required
                        placeholder="587"
                        value={form?.values?.port ?? smtp?.port ?? '587'}
                    />
                </label>

                <label class="form-control w-full">
                    <span class="label-text">{m.common_from_address()}</span>
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

                <label class="form-control w-full md:col-span-2">
                    <span class="label-text">{m.common_username()}</span>
                    <input
                        type="text"
                        name="username"
                        class="input w-full"
                        required
                        autocomplete="username"
                        value={form?.values?.username ?? smtp?.username ?? ''}
                    />
                </label>

                <label class="form-control w-full md:col-span-2">
                    <span class="label-text">{m.common_password()}</span>
                    <input
                        type="password"
                        name="password"
                        class="input w-full"
                        autocomplete="new-password"
                        placeholder={smtp?.password_configured ? m.settings_keep_password() : m.common_required()}
                    />
                </label>

                <div class="card-actions col-span-2 w-full flex-col sm:flex-row sm:justify-end">
                    <button type="submit" class="btn w-full sm:w-auto">{m.common_save()}</button>
                </div>
            </form>

        </div>
    </Card>

    <Card>
        <div class="card-body space-y-4">
            <h2 class="card-title">{m.settings_test_delivery()}</h2>

            {#if testResult?.error}
                <Alert type='error' message={testResult.error}/>
            {/if}
            {#if testResult?.ok}
                <Alert type='success' message={m.settings_test_sent()}/>
            {/if}

            <form method="POST" action="?/test" use:enhance class="flex flex-col sm:flex-row gap-4 sm:items-end">
                <label class="form-control flex-1 w-full">
                    <span class="label-text">{m.settings_test_to()}</span>
                    <input
                        type="email"
                        name="test_to"
                        class="input w-full"
                        placeholder="you@example.com"
                    />
                </label>
                <button type="submit" class="btn w-full shrink-0 sm:w-auto">{m.common_send_test()}</button>
            </form>
        </div>
    </Card>
</div>
