<script lang="ts">
    import LoginForm from '$lib/components/auth/LoginForm.svelte';

    let { form } = $props();
</script>

<LoginForm {form} />