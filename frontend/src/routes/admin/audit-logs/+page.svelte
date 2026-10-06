<script lang="ts">
    import ListAuditLog from '$lib/components/AuditLogs/ListAuditLog.svelte';
    import Card from '$lib/components/Card.svelte';
    import CardPageHeading from '$lib/components/CardPageHeading.svelte';
    import { LogsIcon, Link } from 'lucide-svelte';
    import { entityTitle, eventTitle } from '$lib/audit/present';
    import * as m from '$lib/paraglide/messages.js';

    import { goto } from '$app/navigation';
    import { resolve } from '$app/paths';
    import { page } from '$app/state';

    let { data } = $props();

    let auditEvents = $derived(data.auditEvents);
    let currentPage = $derived(data.page);
    let filterType = $state('');
    let filterEntityType = $state('');
    let filterActorUserId = $state('');
    let filterSopId = $state('');
    $effect(() => {
        filterType = data.filters?.type ?? '';
        filterEntityType = data.filters?.entity_type ?? '';
        filterActorUserId = data.filters?.actor_user_id ?? '';
        filterSopId = data.filters?.sop_id ?? '';
    });
    let eventTypeQuery = $state('');
    let showEventTypeOptions = $state(false);
    let entityTypeQuery = $state('');
    let showEntityTypeOptions = $state(false);
    let actorQuery = $state('');
    let showActorOptions = $state(false);
    let sopQuery = $state('');
    let showSopOptions = $state(false);
    
    let totalPages = $derived(Math.max(1, Math.ceil(data.total / data.logsPerPage)));
    let hasActiveFilters = $derived(
        Boolean(
            data.filters?.type ||
            data.filters?.entity_type ||
            data.filters?.actor_user_id ||
            data.filters?.sop_id
        )
    );
    let filteredSops = $derived(
        data.sops
            .filter((sop) => {
                const q = sopQuery.trim().toLowerCase();
                if (!q) return true;
                return sop.title.toLowerCase().includes(q) || sop.id.toLowerCase().includes(q);
            })
            .slice(0, 10)
    );
    let filteredUsers = $derived(
        data.users
            .filter((user) => {
                const q = actorQuery.trim().toLowerCase();
                if (!q) return true;
                return (
                    user.display_name.toLowerCase().includes(q) ||
                    user.email.toLowerCase().includes(q) ||
                    user.id.toLowerCase().includes(q)
                );
            })
            .slice(0, 10)
    );
    let filteredEventTypes = $derived(
        data.auditFilterOptions.event_types
            .filter((type) => {
                const q = eventTypeQuery.trim().toLowerCase();
                if (!q) return true;
                return type.toLowerCase().includes(q);
            })
            .slice(0, 10)
    );
    let filteredEntityTypes = $derived(
        data.auditFilterOptions.entity_types
            .filter((entityType) => {
                const q = entityTypeQuery.trim().toLowerCase();
                if (!q) return true;
                return entityType.toLowerCase().includes(q);
            })
            .slice(0, 10)
    );

    $effect(() => {
        eventTypeQuery = filterType;
    });

    $effect(() => {
        entityTypeQuery = filterEntityType;
    });

    $effect(() => {
        if (!filterActorUserId) {
            actorQuery = '';
            return;
        }
        const selectedUser = data.users.find((user) => user.id === filterActorUserId);
        actorQuery = selectedUser ? selectedUser.display_name : filterActorUserId;
    });

    $effect(() => {
        if (!filterSopId) {
            sopQuery = '';
            return;
        }
        const selectedSop = data.sops.find((sop) => sop.id === filterSopId);
        sopQuery = selectedSop ? selectedSop.title : filterSopId;
    });

    function changePage(newPage: number) {
        if (newPage < 0 || newPage >= totalPages) return;
        const newUrl = new URL(page.url);
        newUrl.searchParams.set('page', newPage.toString());
        goto(resolve((`/admin/audit-logs${newUrl.search}`) as '/admin/audit-logs'), {
            keepFocus: true,
            noScroll: false
        });
    }

    function applyFilters() {
        const newUrl = new URL(page.url);
        newUrl.searchParams.set('page', '0');

        if (filterType.trim() !== '') newUrl.searchParams.set('type', filterType.trim());
        else newUrl.searchParams.delete('type');

        if (filterEntityType.trim() !== '') newUrl.searchParams.set('entity_type', filterEntityType.trim());
        else newUrl.searchParams.delete('entity_type');

        if (filterActorUserId.trim() !== '') newUrl.searchParams.set('actor_user_id', filterActorUserId.trim());
        else newUrl.searchParams.delete('actor_user_id');

        if (filterSopId.trim() !== '') newUrl.searchParams.set('sop_id', filterSopId.trim());
        else newUrl.searchParams.delete('sop_id');

        goto(resolve((`/admin/audit-logs${newUrl.search}`) as '/admin/audit-logs'), {
            keepFocus: true,
            noScroll: false
        });
    }

    function resetFilters() {
        filterType = '';
        filterEntityType = '';
        filterActorUserId = '';
        filterSopId = '';
        applyFilters();
    }

    function selectSop(id: string, title: string) {
        filterSopId = id;
        sopQuery = title;
        showSopOptions = false;
    }

    function clearSopSelection() {
        filterSopId = '';
        sopQuery = '';
        showSopOptions = false;
    }

    function selectActor(id: string, label: string) {
        filterActorUserId = id;
        actorQuery = label;
        showActorOptions = false;
    }

    function clearActorSelection() {
        filterActorUserId = '';
        actorQuery = '';
        showActorOptions = false;
    }

    function selectEventType(value: string) {
        filterType = value;
        eventTypeQuery = value;
        showEventTypeOptions = false;
    }

    function clearEventTypeSelection() {
        filterType = '';
        eventTypeQuery = '';
        showEventTypeOptions = false;
    }

    function selectEntityType(value: string) {
        filterEntityType = value;
        entityTypeQuery = value;
        showEntityTypeOptions = false;
    }

    function clearEntityTypeSelection() {
        filterEntityType = '';
        entityTypeQuery = '';
        showEntityTypeOptions = false;
    }
