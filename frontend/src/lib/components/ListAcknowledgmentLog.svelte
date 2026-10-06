<script lang="ts">
    import Avatar from './Avatar.svelte';
    import type { AcknowledgmentWithUser } from '$lib/sdk/types';
  import Card from './Card.svelte';
  import * as m from '$lib/paraglide/messages.js';
  import { getLocale } from '$lib/paraglide/runtime';

    interface Props {
        items?: AcknowledgmentWithUser[] | null;
    }

    let { items = [] }: Props = $props();
    
    let acks = $derived(items ?? []);
</script>

<Card>
    <ul class="list">
        <li class="p-4 pb-2 text-xs opacity-60 tracking-widest uppercase font-bold">
            {m.ack_log({ count: String(acks.length) })}
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
                    <div class="text-xs opacity-50 font-mono">{m.ack_user_id({ id: ack.user_id })}</div>
                    <div class="text-xs opacity-50 font-mono uppercase tracking-tighter">
                        {m.ack_signed({ when: new Date(ack.created_at).toLocaleString(getLocale()) })}
                    </div>
                </div>

                {#if ack.ack_type === "approver"}
                    <div class="badge badge-outline badge-primary">{m.ack_approver()}</div>
                {:else if ack.ack_type === "author"}
                    <div class="badge badge-outline badge-accent">{m.ack_author()}</div>
                {:else if ack.ack_type === "reader"}
                    <div class="badge badge-outline badge-secondary">{m.ack_reader()}</div>
                {:else}
                    <div class="badge badge-outline badge-warning">{ack.ack_type}</div>
                {/if}
            </li>
        {:else}
            <li class="p-12 text-center">
                <div class="text-sm opacity-40">{m.ack_empty()}</div>
            </li>
        {/each}
    </ul>
</Card>