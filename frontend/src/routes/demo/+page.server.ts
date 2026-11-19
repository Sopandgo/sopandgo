import { error, redirect } from '@sveltejs/kit';
import { otherDemoRole, type DemoRole } from '#lib/demo.js';
import { visitorAddress } from '#lib/server/visitorAddress.js';
import type { Actions, PageServerLoad } from './$types';

// Only the demo banner's form action lives here.
export const load: PageServerLoad = () => {
    throw redirect(303, '/');
};

export const actions: Actions = {
    /** Signs out of the current demo session and opens one for the chosen role. */
    switch: async (event) => {
        const { locals, request } = event;
        if (!locals.demo) throw error(404, 'Not found');

        const asked = String((await request.formData()).get('role') ?? '');
        const role: DemoRole =
            asked === 'viewer' || asked === 'auditor' ? asked : otherDemoRole(locals.user?.role ?? '');

        await locals.api.auth.logout();
        try {
            await locals.api.auth.openDemoSession(role, visitorAddress(event));
        } catch (err) {
            console.error('Demo role switch failed:', err);
            // Without a session, the next page opens a researcher session again,
            // and the banner says the switch failed.
            throw redirect(303, '/sops?demo_switch=failed');
        }
        throw redirect(303, role === 'auditor' ? '/audit-logs' : '/sops');
    }
};
