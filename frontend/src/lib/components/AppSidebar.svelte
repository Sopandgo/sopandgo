<script lang="ts">
    import { resolve } from '$app/paths';
    import { page } from '$app/state';
    import {
        CodeIcon,
        HouseIcon,
        LogsIcon,
        NotebookIcon,
        SettingsIcon,
        ShieldCheckIcon,
        ShieldUserIcon,
        UsersIcon
    } from '@lucide/svelte';
    import BrandLogo from '#lib/components/BrandLogo.svelte';
    import { brandName, DEFAULT_BRAND_NAME, isCustomBranding } from '#lib/branding.js';
    import * as m from '#lib/paraglide/messages.js';
    import { PRODUCT_URL, versionLabel, versionUrl } from '#lib/product.js';
    import type { User } from '#lib/sdk/types.js';

    let { user }: { user: User } = $props();

    type RailIcon = typeof HouseIcon;

    const version = versionLabel(__APP_VERSION__);
    const versionHref = versionUrl(__APP_VERSION__);

    const name = $derived(brandName(page.data.branding));
    const branded = $derived(isCustomBranding(page.data.branding));
    // Branded installs credit the product; unbranded ones name it. Both link to the website.
    const productLabel = $derived(
        branded ? m.footer_powered_by({ product: DEFAULT_BRAND_NAME }) : DEFAULT_BRAND_NAME
    );

    const pathname = $derived(page.url.pathname);
    const dashboardCurrent = $derived(pathname === '/dashboard');
    const sopsCurrent = $derived(pathname === '/sops' || pathname.startsWith('/sops/'));

    function sectionCurrent(href: string) {
        return pathname === href || pathname.startsWith(`${href}/`);
    }
</script>

<div
    class="rail flex min-h-dvh flex-col items-start border-r border-base-300 bg-base-100 is-drawer-close:w-14 is-drawer-open:w-64"
>
    <a
        href={resolve('dashboard')}
        class="flex h-16 w-full shrink-0 items-center overflow-hidden is-drawer-close:tooltip is-drawer-close:tooltip-right"
        data-tip={name}
        aria-label={name}
    >
        <span class="flex size-14 shrink-0 items-center justify-center">
            <BrandLogo branding={page.data.branding} class="size-8" />
        </span>
        <span class="is-drawer-close:hidden min-w-0 truncate pe-3 font-semibold">{name}</span>
    </a>
    <ul class="menu w-full">
        {@render item('dashboard', m.page_dashboard(), dashboardCurrent, HouseIcon)}
        {@render item('sops', m.page_sops(), sopsCurrent, NotebookIcon)}
    </ul>
    <div class="mt-auto w-full">
        {#if user.role === 'admin'}
            <ul class="menu w-full">
                {@render rule(m.page_admin())}
                {@render item('admin/users', m.nav_users(), sectionCurrent('/admin/users'), UsersIcon)}
                {@render item(
                    'admin/sessions',
                    m.nav_sessions(),
                    sectionCurrent('/admin/sessions'),
                    ShieldUserIcon
                )}
                {@render item(
                    'admin/audit-logs',
                    m.nav_audit_logs(),
                    sectionCurrent('/admin/audit-logs'),
                    LogsIcon
                )}
                {@render item(
                    'admin/integrity',
                    m.nav_integrity(),
                    sectionCurrent('/admin/integrity'),
                    ShieldCheckIcon
                )}
                {@render item(
                    'admin/settings',
                    m.nav_settings(),
                    sectionCurrent('/admin/settings'),
                    SettingsIcon
                )}
            </ul>
        {/if}
        <ul class="menu w-full">
            {@render rule(m.nav_about())}
            <!-- The product row stays when a lab brands the app (attribution). -->
            <li>
                <a
                    href={PRODUCT_URL}
                    target="_blank"
                    rel="noopener noreferrer"
                    class="is-drawer-close:tooltip is-drawer-close:tooltip-right"
                    data-tip={productLabel}
                    aria-label={productLabel}
                >
                    <img src="/favicon.svg" alt="" title={m.footer_gopher()} class="size-4 shrink-0" />
                    <span class="is-drawer-close:hidden truncate">{productLabel}</span>
                </a>
            </li>
            <li>
                <a
                    href={versionHref}
                    target="_blank"
                    rel="noopener noreferrer"
                    class="is-drawer-close:tooltip is-drawer-close:tooltip-right"
                    data-tip="{m.footer_github()} · {version}"
                    aria-label="{m.footer_github()} · {version}"
                >
                    <CodeIcon size={16} aria-hidden="true" />
                    <span class="is-drawer-close:hidden">{version}</span>
                </a>
            </li>
        </ul>
    </div>
</div>

{#snippet item(
    href:
        | 'dashboard'
        | 'sops'
        | 'admin/users'
        | 'admin/sessions'
        | 'admin/audit-logs'
        | 'admin/integrity'
        | 'admin/settings',
    label: string,
    current: boolean,
    Icon: RailIcon
)}
    <li>
        <a
            href={resolve(href)}
            class="is-drawer-close:tooltip is-drawer-close:tooltip-right {current ? 'bg-primary/10 font-medium text-primary' : ''}"
            data-tip={label}
            aria-label={label}
            aria-current={current ? 'page' : undefined}
        >
            <Icon size={16} aria-hidden="true" />
            <span class="is-drawer-close:hidden">{label}</span>
        </a>
    </li>
{/snippet}

{#snippet rule(label = '')}
    <li class="pointer-events-none">
        <div class="menu-title flex h-9 items-center gap-2">
            <span class="h-px min-w-2 flex-1 bg-base-300"></span>
            {#if label}
                <span class="is-drawer-close:hidden shrink-0 text-xs font-semibold">{label}</span>
                <span class="is-drawer-close:hidden h-px min-w-2 flex-1 bg-base-300"></span>
            {/if}
        </div>
    </li>
{/snippet}

<style>
    /* The collapsed tooltip class sets display:inline-block and would shrink these rows. */
    .rail > a,
    .rail :global(.menu > li > a) {
        display: flex;
        align-items: center;
    }

    .rail :global(.menu > li > a) {
        width: 100%;
        height: 2.25rem;
        gap: 0.5rem;
    }
</style>
