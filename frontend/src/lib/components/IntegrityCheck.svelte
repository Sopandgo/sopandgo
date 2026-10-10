<script lang="ts">
    import { applyAction, enhance } from '$app/forms';
    import { page } from '$app/state';
    import { RotateCwIcon, ShieldQuestionMarkIcon } from 'lucide-svelte';
    import IntegrityStatusBadge from './IntegrityStatusBadge.svelte';
    import { integrityResultFromForm, type IntegrityStatus } from '$lib/integrity';
    import * as m from '$lib/paraglide/messages.js';

    /*
     * Posts to the page's ?/verifyAsset or ?/verifyVersion action
     * (see $lib/server/integrityActions) and shows the result in place.
     */
    interface Props {
        kind: 'asset' | 'version';
        id: string;
        /** Status known before any manual check, e.g. from the page load. */
        initial?: IntegrityStatus | null;
        onresult?: (status: IntegrityStatus) => void;
    }

    let { kind, id, initial = null, onresult }: Props = $props();

    const action = $derived(kind === 'asset' ? 'verifyAsset' : 'verifyVersion');
    const field = $derived(kind === 'asset' ? 'asset_id' : 'version_id');

    let checked = $state<IntegrityStatus | null>(null);
    let loading = $state(false);

    const status = $derived(checked ?? integrityResultFromForm(page.form, action, id) ?? initial);
</script>

<form
    method="POST"
    action="?/{action}"
    class="inline-flex items-center gap-1"
    aria-live="polite"
    use:enhance={() => {
        loading = true;
        return async ({ result }) => {
            loading = false;
            if (result.type === 'redirect') {
                await applyAction(result);
                return;
            }
            const next = result.type === 'success' || result.type === 'failure'
                ? integrityResultFromForm(result.data, action, id)
                : null;
            checked = next ?? 'unavailable';
            onresult?.(checked);
        };
    }}
>
    <input type="hidden" name={field} value={id} />

    {#if status && !loading}
        <IntegrityStatusBadge {status} />
        {#if status === 'unavailable'}
            <button type="submit" class="btn btn-ghost btn-sm gap-1">
                <RotateCwIcon class="size-4" aria-hidden="true" />
                {m.integrity_retry()}
            </button>
        {:else}
            <button
                type="submit"
                class="btn btn-ghost btn-square btn-sm"
                title={m.integrity_recheck()}
                aria-label={m.integrity_recheck()}
            >
                <RotateCwIcon class="size-4" aria-hidden="true" />
            </button>
        {/if}
    {:else}
        <!-- Idle and checking share one button so the row does not jump. -->
        <button type="submit" class="btn btn-ghost btn-sm gap-1" disabled={loading}>
            {#if loading}
                <span class="loading loading-spinner loading-xs" aria-hidden="true"></span>
                {m.integrity_checking()}
            {:else}
                <ShieldQuestionMarkIcon class="size-4" aria-hidden="true" />
                {m.integrity_verify()}
            {/if}
        </button>
    {/if}
</form>
