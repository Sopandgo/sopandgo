import { describe, expect, it } from 'vitest';
import type { SOPAsset } from '#lib/sdk/types.js';
import { assetUrl, findAsset, imagesInLine } from './sopAssets';

const asset = (id: string, file_name: string): SOPAsset => ({ id, file_name, content_hash: '', created_at: '' });
const assets = [asset('a1', 'Gel Image.png'), asset('a2', 'stain.jpg')];

describe('findAsset', () => {
    it('matches assets/ hrefs by decoded file name, ignoring case', () => {
        expect(findAsset('assets/Gel%20Image.png', assets)?.id).toBe('a1');
        expect(findAsset('assets/STAIN.JPG', assets)?.id).toBe('a2');
    });

    it('ignores other hrefs, unknown names and broken escapes', () => {
        expect(findAsset('https://example.com/stain.jpg', assets)).toBeUndefined();
        expect(findAsset('assets/missing.png', assets)).toBeUndefined();
        expect(findAsset('assets/%E0%A4%A.png', assets)).toBeUndefined();
        expect(findAsset('assets/stain.jpg', null)).toBeUndefined();
    });
});

describe('imagesInLine', () => {
    it('finds every uploaded image in a line, with its alt text', () => {
        const line = '| ![Before](assets/Gel%20Image.png) | ![](<assets/stain.jpg> "After") |';
        expect(imagesInLine(line, assets)).toEqual([
            { asset: assets[0], alt: 'Before' },
            { asset: assets[1], alt: 'stain.jpg' }
        ]);
    });

    it('leaves out external images, unknown assets and plain links', () => {
        const line = '![x](https://example.com/a.png) ![y](assets/missing.png) [z](assets/stain.jpg)';
        expect(imagesInLine(line, assets)).toEqual([]);
    });
});

describe('assetUrl', () => {
    it('points at the download route', () => {
        expect(assetUrl(asset('id 1', 'x.png'))).toBe('/api/assets/download?id=id%201');
    });
});
