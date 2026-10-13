import { describe, expect, it } from 'vitest';
import { REPO_URL, versionLabel, versionUrl } from './product';

describe('version links', () => {
	it('labels versions with a single v', () => {
		expect(versionLabel('1.1.0')).toBe('v1.1.0');
		expect(versionLabel('v1.1.0')).toBe('v1.1.0');
		expect(versionLabel('dev')).toBe('dev');
		expect(versionLabel('')).toBe('dev');
	});

	it('links tags to their GitHub release', () => {
		expect(versionUrl('1.1.0')).toBe(`${REPO_URL}/releases/tag/v1.1.0`);
		expect(versionUrl('v1.2.0-rc.1')).toBe(`${REPO_URL}/releases/tag/v1.2.0-rc.1`);
	});

	it('links builds between tags to their commit, and dev builds to the repository', () => {
		expect(versionUrl('1.1.0-3-gabc1234')).toBe(`${REPO_URL}/commit/abc1234`);
		expect(versionUrl('dev')).toBe(REPO_URL);
		expect(versionUrl('abc1234')).toBe(REPO_URL);
	});
});
