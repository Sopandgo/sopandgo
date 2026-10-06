<script lang="ts">
    import { resolve } from "$app/paths";
    import type { AuditEvent } from "$lib/sdk/types";
    import Card from "$lib/components/Card.svelte";
    import {
        actorLabel,
        entityTitle,
        eventSummary,
        eventTitle,
        formatEventTime,
        formatPayload,
        versionSopId,
    } from "$lib/audit/present";
    import * as m from "$lib/paraglide/messages.js";
    import {
        BoxIcon,
        ChevronDownIcon,
        CogIcon,
        MailIcon,
        NotebookIcon,
        NotebookTextIcon,
        ShieldCheckIcon,
        ShieldXIcon,
        TagIcon,
        UserIcon,
    } from "lucide-svelte";

    interface Props {
        auditEvents?: AuditEvent[] | null;
        count: number;
        page: number;
        logsPerPage: number;
        filtered?: boolean;
    }

    let {
        auditEvents = [],
        count,
        page,
        logsPerPage,
        filtered = false,
    }: Props = $props();

    let openIds = $state<Record<string, boolean>>({});

    let rangeLabel = $derived.by(() => {
        if (count === 0) return "0";
        const start = page * logsPerPage + 1;
        const end = Math.min(count, (page + 1) * logsPerPage);
        return m.audit_range({ start: String(start), end: String(end), count: String(count) });
    });

    function toggle(id: string) {
        openIds[id] = !openIds[id];
    }

    function rowIcon(entityType: string) {
        switch (entityType.toLowerCase()) {
            case "system":
                return CogIcon;
            case "user":
                return UserIcon;
            case "mail":
            case "email":
                return MailIcon;
            case "sop":
                return NotebookIcon;
            case "sop_version":
            case "acknowledgment":
                return NotebookTextIcon;
            case "tag":
                return TagIcon;
            default:
                return BoxIcon;
        }
    }

    function rowTone(entityType: string, eventType: string): string {
        const event = eventType.toLowerCase();
        if (
            event === "login_failed" ||
            event === "email_failed" ||
            event === "admin_revoked_all_user_sessions" ||
            event === "system_global_revocation"
        ) {
            return "bg-error text-error-content";
        }

        switch (entityType.toLowerCase()) {
            case "system":
            case "mail":
            case "email":
                return "bg-info text-info-content";
            case "user":
                return "bg-success text-success-content";
            case "sop":
            case "sop_version":
            case "sop_asset":
            case "acknowledgment":
                return "bg-primary text-primary-content";
            default:
                return "bg-secondary text-secondary-content";
        }
    }
</script>

<Card>
    <ul class="list">
        <li class="p-4 pb-2 text-xs opacity-60 tracking-widest uppercase font-bold">
            {m.audit_log_count({ range: rangeLabel })}
        </li>

        {#each auditEvents as auditEvent (auditEvent.id)}
            {@const eventId = String(auditEvent.id)}
            {@const Icon = rowIcon(auditEvent.entity_type)}
            {@const summary = eventSummary(auditEvent.event_type, auditEvent.payload)}
            {@const entityType = auditEvent.entity_type.toLowerCase()}
            {@const linkedSopId = versionSopId(auditEvent.payload)}
            {@const open = Boolean(openIds[eventId])}
            <li class="border-b border-base-200 last:border-b-0">
                <button
                    type="button"
                    class="list-row w-full items-center text-left cursor-pointer hover:bg-base-200/50 transition-colors"
                    aria-expanded={open}
                    onclick={() => toggle(eventId)}
                >
                    <div class="avatar avatar-placeholder">
                        <div
                            class="w-12 lg:w-16 rounded-field flex items-center justify-center {rowTone(
                                auditEvent.entity_type,
                                auditEvent.event_type,
                            )}"
                        >
                            <Icon size={24} />
                        </div>
                    </div>

                    <div class="flex-1 min-w-0">
                        <div class="font-bold text-sm lg:text-base">
                            {eventTitle(auditEvent.event_type)}
                        </div>
                        <div class="text-xs opacity-60 break-all">
                            {actorLabel(auditEvent.actor_name)} · {formatEventTime(
                                auditEvent.created_at,
                            )}{#if summary}
                                · {summary}{/if}
                        </div>
                    </div>

                    <div class="flex items-center gap-2 shrink-0">
                        {#if auditEvent.hash_valid}
                            <div class="badge badge-success badge-outline gap-1 h-7">
                                <ShieldCheckIcon class="size-4" />
                                {m.audit_hash_ok()}
                            </div>
                        {:else}
                            <div class="badge badge-error badge-outline gap-1 h-7">
                                <ShieldXIcon class="size-4" />
                                {m.audit_hash_bad()}
                            </div>
                        {/if}

                        <ChevronDownIcon
                            class="size-4 opacity-50 transition-transform {open ? 'rotate-180' : ''}"
                        />
                    </div>
                </button>

                {#if open}
                    <div class="px-4 pb-4 grid gap-3 text-xs">
                        <div>
                            <div class="opacity-50 uppercase tracking-widest font-bold">{m.audit_actor_id()}</div>
                            <div class="font-mono break-all">{auditEvent.actor_user_id || m.common_none()}</div>
                        </div>
                        <div>
                            <div class="opacity-50 uppercase tracking-widest font-bold">{m.audit_entity()}</div>
                            {#if entityType === "sop" && auditEvent.entity_id}
                                <a
                                    class="link font-mono break-all"
                                    href={resolve(`/sops/${auditEvent.entity_id}`)}
                                >
                                    {entityTitle(auditEvent.entity_type)} · {auditEvent.entity_id}
                                </a>
                            {:else if entityType === "sop_version" && linkedSopId && auditEvent.entity_id}
                                <a
                                    class="link font-mono break-all"
                                    href={resolve(
                                        `/sops/${linkedSopId}/v/${auditEvent.entity_id}`,
                                    )}
                                >
                                    {entityTitle(auditEvent.entity_type)} · {auditEvent.entity_id}
                                </a>
                            {:else}
                                <div class="font-mono break-all">
                                    {entityTitle(auditEvent.entity_type)} · {auditEvent.entity_id}
                                </div>
                            {/if}
                        </div>
                        <div>
                            <div class="opacity-50 uppercase tracking-widest font-bold">{m.audit_payload()}</div>
                            <pre class="mt-1 whitespace-pre-wrap break-all font-mono">{formatPayload(
                                    auditEvent.payload,
                                )}</pre>
                        </div>
                        <div>
                            <div class="opacity-50 uppercase tracking-widest font-bold">{m.audit_hash()}</div>
                            <div class="font-mono break-all">{auditEvent.hash}</div>
                        </div>
                        <div>
                            <div class="opacity-50 uppercase tracking-widest font-bold">{m.audit_prev_hash()}</div>
                            <div class="font-mono break-all">{auditEvent.prev_hash}</div>
                        </div>
                    </div>
                {/if}
            </li>
        {:else}
            <li class="p-12 text-center">
                <div class="text-sm opacity-40">
                    {#if filtered}
                        {m.audit_no_match()}
                    {:else}
                        {m.audit_empty()}
                    {/if}
                </div>
            </li>
        {/each}
    </ul>
</Card>
