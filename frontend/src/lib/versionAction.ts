/*
 * What the signed-in user can do with one SOP version, matching what the API
 * allows. Every role with the reader signature (admin, approver, editor and
 * viewer) signs published versions; training coverage expects each of them to.
 * Only approvers and admins review release candidates. The auditor reads and
 * never signs. An archived SOP takes no new signatures or reviews, but a
 * signature already given still shows.
 */

export type VersionAction = 'review' | 'signed' | 'sign' | 'none';

export interface VersionActionInput {
	/** The signed-in user's role, or undefined without a user. */
	role: string | undefined;
	/** The version's status: draft, rc, published, … */
	status: string | null;
	archived: boolean;
	/** Whether the user already gave this version a reader signature. */
	signed: boolean;
}

const READER_ROLES = new Set(['admin', 'approver', 'editor', 'viewer']);
const REVIEWER_ROLES = new Set(['admin', 'approver']);

export function versionAction({ role, status, archived, signed }: VersionActionInput): VersionAction {
	if (!role) return 'none';
	if (status === 'rc') {
		return !archived && REVIEWER_ROLES.has(role) ? 'review' : 'none';
	}
	if (status !== 'published' || !READER_ROLES.has(role)) return 'none';
	if (signed) return 'signed';
	return archived ? 'none' : 'sign';
}
