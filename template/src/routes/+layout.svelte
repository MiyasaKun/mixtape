<script lang="ts">
	import './layout.css';
	import { onMount } from 'svelte';
	import { page } from '$app/state';
	import { goto } from '$app/navigation';
	import { m } from '../paraglide/messages.js';
	import favicon from '$lib/assets/favicon.svg';
	import { user, sessionLoading, isAuthenticated, loadSession } from '$lib/stores/auth.js';
	import { initTheme } from '$lib/stores/theme.js';
	import UserMenu from '$lib/components/UserMenu.svelte';

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

	function isActive(href: string) {
		const path = page.url.pathname;
		return href === '/' ? path === '/' : path.startsWith(href);
	}
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
	<div class="drawer min-h-screen bg-base-200 md:drawer-open">
		<input id="app-drawer" type="checkbox" class="drawer-toggle" />

		<div class="drawer-content flex min-h-screen flex-col">
			<header class="flex items-center gap-3 border-b border-base-300 bg-base-100 px-4 py-3 md:hidden">
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
				<span class="text-lg">🎵 {m.app_title()}</span>
			</header>

			<main id="main-content" class="flex-1 px-4 py-8 md:px-8">
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
				class="flex h-full w-72 flex-col bg-[var(--neu-bg)] text-[var(--neu-text)]"
			>
				<div class="px-6 py-6 text-xl">🎵 {m.app_title()}</div>

				<nav aria-label={m.nav_navigation()} class="flex-1 overflow-y-auto px-3">
					<p class="px-3 pb-2 text-xs tracking-wide opacity-60">{m.nav_navigation()}</p>
					<ul class="space-y-1">
						<li>
							<a
								href="/"
								aria-current={isActive('/') ? 'page' : undefined}
								class="block rounded-xl px-3 py-2 text-sm transition-colors touch-manipulation focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-[var(--neu-accent)] {isActive(
									'/'
								)
									? 'bg-[var(--neu-accent)] text-white shadow-[inset_2px_2px_6px_rgba(0,0,0,0.25)]'
									: 'hover:bg-[var(--neu-light)]/60 dark:hover:bg-[var(--neu-light)]/20'}"
							>
								{m.nav_home()}
							</a>
						</li>
						<li>
							<a
								href="/dashboard"
								aria-current={isActive('/dashboard') ? 'page' : undefined}
								class="block rounded-xl px-3 py-2 text-sm transition-colors touch-manipulation focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-[var(--neu-accent)] {isActive(
									'/dashboard'
								)
									? 'bg-[var(--neu-accent)] text-white shadow-[inset_2px_2px_6px_rgba(0,0,0,0.25)]'
									: 'hover:bg-[var(--neu-light)]/60 dark:hover:bg-[var(--neu-light)]/20'}"
							>
								{m.dashboard_title()}
							</a>
						</li>
						<li>
							<a
								href="/playlists"
								aria-current={isActive('/playlists') ? 'page' : undefined}
								class="block rounded-xl px-3 py-2 text-sm transition-colors touch-manipulation focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-[var(--neu-accent)] {isActive(
									'/playlists'
								)
									? 'bg-[var(--neu-accent)] text-white shadow-[inset_2px_2px_6px_rgba(0,0,0,0.25)]'
									: 'hover:bg-[var(--neu-light)]/60 dark:hover:bg-[var(--neu-light)]/20'}"
							>
								{m.nav_playlists()}
							</a>
						</li>
						<li>
							<a
								href="/settings"
								aria-current={isActive('/settings') ? 'page' : undefined}
								class="block rounded-xl px-3 py-2 text-sm transition-colors touch-manipulation focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-[var(--neu-accent)] {isActive(
									'/settings'
								)
									? 'bg-[var(--neu-accent)] text-white shadow-[inset_2px_2px_6px_rgba(0,0,0,0.25)]'
									: 'hover:bg-[var(--neu-light)]/60 dark:hover:bg-[var(--neu-light)]/20'}"
							>
								{m.settings_title()}
							</a>
						</li>
					</ul>
				</nav>

				<div class="border-t border-[var(--neu-dark)]/30 p-3">
					<p class="px-3 pb-2 text-xs tracking-wide opacity-60">{m.nav_account()}</p>
					<UserMenu user={$user} />
				</div>
			</aside>
		</div>
	</div>
{:else}
	<main id="main-content" class="min-h-screen bg-base-200 px-4 py-8">
		{#if $sessionLoading}
			<p class="text-center text-base-content/70" aria-live="polite">{m.loading()}</p>
		{:else}
			{@render children()}
		{/if}
	</main>
{/if}
