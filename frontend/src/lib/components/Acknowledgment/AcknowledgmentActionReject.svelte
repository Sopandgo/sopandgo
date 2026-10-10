<script lang="ts">
    import { enhance } from '$app/forms';
    import type { User } from '#lib/sdk/types.js';
    import Alert from '#lib/components/Alert.svelte';
    import { BanIcon } from '@lucide/svelte';
    import * as m from '#lib/paraglide/messages.js';

    // 1. Define the shape locally
    interface SignFormResult {
        error?: string;
        inputName?: string;
        success?: boolean;
        /** Which action the result belongs to, so only that form shows the error. */
        action?: string;
    }

    let {
        user,
        form
    } = $props<{
        user: User;
        form?: SignFormResult | null; 
    }>();

    let loading = $state(false);
    const error = $derived(form?.action === 'reject' ? form.error : undefined);
</script>

<!-- No card of its own: it sits in the review card under the approve form. -->
<div>
    <h3 class="font-semibold">{m.ack_reject_title()}</h3>
    <p class="mt-1 text-sm text-base-content/70">
        {m.ack_reject_body()}
    </p>

    <form
        method="POST"
        action="?/reject"    
        use:enhance={() => {
            loading = true;
            return async ({ update }) => {
                await update();
                loading = false;
            };
        }}
    >
        {#if error}
            <Alert type="error" message={error} />
        {/if}

        <div class="form-control w-full my-4">
            <label class="label" for="reason">
                <span class="label-text">{m.ack_reject_reason_before()} <strong>{m.ack_reject_reason_strong()}</strong>{m.ack_reject_reason_after()}</span>
            </label>
            <!-- Kept with the rejection and shown on the version afterwards -->
            <textarea
                id="reason"
                name="reason"
                rows="3"
                maxlength="500"
                class="textarea w-full mt-4 {error ? 'textarea-error' : ''}"
                placeholder={m.ack_reject_placeholder()}
                value={form?.action === 'reject' ? (form.inputName ?? '') : ''}
                required
                autocomplete="off"
            ></textarea>
            <p class="mt-1 text-xs text-base-content/70">{m.ack_reject_reason_help()}</p>
        </div>

        <div class="card-actions justify-end">
            <button class="btn text-error" disabled={loading}>
                {#if loading}
                    <span class="loading loading-spinner loading-xs"></span>
                    {m.ack_rejecting()}
                {:else}
                    <BanIcon class="size-4" />
                    {m.ack_reject_button()}
                {/if}
            </button>
        </div>
    </form>
</div>
