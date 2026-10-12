<script lang="ts">
    import { enhance } from '$app/forms';
    import { resolve } from '$app/paths';
    import Alert from '#lib/components/Alert.svelte';
    import Card from '#lib/components/Card.svelte';
    import CardPageHeading from '#lib/components/CardPageHeading.svelte';
    import LocaleSelect from '#lib/components/LocaleSelect.svelte';
    import { SettingsIcon } from '@lucide/svelte';
    import * as m from '#lib/paraglide/messages.js';
    import type { SecuritySettings, TwoFactorPolicy } from '#lib/sdk/types.js';

    let { data, form }: { data: any; form: any } = $props();

    const smtp = $derived(data.smtp);
    const security = $derived(data.security as SecuritySettings | null);
    const current = $derived<TwoFactorPolicy>(security?.two_factor_policy ?? 'none');
    // svelte-ignore state_referenced_locally
    let selected = $state<TwoFactorPolicy>(data.security?.two_factor_policy ?? 'none');
    $effect(() => {
        selected = current;
    });
    let saving = $state(false);

    // Why a level cannot be chosen yet. Any level above None needs the key and
    // your own app, because admins need an app there; Email also needs email
    // switched on. The current level and None are always selectable.
    function blockedReason(policy: TwoFactorPolicy): string | null {
        if (!security || policy === 'none' || policy === current) return null;
        if (!security.encryption_key_configured) return m.two_factor_key_missing();
        if (!data.user?.two_factor_enabled) return m.security_policy_needs_own();
        if (policy === 'email' && !security.email_transport_on) return m.security_policy_needs_email();
        return null;
    }

    const options = $derived(
        (
            [
                { value: 'none', label: m.security_policy_none(), help: m.security_policy_none_help() },
                { value: 'email', label: m.security_policy_email(), help: m.security_policy_email_help() },
                { value: 'totp', label: m.security_policy_totp(), help: m.security_policy_totp_help() }
            ] as const
        ).map((o) => ({ ...o, blocked: blockedReason(o.value) }))
    );
</script>

<svelte:head>
    <title>{m.page_settings()}</title>
</svelte:head>

<div class="flex flex-col gap-6">
    <Card>
        <div class="card-body space-y-4">
            <CardPageHeading>
                <SettingsIcon class="w-8 h-8" />
                {m.settings_general()}
            </CardPageHeading>
            <p class="text-sm text-base-content/70">{m.settings_general_intro()}</p>
        </div>
    </Card>

    <Card title={m.locale_organization()} description={m.locale_organization_help()}>
        <div class="card-body space-y-4">
            {#if form?.saveDefaultLocale?.error}
                <Alert type="error" message={form.saveDefaultLocale.error} />
            {:else if form?.saveDefaultLocale?.ok}
                <Alert type="success" message={m.locale_saved()} />
            {/if}
            <form method="POST" action="?/saveDefaultLocale" class="flex flex-col gap-4" use:enhance>
                <label class="flex flex-col gap-1.5" for="org-locale">
                    <span class="text-sm font-medium">{m.locale_label()}</span>
                </label>
                <LocaleSelect id="org-locale" value={form?.saveDefaultLocale?.locale ?? smtp?.default_locale ?? 'en'} />
                <div class="flex flex-col sm:flex-row sm:justify-end">
                    <button type="submit" class="btn btn-primary w-full sm:w-auto">{m.locale_save()}</button>
                </div>
            </form>
        </div>
    </Card>

    <Card title={m.security_heading()}>
        <div class="card-body space-y-4">
            {#if form?.saveSecurity?.error}
                <Alert type="error" message={form.saveSecurity.error} />
            {:else if form?.saveSecurity?.ok}
                <Alert type="success" message={m.security_saved()} />
            {/if}

            {#if !security}
                <Alert type="error" message={m.security_load_failed()} />
            {:else}
                {#if current === 'email' && !security.email_transport_on}
                    <Alert type="warning">
                        {m.security_policy_email_off_warning()}
                        <a class="underline" href={resolve('admin/settings/email')}>{m.security_email_fallback_link()}</a>
                    </Alert>
                {/if}
                <form
                    method="POST"
                    action="?/saveSecurity"
                    class="space-y-4"
                    use:enhance={() => {
                        saving = true;
                        return async ({ update }) => {
                            await update({ reset: false });
                            saving = false;
                        };
                    }}
                >
                    <fieldset class="space-y-3" aria-describedby="security-policy-help">
                        <legend class="text-sm font-medium">{m.security_policy_label()}</legend>
                        <p id="security-policy-help" class="text-sm text-base-content/70">{m.security_policy_help()}</p>
                        {#each options as option (option.value)}
                            <label class="flex items-start gap-3 text-sm {option.blocked ? 'cursor-not-allowed' : 'cursor-pointer'}">
                                <input
                                    type="radio"
                                    name="two_factor_policy"
                                    value={option.value}
                                    class="radio radio-sm mt-0.5"
                                    bind:group={selected}
                                    disabled={saving || option.blocked !== null}
                                    aria-describedby="security-policy-{option.value}-help"
                                />
                                <span class="space-y-0.5">
                                    <span class="flex flex-wrap items-center gap-2 font-medium">
                                        {option.label}
                                        {#if option.value === current}
                                            <span class="badge badge-outline badge-sm">{m.security_policy_current()}</span>
                                        {/if}
                                    </span>
                                    <span id="security-policy-{option.value}-help" class="block text-base-content/70">
                                        {option.help}
                                        {#if option.blocked}
                                            <span class="block">{option.blocked}</span>
                                        {/if}
                                    </span>
                                </span>
                            </label>
                        {/each}
                    </fieldset>
                    <div class="flex flex-col sm:flex-row sm:justify-end">
                        <button type="submit" class="btn w-full sm:w-auto" disabled={saving || selected === current}>
                            {#if saving}<span class="loading loading-spinner loading-xs"></span>{/if}
                            {m.security_policy_save()}
                        </button>
                    </div>
                </form>
            {/if}
        </div>
    </Card>
</div>
