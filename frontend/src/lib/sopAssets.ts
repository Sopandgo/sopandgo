import type { SOPAsset } from '$lib/sdk/types';

/*
 * SOP Markdown points at uploaded files as `assets/<file name>`. File names are
 * unique per SOP and assets are never rewritten, so a name always means the same
 * file, whichever version refers to it.
 */

/** The asset an `assets/…` href points at, or undefined for other hrefs and unknown names. */
export function findAsset(href: string, assets: SOPAsset[] | null | undefined): SOPAsset | undefined {
    if (!href.startsWith('assets/')) return undefined;
    let name: string;
    try {
        name = decodeURIComponent(href.slice('assets/'.length));
    } catch {
        return undefined;
    }
    const target = name.toLowerCase();
    return (assets ?? []).find((a) => a.file_name.toLowerCase() === target);
}

/** Where the browser loads an asset from. */
export function assetUrl(asset: SOPAsset): string {
    return `/api/assets/download?id=${encodeURIComponent(asset.id)}`;
}

export interface LineImage {
    asset: SOPAsset;
    alt: string;
}

// ![alt](href) or ![alt](<href> "title"); alt may not contain `]`.
const imagePattern = /!\[([^\]]*)\]\(\s*<?([^)\s>]+)>?(?:\s+"[^"]*")?\s*\)/g;

/** The uploaded images one line of Markdown shows, in order. External images are left out. */
export function imagesInLine(line: string, assets: SOPAsset[] | null | undefined): LineImage[] {
    const out: LineImage[] = [];
    for (const match of line.matchAll(imagePattern)) {
        const asset = findAsset(match[2], assets);
        if (asset) out.push({ asset, alt: match[1] || asset.file_name });
    }
    return out;
}
