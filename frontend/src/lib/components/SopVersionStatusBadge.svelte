<script>
    let { status } = $props();
</script>

{#if status == "draft"}
    <div class="badge badge-secondary">draft</div>
{:else if status == "rc"}
    <div class="badge badge-warning">release candidate</div>
{:else if status == "published"}
    <div class="badge badge-primary">published</div>
{:else if status == "rejected"}
    <div class="badge badge-error">rejected</div>
{:else if status == "superseded"}
    <div class="badge badge-ghost">superseded</div>
{:else}
    <div class="badge badge-ghost">status unclear</div>
{/if}