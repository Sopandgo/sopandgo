<script>
    import * as m from '$lib/paraglide/messages.js';

    let { status } = $props();
</script>

{#if status == "draft"}
    <div class="badge badge-secondary">{m.status_draft()}</div>
{:else if status == "rc"}
    <div class="badge badge-warning">{m.status_rc()}</div>
{:else if status == "published"}
    <div class="badge badge-primary">{m.status_published()}</div>
{:else if status == "rejected"}
    <div class="badge badge-error">{m.status_rejected()}</div>
{:else if status == "superseded"}
    <div class="badge badge-ghost">{m.status_superseded()}</div>
{:else}
    <div class="badge badge-ghost">{m.status_unclear()}</div>
{/if}
