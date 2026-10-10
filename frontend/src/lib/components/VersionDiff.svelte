<script lang="ts">
	import type { SOPAsset, VersionDiff } from '$lib/sdk/types';
	import { assetUrl, imagesInLine } from '$lib/sopAssets';
	import * as m from '$lib/paraglide/messages.js';

	/*
	 * The line diff against the previous version, without a card: the version
	 * page shows it as the Changes tab beside the document. Render it only when
	 * `diff.comparable` is true.
	 *
	 * Added and removed lines that embed an uploaded image show it under the line,
	 * so a swapped figure is visible, not just a changed file name. Assets belong to
	 * the SOP and are never rewritten, so removed lines resolve against the same list.
	 * Unchanged lines stay text: their image is unchanged too.
	 */
	let { diff, assets = [] }: { diff: VersionDiff; assets?: SOPAsset[] | null } = $props();

	const lines = $derived(
		(diff.lines ?? []).map((line) => ({
			...line,
			images: line.kind === 'context' ? [] : imagesInLine(line.text, assets)
		}))
	);
</script>

<p class="text-sm text-base-content/70">{m.diff_compared({ version: String(diff.from_version) })}</p>

{#if lines.length === 0}
	<p class="mt-4 text-sm text-base-content/70">{m.diff_identical()}</p>
{:else}
	<div
		class="mt-4 overflow-x-auto rounded-field border border-base-300 py-2 font-mono text-xs leading-relaxed sm:text-sm"
		aria-label={m.diff_aria()}
	>
		{#each lines as line, index (`${index}-${line.kind}`)}
			<div
				class="flex px-3 {line.kind === 'add' ? 'bg-success/15' : line.kind === 'del'
					? 'bg-error/10'
					: ''}"
			>
				<span class="mr-2 shrink-0 select-none text-base-content/70"
					>{line.kind === 'add' ? '+' : line.kind === 'del' ? '−' : ' '}</span
				>
				<div class="min-w-0 flex-1">
					<div class="min-h-[1lh] whitespace-pre-wrap">{line.text}</div>
					{#if line.images.length > 0}
						<div class="flex flex-wrap gap-2 py-2">
							{#each line.images as image, i (i)}
								<img
									src={assetUrl(image.asset)}
									alt={image.alt}
									title={image.asset.file_name}
									loading="lazy"
									class="h-40 w-auto max-w-full rounded-field border border-base-300 bg-base-100 object-contain"
								/>
							{/each}
						</div>
					{/if}
				</div>
			</div>
		{/each}
	</div>
{/if}
