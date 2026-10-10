<script lang="ts">
    import { CalendarClockIcon, FingerprintPatternIcon } from 'lucide-svelte';
    import Card from './Card.svelte';
    import IntegrityCheck from './IntegrityCheck.svelte';
    import ListRow from './ListRow.svelte';
    import type { IntegrityStatus } from '$lib/integrity';
    import * as m from '$lib/paraglide/messages.js';
    import { getLocale } from '$lib/paraglide/runtime';

    /* Version number and status live in the version page header; this card holds the rest. */
    interface Props {
        versionId: string;
        contentHash: string;
        createdAt: string;
        hashValid: boolean;
    }

    let { versionId, contentHash = '', createdAt = '', hashValid = false }: Props = $props();

    // Load-time result; IntegrityCheck shows any manual re-check in place.
    const loadStatus = $derived<IntegrityStatus>(hashValid ? 'verified' : 'mismatch');

    const formattedDate = $derived(createdAt ? new Date(createdAt).toLocaleString(getLocale()) : m.common_na());
</script>

<Card title={m.details_technical()}>
    <ul class="list">
        <ListRow title={m.common_created({ when: formattedDate })} meta={m.profile_timestamp()} icon={CalendarClockIcon} />

        <ListRow title={m.details_checksum()} icon={FingerprintPatternIcon}>
            {#snippet meta()}<span class="font-mono" title={contentHash}>sha256 {contentHash.slice(0, 8)}</span>{/snippet}
            {#snippet trailing()}<IntegrityCheck kind="version" id={versionId} initial={loadStatus} />{/snippet}
        </ListRow>
    </ul>
</Card>
