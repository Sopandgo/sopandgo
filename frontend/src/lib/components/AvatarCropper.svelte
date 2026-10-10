<script lang="ts">
    import * as m from '$lib/paraglide/messages.js';
    import { CheckIcon, XIcon } from 'lucide-svelte';
    import Alert from '$lib/components/Alert.svelte';

    /*
     * Square crop for a profile picture: drag to pan, slider or +/- to zoom, arrow keys to nudge.
     * Plain canvas, no dependency. Confirm hands back a square JPEG; the picture is shown
     * round everywhere (docs/design/style-guide.md → Avatar), so the preview marks the circle.
     */
    interface Props {
        /** The image the user picked (JPEG, PNG or WebP). */
        file: Blob;
        /** Disables the buttons while the caller uploads the result. */
        busy?: boolean;
        /** Edge length of the exported square, in pixels. */
        outputSize?: number;
        onconfirm: (blob: Blob) => void;
        oncancel: () => void;
    }

    let { file, busy = false, outputSize = 512, onconfirm, oncancel }: Props = $props();

    const VIEW = 320;
    const MAX_ZOOM = 4;
    const NUDGE = 12;

    let bitmap = $state.raw<ImageBitmap>();
    let failed = $state(false);
    let zoom = $state(1);
    // Image centre relative to the viewport centre, in viewport pixels.
    let offsetX = $state(0);
    let offsetY = $state(0);
    let drag = $state.raw<{
        pointerId: number;
        x: number;
        y: number;
        startX: number;
        startY: number;
    } | null>(null);

    // "Cover" scale: at zoom 1 the shorter side of the image just fills the square.
    const scale = $derived(
        bitmap ? Math.max(VIEW / bitmap.width, VIEW / bitmap.height) * zoom : 1
    );
    const maxX = $derived(bitmap ? Math.max(0, (bitmap.width * scale - VIEW) / 2) : 0);
    const maxY = $derived(bitmap ? Math.max(0, (bitmap.height * scale - VIEW) / 2) : 0);

    function clamp(value: number, limit: number) {
        return Math.min(limit, Math.max(-limit, value));
    }

    function setOffset(x: number, y: number) {
        offsetX = clamp(x, maxX);
        offsetY = clamp(y, maxY);
    }

    function setZoom(value: number) {
        zoom = Math.min(MAX_ZOOM, Math.max(1, value));
        setOffset(offsetX, offsetY);
    }

    $effect(() => {
        const source = file;
        let cancelled = false;
        let loaded: ImageBitmap | undefined;
        // The parent keys this component by file, so state starts fresh for every picture.
        createImageBitmap(source)
            .then((decoded) => {
                if (cancelled) {
                    decoded.close();
                    return;
                }
                loaded = decoded;
                bitmap = decoded;
            })
            .catch(() => {
                if (!cancelled) failed = true;
            });
        return () => {
            cancelled = true;
            loaded?.close();
        };
    });

    function drawImage(ctx: CanvasRenderingContext2D, size: number, image: ImageBitmap) {
        const k = size / VIEW;
        const w = image.width * scale * k;
        const h = image.height * scale * k;
        ctx.imageSmoothingQuality = 'high';
        ctx.drawImage(image, size / 2 - w / 2 + offsetX * k, size / 2 - h / 2 + offsetY * k, w, h);
    }

    // Attachment: reruns whenever the bitmap, zoom or offset change.
    function paint(canvas: HTMLCanvasElement) {
        if (!bitmap) return;
        const ctx = canvas.getContext('2d');
        if (!ctx) return;
        const r = VIEW / 2 - 1;
        ctx.clearRect(0, 0, VIEW, VIEW);
        drawImage(ctx, VIEW, bitmap);
        // Dim everything outside the circle that will be shown.
        ctx.fillStyle = 'rgba(0, 0, 0, 0.45)';
        ctx.beginPath();
        ctx.rect(0, 0, VIEW, VIEW);
        ctx.moveTo(VIEW / 2 + r, VIEW / 2);
        ctx.arc(VIEW / 2, VIEW / 2, r, 0, Math.PI * 2);
        ctx.fill('evenodd');
        ctx.strokeStyle = 'rgba(255, 255, 255, 0.85)';
        ctx.lineWidth = 1;
        ctx.beginPath();
        ctx.arc(VIEW / 2, VIEW / 2, r, 0, Math.PI * 2);
        ctx.stroke();
    }

    function onPointerDown(event: PointerEvent & { currentTarget: HTMLElement }) {
        if (!bitmap || busy) return;
        event.currentTarget.setPointerCapture(event.pointerId);
        drag = {
            pointerId: event.pointerId,
            x: event.clientX,
            y: event.clientY,
            startX: offsetX,
            startY: offsetY
        };
    }

    function onPointerMove(event: PointerEvent & { currentTarget: HTMLElement }) {
        if (!drag || drag.pointerId !== event.pointerId) return;
        const k = VIEW / event.currentTarget.getBoundingClientRect().width;
        setOffset(drag.startX + (event.clientX - drag.x) * k, drag.startY + (event.clientY - drag.y) * k);
    }

    function onPointerEnd(event: PointerEvent) {
        if (drag?.pointerId === event.pointerId) drag = null;
    }

    function onKeydown(event: KeyboardEvent) {
        if (!bitmap || busy) return;
        switch (event.key) {
            case 'ArrowLeft':
                setOffset(offsetX - NUDGE, offsetY);
                break;
            case 'ArrowRight':
                setOffset(offsetX + NUDGE, offsetY);
                break;
            case 'ArrowUp':
                setOffset(offsetX, offsetY - NUDGE);
                break;
            case 'ArrowDown':
                setOffset(offsetX, offsetY + NUDGE);
                break;
            case '+':
            case '=':
                setZoom(zoom + 0.1);
                break;
            case '-':
                setZoom(zoom - 0.1);
                break;
            default:
                return;
        }
        event.preventDefault();
    }

    function confirm() {
        if (!bitmap || busy) return;
        const out = document.createElement('canvas');
        out.width = outputSize;
        out.height = outputSize;
        const ctx = out.getContext('2d');
        if (!ctx) {
            failed = true;
            return;
        }
        // JPEG has no alpha: a transparent PNG would otherwise turn black.
        ctx.fillStyle = '#ffffff';
        ctx.fillRect(0, 0, outputSize, outputSize);
        drawImage(ctx, outputSize, bitmap);
        out.toBlob(
            (blob) => {
                if (blob) onconfirm(blob);
                else failed = true;
            },
            'image/jpeg',
            0.9
        );
    }
