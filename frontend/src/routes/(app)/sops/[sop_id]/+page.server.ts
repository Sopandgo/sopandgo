import { error, fail } from '@sveltejs/kit';
import * as m from '$lib/paraglide/messages.js';
import { verifyAsset } from '$lib/server/integrityActions';
import type { Actions, PageServerLoad } from './$types';

export const load: PageServerLoad = async ({ locals, parent }) => {
  const { sop } = await parent();

  const canSeeTraining = locals.user?.role === 'admin' || locals.user?.role === 'approver';

  try {
    const [versions, assets, allTags, trainingCoverage] = await Promise.all([
      locals.api.sops.listVersions(sop.id),
      locals.api.assets.list(sop.id),
      locals.api.tags.list(),
      canSeeTraining ? locals.api.sops.trainingCoverage(sop.id) : Promise.resolve([])
    ]);

    return { sop, versions, assets, allTags, trainingCoverage };
  } catch (err) {
    console.error('SOP Page Load Error:', err);
    throw error(500, 'Could not load SOP page data');
  }
};

export const actions: Actions = {
  verifyAsset,

  attach_tag: async ({ locals, request, params }) => {
    const form = await request.formData();
    const tagId = String(form.get('tag_id') ?? '').trim();
    const sopId = params.sop_id;

    if (!sopId) return fail(400, { message: m.error_missing_sop() });
    if (!tagId) return fail(400, { message: m.error_missing_tag() });

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

  create_and_attach_tag: async ({ locals, request, params }) => {
    const form = await request.formData();
    const title = String(form.get('title') ?? '').trim();
    const sopId = params.sop_id;

    if (!sopId) return fail(400, { message: m.error_missing_sop() });
    if (!title) return fail(400, { message: m.error_missing_title() });

    try {
      const { id } = await locals.api.tags.create(title);
      await locals.api.sops.attachTag(sopId, id);
      return { success: true };
    } catch (err: any) {
      console.error('CREATE_AND_ATTACH_TAG Error:', err);

      if (err?.message === 'TAG_ALREADY_EXISTS') {
        return fail(409, { message: m.error_tag_exists() });
      }
      return fail(500, { message: m.error_create_tag() });
    }
  },

  favorite: async ({ locals, params }) => {
    const sopId = params.sop_id;
    if (!sopId) return fail(400, { message: m.error_missing_sop() });
    try {
      await locals.api.sops.favorite(sopId);
      return { success: true };
    } catch (err) {
      console.error('favorite', err);
      return fail(500, { message: m.error_favorite_failed() });
    }
  },

  unfavorite: async ({ locals, params }) => {
    const sopId = params.sop_id;
    if (!sopId) return fail(400, { message: m.error_missing_sop() });
    try {
      await locals.api.sops.unfavorite(sopId);
      return { success: true };
    } catch (err) {
      console.error('unfavorite', err);
      return fail(500, { message: m.error_unfavorite_failed() });
    }
  }
};