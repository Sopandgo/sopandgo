<script lang="ts">
    import { enhance } from '$app/forms';
    import { resolve } from '$app/paths';
    import { CircleUserIcon, LogOutIcon, PanelLeftCloseIcon, PanelLeftOpenIcon, SettingsIcon } from 'lucide-svelte';
    import Avatar from '$lib/components/Avatar.svelte';
    import GuestLocaleMenu from '$lib/components/GuestLocaleMenu.svelte';
    import GuestThemeToggle from '$lib/components/GuestThemeToggle.svelte';
    import * as m from '$lib/paraglide/messages.js';
    import type { UserRole } from '$lib/sdk/types';

    let { user, rail = false } = $props();

    function roleBadge(role: UserRole): { label: string; tone: string } {
        switch (role) {
            case 'admin':
                return { label: m.role_admin(), tone: 'badge-error' };
            case 'auditor':
                return { label: m.role_auditor(), tone: 'badge-info' };
            case 'editor':
                return { label: m.role_editor(), tone: 'badge-ghost' };
            case 'approver':
                return { label: m.role_approver(), tone: 'badge-ghost' };
            case 'viewer':
                return { label: m.role_viewer(), tone: 'badge-ghost' };
        }
    }
</script>

<div
    class="navbar sticky top-0 z-30 bg-base-100 shadow-sm"
    class:navbar-guest={!user}
    class:navbar-app={!!user}
>
    <div class="navbar-start">
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
        {:else if !user}
            {@render logo()}
        {/if}
    </div>

    <div class="navbar-end">
        <div class="flex items-center gap-2">
            {#if user}
                {@render roleChip(user.role)}
                <div class="dropdown dropdown-end">
                    <div
                        tabindex="0"
                        role="button"
                        class="btn btn-ghost btn-circle avatar avatar-placeholder"
                    >
                        {#if user.role === 'admin'}
                            <Avatar displayName={user.display_name} color="error" size="md" />
                        {:else}
                            <Avatar displayName={user.display_name} color="secondary" size="md" />

                        {/if}
                    </div>

                    <ul
                        class="menu menu-sm dropdown-content bg-base-100 rounded-box z-10 mt-3 w-52 p-2 shadow"
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

                        <li>
                            <form action="/logout" method="POST" use:enhance class="w-full">
                                <button 
                                    type="submit" 
                                    class="text-error w-full font-bold text-left flex gap-4 items-center"
                                >
                                    <LogOutIcon size={16} strokeWidth={3}/>
                                    {m.nav_logout()}
                                </button>
                            </form>
                        </li>
                    </ul>
                </div>
            {:else}
                <GuestThemeToggle />
                <GuestLocaleMenu />
                <a href={resolve('/login')} class="btn btn-primary shrink-0">
                    {m.nav_login()}
                </a>
            {/if}
        </div>
    </div>
</div>

{#snippet logo()}
    <a href={resolve('/')} class="flex min-w-0 flex-1 items-center gap-4">
        <div
            tabindex="0"
            role="button"
            class="btn btn-ghost btn-circle avatar avatar-placeholder shrink-0"
        >
            <img src="/favicon.svg" alt={m.page_home()} />
        </div>
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
    {@const badge = roleBadge(role)}
    <div class="badge badge-sm shrink-0 {badge.tone}">{badge.label}</div>
{/snippet}