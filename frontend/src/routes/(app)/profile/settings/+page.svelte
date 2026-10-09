<script lang="ts">
    import { enhance } from '$app/forms';
    import Alert from '$lib/components/Alert.svelte';
    import Card from '$lib/components/Card.svelte';
    import CardPageHeading from '$lib/components/CardPageHeading.svelte';
    import ChangePassword from '$lib/components/ChangePassword.svelte';
    import LocaleSelect from '$lib/components/LocaleSelect.svelte';
    import * as m from '$lib/paraglide/messages.js';
    import { daisyTheme } from '$lib/theme';
    import { KeyRoundIcon, LanguagesIcon, SunMoonIcon } from 'lucide-svelte';

    let { data, form } = $props();

    const user = $derived(data.user);
    const mustChange = $derived(!!user.must_change_password);
    const localeError = $derived(form?.setLocale?.error);
    const themeError = $derived(form?.setTheme?.error);
    let selectedTheme = $derived(data.user.theme);

    function applyTheme(theme: string) {
        const name = daisyTheme(theme);
        if (name) document.documentElement.setAttribute('data-theme', name);
        else document.documentElement.removeAttribute('data-theme');
    }

    function onThemeChange(event: Event) {
        const formEl = event.currentTarget;
        if (!(formEl instanceof HTMLFormElement)) return;
        const theme = String(new FormData(formEl).get('theme') ?? '');
        if (theme !== 'light' && theme !== 'dark' && theme !== 'system') return;
        selectedTheme = theme;
        applyTheme(theme);
        formEl.requestSubmit();
    }
</script>

<svelte:head>
    <title>{m.page_account_settings()}</title>
</svelte:head>

<div class="flex w-full flex-col gap-6">
    <Card>
        <div class="card-body">
            <CardPageHeading>
                <SunMoonIcon class="h-8 w-8" />
                {m.settings_heading()}
            </CardPageHeading>
            <p class="text-sm font-medium text-base-content/70">{m.settings_intro()}</p>
        </div>
    </Card>

    {#snippet passwordCard()}
        <Card>
            <div class="card-body">
                <h2 class="text-lg font-semibold mb-4">
                    <KeyRoundIcon class="h-4 w-4" />
                    {m.profile_change_password()}
                </h2>
                <ChangePassword {form} showLogoutWarning={true} />
            </div>
        </Card>
    {/snippet}

    {#if mustChange}
        <Alert type="warning" message={m.profile_bootstrap_warning()} />
    {/if}

    <div class="grid grid-cols-1 items-start gap-6 lg:grid-cols-2">
        <div class="flex flex-col gap-6 {mustChange ? 'order-2 lg:order-1' : ''}">
    <Card>
        <div class="card-body gap-4">
            <h2 class="text-lg font-semibold">
                <LanguagesIcon class="h-4 w-4" />
                {m.profile_language()}
            </h2>
            <form method="POST" action="?/setLocale" class="flex flex-col gap-3">
                <label class="form-control w-full" for="profile-locale">
                    <span class="label-text">{m.locale_label()}</span>
                </label>
                <LocaleSelect id="profile-locale" value={user.locale} />
                {#if localeError}
                    <Alert type="error" message={m.locale_failed()} />
                {/if}
                <button type="submit" class="btn btn-primary w-full sm:w-auto">{m.locale_save()}</button>
            </form>
        </div>
    </Card>

    <Card>
        <div class="card-body gap-4">
            <h2 class="text-lg font-semibold">
                <SunMoonIcon class="h-4 w-4" />
                {m.theme_label()}
            </h2>
            <form
                method="POST"
                action="?/setTheme"
                onchange={onThemeChange}
                use:enhance={() => {
                    return async ({ result, update }) => {
                        await update();
                        if (result.type === 'failure') {
                            selectedTheme = user.theme;
                            applyTheme(user.theme);
                        }
                    };
                }}
            >
                <fieldset class="flex flex-col gap-2">
                    <legend class="sr-only">{m.theme_label()}</legend>
                    {#each [
                        { value: 'light', label: m.theme_light() },
                        { value: 'dark', label: m.theme_dark() },
                        { value: 'system', label: m.theme_system() }
                    ] as option (option.value)}
                        <label class="label cursor-pointer justify-start gap-3 rounded-box border border-base-300 px-3 py-2">
                            <input
                                type="radio"
                                name="theme"
                                value={option.value}
                                class="radio"
                                checked={selectedTheme === option.value}
                            />
                            <span>{option.label}</span>
                        </label>
                    {/each}
                </fieldset>
                {#if themeError}
                    <Alert type="error" message={m.theme_failed()} />
                {/if}
            </form>
        </div>
    </Card>
        </div>

        <div class={mustChange ? 'order-1 lg:order-2' : ''}>
            {@render passwordCard()}
        </div>
    </div>
</div>
