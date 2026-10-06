<script lang="ts">
    import { marked } from 'marked';
    import type { SOPAsset } from '$lib/sdk/types';
    import { sanitizeSopHtml } from '$lib/security/sanitizeSopHtml';
    import * as m from '$lib/paraglide/messages.js';

    interface Props {
        content?: string;
        assets?: SOPAsset[] | null;
    }

    let { content = '', assets = [] }: Props = $props();

    const html = $derived.by(() => {
        if (!content) return '';

        const renderer = new marked.Renderer();

        renderer.image = ({ href, text, title }) => {
            const safeHref = typeof href === 'string' ? href : '';
            
            // Match "assets/filename.png"
            if (safeHref.startsWith('assets/')) {
                // Decode and clean the filename from the markdown link
                const targetName = decodeURIComponent(safeHref.replace('assets/', '')).toLowerCase();
                
                // Simple, direct lookup in the SDK-provided list
                const asset = (assets ?? []).find(a => a.file_name.toLowerCase() === targetName);
                
                if (asset) {
                    return `
                        <figure class="not-prose my-10 flex flex-col items-center">
                            <img 
                                src="/api/assets/download?id=${asset.id}" 
                                alt="${text || asset.file_name}" 
                                title="${title || ''}"
                                class="rounded-xl border border-base-300 shadow-lg" 
                            />
                            ${text ? `<figcaption class="mt-4 text-sm opacity-60 italic">${m.figure_caption({ text })}</figcaption>` : ''}
                        </figure>
                    `;
                }
                console.warn(`Asset not found for filename: ${targetName}`);
            }

            return `<img src="${safeHref}" alt="${text || ''}" />`;
        };

        const raw = marked.parse(content, { renderer, async: false });
        return sanitizeSopHtml(typeof raw === 'string' ? raw : '');
    });
</script>

<div class="prose prose-slate max-w-none">
    {@html html}
</div>