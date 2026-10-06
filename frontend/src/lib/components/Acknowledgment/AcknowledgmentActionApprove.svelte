<script lang="ts">
    import { enhance } from '$app/forms';
    import type { User } from '$lib/sdk/types';
    import Alert from '$lib/components/Alert.svelte';
    import { SignatureIcon } from 'lucide-svelte';
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

<div class="bg-base-100 rounded-box shadow-md border border-success w-full">

    <div class="p-4 pb-2 text-xs tracking-widest uppercase font-bold text-success">
        {m.ack_approve_title()}
    </div>
    <div class="card-body">
        <p class="text-sm opacity-70">
            {m.ack_approve_body()}
            <br/>
            {m.ack_approve_supersede()}
        </p>

        <form
            method="POST"
            action="?/approve"    
            use:enhance={() => {
                loading = true;
                return async ({ update }) => {
                    await update();
                    loading = false;
                };
            }}
        >
            {#if form?.error}
                <Alert variant="error" message={form.error} />
            {/if}

            <div class="form-control w-full my-4">
                <label class="label" for="user_display_name">
                    <span class="label-text">{m.ack_type_name()} <span class="font-bold">{user.display_name}</span></span>
                </label>
                <input
                    id="user_display_name"
                    name="user_display_name"
                    type="text"
                    class="input input-bordered w-full mt-4 {form?.error ? 'input-error' : ''}"
                    placeholder={user.display_name}
                    value={form?.inputName ?? ''} 
                    required
                    autocomplete="off"
                />
            </div>

            <div class="card-actions justify-end">
                <button class="btn btn-success" disabled={loading}>
                    {#if loading}
                        <span class="loading loading-spinner loading-xs"></span>
                        {m.ack_signing()}
                    {:else}
                        <SignatureIcon class="w-4 h-4" />
                        {m.ack_approve_publish()}
                    {/if}
                </button>
            </div>
        </form>
    </div>
</div>