<script>
	import { page } from '$app/state';
	import './layout.css';
	import favicon from '$lib/assets/favicon.svg';
	import Navbar from '$lib/components/Navbar.svelte';
	import Footer from '$lib/components/Footer.svelte';

	let { data, children } = $props();

	let guestRoute = $derived(
		page.url.pathname === '/login' || page.url.pathname.startsWith('/reset-password')
	);
</script>

<svelte:head>
	<link rel="icon" href={favicon} />
</svelte:head>

<div class="app-layout bg-base-300">
	<Navbar user={data.user}/>

	<main
		class="app-content flex-grow {guestRoute
			? 'flex min-h-0 flex-1 flex-col px-4 py-0 sm:px-6 lg:px-8'
			: 'mx-auto w-full max-w-7xl px-4 py-8 sm:px-6 lg:px-8'}"
	>
		{@render children()}
	</main>

	<Footer />
</div>
