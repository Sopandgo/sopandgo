import { isRedirect } from '@sveltejs/kit';
import { greetingPeriodForDate, timeZoneCookieName } from '$lib/greeting';
import { favorite, unfavorite } from '$lib/server/favoriteActions';
import type { SOPTrainingCoverage } from '$lib/sdk/types';
import type { Actions, PageServerLoad } from './$types';

/** Each card loads on its own: a failed request leaves its card null, not the page broken. */
async function settle<T>(label: string, request: Promise<T>): Promise<T | null> {
	try {
		return await request;
	} catch (err) {
		if (isRedirect(err)) throw err;
		console.error(`Dashboard ${label} load error:`, err);
		return null;
	}
}

export const load: PageServerLoad = async ({ locals, cookies }) => {
	const greetingPeriod = greetingPeriodForDate(new Date(), cookies.get(timeZoneCookieName));
	const user = locals.user;
	if (!user) {
		return {
			user: null,
			greetingPeriod,
			signatureStatus: [],
			favorites: { sops: [], total: 0 },
			recentPublishes: [],
			trainingCoverage: []
		};
	}

	const canSeeTraining = user.role === 'admin' || user.role === 'approver';

	const [signatureStatus, favorites, recentPublishes, trainingCoverage] = await Promise.all([
		settle('signature status', locals.api.auth.getSignatureStatus()),
		settle('favorites', locals.api.sops.list({ favorites_only: true, limit: 24, offset: 0 })),
		settle('recent publishes', locals.api.sops.listRecentPublishes(8)),
		canSeeTraining
			? settle('training coverage', locals.api.sops.trainingCoverage())
			: Promise.resolve([] as SOPTrainingCoverage[])
	]);

	return {
		user,
		greetingPeriod,
		signatureStatus,
		favorites,
		recentPublishes,
		trainingCoverage
	};
};

export const actions: Actions = {
	favorite,
	unfavorite
};
