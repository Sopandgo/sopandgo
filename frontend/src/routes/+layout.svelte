<script lang="ts">
	import { page } from '$app/state';
	import './layout.css';
	import favicon from '$lib/assets/favicon.svg';
	import AppSidebar from '$lib/components/AppSidebar.svelte';
	import Navbar from '$lib/components/Navbar.svelte';
	import Footer from '$lib/components/Footer.svelte';
	import * as m from '$lib/paraglide/messages.js';

	let { data, children } = $props();

	let guestRoute = $derived(
		page.url.pathname === '/login' || page.url.pathname.startsWith('/reset-password')
	);
	let showRail = $derived(!!data.user && !data.user.must_change_password);
</script>

<svelte:head>
	<link rel="icon" href={favicon} />
</svelte:head>

{#if data.user && !data.user.must_change_password}
	<div class="app-shell drawer lg:drawer-open">
		<input id="app-drawer" type="checkbox" class="drawer-toggle inline" />
		<div class="drawer-content app-layout bg-base-200">
			{@render frame()}
		</div>
		<div class="drawer-side is-drawer-close:overflow-visible">
			<label for="app-drawer" aria-label={m.nav_close_menu()} class="drawer-overlay"></label>
			<AppSidebar user={data.user} />
		</div>
	</div>
{:else}
	<div class="app-layout bg-base-200">
		{@render frame()}
	</div>
{/if}

{#snippet frame()}
	<Navbar user={data.user} rail={showRail} />

	<main
		class="app-content flex-grow {guestRoute ? 'flex min-h-0 flex-1 flex-col px-4 py-0 sm:px-6 lg:px-8' : 'mx-auto w-full max-w-7xl px-4 py-8 sm:px-6 lg:px-8'}"
	>
		{@render children()}
	</main>

	{#if !showRail}
		<Footer />
	{/if}
{/snippet}

<style>
	.app-shell :global(.drawer-side) {
		z-index: 40;
	}

	.app-shell :global(#app-drawer:not(:checked) ~ .drawer-content .rail-when-open),
	.app-shell :global(#app-drawer:checked ~ .drawer-content .rail-when-closed) {
		display: none;
	}
</style>
