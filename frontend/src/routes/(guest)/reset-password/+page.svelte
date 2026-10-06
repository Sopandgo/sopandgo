<script lang="ts">
    import ResetPasswordForm from '$lib/components/ResetPasswordForm.svelte';
    import LocaleSelect from '$lib/components/LocaleSelect.svelte';
    import * as m from '$lib/paraglide/messages.js';

    let { data, form } = $props();
</script>

<svelte:head>
    <title>{m.page_reset_password()}</title>
</svelte:head>

<div class="mb-4 w-full max-w-sm">
    <label class="label" for="reset-locale">
        <span class="label-text">{m.locale_label()}</span>
    </label>
    <LocaleSelect id="reset-locale" mode="cookie" />
</div>

<ResetPasswordForm {form} token={data.token} />