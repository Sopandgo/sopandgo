<script lang="ts">
    import { enhance } from '$app/forms';
    import type { User } from '$lib/sdk/types';
    import Alert from '$lib/components/Alert.svelte';
    import Card from '$lib/components/Card.svelte';
    import { BanIcon } from 'lucide-svelte';
    import * as m from '$lib/paraglide/messages.js';

    // 1. Define the shape locally
    interface SignFormResult {
        error?: string;
        inputName?: string;
        success?: boolean;
    }

    let {
        user,
        form
    } = $props<{
        user: User;
        form?: SignFormResult | null; 
    }>();

    let loading = $state(false);
</script>

<Card title={m.ack_reject_title()} class="w-full">
    <div class="card-body">
        <p class="text-sm text-base-content/70">
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
            {#if form?.error}
                <Alert type="error" message={form.error} />
            {/if}

            <div class="form-control w-full my-4">
                <label class="label" for="reason">
                    <span class="label-text">{m.ack_reject_reason_before()} <strong>{m.ack_reject_reason_strong()}</strong>{m.ack_reject_reason_after()}</span>
                </label>
                <input
                    id="reason"
                    name="reason"
                    type="text"
                    class="input w-full mt-4 {form?.error ? 'input-error' : ''}"
                    placeholder={m.ack_reject_placeholder()}
                    value={form?.inputName ?? ''} 
                    required
                    autocomplete="off"
                />
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
</Card>
