import { redirect } from '@sveltejs/kit';
import type { PageServerLoad } from './$types';

// The audit log moved out of the admin area so auditors can read it.
// Old links and bookmarks keep working, filters included.
export const load: PageServerLoad = ({ url }) => {
    throw redirect(308, `/audit-logs${url.search}`);
};
