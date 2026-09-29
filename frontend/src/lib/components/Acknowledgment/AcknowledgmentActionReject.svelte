<script lang="ts">
    import { enhance } from '$app/forms';
    import type { User } from '$lib/sdk/types';
    import Alert from '$lib/components/Alert.svelte';
    import { BanIcon } from 'lucide-svelte';

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

<div class="bg-base-100 rounded-box shadow-md border border-error w-full">

    <div class="p-4 pb-2 text-xs tracking-widest uppercase font-bold text-error">
        Reject this Standard Operating Procedure
    </div>
    <div class="card-body">
        <p class="text-sm opacity-70">
            With this action, you permanently reject this release candidate.
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
                <Alert variant="error" message={form.error} />
            {/if}

            <div class="form-control w-full my-4">
                <label class="label" for="reason">
                    <span class="label-text">Provide a <strong>reason for rejecting</strong>:</span>
                </label>
                <input
                    id="reason"
                    name="reason"
                    type="text"
                    class="input input-bordered w-full mt-4 {form?.error ? 'input-error' : ''}"
                    placeholder="Rejected because ..."
                    value={form?.inputName ?? ''} 
                    required
                    autocomplete="off"
                />
            </div>

            <div class="card-actions justify-end">
                <button class="btn btn-error" disabled={loading}>
                    {#if loading}
                        <span class="loading loading-spinner loading-xs"></span>
                        Rejecting ...
                    {:else}
                        <BanIcon class="w-4 h-4" />
                        Reject Release Candidate
                    {/if}
                </button>
            </div>
        </form>
    </div>
</div>