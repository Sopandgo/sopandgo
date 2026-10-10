<script lang="ts">
    import { getLocale, locales, setLocale } from '$lib/paraglide/runtime';

    let {
        name = 'locale',
        value = undefined,
        mode = 'field',
        id = 'locale',
        disabled = false
    }: {
        name?: string;
        value?: string;
        mode?: 'field' | 'cookie';
        id?: string;
        disabled?: boolean;
    } = $props();

    const current = $derived(value ?? getLocale());

    function languageName(tag: string) {
        try {
            const label = new Intl.DisplayNames([tag], { type: 'language' }).of(tag) ?? tag;
            return label.charAt(0).toLocaleUpperCase(tag) + label.slice(1);
        } catch {
            return tag;
        }
    }

    const options = $derived(
        locales
            .map((tag) => ({ tag, label: languageName(tag) }))
            .sort((a, b) => a.label.localeCompare(b.label, getLocale(), { sensitivity: 'base' }))
    );

    function onChange(event: Event) {
        if (mode !== 'cookie') return;
        const tag = (event.currentTarget as HTMLSelectElement).value;
        if (locales.includes(tag as (typeof locales)[number])) {
            setLocale(tag as (typeof locales)[number]);
        }
    }
</script>

<select
    {id}
    name={mode === 'field' ? name : undefined}
    class="select w-full"
    {disabled}
    onchange={onChange}
>
    {#each options as { tag, label } (tag)}
        <option value={tag} selected={tag === current}>{label}</option>
    {/each}
</select>
