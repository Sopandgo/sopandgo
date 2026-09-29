<script lang="ts">
	import Breadcrumbs from '$lib/components/Breadcrumbs.svelte';
	import Card from '$lib/components/Card.svelte';
	import {
		greetingPeriodForDate,
		partitionSignatureStatus,
		pendingSignatureCount
	} from '$lib/signatureBuckets';
	import type { UserSignatureStatus } from '$lib/sdk/types';
	import { enhance } from '$app/forms';
	import SopTrainingCoverage from '$lib/components/SopTrainingCoverage.svelte';
	import {
		AlertTriangleIcon,
		ChevronRightIcon,
		FileQuestionIcon,
		HouseIcon,
		MegaphoneIcon,
		NotebookIcon,
		PartyPopperIcon,
		StarIcon,
		UsersIcon
	} from 'lucide-svelte';

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
	const greetingWord = $derived(
		greetingPeriod === 'morning'
			? 'morning'
			: greetingPeriod === 'afternoon'
				? 'afternoon'
				: 'evening'
	);

	function sopLatestHref(sopId: string) {
		return `/sops/${sopId}/v/latest`;
	}

	function rowForSignature(s: UserSignatureStatus) {
		return {
			href: sopLatestHref(s.sop_id),
			title: s.title,
			versionLabel: `Version ${s.latest_version}`
		};
	}
</script>

