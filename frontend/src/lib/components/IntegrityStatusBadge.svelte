<script lang="ts">
    import { FileXIcon, ShieldAlertIcon, ShieldCheckIcon, ShieldXIcon } from 'lucide-svelte';
    import type { IntegrityStatus } from '$lib/integrity';
    import * as m from '$lib/paraglide/messages.js';

    let { status }: { status: IntegrityStatus } = $props();

    const view = $derived(
        {
            verified: { tone: 'badge-success', icon: ShieldCheckIcon, label: m.integrity_verified(), help: m.integrity_verified_help() },
            corrupt: { tone: 'badge-error', icon: ShieldXIcon, label: m.integrity_corrupt(), help: m.integrity_corrupt_help() },
            missing: { tone: 'badge-error', icon: FileXIcon, label: m.integrity_missing(), help: m.integrity_missing_help() },
            unavailable: { tone: 'badge-warning', icon: ShieldAlertIcon, label: m.integrity_unavailable(), help: m.integrity_unavailable_help() }
        }[status]
    );
</script>

<span class="badge {view.tone} badge-outline gap-1 h-7 whitespace-nowrap" title={view.help}>
    <view.icon class="size-3" aria-hidden="true" />
    {view.label}
</span>
