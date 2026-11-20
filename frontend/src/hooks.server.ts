import { redirect } from '@sveltejs/kit';
import { DEMO_MODE, UMAMI_DOMAINS, UMAMI_SCRIPT_URL, UMAMI_WEBSITE_ID } from '$app/env/private';
import { umamiConfig, umamiScriptTag } from '#lib/analytics.js';
import { demoSessionToOpen, isDemoMode } from '#lib/demo.js';
import { shouldReport } from '#lib/errorReporting.js';
import { browserReportingHead, reportServerError } from '#lib/server/errorReporting.js';
import { createSDK } from '#lib/sdk/index.js';
import { paraglideMiddleware } from '#lib/paraglide/server.js';
import { cookieName, getTextDirection, isLocale } from '#lib/paraglide/runtime.js';
import { guestThemeFromCookie, themeAttribute, themeCookieName } from '#lib/theme.js';
import { sequence, type Handle, type HandleServerError, type ServerInit } from '@sveltejs/kit/hooks';

// The Umami tag, decided once at startup: environment values do not change
// while the process runs. Empty unless this is the public demo.
let analyticsHead = '';

export const init: ServerInit = () => {
    const umami = umamiConfig(isDemoMode(DEMO_MODE), {
        scriptUrl: UMAMI_SCRIPT_URL,
        websiteId: UMAMI_WEBSITE_ID,
        domains: UMAMI_DOMAINS
    });
    if (umami && 'skip' in umami) console.warn(`Umami analytics is off: ${umami.skip}`);
    analyticsHead = umami && 'config' in umami ? umamiScriptTag(umami.config) : '';
};

const handleAuth: Handle = async ({ event, resolve }) => {
    const sdk = createSDK(event);
    event.locals.api = sdk;
    const demo = isDemoMode(DEMO_MODE);
    event.locals.demo = demo;
    // Browser error reporting, when an admin turned it on (cached for a minute).
    event.locals.extraHead = analyticsHead + (await browserReportingHead());

    let accessToken = event.cookies.get('access_token');
    const refreshToken = event.cookies.get('refresh_token');

    if (!accessToken && refreshToken) {
        try {
            const result = await sdk.auth.refresh(refreshToken);
            accessToken = result.access_token;
        } catch (err) {
            const e = err as Error;
            console.error('Auth Refresh Error:', e.message);
            event.cookies.delete('refresh_token', { path: '/' });
        }
    }

    if (accessToken) {
        try {
            const user = await sdk.auth.getMe();
            event.locals.user = user;
        } catch {
            event.cookies.delete('access_token', { path: '/' });
            event.locals.user = null;
        }
    } else {
        event.locals.user = null;
    }

    // A demo visitor without a session is signed in as the researcher. A
    // returning visitor keeps the session, and so the role, they had.
    const demoRole = demoSessionToOpen(demo, !!event.locals.user);
    if (demoRole) {
        try {
            await sdk.auth.openDemoSession(demoRole);
            event.locals.user = await sdk.auth.getMe();
        } catch (err) {
            console.error('Demo session failed:', (err as Error).message);
        }
    }

    const user = event.locals.user;
    // In the demo, locale and theme stay in cookies, as for guests. The account
    // shows what the cookies say, so the settings controls work without a write.
    if (demo && user) {
        user.theme = guestThemeFromCookie(event.cookies.get(themeCookieName));
        const cookieLocale = event.cookies.get(cookieName);
        if (cookieLocale && isLocale(cookieLocale)) user.locale = cookieLocale;
    }
    // Kit 3: event.request is readonly — pass a replaced request into resolve().
    let request = event.request;
    if (!demo && user?.locale && isLocale(user.locale)) {
        event.cookies.set(cookieName, user.locale, {
            path: '/',
            httpOnly: false,
            sameSite: 'lax',
            secure: event.url.protocol === 'https:',
            maxAge: 60 * 60 * 24 * 400
        });
        request = requestWithLocaleCookie(event.request, cookieName, user.locale);
    }

    const routeId = event.route.id ?? '';
    const pathname = event.url.pathname;

    if (routeId.startsWith('/admin')) {
        if (!user) throw redirect(303, '/login');
        if (user.role !== 'admin') throw redirect(303, '/dashboard');
    }

    if (routeId.includes('(app)') && !user) {
        throw redirect(303, '/login');
    }

    // The audit log needs audit:read in the API, which admins and auditors hold.
    if (routeId.startsWith('/(app)/audit-logs') && user && !canReadAuditLog(user.role)) {
        throw redirect(303, '/dashboard');
    }

    // A forced password change comes first; then, while the organization
    // requires 2FA, setting up an authenticator. Both happen in account settings.
    const mustVisitSettings = !!user && (user.must_change_password || user.must_enroll_two_factor);
    if (mustVisitSettings) {
        // The logo shows on that page too (footer, favicon), so it stays reachable.
        const allowedWhileForced =
            pathname === '/profile/settings' || pathname === '/logout' || pathname === '/branding/logo';
        if (!allowedWhileForced) {
            throw redirect(303, '/profile/settings');
        }
    }

    if (routeId.includes('(guest)') && user) {
        throw redirect(303, mustVisitSettings ? '/profile/settings' : demo ? '/sops' : '/dashboard');
    }

    return resolve({ ...event, request });
};

const handleParaglide: Handle = ({ event, resolve }) =>
    paraglideMiddleware(event.request, ({ request, locale }) =>
        resolve(
            { ...event, request },
            {
                transformPageChunk: ({ html }) =>
                    html
                        .replace('%paraglide.lang%', locale)
                        .replace('%paraglide.dir%', getTextDirection(locale))
                        .replace('%sopandgo.head%', event.locals.extraHead ?? '')
                        .replace(
                            '%theme%',
                            themeAttribute(
                                event.locals.user?.theme ??
                                    guestThemeFromCookie(event.cookies.get(themeCookieName))
                            )
                        )
            }
        )
    );

export const handle = sequence(handleAuth, handleParaglide);

// Unexpected errors go to Sentry or GlitchTip when an admin turned that on.
// error() responses and 404s are not reported. Returning nothing keeps
// SvelteKit's default message.
export const handleError: HandleServerError = async ({ error, kind, event }) => {
    if (!shouldReport(kind)) return;
    console.error(error);
    await reportServerError(error, {
        route: event.route.id ?? '(unmatched)',
        method: event.request.method
    });
};

function canReadAuditLog(role: string): boolean {
    return role === 'admin' || role === 'auditor';
}

function requestWithLocaleCookie(request: Request, name: string, locale: string): Request {
    const headers = new Headers(request.headers);
    const parts = (headers.get('cookie') ?? '')
        .split(';')
        .map((part) => part.trim())
        .filter((part) => part.length > 0 && !part.startsWith(`${name}=`));
    parts.push(`${name}=${locale}`);
    headers.set('cookie', parts.join('; '));
    return new Request(request, { headers });
}
