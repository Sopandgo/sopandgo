<script lang="ts">
    import LoginForm from '#lib/components/auth/LoginForm.svelte';
    import TwoFactorForm from '#lib/components/auth/TwoFactorForm.svelte';
    import * as m from '#lib/paraglide/messages.js';

    let { data, form } = $props();

    // An action result wins (it just changed the step); otherwise the cookie
    // read in `load` keeps the second step across reloads.
    const twoFactor = $derived(form && 'twoFactor' in form ? form.twoFactor : data.twoFactor);
</script>

<svelte:head>
    <title>{m.page_login()}</title>
</svelte:head>

{#if twoFactor}
    <TwoFactorForm methods={twoFactor.methods} {form} />
{:else}
    <LoginForm {form} />
{/if}
