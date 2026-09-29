<script lang="ts">
    import type { User, AcknowledgmentWithUser } from '$lib/sdk/types';
    import ListAcknowledgmentLog from '$lib/components/ListAcknowledgmentLog.svelte';
    import AcknowledgmentActionReader from './AcknowledgmentActionReader.svelte';
    import Alert from '../Alert.svelte';
    import AcknowledgmentActionApprove from './AcknowledgmentActionApprove.svelte';
    import AcknowledgmentActionReject from './AcknowledgmentActionReject.svelte';

    // 1. Define the shape of the form data here (or in a shared types file)
    interface SignFormResult {
        error?: string;
        inputName?: string;
        success?: boolean;
    }

    let {
        user,
        acks = [],
        form,
        status
    } = $props<{
        user: User;
        acks?: AcknowledgmentWithUser[] | null;
        form?: SignFormResult | null; // Use the interface here
        status: string | null;
    }>();

    let hasSigned = $derived(
        user && acks 
            ? acks.some((a: AcknowledgmentWithUser) => a.user_id === user.id) 
            : false
    );
</script>

<div>
    <ListAcknowledgmentLog items={acks} />

    {#if user && user.role != 'viewer' && user.role != 'auditor'}
        {#if status === 'rc' && (user.role == 'approver' || user.role === 'admin')}
            <div class="mt-8">
                <AcknowledgmentActionApprove {user} {form} />
            </div>
            <div class="mt-8">
                <AcknowledgmentActionReject {user} {form} />
            </div>
        {:else if status === 'published'}
            <div class="mt-8">
                <AcknowledgmentActionReader {user} {form} />
            </div>
        {/if}

        {#if hasSigned}
            <div class="mt-8">    
                <Alert variant="info" message="You have  acknowledged this version." />
            </div>
        {/if}
    {/if}
</div>