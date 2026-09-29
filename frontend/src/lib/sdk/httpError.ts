/** Structured failure from the backend API (JSON body or plain text). */
export class SdkHttpError extends Error {
	override readonly name = 'SdkHttpError';
	readonly status: number;
	readonly rawBody: string;
	/** Top-level JSON `error` string from the API. */
	readonly apiError?: string;
	/** Extra machine code (e.g. markdown policy `code`). */
	readonly policyCode?: string;
	readonly detail?: string;

	constructor(
		status: number,
		rawBody: string,
		parsed?: Partial<{ error: string; code: string; detail: string }>
	) {
		const detail = parsed?.detail?.trim();
		const headline =
			detail || parsed?.error || (rawBody.trim() !== '' ? rawBody.trim() : `Request failed (${status})`);
		super(headline);
		this.status = status;
		this.rawBody = rawBody;
		if (parsed?.error) this.apiError = parsed.error;
		if (parsed?.code) this.policyCode = parsed.code;
		if (parsed?.detail) this.detail = parsed.detail;
	}
}

export async function sdkHttpErrorFromResponse(res: Response): Promise<SdkHttpError> {
	const rawBody = await res.text();
	let parsed: { error?: string; code?: string; detail?: string } | undefined;
	const ct = res.headers.get('content-type') ?? '';
	if (ct.includes('application/json')) {
		try {
			parsed = JSON.parse(rawBody) as { error?: string; code?: string; detail?: string };
		} catch {
			/* ignore */
		}
	}
	return new SdkHttpError(res.status, rawBody, parsed);
}
