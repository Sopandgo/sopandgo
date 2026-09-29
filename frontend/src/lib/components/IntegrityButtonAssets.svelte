<script lang="ts">
  import { 
    ShieldCheckIcon, 
    ShieldQuestionMarkIcon, 
    ShieldXIcon,
    LoaderCircleIcon
  } from "lucide-svelte";

  interface Props {
    assetId: string;
    size?: 'small' | 'normal';
  }

  let { assetId, size = 'normal' }: Props = $props();

  type Status = 'idle' | 'loading' | 'success' | 'error';
  let status = $state<Status>('idle');

  async function handleVerify() {
    if (status === 'loading') return;
    status = 'loading';

    try {
      const res = await fetch('/api/verify/asset', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ assetId })
      });

      if (!res.ok) throw new Error();

      const data = await res.json();
      status = data.success ? 'success' : 'error';
    } catch {
      status = 'error';
    }
  }
</script>

<div class="inline-flex items-center">
  {#if status === 'loading'}
    <div class="flex items-center gap-1 px-3 py-1 text-xs opacity-50 italic">
      <LoaderCircleIcon class="size-4 animate-spin" />
      {#if size !== 'small'}Checking...{/if}
    </div>

  {:else if status === 'success'}
    <div class="badge badge-success badge-outline gap-1 h-7">
      <ShieldCheckIcon class="size-3" />
      {#if size !== 'small'}Verified{/if}
    </div>

  {:else if status === 'error'}
    <div class="badge badge-error badge-outline gap-1 h-7">
      <ShieldXIcon class="size-3" />
      {#if size !== 'small'}Corrupt / Missing{/if}
    </div>

  {:else}
    <button 
      class="btn btn-ghost h-7 gap-1" 
      onclick={handleVerify}
    >
      <ShieldQuestionMarkIcon class="size-3" />
      {#if size !== 'small'}Verify File{/if}
    </button>
  {/if}
</div>