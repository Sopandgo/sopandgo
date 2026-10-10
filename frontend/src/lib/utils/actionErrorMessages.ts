import * as m from '#lib/paraglide/messages.js';
import { SdkHttpError } from '#lib/sdk/httpError.js';

/** User-facing message when publishing a new SOP version fails. */
export function messageForPublishFailure(err: unknown): string {
	if (err instanceof SdkHttpError) {
		const { status, policyCode, detail, apiError } = err;
		const d = detail?.trim();

		switch (policyCode) {
			case 'raw_html':
				return d ? m.error_raw_html_detail({ detail: d }) : m.error_raw_html();
			case 'disallowed_scheme':
				return d ? m.error_disallowed_scheme_detail({ detail: d }) : m.error_disallowed_scheme();
			case 'disallowed_url':
				return d ? m.error_disallowed_url_detail({ detail: d }) : m.error_disallowed_url();
			case 'invalid_url':
				return d ? m.error_invalid_url_detail({ detail: d }) : m.error_invalid_url();
			case 'url_too_long':
				return m.error_url_too_long();
			case 'content_too_large':
				return d ?? m.error_content_too_large();
			default:
				break;
		}

		if (
			apiError === 'invalid_asset_reference' ||
			(d && (d.includes('referenced asset not found') || d.includes('invalid asset URL encoding')))
		) {
			return d ? m.error_asset_reference_detail({ detail: d }) : m.error_asset_reference();
		}

		if (apiError === 'invalid_change_summary') {
			return d ?? m.error_change_summary();
		}

		if (apiError === 'conflict' || status === 409) {
			return d ?? m.error_conflict();
		}

		if (status === 401) return m.error_session_expired();
		if (status === 403) return m.error_publish_forbidden();

		if (d) return d;
		if (status === 400) return m.error_document_rejected();
		if (status >= 500) return m.error_save_version();
		return err.message || m.error_save_version_generic();
	}

	if (err instanceof Error && err.message === 'SERVICE_UNAVAILABLE') {
		return m.error_service_unavailable();
	}

	return m.error_save_generic();
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
			return d ?? m.error_asset_exists();
		}
		if (apiError === 'multipart_invalid') {
			return d ?? m.error_upload_interrupted();
		}
		if (apiError === 'missing_file') {
			return d ?? m.error_no_file();
		}
		if (apiError === 'invalid_asset_filename') {
			return d ?? m.error_bad_filename();
		}
		if (d) return d;
		if (status === 403) return m.error_upload_forbidden();
		if (status >= 500) return m.error_upload_server();
		return err.message || m.error_upload_failed();
	}

	if (err instanceof Error && err.message === 'SERVICE_UNAVAILABLE') {
		return m.error_service_unavailable();
	}

	return m.error_upload_retry();
}
