<script lang="ts">
  import { enhance } from '$app/forms';
  import * as m from '$lib/paraglide/messages.js';
  import Alert from './Alert.svelte';

  type PasswordResult = { ok?: boolean; error?: string };

  interface Props {
    /** Page ActionData; the result is read from `form[resultKey]`. */
    form: Record<string, unknown> | null | undefined;
    actionName?: string;
    resultKey?: string;
    userId?: string | null;
    showLogoutWarning?: boolean;
  }

  let {
    form,
    actionName = 'changePassword',
    resultKey = 'changePassword',
    userId = null,
    showLogoutWarning = true
  }: Props = $props();

  let loading = $state(false);

  const result = $derived(form?.[resultKey] as PasswordResult | undefined);
  const hasError = $derived(!!result?.error);
  const isOk = $derived(!!result?.ok);
</script>

<div class="flex w-full flex-col gap-4">
  {#if result?.error}
    <Alert type="error" message={result.error} class="w-xs" />
  {/if}

  {#if isOk}
    <Alert type="success" message={m.password_updated()} class="w-xs" />
  {/if}

  <form
    class="w-full"
    method="POST"
    action={`?/${actionName}`}
    use:enhance={() => {
  loading = true;
  return async ({ update }) => {
    await update();
    loading = false;
  };
}}

  >
    {#if userId}
      <input type="hidden" name="user_id" value={userId} />
    {/if}

    <fieldset class="fieldset">

      <label class="label" for="new_password">
        {m.password_new()}
      </label>
      <input
        id="new_password"
        name="new_password"
        type="password"
        class="input w-full {hasError ? 'input-error' : ''}"
        placeholder={m.password_new()}
        minlength="12"
        autocomplete="new-password"
        required
      />

      <label class="label" for="confirm_password">
        {m.password_confirm()}
      </label>
      <input
        id="confirm_password"
        name="confirm_password"
        type="password"
        class="input w-full {hasError ? 'input-error' : ''}"
        placeholder={m.password_confirm()}
        minlength="12"
        autocomplete="new-password"
        required
      />

      <button type="submit" class="btn btn-primary mt-4 w-full" disabled={loading}>
        {#if loading}
          <span class="loading loading-spinner"></span>
        {/if}
        {m.password_update()}
      </button>
      {#if showLogoutWarning}
        <Alert type="warning" message={m.password_logout_warning()}/>
      {/if}
    </fieldset>
  </form>

</div>