</script>

<svelte:head>
    <title>{m.page_audit_logs()}</title>
</svelte:head>

<div class="flex flex-col gap-6">
    <Card>
        <div class="card-body">
            <CardPageHeading>
                <LogsIcon class="w-8 h-8" />
                {m.audit_heading()}
            </CardPageHeading>

            <div class="flex flex-col gap-1 mt-1">
                <span class="text-sm font-medium text-base-content/70">
                    {m.audit_monitor()}
                </span> 
                <div class="flex items-center gap-1.5 text-xs text-base-content/50 select-none">
                    <Link class="size-3.5" />
                    <span>{m.audit_chained()}</span>
                </div>
            </div>

            <form class="mt-4 grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-3" onsubmit={(e) => { e.preventDefault(); applyFilters(); }}>
                <label class="form-control w-full">
                    <div class="label"><span class="label-text text-xs">{m.audit_event_type()}</span></div>
                    <div class="relative">
                        <input
                            class="input input-bordered w-full"
                            placeholder={m.audit_search_event()}
                            bind:value={eventTypeQuery}
                            onfocus={() => { showEventTypeOptions = true; }}
                            oninput={() => { showEventTypeOptions = true; filterType = ''; }}
                            onblur={() => { setTimeout(() => { showEventTypeOptions = false; }, 120); }}
                        />
                        {#if filterType}
                            <button type="button" class="btn btn-ghost btn-xs absolute right-2 top-2" onclick={clearEventTypeSelection}>{m.common_clear()}</button>
                        {/if}
                        {#if showEventTypeOptions}
                            <ul class="menu bg-base-100 border border-base-300 rounded-box w-full mt-1 absolute z-20 shadow-lg">
                                <li><button type="button" class="text-left" onclick={clearEventTypeSelection}>{m.audit_all_events()}</button></li>
                                {#if filteredEventTypes.length === 0}
                                    <li><span class="text-xs opacity-60">{m.audit_no_event()}</span></li>
                                {:else}
                                    {#each filteredEventTypes as option (option)}
                                        <li>
                                            <button type="button" class="text-left w-full" onclick={() => selectEventType(option)} title={option}>
                                                <span class="font-medium block truncate">{eventTitle(option)}</span>
                                            </button>
                                        </li>
                                    {/each}
                                {/if}
                            </ul>
                        {/if}
                    </div>
                </label>
                <label class="form-control w-full">
                    <div class="label"><span class="label-text text-xs">{m.audit_entity_type()}</span></div>
                    <div class="relative">
                        <input
                            class="input input-bordered w-full"
                            placeholder={m.audit_search_entity()}
                            bind:value={entityTypeQuery}
                            onfocus={() => { showEntityTypeOptions = true; }}
                            oninput={() => { showEntityTypeOptions = true; filterEntityType = ''; }}
                            onblur={() => { setTimeout(() => { showEntityTypeOptions = false; }, 120); }}
                        />
                        {#if filterEntityType}
                            <button type="button" class="btn btn-ghost btn-xs absolute right-2 top-2" onclick={clearEntityTypeSelection}>{m.common_clear()}</button>
                        {/if}
                        {#if showEntityTypeOptions}
                            <ul class="menu bg-base-100 border border-base-300 rounded-box w-full mt-1 absolute z-20 shadow-lg">
                                <li><button type="button" class="text-left" onclick={clearEntityTypeSelection}>{m.audit_all_entities()}</button></li>
                                {#if filteredEntityTypes.length === 0}
                                    <li><span class="text-xs opacity-60">{m.audit_no_entity()}</span></li>
                                {:else}
                                    {#each filteredEntityTypes as option (option)}
                                        <li>
                                            <button type="button" class="text-left w-full" onclick={() => selectEntityType(option)} title={option}>
                                                <span class="font-medium block truncate">{entityTitle(option)}</span>
                                            </button>
                                        </li>
                                    {/each}
                                {/if}
                            </ul>
                        {/if}
                    </div>
                </label>
                <label class="form-control w-full">
                    <div class="label"><span class="label-text text-xs">{m.audit_actor()}</span></div>
                    <div class="relative">
                        <input
                            class="input input-bordered w-full"
                            placeholder={m.audit_search_actor()}
                            bind:value={actorQuery}
                            onfocus={() => { showActorOptions = true; }}
                            oninput={() => { showActorOptions = true; filterActorUserId = ''; }}
                            onblur={() => {
                                setTimeout(() => {
                                    showActorOptions = false;
                                }, 120);
                            }}
                        />
                        {#if filterActorUserId}
                            <button
                                type="button"
                                class="btn btn-ghost btn-xs absolute right-2 top-2"
                                onclick={clearActorSelection}
                                aria-label={m.audit_clear_actor()}
                            >
                                {m.common_clear()}
                            </button>
                        {/if}
                        {#if showActorOptions}
                            <ul class="menu bg-base-100 border border-base-300 rounded-box w-full mt-1 absolute z-20 shadow-lg">
                                <li>
                                    <button
                                        type="button"
                                        class="text-left"
                                        onclick={clearActorSelection}
                                    >
                                        {m.audit_all_actors()}
                                    </button>
                                </li>
                                {#if filteredUsers.length === 0}
                                    <li><span class="text-xs opacity-60">{m.audit_no_user()}</span></li>
                                {:else}
                                    {#each filteredUsers as user (user.id)}
                                        <li>
                                            <button
                                                type="button"
                                                class="text-left w-full"
                                                onclick={() => selectActor(user.id, user.display_name)}
                                                title={user.display_name}
                                            >
                                                <span class="font-medium block truncate">{user.display_name}</span>
                                            </button>
                                        </li>
                                    {/each}
                                {/if}
                            </ul>
                        {/if}
                    </div>
                </label>
                <label class="form-control w-full">
                    <div class="label"><span class="label-text text-xs">{m.audit_sop_id()}</span></div>
                    <div class="relative">
                        <input
                            class="input input-bordered w-full"
                            placeholder={m.audit_search_sop()}
                            bind:value={sopQuery}
                            onfocus={() => { showSopOptions = true; }}
                            oninput={() => { showSopOptions = true; filterSopId = ''; }}
                            onblur={() => {
                                setTimeout(() => {
                                    showSopOptions = false;
                                }, 120);
                            }}
                        />
                        {#if filterSopId}
                            <button
                                type="button"
                                class="btn btn-ghost btn-xs absolute right-2 top-2"
                                onclick={clearSopSelection}
                                aria-label={m.audit_clear_sop()}
                            >
                                {m.common_clear()}
                            </button>
                        {/if}

                        {#if showSopOptions}
                            <ul class="menu bg-base-100 border border-base-300 rounded-box w-full mt-1 absolute z-20 shadow-lg">
                                <li>
                                    <button
                                        type="button"
                                        class="text-left"
                                        onclick={clearSopSelection}
                                    >
                                        {m.audit_all_sops()}
                                    </button>
                                </li>
                                {#if filteredSops.length === 0}
                                    <li><span class="text-xs opacity-60">{m.audit_no_sop()}</span></li>
                                {:else}
                                    {#each filteredSops as sop (sop.id)}
                                        <li>
                                            <button
                                                type="button"
                                                class="text-left w-full"
                                                onclick={() => selectSop(sop.id, sop.title)}
                                                title={sop.title}
                                            >
                                                <span class="font-medium block truncate">{sop.title}</span>
                                            </button>
                                        </li>
                                    {/each}
                                {/if}
                            </ul>
                        {/if}
                    </div>
                </label>
                <div class="md:col-span-2 lg:col-span-4 flex gap-2">
                    <button type="submit" class="btn btn-primary btn-sm">{m.audit_apply()}</button>
                    <button type="button" class="btn btn-ghost btn-sm" onclick={resetFilters}>{m.common_reset()}</button>
                </div>
            </form>
        </div>
    </Card>

    <ListAuditLog
        {auditEvents}
        count={data.total}
        page={data.page}
        logsPerPage={data.logsPerPage}
        filtered={hasActiveFilters}
    />

    <div class="flex justify-center">
        <div class="join">
            <button 
                class="join-item btn" 
                onclick={() => changePage(currentPage - 1)}
                disabled={currentPage === 0}
            >«</button>
            
            <button class="join-item btn pointer-events-none min-w-[100px]">
                {m.common_page_of({ page: String(currentPage + 1), total: String(totalPages) })}
            </button>
            
            <button 
                class="join-item btn" 
                onclick={() => changePage(currentPage + 1)}
                disabled={currentPage >= totalPages - 1} 
            >»</button>
        </div>
    </div>
</div>