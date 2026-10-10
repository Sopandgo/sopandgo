<script lang="ts">
    import { marked } from 'marked';
    import type { SOPAsset } from '#lib/sdk/types.js';
    import { sanitizeSopHtml } from '#lib/security/sanitizeSopHtml.js';
    import { assetUrl, findAsset } from '#lib/sopAssets.js';
    import * as m from '#lib/paraglide/messages.js';

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
                // Same lookup as the Changes tab, so both show the same file
                const asset = findAsset(safeHref, assets);

                if (asset) {
                    return `
                        <figure class="not-prose my-10 flex flex-col items-center">
                            <img 
                                src="${assetUrl(asset)}" 
                                alt="${text || asset.file_name}" 
                                title="${title || ''}"
                                class="rounded-box border border-base-300" 
                            />
                            ${text ? `<figcaption class="mt-4 text-sm text-base-content/70 italic">${m.figure_caption({ text })}</figcaption>` : ''}
                        </figure>
                    `;
                }
                console.warn(`Asset not found for: ${safeHref}`);
            }

            return `<img src="${safeHref}" alt="${text || ''}" />`;
        };

        const raw = marked.parse(content, { renderer, async: false });
        return sanitizeSopHtml(typeof raw === 'string' ? raw : '');
    });
</script>

<div class="prose max-w-none prose-headings:font-semibold">
    {@html html}
</div>