<div class="flex flex-col gap-6">
	<Breadcrumbs items={[{ label: 'Dashboard', icon: HouseIcon }]} />

	{#if user}
		<section class="rounded-box border border-base-200 bg-base-100 px-5 py-6 shadow-sm sm:px-8">
			<h1 class="text-xl font-semibold sm:text-2xl">
				Good {greetingWord}, {user.display_name}!
			</h1>
			{#if pending === 0}
				<p class="mt-2 text-base-content/80">
					<span class="inline-flex items-center gap-2">
						<PartyPopperIcon class="h-5 w-5 shrink-0 text-success" aria-hidden="true" />
						You are all caught up. Every published SOP you need is signed on the latest version.
					</span>
				</p>
			{:else}
				<p class="mt-2 text-base-content/80">
					You have <strong>{pending}</strong>
					{pending === 1 ? ' SOP' : ' SOPs'} that need your review before you are up to date.
				</p>
				{#if notStarted.length > 0 && actionRequired.length > 0}
					<p class="mt-1 text-sm text-base-content/60">
						{notStarted.length} never signed · {actionRequired.length} updated since you last signed
					</p>
				{/if}
			{/if}
			<div class="mt-5 flex flex-wrap gap-2">
				<a href="/sops" class="btn btn-primary gap-2">
					<NotebookIcon class="h-4 w-4 shrink-0" aria-hidden="true" />
					All SOPs
				</a>
			</div>
		</section>

		<section aria-labelledby="action-required-heading">
			<Card>
				<div class="border-b border-base-200 p-4 sm:p-5">
					<h2 id="action-required-heading" class="text-lg font-semibold">
						Action required
					</h2>
					<p class="mt-1 text-sm text-base-content/70">
						Sign the latest published version of each SOP below.
					</p>
				</div>

				{#if actionRequired.length === 0 && notStarted.length === 0}
					<div class="p-8 text-center text-sm text-base-content/60">
						Nothing pending. Browse the
						<a href="/sops" class="link link-primary">SOP library</a>
						if you need a document.
					</div>
				{:else}
					<ul class="list">
						{#if notStarted.length > 0}
							<li
								class="border-b border-base-200 bg-base-200/20 px-4 py-3 text-xs font-bold uppercase tracking-widest opacity-60"
							>
								New — not signed yet
							</li>
							{#each notStarted as sop (sop.sop_id)}
								{@const row = rowForSignature(sop)}
								<li class="border-b border-base-200/80 last:border-b-0">
									<a
										href={row.href}
										class="list-row hover:bg-base-200/40 flex items-center gap-3 transition-colors group"
									>
										<div class="avatar avatar-placeholder shrink-0" aria-hidden="true">
											<div
												class="bg-base-300 text-base-content flex h-10 w-10 items-center justify-center rounded-field sm:h-12 sm:w-12"
											>
												<FileQuestionIcon class="h-5 w-5" />
											</div>
										</div>
										<div class="min-w-0 flex-1">
											<div
												class="truncate text-sm font-bold group-hover:text-primary sm:text-base"
											>
												{row.title}
											</div>
											<div
												class="mt-0.5 font-mono text-[10px] uppercase tracking-tighter opacity-50"
											>
												{row.versionLabel}
											</div>
										</div>
										<span class="badge badge-ghost badge-sm hidden sm:inline-flex">Review</span>
										<ChevronRightIcon class="h-5 w-5 shrink-0 opacity-40" aria-hidden="true" />
									</a>
								</li>
							{/each}
						{/if}

						{#if actionRequired.length > 0}
							<li
								class="border-b border-base-200 bg-warning/10 px-4 py-3 text-xs font-bold uppercase tracking-widest text-warning"
							>
								Out of date — new version published
							</li>
							{#each actionRequired as sop (sop.sop_id)}
								{@const row = rowForSignature(sop)}
								<li class="border-b border-base-200/80 last:border-b-0">
									<a
										href={row.href}
										class="list-row hover:bg-warning/5 flex items-center gap-3 transition-colors group bg-warning/5"
									>
										<div class="avatar avatar-placeholder shrink-0" aria-hidden="true">
											<div
												class="flex h-10 w-10 items-center justify-center rounded-field bg-warning/20 text-warning sm:h-12 sm:w-12"
											>
												<AlertTriangleIcon class="h-5 w-5" />
											</div>
										</div>
										<div class="min-w-0 flex-1">
											<div
												class="truncate text-sm font-bold group-hover:text-primary sm:text-base"
											>
												{row.title}
											</div>
											<div
												class="mt-0.5 font-mono text-[10px] uppercase tracking-tighter opacity-60"
											>
												{row.versionLabel} published
											</div>
										</div>
										<span class="badge badge-warning badge-sm hidden sm:inline-flex">Update</span>
										<ChevronRightIcon class="h-5 w-5 shrink-0 opacity-40" aria-hidden="true" />
									</a>
								</li>
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
						What's new in the lab
					</h2>
					<p class="mt-1 text-sm text-base-content/70">
						Recently published SOP versions.
					</p>
				</div>
				{#if recentPublishes.length === 0}
					<div class="p-8 text-center text-sm text-base-content/60">
						Nothing has been published yet.
					</div>
				{:else}
					<ul class="list">
						{#each recentPublishes as item (item.version_id)}
							<li class="border-b border-base-200/80 last:border-b-0">
								<a
									href={`/sops/${item.sop_id}/v/latest`}
									class="list-row hover:bg-base-200/40 flex items-center gap-3 transition-colors group"
								>
									<div class="min-w-0 flex-1">
										<div class="truncate text-sm font-bold group-hover:text-primary sm:text-base">
											{item.title}
										</div>
										{#if item.change_summary}
											<div class="mt-0.5 text-sm text-base-content/80">{item.change_summary}</div>
										{/if}
										<div class="mt-0.5 font-mono text-[10px] uppercase tracking-tighter opacity-50">
											Version {item.version}
											{#if item.published_by}
												· {item.published_by}
											{/if}
											· {new Date(item.published_at).toLocaleDateString()}
										</div>
									</div>
									<ChevronRightIcon class="h-5 w-5 shrink-0 opacity-40" aria-hidden="true" />
								</a>
							</li>
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
							Who still needs to sign
						</h2>
						<p class="mt-1 text-sm text-base-content/70">
							Active people who can sign, compared with reader signatures on the latest published
							version.
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
						<h2 id="favorites-heading" class="text-lg font-semibold">Favorite SOPs</h2>
						<p class="mt-1 text-sm text-base-content/70">Quick access to documents you use often.</p>
					</div>
					<div class="flex flex-wrap gap-2 self-start sm:self-auto">
						<a href="/sops" class="btn btn-outline btn-sm">All SOPs</a>
						<a href="/sops?favorites_only=true" class="btn btn-ghost btn-sm"> View all favorites </a>
					</div>
				</div>

				{#if favoriteSops.length === 0}
					<div class="p-8 text-center text-sm text-base-content/60">
						You have not starred any SOPs yet. Open a document and choose
						<strong class="font-medium text-base-content/80">Add to favorites</strong>
						to pin it here.
					</div>
				{:else}
					<ul
						class="grid gap-3 p-4 sm:grid-cols-2 sm:p-5 lg:grid-cols-3"
					>
						{#each favoriteSops as sop (sop.id)}
							<li
								class="card card-border bg-base-200/30 border-base-200/80 shadow-none transition-colors hover:bg-base-200/50"
							>
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
											aria-label={`Remove ${sop.title} from favorites`}
										>
											<StarIcon class="h-4 w-4 fill-current" />
										</button>
									</form>
								</div>
							</li>
						{/each}
					</ul>
				{/if}
			</Card>
		</section>
	{/if}
</div>
