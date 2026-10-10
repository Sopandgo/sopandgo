<script lang="ts">
    import Avatar from './Avatar.svelte';
    import Card from './Card.svelte';
    import ListRow from './ListRow.svelte';
    import type { AcknowledgmentWithUser } from '$lib/sdk/types';
    import * as m from '$lib/paraglide/messages.js';
    import { getLocale } from '$lib/paraglide/runtime';

    interface Props {
        items?: AcknowledgmentWithUser[] | null;
        /** Rows shown before "Show all"; every row when unset. */
        limit?: number;
    }

    let { items = [], limit }: Props = $props();

    const uid = $props.id();

    let acks = $derived(items ?? []);
    let expanded = $state(false);
    let truncated = $derived(limit !== undefined && acks.length > limit && !expanded);
    let shown = $derived(truncated ? acks.slice(0, limit) : acks);

    // Roles are information, not state: one neutral outline badge for every type.
    const ackLabels: Record<string, () => string> = {
        approver: m.ack_approver,
        author: m.ack_author,
        reader: m.ack_reader
    };
</script>

<Card title={m.ack_log({ count: String(acks.length) })}>
    <ul class="list" id="{uid}-list">
        {#each shown as ack, i (i)}
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
    {#if limit !== undefined && acks.length > limit}
        <div class="border-t border-base-300 p-2">
            <button
                type="button"
                class="btn btn-ghost btn-sm w-full"
                aria-expanded={expanded}
                aria-controls="{uid}-list"
                onclick={() => (expanded = !expanded)}
            >
                {expanded ? m.ack_show_fewer() : m.ack_show_all({ count: String(acks.length) })}
            </button>
        </div>
    {/if}
</Card>
