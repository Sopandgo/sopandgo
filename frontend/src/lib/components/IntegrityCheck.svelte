<script lang="ts">
    import { applyAction, enhance } from '$app/forms';
    import { page } from '$app/state';
    import { LoaderCircleIcon, RotateCwIcon, ShieldQuestionMarkIcon } from 'lucide-svelte';
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

    {#if loading}
        <span class="flex items-center gap-1 px-3 h-7 text-xs opacity-60 italic">
            <LoaderCircleIcon class="size-4 animate-spin" aria-hidden="true" />
            {m.integrity_checking()}
        </span>
    {:else if status}
        <IntegrityStatusBadge {status} />
        <button
            type="submit"
            class="btn btn-ghost btn-square btn-xs"
            title={m.integrity_recheck()}
            aria-label={m.integrity_recheck()}
        >
            <RotateCwIcon class="size-3" aria-hidden="true" />
        </button>
    {:else}
        <button type="submit" class="btn btn-ghost h-7 gap-1">
            <ShieldQuestionMarkIcon class="size-3" aria-hidden="true" />
            {m.integrity_verify()}
        </button>
    {/if}
</form>
