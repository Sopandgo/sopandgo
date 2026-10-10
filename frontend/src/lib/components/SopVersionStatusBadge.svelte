<script lang="ts">
    import * as m from '$lib/paraglide/messages.js';

    /*
     * Owns the status → look mapping (docs/design/style-guide.md → StatusBadge).
     * No other file picks a badge colour for a version status.
     */
    let { status }: { status: string | null | undefined } = $props();

    const badges: Record<string, { classes: string; label: () => string }> = {
        draft: { classes: 'badge badge-outline', label: m.status_draft },
        rc: { classes: 'badge badge-soft badge-warning', label: m.status_rc },
        published: { classes: 'badge badge-soft badge-success', label: m.status_published },
        rejected: { classes: 'badge badge-soft badge-error', label: m.status_rejected },
        // Superseded is history, not a state to act on: plain muted text.
        superseded: { classes: 'text-xs text-base-content/70', label: m.status_superseded }
    };

    const badge = $derived(badges[status ?? ''] ?? { classes: 'badge badge-outline', label: m.status_unclear });
</script>

<span class={badge.classes}>{badge.label()}</span>
