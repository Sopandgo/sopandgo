<script lang="ts">
    import { enhance } from '$app/forms';
    import type { User, UserRole } from '$lib/sdk/types';

    // Define the interface locally to avoid $types dependency
    interface FormResult {
        updateRole?: {
            ok?: boolean;
            error?: string;
        };
    }

    // FIX: Added 'form' to the props definition
    let { user, currentUserId, form = null } = $props<{
        user: User;
        currentUserId: string | null;
        form?: FormResult | null;
    }>();

    let loading = $state(false);
    const isSelf = $derived(currentUserId && user.id === currentUserId);

    const ROLE_OPTIONS: { value: UserRole; label: string; description: string }[] = [
        { value: 'admin', label: 'Admin', description: 'Full system access' },
        { value: 'editor', label: 'Editor', description: 'Can create and edit SOPs' },
        { value: 'approver', label: 'Approver', description: 'Can sign off on SOP versions' },
        { value: 'auditor', label: 'Auditor', description: 'SOP read access (audit log UI is admin-only today)' },
        { value: 'viewer', label: 'Viewer', description: 'Read-only access' }
    ];
</script>

<div class="flex items-center gap-2">
    {#if isSelf}
        <select
            class="select select-sm select-ghost cursor-not-allowed"
            disabled={true}
            value={user.role}
        >
            {#each ROLE_OPTIONS as opt}
                <option value={opt.value}>{opt.label}</option>
            {/each}
        </select>
    {:else}
        <form 
            method="POST" 
            action="?/updateRole" 
            use:enhance={() => {
                loading = true;
                return async ({ update, result }) => {
                    await update();
                    loading = false;
                    if (result.type === 'failure') {
                        const data = result.data as FormResult;
                        alert(data?.updateRole?.error || 'Failed to update role');
                    }
                };
            }}
        >
            <input type="hidden" name="user_id" value={user.id} />
            
            <select
                name="role"
                class="select select-sm select-bordered focus:select-primary transition-all"
                disabled={loading}
                value={user.role}
                onchange={(e) => e.currentTarget.form?.requestSubmit()}
            >
                {#each ROLE_OPTIONS as opt}
                    <option value={opt.value} title={opt.description}>
                        {opt.label}
                    </option>
                {/each}
            </select>
            
            {#if loading}
                <span class="loading loading-spinner loading-xs text-primary ml-1"></span>
            {/if}
        </form>
    {/if}
</div>