import { createSDK } from '$lib/sdk';
import { paraglideMiddleware } from '$lib/paraglide/server';
import { cookieName, getTextDirection, isLocale } from '$lib/paraglide/runtime';
import { guestThemeFromCookie, themeAttribute, themeCookieName } from '$lib/theme';
import { sequence } from '@sveltejs/kit/hooks';
import { redirect, type Handle } from '@sveltejs/kit';

const handleAuth: Handle = async ({ event, resolve }) => {
    const sdk = createSDK(event);
    event.locals.api = sdk;

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

    const user = event.locals.user;
    if (user?.locale && isLocale(user.locale)) {
        event.cookies.set(cookieName, user.locale, {
            path: '/',
            httpOnly: false,
            sameSite: 'lax',
            secure: event.url.protocol === 'https:',
            maxAge: 60 * 60 * 24 * 400
        });
        event.request = requestWithLocaleCookie(event.request, cookieName, user.locale);
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

    if (user?.must_change_password) {
        const allowedWhileForced = pathname === '/profile/settings' || pathname === '/logout';
        if (!allowedWhileForced) {
            throw redirect(303, '/profile/settings');
        }
    }

    if (routeId.includes('(guest)') && user) {
        throw redirect(303, user.must_change_password ? '/profile/settings' : '/dashboard');
    }

    return resolve(event);
};

const handleParaglide: Handle = ({ event, resolve }) =>
    paraglideMiddleware(event.request, ({ request, locale }) => {
        event.request = request;
        return resolve(event, {
            transformPageChunk: ({ html }) =>
                html
                    .replace('%paraglide.lang%', locale)
                    .replace('%paraglide.dir%', getTextDirection(locale))
                    .replace(
                        '%theme%',
                        themeAttribute(
                            event.locals.user?.theme ??
                                guestThemeFromCookie(event.cookies.get(themeCookieName))
                        )
                    )
        });
    });

export const handle = sequence(handleAuth, handleParaglide);

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
