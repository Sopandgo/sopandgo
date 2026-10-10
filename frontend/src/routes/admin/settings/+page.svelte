<script lang="ts">
    import { enhance } from '$app/forms';
    import Alert from '$lib/components/Alert.svelte';
    import Card from '$lib/components/Card.svelte';
    import CardPageHeading from '$lib/components/CardPageHeading.svelte';
    import LocaleSelect from '$lib/components/LocaleSelect.svelte';
    import { LanguagesIcon } from '@lucide/svelte';
    import * as m from '$lib/paraglide/messages.js';

    let { data, form }: { data: any; form: any } = $props();

    const smtp = $derived(data.smtp);
</script>

<svelte:head>
    <title>{m.page_settings()}</title>
</svelte:head>

<div class="flex flex-col gap-6">
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
