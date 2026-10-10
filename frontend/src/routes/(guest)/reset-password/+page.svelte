<script lang="ts">
    import ResetPasswordForm from '#lib/components/ResetPasswordForm.svelte';
    import * as m from '#lib/paraglide/messages.js';

    let { data, form } = $props();
</script>

<svelte:head>
    <title>{m.page_reset_password()}</title>
</svelte:head>

<ResetPasswordForm {form} token={data.token} />
