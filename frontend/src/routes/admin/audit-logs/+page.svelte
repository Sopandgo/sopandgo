<script lang="ts">
    import ListAuditLog from '#lib/components/AuditLogs/ListAuditLog.svelte';
    import Card from '#lib/components/Card.svelte';
    import CardPageHeading from '#lib/components/CardPageHeading.svelte';
    import Combobox, { type ComboboxOption } from '#lib/components/Combobox.svelte';
    import { LogsIcon, Link } from '@lucide/svelte';
    import { entityTitle, eventTitle } from '#lib/audit/present.js';
    import * as m from '#lib/paraglide/messages.js';

    import { goto } from '$app/navigation';
    import { page } from '$app/state';

    let { data } = $props();

    const roleLabels: Record<string, () => string> = {
        admin: m.role_admin,
        approver: m.role_approver,
        auditor: m.role_auditor,
        editor: m.role_editor,
        viewer: m.role_viewer
    };

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
    let totalPages = $derived(Math.max(1, Math.ceil(data.total / data.logsPerPage)));
    let hasActiveFilters = $derived(
        Boolean(
            data.filters?.type ||
            data.filters?.entity_type ||
            data.filters?.actor_user_id ||
            data.filters?.sop_id
        )
    );
    const eventOptions = $derived<ComboboxOption[]>(
        data.auditFilterOptions.event_types.map((type) => ({ value: type, label: eventTitle(type) }))
    );
    const entityOptions = $derived<ComboboxOption[]>(
        data.auditFilterOptions.entity_types.map((type) => ({ value: type, label: entityTitle(type) }))
    );
    const actorOptions = $derived<ComboboxOption[]>(
        data.users.map((user) => ({
            value: user.id,
            label: user.display_name,
            hint: roleLabels[user.role]?.(),
            keywords: [user.email]
        }))
    );
    const sopOptions = $derived<ComboboxOption[]>(
        data.sops.map((sop) => ({ value: sop.id, label: sop.title }))
    );

    function changePage(newPage: number) {
        if (newPage < 0 || newPage >= totalPages) return;
        const newUrl = new URL(page.url.href);
        newUrl.searchParams.set('page', newPage.toString());
        goto(newUrl, { reset: false });
    }

    function applyFilters() {
        const newUrl = new URL(page.url.href);
        newUrl.searchParams.set('page', '0');

        if (filterType.trim() !== '') newUrl.searchParams.set('type', filterType.trim());
        else newUrl.searchParams.delete('type');

        if (filterEntityType.trim() !== '') newUrl.searchParams.set('entity_type', filterEntityType.trim());
        else newUrl.searchParams.delete('entity_type');

        if (filterActorUserId.trim() !== '') newUrl.searchParams.set('actor_user_id', filterActorUserId.trim());
        else newUrl.searchParams.delete('actor_user_id');

        if (filterSopId.trim() !== '') newUrl.searchParams.set('sop_id', filterSopId.trim());
        else newUrl.searchParams.delete('sop_id');

        goto(newUrl, { reset: false });
    }

    function resetFilters() {
        filterType = '';
        filterEntityType = '';
        filterActorUserId = '';
        filterSopId = '';
        applyFilters();
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
                <div class="flex items-center gap-1.5 text-xs text-base-content/70 select-none">
                    <Link class="size-3.5" />
                    <span>{m.audit_chained()}</span>
                </div>
            </div>

            <form class="mt-4 grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-3" onsubmit={(e) => { e.preventDefault(); applyFilters(); }}>
                <Combobox
                    label={m.audit_event_type()}
                    placeholder={m.audit_search_event()}
                    options={eventOptions}
                    bind:value={filterType}
                    allLabel={m.audit_all_events()}
                    emptyLabel={m.audit_no_event()}
                />
                <Combobox
                    label={m.audit_entity_type()}
                    placeholder={m.audit_search_entity()}
                    options={entityOptions}
                    bind:value={filterEntityType}
                    allLabel={m.audit_all_entities()}
                    emptyLabel={m.audit_no_entity()}
                />
                <Combobox
                    label={m.audit_actor()}
                    placeholder={m.audit_search_actor()}
                    options={actorOptions}
                    bind:value={filterActorUserId}
                    allLabel={m.audit_all_actors()}
                    emptyLabel={m.audit_no_user()}
                    clearLabel={m.audit_clear_actor()}
                />
                <Combobox
                    label={m.audit_sop_id()}
                    placeholder={m.audit_search_sop()}
                    options={sopOptions}
                    bind:value={filterSopId}
                    allLabel={m.audit_all_sops()}
                    emptyLabel={m.audit_no_sop()}
                    clearLabel={m.audit_clear_sop()}
                />
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
