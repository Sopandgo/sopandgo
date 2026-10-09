<script lang="ts">
    import * as m from '$lib/paraglide/messages.js';

    let { status }: { status: string | null | undefined } = $props();

    const badges: Record<string, { tone: string; label: () => string }> = {
        draft: { tone: 'badge-secondary', label: m.status_draft },
        rc: { tone: 'badge-warning', label: m.status_rc },
        published: { tone: 'badge-primary', label: m.status_published },
        rejected: { tone: 'badge-error', label: m.status_rejected },
        superseded: { tone: 'badge-ghost', label: m.status_superseded }
    };

    const badge = $derived(badges[status ?? ''] ?? { tone: 'badge-ghost', label: m.status_unclear });
</script>

<div class="badge {badge.tone}">{badge.label()}</div>
