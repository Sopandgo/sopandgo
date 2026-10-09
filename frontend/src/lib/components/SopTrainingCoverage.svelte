<script lang="ts">
	import type { SOPTrainingCoverage, TrainingMember } from '$lib/sdk/types';
	import * as m from '$lib/paraglide/messages.js';

	let { items = [] } = $props<{ items?: SOPTrainingCoverage[] }>();

	const rows = $derived(
		[...(items ?? [])].sort((a, b) => (b.unsigned?.length ?? 0) - (a.unsigned?.length ?? 0))
	);
</script>

<ul class="list">
	{#each rows as row (row.sop_id)}
		<li class="border-b border-base-300 last:border-b-0">
			<div class="flex flex-col gap-2 px-4 py-3 sm:px-5">
				<div class="flex flex-wrap items-center justify-between gap-2">
					<a href={`/sops/${row.sop_id}/v/latest`} class="font-semibold underline">
						{row.title}
					</a>
					{#if row.has_published}
						<span class="text-xs font-mono text-base-content/70">
							{m.common_version({ version: String(row.version) })}
						</span>
					{/if}
				</div>
				{#if !row.has_published}
					<p class="text-sm text-base-content/70">{m.training_no_published()}</p>
				{:else if (row.unsigned ?? []).length === 0}
					<p class="text-sm text-success">{m.training_all_signed()}</p>
				{:else}
					<p class="text-sm text-base-content/80">
						{m.training_still()}
						{row.unsigned.map((person: TrainingMember) => person.display_name).join(', ')}
					</p>
					{#if (row.signed ?? []).length > 0}
						<p class="text-xs text-base-content/70">
							{m.training_signed()} {row.signed.map((person: TrainingMember) => person.display_name).join(', ')}
						</p>
					{/if}
				{/if}
			</div>
		</li>
	{:else}
		<li class="p-8 text-center text-sm text-base-content/70">{m.training_empty()}</li>
	{/each}
</ul>
