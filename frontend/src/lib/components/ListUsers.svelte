<script lang="ts">
    import Alert from '$lib/components/Alert.svelte';
    import { enhance } from '$app/forms';
    import Avatar from '$lib/components/Avatar.svelte';
    import ToggleUserStatus from './ToggleUserStatus.svelte';
    import SelectUserRole from './SelectUserRole.svelte';
    import type { User } from '$lib/sdk/types';
    import Card from './Card.svelte';
    import { MailIcon, CheckIcon } from '@lucide/svelte';
    import * as m from '$lib/paraglide/messages.js';
    import { getLocale } from '$lib/paraglide/runtime';

    interface Props {
        items?: User[] | null;
        currentUserId?: string | null;
        mailMode?: 'smtp' | 'manual_links';
        form?: any;
    }

    let { items = [], currentUserId = null, mailMode = 'smtp', form = null }: Props = $props();

    let users = $derived(items ?? []);

    // Track which user just got an email sent (for visual feedback)
    let justSentId = $state<string | null>(null);

    // Reset the "Success" checkmark after 3 seconds
    $effect(() => {
        if (justSentId) {
            const timer = setTimeout(() => { justSentId = null; }, 3000);
            return () => clearTimeout(timer);
        }
    });
</script>

<div class="w-full">
    <Card title={m.users_count({ count: String(users.length) })}>
        <ul class="list">
            {#if form?.triggerPasswordReset?.error}
                <li class="px-4 pt-4">
                    <Alert type="error" message={form.triggerPasswordReset.error} />
                </li>
            {/if}
            {#if form?.triggerPasswordReset?.ok}
                <li class="px-4 pt-4 space-y-2">
                    <Alert
                        type="success"
                        message={form.triggerPasswordReset.message ??
                            (mailMode === 'manual_links' ? m.users_reset_generated() : m.users_reset_sent())}
                    />
                    {#if form?.triggerPasswordReset?.link}
                        <Alert type="info" compact class="break-all">
                            {m.users_reset_link()} <code>{form.triggerPasswordReset.link}</code>
                        </Alert>
                    {/if}
                </li>
            {/if}

            {#each users as user (user.id)}
                <li
                    class="list-row flex-col items-stretch gap-4 border-b border-base-300 py-4 after:hidden last:border-b-0 md:flex-row md:items-center md:gap-4"
                >
                    <div class="flex min-w-0 flex-1 flex-row items-start gap-3 md:items-center md:gap-4">
                        <div class="shrink-0">
                            <Avatar
                                displayName={user.display_name}
                                userId={user.id}
                                hasAvatar={user.has_avatar}
                                avatarContentHash={user.avatar_content_hash}
                            />
                        </div>

                        <div class="min-w-0 flex-1 space-y-0.5">
                            <div class="text-sm font-medium">{user.display_name}</div>
                            <div
                                class="break-words text-xs text-base-content/70"
                            >
                                {user.email}
                            </div>
                            <div class="text-xs text-base-content/70">
                                {m.common_created({ when: new Date(user.created_at).toLocaleString(getLocale()) })}
                            </div>
                        </div>
                    </div>

                    <div
                        class="flex w-full flex-col flex-nowrap gap-3 md:w-auto md:flex-row md:flex-wrap md:items-center md:justify-end md:gap-4"
                    >
                        <div
                            class="w-full min-w-0 md:w-auto [&_form]:block md:[&_form]:inline-block [&_select]:w-full md:[&_select]:w-auto"
                        >
                            <SelectUserRole
                                currentUserId={currentUserId}
                                user={user}
                                form={null}
                            />
                        </div>

                        <form
                            class="w-full md:w-auto"
                            method="POST"
                            action="?/triggerPasswordReset"
                            use:enhance={() => {
                                return async ({ result, update }) => {
                                    // Important: apply action result so `form` gets updated
                                    // and manual-links payload (including link) is available in UI.
                                    await update();
                                    if (result.type === 'success' && mailMode === 'smtp') {
                                        justSentId = user.id;
                                    }
                                };
                            }}
                        >
                            <input type="hidden" name="user_id" value={user.id} />

                            <button
                                type="submit"
                                class="btn btn-sm btn-ghost h-auto min-h-9 w-full flex-nowrap gap-2 py-2 md:w-auto"
                                disabled={justSentId === user.id}
                                title={m.users_send_reset_title()}
                            >
                                {#if justSentId === user.id}
                                    <CheckIcon class="h-4 w-4 shrink-0 text-success" />
                                    <span class="text-success">{m.users_sent()}</span>
                                {:else}
                                    <MailIcon class="h-4 w-4 shrink-0" />
                                    <span>{mailMode === 'manual_links' ? m.users_generate_reset() : m.users_send_reset()}</span>
                                {/if}
                            </button>
                        </form>

                        {#if form?.triggerPasswordReset?.link && form?.triggerPasswordReset?.user_id === user.id}
                            <button
                                type="button"
                                class="btn btn-sm w-full md:w-auto"
                                onclick={() => navigator.clipboard.writeText(form.triggerPasswordReset.link)}
                            >
                                {m.users_copy_reset()}
                            </button>
                        {/if}

                        <div
                            class="flex w-full items-center gap-2 border-t border-base-300 pt-3 md:w-auto md:border-t-0 md:pt-0"
                        >
                            <ToggleUserStatus currentUserId={currentUserId} user={user} />
                        </div>
                    </div>
                </li>
            {:else}
                <li class="p-6 text-sm text-base-content/70">{m.users_empty()}</li>
            {/each}
        </ul>
    </Card>
</div>