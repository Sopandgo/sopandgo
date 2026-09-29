<script lang="ts">
    import Card from '$lib/components/Card.svelte';
    import CardPageHeading from '$lib/components/CardPageHeading.svelte';
    import ListUsers from '$lib/components/ListUsers.svelte';
    import RegisterUser from '$lib/components/RegisterUser.svelte';
    import { UsersIcon } from 'lucide-svelte';
    
    let { data, form } = $props();
</script>


    <Card>
        <div class="card-body">
            <CardPageHeading>
                <UsersIcon class="w-8 h-8" />
                User Management
            </CardPageHeading>

            <div class="flex items-center gap-2 text-base-content/70">
            <span class="text-sm font-medium">Administrate registered users, assign roles, and manage system access.</span> 
            </div>
        </div>
    </Card>

<div class="flex flex-col lg:flex-row gap-6 mt-6">

    <ListUsers
        items={data.users}
        currentUserId={data.user?.id ?? null}
        mailMode={data.smtp?.mail_mode ?? 'smtp'}
        {form}
    />

    <Card>
        <div class="card-body">
            <h2 class="card-title text-xs opacity-60 tracking-widest uppercase font-bold mb-4">Register New User</h2>
            <RegisterUser mailMode={data.smtp?.mail_mode ?? 'smtp'} {form} />
        </div>
    </Card>
</div>