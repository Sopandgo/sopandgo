<script lang="ts">
  import { enhance } from '$app/forms';
  import { tick } from 'svelte';
  import Alert from '#lib/components/Alert.svelte';
  import * as m from '#lib/paraglide/messages.js';
  import Card from './Card.svelte';

  interface FormResult {
    revokeAllSessions?: {
      ok?: boolean;
      error?: string;
    };
  }

  let { form } = $props<{ form: FormResult | null }>();

  let loading = $state(false);
  const result = $derived(form?.revokeAllSessions);
</script>

<Card>
    <div class="card-body">
      <div class="flex items-center gap-3">
        <div class="p-2 bg-error/10 rounded-lg text-error">
          <svg xmlns="http://www.w3.org/2000/svg" class="w-6 h-6" fill="none" viewBox="0 0 24 24" stroke="currentColor">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z" />
          </svg>
        </div>
        <h3 class="card-title text-error">{m.sessions_panic_title()}</h3>
      </div>
      
      <p class="text-sm text-base-content/70 mt-2">
        {m.sessions_panic_body()}
      </p>

      {#if result?.error}
        <div class="mt-4">
          <Alert type="error" message={result.error} />
        </div>
      {/if}

      {#if result?.ok}
        <div class="mt-4">
          <Alert type="success" message={m.sessions_panic_ok()} />
        </div>
      {/if}

      <form
        method="POST"
        action="?/revokeAllSessions"
        class="mt-4"
        use:enhance={() => {
          loading = true;
          return async ({ update, formElement }) => {
            await update({ refreshAll: true });
            loading = false;

            await tick();
            // Safety check for reset
            if (form?.revokeAllSessions?.ok) {
              formElement.reset();
            }
          };
        }}
      >
        <div class="form-control w-full">
          <label class="label" for="reason">
            <span class="label-text font-semibold">{m.sessions_reason()}</span>
          </label>
          <input
            id="reason"
            name="reason"
            type="text"
            class="input w-full"
            placeholder={m.sessions_reason_placeholder()}
            required
            disabled={loading}
          />
        </div>

        <div class="card-actions justify-end mt-4">
          <button 
            class="btn btn-error"
            type="submit" 
            disabled={loading}
          >
            {#if loading}
              <span class="loading loading-spinner loading-sm"></span>
              {m.sessions_executing()}
            {:else}
              {m.sessions_revoke_now()}
            {/if}
          </button>
        </div>
      </form>
    </div>
</Card>
