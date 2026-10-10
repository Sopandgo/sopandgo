<script lang="ts">
	import { onMount } from 'svelte';
	import Alert from '$lib/components/Alert.svelte';
	import Card from '$lib/components/Card.svelte';
	import CardPageHeading from '$lib/components/CardPageHeading.svelte';
	import EmptyState from '$lib/components/EmptyState.svelte';
	import ListHeading from '$lib/components/ListHeading.svelte';
	import ListRow from '$lib/components/ListRow.svelte';
	import FavoriteToggle from '$lib/components/FavoriteToggle.svelte';
	import TagList from '$lib/components/TagList.svelte';
	import { partitionSignatureStatus, pendingSignatureCount } from '$lib/signatureBuckets';
	import {
		greetingPeriodForDate,
		timeZoneCookieAssignment,
		type GreetingPeriod
	} from '$lib/greeting';
	import SopTrainingCoverage from '$lib/components/SopTrainingCoverage.svelte';
	import { CircleCheckIcon, FileTextIcon, MegaphoneIcon, NotebookIcon, StarIcon } from 'lucide-svelte';
	import * as m from '$lib/paraglide/messages.js';
	import { getLocale } from '$lib/paraglide/runtime';

	let { data } = $props();

	const user = $derived(data.user);
	// null means that card's request failed; the card says so instead of looking empty
	const signatureFailed = $derived(data.signatureStatus === null);
	const favoritesFailed = $derived(data.favorites === null);
	const publishesFailed = $derived(data.recentPublishes === null);
	const trainingFailed = $derived(data.trainingCoverage === null);

	const signatureStatus = $derived(data.signatureStatus ?? []);
	const favoriteSops = $derived(data.favorites?.sops ?? []);
	const favoritesTotal = $derived(data.favorites?.total ?? 0);
	const recentPublishes = $derived(data.recentPublishes ?? []);
	const trainingCoverage = $derived(data.trainingCoverage ?? []);
	const showTraining = $derived(user?.role === 'admin' || user?.role === 'approver');

	const { actionRequired, notStarted } = $derived(partitionSignatureStatus(signatureStatus));
	const pending = $derived(pendingSignatureCount(signatureStatus));

	// The server greets in the time zone from the tz cookie; the browser's own clock
	// wins once mounted (first visit, travel) and refreshes the cookie for next time.
	let localGreetingPeriod = $state<GreetingPeriod | null>(null);
	const greetingPeriod = $derived(localGreetingPeriod ?? data.greetingPeriod);

	onMount(() => {
		localGreetingPeriod = greetingPeriodForDate(new Date());
		const timeZone = Intl.DateTimeFormat().resolvedOptions().timeZone;
		if (timeZone) document.cookie = timeZoneCookieAssignment(timeZone, location.protocol === 'https:');
	});
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
	{#if user}
		<Card>
			<div class="card-body">
				<div class="flex items-start justify-between gap-4">
					<div class="flex min-w-0 flex-col gap-1">
						<CardPageHeading>{greeting}</CardPageHeading>
						{#if signatureFailed}
							<!-- No status line: the action required card explains the failure -->
						{:else if pending === 0}
							<p class="flex items-center gap-2 text-sm text-base-content/70">
								<CircleCheckIcon class="size-4 shrink-0 text-success" aria-hidden="true" />
								{m.dashboard_caught_up()}
							</p>
						{:else}
							<p class="text-sm text-base-content/70">
								{pending === 1 ? m.dashboard_pending_one() : m.dashboard_pending_other({ count: String(pending) })}
							</p>
							{#if notStarted.length > 0 && actionRequired.length > 0}
								<p class="text-sm text-base-content/70">
									{m.dashboard_split({
										never: String(notStarted.length),
										updated: String(actionRequired.length)
									})}
								</p>
							{/if}
						{/if}
					</div>
					<a
						href="/sops"
						class="btn btn-primary shrink-0"
						aria-label={m.common_all_sops()}
						title={m.common_all_sops()}
					>
						<NotebookIcon class="size-4" aria-hidden="true" />
						<span class="hidden sm:inline">{m.common_all_sops()}</span>
					</a>
				</div>
			</div>
		</Card>

		<Card
			as="section"
			aria-labelledby="action-required-heading"
			headingId="action-required-heading"
			title={m.dashboard_action_required()}
			description={m.dashboard_action_help()}
		>
			{#if signatureFailed}
				{@render loadFailed()}
			{:else if actionRequired.length === 0 && notStarted.length === 0}
				<EmptyState
					icon={CircleCheckIcon}
					title={m.dashboard_caught_up_title()}
					description={m.dashboard_caught_up_body()}
				/>
			{:else}
				<ul class="list">
					{#if actionRequired.length > 0}
						<ListHeading>{m.dashboard_outdated()}</ListHeading>
						{#each actionRequired as sop (sop.sop_id)}
							<ListRow
								href={sopLatestHref(sop.sop_id)}
								title={sop.title}
								meta={m.common_version_published({ version: String(sop.latest_version) })}
								icon={FileTextIcon}
								attention
							>
								{#snippet trailing()}
									<span class="badge badge-soft badge-warning badge-sm hidden sm:inline-flex">{m.dashboard_update()}</span>
								{/snippet}
							</ListRow>
						{/each}
					{/if}

					{#if notStarted.length > 0}
						<ListHeading>{m.dashboard_new_unsigned()}</ListHeading>
						{#each notStarted as sop (sop.sop_id)}
							<ListRow
								href={sopLatestHref(sop.sop_id)}
								title={sop.title}
								icon={FileTextIcon}
							>
								{#snippet meta()}<span class="font-mono">{m.common_version({ version: String(sop.latest_version) })}</span>{/snippet}
								{#snippet trailing()}
									<span class="badge badge-outline badge-sm hidden sm:inline-flex">{m.dashboard_review()}</span>
								{/snippet}
							</ListRow>
						{/each}
					{/if}
				</ul>
			{/if}
		</Card>

		<Card
			as="section"
			aria-labelledby="whats-new-heading"
			headingId="whats-new-heading"
			title={m.dashboard_whats_new()}
			description={m.dashboard_recent()}
		>
			{#if publishesFailed}
				{@render loadFailed()}
			{:else if recentPublishes.length === 0}
				<EmptyState
					icon={MegaphoneIcon}
					title={m.dashboard_published_empty_title()}
					description={m.dashboard_published_empty_body()}
				/>
			{:else}
				<ul class="list">
					{#each recentPublishes as item (item.version_id)}
						<ListRow href={sopLatestHref(item.sop_id)} title={item.title} icon={FileTextIcon}>
							{#if item.change_summary}<span class="line-clamp-2">{item.change_summary}</span>{/if}
							{#snippet meta()}
								<span class="font-mono">{m.common_version({ version: String(item.version) })}</span>
								{#if item.published_by}· {item.published_by}{/if}
								· {new Date(item.published_at).toLocaleDateString(getLocale())}
							{/snippet}
						</ListRow>
					{/each}
				</ul>
			{/if}
		</Card>

		{#if showTraining}
			<Card
				as="section"
				aria-labelledby="training-heading"
				headingId="training-heading"
				title={m.dashboard_training_title()}
				description={m.dashboard_training_help()}
			>
				{#if trainingFailed}
					{@render loadFailed()}
				{:else}
					<SopTrainingCoverage items={trainingCoverage} />
				{/if}
			</Card>
		{/if}

		<Card
			as="section"
			aria-labelledby="favorites-heading"
			headingId="favorites-heading"
			title={m.dashboard_favorites()}
			description={m.dashboard_favorites_help()}
		>
			{#snippet actions()}
				{#if favoriteSops.length > 0}
					{#if favoritesTotal > favoriteSops.length}
						<span class="self-center text-sm text-base-content/70">
							{m.sops_showing({ shown: String(favoriteSops.length), total: String(favoritesTotal) })}
						</span>
					{/if}
					<a href="/sops?favorites_only=true" class="btn btn-sm">{m.dashboard_view_favorites()}</a>
				{/if}
			{/snippet}

			{#if favoritesFailed}
				{@render loadFailed()}
			{:else if favoriteSops.length === 0}
				<EmptyState
					icon={StarIcon}
					title={m.sops_empty_fav_title()}
					description={m.dashboard_no_favorites()}
				>
					{#snippet actions()}
						<a href="/sops" class="btn">{m.common_all_sops()}</a>
					{/snippet}
				</EmptyState>
			{:else}
				<ul class="list">
					{#each favoriteSops as sop (sop.id)}
						<ListRow href={sopLatestHref(sop.id)} title={sop.title} icon={FileTextIcon}>
							{#snippet metaExtra()}
								<TagList tags={sop.tags} tagHref={(tag) => `/sops?tag_id=${tag.id}`} />
							{/snippet}
							{#snippet trailing()}
								<FavoriteToggle sopId={sop.id} title={sop.title} isFavorite />
							{/snippet}
						</ListRow>
					{/each}
				</ul>
			{/if}
		</Card>
	{/if}
</div>

{#snippet loadFailed()}
	<div class="p-4 sm:p-6">
		<Alert type="error" message={m.dashboard_load_failed()} />
	</div>
{/snippet}
