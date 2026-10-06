<script lang="ts">
    import LoginForm from '$lib/components/auth/LoginForm.svelte';
    import LocaleSelect from '$lib/components/LocaleSelect.svelte';
    import * as m from '$lib/paraglide/messages.js';

    let { form } = $props();
</script>

<svelte:head>
    <title>{m.page_login()}</title>
</svelte:head>

<div class="mb-4 w-full max-w-sm">
    <label class="label" for="guest-locale">
        <span class="label-text">{m.locale_label()}</span>
    </label>
    <LocaleSelect id="guest-locale" mode="cookie" />
</div>

<LoginForm {form} />