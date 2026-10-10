<script lang="ts">
    import { resolve } from '$app/paths';
    import { page } from '$app/state';
    import * as m from '$lib/paraglide/messages.js';

    let { children } = $props();

    const pathname = $derived(page.url.pathname);
    const general = $derived(pathname === '/admin/settings');
    const email = $derived(
        pathname === '/admin/settings/email' || pathname.startsWith('/admin/settings/email/')
    );
    const integrations = $derived(
        pathname === '/admin/settings/integrations' ||
            pathname.startsWith('/admin/settings/integrations/')
    );
    const backup = $derived(
        pathname === '/admin/settings/backup' || pathname.startsWith('/admin/settings/backup/')
    );
</script>

<div class="tabs tabs-lift w-full">
    <a
        href={resolve('/admin/settings')}
        class="tab"
        class:tab-active={general}
        role="tab"
        aria-current={general ? 'page' : undefined}
    >
        {m.settings_general()}
    </a>
    <a
        href={resolve('/admin/settings/email')}
        class="tab"
        class:tab-active={email}
        role="tab"
        aria-current={email ? 'page' : undefined}
    >
        {m.settings_email()}
    </a>
    <a
        href={resolve('/admin/settings/integrations')}
        class="tab"
        class:tab-active={integrations}
        role="tab"
        aria-current={integrations ? 'page' : undefined}
    >
        {m.settings_integrations()}
    </a>
    <a
        href={resolve('/admin/settings/backup')}
        class="tab"
        class:tab-active={backup}
        role="tab"
        aria-current={backup ? 'page' : undefined}
    >
        {m.nav_backup()}
    </a>
    <div class="tab-content border-base-300 bg-base-100 !block settings-tab-panel">
        {@render children()}
    </div>
</div>

<style>
    /*
      First card merges into the lift panel. Later cards stay inset and bordered
      so Email / Integrations / Backup keep clear section grouping.
    */
    .settings-tab-panel :global(> .flex) {
        gap: 1.5rem;
        padding: 0 1rem 1rem;
    }

    .settings-tab-panel :global(> .flex > .card:first-child) {
        border-radius: 0;
        border-width: 0;
        background-color: transparent;
        box-shadow: none;
        margin-inline: -1rem;
    }

    .settings-tab-panel :global(> .flex > .card:not(:first-child)) {
        border-radius: var(--radius-box);
        border-width: var(--border);
        border-style: solid;
        border-color: var(--color-base-300);
        background-color: var(--color-base-200);
        box-shadow: none;
    }

    /*
      A default btn fills with base-200, the colour of these cards, and with --depth 0
      its border is that fill too, so Save would read as plain text. Give default
      buttons paper and a hairline here; ghost, primary and error keep their own look.
    */
    .settings-tab-panel
        :global(> .flex > .card:not(:first-child) .btn:not(.btn-ghost, .btn-primary, .btn-error, .btn-link)) {
        --btn-color: var(--color-base-100);
        --btn-border: var(--color-base-300);
    }
</style>
