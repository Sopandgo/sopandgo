<script lang="ts">
    import { enhance } from '$app/forms';
    import { resolve } from '$app/paths';
    import Alert from '#lib/components/Alert.svelte';
    import Card from '#lib/components/Card.svelte';
    import CardPageHeading from '#lib/components/CardPageHeading.svelte';
    import LocaleSelect from '#lib/components/LocaleSelect.svelte';
    import { SettingsIcon } from '@lucide/svelte';
    import * as m from '#lib/paraglide/messages.js';

    let { data, form }: { data: any; form: any } = $props();

    const smtp = $derived(data.smtp);
    const security = $derived(data.security);
    const required = $derived(!!security?.require_two_factor);
    // Turning the rule on needs the key and an admin who already uses 2FA, so
    // nobody locks themselves out by flipping it. Turning it off is always allowed.
    const blockedReason = $derived(
        required || !security
            ? null
            : !security.encryption_key_configured
              ? m.two_factor_key_missing()
              : !data.user?.two_factor_enabled
                ? m.security_require_needs_own()
                : null
    );
    let saving = $state(false);
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
                <form
                    method="POST"
                    action="?/saveSecurity"
                    class="space-y-1.5"
                    onchange={(event) => event.currentTarget.requestSubmit()}
                    use:enhance={() => {
                        saving = true;
                        return async ({ update }) => {
                            await update();
                            saving = false;
                        };
                    }}
                >
                    <label class="flex items-center gap-3 text-sm">
                        <input
                            type="checkbox"
                            name="require_two_factor"
                            class="toggle toggle-success"
                            checked={required}
                            disabled={saving || blockedReason !== null}
                            aria-describedby="security-require-help"
                        />
                        <span class="font-medium">{m.security_require_label()}</span>
                        <span class="text-base-content/70">{required ? m.two_factor_state_on() : m.two_factor_state_off()}</span>
                    </label>
                    <div id="security-require-help" class="space-y-1 text-sm text-base-content/70">
                        <p>{m.security_require_help()}</p>
                        {#if blockedReason}
                            <p>{blockedReason}</p>
                        {/if}
                    </div>
                </form>

                <!-- Read-only: follows whether an email transport is switched on under Email. -->
                <div class="space-y-1.5 border-t border-base-300 pt-4">
                    <div class="flex flex-wrap items-center gap-3 text-sm">
                        <span class="font-medium">{m.security_email_fallback_label()}</span>
                        {#if security.email_fallback_available}
                            <span class="badge badge-soft badge-success">{m.two_factor_state_on()}</span>
                        {:else}
                            <span class="badge badge-outline">{m.two_factor_state_off()}</span>
                        {/if}
                    </div>
                    <p class="text-sm text-base-content/70">
                        {security.email_fallback_available
                            ? m.security_email_fallback_on_help()
                            : m.security_email_fallback_off()}
                        <a class="underline" href={resolve('admin/settings/email')}>{m.security_email_fallback_link()}</a>
                    </p>
                </div>
            {/if}
        </div>
    </Card>
</div>
