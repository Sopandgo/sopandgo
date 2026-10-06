<script lang="ts">
    import { enhance } from '$app/forms';
    import { HouseIcon, NotebookIcon, NotebookPenIcon, PlusIcon, RocketIcon } from 'lucide-svelte';
    import type { ActionData } from './$types';
    import Breadcrumbs from '$lib/components/Breadcrumbs.svelte';
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
    <Breadcrumbs items={[
        { label: m.page_dashboard(), href: '/sops', icon: HouseIcon},
        { label: m.page_sops(), href: '/sops', icon: NotebookIcon},
        { label: m.common_new_sop(), icon: PlusIcon},
    ]}/>

    <Card>
        <div class="card-body">
            <CardPageHeading color="accent">
                <NotebookPenIcon class="w-8 h-8" />
                {m.sops_heading()}
            </CardPageHeading>

            <div class="flex items-center gap-2 text-base-content/70 mb-4">
                <span class="text-sm font-medium">
                    {m.sops_register_help()}
                </span> 
            </div>

            {#if form?.message}
                <div role="alert" class="alert alert-error my-4 shadow-sm">
                    <svg xmlns="http://www.w3.org/2000/svg" class="stroke-current shrink-0 h-6 w-6" fill="none" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M10 14l2-2m0 0l2-2m-2 2l-2-2m2 2l2 2m7-2a9 9 0 11-18 0 9 9 0 0118 0z" /></svg>
                    <span>{form.message}</span>
                </div>
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
                        class="input input-bordered w-full {form?.message ? 'input-error' : ''}" 
                        value={form?.title ?? ''}
                        required 
                    />
                </div>

                <div class="card-actions justify-end mt-4 flex items-center gap-4">
                    <a href="/sops" class="btn btn-ghost">{m.common_cancel()}</a>
                    <button type="submit" class="btn btn-accent min-w-[120px]" disabled={loading}>
                        {#if loading}
                            <span class="loading loading-spinner"></span>
                        {/if}
                        {m.sops_register()}
                        <RocketIcon class="w-5 h-5"/>
                    </button>
                </div>
            </form>
        </div>
    </Card>
</div>