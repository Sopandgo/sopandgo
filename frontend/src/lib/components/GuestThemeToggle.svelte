<script lang="ts">
    import { MoonIcon, SunIcon } from '@lucide/svelte';
    import * as m from '#lib/paraglide/messages.js';
    import { daisyTheme, themeCookieAssignment } from '#lib/theme.js';

    function effectiveTheme(): 'light' | 'dark' {
        const name = document.documentElement.getAttribute('data-theme');
        if (name === 'sop-dark') return 'dark';
        if (name === 'sop-light') return 'light';
        return window.matchMedia('(prefers-color-scheme: dark)').matches ? 'dark' : 'light';
    }

    function toggleTheme() {
        const next = effectiveTheme() === 'dark' ? 'light' : 'dark';
        const name = daisyTheme(next);
        if (name) document.documentElement.setAttribute('data-theme', name);
        document.cookie = themeCookieAssignment(next, location.protocol === 'https:');
    }
</script>

<button
    type="button"
    class="btn btn-ghost btn-circle shrink-0"
    aria-label={m.theme_label()}
    title={m.theme_label()}
    onclick={toggleTheme}
>
    <span class="theme-icon-sun">
        <SunIcon size={16} aria-hidden="true" />
    </span>
    <span class="theme-icon-moon">
        <MoonIcon size={16} aria-hidden="true" />
    </span>
</button>

<style>
    .theme-icon-sun {
        display: none;
    }

    .theme-icon-moon {
        display: block;
    }

    :global(html[data-theme='sop-dark']) .theme-icon-sun {
        display: block;
    }

    :global(html[data-theme='sop-dark']) .theme-icon-moon {
        display: none;
    }

    @media (prefers-color-scheme: dark) {
        :global(html:not([data-theme])) .theme-icon-sun {
            display: block;
        }

        :global(html:not([data-theme])) .theme-icon-moon {
            display: none;
        }
    }
</style>
