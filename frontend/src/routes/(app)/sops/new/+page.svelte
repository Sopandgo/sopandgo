<script lang="ts">
    import { enhance } from '$app/forms';
    import { HouseIcon, NotebookIcon, NotebookPenIcon, PlusIcon, RocketIcon } from 'lucide-svelte';
    import type { ActionData } from './$types';
    import Breadcrumbs from '$lib/components/Breadcrumbs.svelte';
  import Card from '$lib/components/Card.svelte';
  import CardPageHeading from '$lib/components/CardPageHeading.svelte';

    let { form } = $props<{ form: ActionData }>();
    
    let loading = $state(false);
</script>

<div class="flex flex-col gap-6">
    <Breadcrumbs items={[
        { label: 'Dashboard', href: '/sops', icon: HouseIcon},
        { label: 'SOPs', href: '/sops', icon: NotebookIcon},
        { label: 'New SOP', icon: PlusIcon},
    ]}/>

    <Card>
        <div class="card-body">
            <CardPageHeading color="accent">
                <NotebookPenIcon class="w-8 h-8" />
                Standard Operating Procedures
            </CardPageHeading>

            <div class="flex items-center gap-2 text-base-content/70 mb-4">
                <span class="text-sm font-medium">
                    Register a new Standard Operating Procedure container.
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
                        <span class="label-text font-medium">SOP Title</span>
                    </label>
                    <input 
                        id="title"
                        name="title" 
                        type="text" 
                        placeholder="e.g., Server Maintenance Protocol" 
                        class="input input-bordered w-full {form?.message ? 'input-error' : ''}" 
                        value={form?.title ?? ''}
                        required 
                    />
                </div>

                <div class="card-actions justify-end mt-4 flex items-center gap-4">
                    <a href="/sops" class="btn btn-ghost">Cancel</a>
                    <button type="submit" class="btn btn-accent min-w-[120px]" disabled={loading}>
                        {#if loading}
                            <span class="loading loading-spinner"></span>
                        {/if}
                        Register SOP
                        <RocketIcon class="w-5 h-5"/>
                    </button>
                </div>
            </form>
        </div>
    </Card>
</div>