<script lang="ts">
	import type { VersionDiff } from '$lib/sdk/types';
	import * as m from '$lib/paraglide/messages.js';

	let { diff } = $props<{ diff: VersionDiff | null }>();

	const lines = $derived(diff?.lines ?? []);
</script>

{#if diff?.comparable}
	<section aria-labelledby="version-diff-heading">
		<div class="card bg-base-100 border border-base-200 shadow-sm">
			<div class="border-b border-base-200 p-4 sm:p-5">
				<h2 id="version-diff-heading" class="text-lg font-semibold">{m.editor_what_changed()}</h2>
				<p class="mt-1 text-sm text-base-content/70">
					{m.diff_compared({ version: String(diff.from_version) })}
				</p>
			</div>
			{#if lines.length === 0}
				<p class="p-4 text-sm text-base-content/60">{m.diff_identical()}</p>
			{:else}
				<div
					class="max-h-[28rem] overflow-auto p-4 font-mono text-xs leading-relaxed sm:text-sm"
					aria-label={m.diff_aria()}
				>
					{#each lines as line, index (`${index}-${line.kind}`)}
						<div
							class="whitespace-pre-wrap {line.kind === 'add'
								? 'bg-success/15'
								: line.kind === 'del'
									? 'bg-error/10'
									: ''}"
						>
							<span class="mr-2 select-none opacity-50"
								>{line.kind === 'add' ? '+' : line.kind === 'del' ? '−' : ' '}</span
							>{line.text}
						</div>
					{/each}
				</div>
			{/if}
		</div>
	</section>
{/if}
