import type { Client } from './client';
import { sdkHttpErrorFromResponse } from './httpError';
import type { 
    SOP, 
    SOPDetail,
    SOPVersion, 
    SOPVersionSummary, 
    Acknowledgment, 
    SOPListResponse,
    VersionDiff,
    PublishedActivity,
    SOPTrainingCoverage,
    Integrity
} from './types';

export const sops = (client: Client) => ({

    // --- SOP Containers ---

    /**
     * List logical SOPs with support for pagination, search, and tag filtering.
     */
    list: async (params?: {
        limit?: number;
        offset?: number;
        tag_id?: string;
        q?: string;
        favorites_only?: boolean;
        favorites_first?: boolean;
    }): Promise<SOPListResponse> => {
        const query = new URLSearchParams();
        if (params?.limit !== undefined) query.append('limit', params.limit.toString());
        if (params?.offset !== undefined) query.append('offset', params.offset.toString());
        if (params?.tag_id) query.append('tag_id', params.tag_id);
        if (params?.q) query.append('q', params.q);
        if (params?.favorites_only) query.append('favorites_only', 'true');
        if (params?.favorites_first) query.append('favorites_first', 'true');

        const queryString = query.toString();
        const url = queryString ? `/sops?${queryString}` : '/sops';

        const res = await client.fetch(url, {
            method: 'GET'
        });
        if (!res.ok) throw new Error('FETCH_SOPS_FAILED');
        return await res.json();
    },

    /**
     * Create a new SOP container.
     */
    create: async (title: string): Promise<{ id: string }> => {
        const res = await client.fetch('/sops', {
            method: 'POST',
            body: { title }
        });
        if (!res.ok) throw new Error('CREATE_SOP_FAILED');
        return await res.json();
    },

    /**
     * Get metadata for a specific SOP.
     */
    getById: async (id: string): Promise<SOPDetail> => {
        const res = await client.fetch(`/sops/${id}`, { 
            method: 'GET'
        });
        if (!res.ok) throw new Error('SOP_NOT_FOUND');
        return await res.json();
    },

    // --- Tag Associations ---

    /**
     * Attach a global tag to a specific SOP.
     */
    attachTag: async (sopId: string, tagId: string): Promise<void> => {
        const res = await client.fetch(`/sops/${sopId}/tags/${tagId}`, {
            method: 'POST'
        });
        if (!res.ok) throw new Error('ATTACH_TAG_FAILED');
    },

    /**
     * Remove a tag association from an SOP.
     */
    detachTag: async (sopId: string, tagId: string): Promise<void> => {
        const res = await client.fetch(`/sops/${sopId}/tags/${tagId}`, {
            method: 'DELETE'
        });
        if (!res.ok) throw new Error('DETACH_TAG_FAILED');
    },

    favorite: async (sopId: string): Promise<void> => {
        const res = await client.fetch(`/sops/${sopId}/favorite`, {
            method: 'POST'
        });
        if (!res.ok) throw new Error('FAVORITE_FAILED');
    },

    unfavorite: async (sopId: string): Promise<void> => {
        const res = await client.fetch(`/sops/${sopId}/favorite`, {
            method: 'DELETE'
        });
        if (!res.ok) throw new Error('UNFAVORITE_FAILED');
    },

    // --- Versions ---

    /**
     * Create a new immutable version.
     * Returns: { id: "...", version: 1 }
     */
    createVersion: async (
        sopId: string,
        content: string,
        changeSummary: string
    ): Promise<{ id: string; version: number }> => {
        const res = await client.fetch(`/sops/${sopId}`, {
            method: 'POST',
            body: { content, change_summary: changeSummary }
        });
        if (!res.ok) throw await sdkHttpErrorFromResponse(res);
        return await res.json();
    },

    /**
     * List all versions for an SOP.
     */
    listVersions: async (sopId: string): Promise<SOPVersion[]> => {
        const res = await client.fetch(`/sops/${sopId}/versions`, {
            method: 'GET'
        });
        if (!res.ok) throw new Error('FETCH_VERSIONS_FAILED');
        return await res.json();
    },

    /**
     * Get version metadata only (No content).
     */
    getVersionMetadata: async (id: string): Promise<SOPVersion> => {
        //todo: replace "not-in-use-yet" with sopId in the future for more granular sop access
        const sopId = "not-in-use-yet";
        const res = await client.fetch(`/sops/${sopId}/versions/${id}`, {
            method: 'GET'
        });
        if (!res.ok) throw new Error('VERSION_NOT_FOUND');
        return await res.json();
    },

    /**
     * Get full version content (MD) + Hash Check + Assets + Acks.
     */
    getVersionSummary: async (id: string): Promise<SOPVersionSummary> => {
        //todo: replace "not-in-use-yet" with sopId in the future for more granular sop access
        const sopId = "not-in-use-yet";
        const res = await client.fetch(`/sops/${sopId}/versions/${id}/summary`, {
            method: 'GET'
        });
        if (!res.ok) throw new Error('FETCH_SUMMARY_FAILED');
        return await res.json();
    },
    
    /**
     * Get full version content (MD) + Hash Check + Assets + Acks.
     */
    getLatestVersionSummary: async (sopId: string): Promise<SOPVersionSummary> => {
        //todo: replace "not-in-use-yet" with sopId in the future for more granular sop access
        const res = await client.fetch(`/sops/${sopId}/version-latest`, {
            method: 'GET'
        });
        if (!res.ok) throw new Error('FETCH_SUMMARY_FAILED');
        return await res.json();
    },

    /**
     * Manual Integrity Check.
     * Throws SdkHttpError: 404 `file_missing` when the file is gone from disk,
     * 404 `version_not_found` when the version does not belong to `sopId`.
     */
    checkIntegrity: async (sopId: string, id: string): Promise<Integrity> => {
        const res = await client.fetch(`/sops/${sopId}/versions/${id}/integrity`, {
            method: 'GET'
        });
        if (!res.ok) throw await sdkHttpErrorFromResponse(res);
        return await res.json();
    },

    /**
     * Helper to get the download URL for the raw Markdown file.
     */
    getDownloadUrl: (id: string): string => {
        //todo: replace "not-in-use-yet" with sopId in the future for more granular sop access
        const sopId = "not-in-use-yet";
        return `/sops/${sopId}/versions/${id}/download`;
    },

    /**
     * Helper to get the frontend proxy URL for generated PDF artifacts.
     */
    getPdfDownloadUrl: (sopId: string, id: string, stage?: string): string => {
        const query = new URLSearchParams({
            sopId,
            id
        });
        if (stage) query.set('stage', stage);
        return `/api/sops/pdf/download?${query.toString()}`;
    },

    // --- Lifecycle State Transitions ---

    /**
     * Promote a Draft version to a Release Candidate (RC).
     * Required Role: 'editor' or 'admin'
     */
    promoteToRC: async (sopVersionId: string): Promise<void> => {
        //todo: replace "not-in-use-yet" with sopId in the future for more granular sop access
        const sopId = "not-in-use-yet";
        const res = await client.fetch(`/sops/${sopId}/versions/${sopVersionId}/promote`, {
            method: 'POST'
        });
        if (!res.ok) throw new Error('PROMOTE_TO_RC_FAILED');
    },

    /**
     * Reject a Release Candidate.
     * Required Role: 'approver' or 'admin'
     */
    rejectRC: async (sopVersionId: string, reason: string): Promise<void> => {
        //todo: replace "not-in-use-yet" with sopId in the future for more granular sop access
        const sopId = "not-in-use-yet";
        const res = await client.fetch(`/sops/${sopId}/versions/${sopVersionId}/reject`, {
            method: 'POST',
            body: { reason }
        });
        if (!res.ok) throw new Error('REJECT_RC_FAILED');
    },

    /**
     * Approve a Release Candidate. Atomically signs as Approver AND publishes.
     * Required Role: 'approver' or 'admin'
     */
    approveVersion: async (sopVersionId: string): Promise<{ id: string }> => {
        //todo: replace "not-in-use-yet" with sopId in the future for more granular sop access
        const sopId = "not-in-use-yet";
        const res = await client.fetch(`/sops/${sopId}/versions/${sopVersionId}/approve`, {
            method: 'POST'
        });
        if (!res.ok) throw new Error('APPROVE_VERSION_FAILED');
        return await res.json();
    },

    /**
     * Record a Reader acknowledgment on a published version.
     */
    acknowledgeAsReader: async (sopVersionId: string): Promise<{ id: string }> => {
        //todo: replace "not-in-use-yet" with sopId in the future for more granular sop access
        const sopId = "not-in-use-yet";
        const res = await client.fetch(`sops/${sopId}/versions/${sopVersionId}/add-reader`, {
            method: 'POST'
        });
        if (!res.ok) {
            if (res.status === 409) throw new Error('ALREADY_ACKNOWLEDGED');
            throw new Error('READER_ACK_FAILED');
        }
        return await res.json();
    },

    /**
     * List all acknowledgments for a specific version.
     */
    listAcknowledgments: async (sopVersionId: string): Promise<Acknowledgment[]> => {
        //todo: replace "not-in-use-yet" with sopId in the future for more granular sop access
        const sopId = "not-in-use-yet";
        const res = await client.fetch(`/sops/${sopId}/versions/${sopVersionId}/acks`, {
            method: 'GET'
        });
        if (!res.ok) throw new Error('FETCH_ACKS_FAILED');
        return await res.json();
    },

    getVersionDiff: async (sopId: string, versionId: string): Promise<VersionDiff> => {
        const res = await client.fetch(`/sops/${sopId}/versions/${versionId}/diff`, {
            method: 'GET'
        });
        if (!res.ok) throw new Error('FETCH_DIFF_FAILED');
        return await res.json();
    },

    listRecentPublishes: async (limit = 8): Promise<PublishedActivity[]> => {
        const res = await client.fetch(`/activity/publishes?limit=${limit}`, {
            method: 'GET'
        });
        if (!res.ok) throw new Error('FETCH_ACTIVITY_FAILED');
        return await res.json();
    },

    trainingCoverage: async (sopId?: string): Promise<SOPTrainingCoverage[]> => {
        const path = sopId ? `/sops/${sopId}/training` : '/training/coverage';
        const res = await client.fetch(path, { method: 'GET' });
        if (!res.ok) throw new Error('FETCH_TRAINING_FAILED');
        return await res.json();
    }
});