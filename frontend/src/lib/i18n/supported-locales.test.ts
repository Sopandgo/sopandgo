import { readFileSync } from 'node:fs';
import { resolve } from 'node:path';
import { describe, expect, it } from 'vitest';

describe('supported locales', () => {
    it('matches Paraglide language tags to i18n/supported-locales.json', () => {
        const listPath = resolve(process.cwd(), '../i18n/supported-locales.json');
        const settingsPath = resolve(process.cwd(), 'project.inlang/settings.json');
        const supported = JSON.parse(readFileSync(listPath, 'utf8')) as string[];
        const settings = JSON.parse(readFileSync(settingsPath, 'utf8')) as { locales: string[] };
        expect(settings.locales).toEqual(supported);
        expect(supported).toEqual(['en', 'de', 'fr', 'es', 'pt', 'zh', 'it', 'nl', 'pl', 'ja', 'ko', 'tr', 'sv', 'cs', 'sk']);
    });
});
