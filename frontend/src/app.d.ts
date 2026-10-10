import type { createSDK } from '#lib/sdk/index.js';
import type { User } from '#lib/sdk/types.js';

declare global {
    namespace App {
        // interface Error {}
        
        interface Locals {
            user: User | null;
            // FIX: Use 'createSDK' here to match your SDK export
            api: ReturnType<typeof createSDK>; 
        }

        interface PageData {
            // This ensures every page knows 'user' exists in its 'data' prop
            user: User | null;          
        }
        
        // interface PageState {}
        // interface Platform {}
    }
    
    // Keep this if you are using Vite 'define' to inject versions
    declare const __APP_VERSION__: string;
}

export {};
