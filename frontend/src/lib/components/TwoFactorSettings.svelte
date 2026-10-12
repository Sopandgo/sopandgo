<script lang="ts">
    import { enhance } from '$app/forms';
    import Alert from '#lib/components/Alert.svelte';
    import Card from '#lib/components/Card.svelte';
    import * as m from '#lib/paraglide/messages.js';
    import type { TwoFactorSetup, TwoFactorStatus } from '#lib/sdk/types.js';
    import { CopyIcon, DownloadIcon, ShieldCheckIcon } from '@lucide/svelte';

    /*
     * Account settings → Two-factor authentication. One password field serves
     * set-up, replacement, new recovery codes, and turning 2FA off; the button
     * picks the action. The secret and recovery codes are shown once and kept
     * only in this component's state.
     */
    interface Props {
        status: TwoFactorStatus | null;
        form?: any;
    }

    let { status, form = null }: Props = $props();

    let setup = $state<TwoFactorSetup | null>(null);
    let recoveryCodes = $state<string[] | null>(null);
    let busy = $state(false);
    let copied = $state<'secret' | 'codes' | null>(null);

    const passwordError = $derived(
        form?.beginTwoFactor?.error ?? form?.disableTwoFactor?.error ?? form?.regenerateRecoveryCodes?.error
    );
    const confirmError = $derived(form?.confirmTwoFactor?.error);
    const disabledOk = $derived(!!form?.disableTwoFactor?.ok);

    async function copy(text: string, what: 'secret' | 'codes') {
        try {
            await navigator.clipboard.writeText(text);
            copied = what;
            setTimeout(() => (copied = null), 2000);
        } catch {
            copied = null;
        }
    }

    function downloadCodes(codes: string[]) {
        const body = `${m.two_factor_recovery_file_heading()}\n\n${codes.join('\n')}\n`;
        const url = URL.createObjectURL(new Blob([body], { type: 'text/plain' }));
        const a = document.createElement('a');
        a.href = url;
        a.download = 'sopandgo-recovery-codes.txt';
        a.click();
        URL.revokeObjectURL(url);
    }

    function track() {
        busy = true;
        return async ({ result, update }: { result: any; update: () => Promise<void> }) => {
            await update();
            busy = false;
            if (result.type !== 'success') return;
            const data = result.data ?? {};
            if (data.beginTwoFactor?.setup) {
                setup = data.beginTwoFactor.setup;
                recoveryCodes = null;
            }
            if (data.confirmTwoFactor?.ok) {
                setup = null;
                const codes = data.confirmTwoFactor.recoveryCodes ?? [];
                recoveryCodes = codes.length > 0 ? codes : null;
            }
            if (data.regenerateRecoveryCodes?.recoveryCodes) {
                recoveryCodes = data.regenerateRecoveryCodes.recoveryCodes;
            }
            if (data.disableTwoFactor?.ok) {
                setup = null;
                recoveryCodes = null;
            }
        };
    }
</script>

