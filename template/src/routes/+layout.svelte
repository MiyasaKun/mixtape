<script lang="ts">
	import './layout.css';
	import { onMount } from 'svelte';
	import { page } from '$app/state';
	import { goto } from '$app/navigation';
	import { m } from '../paraglide/messages.js';
	import favicon from '#lib/assets/favicon.svg';
	import { user, sessionLoading, isAuthenticated, loadSession } from '#lib/stores/auth.js';
	import { initTheme } from '#lib/stores/theme.js';
	import UserMenu from '#lib/components/UserMenu.svelte';
	import NavLink from '#lib/components/NavLink.svelte';

	let { children } = $props();

	const isLoginRoute = $derived(page.url.pathname === '/login');
	const showChrome = $derived($isAuthenticated && !isLoginRoute);

	onMount(() => {
		initTheme();
		loadSession();
	});

	// Every route except /login requires a session.
	$effect(() => {
		if ($sessionLoading) return;
		if (!$isAuthenticated && !isLoginRoute) {
			goto('/login');
		} else if ($isAuthenticated && isLoginRoute) {
			goto('/dashboard');
		}
	});

	// Tape-roll logo mark: a single badge, not an emoji.
	const logo = 'M9.5 4.5a3 3 0 1 1 3 3v7.9a3.6 3.6 0 1 1-3.6-3.6 3.65 3.65 0 0 1 1.6.37V6.6a3 3 0 0 1-1-2.1Z';
</script>

<svelte:head>
	<link rel="icon" href={favicon} />
	<meta name="theme-color" content="#e0e5ec" />
</svelte:head>

<a
	href="#main-content"
	class="sr-only focus:not-sr-only focus:absolute focus:z-50 focus:top-2 focus:left-2 focus:rounded focus:bg-base-100 focus:px-4 focus:py-2 focus:shadow"
>
	{m.skip_to_content()}
</a>

{#if showChrome}
	<div class="drawer min-h-svh bg-base-200 md:drawer-open">
		<input id="app-drawer" type="checkbox" class="drawer-toggle" />

		<div class="drawer-content flex min-h-svh flex-col">
			<header class="flex items-center gap-3 border-b border-black/[0.06] bg-base-200 px-4 py-3 md:hidden">
				<label for="app-drawer" class="btn btn-ghost btn-square touch-manipulation" aria-label={m.open_navigation()}>
					<svg
						xmlns="http://www.w3.org/2000/svg"
						fill="none"
						viewBox="0 0 24 24"
						stroke-width="1.5"
						stroke="currentColor"
						class="h-6 w-6"
						aria-hidden="true"
					>
						<path stroke-linecap="round" stroke-linejoin="round" d="M3.75 6.75h16.5M3.75 12h16.5M3.75 17.25h16.5" />
					</svg>
				</label>
				<span class="brandmark text-lg">{m.app_title()}</span>
			</header>

			<main id="main-content" class="flex-1 px-4 pb-10 pt-6 md:px-8 md:pt-10 pb-[env(safe-area-inset-bottom)]">
				{#if $sessionLoading}
					<p class="text-center text-base-content/70" aria-live="polite">{m.loading()}</p>
				{:else}
					{@render children()}
				{/if}
			</main>
		</div>

		<div class="drawer-side z-40">
			<label for="app-drawer" class="drawer-overlay" aria-label={m.close_navigation()}></label>

			<aside
				class="flex h-full w-72 flex-col bg-[var(--neu-bg)] pb-[env(safe-area-inset-bottom)] text-[var(--neu-text)]"
			>
				<a href="/" class="flex items-center gap-2.5 px-6 py-6 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-[var(--neu-accent)] rounded-xl">
					<svg viewBox="0 0 24 24" class="h-6 w-6 text-[var(--neu-accent)]" fill="currentColor" aria-hidden="true">
						<path d={logo} />
					</svg>
					<span class="brandmark text-lg">{m.app_title()}</span>
				</a>

				<nav aria-label={m.nav_navigation()} class="flex-1 overflow-y-auto px-3">
					<ul class="space-y-0.5">
						<li><NavLink href="/" label={m.nav_home()} /></li>
						<li><NavLink href="/dashboard" label={m.dashboard_title()} /></li>
						<li><NavLink href="/playlists" label={m.nav_playlists()} /></li>
						<li><NavLink href="/settings" label={m.settings_title()} /></li>
					</ul>
				</nav>

				<div class="border-t border-black/[0.08] dark:border-white/10 p-3">
					<UserMenu user={$user} />
				</div>
			</aside>
		</div>
	</div>
{:else}
	<main id="main-content" class="min-h-svh bg-base-200 px-4 py-8">
		{#if $sessionLoading}
			<p class="text-center text-base-content/70" aria-live="polite">{m.loading()}</p>
		{:else}
			{@render children()}
		{/if}
	</main>
{/if}
