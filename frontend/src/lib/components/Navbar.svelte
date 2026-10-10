<script lang="ts">
    import { enhance } from '$app/forms';
    import { resolve } from '$app/paths';
    import { page } from '$app/state';
    import { breadcrumbsFor } from '$lib/breadcrumbs';
    import Breadcrumbs from '$lib/components/Breadcrumbs.svelte';
    import { CircleUserIcon, LogOutIcon, PanelLeftCloseIcon, PanelLeftOpenIcon, SettingsIcon } from 'lucide-svelte';
    import AccountThemeSwitch from '$lib/components/AccountThemeSwitch.svelte';
    import Avatar from '$lib/components/Avatar.svelte';
    import GuestLocaleMenu from '$lib/components/GuestLocaleMenu.svelte';
    import GuestThemeToggle from '$lib/components/GuestThemeToggle.svelte';
    import * as m from '$lib/paraglide/messages.js';
    import type { UserRole } from '$lib/sdk/types';

    let { user, rail = false } = $props();

    const roleLabels: Record<UserRole, () => string> = {
        admin: m.role_admin,
        auditor: m.role_auditor,
        editor: m.role_editor,
        approver: m.role_approver,
        viewer: m.role_viewer
    };

    const crumbs = $derived(rail ? breadcrumbsFor(page.route.id, page.data) : []);
</script>

<div
    class="navbar sticky top-0 z-30 border-b border-base-300 bg-base-100"
    class:navbar-guest={!user}
    class:navbar-app={!!user}
>
    <div class="navbar-start gap-2">
        {#if rail}
            <label
                for="app-drawer"
                aria-label={m.nav_menu()}
                class="btn btn-square btn-ghost drawer-button shrink-0"
            >
                <span class="rail-when-closed inline-flex">
                    <PanelLeftOpenIcon size={18} aria-hidden="true" />
                </span>
                <span class="rail-when-open inline-flex">
                    <PanelLeftCloseIcon size={18} aria-hidden="true" />
                </span>
            </label>
            <Breadcrumbs items={crumbs} />
        {:else if !user}
            {@render logo()}
        {/if}
    </div>

    <div class="navbar-end">
        <div class="flex items-center gap-2">
            {#if user}
                {@render roleChip(user.role)}
                <div class="dropdown dropdown-end">
                    <!-- Plain button around the avatar; it keeps the global focus ring. -->
                    <button
                        type="button"
                        class="flex cursor-pointer rounded-full"
                        aria-label={user.display_name}
                        title={user.display_name}
                    >
                        <Avatar
                            displayName={user.display_name}
                            size="md"
                            userId={user.id}
                            hasAvatar={user.has_avatar}
                            avatarContentHash={user.avatar_content_hash}
                        />
                    </button>

                    <ul
                        class="menu menu-sm dropdown-content z-10 mt-3 w-52 rounded-box border border-base-300 bg-base-100 p-2 shadow-overlay"
                    >
                        <li>
                            <a href={resolve('/profile')} class="w-full text-left flex gap-4 items-center">
                                <CircleUserIcon size={16}/>
                                {user.display_name}
                            </a>
                        </li>
                        <li>
                            <a href={resolve('/profile/settings')} class="w-full text-left flex gap-4 items-center">
                                <SettingsIcon size={16}/>
                                {m.nav_account_settings()}
                            </a>
                        </li>
                        <div class="divider my-0"></div>

                        <!-- menu-title keeps the row free of menu-item hover styles. -->
                        <li class="menu-title p-0 text-xs font-normal text-base-content">
                            <AccountThemeSwitch theme={user.theme} />
                        </li>

                        <div class="divider my-0"></div> 

                        <li>
                            <form action="/logout" method="POST" use:enhance class="w-full">
                                <button 
                                    type="submit" 
                                    class="flex w-full items-center gap-4 text-left"
                                >
                                    <LogOutIcon size={16} />
                                    {m.nav_logout()}
                                </button>
                            </form>
                        </li>
                    </ul>
                </div>
            {:else}
                <GuestThemeToggle />
                <GuestLocaleMenu />
                <a href={resolve('/login')} class="btn shrink-0">
                    {m.nav_login()}
                </a>
            {/if}
        </div>
    </div>
</div>

{#snippet logo()}
    <a href={resolve('/')} class="flex min-w-0 flex-1 items-center gap-4">
        <img src="/favicon.svg" alt={m.page_home()} class="size-8 shrink-0" />
        <span class="min-w-0 truncate font-semibold">
            SOP and GO
        </span>
    </a>
{/snippet}

<style>
    .navbar-guest .navbar-start {
        width: auto;
        flex: 1 1 auto;
        min-width: 0;
    }

    .navbar-guest .navbar-end {
        width: auto;
        flex: 0 0 auto;
        margin-inline-start: auto;
    }

    .navbar-app .navbar-start {
        width: auto;
        flex: 1 1 auto;
        min-width: 0;
    }

    .navbar-app .navbar-end {
        width: auto;
        flex: 0 0 auto;
        min-width: 0;
    }

    .navbar-app .navbar-end {
        margin-inline-start: auto;
    }
</style>

{#snippet roleChip(role: UserRole)}
    <span class="badge badge-outline badge-sm shrink-0">{roleLabels[role]()}</span>
{/snippet}