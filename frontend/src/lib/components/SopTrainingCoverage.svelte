<script lang="ts">
	import { FileTextIcon, UsersIcon } from 'lucide-svelte';
	import type { SOPTrainingCoverage, TrainingMember } from '$lib/sdk/types';
	import EmptyState from './EmptyState.svelte';
	import ListRow from './ListRow.svelte';
	import * as m from '$lib/paraglide/messages.js';

	let { items = [] } = $props<{ items?: SOPTrainingCoverage[] }>();

	// Only published versions can be signed; most unsigned people first.
	const rows = $derived(
		(items ?? [])
			.filter((row: SOPTrainingCoverage) => row.has_published)
			.sort(
				(a: SOPTrainingCoverage, b: SOPTrainingCoverage) =>
					(b.unsigned?.length ?? 0) - (a.unsigned?.length ?? 0)
			)
	);

	function names(people: TrainingMember[] | undefined) {
		return (people ?? []).map((person) => person.display_name).join(', ');
	}
</script>

{#if rows.length === 0}
	<EmptyState
		icon={UsersIcon}
		title={m.training_empty_title()}
		description={m.training_empty_body()}
	/>
{:else}
	<ul class="list">
		{#each rows as row (row.sop_id)}
			{@const unsigned = row.unsigned?.length ?? 0}
			<ListRow href={`/sops/${row.sop_id}/v/latest`} title={row.title} icon={FileTextIcon}>
				{#if unsigned > 0}
					<p>{m.training_still()} {names(row.unsigned)}</p>
					{#if (row.signed ?? []).length > 0}
						<p class="text-xs text-base-content/70">{m.training_signed()} {names(row.signed)}</p>
					{/if}
				{/if}
				{#snippet meta()}<span class="font-mono">{m.common_version({ version: String(row.version) })}</span>{/snippet}
				{#snippet trailing()}
					{#if unsigned > 0}
						<span class="badge badge-soft badge-warning badge-sm">
							{m.training_unsigned_badge({ count: String(unsigned) })}
						</span>
					{:else}
						<span class="badge badge-soft badge-success badge-sm">{m.training_all_signed_badge()}</span>
					{/if}
				{/snippet}
			</ListRow>
		{/each}
	</ul>
{/if}
