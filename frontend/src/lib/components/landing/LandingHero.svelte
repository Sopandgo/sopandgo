<section class="hero rounded-box bg-base-100 shadow-sm">
	<div class="hero-content flex-col gap-10 py-14 text-center lg:flex-row lg:items-start lg:py-16 lg:text-left">
		<div class="max-w-xl space-y-4">
			<p class="text-sm font-semibold uppercase tracking-wide text-primary">SOP and GO</p>
			<h1 class="text-4xl font-bold leading-tight sm:text-5xl">SOP management made simple.</h1>
			<p class="text-base-content/80 text-lg">
				A high-integrity, self-hosted SOP system for labs and small research teams — where
				procedures matter, but full-scale QMS software is unnecessary, expensive, or obstructive.
			</p>
			<div class="flex flex-col gap-3 sm:flex-row sm:flex-wrap sm:items-center">
				<a href="/login" class="btn btn-primary btn-lg">Sign in</a>
				<a
					href="https://github.com/Sopandgo/sopandgo/blob/main/docs/ops/evaluating.md"
					class="btn btn-outline btn-lg"
					rel="noreferrer"
					target="_blank"
				>
					Try locally
				</a>
				<a
					href="https://github.com/Sopandgo/sopandgo"
					class="btn btn-ghost btn-lg"
					rel="noreferrer"
					target="_blank"
				>
					View on GitHub
				</a>
			</div>
			<p class="text-base-content/60 text-sm">
				Evaluating? Run it with Docker and the sample data — see
				<a
					class="link"
					href="https://github.com/Sopandgo/sopandgo/blob/main/docs/ops/evaluating.md"
					rel="noreferrer"
					target="_blank"
					>Evaluating sopandgo</a
				>. Joining your lab’s instance? Ask your admin for an invite, then sign in here.
			</p>
		</div>
		<div class="card bg-base-200 max-w-md shadow-inner">
			<div class="card-body gap-4 text-left">
				<h2 class="card-title text-lg">At a glance</h2>
				<ul class="space-y-3 text-sm">
					<li class="list-row">
						<span class="badge badge-primary badge-sm shrink-0">SOPs</span>
						<span>Versioned Markdown, immutable history, search and organization.</span>
					</li>
					<li class="list-row">
						<span class="badge badge-secondary badge-sm shrink-0">Proof</span>
						<span>Acknowledgments tied to exact published versions — not moving targets.</span>
					</li>
					<li class="list-row">
						<span class="badge badge-neutral badge-sm shrink-0">Host</span>
						<span>One stack you run yourself; content stays on disk in readable files.</span>
					</li>
				</ul>
			</div>
		</div>
	</div>
</section>

<style>
	.list-row {
		display: flex;
		gap: 0.75rem;
		align-items: flex-start;
	}
</style>
