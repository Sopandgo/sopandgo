<script lang="ts">
    import { enhance } from '$app/forms';
    import { page } from '$app/state';
    import Alert from '#lib/components/Alert.svelte';
    import BrandLogo from '#lib/components/BrandLogo.svelte';
    import Card from '#lib/components/Card.svelte';
    import CardPageHeading from '#lib/components/CardPageHeading.svelte';
    import ConfirmDialog from '#lib/components/ConfirmDialog.svelte';
    import {
        brandName,
        contrastRatio,
        DEFAULT_BRAND_NAME,
        isCustomBranding,
        isHexColour,
        MIN_CONTRAST,
        THEME_BACKGROUND
    } from '#lib/branding.js';
    import { PaletteIcon, RotateCcwIcon } from '@lucide/svelte';
    import * as m from '#lib/paraglide/messages.js';
    import type { Branding } from '#lib/sdk/types.js';

    let { form }: { form: any } = $props();

    const branding = $derived(page.data.branding as Branding);

    type Theme = 'light' | 'dark';

    // Form state starts from what is saved; a failed save keeps what was typed.
    // svelte-ignore state_referenced_locally
    let name = $state(form?.saveText?.input?.name ?? page.data.branding?.name ?? '');
    let colours = $state<Record<Theme, string>>({
        // svelte-ignore state_referenced_locally
        light: form?.saveText?.input?.primary_light ?? page.data.branding?.primary_light ?? '',
        // svelte-ignore state_referenced_locally
        dark: form?.saveText?.input?.primary_dark ?? page.data.branding?.primary_dark ?? ''
    });
    let saving = $state(false);
    let uploading = $state(false);
    let logoFile = $state<FileList | null>(null);
    let resetOpen = $state(false);
    let resetting = $state(false);

    /** Live check, so a colour the server will refuse is flagged before saving. */
    function colourProblem(theme: Theme): string | null {
        const value = colours[theme].trim();
        if (!value) return null;
        if (!isHexColour(value)) return m.branding_colour_invalid();
        if (contrastRatio(value, THEME_BACKGROUND[theme]) < MIN_CONTRAST) return m.branding_colour_contrast();
        return null;
    }

    const problems = $derived({ light: colourProblem('light'), dark: colourProblem('dark') });
    const canSave = $derived(!saving && !problems.light && !problems.dark);

    const themes: { theme: Theme; label: () => string; daisy: string }[] = [
        { theme: 'light', label: m.branding_colour_light, daisy: 'sop-light' },
        { theme: 'dark', label: m.branding_colour_dark, daisy: 'sop-dark' }
    ];
</script>

<svelte:head>
    <title>{m.settings_branding()}</title>
</svelte:head>