<Card>
    <div class="card-body gap-4">
        <div class="flex flex-wrap items-center justify-between gap-2">
            <h2 class="flex items-center gap-2 text-lg font-semibold">
                <ShieldCheckIcon class="h-4 w-4" />
                {m.two_factor_heading()}
            </h2>
            {#if status?.enabled}
                <span class="badge badge-soft badge-success">{m.two_factor_state_on()}</span>
            {:else}
                <span class="badge badge-outline">{m.two_factor_state_off()}</span>
            {/if}
        </div>

        {#if !status}
            <Alert type="error" message={m.two_factor_status_failed()} />
        {:else}
            <p class="text-sm text-base-content/70">
                {status.enabled ? m.two_factor_intro_on() : m.two_factor_intro_off()}
            </p>

            {#if status.required && !status.enabled}
                <Alert type="warning" message={m.two_factor_required_notice()} />
            {/if}
            {#if disabledOk}
                <Alert type="success" message={m.two_factor_disabled_ok()} />
            {/if}

            {#if recoveryCodes}
                <div class="space-y-3">
                    <Alert type="warning" message={m.two_factor_recovery_save()} />
                    <ul class="grid grid-cols-2 gap-2 font-mono text-sm" aria-label={m.two_factor_recovery_heading()}>
                        {#each recoveryCodes as code (code)}
                            <li class="rounded-field border border-base-300 px-2 py-1 text-center">{code}</li>
                        {/each}
                    </ul>
                    <div class="flex flex-wrap gap-2">
                        <button type="button" class="btn btn-sm" onclick={() => copy(recoveryCodes!.join('\n'), 'codes')}>
                            <CopyIcon class="size-4" />
                            {copied === 'codes' ? m.two_factor_copied() : m.two_factor_recovery_copy()}
                        </button>
                        <button type="button" class="btn btn-sm" onclick={() => downloadCodes(recoveryCodes!)}>
                            <DownloadIcon class="size-4" />
                            {m.two_factor_recovery_download()}
                        </button>
                        <button type="button" class="btn btn-ghost btn-sm" onclick={() => (recoveryCodes = null)}>
                            {m.two_factor_recovery_done()}
                        </button>
                    </div>
                </div>
            {/if}

            {#if setup}
                <form method="POST" action="?/confirmTwoFactor" class="space-y-3" use:enhance={track}>
                    <ol class="list-decimal space-y-3 ps-5 text-sm">
                        <li>
                            {m.two_factor_setup_scan()}
                            <!-- White behind the code in both themes: scanners need a light quiet zone. -->
                            <img
                                src={setup.qr_code_data_url}
                                alt={m.two_factor_setup_qr_alt()}
                                width="200"
                                height="200"
                                class="mt-2 rounded-box border border-base-300 bg-white p-2"
                            />
                        </li>
                        <li>
                            {m.two_factor_setup_manual()}
                            <div class="mt-1 flex flex-wrap items-center gap-2">
                                <code class="font-mono text-xs break-all">{setup.secret}</code>
                                <button
                                    type="button"
                                    class="btn btn-square btn-ghost btn-sm"
                                    aria-label={m.two_factor_setup_copy_secret()}
                                    title={m.two_factor_setup_copy_secret()}
                                    onclick={() => copy(setup!.secret, 'secret')}
                                >
                                    <CopyIcon class="size-4" />
                                </button>
                                {#if copied === 'secret'}
                                    <span class="text-xs text-base-content/70">{m.two_factor_copied()}</span>
                                {/if}
                            </div>
                        </li>
                        <li>
                            <label for="two-factor-confirm-code">{m.two_factor_setup_enter_code()}</label>
                            <input
                                id="two-factor-confirm-code"
                                name="code"
                                type="text"
                                inputmode="numeric"
                                autocomplete="one-time-code"
                                maxlength="8"
                                class="input mt-1 w-full font-mono {confirmError ? 'input-error' : ''}"
                                aria-invalid={confirmError ? 'true' : undefined}
                                aria-describedby={confirmError ? 'two-factor-confirm-error' : undefined}
                                required
                            />
                            {#if confirmError}
                                <p id="two-factor-confirm-error" class="mt-1 text-xs text-error">{confirmError}</p>
                            {/if}
                        </li>
                    </ol>
                    <div class="flex flex-wrap justify-end gap-2">
                        <button type="button" class="btn" onclick={() => (setup = null)} disabled={busy}>
                            {m.common_cancel()}
                        </button>
                        <button type="submit" class="btn" disabled={busy}>
                            {#if busy}<span class="loading loading-spinner loading-xs"></span>{/if}
                            {m.two_factor_setup_confirm()}
                        </button>
                    </div>
                </form>
            {:else if !status.encryption_key_configured && !status.enabled}
                <Alert type="info" message={m.two_factor_key_missing()} />
            {:else}
                <form method="POST" action="?/beginTwoFactor" class="space-y-3" use:enhance={track}>
                    <div>
                        <label class="label mb-1" for="two-factor-password">{m.two_factor_password_label()}</label>
                        <input
                            id="two-factor-password"
                            name="current_password"
                            type="password"
                            autocomplete="current-password"
                            class="input w-full {passwordError ? 'input-error' : ''}"
                            aria-invalid={passwordError ? 'true' : undefined}
                            aria-describedby={passwordError ? 'two-factor-password-error' : 'two-factor-password-help'}
                            required
                        />
                        {#if passwordError}
                            <p id="two-factor-password-error" class="mt-1 text-xs text-error">{passwordError}</p>
                        {:else}
                            <p id="two-factor-password-help" class="mt-1 text-xs text-base-content/70">
                                {m.two_factor_password_help()}
                            </p>
                        {/if}
                    </div>

                    {#if status.enabled && status.recovery_codes_available}
                        <p class="text-sm text-base-content/70">
                            {m.two_factor_recovery_remaining({ count: String(status.recovery_codes_remaining) })}
                        </p>
                    {/if}

                    <div class="flex flex-wrap justify-end gap-2">
                        {#if status.enabled}
                            {#if status.recovery_codes_available}
                                <button type="submit" class="btn" formaction="?/regenerateRecoveryCodes" disabled={busy}>
                                    {m.two_factor_recovery_regenerate()}
                                </button>
                            {/if}
                            {#if !status.required}
                                <button type="submit" class="btn text-error" formaction="?/disableTwoFactor" disabled={busy}>
                                    {m.two_factor_disable()}
                                </button>
                            {/if}
                            <button type="submit" class="btn" disabled={busy}>
                                {m.two_factor_replace()}
                            </button>
                        {:else}
                            <button type="submit" class="btn" disabled={busy}>
                                {#if busy}<span class="loading loading-spinner loading-xs"></span>{/if}
                                {m.two_factor_setup_start()}
                            </button>
                        {/if}
                    </div>
                    {#if status.enabled && status.required}
                        <p class="text-xs text-base-content/70">{m.two_factor_required_cannot_disable()}</p>
                    {/if}
                </form>
            {/if}
        {/if}
    </div>
</Card>
