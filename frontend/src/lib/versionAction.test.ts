import { describe, expect, it } from 'vitest';
import { versionAction } from './versionAction';

const published = { status: 'published', archived: false, signed: false };

describe('versionAction', () => {
	it('lets a viewer sign a published version as a reader', () => {
		expect(versionAction({ role: 'viewer', ...published })).toBe('sign');
	});

	it('lets every role with the reader signature sign', () => {
		for (const role of ['admin', 'approver', 'editor', 'viewer']) {
			expect(versionAction({ role, ...published })).toBe('sign');
		}
	});

	it('never offers the auditor a signature', () => {
		expect(versionAction({ role: 'auditor', ...published })).toBe('none');
		expect(versionAction({ role: 'auditor', ...published, status: 'rc' })).toBe('none');
	});

	it('shows a signature already given, also on an archived SOP', () => {
		expect(versionAction({ role: 'viewer', ...published, signed: true })).toBe('signed');
		expect(versionAction({ role: 'viewer', ...published, signed: true, archived: true })).toBe('signed');
	});

	it('takes no new signature on an archived SOP', () => {
		expect(versionAction({ role: 'viewer', ...published, archived: true })).toBe('none');
	});

	it('offers review of a release candidate to approvers and admins only', () => {
		const rc = { ...published, status: 'rc' };
		expect(versionAction({ role: 'approver', ...rc })).toBe('review');
		expect(versionAction({ role: 'admin', ...rc })).toBe('review');
		expect(versionAction({ role: 'editor', ...rc })).toBe('none');
		expect(versionAction({ role: 'viewer', ...rc })).toBe('none');
		expect(versionAction({ role: 'approver', ...rc, archived: true })).toBe('none');
	});

	it('offers nothing on a draft or without a user', () => {
		expect(versionAction({ role: 'admin', ...published, status: 'draft' })).toBe('none');
		expect(versionAction({ role: undefined, ...published })).toBe('none');
	});
});
