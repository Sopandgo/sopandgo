<script lang="ts">
    import { enhance } from '$app/forms';
    import * as m from '$lib/paraglide/messages.js';
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
            class="toggle toggle-success"
            checked={user.is_active}
            disabled={true}
        />
        <span class="text-xs text-base-content/70">
            {user.is_active ? m.common_active() : m.common_disabled()}
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
                        alert(data?.updateStatus?.error || m.error_update_status_alert());
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
                    class="toggle toggle-success pointer-events-none"
                    checked={user.is_active}
                    disabled={loading}
                />
                <span class="text-xs text-base-content/70 group-hover:opacity-100 transition-opacity">
                    {user.is_active ? m.common_active() : m.common_disabled()}
                </span>
            </button>
        </form>
    {/if}
</div>