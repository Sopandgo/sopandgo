<script lang="ts">
    import type { Snippet } from 'svelte';
    import * as m from '$lib/paraglide/messages.js';

    /*
     * A modal that confirms a destructive action (docs/design/style-guide.md →
     * ConfirmDialog). Native <dialog>: showModal() traps focus, Escape and Cancel
     * close it. Place it inside the <form> it confirms, so fields in `children`
     * are submitted with the form and the `confirm` snippet's button submits it.
     */
    interface Props {
        title: string;
        /** Bindable; true shows the dialog. */
        open?: boolean;
        /** While true, Cancel and Escape do nothing (the action is running). */
        busy?: boolean;
        children: Snippet;
        /** The confirm button (`btn btn-error`, type="submit"). */
        confirm: Snippet;
    }

    let { title, open = $bindable(false), busy = false, children, confirm }: Props = $props();

    const uid = $props.id();
    let dialog: HTMLDialogElement | undefined = $state();

    $effect(() => {
        if (!dialog) return;
        if (open && !dialog.open) dialog.showModal();
        if (!open && dialog.open) dialog.close();
    });

    function close() {
        if (!busy) open = false;
    }
</script>

<dialog
    bind:this={dialog}
    class="modal"
    aria-labelledby="{uid}-title"
    oncancel={(e) => {
        // Escape: keep the dialog while the action runs; otherwise sync `open`.
        e.preventDefault();
        close();
    }}
    onclose={() => (open = false)}
>
    <div class="modal-box rounded-box border border-base-300 bg-base-100 shadow-overlay">
        <h2 id="{uid}-title" class="text-lg font-semibold">{title}</h2>
        <div class="mt-4 space-y-4 text-sm">
            {@render children()}
        </div>
        <div class="modal-action">
            <button type="button" class="btn" onclick={close} disabled={busy}>{m.common_cancel()}</button>
            {@render confirm()}
        </div>
    </div>
</dialog>
