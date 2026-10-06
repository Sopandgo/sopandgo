<script lang="ts">
    import Card from '$lib/components/Card.svelte';
    import CardPageHeading from '$lib/components/CardPageHeading.svelte';
    import ListUserSignatures from '$lib/components/ListUserSignatures.svelte';
    import { resolve } from '$app/paths';
    import * as m from '$lib/paraglide/messages.js';
    import { getLocale } from '$lib/paraglide/runtime';
    import type { UserRole } from '$lib/sdk/types';
    import {
        CalendarClockIcon,
        IdCardIcon,
        MailIcon,
        SettingsIcon,
        StampIcon,
        UserIcon
    } from 'lucide-svelte';

    let { data } = $props();

    const user = $derived(data.user);

    function roleLabel(role: UserRole) {
        switch (role) {
            case 'admin':
                return m.role_admin();
            case 'editor':
                return m.role_editor();
            case 'approver':
                return m.role_approver();
            case 'auditor':
                return m.role_auditor();
            case 'viewer':
                return m.role_viewer();
            default:
                return role;
        }
    }
</script>

<svelte:head>
    <title>{m.page_profile()}</title>
</svelte:head>

<Card>
    <div class="card-body">
        <div class="flex flex-wrap items-start justify-between gap-3">
            <CardPageHeading>
                <UserIcon class="w-8 h-8" />
                {m.profile_heading()}
            </CardPageHeading>
            <a href={resolve('/profile/settings')} class="btn btn-ghost">
                <SettingsIcon class="w-4 h-4" />
                {m.nav_account_settings()}
            </a>
        </div>
        <p class="text-sm font-medium text-base-content/70">{m.profile_intro()}</p>
    </div>
</Card>

<div class="mt-6 flex flex-col gap-6">
    <Card>
        <ul class="list">
            <li class="border-b border-base-200/50 p-4 pb-2 text-xs font-bold tracking-widest uppercase opacity-60">
                {m.profile_information()}
            </li>

            <li class="list-row items-center">
                <div>
                    <UserIcon size={24} class="p-1 opacity-70" />
                </div>
                <div class="flex-1">
                    <div class="text-sm font-medium">{user.display_name}</div>
                </div>
            </li>

            <li class="list-row items-center">
                <div>
                    <MailIcon size={24} class="p-1 opacity-70" />
                </div>
                <div class="flex-1">
                    <div class="text-sm font-medium">{user.email}</div>
                </div>
            </li>

            <li class="list-row items-center">
                <div>
                    <StampIcon size={24} class="p-1 opacity-70" />
                </div>
                <div class="flex-1">
                    <div class="text-sm font-medium">{m.profile_role({ role: roleLabel(user.role) })}</div>
                </div>
            </li>

            <li class="list-row items-center">
                <div>
                    <CalendarClockIcon size={24} class="p-1 opacity-70" />
                </div>
                <div class="flex-1">
                    <div class="text-sm font-medium">
                        {m.common_created({ when: new Date(user.created_at).toLocaleString(getLocale()) })}
                    </div>
                    <div class="font-mono text-xs italic opacity-40">
                        {m.profile_timestamp()}
                    </div>
                </div>
            </li>

            <li class="list-row items-center">
                <div>
                    <IdCardIcon size={24} class="p-1 opacity-70" />
                </div>
                <div class="flex-1">
                    <div class="text-sm font-medium">{m.common_id({ id: user.id })}</div>
                </div>
            </li>
        </ul>
    </Card>

    <ListUserSignatures status={data.signatureStatus || []} />
</div>
