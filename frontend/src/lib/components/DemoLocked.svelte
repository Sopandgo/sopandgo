<script lang="ts">
    import type { Snippet } from 'svelte';
    import { ArrowRightIcon } from '@lucide/svelte';
    import { DEMO_INSTALL_URL } from '#lib/demo.js';
    import * as m from '#lib/paraglide/messages.js';

    /*
     * Wraps write controls on a read-only demo. Everything inside is disabled
     * through a disabled fieldset, and hovering or focusing the wrapper shows
     * why, with a link to the install instructions. The API refuses these
     * writes on its own; this only explains it.
     */
    interface Props {
        children: Snippet;
        /** Full-width wrapper for a whole form instead of one inline control. */
        block?: boolean;
    }

    let { children, block = false }: Props = $props();
    const tipId = `demo-locked-${Math.random().toString(36).slice(2, 10)}`;
</script>

<div class="dropdown dropdown-hover {block ? 'dropdown-top block w-full' : 'dropdown-end inline-block'}">
    <!-- Disabled controls cannot take focus, so the wrapper does, for keyboard users. -->
    <div
        tabindex="0"
        role="button"
        aria-disabled="true"
        aria-describedby={tipId}
        class="cursor-not-allowed rounded-field"
    >
        <fieldset disabled class="contents">
            {@render children()}
        </fieldset>
    </div>
    <div
        id={tipId}
        role="tooltip"
        tabindex="-1"
        class="dropdown-content z-20 w-64 rounded-box border border-base-300 bg-base-100 p-3 text-sm shadow-overlay"
    >
        <p>{m.demo_disabled()}</p>
        <a
            class="link mt-2 inline-flex items-center gap-1 font-medium"
            href={DEMO_INSTALL_URL}
            target="_blank"
            rel="noopener noreferrer"
        >
            {m.demo_install()}
            <ArrowRightIcon size={16} aria-hidden="true" />
        </a>
    </div>
</div>
