<script lang="ts">
	const shots = [
		{
			src: '/landing/browse-sops.png',
			alt: 'SOP library with search, tags, favorites, and a list of procedures.',
			title: 'Browse and organize',
			caption:
				'Find procedures quickly with search, tags, and per-user favorites — then open any SOP from a single library view.'
		},
		{
			src: '/landing/version-ack.png',
			alt: 'SOP detail showing tags, immutable version history, and associated assets.',
			title: 'Versions and acknowledgments',
			caption:
				'Each version is immutable; tags and assets stay with the procedure, and acknowledgments align to the version people actually read.'
		},
		{
			src: '/landing/backup-admin.png',
			alt: 'User management with account list, roles, and a form to register new users.',
			title: 'Administration and backups',
			caption:
				'Manage users, roles, and invites from built-in admin tools, alongside mail, sessions, and backup exports for day-two operations.'
		}
	] as const;
</script>

<section class="space-y-6" aria-labelledby="landing-previews-heading">
	<div class="max-w-2xl space-y-2">
		<h2 id="landing-previews-heading" class="text-2xl font-bold">Inside the application</h2>
		<p class="text-base-content/70 text-sm md:text-base">
			Product screens from the SOP library, versioned procedures, and administration tools as they
			appear in the application today.
		</p>
	</div>
	<div class="grid gap-8 lg:grid-cols-3">
		{#each shots as item, i (item.src)}
			<figure class="card bg-base-100 shadow-sm">
				<div class="bg-base-200/80 border-base-300 border-b p-2">
					<img
						src={item.src}
						alt={item.alt}
						width="1483"
						height="1307"
						class="w-full rounded-md"
						loading={i === 0 ? 'eager' : 'lazy'}
						fetchpriority={i === 0 ? 'high' : undefined}
						decoding="async"
					/>
				</div>
				<figcaption class="card-body gap-2 pt-4">
					<h3 class="card-title text-base">{item.title}</h3>
					<p class="text-base-content/70 text-sm">{item.caption}</p>
				</figcaption>
			</figure>
		{/each}
	</div>
</section>
