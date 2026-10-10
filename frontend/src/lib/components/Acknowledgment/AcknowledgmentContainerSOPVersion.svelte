<script lang="ts">
    import type { User, AcknowledgmentWithUser } from '#lib/sdk/types.js';
    import AcknowledgmentActionReader from './AcknowledgmentActionReader.svelte';
    import Alert from '../Alert.svelte';
    import Card from '../Card.svelte';
    import AcknowledgmentActionApprove from './AcknowledgmentActionApprove.svelte';
    import AcknowledgmentActionReject from './AcknowledgmentActionReject.svelte';
    import * as m from '#lib/paraglide/messages.js';
    import { getLocale } from '#lib/paraglide/runtime.js';

    /*
     * What the signed-in user can do with this version: sign it as a reader, see
     * that they already did, or review a release candidate. Renders nothing when
     * there is nothing to do. The signature log is a separate card on the page.
     */
    interface SignFormResult {
        error?: string;
        inputName?: string;
        success?: boolean;
        action?: string;
    }

    let {
        user,
        acks = [],
        form,
        status
    } = $props<{
        user: User;
        acks?: AcknowledgmentWithUser[] | null;
        form?: SignFormResult | null;
        status: string | null;
    }>();

    // Only a reader signature counts as having read it, as in training coverage;
    // author and approver acknowledgments do not.
    let ownSignature = $derived(
        (acks ?? []).find((a: AcknowledgmentWithUser) => a.user_id === user?.id && a.ack_type === 'reader')
    );

    let canAct = $derived(!!user && user.role !== 'viewer' && user.role !== 'auditor');
    let canReview = $derived(canAct && status === 'rc' && (user.role === 'approver' || user.role === 'admin'));
    let canSign = $derived(canAct && status === 'published');
</script>

{#if canReview}
    <Card title={m.ack_review_title()}>
        <div class="card-body gap-6">
            <AcknowledgmentActionApprove {user} {form} />
            <div class="border-t border-base-300 pt-6">
                <AcknowledgmentActionReject {user} {form} />
            </div>
        </div>
    </Card>
{:else if canSign && ownSignature}
    <Card title={m.ack_reader_title()}>
        <div class="card-body">
            <Alert
                type="success"
                compact
                message={m.ack_you_signed_on({ when: new Date(ownSignature.created_at).toLocaleString(getLocale()) })}
            />
        </div>
    </Card>
{:else if canSign}
    <AcknowledgmentActionReader {user} {form} />
{/if}
