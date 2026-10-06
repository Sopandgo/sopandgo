<script lang="ts">
    import { enhance } from '$app/forms';
    import Avatar from '$lib/components/Avatar.svelte';
    import ToggleUserStatus from './ToggleUserStatus.svelte';
    import SelectUserRole from './SelectUserRole.svelte';
    import type { User } from '$lib/sdk/types';
    import Card from './Card.svelte';
    import { MailIcon, CheckIcon } from 'lucide-svelte';
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
    <Card>
        <ul class="list">
            {#if form?.triggerPasswordReset?.error}
                <li class="px-4 pt-4">
                    <div class="alert alert-error text-sm">{form.triggerPasswordReset.error}</div>
                </li>
            {/if}
            {#if form?.triggerPasswordReset?.ok}
                <li class="px-4 pt-4 space-y-2">
                    <div class="alert alert-success text-sm">
                        {form.triggerPasswordReset.message ??
                            (mailMode === 'manual_links' ? m.users_reset_generated() : m.users_reset_sent())}
                    </div>
                    {#if form?.triggerPasswordReset?.link}
                        <div class="alert alert-info text-xs break-all">
                            <span>{m.users_reset_link()} <code>{form.triggerPasswordReset.link}</code></span>
                        </div>
                    {/if}
                </li>
            {/if}
            <li class="p-4 pb-2 text-xs opacity-60 tracking-widest uppercase font-bold">
                {m.users_count({ count: String(users.length) })}
            </li>

            {#each users as user (user.id)}
                <li
                    class="list-row flex-col items-stretch gap-4 py-4 md:flex-row md:items-center md:gap-4 hover:bg-base-200/50 transition-colors"
                >
                    <div class="flex min-w-0 flex-1 flex-row items-start gap-3 md:items-center md:gap-4">
                        <div class="shrink-0">
                            <Avatar displayName={user.display_name} color="secondary" />
                        </div>

                        <div class="min-w-0 flex-1 space-y-0.5">
                            <div class="font-bold text-sm lg:text-base">{user.display_name}</div>
                            <div
                                class="break-words text-xs font-mono uppercase tracking-tighter opacity-50"
                            >
                                {user.email}
                            </div>
                            <div class="text-xs opacity-50 font-mono uppercase tracking-tighter">
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
                                class="btn btn-xs btn-outline w-full md:w-auto"
                                onclick={() => navigator.clipboard.writeText(form.triggerPasswordReset.link)}
                            >
                                {m.users_copy_reset()}
                            </button>
                        {/if}

                        <div
                            class="flex w-full items-center gap-2 border-t border-base-200/60 pt-3 md:w-auto md:border-t-0 md:pt-0"
                        >
                            <ToggleUserStatus currentUserId={currentUserId} user={user} />
                        </div>
                    </div>
                </li>
            {:else}
                <li class="p-12 text-center">
                    <div class="text-sm opacity-40">{m.users_empty()}</div>
                </li>
            {/each}
        </ul>
    </Card>
</div>