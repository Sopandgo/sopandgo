<script lang="ts">
    import Avatar from './Avatar.svelte';
    import type { AcknowledgmentWithUser } from '$lib/sdk/types';
  import Card from './Card.svelte';

    interface Props {
        items?: AcknowledgmentWithUser[] | null;
    }

    let { items = [] }: Props = $props();
    
    let acks = $derived(items ?? []);
</script>

<Card>
    <ul class="list">
        <li class="p-4 pb-2 text-xs opacity-60 tracking-widest uppercase font-bold">
            Acknowledgment Log ({acks.length})
        </li>

        {#each acks as ack, i (i)}
            <li class="list-row items-center hover:bg-base-200/50 transition-colors">
                {#if ack.ack_type === "approver"}
                    <Avatar color="primary" displayName={ack.user.display_name}/>
                {:else if ack.ack_type === "author"}
                    <Avatar color="accent" displayName={ack.user.display_name}/>
                {:else if ack.ack_type === "reader"}
                    <Avatar color="secondary" displayName={ack.user.display_name}/>
                {:else}
                    <Avatar color="warning" displayName={ack.user.display_name}/>
                {/if}

                <div class="flex-1">
                    <div class="font-bold text-sm lg:text-base">{ack.user.display_name}</div>
                    <div class="text-xs opacity-50 font-mono">USER ID: {ack.user_id}</div>
                    <div class="text-xs opacity-50 font-mono uppercase tracking-tighter">
                        Signed: {new Date(ack.created_at).toLocaleString()}
                    </div>
                </div>

                {#if ack.ack_type === "approver"}
                    <div class="badge badge-outline badge-primary">approver</div>
                {:else if ack.ack_type === "author"}
                    <div class="badge badge-outline badge-accent">author</div>
                {:else if ack.ack_type === "reader"}
                    <div class="badge badge-outline badge-secondary">reader</div>
                {:else}
                    <div class="badge badge-outline badge-warning">{ack.ack_type}</div>
                {/if}
            </li>
        {:else}
            <li class="p-12 text-center">
                <div class="text-sm opacity-40">No acknowledgments yet.</div>
            </li>
        {/each}
    </ul>
</Card>