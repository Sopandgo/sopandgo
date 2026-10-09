<script lang="ts">
    import Alert from '$lib/components/Alert.svelte';
    import Card from '$lib/components/Card.svelte';
    import { enhance } from '$app/forms';
    import * as m from '$lib/paraglide/messages.js';

    let { form } = $props();
    let loading = $state(false);
</script>

<Card class="mx-auto"> 
    <div class="card-body"> {#if form?.message}
            <Alert variant="error" message={form.message} class="mb-4 shadow-sm" />
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
</Card>