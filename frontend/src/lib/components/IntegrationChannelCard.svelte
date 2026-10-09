<script lang="ts">
    import { enhance } from '$app/forms';
    import type { Snippet } from 'svelte';
    import { ChevronDownIcon } from 'lucide-svelte';
    import Alert from '$lib/components/Alert.svelte';
    import Card from '$lib/components/Card.svelte';
    import * as m from '$lib/paraglide/messages.js';
    import type { IntegrationEvent, PublicIntegrationChannel } from '$lib/sdk/types';

    /*
     * One outbound channel on Settings → Integrations, collapsed to a summary row
     * (name, status word, event count) until opened. Save and Send test share one
     * form; the test action ignores the fields and sends with the saved settings.
     */
    type ActionResult = { error?: string; ok?: boolean } | undefined;

    interface Props {
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
        savedMessage: string;
        testSentMessage: string;
        /** Lead text above the fields (optional). */
        description?: Snippet;
        /** Channel-specific inputs (URL, token). */
        fields: Snippet;
    }

    let {
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
        savedMessage,
        testSentMessage,
        description,
        fields
    }: Props = $props();

    const hasResult = () => Boolean(saveResult || testResult);

    // Collapsed by default; opens when its own save or test returns, so the alert is seen.
    let open = $state(hasResult());
    $effect(() => {
        if (hasResult()) open = true;
    });

    let pending = $state<'save' | 'test' | null>(null);

    const status = $derived(
        !channel?.configured ? 'not_configured' : channel.enabled ? 'on' : 'off'
    );
    const selectedCount = $derived(
        knownEvents.filter((e) => (channel?.events ?? []).includes(e)).length
    );
</script>

<Card>
    <details class="group" bind:open>
        <summary
            class="flex cursor-pointer list-none items-center gap-3 rounded-box p-4 transition-colors duration-150 hover:bg-base-200 focus-visible:outline-offset-[-2px] group-open:rounded-b-none group-open:border-b group-open:border-base-300 sm:p-6 [&::-webkit-details-marker]:hidden"
        >
            <div class="min-w-0 flex-1">
                <h2 class="text-lg font-semibold">{title}</h2>
                <p class="mt-1 text-sm text-base-content/70">
                    {m.settings_integration_events_selected({
                        count: String(selectedCount),
                        total: String(knownEvents.length)
                    })}
                </p>
            </div>
            {#if status === 'on'}
                <span class="badge badge-soft badge-success">{m.settings_integration_on()}</span>
            {:else if status === 'off'}
                <span class="badge badge-outline">{m.settings_integration_off()}</span>
            {:else}
                <span class="badge badge-outline">{m.settings_integration_not_configured()}</span>
            {/if}
            <ChevronDownIcon
                class="size-5 shrink-0 text-base-content/70 transition-transform duration-150 group-open:rotate-180 motion-reduce:transition-none"
                aria-hidden="true"
            />
        </summary>

        <div class="card-body space-y-4">
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
                <label class="flex w-fit cursor-pointer items-center gap-3 text-sm">
                    <input
                        type="checkbox"
                        name="enabled"
                        class="toggle toggle-success peer"
                        checked={channel?.enabled ?? false}
                    />
                    <span class="font-medium">{enableLabel}</span>
                    <!-- The word repeats the toggle's state for sighted users; the checkbox conveys it to assistive tech. -->
                    <span class="text-base-content/70 peer-checked:hidden" aria-hidden="true">
                        {m.settings_integration_off()}
                    </span>
                    <span class="hidden text-base-content/70 peer-checked:inline" aria-hidden="true">
                        {m.settings_integration_on()}
                    </span>
                </label>

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
        </div>
    </details>
</Card>
