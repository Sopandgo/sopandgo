<script lang="ts">
    import Card from '$lib/components/Card.svelte';
    import CardPageHeading from '$lib/components/CardPageHeading.svelte';
    import ListUsers from '$lib/components/ListUsers.svelte';
    import RegisterUser from '$lib/components/RegisterUser.svelte';
    import { UsersIcon } from 'lucide-svelte';
    import * as m from '$lib/paraglide/messages.js';
    
    let { data, form } = $props();
</script>

<svelte:head>
    <title>{m.page_users()}</title>
</svelte:head>


    <Card>
        <div class="card-body">
            <CardPageHeading>
                <UsersIcon class="w-8 h-8" />
                {m.users_heading()}
            </CardPageHeading>

            <div class="flex items-center gap-2 text-base-content/70">
            <span class="text-sm font-medium">{m.users_intro()}</span> 
            </div>
        </div>
    </Card>

<div class="flex flex-col lg:flex-row gap-6 mt-6">

    <ListUsers
        items={data.users}
        currentUserId={data.user?.id ?? null}
        mailMode={data.smtp?.effective_mail_mode ?? 'manual_links'}
        {form}
    />

    <Card>
        <div class="card-body">
            <h2 class="text-lg font-semibold mb-4">{m.users_register()}</h2>
            <RegisterUser mailMode={data.smtp?.effective_mail_mode ?? 'manual_links'} defaultLocale={data.smtp?.default_locale ?? 'en'} {form} />
        </div>
    </Card>
</div>