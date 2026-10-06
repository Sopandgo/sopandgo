<script lang="ts">
    import * as m from '$lib/paraglide/messages.js';

    // Receive 'form' (action result) and 'token' (from URL)
    let { form, token } = $props(); 
</script>

<div class="card w-full max-w-sm shadow-2xl bg-base-100">
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
            <div role="alert" class="alert alert-error mt-4 py-2 text-sm">
                <svg xmlns="http://www.w3.org/2000/svg" class="stroke-current shrink-0 h-6 w-6" fill="none" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M10 14l2-2m0 0l2-2m-2 2l-2-2m2 2l2 2m7-2a9 9 0 11-18 0 9 9 0 0118 0z" /></svg>
                <span>{form.message}</span>
            </div>
        {/if}

        <div class="form-control mt-6">
            <button class="btn btn-primary">{m.reset_submit()}</button>
        </div>
    </form>
</div>