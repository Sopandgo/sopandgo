import { describe, expect, it } from 'vitest';
import { favoriteErrorFromForm } from './favorites';

describe('favoriteErrorFromForm', () => {
    it('returns the error for the SOP the action ran on', () => {
        const form = { favorite: { sopId: 'a', error: 'Failed to favorite' } };
        expect(favoriteErrorFromForm(form, 'a')).toBe('Failed to favorite');
    });

    it('ignores errors for other SOPs', () => {
        expect(favoriteErrorFromForm({ favorite: { sopId: 'a', error: 'x' } }, 'b')).toBeNull();
    });

    it('returns null after a success, another action, or no action', () => {
        expect(favoriteErrorFromForm({ favorite: { sopId: 'a' } }, 'a')).toBeNull();
        expect(favoriteErrorFromForm({ message: 'other' }, 'a')).toBeNull();
        expect(favoriteErrorFromForm(null, 'a')).toBeNull();
        expect(favoriteErrorFromForm(undefined, 'a')).toBeNull();
    });
});
