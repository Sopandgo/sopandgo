import type { UserSignatureStatus } from '$lib/sdk/types';

export function partitionSignatureStatus(status: UserSignatureStatus[]) {
	const actionRequired: UserSignatureStatus[] = [];
	const notStarted: UserSignatureStatus[] = [];
	const upToDate: UserSignatureStatus[] = [];

	for (const s of status) {
		if (s.has_signed_latest) upToDate.push(s);
		else if (s.signed_older_version) actionRequired.push(s);
		else notStarted.push(s);
	}

	return { actionRequired, notStarted, upToDate };
}

export function pendingSignatureCount(status: UserSignatureStatus[]) {
	const { actionRequired, notStarted } = partitionSignatureStatus(status);
	return actionRequired.length + notStarted.length;
}
