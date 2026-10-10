import { error, fail } from '@sveltejs/kit';
import * as m from '$lib/paraglide/messages.js';
import type { Actions, PageServerLoad } from './$types';

export const load: PageServerLoad = async ({ locals }) => {
	const user = locals.user;
	if (!user) {
		return {
			user: null,
			signatureStatus: [],
			favorites: { sops: [], total: 0 },
			recentPublishes: [],
			trainingCoverage: []
		};
	}

	const canSeeTraining = user.role === 'admin' || user.role === 'approver';

	try {
		const [signatureStatus, favorites, recentPublishes, trainingCoverage] = await Promise.all([
			locals.api.auth.getSignatureStatus(),
			locals.api.sops.list({ favorites_only: true, limit: 24, offset: 0 }),
			locals.api.sops.listRecentPublishes(8),
			canSeeTraining ? locals.api.sops.trainingCoverage() : Promise.resolve([])
		]);

		return {
			user,
			signatureStatus,
			favorites,
			recentPublishes,
			trainingCoverage
		};
	} catch (err) {
		console.error('Dashboard load error:', err);
		throw error(500, m.error_service_unavailable());
	}
};

export const actions: Actions = {
	favorite: async ({ locals, request }) => {
		const form = await request.formData();
		const sopId = String(form.get('sop_id') ?? '').trim();
		if (!sopId) return fail(400, { message: m.error_missing_sop() });
		try {
			await locals.api.sops.favorite(sopId);
			return { success: true };
		} catch (err) {
			console.error('favorite', err);
			return fail(500, { message: m.error_favorite_failed() });
		}
	},
	unfavorite: async ({ locals, request }) => {
		const form = await request.formData();
		const sopId = String(form.get('sop_id') ?? '').trim();
		if (!sopId) return fail(400, { message: m.error_missing_sop() });
		try {
			await locals.api.sops.unfavorite(sopId);
			return { success: true };
		} catch (err) {
			console.error('unfavorite', err);
			return fail(500, { message: m.error_unfavorite_failed() });
		}
	}
};
