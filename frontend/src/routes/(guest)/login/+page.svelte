<script lang="ts">
    import LoginForm from '#lib/components/auth/LoginForm.svelte';
    import * as m from '#lib/paraglide/messages.js';

    let { form } = $props();
</script>

<svelte:head>
    <title>{m.page_login()}</title>
</svelte:head>

<LoginForm {form} />
