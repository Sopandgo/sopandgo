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
        // Admins see role and email hints in the actor filter. Auditors cannot
        // list users, so they pick from the actors the log itself names.
        const isAdmin = locals.user?.role === 'admin';
        const [auditRes, users, sopList, auditFilterOptions] = await Promise.all([
            locals.api.admin.getAuditLogs(logsPerPage, offset, filters),
            isAdmin ? locals.api.admin.listUsers() : Promise.resolve(null),
            // Archived SOPs keep their audit history, so the filter lists them too
            locals.api.sops.list({ limit: 500, offset: 0, archived: 'all' }),
            locals.api.admin.getAuditFilterOptions()
        ]);
        
        return { 
            auditEvents: auditRes.events,
            total: auditRes.total,
            page,
            logsPerPage,
            filters,
            actors: users
                ? users.map((u) => ({ id: u.id, display_name: u.display_name, role: u.role, email: u.email }))
                : auditFilterOptions.actors.map((a) => ({ ...a, role: undefined, email: undefined })),
            sops: sopList.sops,
            auditFilterOptions
        };
    } catch (err) {
        console.error('Failed to load audit logs:', err);
        throw error(500, 'Failed to load audit logs');
    }
}