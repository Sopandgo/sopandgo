<script lang="ts">
  import { enhance } from '$app/forms';
  import { tick } from 'svelte';
  import Alert from '$lib/components/Alert.svelte';
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

  let { form = null, mailMode = 'smtp' }: { form: FormResult | null; mailMode?: 'smtp' | 'manual_links' } = $props();

  let loading = $state(false);

  // Derived states for UI logic
  const result = $derived(form?.registerUser);
  const hasError = $derived(!!result?.error);
  const isOk = $derived(!!result?.ok);
  const values = $derived(form?.values ?? {});

  const ROLES: { value: UserRole; label: string }[] = [
    { value: 'admin', label: 'admin' },
    { value: 'editor', label: 'editor' },
    { value: 'approver', label: 'approver' },
    { value: 'auditor', label: 'auditor' },
    { value: 'viewer', label: 'viewer' }
  ];
</script>

<div class="flex flex-col gap-4">
  {#if result?.error}
    <Alert variant="error" message={result.error} class="shadow-lg" />
  {/if}

  {#if isOk}
    <Alert variant="success" message="User registered successfully." class="shadow-lg" />
  {/if}

  {#if result?.warning}
    <Alert variant="warning" message={result.warning} class="shadow-lg" />
  {/if}

  {#if result?.link}
    <div class="alert alert-info text-xs break-all">
      <span>Invite link: <code>{result.link}</code></span>
    </div>
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
        <span class="label-text">Display name</span>
      </label>
      <input
        id="display_name"
        name="display_name"
        type="text"
        class="input input-accent input-bordered w-full {hasError ? 'input-error' : ''}"
        placeholder="Full Name"
        value={values.display_name ?? ''}
        required
        disabled={loading}
      />

      <label class="label mt-2" for="email">
        <span class="label-text">Email</span>
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

      <label class="label mt-2" for="role">
        <span class="label-text">Role</span>
      </label>
      <select
        id="role"
        name="role"
        class="select select-accent select-bordered w-full {hasError ? 'select-error' : ''}"
        required
        disabled={loading}
      >
        <option value="" disabled selected={!values.role}>Select a role</option>
        {#each ROLES as role}
          <option value={role.value} selected={values.role === role.value}>
            {role.label}
          </option>
        {/each}
      </select>

      <button type="submit" class="btn btn-accent mt-6 w-full" disabled={loading}>
        {#if loading}
          <span class="loading loading-spinner"></span>
        {/if}
        Create user
      </button>

      <div class="mt-4">
        <Alert
          variant="warning"
          message={mailMode === 'manual_links'
            ? 'Manual links mode: after create, copy the generated link and share it securely.'
            : 'The user will be required to set their password.'}
          class="text-xs"
        />
      </div>
    </fieldset>
  </form>
</div>