<script lang="ts">
    import { enhance } from '$app/forms';
    import Alert from '#lib/components/Alert.svelte';
    import Card from '#lib/components/Card.svelte';
    import CardPageHeading from '#lib/components/CardPageHeading.svelte';
    import { BugIcon } from '@lucide/svelte';
    import * as m from '#lib/paraglide/messages.js';
    import type { ErrorReportingSettings } from '#lib/sdk/types.js';

    let { data, form }: { data: any; form: any } = $props();

    const saved = $derived(data.settings as ErrorReportingSettings | null);
    type Values = { enabled: boolean; dsn: string; environment: string; browser_enabled: boolean };
    const shown = $derived<Values>(
        (form?.save?.values as Values | undefined) ??
            saved ?? { enabled: false, dsn: '', environment: 'production', browser_enabled: false }
    );

    // svelte-ignore state_referenced_locally
    let enabled = $state(shown.enabled);
    // svelte-ignore state_referenced_locally
    let dsn = $state(shown.dsn);
    $effect(() => {
        enabled = shown.enabled;
        dsn = shown.dsn;
    });

    const plainHttp = $derived(dsn.trim().toLowerCase().startsWith('http://'));
    let pending = $state<'save' | 'test' | null>(null);
</script>

<svelte:head>
    <title>{m.settings_error_reporting()}</title>
</svelte:head>

<div class="flex flex-col gap-6">
    <Card>
        <div class="card-body space-y-4">
            <CardPageHeading>
                <BugIcon class="w-8 h-8" />
                {m.settings_error_reporting()}
            </CardPageHeading>
            <p class="text-sm text-base-content/70">{m.error_reporting_intro()}</p>
        </div>
    </Card>

    <Card title={m.error_reporting_what_heading()}>
        <div class="card-body space-y-2 text-sm">
            <p>{m.error_reporting_sent()}</p>
            <p>{m.error_reporting_not_sent()}</p>
            <p class="text-base-content/70">{m.error_reporting_dsn_public()}</p>
        </div>
    </Card>

    <Card title={m.error_reporting_settings_heading()}>
        <div class="card-body space-y-4">
            {#if !saved}
                <Alert type="error" message={m.error_reporting_load_failed()} />
            {/if}
            {#if form?.save?.error}
                <Alert type="error" message={form.save.error} />
            {:else if form?.save?.ok}
                <Alert type="success" message={m.error_reporting_saved()} />
            {/if}
            {#if form?.test?.error}
                <Alert type="error" message={form.test.error} />
            {:else if form?.test?.ok}
                <Alert type="success" message={m.error_reporting_test_sent({ id: form.test.eventId })} />
            {/if}

            <form
                method="POST"
                action="?/save"
                class="grid grid-cols-1 gap-5"
                use:enhance={({ submitter }) => {
                    pending = submitter?.getAttribute('formaction') ? 'test' : 'save';
                    return async ({ update }) => {
                        // Testing must not wipe edits the admin has not saved yet.
                        await update({ reset: false });
                        pending = null;
                    };
                }}
            >
                <label class="flex cursor-pointer items-center gap-3 text-sm font-medium">
                    <input type="checkbox" name="enabled" class="toggle toggle-success" bind:checked={enabled} />
                    {m.error_reporting_enable()}
                </label>

                <label class="flex flex-col gap-1.5">
                    <span class="text-sm font-medium">{m.error_reporting_dsn()}</span>
                    <input
                        type="text"
                        name="dsn"
                        class="input w-full font-mono text-xs"
                        autocomplete="off"
                        spellcheck="false"
                        placeholder="https://<key>@glitchtip.example.org/<project>"
                        aria-describedby="error-reporting-dsn-help"
                        bind:value={dsn}
                    />
                    <span id="error-reporting-dsn-help" class="text-xs text-base-content/70">
                        {m.error_reporting_dsn_help()}
                    </span>
                </label>
                {#if plainHttp}
                    <Alert type="warning" message={m.error_reporting_http_warning()} />
                {/if}

                <label class="flex flex-col gap-1.5">
                    <span class="text-sm font-medium">{m.error_reporting_environment()}</span>
                    <input
                        type="text"
                        name="environment"
                        class="input w-full sm:max-w-xs"
                        autocomplete="off"
                        spellcheck="false"
                        maxlength="64"
                        aria-describedby="error-reporting-environment-help"
                        value={shown.environment}
                    />
                    <span id="error-reporting-environment-help" class="text-xs text-base-content/70">
                        {m.error_reporting_environment_help()}
                    </span>
                </label>

                <label class="flex items-start gap-3 text-sm {enabled ? 'cursor-pointer' : 'cursor-not-allowed'}">
                    <input
                        type="checkbox"
                        name="browser_enabled"
                        class="checkbox checkbox-sm checkbox-primary mt-0.5"
                        checked={shown.browser_enabled}
                        disabled={!enabled}
                        aria-describedby="error-reporting-browser-help"
                    />
                    <span class="space-y-0.5">
                        <span class="block font-medium">{m.error_reporting_browser()}</span>
                        <span id="error-reporting-browser-help" class="block text-base-content/70">
                            {m.error_reporting_browser_help()}
                        </span>
                    </span>
                </label>

                <div
                    class="flex flex-col-reverse gap-3 border-t border-base-300 pt-4 sm:flex-row sm:items-center sm:justify-between"
                >
                    <p class="text-xs text-base-content/70">{m.settings_integrations_test_hint()}</p>
                    <!-- Save comes first in the DOM so Enter in a field saves rather than tests. -->
                    <div class="flex flex-col gap-2 sm:flex-row-reverse">
                        <button type="submit" class="btn btn-primary w-full sm:w-auto" disabled={pending !== null}>
                            {#if pending === 'save'}
                                <span class="loading loading-spinner loading-xs"></span>
                                {m.common_saving()}
                            {:else}
                                {m.error_reporting_save()}
                            {/if}
                        </button>
                        <button
                            type="submit"
                            formaction="?/test"
                            class="btn btn-ghost w-full sm:w-auto"
                            disabled={pending !== null || !saved?.enabled}
                        >
                            {#if pending === 'test'}
                                <span class="loading loading-spinner loading-xs"></span>
                                {m.common_sending()}
                            {:else}
                                {m.common_send_test()}
                            {/if}
                        </button>
                    </div>
                </div>
            </form>
        </div>
    </Card>
</div>
