<script lang="ts">
	import './layout.css';
	import { m } from '../paraglide/messages.js';
	import { setLocale, getLocale } from '../paraglide/runtime.js';
	import favicon from '$lib/assets/favicon.svg';

	let { children } = $props();
	let currentLocale = $derived(getLocale());
</script>

<svelte:head><link rel="icon" href={favicon} /></svelte:head>

<div class="min-h-screen bg-base-200">
	<div class="navbar bg-base-100 shadow-lg">
		<div class="flex-1">
			<a href="/" class="btn btn-ghost text-xl">🎵 {m.app_title()}</a>
		</div>
		<div class="flex-none">
			<div class="dropdown dropdown-end">
				<div tabindex="0" role="button" class="btn btn-ghost">
					🌐 {currentLocale.toUpperCase()}
					<svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke-width="1.5" stroke="currentColor" class="w-4 h-4">
						<path stroke-linecap="round" stroke-linejoin="round" d="m19.5 8.25-7.5 7.5-7.5-7.5" />
					</svg>
				</div>
				<ul tabindex="0" class="dropdown-content menu bg-base-100 rounded-box z-[1] w-32 p-2 shadow">
					<li><button onclick={() => setLocale('en')}>English</button></li>
					<li><button onclick={() => setLocale('de')}>Deutsch</button></li>
				</ul>
			</div>
		</div>
	</div>

	<main class="container mx-auto px-4 py-8">
		{@render children()}
	</main>
</div>
