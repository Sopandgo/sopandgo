import { SdkHttpError } from '$lib/sdk/httpError';

/** User-facing message when publishing a new SOP version fails. */
export function messageForPublishFailure(err: unknown): string {
	if (err instanceof SdkHttpError) {
		const { status, policyCode, detail, apiError } = err;
		const d = detail?.trim();

		switch (policyCode) {
			case 'raw_html':
				return d
					? `This document cannot include raw HTML (${d}). Use Markdown only—not HTML tags.`
					: 'This document cannot include raw HTML. Use Markdown only—not HTML tags.';
			case 'disallowed_scheme':
				return d
					? `A link or image uses a URL that is not allowed (${d}). Use https://, http://, mailto:, or assets/your-file.png for attachments.`
					: 'A link or image uses a URL that is not allowed. Use https://, mailto:, or assets/… for files.';
			case 'disallowed_url':
				return d ? `A link URL is not allowed: ${d}` : 'A link in the document uses a URL that is not allowed.';
			case 'invalid_url':
				return d ? `A link could not be read: ${d}` : 'A link in the document is invalid.';
			case 'url_too_long':
				return 'A mailto link in the document is too long. Shorten or remove it.';
			case 'content_too_large':
				return d ?? 'This document is too large to save.';
			default:
				break;
		}

		if (
			apiError === 'invalid_asset_reference' ||
			(d && (d.includes('referenced asset not found') || d.includes('invalid asset URL encoding')))
		) {
			return d
				? `Attachment reference: ${d}. Upload the file under Assets in the sidebar and use exactly ![description](assets/filename).`
				: 'The document references a file that is not uploaded for this SOP. Add it under Assets or fix the path.';
		}

		if (apiError === 'invalid_change_summary') {
			return d ?? 'Add a short note describing what changed and why.';
		}

		if (apiError === 'conflict' || status === 409) {
			return d ?? 'A version is already waiting for approval. Resolve that before creating another draft.';
		}

		if (status === 401) return 'Your session expired. Sign in again and retry.';
		if (status === 403) return 'You do not have permission to publish this SOP.';

		if (d) return d;
		if (status === 400) {
			return 'The server rejected this document. Check Markdown, links, and attachment references.';
		}
		if (status >= 500) return 'The server could not save this version. Try again in a moment.';
		return err.message || 'Could not save this version.';
	}

	if (err instanceof Error && err.message === 'SERVICE_UNAVAILABLE') {
		return 'Could not reach the server. Check your connection and try again.';
	}

	return 'Something went wrong while saving. Please try again.';
}

/** User-facing message when an asset upload fails. */
export function messageForAssetUploadFailure(err: unknown): string {
	if (err instanceof SdkHttpError) {
		const { status, detail, apiError } = err;
		const d = detail?.trim();

		if (d && ['asset_exists', 'multipart_invalid', 'missing_file', 'invalid_asset_filename'].includes(apiError ?? '')) {
			return d;
		}
		if (apiError === 'asset_exists' || status === 409) {
			return d ?? 'That file name already exists for this SOP. Use a different name or remove the old asset first.';
		}
		if (apiError === 'multipart_invalid') {
			return d ?? 'The upload was too large or was interrupted. Try a smaller file or retry.';
		}
		if (apiError === 'missing_file') {
			return d ?? 'No file was selected.';
		}
		if (apiError === 'invalid_asset_filename') {
			return d ?? 'That file name is not allowed. Use a simple name (e.g. diagram.png) with no folders.';
		}
		if (d) return d;
		if (status === 403) return 'You do not have permission to upload files for this SOP.';
		if (status >= 500) return 'The server could not save the file. Try again in a moment.';
		return err.message || 'Upload failed.';
	}

	if (err instanceof Error && err.message === 'SERVICE_UNAVAILABLE') {
		return 'Could not reach the server. Check your connection and try again.';
	}

	return 'Upload failed. Please try again.';
}
