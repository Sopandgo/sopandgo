import { getLocale } from '$lib/paraglide/runtime.js';
import * as m from '$lib/paraglide/messages.js';

export interface Crumb {
	label: string;
	/** Omitted for the current page and for sections without a page of their own. */
	href?: string;
}

type CrumbData = {
	sop?: { id: string; title: string };
	versionSummary?: { version: number; created_at: string; status?: string };
	[key: string]: unknown;
};

/**
 * Trail shown in the navbar for a route. The sidebar already marks the
 * section, so trails start at the section, not at the dashboard.
 */
export function breadcrumbsFor(routeId: string | null, data: CrumbData): Crumb[] {
	const sops: Crumb = { label: m.page_sops(), href: '/sops' };
	const sop = (): Crumb => ({ label: data.sop?.title ?? m.page_sop(), href: `/sops/${data.sop?.id}` });
	const settings: Crumb = { label: m.page_settings(), href: '/admin/settings' };

	switch (routeId) {
		case '/(app)/dashboard':
			return [{ label: m.page_dashboard() }];
		case '/(app)/sops':
			return [{ label: m.page_sops() }];
		case '/(app)/sops/new':
			return [sops, { label: m.page_new_sop() }];
		case '/(app)/sops/[sop_id]':
			return [sops, { label: sop().label }];
		case '/(app)/sops/[sop_id]/new':
			return [sops, sop(), { label: m.page_new_version() }];
		case '/(app)/sops/[sop_id]/v/[version_id]': {
			const v = data.versionSummary;
			// Only one version is published at a time; it is the active one.
			const versionLabel = v?.status === 'published' ? m.version_breadcrumb_latest : m.version_breadcrumb;
			const label = v
				? versionLabel({
						version: String(v.version),
						date: new Date(v.created_at).toLocaleDateString(getLocale())
					})
				: m.page_sop_version();
			return [sops, sop(), { label }];
		}
		case '/(app)/profile':
			return [{ label: m.nav_profile() }];
		case '/(app)/profile/settings':
			return [{ label: m.nav_profile(), href: '/profile' }, { label: m.page_account_settings() }];
		case '/admin/users':
			return [{ label: m.nav_users() }];
		case '/admin/sessions':
			return [{ label: m.nav_sessions() }];
		case '/admin/audit-logs':
			return [{ label: m.nav_audit_logs() }];
		case '/admin/integrity':
			return [{ label: m.nav_integrity() }];
		case '/admin/settings':
			return [{ label: m.page_settings() }];
		case '/admin/settings/email':
			return [settings, { label: m.settings_email() }];
		case '/admin/settings/integrations':
			return [settings, { label: m.settings_integrations() }];
		case '/admin/settings/backup':
			return [settings, { label: m.nav_backup() }];
		default:
			return [];
	}
}
