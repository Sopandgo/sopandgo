<script lang="ts">
    import type { Snippet } from 'svelte';
    import type { SOPArchive } from '#lib/sdk/types.js';
    import Alert from './Alert.svelte';
    import * as m from '#lib/paraglide/messages.js';
    import { getLocale } from '#lib/paraglide/runtime.js';

    /*
     * The one notice an archived SOP's pages show: who archived it, when and why,
     * and that nothing on it changes any more. `actions` holds the unarchive form.
     */
    let { archive, actions }: { archive: SOPArchive; actions?: Snippet } = $props();

    let title = $derived(
        m.sop_archived_title({
            date: new Date(archive.archived_at).toLocaleDateString(getLocale()),
            name: archive.actor_name || archive.actor_user_id
        })
    );
</script>

<Alert type="warning" role="note" {title}>
    <div class="flex flex-col gap-3 sm:flex-row sm:items-start sm:justify-between">
        <div class="flex flex-col gap-2">
            <p class="whitespace-pre-wrap">{archive.reason}</p>
            <p>{m.sop_archived_body()}</p>
        </div>
        {#if actions}
            <div class="shrink-0">{@render actions()}</div>
        {/if}
    </div>
</Alert>
