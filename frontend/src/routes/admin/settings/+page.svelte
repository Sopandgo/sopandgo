<script lang="ts">
    import { enhance } from '$app/forms';
    import Alert from '$lib/components/Alert.svelte';
    import Card from '$lib/components/Card.svelte';
    import CardPageHeading from '$lib/components/CardPageHeading.svelte';
    import { MailIcon, WebhookIcon } from 'lucide-svelte';
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
        sop_published: 'SOP published',
        sop_rc: 'Release candidate ready',
        sop_rejected: 'SOP rejected',
        backup_s3_failed: 'S3 backup failed',
        integrity_check_failed: 'Integrity check failed'
    };

    const knownEvents = $derived(
        (integrations?.known_events as IntegrationEvent[] | undefined) ??
            (Object.keys(eventLabels) as IntegrationEvent[])
    );

    function eventChecked(channelEvents: string[] | undefined, event: IntegrationEvent) {
        return (channelEvents ?? []).includes(event);
    }
</script>


<div class="flex flex-col gap-6">
    <Card>
        <div class="card-body space-y-4">
            <CardPageHeading>
                <MailIcon class="w-8 h-8" />
                Email
            </CardPageHeading>

            <p class="text-sm text-base-content/70">
                Outgoing mail can use <strong class="font-medium">Resend</strong> (API) or standard <strong class="font-medium">SMTP</strong>.
                Secrets you save in the admin UI are encrypted at rest with
                <code class="text-xs bg-base-200 px-1 rounded">SMTP_SECRET_ENCRYPTION_KEY</code> in the server
                environment (SMTP password and Resend API key). If you rotate that key, re-enter the secret and save.
            </p>

            {#if saveModeResult?.error}
                <Alert variant='error' message={saveModeResult.error}/>
            {/if}
            {#if saveModeResult?.ok}
                <Alert variant='success' message='Mail mode updated.'/>
            {/if}

            {#if saveTransportResult?.error}
                <Alert variant='error' message={saveTransportResult.error}/>
            {/if}
            {#if saveTransportResult?.ok}
                <Alert variant='success' message='Outbound transport updated.'/>
            {/if}

            <form method="POST" action="?/saveTransport" use:enhance class="">
                <label class="form-control w-full mb-4">
                    <span class="label-text">Outbound transport</span>
                    <select
                        name="mail_transport"
                        class="select select-bordered w-full"
                        value={saveTransportResult?.transport ?? smtp?.mail_transport ?? 'smtp'}
                    >
                        <option value="smtp">SMTP (your mail server)</option>
                        <option value="resend">Resend (https://resend.com)</option>
                    </select>
                </label>
                <div class="card-actions mt-4 w-full flex-col sm:flex-row sm:justify-end">
                    <button type="submit" class="btn btn-primary w-full sm:w-auto">Save transport</button>
                </div>
            </form>

            <form method="POST" action="?/saveMode" use:enhance class="">
                <label class="form-control w-full mb-4">
                    <span class="label-text">Mail delivery mode</span>
                    <select
                        name="mail_mode"
                        class="select select-bordered w-full"
                        value={saveModeResult?.mode ?? smtp?.mail_mode ?? 'smtp'}
                    >
                        <option value="smtp">SMTP delivery (send invites/resets by email)</option>
                        <option value="manual_links">Manual links (admin copies and shares reset links)</option>
                    </select>
                    <!-- <span class="label-text-alt text-base-content/70 mt-1">
                        Manual links mode skips SMTP delivery and returns one-time links to admins after user create/reset actions.
                    </span> -->
                </label>
                <div class="card-actions mt-4 w-full flex-col sm:flex-row sm:justify-end">
                    <button type="submit" class="btn btn-primary w-full sm:w-auto">Save mail mode</button>
                </div>
            </form>
        </div>
    </Card>

    <Card>
        <div class="card-body space-y-4">
            <h2 class="card-title">Resend settings</h2>
            <p class="text-sm text-base-content/70">
                Create an API key in the Resend dashboard. The verified sender must match your domain (or use Resend’s
                sandbox sender for testing).
            </p>

            {#if saveResendResult?.error}
                <Alert variant='error' message={saveResendResult.error}/>
            {/if}
            {#if saveResendResult?.ok}
                <Alert variant='success' message='Resend settings saved.'/>
            {/if}

            <form method="POST" action="?/saveResend" use:enhance class="grid grid-cols-1 gap-4">
                <label class="form-control w-full">
                    <span class="label-text">From address</span>
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
                    <span class="label-text">API key</span>
                    <input
                        type="password"
                        name="resend_api_key"
                        class="input input-bordered w-full"
                        autocomplete="new-password"
                        placeholder={smtp?.resend_api_key_configured ? 'Leave blank to keep current key' : 'Required'}
                    />
                </label>

                <div class="card-actions w-full flex-col sm:flex-row sm:justify-end">
                    <button type="submit" class="btn btn-primary w-full sm:w-auto">Save Resend</button>
                </div>
            </form>
        </div>
    </Card>

    <Card>
        <div class="card-body space-y-4">
            <h2 class="card-title">SMTP settings</h2>

            {#if smtp && !smtp.encryption_key_set}
                <Alert variant='error' message='
                    The server process does not have <code>SMTP_SECRET_ENCRYPTION_KEY</code> set. Saving email
                    settings and sending mail will fail until you set a 32-byte key in your server environment
                    (for Docker: <code>docker-compose.yml</code> under <code>environment</code>; for local dev:
                    your shell/IDE run configuration), then restart the backend/container.
                '/>
            {/if}

            {#if smtp?.mail_mode === 'manual_links'}
                <Alert variant='info' message='
                    Manual links mode is active. SMTP and Resend settings below are optional until you switch back to email delivery.
                '/>
            {/if}

            {#if saveResult?.error}
                <Alert variant='error' message={saveResult.error}/>
            {/if}
            {#if saveResult?.ok}
                <Alert variant='success' message='SMTP settings saved.'/>
            {/if}

            <form
                method="POST"
                action="?/save"
                use:enhance
                class="grid grid-cols-1 md:grid-cols-2 gap-4"
            >
                <label class="form-control w-full md:col-span-2">
                    <span class="label-text">SMTP host</span>
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
                    <span class="label-text">Port</span>
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
                    <span class="label-text">From address</span>
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
                    <span class="label-text">Username</span>
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
                    <span class="label-text">Password</span>
                    <input
                        type="password"
                        name="password"
                        class="input input-bordered w-full"
                        autocomplete="new-password"
                        placeholder={smtp?.password_configured ? 'Leave blank to keep current password' : 'Required'}
                    />
                </label>

                <div class="card-actions col-span-2 w-full flex-col sm:flex-row sm:justify-end">
                    <button type="submit" class="btn btn-primary w-full sm:w-auto">Save</button>
                </div>
            </form>

        </div>
    </Card>

    <Card>
        <div class="card-body space-y-4">
            <h2 class="card-title">Test delivery</h2>

            {#if testResult?.error}
                <Alert variant='error' message={testResult.error}/>
            {/if}
            {#if testResult?.ok}
                <Alert variant='success' message='Test message sent. Check the inbox.'/>
            {/if}

            <form method="POST" action="?/test" use:enhance class="flex flex-col sm:flex-row gap-4 sm:items-end">
                <label class="form-control flex-1 w-full">
                    <span class="label-text">Send test email to</span>
                    <input
                        type="email"
                        name="test_to"
                        class="input input-bordered w-full"
                        placeholder="you@example.com"
                    />
                </label>
                <button type="submit" class="btn btn-secondary w-full shrink-0 sm:w-auto">Send test</button>
            </form>
        </div>
    </Card>

    <Card>
        <div class="card-body space-y-4">
            <CardPageHeading>
                <WebhookIcon class="w-8 h-8" />
                Integrations
            </CardPageHeading>
            <p class="text-sm text-base-content/70">
                Instance-level outbound notifications (Slack Incoming Webhooks, Gotify, and a generic HTTP webhook).
                Secrets are encrypted with the same
                <code class="text-xs bg-base-200 px-1 rounded">SMTP_SECRET_ENCRYPTION_KEY</code>
                used for email. Failed deliveries are logged and never undo SOP lifecycle actions.
            </p>
            {#if integrations && !integrations.encryption_key_set}
                <Alert
                    variant="error"
                    message="SMTP_SECRET_ENCRYPTION_KEY is not set. You can enable channels later, but Slack webhook URLs and Gotify/webhook tokens cannot be stored until the key is configured."
                />
            {/if}
        </div>
    </Card>

    <Card>
        <div class="card-body space-y-4">
            <h2 class="card-title">Slack Incoming Webhook</h2>
            {#if saveSlackResult?.error}
                <Alert variant="error" message={saveSlackResult.error} />
            {/if}
            {#if saveSlackResult?.ok}
                <Alert variant="success" message="Slack settings saved." />
            {/if}
            {#if testSlackResult?.error}
                <Alert variant="error" message={testSlackResult.error} />
            {/if}
            {#if testSlackResult?.ok}
                <Alert variant="success" message="Slack test notification sent." />
            {/if}

            <form method="POST" action="?/saveSlack" use:enhance class="grid grid-cols-1 gap-4">
                <label class="label cursor-pointer justify-start gap-3">
                    <input
                        type="checkbox"
                        name="enabled"
                        class="toggle toggle-primary"
                        checked={integrations?.slack?.enabled ?? false}
                    />
                    <span class="label-text">Enable Slack notifications</span>
                </label>

                <label class="form-control w-full">
                    <span class="label-text">Webhook URL (https)</span>
                    <input
                        type="password"
                        name="webhook_url"
                        class="input input-bordered w-full"
                        autocomplete="off"
                        placeholder={integrations?.slack?.secret_configured
                            ? 'Leave blank to keep current webhook URL'
                            : 'https://hooks.slack.com/services/...'}
                    />
                </label>

                <fieldset class="space-y-2">
                    <legend class="text-sm font-medium">Events</legend>
                    {#each knownEvents as event}
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
                    <button type="submit" class="btn btn-primary w-full sm:w-auto">Save Slack</button>
                </div>
            </form>
            <form method="POST" action="?/testSlack" use:enhance>
                <button type="submit" class="btn btn-secondary btn-sm">Send test</button>
            </form>
        </div>
    </Card>

    <Card>
        <div class="card-body space-y-4">
            <h2 class="card-title">Gotify</h2>
            {#if saveGotifyResult?.error}
                <Alert variant="error" message={saveGotifyResult.error} />
            {/if}
            {#if saveGotifyResult?.ok}
                <Alert variant="success" message="Gotify settings saved." />
            {/if}
            {#if testGotifyResult?.error}
                <Alert variant="error" message={testGotifyResult.error} />
            {/if}
            {#if testGotifyResult?.ok}
                <Alert variant="success" message="Gotify test notification sent." />
            {/if}

            <form method="POST" action="?/saveGotify" use:enhance class="grid grid-cols-1 gap-4">
                <label class="label cursor-pointer justify-start gap-3">
                    <input
                        type="checkbox"
                        name="enabled"
                        class="toggle toggle-primary"
                        checked={integrations?.gotify?.enabled ?? false}
                    />
                    <span class="label-text">Enable Gotify notifications</span>
                </label>

                <label class="form-control w-full">
                    <span class="label-text">Gotify base URL</span>
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
                    <span class="label-text">Application token</span>
                    <input
                        type="password"
                        name="token"
                        class="input input-bordered w-full"
                        autocomplete="off"
                        placeholder={integrations?.gotify?.secret_configured
                            ? 'Leave blank to keep current token'
                            : 'Required'}
                    />
                </label>

                <fieldset class="space-y-2">
                    <legend class="text-sm font-medium">Events</legend>
                    {#each knownEvents as event}
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
                    <button type="submit" class="btn btn-primary w-full sm:w-auto">Save Gotify</button>
                </div>
            </form>
            <form method="POST" action="?/testGotify" use:enhance>
                <button type="submit" class="btn btn-secondary btn-sm">Send test</button>
            </form>
        </div>
    </Card>

    <Card>
        <div class="card-body space-y-4">
            <h2 class="card-title">Generic HTTP webhook</h2>
            <p class="text-sm text-base-content/70">
                POSTs a JSON envelope (<code class="text-xs">event</code>,
                <code class="text-xs">occurred_at</code>, <code class="text-xs">title</code>,
                <code class="text-xs">message</code>, SOP fields) for n8n, Zapier, Discord/Teams connectors, or custom scripts.
                Optional bearer token; send <code class="text-xs">-</code> to clear a stored token.
            </p>
            {#if saveWebhookResult?.error}
                <Alert variant="error" message={saveWebhookResult.error} />
            {/if}
            {#if saveWebhookResult?.ok}
                <Alert variant="success" message="Webhook settings saved." />
            {/if}
            {#if testWebhookResult?.error}
                <Alert variant="error" message={testWebhookResult.error} />
            {/if}
            {#if testWebhookResult?.ok}
                <Alert variant="success" message="Webhook test notification sent." />
            {/if}

            <form method="POST" action="?/saveWebhook" use:enhance class="grid grid-cols-1 gap-4">
                <label class="label cursor-pointer justify-start gap-3">
                    <input
                        type="checkbox"
                        name="enabled"
                        class="toggle toggle-primary"
                        checked={integrations?.webhook?.enabled ?? false}
                    />
                    <span class="label-text">Enable generic webhook</span>
                </label>

                <label class="form-control w-full">
                    <span class="label-text">Webhook URL</span>
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
                    <span class="label-text">Bearer token (optional)</span>
                    <input
                        type="password"
                        name="bearer_token"
                        class="input input-bordered w-full"
                        autocomplete="off"
                        placeholder={integrations?.webhook?.secret_configured
                            ? 'Leave blank to keep; enter - to clear'
                            : 'Optional'}
                    />
                </label>

                <fieldset class="space-y-2">
                    <legend class="text-sm font-medium">Events</legend>
                    {#each knownEvents as event}
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
                    <button type="submit" class="btn btn-primary w-full sm:w-auto">Save webhook</button>
                </div>
            </form>
            <form method="POST" action="?/testWebhook" use:enhance>
                <button type="submit" class="btn btn-secondary btn-sm">Send test</button>
            </form>
        </div>
    </Card>
</div>