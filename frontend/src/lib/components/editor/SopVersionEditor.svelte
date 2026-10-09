<script lang="ts">
    import Alert from '$lib/components/Alert.svelte';
    import Card from '$lib/components/Card.svelte';
    import { enhance } from '$app/forms';
    import MarkdownRenderer from '$lib/components/MarkdownRenderer.svelte';
    import type { SOPAsset } from '$lib/sdk/types';
    import { EyeIcon, FilePenIcon, LoaderCircleIcon, RocketIcon } from 'lucide-svelte';
    import * as m from '$lib/paraglide/messages.js';

    interface FormResult {
        content?: string;
        change_summary?: string;
        message?: string;
        ok?: boolean;
    }

    let { sopId, form, assets = [], content = $bindable('') } = $props<{ 
        sopId: string;
        form: FormResult | null;
        assets?: SOPAsset[];
        content?: string;
    }>();

    let publishing = $state(false);
    let changeSummary = $derived(form?.change_summary ?? '');
    
    let previewContent = $state('');
    /** Switch between markdown source and rendered preview (full width of the editor card) */
    let editorTab = $state<'edit' | 'preview'>('edit');

    // This effect handles both the initial load AND subsequent form errors
    $effect(() => {
        if (form?.content) {
            content = form.content;
            previewContent = form.content;
        }
    });

    const PREVIEW_DEBOUNCE_MS = 200;

    $effect(() => {
        const next = content;
        const id = setTimeout(() => {
            previewContent = next;
        }, PREVIEW_DEBOUNCE_MS);
        return () => clearTimeout(id);
    });
</script>

<div class="h-full">
    {#if form?.message}
        <Alert variant="error" message={form.message} class="mb-6 shadow-sm text-sm" />
    {/if}

    <Card class="h-full">
    <form 
        method="POST" 
        action="?/publish" 
        class="h-full"
        use:enhance={() => {
            publishing = true;
            return async ({ update }) => {
                await update();
                publishing = false;
            };
        }}
    >
        <div class="card-body gap-4 flex flex-col h-full min-h-0">
            <h3 class="card-title text-sm uppercase tracking-wider opacity-70 flex items-center gap-2 shrink-0">
                <FilePenIcon class="w-4 h-4" />
                {m.editor_document()}
            </h3>

            <label class="form-control w-full shrink-0">
                <span class="label-text font-medium">{m.editor_what_changed()}</span>
                <input
                    id="change_summary"
                    name="change_summary"
                    bind:value={changeSummary}
                    class="input input-bordered w-full"
                    maxlength="500"
                    required
                    placeholder={m.editor_change_placeholder()}
                />
                <span class="label-text-alt text-base-content/60 pt-1">
                    {m.editor_change_help()}
                </span>
            </label>

            <div role="tablist" class="tabs tabs-boxed w-full shrink-0" aria-label={m.editor_view()}>
                <button
                    type="button"
                    role="tab"
                    aria-selected={editorTab === 'edit'}
                    id="sop-editor-tab-edit"
                    class="tab gap-1.5 grow sm:grow-0 {editorTab === 'edit' ? 'tab-active' : ''}"
                    onclick={() => (editorTab = 'edit')}
                >
                    <FilePenIcon class="w-4 h-4 shrink-0" />
                    {m.editor_edit()}
                </button>
                <button
                    type="button"
                    role="tab"
                    aria-selected={editorTab === 'preview'}
                    id="sop-editor-tab-preview"
                    class="tab gap-1.5 grow sm:grow-0 {editorTab === 'preview' ? 'tab-active' : ''}"
                    onclick={() => (editorTab = 'preview')}
                >
                    <EyeIcon class="w-4 h-4 shrink-0" />
                    {m.editor_preview()}
                </button>
            </div>

            <div class="flex flex-col flex-grow min-h-0 w-full">
                <div
                    class="form-control flex flex-col w-full flex-grow min-h-0 {editorTab === 'edit' ? '' : 'hidden'}"
                    role="tabpanel"
                    aria-labelledby="sop-editor-tab-edit"
                    id="sop-editor-panel-edit"
                >
                    <textarea 
                        id="content"
                        name="content" 
                        bind:value={content}
                        class="textarea w-full font-mono flex-grow min-h-[min(55vh,520px)] text-sm leading-relaxed" 
                        placeholder={m.editor_placeholder()}
                        required
                    ></textarea>
                    
                    <div class="label pt-2">
                        <span class="label-text-alt text-base-content/60">
                            {m.editor_asset_hint()} <code>![Alt Text](assets/filename.png)</code>
                        </span>
                    </div>
                </div>

                <div
                    class="flex flex-col flex-grow min-h-0 border border-base-200 rounded-box bg-base-100 overflow-hidden {editorTab === 'preview' ? '' : 'hidden'}"
                    role="tabpanel"
                    aria-labelledby="sop-editor-tab-preview"
                    id="sop-editor-panel-preview"
                >
                    <div
                        class="text-xs opacity-60 tracking-widest uppercase font-bold px-4 py-2 border-b border-base-200 bg-base-200/30 shrink-0"
                    >
                        {m.editor_preview()}
                    </div>
                    <div
                        class="p-4 overflow-y-auto flex-grow min-h-[min(55vh,520px)]"
                        role="region"
                        aria-label={m.editor_preview_region()}
                        tabindex="-1"
                    >
                        {#if !previewContent.trim()}
                            <p class="text-sm text-base-content/50 italic">
                                {m.editor_preview_empty()}
                            </p>
                        {:else}
                            <MarkdownRenderer content={previewContent} {assets} />
                        {/if}
                    </div>
                </div>
            </div>

            <div class="card-actions justify-end mt-auto pt-4 border-t border-base-200">
                <a href={`/sops/${sopId}`} class="btn btn-ghost">{m.common_cancel()}</a>
                <button type="submit" class="btn btn-accent min-w-[150px] flex items-center justify-center gap-2" disabled={publishing}>
                    {#if publishing}
                        <LoaderCircleIcon class="animate-spin w-4 h-4 mr-2"/>
                    {/if}
                    <RocketIcon class="w-5 h-5" />
                    {m.editor_publish()}
                </button>
            </div>
        </div>
    </form>
    </Card>
</div>