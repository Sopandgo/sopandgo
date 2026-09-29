<script lang="ts">
    import Card from '$lib/components/Card.svelte';
    import CardPageHeading from '$lib/components/CardPageHeading.svelte';
    import ListSessions from '$lib/components/ListSessions.svelte';
    import PanicRevokeAll from '$lib/components/PanicRevokeAll.svelte';
    import { UsersIcon } from 'lucide-svelte';

    let { data, form } = $props();

    let sessions = $derived(data.sessions ?? []);
</script>

<div class="flex flex-col gap-6">
    <Card>
        <div class="card-body">
            <CardPageHeading>
                <UsersIcon class="w-8 h-8" />
                Session Management
            </CardPageHeading>

            <div class="flex items-center gap-2 text-base-content/70">
            <span class="text-sm font-medium">Monitor active users and perform emergency session revocations if security is compromised.</span> 
            </div>
        </div>
    </Card>



    <PanicRevokeAll {form} />

    <ListSessions items={sessions} {form} />
</div>