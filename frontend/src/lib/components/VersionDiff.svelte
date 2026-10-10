<script lang="ts">
	import type { VersionDiff } from '$lib/sdk/types';
	import * as m from '$lib/paraglide/messages.js';

	/*
	 * The line diff against the previous version, without a card: the version
	 * page shows it as the Changes tab beside the document. Render it only when
	 * `diff.comparable` is true.
	 */
	let { diff }: { diff: VersionDiff } = $props();

	const lines = $derived(diff.lines ?? []);
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
				class="whitespace-pre-wrap px-3 {line.kind === 'add' ? 'bg-success/15' : line.kind === 'del'
					? 'bg-error/10'
					: ''}"
			>
				<span class="mr-2 select-none text-base-content/70"
					>{line.kind === 'add' ? '+' : line.kind === 'del' ? '−' : ' '}</span
				>{line.text}
			</div>
		{/each}
	</div>
{/if}
