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
  <Alert type="error" message={revokeResult.error} />
{/if}

{#if revokeOkFor}
  <Alert type="success" message={m.sessions_revoked()} />
{/if}

<Card title={m.sessions_count({ count: String(sessions.length) })}>
  <ul class="list">

    {#each sessions as s, i (s.session_id ?? s.id ?? `${sessionUserId(s)}-${i}`)}
      <li class="list-row flex items-center border-b border-base-300 after:hidden last:border-b-0">
        <div class="flex-1">
          <div class="text-sm font-medium">
            {sessionUserLabel(s)}
          </div>

          {#if sessionEmail(s)}
            <div class="text-xs text-base-content/70">
              {sessionEmail(s)}
            </div>
          {/if}

          {#if sessionUserId(s)}
            <div class="text-xs text-base-content/70 font-mono">
              {m.sessions_user_id({ id: sessionUserId(s) })}
            </div>
          {/if}

          <div class="text-xs text-base-content/70">
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
              class="btn btn-sm text-error"
              disabled={!sessionUserId(s) || revokingUserId === sessionUserId(s)}
              title={m.sessions_revoke_title()}
            >
              {#if revokingUserId === sessionUserId(s)}
                <span class="loading loading-spinner loading-xs"></span>
              {/if}
              {m.sessions_revoke()}
            </button>
          </form>
        </div>
      </li>
    {:else}
      <li class="p-6 text-sm text-base-content/70">{m.sessions_empty()}</li>
    {/each}
  </ul>
</Card>
