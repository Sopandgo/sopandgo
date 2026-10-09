<script lang="ts">
    import Alert from '$lib/components/Alert.svelte';
    import Card from '$lib/components/Card.svelte';
    import * as m from '$lib/paraglide/messages.js';

    // Receive 'form' (action result) and 'token' (from URL)
    let { form, token } = $props(); 
</script>

<Card class="w-full max-w-sm">
    <form method="POST" class="card-body">
        <h2 class="card-title justify-center mb-4">{m.reset_heading()}</h2>

        <input type="hidden" name="token" value={token} />

        <div class="form-control">
            <label class="label" for="password">
                <span class="label-text">{m.reset_new_password()}</span>
            </label>
            <input 
                type="password" 
                name="password" 
                id="password"
                placeholder="••••••••" 
                class="input input-bordered" 
                class:input-error={form?.message}
                required 
            />
        </div>

        {#if form?.message}
            <Alert variant="error" message={form.message} class="mt-4 py-2 text-sm" />
        {/if}

        <div class="form-control mt-6">
            <button class="btn btn-primary">{m.reset_submit()}</button>
        </div>
    </form>
</Card>