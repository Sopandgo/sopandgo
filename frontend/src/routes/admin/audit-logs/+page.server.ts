import { error } from "@sveltejs/kit";
import type { PageServerLoad } from "./$types";

export const load: PageServerLoad = async ({ locals, url }) => {
    const page = Number(url.searchParams.get('page')) || 0;
    const logsPerPage = 10;
    const offset = page * logsPerPage;
    const filters = {
        type: url.searchParams.get('type') || undefined,
        entity_type: url.searchParams.get('entity_type') || undefined,
        actor_user_id: url.searchParams.get('actor_user_id') || undefined,
        sop_id: url.searchParams.get('sop_id') || undefined
    };

    try {
        const [auditRes, users, sopList, auditFilterOptions] = await Promise.all([
            locals.api.admin.getAuditLogs(logsPerPage, offset, filters),
            locals.api.admin.listUsers(),
            locals.api.sops.list({ limit: 500, offset: 0 }),
            locals.api.admin.getAuditFilterOptions()
        ]);
        
        return { 
            auditEvents: auditRes.events,
            total: auditRes.total,
            page,
            logsPerPage,
            filters,
            users,
            sops: sopList.sops,
            auditFilterOptions
        };
    } catch (err) {
        console.error('Failed to load audit logs:', err);
        throw error(500, 'Failed to load audit logs');
    }
}