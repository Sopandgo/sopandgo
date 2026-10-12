<script lang="ts">
    import { enhance } from '$app/forms';
    import Alert from '#lib/components/Alert.svelte';
    import ConfirmDialog from '#lib/components/ConfirmDialog.svelte';
    import * as m from '#lib/paraglide/messages.js';
    import type { User } from '#lib/sdk/types.js';
    import { ShieldOffIcon } from '@lucide/svelte';

    /*
     * Admin → Users: clears a normal user's 2FA so they set it up again. Admin
     * accounts are not offered here; they recover with recovery codes or the
     * server's recover-admin command.
     */
    let { user }: { user: User } = $props();

    let open = $state(false);
    let busy = $state(false);
</script>

<form
    class="w-full md:w-auto"
    method="POST"
    action="?/resetTwoFactor"
    use:enhance={() => {
        busy = true;
        return async ({ update }) => {
            await update();
            busy = false;
            open = false;
        };
    }}
>
    <input type="hidden" name="user_id" value={user.id} />
    <button
        type="button"
        class="btn btn-sm btn-ghost text-error h-auto min-h-9 w-full flex-nowrap gap-2 py-2 md:w-auto"
        onclick={() => (open = true)}
    >
        <ShieldOffIcon class="h-4 w-4 shrink-0" />
        <span>{m.users_reset_two_factor()}</span>
    </button>

    <ConfirmDialog title={m.users_reset_two_factor_title({ name: user.display_name })} bind:open {busy}>
        <Alert type="warning" message={m.users_reset_two_factor_warning()} />
        {#snippet confirm()}
            <button type="submit" class="btn btn-error" disabled={busy}>
                {#if busy}<span class="loading loading-spinner loading-xs"></span>{/if}
                {m.users_reset_two_factor_confirm()}
            </button>
        {/snippet}
    </ConfirmDialog>
</form>
