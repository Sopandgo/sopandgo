<script lang="ts">
    import { enhance } from '$app/forms';
    import { ArchiveIcon, CircleUserIcon, LogOutIcon, LogsIcon, SettingsIcon, ShieldUserIcon, UsersIcon } from 'lucide-svelte';
    import Avatar from '$lib/components/Avatar.svelte';

    let { user } = $props();
</script>

<div class="navbar sticky top-0 z-50 bg-base-100 shadow-sm">
    <div class="navbar-start">
        {#if user}
            {@render logo("/dashboard")}
        {:else}
            {@render logo("/")}
        {/if}
    </div>

    {#if user?.role === 'admin'}
        <div class="navbar-center">
            <div class="badge badge-lg badge-dash badge-error">Admin</div>
        </div>
    {:else if user?.role === 'auditor'}
        <div class="navbar-center">
            <div class="badge badge-lg badge-dash badge-info">Auditor</div>
        </div>
    {/if}

    <div class="navbar-end">
        <div class="flex gap-2">
            {#if user}
                <div class="dropdown dropdown-end">
                    <div
                        tabindex="0"
                        role="button"
                        class="btn btn-ghost btn-circle avatar avatar-placeholder"
                    >
                        {#if user.role === 'admin'}
                            <Avatar displayName={user.display_name} color={"error"} size="md" />
                        {:else}
                            <Avatar displayName={user.display_name} color={"secondary"} size="md" />

                        {/if}
                    </div>

                    <ul
                        class="menu menu-sm dropdown-content bg-base-100 rounded-box z-10 mt-3 w-52 p-2 shadow"
                    >
                        <li>
                            <a href="/profile" class="w-full text-left flex gap-4 items-center">
                                <CircleUserIcon size={16}/>
                                {user.display_name}
                            </a>
                        </li>

                        {#if user.role === 'admin'}
                            <div class="divider my-0"></div> 

                            <li>
                                <a href="/admin/users" class="w-full text-left flex gap-4 items-center">
                                    <UsersIcon size={16}/>
                                    Users
                                </a>
                            </li>
                            <li>
                                <a href="/admin/sessions" class="w-full text-left flex gap-4 items-center">
                                    <ShieldUserIcon size={16}/>
                                    Sessions
                                </a>
                            </li>
                            <li>
                                <a href="/admin/audit-logs" class="w-full text-left flex gap-4 items-center">
                                    <LogsIcon size={16}/>
                                    Audit Logs
                                </a>
                            </li>
                            <li>
                                <a href="/admin/settings" class="w-full text-left flex gap-4 items-center">
                                    <SettingsIcon size={16}/>
                                    Settings
                                </a>
                            </li>
                            <li>
                                <a href="/admin/backup" class="w-full text-left flex gap-4 items-center">
                                    <ArchiveIcon size={16}/>
                                    Backup
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
                                    Logout
                                </button>
                            </form>
                        </li>
                    </ul>
                </div>
            {:else}
                <a href="/login" class="btn btn-ghost">
                    Login
                </a>
            {/if}
        </div>
    </div>
</div>

{#snippet logo(url:string)}
    <a href={url} class="flex-1 flex gap-4 items-center">
        <div
            tabindex="0"
            role="button"
            class="btn btn-ghost btn-circle avatar avatar-placeholder"
        >
            <img src="/favicon.svg" alt="sopandgo icon" />
        </div>
        <span class="font-semibold">
            SOP and GO
        </span>
    </a>
{/snippet}