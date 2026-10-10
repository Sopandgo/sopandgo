<script lang="ts">
    import Alert from '$lib/components/Alert.svelte';
    import { enhance } from '$app/forms';
    import { NotebookPenIcon, RocketIcon } from 'lucide-svelte';
    import type { ActionData } from './$types';
  import Card from '$lib/components/Card.svelte';
  import CardPageHeading from '$lib/components/CardPageHeading.svelte';
  import * as m from '$lib/paraglide/messages.js';

    let { form } = $props<{ form: ActionData }>();
    
    let loading = $state(false);
</script>

<svelte:head>
    <title>{m.page_new_sop()}</title>
</svelte:head>

<div class="flex flex-col gap-6">
    <Card>
        <div class="card-body">
            <CardPageHeading>
                <NotebookPenIcon class="w-8 h-8" />
                {m.sops_heading()}
            </CardPageHeading>

            <div class="flex items-center gap-2 text-base-content/70 mb-4">
                <span class="text-sm font-medium">
                    {m.sops_create_help()}
                </span> 
            </div>

            {#if form?.message}
                <Alert type="error" message={form.message} class="my-4" />
            {/if}

            <form 
                method="POST" 
                use:enhance={() => {
                    loading = true;
                    return async ({ update }) => {
                        try {
                            // This waits for the redirect page to fully load
                            await update(); 
                        } catch (err) {
                            console.error("Navigation failed:", err);
                        } finally {
                            // CRITICAL: This runs even if the redirect crashes/fails
                            loading = false; 
                        }
                    };
                }}
            >
                <div class="form-control w-full">
                    <label class="label" for="title">
                        <span class="label-text font-medium">{m.sops_title_label()}</span>
                    </label>
                    <input 
                        id="title"
                        name="title" 
                        type="text" 
                        placeholder={m.sops_title_placeholder()} 
                        class="input w-full {form?.message ? 'input-error' : ''}" 
                        value={form?.title ?? ''}
                        required 
                    />
                </div>

                <div class="card-actions justify-end mt-4 flex items-center gap-4">
                    <a href="/sops" class="btn btn-ghost">{m.common_cancel()}</a>
                    <button type="submit" class="btn min-w-[120px]" disabled={loading}>
                        {#if loading}
                            <span class="loading loading-spinner"></span>
                        {/if}
                        {m.sops_create()}
                        <RocketIcon class="w-5 h-5"/>
                    </button>
                </div>
            </form>
        </div>
    </Card>
</div>