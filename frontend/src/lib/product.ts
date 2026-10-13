/** Where the product name links (style guide → Product name). */
export const PRODUCT_URL = 'https://sopandgo.com';

/** Source code; every version label links here. */
export const REPO_URL = 'https://github.com/Sopandgo/sopandgo';

const RELEASE = /^\d+\.\d+\.\d+(?:-[0-9A-Za-z.]+)?$/;
// `git describe` in a checkout: 1.1.0-3-gabc1234 (commits after the tag).
const DESCRIBE = /-\d+-g([0-9a-f]{7,40})$/;

/** "v1.2.3", or "dev" for an untagged build. */
export function versionLabel(raw: string): string {
	const v = String(raw || 'dev').replace(/^v/, '');
	return v === 'dev' ? 'dev' : `v${v}`;
}

/**
 * GitHub page for a version: the release for a tag, the commit for a build
 * between tags, the repository for a dev build.
 */
export function versionUrl(raw: string): string {
	const v = String(raw || 'dev').replace(/^v/, '');
	const describe = v.match(DESCRIBE);
	if (describe) return `${REPO_URL}/commit/${describe[1]}`;
	if (RELEASE.test(v)) return `${REPO_URL}/releases/tag/v${v}`;
	return REPO_URL;
}
