<script lang="ts">
    import { enhance } from '$app/forms';
    import Alert from '#lib/components/Alert.svelte';
    import Card from '#lib/components/Card.svelte';
    import CardPageHeading from '#lib/components/CardPageHeading.svelte';
    import LocaleSelect from '#lib/components/LocaleSelect.svelte';
    import { LanguagesIcon, ShieldCheckIcon } from '@lucide/svelte';
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
            <h2 class="flex items-center gap-2 text-lg font-semibold">
                <ShieldCheckIcon class="h-4 w-4" />
                {m.security_heading()}
            </h2>
            <p class="text-sm text-base-content/70">{m.security_require_help()}</p>
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
                            aria-describedby={blockedReason ? 'security-blocked' : undefined}
                        />
                        <span>{m.security_require_label()}</span>
                        <span class="text-base-content/70">{required ? m.two_factor_state_on() : m.two_factor_state_off()}</span>
                    </label>
                </form>
                {#if blockedReason}
                    <p id="security-blocked" class="text-xs text-base-content/70">{blockedReason}</p>
                {/if}
                {#if !security.email_fallback_available}
                    <p class="text-xs text-base-content/70">{m.security_email_fallback_off()}</p>
                {/if}
            {/if}
        </div>
    </Card>

    <Card>
        <div class="card-body space-y-4">
            <CardPageHeading>
                <LanguagesIcon class="w-8 h-8" />
                {m.locale_organization()}
            </CardPageHeading>
            <p class="text-sm text-base-content/70">{m.locale_organization_help()}</p>
            {#if form?.saveDefaultLocale?.error}
                <Alert type="error" message={form.saveDefaultLocale.error} />
            {/if}
            {#if form?.saveDefaultLocale?.ok}
                <Alert type="success" message={m.locale_saved()} />
            {/if}
            <form method="POST" action="?/saveDefaultLocale" use:enhance>
                <label class="form-control w-full mb-4" for="org-locale">
                    <span class="label-text">{m.locale_label()}</span>
                </label>
                <LocaleSelect id="org-locale" value={form?.saveDefaultLocale?.locale ?? smtp?.default_locale ?? 'en'} />
                <div class="card-actions mt-4 w-full flex-col sm:flex-row sm:justify-end">
                    <button type="submit" class="btn btn-primary w-full sm:w-auto">{m.locale_save()}</button>
                </div>
            </form>
        </div>
    </Card>
</div>
