<script lang="ts">
    import { enhance } from '$app/forms';
    import * as m from '$lib/paraglide/messages.js';

    let { form } = $props();
    let loading = $state(false);
</script>

<div class="card card-border bg-base-100 shadow-md mx-auto"> 
    <div class="card-body"> {#if form?.message}
            <div role="alert" class="alert alert-error mb-4 shadow-sm">
                <svg xmlns="http://www.w3.org/2000/svg" class="stroke-current shrink-0 h-6 w-6" fill="none" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M10 14l2-2m0 0l2-2m-2 2l-2-2m2 2l2 2m7-2a9 9 0 11-18 0 9 9 0 0118 0z" /></svg>
                <span>{form.message}</span>
            </div>
        {/if}

        <form 
            method="POST" 
            use:enhance={() => {
                loading = true;
                return async ({ update }) => {
                    await update();
                    loading = false;
                };
            }}
        >
            <fieldset class="fieldset w-full">
                <legend class="fieldset-legend text-lg font-bold">{m.login_heading()}</legend>

                <div class="w-full">
                    <label class="label mb-1" for="email">{m.common_email()}</label>
                    <input 
                        id="email"
                        name="email" 
                        type="text" 
                        class="input w-full {form?.message ? 'input-error' : ''}" 
                        placeholder="mail@example.com" 
                        value={form?.email ?? ''}
                        required 
                    />
                </div>

                <div class="w-full mt-2">
                    <label class="label mb-1" for="password">{m.common_password()}</label>
                    <input 
                        id="password"
                        name="password" 
                        type="password" 
                        class="input w-full {form?.message ? 'input-error' : ''}" 
                        placeholder="••••••••" 
                        required 
                    />
                </div>

                <button type="submit" class="btn btn-primary mt-6 w-full" disabled={loading}>
                    {#if loading}
                        <span class="loading loading-spinner"></span>
                    {/if}
                    {m.login_submit()}
                </button>
            </fieldset>
        </form>
    </div>
</div>