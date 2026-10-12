<script lang="ts">
    import type { SOPArchive } from '#lib/sdk/types.js';
    import Alert from './Alert.svelte';
    import * as m from '#lib/paraglide/messages.js';
    import { getLocale } from '#lib/paraglide/runtime.js';

    /*
     * The one notice an archived SOP's pages show, above the heading card: who
     * archived it, when and why, and that nothing on it changes any more.
     * Unarchiving lives in its own card at the bottom of the SOP page.
     */
    let { archive }: { archive: SOPArchive } = $props();

    let title = $derived(
        m.sop_archived_title({
            date: new Date(archive.archived_at).toLocaleDateString(getLocale()),
            name: archive.actor_name || archive.actor_user_id
        })
    );
</script>

<Alert type="warning" role="note" {title}>
    <div class="flex flex-col gap-2">
        <p class="whitespace-pre-wrap">{archive.reason}</p>
        <p>{m.sop_archived_body()}</p>
    </div>
</Alert>
