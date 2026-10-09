<script lang="ts">
    import Avatar from './Avatar.svelte';
    import Card from './Card.svelte';
    import ListRow from './ListRow.svelte';
    import type { AcknowledgmentWithUser } from '$lib/sdk/types';
    import * as m from '$lib/paraglide/messages.js';
    import { getLocale } from '$lib/paraglide/runtime';

    interface Props {
        items?: AcknowledgmentWithUser[] | null;
    }

    let { items = [] }: Props = $props();

    let acks = $derived(items ?? []);

    // Roles are information, not state: one neutral outline badge for every type.
    const ackLabels: Record<string, () => string> = {
        approver: m.ack_approver,
        author: m.ack_author,
        reader: m.ack_reader
    };
</script>

<Card title={m.ack_log({ count: String(acks.length) })}>
    <ul class="list">
        {#each acks as ack, i (i)}
            <ListRow title={ack.user.display_name}>
                {#snippet leading()}<Avatar displayName={ack.user.display_name} />{/snippet}
                {#snippet meta()}
                    <span class="font-mono" title={ack.user_id}>{ack.user_id.slice(0, 8)}</span>
                    · {m.ack_signed({ when: new Date(ack.created_at).toLocaleString(getLocale()) })}
                {/snippet}
                {#snippet trailing()}
                    <span class="badge badge-outline badge-sm">{ackLabels[ack.ack_type]?.() ?? ack.ack_type}</span>
                {/snippet}
            </ListRow>
        {:else}
            <li class="p-6 text-sm text-base-content/70">{m.ack_empty()}</li>
        {/each}
    </ul>
</Card>
