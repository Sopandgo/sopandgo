<script lang="ts">
    import { CalendarClockIcon, FingerprintPatternIcon, GitBranchIcon, WaypointsIcon } from '@lucide/svelte';
    import Card from './Card.svelte';
    import IntegrityCheck from './IntegrityCheck.svelte';
    import ListRow from './ListRow.svelte';
    import SopVersionStatusBadge from './SopVersionStatusBadge.svelte';
    import type { IntegrityStatus } from '#lib/integrity.js';
    import * as m from '#lib/paraglide/messages.js';
    import { getLocale } from '#lib/paraglide/runtime.js';

    /*
     * The version's facts in the side column. Number and status repeat the page
     * header on purpose, so the column reads on its own; the number links to the
     * SOP page with its tags and the other versions.
     */
    interface Props {
        sopId: string;
        versionId: string;
        contentHash: string;
        createdAt: string;
        hashValid: boolean;
        status: string;
        version: number | string;
    }

    let { sopId, versionId, contentHash = '', createdAt = '', hashValid = false, status = '', version }: Props =
        $props();

    // Load-time result; IntegrityCheck shows any manual re-check in place.
    const loadStatus = $derived<IntegrityStatus>(hashValid ? 'verified' : 'mismatch');

    const formattedDate = $derived(createdAt ? new Date(createdAt).toLocaleString(getLocale()) : m.common_na());
</script>

<Card title={m.details_technical()}>
    <ul class="list">
        <ListRow
            href={`/sops/${sopId}`}
            title={m.details_version({ version: String(version) })}
            meta={m.details_version_number()}
            icon={GitBranchIcon}
        />

        <ListRow title={m.common_created({ when: formattedDate })} meta={m.profile_timestamp()} icon={CalendarClockIcon} />

        <ListRow title={m.details_status()} meta={m.details_status_help()} icon={WaypointsIcon}>
            {#snippet trailing()}<SopVersionStatusBadge {status} />{/snippet}
        </ListRow>

        <ListRow title={m.details_checksum()} icon={FingerprintPatternIcon}>
            {#snippet meta()}<span class="font-mono" title={contentHash}>sha256 {contentHash.slice(0, 8)}</span>{/snippet}
            {#snippet trailing()}<IntegrityCheck kind="version" id={versionId} initial={loadStatus} />{/snippet}
        </ListRow>
    </ul>
</Card>
