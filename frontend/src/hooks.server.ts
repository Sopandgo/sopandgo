import { createSDK } from '$lib/sdk';
import { redirect, type Handle } from '@sveltejs/kit';

export const handle: Handle = async ({ event, resolve }) => {
    // 1. Initialize SDK
    // Passing 'event' allows the SDK to read/write cookies directly
    const sdk = createSDK(event);
    event.locals.api = sdk;

    let accessToken = event.cookies.get('access_token');
    const refreshToken = event.cookies.get('refresh_token');
    
    // 2. SILENT REFRESH LOGIC
    // We only try to refresh if we have a refresh token but NO access token.
    if (!accessToken && refreshToken) {
        try {
            // SDK calls backend AND updates cookies automatically via setSession()
            const result = await sdk.auth.refresh(refreshToken);
            
            // Update local variable so Step 3 can use the new token immediately
            accessToken = result.access_token;
        } catch (err) {
            const e = err as Error;
            console.error('Auth Refresh Error:', e.message);
            // If refresh fails, kill the refresh token to prevent infinite loops
            event.cookies.delete('refresh_token', { path: '/' });
        }
    }

    // 3. IDENTITY POPULATION
    if (accessToken) {
        try {
            // Fetch user profile using the valid access token
            const user = await sdk.auth.getMe();
            event.locals.user = user;
        } catch (err) {
            // Token likely invalid/expired despite refresh attempt
            // Clear it so the UI updates to "Logged Out" state
            event.cookies.delete('access_token', { path: '/' });
            event.locals.user = null;
        }
    } else {
        event.locals.user = null;
    }

    // 4. SECURE GUARDS
    const routeId = event.route.id ?? '';
    const pathname = event.url.pathname;
    const user = event.locals.user;

    // A. Admin Guard (Strict)
    if (routeId.startsWith('/admin')) {
        if (!user) throw redirect(303, '/login');
        if (user.role !== 'admin') throw redirect(303, '/dashboard');
    }

    // B. App Guard (Logged-in only)
    // Matches any route inside the (app) folder
    if (routeId.includes('(app)') && !user) {
        throw redirect(303, '/login');
    }

    // C. Forced password change (bootstrap / flagged accounts)
    if (user?.must_change_password) {
        const allowedWhileForced =
            pathname === '/profile' ||
            pathname === '/logout';
        if (!allowedWhileForced) {
            throw redirect(303, '/profile');
        }
    }

    // D. Guest Guard (Logged-out only)
    // Matches routes inside (guest), like /login and /register.
    if (routeId.includes('(guest)') && user) {
        throw redirect(303, user.must_change_password ? '/profile' : '/dashboard');
    }

    return await resolve(event);
};
