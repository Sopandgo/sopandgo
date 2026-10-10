<script lang="ts">
    import { getLocale, locales, setLocale } from '#lib/paraglide/runtime.js';
    import * as m from '#lib/paraglide/messages.js';

    const current = $derived(getLocale());

    function languageName(tag: string) {
        try {
            const label = new Intl.DisplayNames([tag], { type: 'language' }).of(tag) ?? tag;
            return label.charAt(0).toLocaleUpperCase(tag) + label.slice(1);
        } catch {
            return tag;
        }
    }

    function choose(tag: string) {
        if (!locales.includes(tag as (typeof locales)[number])) return;
        setLocale(tag as (typeof locales)[number]);
    }
</script>

<div class="dropdown dropdown-end">
    <div
        tabindex="0"
        role="button"
        class="btn btn-ghost shrink-0"
        aria-haspopup="menu"
        aria-label="{m.locale_label()}: {languageName(current)}"
    >
        <span class="max-w-32 truncate">{languageName(current)}</span>
    </div>
    <ul
        class="menu dropdown-content bg-base-100 rounded-box z-10 mt-3 max-h-80 w-52 overflow-y-auto p-2 shadow-overlay"
        role="menu"
    >
        {#each locales as tag (tag)}
            <li>
                <button
                    type="button"
                    class="w-full text-left"
                    class:font-semibold={tag === current}
                    aria-current={tag === current ? 'true' : undefined}
                    onclick={() => choose(tag)}
                >
                    {languageName(tag)}
                </button>
            </li>
        {/each}
    </ul>
</div>
