<script lang="ts">
    import { enhance } from '$app/forms';
    import { resolve } from '$app/paths';
    import Alert from '$lib/components/Alert.svelte';
    import Avatar from '$lib/components/Avatar.svelte';
    import AvatarCropper from '$lib/components/AvatarCropper.svelte';
    import Card from '$lib/components/Card.svelte';
    import CardPageHeading from '$lib/components/CardPageHeading.svelte';
    import ChangePassword from '$lib/components/ChangePassword.svelte';
    import LocaleSelect from '$lib/components/LocaleSelect.svelte';
    import * as m from '$lib/paraglide/messages.js';
    import { applyTheme } from '$lib/theme';
    import {
        ImageIcon,
        KeyRoundIcon,
        LanguagesIcon,
        LogOutIcon,
        MonitorSmartphoneIcon,
        SettingsIcon,
        SunMoonIcon,
        Trash2Icon,
        UploadIcon,
        UserIcon
    } from 'lucide-svelte';

    let { data, form } = $props();

    const user = $derived(data.user);
    const mustChange = $derived(!!user.must_change_password);
    const localeError = $derived(form?.setLocale?.error);
    const themeError = $derived(form?.setTheme?.error);
    const signOutResult = $derived(form?.signOutOthers);
    let signingOut = $state(false);
    let selectedTheme = $derived(data.user.theme);

    const AVATAR_TYPES = ['image/jpeg', 'image/png', 'image/webp'];
    const MAX_AVATAR_BYTES = 5 * 1024 * 1024;
    const avatarError = $derived(form?.uploadAvatar?.error ?? form?.removeAvatar?.error);
    let fileInput = $state<HTMLInputElement>();
    let uploadForm = $state<HTMLFormElement>();
    let pendingFile = $state.raw<File | null>(null);
    let croppedBlob: Blob | null = null;
    let pickError = $state(false);
    let uploading = $state(false);
    let removing = $state(false);

    function onAvatarPicked(event: Event & { currentTarget: HTMLInputElement }) {
        const file = event.currentTarget.files?.[0];
        // Clear the input so choosing the same file again still fires `change`.
        event.currentTarget.value = '';
        if (!file) return;
        if (!AVATAR_TYPES.includes(file.type) || file.size > MAX_AVATAR_BYTES) {
            pickError = true;
            return;
        }
        pickError = false;
        pendingFile = file;
    }

    function onCropConfirmed(blob: Blob) {
        croppedBlob = blob;
        uploadForm?.requestSubmit();
    }

    function cancelCrop() {
        pendingFile = null;
        croppedBlob = null;
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
            <div class="flex flex-wrap items-start justify-between gap-3">
                <CardPageHeading>
                    <SettingsIcon class="h-8 w-8" />
                    {m.settings_heading()}
                </CardPageHeading>
                <a href={resolve('/profile')} class="btn">
                    <UserIcon class="size-4" />
                    {m.nav_profile()}
                </a>
            </div>
            <p class="text-sm font-medium text-base-content/70">{m.settings_intro()}</p>
        </div>
    </Card>

    {#snippet passwordCard()}
        <Card>
            <div class="card-body">
                <h2 class="mb-4 flex items-center gap-2 text-lg font-semibold">
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
            <h2 class="flex items-center gap-2 text-lg font-semibold">
                <ImageIcon class="h-4 w-4" />
                {m.settings_avatar()}
            </h2>
            {#if pendingFile}
                <!-- The cropped square is attached in `enhance`; the form has no file input of its own. -->
                <form
                    bind:this={uploadForm}
                    method="POST"
                    action="?/uploadAvatar"
                    enctype="multipart/form-data"
                    use:enhance={({ formData, cancel }) => {
                        if (!croppedBlob) {
                            cancel();
                            return;
                        }
                        formData.set('file', croppedBlob, 'avatar.jpg');
                        uploading = true;
                        return async ({ result, update }) => {
                            await update();
                            uploading = false;
                            if (result.type === 'success') cancelCrop();
                        };
                    }}
                >
                    {#key pendingFile}
                        <AvatarCropper
                            file={pendingFile}
                            busy={uploading}
                            onconfirm={onCropConfirmed}
                            oncancel={cancelCrop}
                        />
                    {/key}
                </form>
            {:else}
                <p class="text-sm text-base-content/70">{m.settings_avatar_intro()}</p>
                <div class="flex flex-wrap items-center gap-4">
                    <Avatar
                        displayName={user.display_name}
                        size="lg"
                        userId={user.id}
                        hasAvatar={user.has_avatar}
                        avatarContentHash={user.avatar_content_hash}
                    />
                    <div class="flex flex-wrap items-center gap-2">
                        <input
                            bind:this={fileInput}
                            type="file"
                            accept={AVATAR_TYPES.join(',')}
                            class="hidden"
                            tabindex="-1"
                            onchange={onAvatarPicked}
                        />
                        <button type="button" class="btn" onclick={() => fileInput?.click()}>
                            <UploadIcon class="size-4" />
                            {m.settings_avatar_change()}
                        </button>
                        {#if user.has_avatar}
                            <form
                                method="POST"
                                action="?/removeAvatar"
                                use:enhance={() => {
                                    removing = true;
                                    return async ({ update }) => {
                                        await update();
                                        removing = false;
                                    };
                                }}
                            >
                                <button type="submit" class="btn" disabled={removing}>
                                    {#if removing}
                                        <span class="loading loading-spinner"></span>
                                    {:else}
                                        <Trash2Icon class="size-4" />
                                    {/if}
                                    {m.settings_avatar_remove()}
                                </button>
                            </form>
                        {/if}
                    </div>
                </div>
            {/if}
            {#if pickError || avatarError}
                <Alert type="error" message={m.settings_avatar_error()} />
            {/if}
        </div>
    </Card>

    <Card>
        <div class="card-body gap-4">
            <h2 class="flex items-center gap-2 text-lg font-semibold">
                <LanguagesIcon class="h-4 w-4" />
                {m.profile_language()}
            </h2>
            <!-- Saves on change like Appearance. Not enhanced: the full reload renders the page in the new language. -->
            <form
                method="POST"
                action="?/setLocale"
                class="flex flex-col gap-3"
                onchange={(event) => event.currentTarget.requestSubmit()}
            >
                <label class="sr-only" for="profile-locale">{m.locale_label()}</label>
                <LocaleSelect id="profile-locale" value={user.locale} />
                {#if localeError}
                    <Alert type="error" message={m.locale_failed()} />
                {/if}
            </form>
        </div>
    </Card>

    <Card>
        <div class="card-body gap-4">
            <h2 class="flex items-center gap-2 text-lg font-semibold">
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
                        <label class="label cursor-pointer justify-start gap-3 py-1">
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

        <div class="flex flex-col gap-6 {mustChange ? 'order-1 lg:order-2' : ''}">
            {@render passwordCard()}

            {#if !mustChange}
                <Card>
                    <div class="card-body gap-4">
                        <h2 class="flex items-center gap-2 text-lg font-semibold">
                            <MonitorSmartphoneIcon class="h-4 w-4" />
                            {m.sessions_heading()}
                        </h2>
                        <p class="text-sm text-base-content/70">{m.sessions_help()}</p>
                        {#if signOutResult?.ok}
                            <Alert type="success" message={m.sessions_signed_out()} />
                        {:else if signOutResult?.error}
                            <Alert type="error" message={m.sessions_sign_out_failed()} />
                        {/if}
                        <form
                            method="POST"
                            action="?/signOutOthers"
                            use:enhance={() => {
                                signingOut = true;
                                return async ({ update }) => {
                                    await update();
                                    signingOut = false;
                                };
                            }}
                        >
                            <button type="submit" class="btn w-full" disabled={signingOut}>
                                {#if signingOut}
                                    <span class="loading loading-spinner"></span>
                                {:else}
                                    <LogOutIcon class="size-4" />
                                {/if}
                                {m.sessions_sign_out_others()}
                            </button>
                        </form>
                    </div>
                </Card>
            {/if}
        </div>
    </div>
</div>
