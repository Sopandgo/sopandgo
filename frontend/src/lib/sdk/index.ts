import type { RequestEvent } from '@sveltejs/kit';
import { createClient } from './client';
import { auth } from './auth';
import { sops } from './sops';
import { assets } from './assets';
import { admin } from './admin';
import { tags } from './tags';

export const createSDK = (event: RequestEvent) => {
    const client = createClient(event);

    return {
        // 1. Expose the raw fetch/get/post helpers (fixes "locals.api.get is not a function")
        ...client, 

        // 2. Expose your new Domain Modules
        auth: auth(client, event), // Pass event for cookie handling
        sops: sops(client),
        assets: assets(client),
        admin: admin(client),
        tags: tags(client),
    };
};