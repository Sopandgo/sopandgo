<script lang="ts">
    import { enhance } from '$app/forms';
    import Alert from '$lib/components/Alert.svelte';
    import Card from '$lib/components/Card.svelte';
    import CardPageHeading from '$lib/components/CardPageHeading.svelte';
    import { LanguagesIcon, MailIcon, WebhookIcon } from 'lucide-svelte';
    import LocaleSelect from '$lib/components/LocaleSelect.svelte';
    import * as m from '$lib/paraglide/messages.js';
    import type { IntegrationEvent } from '$lib/sdk/types';

    let { data, form }: { data: any; form: any } = $props();

    const smtp = $derived(data.smtp);
    const integrations = $derived(data.integrations);
    const saveModeResult = $derived(form?.saveMode as { error?: string; ok?: boolean; mode?: 'smtp' | 'manual_links' } | undefined);
    const saveTransportResult = $derived(
        form?.saveTransport as { error?: string; ok?: boolean; transport?: 'smtp' | 'resend' } | undefined
    );
    const saveResendResult = $derived(form?.saveResend as { error?: string; ok?: boolean } | undefined);
    const saveResult = $derived(form?.save as { error?: string; ok?: boolean } | undefined);
    const testResult = $derived(form?.test as { error?: string; ok?: boolean } | undefined);
    const saveSlackResult = $derived(form?.saveSlack as { error?: string; ok?: boolean } | undefined);
    const saveGotifyResult = $derived(form?.saveGotify as { error?: string; ok?: boolean } | undefined);
    const saveWebhookResult = $derived(form?.saveWebhook as { error?: string; ok?: boolean } | undefined);
    const testSlackResult = $derived(form?.testSlack as { error?: string; ok?: boolean } | undefined);
    const testGotifyResult = $derived(form?.testGotify as { error?: string; ok?: boolean } | undefined);
    const testWebhookResult = $derived(form?.testWebhook as { error?: string; ok?: boolean } | undefined);

    const eventLabels: Record<IntegrationEvent, string> = {
        sop_published: m.notify_event_published(),
        sop_rc: m.notify_event_rc(),
        sop_rejected: m.notify_event_rejected(),
        backup_s3_failed: m.notify_event_backup(),
        integrity_check_failed: m.notify_event_integrity()
    };

    const knownEvents = $derived(
        (integrations?.known_events as IntegrationEvent[] | undefined) ??
            (Object.keys(eventLabels) as IntegrationEvent[])
    );

    function eventChecked(channelEvents: string[] | undefined, event: IntegrationEvent) {
        return (channelEvents ?? []).includes(event);
    }
</script>

<svelte:head>
    <title>{m.page_settings()}</title>
</svelte:head>


