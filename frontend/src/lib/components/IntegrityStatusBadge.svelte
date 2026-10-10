<script lang="ts">
    import { FileXIcon, ShieldAlertIcon, ShieldCheckIcon, ShieldXIcon } from '@lucide/svelte';
    import type { IntegrityStatus } from '$lib/integrity';
    import * as m from '$lib/paraglide/messages.js';

    /*
     * docs/design/style-guide.md → IntegrityStatus. Every state has its own word
     * and icon; "Hash mismatch" is only ever shown for a confirmed mismatch.
     */
    let { status }: { status: IntegrityStatus } = $props();

    const view = $derived(
        {
            verified: { classes: 'badge-soft badge-success', icon: ShieldCheckIcon, label: m.integrity_verified(), help: m.integrity_verified_help() },
            mismatch: { classes: 'badge-soft badge-error', icon: ShieldXIcon, label: m.integrity_hash_mismatch(), help: m.integrity_mismatch_help() },
            missing: { classes: 'badge-soft badge-warning', icon: FileXIcon, label: m.integrity_missing(), help: m.integrity_missing_help() },
            unavailable: { classes: 'badge-outline', icon: ShieldAlertIcon, label: m.integrity_unavailable(), help: m.integrity_unavailable_help() }
        }[status]
    );
</script>

<span class="badge {view.classes} gap-1 whitespace-nowrap" title={view.help}>
    <view.icon class="size-4" aria-hidden="true" />
    {view.label}
</span>
