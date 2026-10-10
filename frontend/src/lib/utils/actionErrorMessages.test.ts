import { describe, expect, it } from 'vitest';
import * as m from '#lib/paraglide/messages.js';
import { SdkHttpError } from '#lib/sdk/httpError.js';
import { messageForPublishFailure } from './actionErrorMessages';

describe('publish failure messages', () => {
    it('resolves a known policy code through the catalog', () => {
        const err = new SdkHttpError(400, 'bad', { code: 'raw_html' });
        expect(messageForPublishFailure(err)).toBe(m.error_raw_html());
        expect(m.error_raw_html({}, { locale: 'de' })).toContain('Markdown');
    });
});
