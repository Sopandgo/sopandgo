import { describe, expect, it } from 'vitest';
import { breadcrumbsFor } from './breadcrumbs';

const sop = { id: 's1', title: 'Cleaning the centrifuge' };

describe('breadcrumbsFor', () => {
	it('starts SOP trails at the section, not the dashboard', () => {
		expect(breadcrumbsFor('/(app)/sops', {})).toEqual([{ label: 'SOPs' }]);
		expect(breadcrumbsFor('/(app)/sops/[sop_id]', { sop })).toEqual([
			{ label: 'SOPs', href: '/sops' },
			{ label: sop.title }
		]);
	});

	it('links back to the SOP from its child pages', () => {
		const trail = breadcrumbsFor('/(app)/sops/[sop_id]/new', { sop });
		expect(trail.map((c) => c.href)).toEqual(['/sops', '/sops/s1', undefined]);
		expect(trail[2].label).toBe('New version');
	});

	it('names the version with its number', () => {
		const trail = breadcrumbsFor('/(app)/sops/[sop_id]/v/[version_id]', {
			sop,
			versionSummary: { version: 3, created_at: '2026-10-01T12:00:00Z' }
		});
		expect(trail).toHaveLength(3);
		expect(trail[2].label).toMatch(/^Version 3 - /);
	});

	it('marks the published version as latest', () => {
		const versionSummary = { version: 3, created_at: '2026-10-01T12:00:00Z' };
		const route = '/(app)/sops/[sop_id]/v/[version_id]';
		expect(
			breadcrumbsFor(route, { sop, versionSummary: { ...versionSummary, status: 'published' } })[2].label
		).toMatch(/^Version 3 \(latest\) - /);
		expect(
			breadcrumbsFor(route, { sop, versionSummary: { ...versionSummary, status: 'superseded' } })[2].label
		).toMatch(/^Version 3 - /);
	});

	it('nests admin settings tabs under Settings', () => {
		expect(breadcrumbsFor('/admin/settings/email', {})).toEqual([
			{ label: 'Settings', href: '/admin/settings' },
			{ label: 'Email' }
		]);
	});

	it('returns no trail for unknown routes', () => {
		expect(breadcrumbsFor('/(guest)/login', {})).toEqual([]);
		expect(breadcrumbsFor(null, {})).toEqual([]);
	});
});
