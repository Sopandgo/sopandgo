<script lang="ts">
    import { StarIcon } from 'lucide-svelte';
    import { enhance } from '$app/forms';
    import * as m from '$lib/paraglide/messages.js';

    /*
     * Star button that adds or removes an SOP from the reader's favorites
     * (docs/design/style-guide.md → ListRow, favorite toggle). Posts to the
     * page's `?/favorite` / `?/unfavorite` actions with the SOP id as `sop_id`.
     */
    interface Props {
        sopId: string;
        title: string;
        isFavorite: boolean;
    }

    let { sopId, title, isFavorite }: Props = $props();

    const label = $derived(
        isFavorite ? m.aria_remove_favorite({ title }) : m.aria_add_favorite({ title })
    );
</script>

<form method="POST" action={isFavorite ? '?/unfavorite' : '?/favorite'} use:enhance>
    <input type="hidden" name="sop_id" value={sopId} />
    <button
        type="submit"
        class="btn btn-square btn-ghost btn-sm"
        aria-label={label}
        aria-pressed={isFavorite}
        title={label}
    >
        <StarIcon class="size-4 {isFavorite ? 'fill-current' : ''}" aria-hidden="true" />
    </button>
</form>
