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

export type GreetingPeriod = 'morning' | 'afternoon' | 'evening';

/** Uses local hours of the given instant (typically server time in `load`). */
export function greetingPeriodForDate(d: Date): GreetingPeriod {
	const h = d.getHours();
	if (h < 12) return 'morning';
	if (h < 17) return 'afternoon';
	return 'evening';
}
