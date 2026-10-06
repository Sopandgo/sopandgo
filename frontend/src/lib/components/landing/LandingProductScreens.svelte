<script lang="ts">
	import * as m from '$lib/paraglide/messages.js';

	const shots = $derived([
		{
			src: '/landing/browse-sops.png',
			alt: m.landing_shot_browse_alt(),
			title: m.landing_shot_browse_title(),
			caption: m.landing_shot_browse_caption()
		},
		{
			src: '/landing/version-ack.png',
			alt: m.landing_shot_version_alt(),
			title: m.landing_shot_version_title(),
			caption: m.landing_shot_version_caption()
		},
		{
			src: '/landing/backup-admin.png',
			alt: m.landing_shot_admin_alt(),
			title: m.landing_shot_admin_title(),
			caption: m.landing_shot_admin_caption()
		}
	]);
</script>

<section class="space-y-6" aria-labelledby="landing-previews-heading">
	<div class="max-w-2xl space-y-2">
		<h2 id="landing-previews-heading" class="text-2xl font-bold">{m.landing_inside()}</h2>
		<p class="text-base-content/70 text-sm md:text-base">
			{m.landing_inside_body()}
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
