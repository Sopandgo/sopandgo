<script lang="ts">
    import type { Tag } from '#lib/sdk/types.js';
    import * as m from '#lib/paraglide/messages.js';

    /*
     * Tags on a ListRow's meta line (docs/design/style-guide.md → ListRow): at most
     * three, then a ghost "+N" badge that shows the rest in place and turns into
     * "Show fewer". The active tag always stays visible. With `onTagClick` each tag
     * is a filter button; with `tagHref` it is a link. Every tag is `relative z-10`
     * so it stays clickable above the row's stretched link.
     */
    interface Props {
        tags: Tag[];
        /** The tag the list is filtered by. */
        activeTagId?: string;
        onTagClick?: (tagId: string) => void;
        tagHref?: (tag: Tag) => string;
    }

    let { tags, activeTagId = '', onTagClick, tagHref }: Props = $props();

    const MAX_TAGS = 3;
    let expanded = $state(false);

    const ordered = $derived.by(() => {
        const i = tags.findIndex((t) => t.id === activeTagId);
        if (i < MAX_TAGS) return tags;
        return [tags[i], ...tags.slice(0, i), ...tags.slice(i + 1)];
    });
    const hidden = $derived(ordered.slice(MAX_TAGS));
    const shown = $derived(expanded ? ordered : ordered.slice(0, MAX_TAGS));

    function tagClass(tag: Tag) {
        return `badge badge-sm relative z-10 ${activeTagId === tag.id ? 'badge-soft badge-primary' : 'badge-outline'}`;
    }
</script>

{#each shown as tag (tag.id)}
    {#if onTagClick}
        <button
            type="button"
            onclick={() => onTagClick(tag.id)}
            class="{tagClass(tag)} cursor-pointer"
            aria-pressed={activeTagId === tag.id}
            aria-label={m.sops_filter_tag({ title: tag.title })}
        >
            {tag.title}
        </button>
    {:else if tagHref}
        <a href={tagHref(tag)} class="{tagClass(tag)} hover:underline" aria-label={m.sops_filter_tag({ title: tag.title })}>
            {tag.title}
        </a>
    {:else}
        <span class={tagClass(tag)}>{tag.title}</span>
    {/if}
{/each}
{#if hidden.length > 0}
    {#if expanded}
        <button
            type="button"
            onclick={() => (expanded = false)}
            class="badge badge-sm badge-ghost relative z-10 cursor-pointer"
            aria-expanded="true"
        >
            {m.sops_fewer_tags()}
        </button>
    {:else}
        <button
            type="button"
            onclick={() => (expanded = true)}
            class="badge badge-sm badge-ghost relative z-10 cursor-pointer"
            aria-expanded="false"
            aria-label={m.sops_more_tags_aria({ count: String(hidden.length) })}
            title={hidden.map((t) => t.title).join(', ')}
        >
            +{hidden.length}
        </button>
    {/if}
{/if}
