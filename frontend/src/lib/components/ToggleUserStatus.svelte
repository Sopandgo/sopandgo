<script lang="ts">
    import { enhance } from '$app/forms';
    import type { User } from '$lib/sdk/types';

    let { user, currentUserId } = $props<{
        user: User;
        currentUserId: string | null;
    }>();

    let loading = $state(false);
    const isSelf = $derived(currentUserId && user.id === currentUserId);
</script>



<div class="flex items-center gap-2">
    {#if isSelf}
        <input
            type="checkbox"
            class="toggle"
            checked={user.is_active}
            disabled={true}
        />
        <span class="text-xs opacity-60">
            {user.is_active ? 'active' : 'disabled'}
        </span>
    {:else}
        <form 
            method="POST" 
            action="?/updateStatus" 
            use:enhance={() => {
                loading = true;
                return async ({ update, result }) => {
                    await update();
                    loading = false;

                    if (result.type === 'failure') {
                        const data = result.data as { updateStatus?: { error?: string } };
                        alert(data?.updateStatus?.error || 'Failed to update status');
                    }
                };
            }}
        >
            <input type="hidden" name="user_id" value={user.id} />
            <input type="hidden" name="active" value={(!user.is_active).toString()} />
            
            <button 
                type="submit" 
                class="flex items-center gap-2 group" 
                disabled={loading}
            >
                <input
                    type="checkbox"
                    class="toggle toggle-secondary pointer-events-none"
                    checked={user.is_active}
                    disabled={loading}
                />
                <span class="text-xs opacity-60 group-hover:opacity-100 transition-opacity">
                    {user.is_active ? 'active' : 'disabled'}
                </span>
            </button>
        </form>
    {/if}
</div>