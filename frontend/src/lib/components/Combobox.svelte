<script lang="ts" module>
    export interface ComboboxOption {
        value: string;
        label: string;
        /** Secondary text shown after the label, e.g. a role. */
        hint?: string;
        /** Extra text to match against (ids, emails, raw keys). */
        keywords?: string[];
    }
</script>

<script lang="ts">
    import * as m from '#lib/paraglide/messages.js';

    /*
     * Searchable single-select following the WAI-ARIA combobox pattern
     * (https://www.w3.org/WAI/ARIA/apg/patterns/combobox/): focus stays in the
     * input, Arrow keys move the active option, Enter selects, Escape closes and
     * restores the current selection. Typing never changes `value` by itself.
     */
    interface Props {
        label: string;
        /** Keep the label for screen readers only, e.g. in a toolbar where the placeholder says enough. */
        hideLabel?: boolean;
        options: ComboboxOption[];
        value?: string;
        placeholder?: string;
        /** First option, selects the empty value ("All …"). */
        allLabel: string;
        emptyLabel: string;
        clearLabel?: string;
        /** Maximum matches shown at once. */
        limit?: number;
        class?: string;
    }

    let {
        label,
        hideLabel = false,
        options,
        value = $bindable(''),
        placeholder,
        allLabel,
        emptyLabel,
        clearLabel,
        limit = 10,
        class: extraClass = ''
    }: Props = $props();

    const id = $props.id();
    const listId = `${id}-listbox`;

    let open = $state(false);
    let query = $state('');
    /** True once the user has typed since the list opened. */
    let searching = $state(false);
    let activeIndex = $state(0);

    const selected = $derived(options.find((o) => o.value === value));
    const selectedLabel = $derived(selected?.label ?? value);

    // Keep the input text in sync with the selection when it changes from outside.
    $effect(() => {
        if (!searching) query = selectedLabel;
    });

    const matches = $derived.by(() => {
        const q = searching ? query.trim().toLowerCase() : '';
        const found = q
            ? options.filter((o) =>
                  [o.label, o.value, o.hint ?? '', ...(o.keywords ?? [])].some((text) => text.toLowerCase().includes(q))
              )
            : options;
        return found.slice(0, limit);
    });

    /** Index 0 is the "all" entry; matches follow. */
    const entries = $derived([{ value: '', label: allLabel }, ...matches]);

    function optionId(index: number) {
        return `${id}-option-${index}`;
    }

    function openList() {
        if (open) return;
        open = true;
        activeIndex = Math.max(0, entries.findIndex((e) => e.value === value));
    }

    function close() {
        open = false;
        searching = false;
        query = selectedLabel;
    }

    function choose(next: string) {
        value = next;
        searching = false;
        open = false;
        query = options.find((o) => o.value === next)?.label ?? next;
    }

    function onkeydown(event: KeyboardEvent) {
        switch (event.key) {
            case 'ArrowDown':
                event.preventDefault();
                if (!open) openList();
                else activeIndex = Math.min(activeIndex + 1, entries.length - 1);
                break;
            case 'ArrowUp':
                event.preventDefault();
                if (!open) openList();
                else activeIndex = Math.max(activeIndex - 1, 0);
                break;
            case 'Enter':
                if (open) {
                    event.preventDefault();
                    choose(entries[activeIndex]?.value ?? value);
                }
                break;
            case 'Escape':
                // First Escape closes the list; a second one clears the selection.
                if (open) {
                    event.preventDefault();
                    close();
                } else if (value) {
                    event.preventDefault();
                    choose('');
                }
                break;
        }
    }

    function oninput() {
        searching = true;
        open = true;
        activeIndex = matches.length > 0 ? 1 : 0;
    }

    function scrollActiveIntoView(node: HTMLElement, active: boolean) {
        const run = (isActive: boolean) => {
            if (isActive) node.scrollIntoView({ block: 'nearest' });
        };
        run(active);
        return { update: run };
    }
</script>

<div class="form-control w-full {extraClass}">
    <label class="label {hideLabel ? 'sr-only' : ''}" for={id}><span class="label-text text-xs">{label}</span></label>
    <div class="relative">
        <input
            {id}
            class="input w-full {value ? 'pr-16' : ''}"
            role="combobox"
            autocomplete="off"
            aria-autocomplete="list"
            aria-expanded={open}
            aria-controls={listId}
            aria-activedescendant={open ? optionId(activeIndex) : undefined}
            {placeholder}
            bind:value={query}
            onfocus={openList}
            onclick={openList}
            onblur={close}
            {oninput}
            {onkeydown}
        />

        {#if value}
            <button
                type="button"
                class="btn btn-ghost btn-xs absolute right-2 top-2"
                aria-label={clearLabel ?? m.common_clear()}
                onclick={() => choose('')}
            >
                {m.common_clear()}
            </button>
        {/if}

        <ul
            id={listId}
            role="listbox"
            aria-label={label}
            class="menu absolute z-20 mt-1 max-h-72 w-full flex-nowrap overflow-y-auto rounded-box border border-base-300 bg-base-100 shadow-overlay"
            hidden={!open}
        >
            {#each entries as entry, index (entry.value)}
                <!-- Keyboard selection is handled on the input (aria-activedescendant);
                     mousedown is prevented so the input keeps focus and blur does not close the list first. -->
                <!-- svelte-ignore a11y_click_events_have_key_events -->
                <li
                    id={optionId(index)}
                    role="option"
                    aria-selected={entry.value === value}
                    title={entry.label}
                    use:scrollActiveIntoView={open && index === activeIndex}
                    onmousedown={(e) => e.preventDefault()}
                    onclick={() => choose(entry.value)}
                >
                    <span
                        class="flex w-full items-center gap-2 rounded-field text-sm {entry.value === value ? 'bg-primary/10' : ''} {index === activeIndex ? 'outline-2 -outline-offset-2 outline-primary' : ''}"
                    >
                        <span class="truncate">{entry.label}</span>
                        {#if 'hint' in entry && entry.hint}
                            <span class="ml-auto shrink-0 text-xs text-base-content/70">{entry.hint}</span>
                        {/if}
                    </span>
                </li>
            {/each}
            {#if matches.length === 0}
                <li role="option" aria-disabled="true" aria-selected="false">
                    <span class="pointer-events-none text-sm text-base-content/70">{emptyLabel}</span>
                </li>
            {/if}
        </ul>
    </div>
</div>
