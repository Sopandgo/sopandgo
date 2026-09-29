import { BACKEND_URL } from '$env/static/private';
import type { RequestEvent } from '@sveltejs/kit';

export interface FetchOptions {
    headers?: Record<string, string> | Headers;
    body?: any; // We allow 'any' because we automatically stringify objects
    method?: string;
    // Add other standard fetch options if needed (signal, mode, etc.)
}

export type Client = ReturnType<typeof createClient>;

/**
 * The internal fetcher that handles headers, FormData, and Auth.
 */
export function createClient(event: RequestEvent) {
    
    const baseFetch = async (path: string, options: FetchOptions = {}, authenticated = true) => {
        // Ensure no double slashes if BACKEND_URL ends with /
        const cleanBase = BACKEND_URL.replace(/\/$/, '');
        const cleanPath = path.startsWith('/') ? path : `/${path}`;
        const url = `${cleanBase}/api${cleanPath}`;
        
        const headers = new Headers(options.headers);
        const isFormData = options.body instanceof FormData;

        // 1. Content-Type Management
        if (!isFormData && !headers.has('Content-Type')) {
            headers.set('Content-Type', 'application/json');
        }
        if (isFormData) {
            headers.delete('Content-Type'); // Let browser set boundary
        }

        // 2. Auth Injection
        if (authenticated) {
            const token = event.cookies.get('access_token');
            if (token) {
                headers.set('Authorization', `Bearer ${token}`);
            }
        }

        // 3. Execution with Error Guard
        try {
            const res = await event.fetch(url, {
                ...options,
                method: options.method || 'GET',
                headers,
                // If it's JSON/Text, stringify it. If FormData, pass raw.
                body: isFormData ? options.body : (options.body && typeof options.body === 'object' ? JSON.stringify(options.body) : options.body)
            });
            return res;
        } catch (err) {
            console.error(`[SDK] Connection failed to ${url}`, err);
            // We re-throw so the specific module (auth/assets) can decide how to handle it
            throw new Error('SERVICE_UNAVAILABLE');
        }
    };

    return {
        fetch: baseFetch,

        // --- Convenience Methods (Restored for compatibility) ---
        
        get: (path: string, opt?: FetchOptions) => 
            baseFetch(path, { ...opt, method: 'GET' }),

        post: (path: string, body: any, opt?: FetchOptions) => 
            baseFetch(path, { ...opt, method: 'POST', body }),

        put: (path: string, body: any, opt?: FetchOptions) => 
            baseFetch(path, { ...opt, method: 'PUT', body }),

        del: (path: string, opt?: FetchOptions) => 
            baseFetch(path, { ...opt, method: 'DELETE' }),
            
        // Public/Unauthenticated Helper
        public: (path: string, opt?: FetchOptions) => 
            baseFetch(path, opt, false)
    };
}