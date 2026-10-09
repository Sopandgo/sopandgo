<script lang="ts">
  import { enhance } from '$app/forms';
  import { tick } from 'svelte';
  import Alert from '$lib/components/Alert.svelte';
  import LocaleSelect from '$lib/components/LocaleSelect.svelte';
  import * as m from '$lib/paraglide/messages.js';
  import type { UserRole } from '$lib/sdk/types';

  // Define the shape of the action result locally
  interface FormResult {
    registerUser?: {
      ok?: boolean;
      error?: string;
      warning?: string;
      link?: string;
    };
    values?: {
      display_name?: string;
      email?: string;
      role?: UserRole;
    };
  }

  let { form = null, mailMode = 'smtp', defaultLocale = 'en' }: { form: FormResult | null; mailMode?: 'smtp' | 'manual_links'; defaultLocale?: string } = $props();

  let loading = $state(false);

  // Derived states for UI logic
  const result = $derived(form?.registerUser);
  const hasError = $derived(!!result?.error);
  const isOk = $derived(!!result?.ok);
  const values = $derived(form?.values ?? {});

  const ROLES = $derived<{ value: UserRole; label: string }[]>([
    { value: 'admin', label: m.role_admin() },
    { value: 'editor', label: m.role_editor() },
    { value: 'approver', label: m.role_approver() },
    { value: 'auditor', label: m.role_auditor() },
    { value: 'viewer', label: m.role_viewer() }
  ]);
</script>

<div class="flex flex-col gap-4">
  {#if result?.error}
    <Alert variant="error" message={result.error} class="shadow-lg" />
  {/if}

  {#if isOk}
    <Alert variant="success" message={m.users_registered()} class="shadow-lg" />
  {/if}

  {#if result?.warning}
    <Alert variant="warning" message={result.warning} class="shadow-lg" />
  {/if}

  {#if result?.link}
    <Alert variant="info" compact class="break-all">
      {m.users_invite_link()} <code>{result.link}</code>
    </Alert>
  {/if}

  <form
    method="POST"
    action="?/registerUser"
    use:enhance={() => {
      loading = true;
      return async ({ update, formElement }) => {
        // update() refreshes the page data (the user list)
        await update();
        loading = false;

        // Only reset if the operation was a success
        await tick();
        if (form?.registerUser?.ok) {
          formElement.reset();
        }
      };
    }}
  >
    <fieldset class="fieldset">
      <label class="label" for="display_name">
        <span class="label-text">{m.users_display_name()}</span>
      </label>
      <input
        id="display_name"
        name="display_name"
        type="text"
        class="input input-accent input-bordered w-full {hasError ? 'input-error' : ''}"
        placeholder={m.users_full_name_placeholder()}
        value={values.display_name ?? ''}
        required
        disabled={loading}
      />

      <label class="label mt-2" for="email">
        <span class="label-text">{m.common_email()}</span>
      </label>
      <input
        id="email"
        name="email"
        type="email"
        class="input input-accent input-bordered w-full {hasError ? 'input-error' : ''}"
        placeholder="user@example.com"
        value={values.email ?? ''}
        required
        disabled={loading}
      />

      <label class="label mt-2" for="locale">
        <span class="label-text">{m.locale_initial()}</span>
      </label>
      <LocaleSelect id="locale" value={defaultLocale} />
      <p class="text-xs text-base-content/70 mt-1">{m.locale_initial_help()}</p>

      <label class="label mt-2" for="role">
        <span class="label-text">{m.common_role()}</span>
      </label>
      <select
        id="role"
        name="role"
        class="select select-accent select-bordered w-full {hasError ? 'select-error' : ''}"
        required
        disabled={loading}
      >
        <option value="" disabled selected={!values.role}>{m.users_select_role()}</option>
        {#each ROLES as role (role.value)}
          <option value={role.value} selected={values.role === role.value}>
            {role.label}
          </option>
        {/each}
      </select>

      <button type="submit" class="btn btn-accent mt-6 w-full" disabled={loading}>
        {#if loading}
          <span class="loading loading-spinner"></span>
        {/if}
        {m.users_create()}
      </button>

      <div class="mt-4">
        <Alert
          variant="warning"
          message={mailMode === 'manual_links'
            ? m.users_manual_links_help()
            : m.users_password_help()}
          class="text-xs"
        />
      </div>
    </fieldset>
  </form>
</div>