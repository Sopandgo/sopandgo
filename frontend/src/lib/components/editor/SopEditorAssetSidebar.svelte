<script lang="ts">
    import { enhance } from '$app/forms';
    import {
        PaperclipIcon,
        CopyIcon,
        FileImageIcon,
        UploadIcon,
        LoaderCircleIcon,
        FileUpIcon
    } from 'lucide-svelte';
    import type { SOPAsset } from '$lib/sdk/types';
    import type { ActionData } from '../../../routes/(app)/sops/[sop_id]/new/$types';

    let { assets = [], form, content = $bindable('') } = $props<{ 
        assets?: SOPAsset[] | null;
        form: ActionData;
        content?: string;
    }>();

    let safeAssets = $derived(assets ?? []);

    let uploading = $state(false);
    let fileInput: HTMLInputElement;
    let docxInput: HTMLInputElement | undefined = $state();
    let importing = $state(false);
    let importError = $state<string | null>(null);

    const canImportDocx = $derived(content.trim().length === 0);

    function copyToClipboard(text: string) {
        navigator.clipboard.writeText(text);
    }

    function openDocxPicker() {
        importError = null;
        if (!canImportDocx) return;
        docxInput?.click();
    }

    async function onDocxSelected(ev: Event) {
        const input = ev.currentTarget as HTMLInputElement;
        const file = input.files?.[0];
        input.value = '';
        if (!file) return;

        if (content.trim().length > 0) {
            importError = 'Clear the document before importing from Word.';
            return;
        }

        importing = true;
        importError = null;
        try {
            const { importDocxToMarkdown } = await import('$lib/utils/importDocxToMarkdown');
            const { markdown } = await importDocxToMarkdown(file);
            content = markdown;
        } catch (e) {
            const msg = e instanceof Error ? e.message : 'Could not import this file.';
            importError = msg;
        } finally {
            importing = false;
        }
    }
</script>

<div class="flex flex-col gap-6">
    <div class="card bg-base-100 shadow-sm border border-base-200">
        <div class="card-body p-4 gap-3">
            <h3 class="card-title text-sm uppercase tracking-wider opacity-70 flex items-center gap-2">
                <FileUpIcon class="w-4 h-4" />
                Import from Word
            </h3>
            <p class="text-xs text-base-content/60">
                Only when the draft is empty. Converts .docx to markdown in the editor.
            </p>
            <input
                bind:this={docxInput}
                type="file"
                class="sr-only"
                accept=".docx,application/vnd.openxmlformats-officedocument.wordprocessingml.document"
                onchange={onDocxSelected}
                aria-hidden="true"
                tabindex="-1"
            />
            <button
                type="button"
                class="btn btn-secondary btn-block"
                disabled={!canImportDocx || importing}
                title={canImportDocx ? 'Import a Word document (.docx)' : 'Clear the document to import from Word'}
                onclick={openDocxPicker}
            >
                {#if importing}
                    <LoaderCircleIcon class="w-5 h-5 shrink-0 animate-spin opacity-70" />
                {:else}
                    <FileUpIcon class="w-5 h-5 shrink-0" />
                {/if}
                Choose .docx file
            </button>
            {#if !canImportDocx}
                <p class="text-xs text-base-content/60">
                    Clear the document in the editor to import again.
                </p>
            {/if}
            {#if importError}
                <div role="alert" class="alert alert-warning text-xs py-2">
                    <span class="whitespace-pre-wrap">{importError}</span>
                </div>
            {/if}
        </div>
    </div>

    <div class="card bg-base-100 shadow-sm border border-base-200">
        <div class="card-body p-4">
            <h3 class="card-title text-sm uppercase tracking-wider opacity-70 flex items-center gap-2">
                <PaperclipIcon class="w-4 h-4" />
                Assets
            </h3>
            
            <div class="divider my-1"></div>

            {#if form?.uploadError}
                <div class="alert alert-error text-xs p-2 mb-2 rounded-md">
                    <span class="whitespace-pre-wrap">{form.uploadError}</span>
                </div>
            {/if}

            {#if safeAssets.length > 0}
                <ul class="flex flex-col gap-2 max-h-[400px] overflow-y-auto pr-1 custom-scrollbar">
                    {#each safeAssets as asset}
                        <li class="flex items-center justify-between p-2 rounded-lg hover:bg-base-200 group border border-transparent hover:border-base-300 transition-all">
                            <div class="flex items-center gap-3 overflow-hidden">
                                <div class="bg-base-200 p-2 rounded-md text-primary">
                                    <FileImageIcon class="w-5 h-5" />
                                </div>
                                <div class="flex flex-col min-w-0">
                                    <span class="font-medium text-sm truncate max-w-[150px]" title={asset.file_name}>
                                        {asset.file_name}
                                    </span>
                                    <span class="text-xs opacity-50 truncate">
                                        {new Date(asset.created_at).toLocaleDateString()}
                                    </span>
                                </div>
                            </div>
                            
                            <button 
                                type="button"
                                class="btn btn-ghost btn-xs btn-square opacity-0 group-hover:opacity-100 transition-opacity"
                                title="Copy Markdown Link"
                                onclick={() => copyToClipboard(`![${asset.file_name}](assets/${encodeURIComponent(asset.file_name)})`)}
                            >
                                <CopyIcon class="w-4 h-4" />
                            </button>
                        </li>
                    {/each}
                </ul>
            {:else}
                <div class="text-center py-8 opacity-50">
                    <p class="text-sm">No assets yet.</p>
                    <p class="text-xs mt-1">Use Upload Asset below.</p>
                </div>
            {/if}

            <div class="divider my-1"></div>

            <button 
                type="button" 
                class="btn btn-secondary" 
                aria-label="Upload Asset"
                onclick={() => fileInput.click()}
                disabled={uploading}
            >
                {#if uploading}
                    <LoaderCircleIcon class="h-5 w-5 shrink-0 animate-spin" />
                {:else}
                    <UploadIcon class="h-5 w-5 shrink-0" />
                {/if}
                Upload Asset
            </button>
        </div>
    </div>
    
    <form 
        method="POST" 
        action="?/upload" 
        enctype="multipart/form-data"
        use:enhance={() => {
            uploading = true;
            return async ({ update }) => {
                await update();
                uploading = false;
                if (fileInput) fileInput.value = ''; 
            };
        }}
        class="hidden"
    >
        <input 
            bind:this={fileInput}
            type="file" 
            name="file" 
            accept="image/*,application/pdf"
            onchange={(e) => {
                 if (e.currentTarget.files?.length) {
                     e.currentTarget.form?.requestSubmit();
                 }
            }}
        />
    </form>
    
    <div class="alert shadow-sm text-sm">
        <span>
            <strong>Tip:</strong> Uploading here refreshes the list instantly.
        </span>
    </div>
</div>