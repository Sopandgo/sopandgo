<script lang="ts">
    import Alert from '#lib/components/Alert.svelte';
    import Card from '#lib/components/Card.svelte';
    import { enhance } from '$app/forms';
    import * as m from '#lib/paraglide/messages.js';
    import type { TwoFactorMethod } from '#lib/sdk/types.js';

    /*
     * Second sign-in step. The challenge token stays in an httpOnly cookie;
     * this form only sends the method and the code.
     */
    interface Props {
        methods: TwoFactorMethod[];
        form?: { message?: string; method?: TwoFactorMethod; emailSent?: boolean } | null;
    }

    let { methods, form = null }: Props = $props();

    // An app code is the default when the user has one; under the email policy
    // without an app, email is the only method and the code is already on its way.
    // svelte-ignore state_referenced_locally
    let method = $state<TwoFactorMethod>(
        form?.method && methods.includes(form.method) ? form.method : (methods[0] ?? 'totp')
    );
    let loading = $state(false);
    let sending = $state(false);

    const labels: Record<TwoFactorMethod, () => string> = {
        totp: () => m.two_factor_method_totp(),
        email: () => m.two_factor_method_email(),
        recovery_code: () => m.two_factor_method_recovery()
    };
    const help: Record<TwoFactorMethod, () => string> = {
        totp: () => m.two_factor_login_help_totp(),
        email: () => m.two_factor_login_help_email(),
        recovery_code: () => m.two_factor_login_help_recovery()
    };
</script>

<Card class="mx-auto">
    <div class="card-body gap-4">
        <h1 class="text-lg font-semibold">{m.two_factor_login_heading()}</h1>

        {#if form?.message}
            <Alert type="error" message={form.message} />
        {:else if form?.emailSent && method === 'email'}
            <Alert type="success" message={m.two_factor_email_sent()} />
        {/if}

        {#if methods.length > 1}
            <div class="join w-full" role="group" aria-label={m.two_factor_method_label()}>
                {#each methods as option (option)}
                    <button
                        type="button"
                        class="btn join-item btn-sm flex-1 {method === option ? 'bg-primary/10 font-semibold' : ''}"
                        aria-pressed={method === option}
                        onclick={() => (method = option)}
                    >
                        {labels[option]()}
                    </button>
                {/each}
            </div>
        {/if}

        <p class="text-sm text-base-content/70">{help[method]()}</p>

        {#if method === 'email'}
            <form
                method="POST"
                action="?/sendEmail"
                use:enhance={() => {
                    sending = true;
                    return async ({ update }) => {
                        await update();
                        sending = false;
                    };
                }}
            >
                <button type="submit" class="btn w-full" disabled={sending}>
                    {#if sending}
                        <span class="loading loading-spinner loading-xs"></span>
                    {/if}
                    {m.two_factor_email_send()}
                </button>
            </form>
        {/if}

        <form
            method="POST"
            action="?/verify"
            use:enhance={() => {
                loading = true;
                return async ({ update }) => {
                    await update();
                    loading = false;
                };
            }}
        >
            <input type="hidden" name="method" value={method} />
            <label class="label mb-1" for="two-factor-code">{m.two_factor_code_label()}</label>
            {#key method}
                <input
                    id="two-factor-code"
                    name="code"
                    type="text"
                    class="input w-full font-mono {form?.message ? 'input-error' : ''}"
                    inputmode={method === 'recovery_code' ? 'text' : 'numeric'}
                    autocomplete="one-time-code"
                    autocapitalize="characters"
                    spellcheck="false"
                    maxlength={method === 'recovery_code' ? 16 : 8}
                    placeholder={method === 'recovery_code' ? 'XXXXX-XXXXX' : '123456'}
                    aria-invalid={form?.message ? 'true' : undefined}
                    required
                />
            {/key}
            <button type="submit" class="btn btn-primary mt-4 w-full" disabled={loading}>
                {#if loading}
                    <span class="loading loading-spinner loading-xs"></span>
                {/if}
                {m.two_factor_verify()}
            </button>
        </form>

        <form method="POST" action="?/cancel" use:enhance>
            <button type="submit" class="btn btn-ghost btn-sm w-full">{m.two_factor_login_back()}</button>
        </form>
    </div>
</Card>
