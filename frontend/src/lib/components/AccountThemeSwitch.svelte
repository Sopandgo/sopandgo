<script lang="ts">
    import { enhance } from '$app/forms';
    import { invalidateAll } from '$app/navigation';
    import { MonitorIcon, MoonIcon, SunIcon } from 'lucide-svelte';
    import * as m from '$lib/paraglide/messages.js';
    import { applyTheme, isThemePreference, type ThemePreference } from '$lib/theme';

    let { theme }: { theme: ThemePreference } = $props();

    // Follows the saved value, but switches at once on click.
    let selected = $derived(theme);

    const options = [
        { value: 'light', label: m.theme_light, icon: SunIcon },
        { value: 'dark', label: m.theme_dark, icon: MoonIcon },
        { value: 'system', label: m.theme_system, icon: MonitorIcon }
    ] as const;
</script>

<!-- Saves to the account through the settings action, so it applies on every device. -->
<form
    method="POST"
    action="/profile/settings?/setTheme"
    class="flex justify-center px-2.5 py-1"
    use:enhance={({ submitter, cancel }) => {
        const next = submitter instanceof HTMLButtonElement ? submitter.value : '';
        if (!isThemePreference(next) || next === selected) {
            cancel();
            return;
        }
        const previous = selected;
        selected = next;
        applyTheme(next);
        return async ({ result }) => {
            if (result.type === 'success') {
                await invalidateAll();
            } else {
                selected = previous;
                applyTheme(previous);
            }
        };
    }}
>
    <div class="join" role="group" aria-label={m.theme_label()}>
        {#each options as option (option.value)}
            <button
                type="submit"
                name="theme"
                value={option.value}
                class="btn btn-xs btn-square join-item"
                class:btn-active={selected === option.value}
                aria-pressed={selected === option.value}
                aria-label={option.label()}
                title={option.label()}
            >
                <option.icon size={16} aria-hidden="true" />
            </button>
        {/each}
    </div>
</form>
