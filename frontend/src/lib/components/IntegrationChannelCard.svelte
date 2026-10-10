<script lang="ts">
    import { enhance } from '$app/forms';
    import type { Snippet } from 'svelte';
    import Alert from '$lib/components/Alert.svelte';
    import CollapsibleCard from '$lib/components/CollapsibleCard.svelte';
    import * as m from '$lib/paraglide/messages.js';
    import type { IntegrationEvent, PublicIntegrationChannel } from '$lib/sdk/types';

    /*
     * One outbound channel on Settings → Integrations, collapsed to a header row
     * until opened. Once the channel is configured the header carries its on/off
     * toggle, so every channel's state is visible without opening it. Save and
     * Send test share one form; the test action sends with the saved settings.
     */
    type ActionResult = { error?: string; ok?: boolean } | undefined;

    interface Props {
        /** Channel key the `setEnabled` action switches. */
        channelKey: 'slack' | 'gotify' | 'webhook';
        title: string;
        channel: PublicIntegrationChannel | undefined;
        knownEvents: IntegrationEvent[];
        eventLabels: Record<IntegrationEvent, string>;
        enableLabel: string;
        saveLabel: string;
        /** Form action names, e.g. `saveSlack` and `testSlack`. */
        saveAction: string;
        testAction: string;
        saveResult: ActionResult;
        testResult: ActionResult;
        toggleResult: ActionResult;
        savedMessage: string;
        testSentMessage: string;
        /** Lead text above the fields (optional). */
        description?: Snippet;
        /** Channel-specific inputs (URL, token). */
        fields: Snippet;
    }

    let {
        channelKey,
        title,
        channel,
        knownEvents,
        eventLabels,
        enableLabel,
        saveLabel,
        saveAction,
        testAction,
        saveResult,
        testResult,
        toggleResult,
        savedMessage,
        testSentMessage,
        description,
        fields
    }: Props = $props();

    // The header toggle flips at once and follows the saved value after each reload.
    const savedEnabled = () => channel?.enabled ?? false;
    let enabled = $state(savedEnabled());
    $effect(() => {
        enabled = savedEnabled();
    });

    let pending = $state<'save' | 'test' | null>(null);

    const configured = $derived(channel?.configured ?? false);
    const selectedCount = $derived(
        knownEvents.filter((e) => (channel?.events ?? []).includes(e)).length
    );
</script>

{#snippet toggleError()}
    <Alert type="error" message={toggleResult?.error} />
{/snippet}

<CollapsibleCard
    {title}
    meta={m.settings_integration_events_selected({
        count: String(selectedCount),
        total: String(knownEvents.length)
    })}
    openWhen={Boolean(saveResult || testResult)}
    notice={toggleResult?.error ? toggleError : undefined}
>
    {#snippet trailing()}
        {#if configured}
            <form
                method="POST"
                action="?/setEnabled"
                use:enhance={() => {
                    return async ({ result, update }) => {
                        await update({ reset: false });
                        // On failure the data does not change, so put the toggle back by hand.
                        if (result.type === 'failure' || result.type === 'error') {
                            enabled = channel?.enabled ?? false;
                        }
                    };
                }}
            >
                <input type="hidden" name="channel" value={channelKey} />
                <label class="flex cursor-pointer items-center gap-2 text-sm">
                    <!-- The word repeats the toggle's state for sighted users; the checkbox conveys it to assistive tech. -->
                    <span class="text-base-content/70" aria-hidden="true">
                        {enabled ? m.settings_integration_on() : m.settings_integration_off()}
                    </span>
                    <input
                        type="checkbox"
                        name="enabled"
                        class="toggle toggle-success"
                        aria-label={enableLabel}
                        bind:checked={enabled}
                        onchange={(e) => e.currentTarget.form?.requestSubmit()}
                    />
                </label>
            </form>
        {:else}
            <span class="badge badge-outline">{m.common_not_configured()}</span>
        {/if}
    {/snippet}

    {#if description}
        <div class="text-sm text-base-content/70">{@render description()}</div>
    {/if}
    {#if saveResult?.error}
        <Alert type="error" message={saveResult.error} />
    {:else if saveResult?.ok}
        <Alert type="success" message={savedMessage} />
    {/if}
    {#if testResult?.error}
        <Alert type="error" message={testResult.error} />
    {:else if testResult?.ok}
        <Alert type="success" message={testSentMessage} />
    {/if}

    <form
        method="POST"
        action="?/{saveAction}"
        class="grid grid-cols-1 gap-5"
        use:enhance={({ submitter }) => {
            pending = submitter?.getAttribute('formaction') ? 'test' : 'save';
            return async ({ update }) => {
                // Testing must not wipe edits the admin has not saved yet.
                await update({ reset: pending === 'save' });
                pending = null;
            };
        }}
    >
        <!-- On/off lives only in the header toggle, shown once this is configured; saving the fields keeps it. -->
        {#if configured && enabled}
            <input type="hidden" name="enabled" value="on" />
        {/if}

        {@render fields()}

        <fieldset>
            <legend class="mb-2 text-sm font-medium">{m.common_events()}</legend>
            <div class="grid grid-cols-1 gap-x-6 gap-y-2 sm:grid-cols-2">
                {#each knownEvents as event (event)}
                    <label class="flex cursor-pointer items-center gap-3 text-sm">
                        <input
                            type="checkbox"
                            name="events"
                            value={event}
                            class="checkbox checkbox-sm checkbox-primary"
                            checked={(channel?.events ?? []).includes(event)}
                        />
                        <span>{eventLabels[event] ?? event}</span>
                    </label>
                {/each}
            </div>
        </fieldset>

        <div
            class="flex flex-col-reverse gap-3 border-t border-base-300 pt-4 sm:flex-row sm:items-center sm:justify-between"
        >
            <p class="text-xs text-base-content/70">{m.settings_integrations_test_hint()}</p>
            <!-- Save comes first in the DOM so Enter in a field saves rather than tests. -->
            <div class="flex flex-col gap-2 sm:flex-row-reverse">
                <button type="submit" class="btn w-full sm:w-auto" disabled={pending !== null}>
                    {#if pending === 'save'}
                        <span class="loading loading-spinner loading-xs"></span>
                        {m.common_saving()}
                    {:else}
                        {saveLabel}
                    {/if}
                </button>
                <button
                    type="submit"
                    formaction="?/{testAction}"
                    class="btn btn-ghost w-full sm:w-auto"
                    disabled={pending !== null}
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
</CollapsibleCard>
