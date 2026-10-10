<script lang="ts">
    import Avatar from '#lib/components/Avatar.svelte';
    import Card from '#lib/components/Card.svelte';
    import CardPageHeading from '#lib/components/CardPageHeading.svelte';
    import ListRow from '#lib/components/ListRow.svelte';
    import ListUserSignatures from '#lib/components/ListUserSignatures.svelte';
    import { resolve } from '$app/paths';
    import * as m from '#lib/paraglide/messages.js';
    import { getLocale } from '#lib/paraglide/runtime.js';
    import type { UserRole } from '#lib/sdk/types.js';
    import {
        CalendarClockIcon,
        IdCardIcon,
        MailIcon,
        SettingsIcon,
        StampIcon,
        UserIcon
    } from '@lucide/svelte';

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
        <div
            class="flex flex-wrap items-start justify-between gap-3"
        >
            <CardPageHeading><UserIcon class="w-8 h-8" />{m.profile_heading()}</CardPageHeading>

            <a href={resolve('profile/settings')} class="btn">
                <SettingsIcon class="size-4" />
                {m.nav_account_settings()}
            </a>
        </div>
        <p class="text-sm text-base-content/70">{m.profile_intro()}</p>
    </div>
</Card>

<div class="mt-6 flex flex-col gap-6">
    <Card title={m.profile_information()}>
        <ul class="list">
            <ListRow title={user.display_name}>
                {#snippet leading()}
                    <Avatar
                        displayName={user.display_name}
                        size="lg"
                        userId={user.id}
                        hasAvatar={user.has_avatar}
                        avatarContentHash={user.avatar_content_hash}
                    />
                {/snippet}
            </ListRow>
            <ListRow title={user.email} icon={MailIcon} />
            <ListRow title={m.common_role()} icon={StampIcon}>
                {#snippet trailing()}<span class="badge badge-outline">{roleLabel(user.role)}</span>{/snippet}
            </ListRow>
            <ListRow
                title={m.common_created({ when: new Date(user.created_at).toLocaleString(getLocale()) })}
                meta={m.profile_timestamp()}
                icon={CalendarClockIcon}
            />
            <ListRow title={m.common_id_label()} icon={IdCardIcon}>
                {#snippet meta()}<span class="font-mono">{user.id}</span>{/snippet}
            </ListRow>
        </ul>
    </Card>

    <ListUserSignatures status={data.signatureStatus || []} />
</div>
