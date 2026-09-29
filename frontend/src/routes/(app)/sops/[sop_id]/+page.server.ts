import { error, fail } from '@sveltejs/kit';
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
  attach_tag: async ({ locals, request, params }) => {
    const form = await request.formData();
    const tagId = String(form.get('tag_id') ?? '').trim();
    const sopId = params.sop_id;

    if (!sopId) return fail(400, { message: 'Missing sop_id' });
    if (!tagId) return fail(400, { message: 'Missing tag_id' });

    try {
      await locals.api.sops.attachTag(sopId, tagId);
      return { success: true };
    } catch (err) {
      console.error('ATTACH_TAG Error:', err);
      return fail(500, { message: 'Failed to attach tag' });
    }
  },

  detach_tag: async ({ locals, request, params }) => {
    const form = await request.formData();
    const tagId = String(form.get('tag_id') ?? '').trim();
    const sopId = params.sop_id;

    if (!sopId) return fail(400, { message: 'Missing sop_id' });
    if (!tagId) return fail(400, { message: 'Missing tag_id' });

    try {
      await locals.api.sops.detachTag(sopId, tagId);
      return { success: true };
    } catch (err) {
      console.error('DETACH_TAG Error:', err);
      return fail(500, { message: 'Failed to detach tag' });
    }
  },

  create_and_attach_tag: async ({ locals, request, params }) => {
    const form = await request.formData();
    const title = String(form.get('title') ?? '').trim();
    const sopId = params.sop_id;

    if (!sopId) return fail(400, { message: 'Missing sop_id' });
    if (!title) return fail(400, { message: 'Missing title' });

    try {
      const { id } = await locals.api.tags.create(title);
      await locals.api.sops.attachTag(sopId, id);
      return { success: true };
    } catch (err: any) {
      console.error('CREATE_AND_ATTACH_TAG Error:', err);

      if (err?.message === 'TAG_ALREADY_EXISTS') {
        return fail(409, { message: 'Tag already exists' });
      }
      return fail(500, { message: 'Failed to create/attach tag' });
    }
  },

  favorite: async ({ locals, params }) => {
    const sopId = params.sop_id;
    if (!sopId) return fail(400, { message: 'Missing sop_id' });
    try {
      await locals.api.sops.favorite(sopId);
      return { success: true };
    } catch (err) {
      console.error('favorite', err);
      return fail(500, { message: 'Failed to favorite' });
    }
  },

  unfavorite: async ({ locals, params }) => {
    const sopId = params.sop_id;
    if (!sopId) return fail(400, { message: 'Missing sop_id' });
    try {
      await locals.api.sops.unfavorite(sopId);
      return { success: true };
    } catch (err) {
      console.error('unfavorite', err);
      return fail(500, { message: 'Failed to unfavorite' });
    }
  }
};