<div class="flex flex-col gap-6">
    <Card>
        <div class="card-body space-y-4">
            <CardPageHeading>
                <PaletteIcon class="w-8 h-8" />
                {m.settings_branding()}
            </CardPageHeading>
            <p class="text-sm text-base-content/70">
                {m.branding_intro({ product: DEFAULT_BRAND_NAME })}
            </p>
        </div>
    </Card>

    <Card title={m.branding_name_and_colour()}>
        <div class="card-body space-y-4">
            {#if form?.saveText?.error}
                <Alert type="error" message={form.saveText.error} />
            {:else if form?.saveText?.ok}
                <Alert type="success" message={m.branding_saved()} />
            {/if}
            <form
                method="POST"
                action="?/saveText"
                class="space-y-6"
                use:enhance={() => {
                    saving = true;
                    return async ({ update }) => {
                        await update({ reset: false });
                        saving = false;
                    };
                }}
            >
                <div class="flex flex-col gap-1.5">
                    <label class="text-sm font-medium" for="branding-name">{m.branding_name_label()}</label>
                    <input
                        id="branding-name"
                        name="name"
                        type="text"
                        class="input w-full sm:max-w-md"
                        maxlength="40"
                        placeholder={DEFAULT_BRAND_NAME}
                        bind:value={name}
                        aria-describedby="branding-name-help"
                    />
                    <p id="branding-name-help" class="text-sm text-base-content/70">{m.branding_name_help()}</p>
                </div>

                <fieldset class="space-y-3">
                    <legend class="text-sm font-medium">{m.branding_colour_label()}</legend>
                    <p class="text-sm text-base-content/70">{m.branding_colour_help()}</p>
                    <div class="grid grid-cols-1 gap-4 md:grid-cols-2">
                        {#each themes as t (t.theme)}
                            {@const value = colours[t.theme].trim()}
                            {@const valid = isHexColour(value) && !problems[t.theme]}
                            <div class="space-y-2">
                                <label class="text-sm" for="branding-{t.theme}">{t.label()}</label>
                                <div class="flex items-center gap-2">
                                    <input
                                        type="color"
                                        class="h-10 w-12 shrink-0 cursor-pointer rounded-field border border-base-300 bg-base-100 p-1"
                                        aria-label={t.label()}
                                        title={t.label()}
                                        value={isHexColour(value) ? value.toLowerCase() : THEME_BACKGROUND[t.theme === 'light' ? 'dark' : 'light']}
                                        oninput={(e) => (colours[t.theme] = e.currentTarget.value)}
                                    />
                                    <input
                                        id="branding-{t.theme}"
                                        name="primary_{t.theme}"
                                        type="text"
                                        class="input w-full font-mono text-sm {problems[t.theme] ? 'input-error' : ''}"
                                        placeholder={m.branding_colour_default()}
                                        bind:value={colours[t.theme]}
                                        aria-invalid={problems[t.theme] ? 'true' : undefined}
                                        aria-describedby={problems[t.theme] ? `branding-${t.theme}-error` : undefined}
                                    />
                                    {#if value}
                                        <button
                                            type="button"
                                            class="btn btn-ghost shrink-0"
                                            onclick={() => (colours[t.theme] = '')}
                                        >
                                            {m.branding_colour_reset()}
                                        </button>
                                    {/if}
                                </div>
                                {#if problems[t.theme]}
                                    <p id="branding-{t.theme}-error" class="text-sm text-error">{problems[t.theme]}</p>
                                {/if}
                                <!-- Preview in the real theme, with the colour applied only to this box. -->
                                <div
                                    data-theme={t.daisy}
                                    class="flex flex-wrap items-center gap-3 rounded-box border border-base-300 bg-base-100 p-3 text-base-content"
                                    style={valid
                                        ? `--color-primary:${value};--color-primary-content:${THEME_BACKGROUND[t.theme]}`
                                        : undefined}
                                >
                                    <span class="btn btn-primary btn-sm pointer-events-none" aria-hidden="true">
                                        {m.branding_preview_button()}
                                    </span>
                                    <span class="text-sm font-medium text-primary">{m.branding_preview_link()}</span>
                                </div>
                            </div>
                        {/each}
                    </div>
                </fieldset>

                <div class="flex flex-col sm:flex-row sm:justify-end">
                    <button type="submit" class="btn btn-primary w-full sm:w-auto" disabled={!canSave}>
                        {#if saving}<span class="loading loading-spinner loading-xs"></span>{/if}
                        {m.branding_save()}
                    </button>
                </div>
            </form>
        </div>
    </Card>

    <Card title={m.branding_logo_label()} description={m.branding_logo_help()}>
        <div class="card-body space-y-4">
            {#if form?.logo?.error}
                <Alert type="error" message={form.logo.error} />
            {:else if form?.logo?.ok}
                <Alert type="success" message={m.branding_saved()} />
            {/if}

            <div class="grid grid-cols-1 gap-4 sm:grid-cols-2">
                {#each themes as t (t.theme)}
                    <div
                        data-theme={t.daisy}
                        class="flex min-w-0 items-center gap-4 rounded-box border border-base-300 bg-base-100 p-4 text-base-content"
                    >
                        <BrandLogo {branding} alt={m.footer_logo_alt({ name: brandName(branding) })} class="size-8 shrink-0" />
                        <span class="min-w-0 truncate font-semibold">{brandName(branding)}</span>
                    </div>
                {/each}
            </div>

            <div class="flex flex-col gap-3 sm:flex-row sm:items-end sm:justify-between">
                <form
                    method="POST"
                    action="?/uploadLogo"
                    enctype="multipart/form-data"
                    class="flex flex-col gap-3 sm:flex-row sm:items-end"
                    use:enhance={() => {
                        uploading = true;
                        return async ({ update }) => {
                            await update();
                            uploading = false;
                            logoFile = null;
                        };
                    }}
                >
                    <label class="flex flex-col gap-1.5" for="branding-logo">
                        <span class="text-sm font-medium">{m.branding_logo_choose()}</span>
                        <input
                            id="branding-logo"
                            name="file"
                            type="file"
                            accept="image/png,image/jpeg,image/webp"
                            class="file-input w-full sm:w-auto"
                            bind:files={logoFile}
                            required
                        />
                    </label>
                    <button type="submit" class="btn" disabled={uploading || !logoFile?.length}>
                        {#if uploading}<span class="loading loading-spinner loading-xs"></span>{/if}
                        {m.branding_logo_upload()}
                    </button>
                </form>

                {#if branding?.has_logo}
                    <form method="POST" action="?/removeLogo" use:enhance>
                        <button type="submit" class="btn btn-ghost w-full sm:w-auto">{m.branding_logo_remove()}</button>
                    </form>
                {/if}
            </div>
        </div>
    </Card>

    <Card title={m.branding_reset_heading()} description={m.branding_reset_help({ product: DEFAULT_BRAND_NAME })}>
        <div class="card-body space-y-4">
            {#if form?.reset?.error}
                <Alert type="error" message={form.reset.error} />
            {:else if form?.reset?.ok}
                <Alert type="success" message={m.branding_reset_done()} />
            {/if}
            <form
                method="POST"
                action="?/reset"
                use:enhance={() => {
                    resetting = true;
                    return async ({ result, update }) => {
                        await update();
                        resetting = false;
                        resetOpen = false;
                        if (result.type === 'success') {
                            // The saved values are gone; clear what the fields above still show.
                            name = '';
                            colours = { light: '', dark: '' };
                        }
                    };
                }}
            >
                <button
                    type="button"
                    class="btn btn-ghost text-error w-full sm:w-auto"
                    disabled={!isCustomBranding(branding)}
                    onclick={() => (resetOpen = true)}
                >
                    <RotateCcwIcon size={16} aria-hidden="true" />
                    {m.branding_reset_button()}
                </button>
                <ConfirmDialog title={m.branding_reset_confirm_title()} bind:open={resetOpen} busy={resetting}>
                    <p>{m.branding_reset_confirm_body({ product: DEFAULT_BRAND_NAME })}</p>
                    {#snippet confirm()}
                        <button type="submit" class="btn btn-error" disabled={resetting}>
                            {#if resetting}<span class="loading loading-spinner loading-xs"></span>{/if}
                            {m.branding_reset_confirm()}
                        </button>
                    {/snippet}
                </ConfirmDialog>
            </form>
        </div>
    </Card>
</div>
