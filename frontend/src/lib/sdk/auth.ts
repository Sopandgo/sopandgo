import type { RequestEvent } from '@sveltejs/kit';
import type { CookieSerializeOptions } from 'cookie';
import type { Client } from './client';
import type { User, UserSignatureStatus } from './types';

interface AuthResult {
    access_token: string;
    refresh_token?: string;
}

export const auth = (client: Client, event: RequestEvent) => {

    // 1. Helper to ensure consistent cookie options
    // This is crucial: 'delete' must use the exact same options as 'set'
    const getCookieOptions = (): CookieSerializeOptions & { path: string } => {
        const isSecure = event.url.protocol === 'https:';
        return {
            path: '/',
            httpOnly: true,
            secure: isSecure,
            sameSite: 'lax'
        };
    };

    const setSession = (result: AuthResult) => {
        const defaults = getCookieOptions();

        // Access Token: 5 minutes
        event.cookies.set('access_token', result.access_token, { 
            ...defaults, 
            maxAge: 60 * 5 
        });

        // Refresh Token: 7 days
        if (result.refresh_token) {
            event.cookies.set('refresh_token', result.refresh_token, {
                ...defaults,
                maxAge: 60 * 60 * 24 * 7
            });
        }
    };

    return {
        login: async (email: string, password: string): Promise<AuthResult> => {
            const res = await client.fetch('/auth/login', {
                method: 'POST',
                body: { email, password }
            }, false);

            if (!res.ok) {
                if (res.status === 401) throw new Error('INVALID_CREDENTIALS');
                if (res.status === 403) throw new Error('ACCOUNT_INACTIVE');
                throw new Error('LOGIN_FAILED');
            }

            const result: AuthResult = await res.json();
            setSession(result); 
            return result;
        },

        refresh: async (refreshToken: string): Promise<AuthResult> => {
            const res = await client.fetch('/auth/refresh', {
                method: 'POST',
                body: { refresh_token: refreshToken }
            }, false);

            if (!res.ok) throw new Error('REFRESH_FAILED');
            
            const result: AuthResult = await res.json();
            setSession(result);
            return result;
        },

        getMe: async (): Promise<User> => {
            const res = await client.fetch('/auth/me', {
                method: 'GET'
            });
            if (!res.ok) throw new Error('USER_FETCH_FAILED');
            return await res.json();
        },

        getSignatureStatus: async (): Promise<UserSignatureStatus[]> => {
            const res = await client.fetch('/auth/me/signature-status', {
                method: 'GET'
            });
            if (!res.ok) throw new Error('FETCH_SIGNATURE_STATUS_FAILED');
            return await res.json();
        },

        /**
         * Updates a user's password.
         * Match: POST /api/auth/update-password
         * Note: Admins can use this to reset other users' passwords by passing a different userId. REMOVE THIS ADMIN ENDPOINT FROM PRODUCTION!!!
         */
        updateLocale: async (locale: string): Promise<void> => {
            const res = await client.fetch('/auth/me/locale', {
                method: 'PATCH',
                body: { locale }
            });
            if (res.status === 400) throw new Error('UNSUPPORTED_LOCALE');
            if (!res.ok) throw new Error('UPDATE_LOCALE_FAILED');
        },

        updateTheme: async (theme: string): Promise<void> => {
            const res = await client.fetch('/auth/me/theme', {
                method: 'PATCH',
                body: { theme }
            });
            if (res.status === 400) throw new Error('UNSUPPORTED_THEME');
            if (!res.ok) throw new Error('UPDATE_THEME_FAILED');
        },

        updatePassword: async (newPassword: string): Promise<void> => {
            const res = await client.fetch('/auth/me/update-password', {
                method: 'PATCH',
                body: { new_password: newPassword }
            });

            if (!res.ok) {
                // Backend returns 400 if password validation fails (e.g. too short)
                if (res.status === 400) throw new Error('WEAK_PASSWORD');
                if (res.status === 403) throw new Error('FORBIDDEN_PASSWORD_CHANGE');
                throw new Error('UPDATE_PASSWORD_FAILED');
            }
        },

        resetPassword: async (token: string, newPassword: string): Promise<void> => {
            const res = await client.fetch('/auth/reset-password', {
                method: 'PATCH',
                body: { token, new_password: newPassword }
            });

            if (!res.ok) {
                const errorMsg = await res.text().catch(() => '');

                if (res.status === 401) {
                    throw new Error('TOKEN_INVALID_OR_EXPIRED'); 
                }

                if (res.status === 400) {
                    throw new Error(errorMsg || 'WEAK_PASSWORD');
                }

                throw new Error('UPDATE_PASSWORD_FAILED');
            }
        },

        logout: async (): Promise<boolean> => {
            const refreshToken = event.cookies.get('refresh_token');

            if (refreshToken) {
                // We use catch() here so that if the backend is down or the token is 
                // already invalid, the frontend still proceeds to clear local cookies.
                await client.fetch('/auth/logout', { 
                    method: 'POST',
                    body: { refresh_token: refreshToken }
                }).catch((err) => console.warn('Logout backend sync failed', err));
            }
            
            // FIX: Retrieve the exact same options used to create the cookies
            const opts = getCookieOptions();
            
            event.cookies.delete('access_token', opts);
            event.cookies.delete('refresh_token', opts);
            
            return true;
        }
    };
};