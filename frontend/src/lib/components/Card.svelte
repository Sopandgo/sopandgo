<script lang="ts">
  let { children } = $props();
</script>

<div class="card card-border bg-base-100 shadow-md">
  {@render children()}
</div>