import { describe, expect, it } from 'vitest';
import { partitionSignatureStatus, pendingSignatureCount } from './signatureBuckets';
import type { UserSignatureStatus } from '#lib/sdk/types.js';

const base = (over: Partial<UserSignatureStatus>): UserSignatureStatus => ({
	sop_id: 's1',
	title: 'T',
	latest_version_id: 'v1',
	latest_version: 1,
	has_signed_latest: false,
	signed_older_version: false,
	...over
});

describe('partitionSignatureStatus', () => {
	it('splits new, out-of-date, and up to date', () => {
		const status = [
			base({ sop_id: 'a', has_signed_latest: true }),
			base({ sop_id: 'b', signed_older_version: true }),
			base({ sop_id: 'c' })
		];
		const { actionRequired, notStarted, upToDate } = partitionSignatureStatus(status);
		expect(upToDate.map((s) => s.sop_id)).toEqual(['a']);
		expect(actionRequired.map((s) => s.sop_id)).toEqual(['b']);
		expect(notStarted.map((s) => s.sop_id)).toEqual(['c']);
	});
});

describe('pendingSignatureCount', () => {
	it('counts items that are not up to date', () => {
		expect(
			pendingSignatureCount([
				base({ has_signed_latest: true }),
				base({ signed_older_version: true }),
				base({})
			])
		).toBe(2);
	});
});
