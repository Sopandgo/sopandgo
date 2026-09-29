<script>
	import LandingHero from '$lib/components/landing/LandingHero.svelte';
	import LandingMarketing from '$lib/components/landing/LandingMarketing.svelte';
	import LandingProductScreens from '$lib/components/landing/LandingProductScreens.svelte';
	import LandingScopeNote from '$lib/components/landing/LandingScopeNote.svelte';
</script>

<div class="landing">
	<LandingScopeNote />
	<LandingHero />
	<LandingMarketing />
	<LandingProductScreens />
</div>

<style>
	.landing {
		display: flex;
		flex-direction: column;
		gap: 3rem;
		width: 100%;
	}
</style>
