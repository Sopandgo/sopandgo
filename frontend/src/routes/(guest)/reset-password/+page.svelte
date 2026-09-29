<script lang="ts">
    import ResetPasswordForm from '$lib/components/ResetPasswordForm.svelte';

    let { data, form } = $props();
</script>

<ResetPasswordForm {form} token={data.token} />