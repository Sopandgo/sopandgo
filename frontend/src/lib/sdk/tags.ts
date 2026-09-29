import type { Client } from './client';
import type { Tag } from './types';

export const tags = (client: Client) => ({
    
    /**
     * List all global tags available in the system.
     */
    list: async (): Promise<Tag[]> => {
        const res = await client.fetch('/tags', {
            method: 'GET'
        });
        if (!res.ok) throw new Error('FETCH_TAGS_FAILED');
        return await res.json();
    },

    /**
     * Create a new global tag.
     */
    create: async (title: string): Promise<{ id: string }> => {
        const res = await client.fetch('/tags', {
            method: 'POST',
            body: { title }
        });
        // Throw a specific error for conflicts so the UI can catch it easily
        if (res.status === 409) throw new Error('TAG_ALREADY_EXISTS');
        if (!res.ok) throw new Error('CREATE_TAG_FAILED');
        return await res.json();
    },

    /**
     * Retire or revive a global tag.
     */
    setStatus: async (tagId: string, is_active: boolean): Promise<void> => {
        const res = await client.fetch(`/tags/${tagId}/status`, {
            method: 'PATCH',
            body: { is_active }
        });
        if (!res.ok) throw new Error('UPDATE_TAG_STATUS_FAILED');
    }
});