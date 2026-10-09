<script lang="ts">
	import Breadcrumbs from '$lib/components/Breadcrumbs.svelte';
	import Card from '$lib/components/Card.svelte';
	import ListRow from '$lib/components/ListRow.svelte';
	import {
		greetingPeriodForDate,
		partitionSignatureStatus,
		pendingSignatureCount
	} from '$lib/signatureBuckets';
	import { enhance } from '$app/forms';
	import SopTrainingCoverage from '$lib/components/SopTrainingCoverage.svelte';
	import {
		AlertTriangleIcon,
		FileQuestionIcon,
		HouseIcon,
		MegaphoneIcon,
		NotebookIcon,
		PartyPopperIcon,
		StarIcon,
		UsersIcon
	} from 'lucide-svelte';
	import * as m from '$lib/paraglide/messages.js';
	import { getLocale } from '$lib/paraglide/runtime';

	let { data } = $props();

	const user = $derived(data.user);
	const signatureStatus = $derived(data.signatureStatus ?? []);
	const favoriteSops = $derived(data.favorites?.sops ?? []);
	const recentPublishes = $derived(data.recentPublishes ?? []);
	const trainingCoverage = $derived(data.trainingCoverage ?? []);
	const showTraining = $derived(user?.role === 'admin' || user?.role === 'approver');

	const { actionRequired, notStarted } = $derived(partitionSignatureStatus(signatureStatus));
	const pending = $derived(pendingSignatureCount(signatureStatus));

	const greetingPeriod = $derived(greetingPeriodForDate(new Date()));
	const greeting = $derived(
		greetingPeriod === 'morning'
			? m.dashboard_greeting_morning({ name: user?.display_name ?? '' })
			: greetingPeriod === 'afternoon'
				? m.dashboard_greeting_afternoon({ name: user?.display_name ?? '' })
				: m.dashboard_greeting_evening({ name: user?.display_name ?? '' })
	);

	function sopLatestHref(sopId: string) {
		return `/sops/${sopId}/v/latest`;
	}
</script>

<svelte:head>
	<title>{m.page_dashboard()}</title>
</svelte:head>

