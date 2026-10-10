<script lang="ts">
    import Alert from '$lib/components/Alert.svelte';
    import Card from '$lib/components/Card.svelte';
    import CardPageHeading from '$lib/components/CardPageHeading.svelte';
    import IntegrationChannelCard from '$lib/components/IntegrationChannelCard.svelte';
    import { WebhookIcon } from 'lucide-svelte';
    import * as m from '$lib/paraglide/messages.js';
    import type { IntegrationEvent } from '$lib/sdk/types';

    let { data, form }: { data: any; form: any } = $props();

    const integrations = $derived(data.integrations);
    const saveSlackResult = $derived(form?.saveSlack as { error?: string; ok?: boolean } | undefined);
    const saveGotifyResult = $derived(form?.saveGotify as { error?: string; ok?: boolean } | undefined);
    const saveWebhookResult = $derived(form?.saveWebhook as { error?: string; ok?: boolean } | undefined);
    const testSlackResult = $derived(form?.testSlack as { error?: string; ok?: boolean } | undefined);
    const testGotifyResult = $derived(form?.testGotify as { error?: string; ok?: boolean } | undefined);
    const testWebhookResult = $derived(form?.testWebhook as { error?: string; ok?: boolean } | undefined);

    // The header toggle's result, for the one channel it switched.
    function toggleResult(channel: 'slack' | 'gotify' | 'webhook') {
        const r = form?.setEnabled as { channel?: string; error?: string; ok?: boolean } | undefined;
        return r?.channel === channel ? r : undefined;
    }

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
</script>

<svelte:head>
    <title>{m.settings_integrations()}</title>
</svelte:head>

<div class="flex flex-col gap-6">
    <Card>
        <div class="card-body space-y-4">
            <CardPageHeading>
                <WebhookIcon class="w-8 h-8" />
                {m.settings_integrations()}
            </CardPageHeading>
            <p class="text-sm text-base-content/70">
                {m.settings_integrations_before()}
                <code class="text-xs bg-base-200 px-1 rounded">SECRET_ENCRYPTION_KEY</code>
                {m.settings_integrations_after()}
            </p>
            {#if integrations && !integrations.encryption_key_set}
                <Alert
                    type="error"
                    message={m.settings_integrations_key_missing()}
                />
            {/if}
        </div>
    </Card>

    <IntegrationChannelCard
        channelKey="slack"
        title={m.settings_slack()}
        channel={integrations?.slack}
        {knownEvents}
        {eventLabels}
        enableLabel={m.settings_slack_enable()}
        saveLabel={m.settings_save_slack()}
        saveAction="saveSlack"
        testAction="testSlack"
        saveResult={saveSlackResult}
        testResult={testSlackResult}
        toggleResult={toggleResult('slack')}
        savedMessage={m.settings_slack_saved()}
        testSentMessage={m.settings_slack_test_sent()}
    >
        {#snippet fields()}
            <label class="flex flex-col gap-1.5">
                <span class="text-sm font-medium">{m.settings_webhook_url()}</span>
                <input
                    type="password"
                    name="webhook_url"
                    class="input w-full"
                    autocomplete="off"
                    placeholder={integrations?.slack?.secret_configured
                        ? m.settings_keep_webhook()
                        : 'https://hooks.slack.com/services/...'}
                />
            </label>
        {/snippet}
    </IntegrationChannelCard>

    <IntegrationChannelCard
        channelKey="gotify"
        title={m.settings_gotify()}
        channel={integrations?.gotify}
        {knownEvents}
        {eventLabels}
        enableLabel={m.settings_gotify_enable()}
        saveLabel={m.settings_save_gotify()}
        saveAction="saveGotify"
        testAction="testGotify"
        saveResult={saveGotifyResult}
        testResult={testGotifyResult}
        toggleResult={toggleResult('gotify')}
        savedMessage={m.settings_gotify_saved()}
        testSentMessage={m.settings_gotify_test_sent()}
    >
        {#snippet fields()}
            <label class="flex flex-col gap-1.5">
                <span class="text-sm font-medium">{m.settings_gotify_url()}</span>
                <input
                    type="url"
                    name="url"
                    class="input w-full"
                    autocomplete="off"
                    placeholder="https://gotify.example.com"
                    value={integrations?.gotify?.url ?? ''}
                />
            </label>
            <label class="flex flex-col gap-1.5">
                <span class="text-sm font-medium">{m.settings_gotify_token()}</span>
                <input
                    type="password"
                    name="token"
                    class="input w-full"
                    autocomplete="off"
                    placeholder={integrations?.gotify?.secret_configured
                        ? m.settings_keep_token()
                        : m.common_required()}
                />
            </label>
        {/snippet}
    </IntegrationChannelCard>

    <IntegrationChannelCard
        channelKey="webhook"
        title={m.settings_webhook()}
        channel={integrations?.webhook}
        {knownEvents}
        {eventLabels}
        enableLabel={m.settings_webhook_enable()}
        saveLabel={m.settings_save_webhook()}
        saveAction="saveWebhook"
        testAction="testWebhook"
        saveResult={saveWebhookResult}
        testResult={testWebhookResult}
        toggleResult={toggleResult('webhook')}
        savedMessage={m.settings_webhook_saved()}
        testSentMessage={m.settings_webhook_test_sent()}
    >
        {#snippet description()}
            <p>
                {m.settings_webhook_lead()}<code class="font-mono text-xs">event</code>,
                <code class="font-mono text-xs">occurred_at</code>, <code class="font-mono text-xs">title</code>,
                <code class="font-mono text-xs">message</code>{m.settings_webhook_mid()} <code class="font-mono text-xs">-</code> {m.settings_webhook_tail()}
            </p>
        {/snippet}
        {#snippet fields()}
            <label class="flex flex-col gap-1.5">
                <span class="text-sm font-medium">{m.settings_webhook_url_label()}</span>
                <input
                    type="url"
                    name="url"
                    class="input w-full"
                    autocomplete="off"
                    placeholder="https://example.com/hooks/sopandgo"
                    value={integrations?.webhook?.url ?? ''}
                />
            </label>
            <label class="flex flex-col gap-1.5">
                <span class="text-sm font-medium">{m.settings_bearer()}</span>
                <input
                    type="password"
                    name="bearer_token"
                    class="input w-full"
                    autocomplete="off"
                    placeholder={integrations?.webhook?.secret_configured
                        ? m.settings_keep_or_clear()
                        : m.common_optional()}
                />
            </label>
        {/snippet}
    </IntegrationChannelCard>
</div>
