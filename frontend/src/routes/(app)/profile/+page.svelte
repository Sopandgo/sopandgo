<script lang="ts">
    import Alert from "$lib/components/Alert.svelte";
    import Card from "$lib/components/Card.svelte";
    import CardPageHeading from "$lib/components/CardPageHeading.svelte";
    import ChangePassword from "$lib/components/ChangePassword.svelte";
    import ListUserSignatures from "$lib/components/ListUserSignatures.svelte";
    import LocaleSelect from "$lib/components/LocaleSelect.svelte";
    import * as m from "$lib/paraglide/messages.js";
    import { getLocale } from "$lib/paraglide/runtime";
    import type { UserRole } from "$lib/sdk/types";
    import {
        CalendarClockIcon,
        IdCardIcon,
        LanguagesIcon,
        MailIcon,
        StampIcon,
        UserIcon,
    } from "lucide-svelte";
    let { data, form } = $props();

    const user = $derived(data.user);
    const mustChange = $derived(!!user?.must_change_password);
    const localeError = $derived(form?.setLocale?.error);

    function roleLabel(role: UserRole) {
        switch (role) {
            case 'admin':
                return m.role_admin();
            case 'editor':
                return m.role_editor();
            case 'approver':
                return m.role_approver();
            case 'auditor':
                return m.role_auditor();
            case 'viewer':
                return m.role_viewer();
            default:
                return role;
        }
    }
</script>

<svelte:head>
    <title>{m.page_profile()}</title>
</svelte:head>

{#if mustChange}
    <Card>
        <div class="card-body gap-4">
            <CardPageHeading>
                <UserIcon class="w-8 h-8" />
                {m.profile_change_password()}
            </CardPageHeading>
            <Alert
                variant="warning"
                message={m.profile_bootstrap_warning()}
            />
            <ChangePassword {form} showLogoutWarning={true} />
            <form method="POST" action="?/setLocale" class="flex flex-col gap-3">
                <label class="form-control w-full" for="locale">
                    <span class="label-text">{m.locale_label()}</span>
                </label>
                <LocaleSelect id="locale" value={user.locale} />
                {#if localeError}
                    <Alert variant="error" message={m.locale_failed()} />
                {/if}
                <button type="submit" class="btn btn-primary w-full sm:w-auto">{m.locale_save()}</button>
            </form>
        </div>
    </Card>
{:else}
    <Card>
        <div class="card-body">
            <CardPageHeading>
                <UserIcon class="w-8 h-8" />
                {m.profile_heading()}
            </CardPageHeading>

            <div class="flex items-center gap-2 text-base-content/70">
                <span class="text-sm font-medium"
                    >{m.profile_intro()}</span
                >
            </div>
        </div>
    </Card>

    <div class="flex flex-col lg:flex-row gap-6 mt-6">
        <ul class="list w-full gap-6">
            <Card>
                <li
                    class="p-4 pb-2 text-xs opacity-60 tracking-widest uppercase font-bold border-b border-base-200/50"
                >
                    {m.profile_information()}
                </li>

                <li class="list-row items-center">
                    <div>
                        <UserIcon size={24} class="p-1 opacity-70" />
                    </div>
                    <div class="flex-1">
                        <div class="text-sm font-medium">{user.display_name}</div>
                    </div>
                </li>

                <li class="list-row items-center">
                    <div>
                        <MailIcon size={24} class="p-1 opacity-70" />
                    </div>
                    <div class="flex-1">
                        <div class="text-sm font-medium">{user.email}</div>
                    </div>
                </li>

                <li class="list-row items-center">
                    <div>
                        <StampIcon size={24} class="p-1 opacity-70" />
                    </div>
                    <div class="flex-1">
                        <div class="text-sm font-medium">{m.profile_role({ role: roleLabel(user.role) })}</div>
                    </div>
                </li>

                <li class="list-row items-center">
                    <div>
                        <CalendarClockIcon size={24} class="p-1 opacity-70" />
                    </div>
                    <div class="flex-1">
                        <div class="text-sm font-medium">
                            {m.common_created({ when: new Date(user.created_at).toLocaleString(getLocale()) })}
                        </div>
                        <div class="text-xs opacity-40 font-mono italic">
                            {m.profile_timestamp()}
                        </div>
                    </div>
                </li>

                <li class="list-row items-center">
                    <div>
                        <IdCardIcon size={24} class="p-1 opacity-70" />
                    </div>
                    <div class="flex-1">
                        <div class="text-sm font-medium">{m.common_id({ id: user.id })}</div>
                    </div>
                </li>
            </Card>

            <ListUserSignatures status={data.signatureStatus || []} />
        </ul>

        <Card>
            <div class="card-body gap-4">
                <h2 class="card-title text-xs opacity-60 tracking-widest uppercase font-bold">
                    <LanguagesIcon class="w-4 h-4" />
                    {m.profile_language()}
                </h2>
                <form method="POST" action="?/setLocale" class="flex flex-col gap-3">
                    <label class="form-control w-full" for="profile-locale">
                        <span class="label-text">{m.locale_label()}</span>
                    </label>
                    <LocaleSelect id="profile-locale" value={user.locale} />
                    {#if localeError}
                        <Alert variant="error" message={m.locale_failed()} />
                    {/if}
                    <button type="submit" class="btn btn-primary w-full sm:w-auto">{m.locale_save()}</button>
                </form>
            </div>
        </Card>

        <Card>
            <div class="card-body">
                <h2
                    class="card-title text-xs opacity-60 tracking-widest uppercase font-bold mb-4"
                >
                    {m.profile_change_password()}
                </h2>
                <ChangePassword {form} showLogoutWarning={true} />
            </div>
        </Card>
    </div>
{/if}