<div class="flex flex-col gap-6">
    <Card>
        <div class="card-body space-y-4">
            <CardPageHeading>
                <LanguagesIcon class="w-8 h-8" />
                {m.locale_organization()}
            </CardPageHeading>
            <p class="text-sm text-base-content/70">{m.locale_organization_help()}</p>
            {#if form?.saveDefaultLocale?.error}
                <Alert variant="error" message={form.saveDefaultLocale.error} />
            {/if}
            {#if form?.saveDefaultLocale?.ok}
                <Alert variant="success" message={m.locale_saved()} />
            {/if}
            <form method="POST" action="?/saveDefaultLocale" use:enhance>
                <label class="form-control w-full mb-4" for="org-locale">
                    <span class="label-text">{m.locale_label()}</span>
                </label>
                <LocaleSelect id="org-locale" value={form?.saveDefaultLocale?.locale ?? smtp?.default_locale ?? 'en'} />
                <div class="card-actions mt-4 w-full flex-col sm:flex-row sm:justify-end">
                    <button type="submit" class="btn btn-primary w-full sm:w-auto">{m.locale_save()}</button>
                </div>
            </form>
        </div>
    </Card>
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
                <Alert variant='error' message={saveModeResult.error}/>
            {/if}
            {#if saveModeResult?.ok}
                <Alert variant='success' message={m.settings_mail_mode_saved()}/>
            {/if}

            {#if saveTransportResult?.error}
                <Alert variant='error' message={saveTransportResult.error}/>
            {/if}
            {#if saveTransportResult?.ok}
                <Alert variant='success' message={m.settings_transport_saved()}/>
            {/if}

            <form method="POST" action="?/saveTransport" use:enhance class="">
                <label class="form-control w-full mb-4">
                    <span class="label-text">{m.settings_transport()}</span>
                    <select
                        name="mail_transport"
                        class="select select-bordered w-full"
                        value={saveTransportResult?.transport ?? smtp?.mail_transport ?? 'smtp'}
                    >
                        <option value="smtp">{m.settings_transport_smtp()}</option>
                        <option value="resend">{m.settings_transport_resend()}</option>
                    </select>
                </label>
                <div class="card-actions mt-4 w-full flex-col sm:flex-row sm:justify-end">
                    <button type="submit" class="btn btn-primary w-full sm:w-auto">{m.settings_save_transport()}</button>
                </div>
            </form>

            <form method="POST" action="?/saveMode" use:enhance class="">
                <label class="form-control w-full mb-4">
                    <span class="label-text">{m.settings_mail_mode()}</span>
                    <select
                        name="mail_mode"
                        class="select select-bordered w-full"
                        value={saveModeResult?.mode ?? smtp?.mail_mode ?? 'smtp'}
                    >
                        <option value="smtp">{m.settings_mode_smtp()}</option>
                        <option value="manual_links">{m.settings_mode_manual()}</option>
                    </select>
                    <!-- <span class="label-text-alt text-base-content/70 mt-1">
                        Manual links mode skips SMTP delivery and returns one-time links to admins after user create/reset actions.
                    </span> -->
                </label>
                <div class="card-actions mt-4 w-full flex-col sm:flex-row sm:justify-end">
                    <button type="submit" class="btn btn-primary w-full sm:w-auto">{m.settings_save_mode()}</button>
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
                <Alert variant='error' message={saveResendResult.error}/>
            {/if}
            {#if saveResendResult?.ok}
                <Alert variant='success' message={m.settings_resend_saved()}/>
            {/if}

            <form method="POST" action="?/saveResend" use:enhance class="grid grid-cols-1 gap-4">
                <label class="form-control w-full">
                    <span class="label-text">{m.common_from_address()}</span>
                    <input
                        type="text"
                        name="resend_from_address"
                        class="input input-bordered w-full"
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
                        class="input input-bordered w-full"
                        autocomplete="new-password"
                        placeholder={smtp?.resend_api_key_configured ? m.settings_keep_key() : m.common_required()}
                    />
                </label>

                <div class="card-actions w-full flex-col sm:flex-row sm:justify-end">
                    <button type="submit" class="btn btn-primary w-full sm:w-auto">{m.settings_save_resend()}</button>
                </div>
            </form>
        </div>
    </Card>

    <Card>
        <div class="card-body space-y-4">
            <h2 class="card-title">{m.settings_smtp()}</h2>

            {#if smtp && !smtp.encryption_key_set}
                <Alert variant='error' message={m.settings_encryption_missing()}/>
            {/if}

            {#if smtp?.mail_mode === 'manual_links'}
                <Alert variant='info' message={m.settings_manual_active()}/>
            {/if}

            {#if saveResult?.error}
                <Alert variant='error' message={saveResult.error}/>
            {/if}
            {#if saveResult?.ok}
                <Alert variant='success' message={m.settings_smtp_saved()}/>
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
                        class="input input-bordered w-full"
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
                        class="input input-bordered w-full"
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
                        class="input input-bordered w-full"
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
                        class="input input-bordered w-full"
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
                        class="input input-bordered w-full"
                        autocomplete="new-password"
                        placeholder={smtp?.password_configured ? m.settings_keep_password() : m.common_required()}
                    />
                </label>

                <div class="card-actions col-span-2 w-full flex-col sm:flex-row sm:justify-end">
                    <button type="submit" class="btn btn-primary w-full sm:w-auto">{m.common_save()}</button>
                </div>
            </form>

        </div>
    </Card>

    <Card>
        <div class="card-body space-y-4">
            <h2 class="card-title">{m.settings_test_delivery()}</h2>

            {#if testResult?.error}
                <Alert variant='error' message={testResult.error}/>
            {/if}
            {#if testResult?.ok}
                <Alert variant='success' message={m.settings_test_sent()}/>
            {/if}

            <form method="POST" action="?/test" use:enhance class="flex flex-col sm:flex-row gap-4 sm:items-end">
                <label class="form-control flex-1 w-full">
                    <span class="label-text">{m.settings_test_to()}</span>
                    <input
                        type="email"
                        name="test_to"
                        class="input input-bordered w-full"
                        placeholder="you@example.com"
                    />
                </label>
                <button type="submit" class="btn btn-secondary w-full shrink-0 sm:w-auto">{m.common_send_test()}</button>
            </form>
        </div>
    </Card>

    <Card>
        <div class="card-body space-y-4">
            <CardPageHeading>
                <WebhookIcon class="w-8 h-8" />
                {m.settings_integrations()}
            </CardPageHeading>
            <p class="text-sm text-base-content/70">
                {m.settings_integrations_before()}
                <code class="text-xs bg-base-200 px-1 rounded">SMTP_SECRET_ENCRYPTION_KEY</code>
                {m.settings_integrations_after()}
            </p>
            {#if integrations && !integrations.encryption_key_set}
                <Alert
                    variant="error"
                    message={m.settings_integrations_key_missing()}
                />
            {/if}
        </div>
    </Card>

    <Card>
        <div class="card-body space-y-4">
            <h2 class="card-title">{m.settings_slack()}</h2>
            {#if saveSlackResult?.error}
                <Alert variant="error" message={saveSlackResult.error} />
            {/if}
            {#if saveSlackResult?.ok}
                <Alert variant="success" message={m.settings_slack_saved()} />
            {/if}
            {#if testSlackResult?.error}
                <Alert variant="error" message={testSlackResult.error} />
            {/if}
            {#if testSlackResult?.ok}
                <Alert variant="success" message={m.settings_slack_test_sent()} />
            {/if}

            <form method="POST" action="?/saveSlack" use:enhance class="grid grid-cols-1 gap-4">
                <label class="label cursor-pointer justify-start gap-3">
                    <input
                        type="checkbox"
                        name="enabled"
                        class="toggle toggle-primary"
                        checked={integrations?.slack?.enabled ?? false}
                    />
                    <span class="label-text">{m.settings_slack_enable()}</span>
                </label>

                <label class="form-control w-full">
                    <span class="label-text">{m.settings_webhook_url()}</span>
                    <input
                        type="password"
                        name="webhook_url"
                        class="input input-bordered w-full"
                        autocomplete="off"
                        placeholder={integrations?.slack?.secret_configured
                            ? m.settings_keep_webhook()
                            : 'https://hooks.slack.com/services/...'}
                    />
                </label>

                <fieldset class="space-y-2">
                    <legend class="text-sm font-medium">{m.common_events()}</legend>
                    {#each knownEvents as event (event)}
                        <label class="label cursor-pointer justify-start gap-3 py-1">
                            <input
                                type="checkbox"
                                name="events"
                                value={event}
                                class="checkbox checkbox-sm"
                                checked={eventChecked(integrations?.slack?.events, event)}
                            />
                            <span class="label-text">{eventLabels[event] ?? event}</span>
                        </label>
                    {/each}
                </fieldset>

                <div class="card-actions w-full flex-col sm:flex-row sm:justify-end gap-2">
                    <button type="submit" class="btn btn-primary w-full sm:w-auto">{m.settings_save_slack()}</button>
                </div>
            </form>
            <form method="POST" action="?/testSlack" use:enhance>
                <button type="submit" class="btn btn-secondary btn-sm">{m.common_send_test()}</button>
            </form>
        </div>
    </Card>

    <Card>
        <div class="card-body space-y-4">
            <h2 class="card-title">{m.settings_gotify()}</h2>
            {#if saveGotifyResult?.error}
                <Alert variant="error" message={saveGotifyResult.error} />
            {/if}
            {#if saveGotifyResult?.ok}
                <Alert variant="success" message={m.settings_gotify_saved()} />
            {/if}
            {#if testGotifyResult?.error}
                <Alert variant="error" message={testGotifyResult.error} />
            {/if}
            {#if testGotifyResult?.ok}
                <Alert variant="success" message={m.settings_gotify_test_sent()} />
            {/if}

            <form method="POST" action="?/saveGotify" use:enhance class="grid grid-cols-1 gap-4">
                <label class="label cursor-pointer justify-start gap-3">
                    <input
                        type="checkbox"
                        name="enabled"
                        class="toggle toggle-primary"
                        checked={integrations?.gotify?.enabled ?? false}
                    />
                    <span class="label-text">{m.settings_gotify_enable()}</span>
                </label>

                <label class="form-control w-full">
                    <span class="label-text">{m.settings_gotify_url()}</span>
                    <input
                        type="url"
                        name="url"
                        class="input input-bordered w-full"
                        autocomplete="off"
                        placeholder="https://gotify.example.com"
                        value={integrations?.gotify?.url ?? ''}
                    />
                </label>

                <label class="form-control w-full">
                    <span class="label-text">{m.settings_gotify_token()}</span>
                    <input
                        type="password"
                        name="token"
                        class="input input-bordered w-full"
                        autocomplete="off"
                        placeholder={integrations?.gotify?.secret_configured
                            ? m.settings_keep_token()
                            : m.common_required()}
                    />
                </label>

                <fieldset class="space-y-2">
                    <legend class="text-sm font-medium">{m.common_events()}</legend>
                    {#each knownEvents as event (event)}
                        <label class="label cursor-pointer justify-start gap-3 py-1">
                            <input
                                type="checkbox"
                                name="events"
                                value={event}
                                class="checkbox checkbox-sm"
                                checked={eventChecked(integrations?.gotify?.events, event)}
                            />
                            <span class="label-text">{eventLabels[event] ?? event}</span>
                        </label>
                    {/each}
                </fieldset>

                <div class="card-actions w-full flex-col sm:flex-row sm:justify-end">
                    <button type="submit" class="btn btn-primary w-full sm:w-auto">{m.settings_save_gotify()}</button>
                </div>
            </form>
            <form method="POST" action="?/testGotify" use:enhance>
                <button type="submit" class="btn btn-secondary btn-sm">{m.common_send_test()}</button>
            </form>
        </div>
    </Card>

    <Card>
        <div class="card-body space-y-4">
            <h2 class="card-title">{m.settings_webhook()}</h2>
            <p class="text-sm text-base-content/70">
                {m.settings_webhook_lead()}<code class="text-xs">event</code>,
                <code class="text-xs">occurred_at</code>, <code class="text-xs">title</code>,
                <code class="text-xs">message</code>{m.settings_webhook_mid()} <code class="text-xs">-</code> {m.settings_webhook_tail()}
            </p>
            {#if saveWebhookResult?.error}
                <Alert variant="error" message={saveWebhookResult.error} />
            {/if}
            {#if saveWebhookResult?.ok}
                <Alert variant="success" message={m.settings_webhook_saved()} />
            {/if}
            {#if testWebhookResult?.error}
                <Alert variant="error" message={testWebhookResult.error} />
            {/if}
            {#if testWebhookResult?.ok}
                <Alert variant="success" message={m.settings_webhook_test_sent()} />
            {/if}

            <form method="POST" action="?/saveWebhook" use:enhance class="grid grid-cols-1 gap-4">
                <label class="label cursor-pointer justify-start gap-3">
                    <input
                        type="checkbox"
                        name="enabled"
                        class="toggle toggle-primary"
                        checked={integrations?.webhook?.enabled ?? false}
                    />
                    <span class="label-text">{m.settings_webhook_enable()}</span>
                </label>

                <label class="form-control w-full">
                    <span class="label-text">{m.settings_webhook_url_label()}</span>
                    <input
                        type="url"
                        name="url"
                        class="input input-bordered w-full"
                        autocomplete="off"
                        placeholder="https://example.com/hooks/sopandgo"
                        value={integrations?.webhook?.url ?? ''}
                    />
                </label>

                <label class="form-control w-full">
                    <span class="label-text">{m.settings_bearer()}</span>
                    <input
                        type="password"
                        name="bearer_token"
                        class="input input-bordered w-full"
                        autocomplete="off"
                        placeholder={integrations?.webhook?.secret_configured
                            ? m.settings_keep_or_clear()
                            : m.common_optional()}
                    />
                </label>

                <fieldset class="space-y-2">
                    <legend class="text-sm font-medium">{m.common_events()}</legend>
                    {#each knownEvents as event (event)}
                        <label class="label cursor-pointer justify-start gap-3 py-1">
                            <input
                                type="checkbox"
                                name="events"
                                value={event}
                                class="checkbox checkbox-sm"
                                checked={eventChecked(integrations?.webhook?.events, event)}
                            />
                            <span class="label-text">{eventLabels[event] ?? event}</span>
                        </label>
                    {/each}
                </fieldset>

                <div class="card-actions w-full flex-col sm:flex-row sm:justify-end">
                    <button type="submit" class="btn btn-primary w-full sm:w-auto">{m.settings_save_webhook()}</button>
                </div>
            </form>
            <form method="POST" action="?/testWebhook" use:enhance>
                <button type="submit" class="btn btn-secondary btn-sm">{m.common_send_test()}</button>
            </form>
        </div>
    </Card>
</div>