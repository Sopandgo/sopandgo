<script lang="ts">
    import Card from '$lib/components/Card.svelte';
    import CardPageHeading from '$lib/components/CardPageHeading.svelte';
    import ListSessions from '$lib/components/ListSessions.svelte';
    import PanicRevokeAll from '$lib/components/PanicRevokeAll.svelte';
    import { UsersIcon } from '@lucide/svelte';
    import * as m from '$lib/paraglide/messages.js';

    let { data, form } = $props();

    let sessions = $derived(data.sessions ?? []);
</script>

<svelte:head>
    <title>{m.page_sessions()}</title>
</svelte:head>

<div class="flex flex-col gap-6">
    <Card>
        <div class="card-body">
            <CardPageHeading>
                <UsersIcon class="w-8 h-8" />
                {m.sessions_heading()}
            </CardPageHeading>

            <div class="flex items-center gap-2 text-base-content/70">
            <span class="text-sm font-medium">{m.sessions_intro()}</span> 
            </div>
        </div>
    </Card>



    <PanicRevokeAll {form} />

    <ListSessions items={sessions} {form} />
</div>