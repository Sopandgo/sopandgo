<script lang="ts">
    import { enhance } from '$app/forms';
    import { resolve } from '$app/paths';
    import { ArchiveIcon, CircleUserIcon, LogOutIcon, LogsIcon, SettingsIcon, ShieldUserIcon, UsersIcon } from 'lucide-svelte';
    import Avatar from '$lib/components/Avatar.svelte';
    import GuestLocaleMenu from '$lib/components/GuestLocaleMenu.svelte';
    import GuestThemeToggle from '$lib/components/GuestThemeToggle.svelte';
    import * as m from '$lib/paraglide/messages.js';

    let { user } = $props();
</script>

<div class="navbar sticky top-0 z-50 bg-base-100 shadow-sm" class:navbar-guest={!user}>
    <div class="navbar-start">
        {#if user}
            {@render logo('/dashboard')}
        {:else}
            {@render logo('/')}
        {/if}
    </div>

    {#if user?.role === 'admin'}
        <div class="navbar-center">
            <div class="badge badge-lg badge-dash badge-error">{m.role_admin()}</div>
        </div>
    {:else if user?.role === 'auditor'}
        <div class="navbar-center">
            <div class="badge badge-lg badge-dash badge-info">{m.role_auditor()}</div>
        </div>
    {/if}

    <div class="navbar-end">
        <div class="flex items-center gap-2">
            {#if user}
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

                        {#if user.role === 'admin'}
                            <div class="divider my-0"></div> 

                            <li>
                                <a href={resolve('/admin/users')} class="w-full text-left flex gap-4 items-center">
                                    <UsersIcon size={16}/>
                                    {m.nav_users()}
                                </a>
                            </li>
                            <li>
                                <a href={resolve('/admin/sessions')} class="w-full text-left flex gap-4 items-center">
                                    <ShieldUserIcon size={16}/>
                                    {m.nav_sessions()}
                                </a>
                            </li>
                            <li>
                                <a href={resolve('/admin/audit-logs')} class="w-full text-left flex gap-4 items-center">
                                    <LogsIcon size={16}/>
                                    {m.nav_audit_logs()}
                                </a>
                            </li>
                            <li>
                                <a href={resolve('/admin/settings')} class="w-full text-left flex gap-4 items-center">
                                    <SettingsIcon size={16}/>
                                    {m.nav_settings()}
                                </a>
                            </li>
                            <li>
                                <a href={resolve('/admin/backup')} class="w-full text-left flex gap-4 items-center">
                                    <ArchiveIcon size={16}/>
                                    {m.nav_backup()}
                                </a>
                            </li>
                        {/if}

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

{#snippet logo(url: '/' | '/dashboard')}
    <a href={resolve(url)} class="flex min-w-0 flex-1 items-center gap-4">
        <div
            tabindex="0"
            role="button"
            class="btn btn-ghost btn-circle avatar avatar-placeholder shrink-0"
        >
            <img src="/favicon.svg" alt={m.page_home()} />
        </div>
        <span class="truncate font-semibold">
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
</style>