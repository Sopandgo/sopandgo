<script lang="ts">
	import Card from './Card.svelte';
	import type { VersionDiff } from '$lib/sdk/types';
	import * as m from '$lib/paraglide/messages.js';

	let { diff } = $props<{ diff: VersionDiff | null }>();

	const lines = $derived(diff?.lines ?? []);
</script>

{#if diff?.comparable}
	<section aria-labelledby="version-diff-heading">
		<Card
			headingId="version-diff-heading"
			title={m.editor_what_changed()}
			description={m.diff_compared({ version: String(diff.from_version) })}
		>
			{#if lines.length === 0}
				<p class="p-4 text-sm text-base-content/70">{m.diff_identical()}</p>
			{:else}
				<div
					class="max-h-[28rem] overflow-auto p-4 font-mono text-xs leading-relaxed sm:text-sm"
					aria-label={m.diff_aria()}
				>
					{#each lines as line, index (`${index}-${line.kind}`)}
						<div
							class="whitespace-pre-wrap {line.kind === 'add' ? 'bg-success/15' : line.kind === 'del'
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
		</Card>
	</section>
{/if}
