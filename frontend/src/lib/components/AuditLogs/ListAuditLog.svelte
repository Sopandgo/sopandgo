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
    } from "@lucide/svelte";

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

    // Colour means state: only failures are tinted; everything else is neutral.
    function rowTone(eventType: string): string {
        const event = eventType.toLowerCase();
        return event === "login_failed" || event === "email_failed"
            ? "bg-error/10 text-error"
            : "bg-base-200 text-base-content/70";
    }
</script>

<Card title={m.audit_log_count({ range: rangeLabel })}>
    <ul class="list">

        {#each auditEvents as auditEvent (auditEvent.id)}
            {@const eventId = String(auditEvent.id)}
            {@const Icon = rowIcon(auditEvent.entity_type)}
            {@const summary = eventSummary(auditEvent.event_type, auditEvent.payload)}
            {@const entityType = auditEvent.entity_type.toLowerCase()}
            {@const linkedSopId = versionSopId(auditEvent.payload)}
            {@const open = Boolean(openIds[eventId])}
            <li class="border-b border-base-300 last:border-b-0">
                <button
                    type="button"
                    class="list-row w-full cursor-pointer items-center text-left transition-colors duration-150 after:hidden hover:bg-base-200 focus-visible:-outline-offset-2"
                    aria-expanded={open}
                    onclick={() => toggle(eventId)}
                >
                    <div class="flex size-10 shrink-0 items-center justify-center rounded-field {rowTone(auditEvent.event_type)}" aria-hidden="true">
                        <Icon class="size-5" />
                    </div>

                    <div class="flex-1 min-w-0">
                        <div class="text-sm font-medium">
                            {eventTitle(auditEvent.event_type)}
                        </div>
                        <div class="text-xs text-base-content/70 break-all">
                            {actorLabel(auditEvent.actor_name)} · {formatEventTime(
                                auditEvent.created_at,
                            )}{#if summary}
                                · {summary}{/if}
                        </div>
                    </div>

                    <div class="flex items-center gap-2 shrink-0">
                        {#if auditEvent.hash_valid}
                            <div class="badge badge-soft badge-success gap-1">
                                <ShieldCheckIcon class="size-4" aria-hidden="true" />
                                {m.audit_hash_ok()}
                            </div>
                        {:else}
                            <div class="badge badge-soft badge-error gap-1">
                                <ShieldXIcon class="size-4" aria-hidden="true" />
                                {m.audit_hash_bad()}
                            </div>
                        {/if}

                        <ChevronDownIcon
                            class="size-4 text-base-content/70 transition-transform {open ? 'rotate-180' : ''}"
                        />
                    </div>
                </button>

                {#if open}
                    <div class="grid gap-3 px-4 pb-4 text-xs">
                        <div>
                            <div class="font-medium text-base-content/70">{m.audit_actor_id()}</div>
                            <div class="font-mono break-all">{auditEvent.actor_user_id || m.common_none()}</div>
                        </div>
                        <div>
                            <div class="font-medium text-base-content/70">{m.audit_entity()}</div>
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
                            <div class="font-medium text-base-content/70">{m.audit_payload()}</div>
                            <pre class="mt-1 whitespace-pre-wrap break-all font-mono">{formatPayload(
                                    auditEvent.payload,
                                )}</pre>
                        </div>
                        <div>
                            <div class="font-medium text-base-content/70">{m.audit_hash()}</div>
                            <div class="font-mono break-all">{auditEvent.hash}</div>
                        </div>
                        <div>
                            <div class="font-medium text-base-content/70">{m.audit_prev_hash()}</div>
                            <div class="font-mono break-all">{auditEvent.prev_hash}</div>
                        </div>
                    </div>
                {/if}
            </li>
        {:else}
            <li class="p-6">
                <div class="text-sm text-base-content/70">
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
