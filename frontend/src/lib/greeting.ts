export type GreetingPeriod = 'morning' | 'afternoon' | 'evening';

/**
 * Non-httpOnly cookie with the browser's IANA time zone (e.g. `Europe/Vienna`),
 * so the dashboard greeting rendered on the server matches the reader's clock.
 */
export const timeZoneCookieName = 'tz';

/** Match the locale and theme cookie lifetime (400 days). */
export const timeZoneCookieMaxAge = 60 * 60 * 24 * 400;

export function greetingPeriodForHour(hour: number): GreetingPeriod {
	if (hour < 12) return 'morning';
	if (hour < 17) return 'afternoon';
	return 'evening';
}

/** True for a time zone this runtime's Intl knows. */
export function isTimeZone(value: string | undefined): value is string {
	if (!value) return false;
	try {
		new Intl.DateTimeFormat('en-US', { timeZone: value });
		return true;
	} catch {
		return false;
	}
}

/**
 * Greeting for the hour of `d` in `timeZone`. Without a known time zone it uses
 * the runtime's own clock: right in the browser, only a guess on the server.
 */
export function greetingPeriodForDate(d: Date, timeZone?: string): GreetingPeriod {
	if (!isTimeZone(timeZone)) return greetingPeriodForHour(d.getHours());
	const hour = new Intl.DateTimeFormat('en-US', { timeZone, hour: 'numeric', hourCycle: 'h23' })
		.formatToParts(d)
		.find((part) => part.type === 'hour')?.value;
	return greetingPeriodForHour(Number(hour));
}

/** `document.cookie` assignment that remembers the browser's time zone. */
export function timeZoneCookieAssignment(timeZone: string, secure: boolean): string {
	const secureAttr = secure ? '; Secure' : '';
	return `${timeZoneCookieName}=${encodeURIComponent(timeZone)}; Path=/; Max-Age=${timeZoneCookieMaxAge}; SameSite=Lax${secureAttr}`;
}
