<script lang="ts">
    import { enhance } from '$app/forms';
    import * as m from '#lib/paraglide/messages.js';
    import type { User, UserRole } from '#lib/sdk/types.js';

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

    const ROLE_OPTIONS = $derived<{ value: UserRole; label: string; description: string }[]>([
        { value: 'admin', label: m.role_admin(), description: m.role_desc_admin() },
        { value: 'editor', label: m.role_editor(), description: m.role_desc_editor() },
        { value: 'approver', label: m.role_approver(), description: m.role_desc_approver() },
        { value: 'auditor', label: m.role_auditor(), description: m.role_desc_auditor() },
        { value: 'viewer', label: m.role_viewer(), description: m.role_desc_viewer() }
    ]);
</script>

<div class="flex items-center gap-2">
    {#if isSelf}
        <select
            class="select select-sm select-ghost cursor-not-allowed"
            disabled={true}
            value={user.role}
        >
            {#each ROLE_OPTIONS as opt (opt.value)}
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
                        alert(data?.updateRole?.error || m.error_update_role_alert());
                    }
                };
            }}
        >
            <input type="hidden" name="user_id" value={user.id} />
            
            <select
                name="role"
                class="select select-sm focus:select-primary transition-all"
                disabled={loading}
                value={user.role}
                onchange={(e) => e.currentTarget.form?.requestSubmit()}
            >
                {#each ROLE_OPTIONS as opt (opt.value)}
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
