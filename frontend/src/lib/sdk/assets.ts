import type { Client } from './client';
import { sdkHttpErrorFromResponse } from './httpError';
import type { SOPAsset } from './types';

export const assets = (client: Client) => ({
    
    /**
     * Upload a new asset.
     * URL: POST /api/sops/{sopId}/assets
     */
    upload: async (file: File, sopId: string): Promise<SOPAsset> => {
        const formData = new FormData();
        formData.append('file', file); // Only send the file in the body

        // Notice the updated URL template matching your new route
        const res = await client.fetch(`/sops/${sopId}/assets`, {
            method: 'POST',
            body: formData 
        });

        if (!res.ok) throw await sdkHttpErrorFromResponse(res);
        return await res.json();
    },

    /**
     * List assets for a specific SOP with body { "sop_id": "..." }
     */
    list: async (sopId: string): Promise<SOPAsset[]> => {
        const res = await client.fetch(`sops/${sopId}/assets`, { 
            method: 'GET'
        });
        
        if (!res.ok) throw new Error('FETCH_ASSETS_FAILED');
        return await res.json();
    },

    /**
     * Get asset metadata with body { "id": "..." }
     */
    getMetadata: async (id: string): Promise<SOPAsset> => {
        //todo: replace "not-in-use-yet" with sopId in the future for more granular sop access
        const sopId = "not-in-use-yet";
        const res = await client.fetch(`/sops/${sopId}/assets/${id}`, { 
            method: 'GET'
        });

        if (!res.ok) throw new Error('ASSET_NOT_FOUND');
        return await res.json();
    },

    /**
     * Helper to get the download URL.
     */
    getDownloadUrl: (id: string): string => {
        //todo: replace "not-in-use-yet" with sopId in the future for more granular sop access
        const sopId = "not-in-use-yet";
        return `/sops/${sopId}/assets/${id}/download`;
    },
    
    /**
     * Check asset integrity (Hash check).
    */
   checkIntegrity: async (sopId: string, assetId: string): Promise<{ hash_valid: boolean }> => {
        //todo: replace "not-in-use-yet" with sopId in the future for more granular sop access
        sopId = "not-in-use-yet";
        const res = await client.fetch(`/sops/${sopId}/assets/${assetId}/integrity`, {
            method: 'GET'
        });
        
        if (!res.ok) throw new Error('INTEGRITY_CHECK_FAILED');
        return await res.json();
    }
});