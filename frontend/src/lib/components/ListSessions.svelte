<script lang="ts">
  import { enhance } from '$app/forms';
  import Alert from '$lib/components/Alert.svelte';
  import * as m from '$lib/paraglide/messages.js';
  import { getLocale } from '$lib/paraglide/runtime';
  import Card from './Card.svelte';

  /** Flexible session row from the admin sessions API */
  type Session = {
    session_id?: string;
    id?: string;
    user_id?: string;
    userId?: string;
    user_email?: string;
    email?: string;
    user_display_name?: string;
    display_name?: string;
    user_name?: string;
    expires_at?: string;
    expiry?: string;
    expires?: string;
    expiresAt?: string;
    [key: string]: unknown;
  };

  let { items = [], form = null }: { items?: Session[]; form?: { revokeUserSessions?: { ok?: boolean; user_id?: string; error?: string } } | null } = $props();

  let sessions = $derived(items);

  let revokingUserId = $state<string | null>(null);

  const revokeResult = $derived(form?.revokeUserSessions);
  const revokeOkFor = $derived(revokeResult?.ok ? revokeResult?.user_id : null);

  function fmtDate(v: unknown) {
    try {
      if (v == null || v === '') return '—';
      const d = new Date(v as string | number | Date);
      return Number.isNaN(d.getTime()) ? String(v) : d.toLocaleString(getLocale());
    } catch {
      return String(v ?? '—');
    }
  }

  function sessionUserLabel(s: Session) {
    return (
      s.user_display_name ||
      s.display_name ||
      s.user_name ||
      s.email ||
      s.user_email ||
      s.user_id ||
      m.sessions_unknown()
    );
  }

  function sessionEmail(s: Session) {
    return s.user_email || s.email || '';
  }

  function sessionUserId(s: Session) {
    return s.user_id || s.userId || '';
  }

  function sessionExpires(s: Session) {
    return s.expires_at || s.expiry || s.expires || s.expiresAt || null;
  }
</script>

{#if revokeResult?.error}
  <Alert variant="error" message={revokeResult.error} class="shadow-lg" />
{/if}

{#if revokeOkFor}
  <Alert variant="success" message={m.sessions_revoked()} class="shadow-lg" />
{/if}

<Card>
  <ul class="list">
    <li class="p-4 pb-2 text-xs opacity-60 tracking-widest uppercase font-bold">
      {m.sessions_count({ count: String(sessions.length) })}
    </li>

    {#each sessions as s, i (s.session_id ?? s.id ?? `${sessionUserId(s)}-${i}`)}
      <li class="list-row items-center hover:bg-base-200/50 transition-colors flex">
        <div class="flex-1">
          <div class="font-bold text-sm lg:text-base">
            {sessionUserLabel(s)}
          </div>

          {#if sessionEmail(s)}
            <div class="text-xs opacity-50 font-mono uppercase tracking-tighter">
              {sessionEmail(s)}
            </div>
          {/if}

          {#if sessionUserId(s)}
            <div class="text-xs opacity-50 font-mono uppercase tracking-tighter">
              {m.sessions_user_id({ id: sessionUserId(s) })}
            </div>
          {/if}

          <div class="text-xs opacity-50 font-mono uppercase tracking-tighter">
            {m.sessions_expires({ when: fmtDate(sessionExpires(s)) })}
          </div>
        </div>

        <div>
          <form
            method="POST"
            action="?/revokeUserSessions"
            use:enhance={() => {
              const uid = sessionUserId(s);
              revokingUserId = uid || 'unknown';

              return async ({ update }) => {
                await update({ invalidateAll: true });
                revokingUserId = null;
              };
            }}
          >
            <input type="hidden" name="user_id" value={sessionUserId(s)} />

            <button
              type="submit"
              class="btn btn-error"
              disabled={!sessionUserId(s) || revokingUserId === sessionUserId(s)}
              title={m.sessions_revoke_title()}
            >
              {#if revokingUserId === sessionUserId(s)}
                <span class="loading loading-spinner"></span>
              {/if}
              {m.sessions_revoke()}
            </button>
          </form>
        </div>
      </li>
    {:else}
      <li class="p-12 text-center">
        <div class="text-sm opacity-40">{m.sessions_empty()}</div>
      </li>
    {/each}
  </ul>
</Card>
