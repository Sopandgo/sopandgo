import { error, fail } from '@sveltejs/kit';
import * as m from '#lib/paraglide/messages.js';
import { favorite, unfavorite } from '#lib/server/favoriteActions.js';
import { verifyAsset } from '#lib/server/integrityActions.js';
import type { Actions, PageServerLoad } from './$types';

export const load: PageServerLoad = async ({ locals, parent }) => {
  const { sop } = await parent();

  try {
    const [versions, assets, allTags] = await Promise.all([
      locals.api.sops.listVersions(sop.id),
      locals.api.assets.list(sop.id),
      locals.api.tags.list()
    ]);

    return { sop, versions, assets, allTags };
  } catch (err) {
    console.error('SOP Page Load Error:', err);
    throw error(500, 'Could not load SOP page data');
  }
};

export const actions: Actions = {
  verifyAsset,

  // One field for both cases: attach the tag with this title, or create it first
  add_tag: async ({ locals, request, params }) => {
    const form = await request.formData();
    const title = String(form.get('title') ?? '').trim();
    const sopId = params.sop_id;

    if (!sopId) return fail(400, { message: m.error_missing_sop() });
    if (!title) return fail(400, { message: m.error_missing_title() });

    let tagId: string;
    try {
      const existing = (await locals.api.tags.list()).find(
        (t) => t.title.toLowerCase() === title.toLowerCase()
      );
      if (existing && existing.is_active === false) {
        return fail(409, { message: m.error_tag_retired({ title: existing.title }) });
      }
      tagId = existing ? existing.id : (await locals.api.tags.create(title)).id;
    } catch (err) {
      console.error('ADD_TAG Error:', err);
      if ((err as Error)?.message === 'TAG_ALREADY_EXISTS') {
        return fail(409, { message: m.error_tag_exists() });
      }
      return fail(500, { message: m.error_create_tag() });
    }

    try {
      await locals.api.sops.attachTag(sopId, tagId);
      return { success: true };
    } catch (err) {
      console.error('ATTACH_TAG Error:', err);
      return fail(500, { message: m.error_attach_tag() });
    }
  },

  detach_tag: async ({ locals, request, params }) => {
    const form = await request.formData();
    const tagId = String(form.get('tag_id') ?? '').trim();
    const sopId = params.sop_id;

    if (!sopId) return fail(400, { message: m.error_missing_sop() });
    if (!tagId) return fail(400, { message: m.error_missing_tag() });

    try {
      await locals.api.sops.detachTag(sopId, tagId);
      return { success: true };
    } catch (err) {
      console.error('DETACH_TAG Error:', err);
      return fail(500, { message: m.error_detach_tag() });
    }
  },

  favorite,
  unfavorite
};