</script>

<div class="flex flex-col gap-4">
    <h3 class="text-base font-semibold">{m.settings_avatar_crop_title()}</h3>
    <p class="text-sm text-base-content/70">{m.settings_avatar_crop_hint()}</p>

    {#if failed}
        <Alert type="error" message={m.settings_avatar_error()} />
    {:else}
        <div class="mx-auto w-full max-w-xs">
            <button
                type="button"
                aria-label={m.settings_avatar_crop_title()}
                class="block aspect-square w-full touch-none select-none overflow-hidden rounded-box border border-base-300 bg-base-200 focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-primary {drag
                    ? 'cursor-grabbing'
                    : 'cursor-grab'}"
                onpointerdown={onPointerDown}
                onpointermove={onPointerMove}
                onpointerup={onPointerEnd}
                onpointercancel={onPointerEnd}
                onkeydown={onKeydown}
            >
                <canvas {@attach paint} width={VIEW} height={VIEW} class="size-full" aria-hidden="true"
                ></canvas>
            </button>
        </div>

        <label class="flex flex-col gap-1 text-sm font-medium" for="avatar-zoom">
            {m.settings_avatar_zoom()}
            <input
                id="avatar-zoom"
                type="range"
                class="range range-sm"
                min="1"
                max={MAX_ZOOM}
                step="0.01"
                value={zoom}
                disabled={!bitmap || busy}
                oninput={(event) => setZoom(event.currentTarget.valueAsNumber)}
            />
        </label>
    {/if}

    <div class="flex flex-wrap justify-end gap-2">
        <button type="button" class="btn" onclick={oncancel} disabled={busy}>
            <XIcon class="size-4" />
            {m.settings_avatar_crop_cancel()}
        </button>
        <button
            type="button"
            class="btn btn-primary"
            onclick={confirm}
            disabled={!bitmap || failed || busy}
        >
            {#if busy}
                <span class="loading loading-spinner"></span>
            {:else}
                <CheckIcon class="size-4" />
            {/if}
            {m.settings_avatar_crop_confirm()}
        </button>
    </div>
</div>
