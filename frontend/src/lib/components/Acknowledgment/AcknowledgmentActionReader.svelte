<script lang="ts">
    import { enhance } from '$app/forms';
    import type { User } from '$lib/sdk/types';
    import Alert from '$lib/components/Alert.svelte';
    import Card from '$lib/components/Card.svelte';
    import { SignatureIcon } from 'lucide-svelte';
    import * as m from '$lib/paraglide/messages.js';

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
    const error = $derived(form?.action === 'sign' ? form.error : undefined);
</script>

<Card title={m.ack_reader_title()} class="w-full">
    <div class="card-body">
        <p class="text-sm text-base-content/70">
            {m.ack_reader_body()}
        </p>

        <form
            method="POST"
            action="?/sign"    
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
                <label class="label" for="user_display_name">
                    <span class="label-text">{m.ack_type_name()} <span class="font-medium">{user.display_name}</span></span>
                </label>
                <input
                    id="user_display_name"
                    name="user_display_name"
                    type="text"
                    class="input w-full mt-4 {error ? 'input-error' : ''}"
                    placeholder={user.display_name}
                    value={form?.inputName ?? ''} 
                    required
                    autocomplete="off"
                />
            </div>

            <div class="card-actions justify-end">
                <button class="btn btn-primary" disabled={loading}>
                    {#if loading}
                        <span class="loading loading-spinner loading-xs"></span>
                        {m.ack_signing()}
                    {:else}
                        <SignatureIcon class="size-4" />
                        {m.ack_sign_reader()}
                    {/if}
                </button>
            </div>
        </form>
    </div>
</Card>
