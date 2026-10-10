import { describe, expect, it } from 'vitest';
import {
	greetingPeriodForDate,
	greetingPeriodForHour,
	isTimeZone,
	timeZoneCookieAssignment
} from './greeting';

describe('greetingPeriodForHour', () => {
	it('returns morning before noon', () => {
		expect(greetingPeriodForHour(0)).toBe('morning');
		expect(greetingPeriodForHour(11)).toBe('morning');
	});
	it('returns afternoon before 17:00', () => {
		expect(greetingPeriodForHour(12)).toBe('afternoon');
		expect(greetingPeriodForHour(16)).toBe('afternoon');
	});
	it('returns evening from 17:00', () => {
		expect(greetingPeriodForHour(17)).toBe('evening');
		expect(greetingPeriodForHour(23)).toBe('evening');
	});
});

describe('greetingPeriodForDate', () => {
	const instant = new Date('2020-01-01T10:00:00Z');

	it('uses the hour in the given time zone', () => {
		expect(greetingPeriodForDate(instant, 'UTC')).toBe('morning');
		expect(greetingPeriodForDate(instant, 'Asia/Tokyo')).toBe('evening');
		expect(greetingPeriodForDate(instant, 'America/New_York')).toBe('morning');
		expect(greetingPeriodForDate(new Date('2020-01-01T13:30:00Z'), 'Europe/Vienna')).toBe('afternoon');
	});

	it('treats midnight as morning', () => {
		expect(greetingPeriodForDate(new Date('2020-01-01T00:00:00Z'), 'UTC')).toBe('morning');
	});

	it('falls back to local hours without a known time zone', () => {
		const local = new Date('2020-01-01T16:00:00');
		expect(greetingPeriodForDate(local)).toBe('afternoon');
		expect(greetingPeriodForDate(local, 'Not/AZone')).toBe('afternoon');
	});
});

describe('isTimeZone', () => {
	it('accepts IANA names and rejects anything else', () => {
		expect(isTimeZone('Europe/Vienna')).toBe(true);
		expect(isTimeZone('UTC')).toBe(true);
		expect(isTimeZone('Not/AZone')).toBe(false);
		expect(isTimeZone('')).toBe(false);
		expect(isTimeZone(undefined)).toBe(false);
	});
});

describe('timeZoneCookieAssignment', () => {
	it('encodes the zone and adds Secure on https', () => {
		expect(timeZoneCookieAssignment('Europe/Vienna', true)).toBe(
			'tz=Europe%2FVienna; Path=/; Max-Age=34560000; SameSite=Lax; Secure'
		);
		expect(timeZoneCookieAssignment('UTC', false)).not.toContain('Secure');
	});
});