<div class="flex flex-col gap-6">
	<Breadcrumbs items={[{ label: m.page_dashboard(), icon: HouseIcon }]} />

	{#if user}
		<section class="rounded-box border border-base-200 bg-base-100 px-5 py-6 shadow-sm sm:px-8">
			<h1 class="text-xl font-semibold sm:text-2xl">
				{greeting}
			</h1>
			{#if pending === 0}
				<p class="mt-2 text-base-content/80">
					<span class="inline-flex items-center gap-2">
						<PartyPopperIcon class="h-5 w-5 shrink-0 text-success" aria-hidden="true" />
						{m.dashboard_caught_up()}
					</span>
				</p>
			{:else}
				<p class="mt-2 text-base-content/80">
					{pending === 1 ? m.dashboard_pending_one() : m.dashboard_pending_other({ count: String(pending) })}
				</p>
				{#if notStarted.length > 0 && actionRequired.length > 0}
					<p class="mt-1 text-sm text-base-content/60">
						{m.dashboard_split({
							never: String(notStarted.length),
							updated: String(actionRequired.length)
						})}
					</p>
				{/if}
			{/if}
			<div class="mt-5 flex flex-wrap gap-2">
				<a href="/sops" class="btn btn-primary gap-2">
					<NotebookIcon class="h-4 w-4 shrink-0" aria-hidden="true" />
					{m.common_all_sops()}
				</a>
			</div>
		</section>

		<section aria-labelledby="action-required-heading">
			<Card>
				<div class="border-b border-base-200 p-4 sm:p-5">
					<h2 id="action-required-heading" class="text-lg font-semibold">
						{m.dashboard_action_required()}
					</h2>
					<p class="mt-1 text-sm text-base-content/70">
						{m.dashboard_action_help()}
					</p>
				</div>

				{#if actionRequired.length === 0 && notStarted.length === 0}
					<div class="p-8 text-center text-sm text-base-content/60">
						{m.dashboard_nothing_before()}<a href="/sops" class="link link-primary">{m.dashboard_sop_library()}</a>{m.dashboard_nothing_after()}
					</div>
				{:else}
					<ul class="list">
						{#if notStarted.length > 0}
							<li
								class="border-b border-base-200 bg-base-200/20 px-4 py-3 text-xs font-bold uppercase tracking-widest opacity-60"
							>
								{m.dashboard_new_unsigned()}
							</li>
							{#each notStarted as sop (sop.sop_id)}
								<ListRow
									href={sopLatestHref(sop.sop_id)}
									title={sop.title}
									meta={m.common_version({ version: String(sop.latest_version) })}
									icon={FileQuestionIcon}
									chevron
								>
									{#snippet trailing()}
										<span class="badge badge-ghost badge-sm hidden sm:inline-flex">{m.dashboard_review()}</span>
									{/snippet}
								</ListRow>
							{/each}
						{/if}

						{#if actionRequired.length > 0}
							<li
								class="border-b border-base-200 bg-warning/10 px-4 py-3 text-xs font-bold uppercase tracking-widest text-warning"
							>
								{m.dashboard_outdated()}
							</li>
							{#each actionRequired as sop (sop.sop_id)}
								<ListRow
									href={sopLatestHref(sop.sop_id)}
									title={sop.title}
									meta={m.common_version_published({ version: String(sop.latest_version) })}
									icon={AlertTriangleIcon}
									iconTone="warning"
									tone="warning"
									chevron
								>
									{#snippet trailing()}
										<span class="badge badge-warning badge-sm hidden sm:inline-flex">{m.dashboard_update()}</span>
									{/snippet}
								</ListRow>
							{/each}
						{/if}
					</ul>
				{/if}
			</Card>
		</section>

		<section aria-labelledby="whats-new-heading">
			<Card>
				<div class="border-b border-base-200 p-4 sm:p-5">
					<h2 id="whats-new-heading" class="flex items-center gap-2 text-lg font-semibold">
						<MegaphoneIcon class="h-5 w-5 shrink-0" aria-hidden="true" />
						{m.dashboard_whats_new()}
					</h2>
					<p class="mt-1 text-sm text-base-content/70">
						{m.dashboard_recent()}
					</p>
				</div>
				{#if recentPublishes.length === 0}
					<div class="p-8 text-center text-sm text-base-content/60">
						{m.dashboard_nothing_published()}
					</div>
				{:else}
					<ul class="list">
						{#each recentPublishes as item (item.version_id)}
							<ListRow
								href={sopLatestHref(item.sop_id)}
								title={item.title}
								meta={[
									m.common_version({ version: String(item.version) }),
									item.published_by,
									new Date(item.published_at).toLocaleDateString(getLocale())
								]
									.filter(Boolean)
									.join(' · ')}
								chevron
							>
								{#if item.change_summary}{item.change_summary}{/if}
							</ListRow>
						{/each}
					</ul>
				{/if}
			</Card>
		</section>

		{#if showTraining}
			<section aria-labelledby="training-heading">
				<Card>
					<div class="border-b border-base-200 p-4 sm:p-5">
						<h2 id="training-heading" class="flex items-center gap-2 text-lg font-semibold">
							<UsersIcon class="h-5 w-5 shrink-0" aria-hidden="true" />
							{m.dashboard_training_title()}
						</h2>
						<p class="mt-1 text-sm text-base-content/70">
							{m.dashboard_training_help()}
						</p>
					</div>
					<SopTrainingCoverage items={trainingCoverage} />
				</Card>
			</section>
		{/if}

		<section aria-labelledby="favorites-heading">
			<Card>
				<div
					class="flex flex-col gap-2 border-b border-base-200 p-4 sm:flex-row sm:items-end sm:justify-between sm:p-5"
				>
					<div>
						<h2 id="favorites-heading" class="text-lg font-semibold">{m.dashboard_favorites()}</h2>
						<p class="mt-1 text-sm text-base-content/70">{m.dashboard_favorites_help()}</p>
					</div>
					<div class="flex flex-wrap gap-2 self-start sm:self-auto">
						<a href="/sops" class="btn btn-outline btn-sm">{m.common_all_sops()}</a>
						<a href="/sops?favorites_only=true" class="btn btn-ghost btn-sm"> {m.dashboard_view_favorites()} </a>
					</div>
				</div>

				{#if favoriteSops.length === 0}
					<div class="p-8 text-center text-sm text-base-content/60">
						{m.dashboard_no_favorites_before()}<strong class="font-medium text-base-content/80">{m.sops_add_to_favorites()}</strong>{m.dashboard_no_favorites_after()}
					</div>
				{:else}
					<ul
						class="grid gap-3 p-4 sm:grid-cols-2 sm:p-5 lg:grid-cols-3"
					>
						{#each favoriteSops as sop (sop.id)}
							<Card as="li" variant="subtle" class="border-base-200/80">
								<div class="card-body flex flex-row items-start gap-3 p-4">
									<div class="avatar avatar-placeholder shrink-0" aria-hidden="true">
										<div
											class="bg-secondary text-secondary-content flex h-10 w-10 items-center justify-center rounded-field"
										>
											<NotebookIcon class="h-5 w-5" />
										</div>
									</div>
									<div class="min-w-0 flex-1">
										<a
											href={sopLatestHref(sop.id)}
											class="link-hover link-primary font-semibold leading-snug"
										>
											{sop.title}
										</a>
										<div class="mt-2 flex flex-wrap items-center gap-2">
											{#each sop.tags as tag (tag.id)}
												<span class="badge badge-ghost badge-xs">{tag.title}</span>
											{/each}
										</div>
									</div>
									<form method="POST" action="?/unfavorite" use:enhance class="shrink-0">
										<input type="hidden" name="sop_id" value={sop.id} />
										<button
											type="submit"
											class="btn btn-square btn-ghost btn-sm text-warning"
											aria-label={m.aria_remove_favorite({ title: sop.title })}
										>
											<StarIcon class="h-4 w-4 fill-current" />
										</button>
									</form>
								</div>
							</Card>
						{/each}
					</ul>
				{/if}
			</Card>
		</section>
	{/if}
</div>
