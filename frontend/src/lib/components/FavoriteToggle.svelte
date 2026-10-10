<script lang="ts">
    import { StarIcon } from '@lucide/svelte';
    import { enhance } from '$app/forms';
    import { page } from '$app/state';
    import { favoriteErrorFromForm } from '#lib/favorites.js';
    import * as m from '#lib/paraglide/messages.js';

    /*
     * Star button that adds or removes an SOP from the reader's favorites
     * (docs/design/style-guide.md → ListRow, favorite toggle). Posts to the
     * page's `?/favorite` / `?/unfavorite` actions ($lib/server/favoriteActions)
     * with the SOP id as `sop_id`. A failure shows in words next to the star.
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
    const error = $derived(favoriteErrorFromForm(page.form, sopId));
    const errorId = $derived(`favorite-error-${sopId}`);
</script>

<form
    method="POST"
    action={isFavorite ? '?/unfavorite' : '?/favorite'}
    class="flex items-center gap-2"
    use:enhance
>
    <input type="hidden" name="sop_id" value={sopId} />
    {#if error}
        <span id={errorId} role="alert" class="text-xs text-error">{error}</span>
    {/if}
    <button
        type="submit"
        class="btn btn-square btn-ghost btn-sm"
        aria-label={label}
        aria-pressed={isFavorite}
        aria-describedby={error ? errorId : undefined}
        title={label}
    >
        <StarIcon class="size-4 {isFavorite ? 'fill-current' : ''}" aria-hidden="true" />
    </button>
</form>
