<script lang="ts">
    import { ArrowRightIcon, InfoIcon } from '@lucide/svelte';
    import { DEMO_INSTALL_URL, type DemoRole } from '#lib/demo.js';
    import * as m from '#lib/paraglide/messages.js';

    /*
     * Shown on every page of a read-only public demo: what it is, where to
     * install SOPandGO, and a switch between the researcher and auditor views.
     * The switch posts to /demo?/switch, which replaces the session.
     * The data-umami-event attributes count clicks when the demo loads Umami
     * (see $lib/analytics); without it they do nothing.
     */
    let { role, error = null }: { role: string | null; error?: string | null } = $props();

    const roles: { value: DemoRole; label: () => string }[] = [
        { value: 'viewer', label: m.demo_role_researcher },
        { value: 'auditor', label: m.demo_role_auditor }
    ];
</script>

<div
    role="region"
    aria-label={m.demo_region()}
    class="border-b border-info/40 bg-info/10 px-4 py-2 text-sm sm:px-6 lg:px-8"
>
    <div class="flex flex-wrap items-center gap-x-4 gap-y-2">
        <span class="inline-flex items-center gap-2">
            <InfoIcon size={16} class="shrink-0 text-info" aria-hidden="true" />
            {m.demo_banner()}
        </span>
        <a
            class="link inline-flex items-center gap-1 font-medium"
            href={DEMO_INSTALL_URL}
            target="_blank"
            rel="noopener noreferrer"
            data-umami-event="demo-install-click"
        >
            {m.demo_install()}
            <ArrowRightIcon size={16} aria-hidden="true" />
        </a>
        {#if role}
            <form method="POST" action="/demo?/switch" class="ms-auto flex items-center gap-2">
                <span id="demo-view-as">{m.demo_view_as()}</span>
                <div class="join" role="group" aria-labelledby="demo-view-as">
                    {#each roles as option (option.value)}
                        <button
                            type="submit"
                            name="role"
                            value={option.value}
                            class="btn btn-xs join-item"
                            class:btn-active={role === option.value}
                            aria-pressed={role === option.value}
                            disabled={role === option.value}
                            data-umami-event="demo-switch-role"
                            data-umami-event-to={option.value}
                        >
                            {option.label()}
                        </button>
                    {/each}
                </div>
            </form>
        {/if}
        {#if error}
            <span role="alert" class="w-full text-error">{error}</span>
        {/if}
    </div>
</div>
