<script lang="ts">
	import { m } from '../../paraglide/messages.js';
	import { user } from '#lib/stores/auth.js';
	import { getPlaylists } from '#lib/api/client.js';
	import { createPagedResource, rememberPlaylists } from '#lib/stores/playlists.js';
	import CoverArt from '#lib/components/CoverArt.svelte';
import { trackCount } from '#lib/utils/plural.js';
import { getLocale } from '../../paraglide/runtime.js';

	const connected = $derived($user?.spotifyConnected ?? false);
	const resource = createPagedResource((offset) => getPlaylists(offset, 50));

	let sentinel = $state<HTMLDivElement | undefined>(undefined);

	// Keep the detail view's header available after client-side navigation.
	$effect(() => {
		rememberPlaylists($resource.items);
	});

	// (Re)load whenever the Spotify connection becomes available.
	$effect(() => {
		if (connected && $resource.status === 'idle') resource.load();
	});

	// Infinite scroll: auto-load while a reachable "Load more" button remains.
	$effect(() => {
		if (!connected || !sentinel || $resource.next === null) return;
		const observer = new IntersectionObserver(
			(entries) => {
				if (entries.some((entry) => entry.isIntersecting)) resource.loadMore();
			},
			{ rootMargin: '240px' }
		);
		observer.observe(sentinel);
		return () => observer.disconnect();
	});
</script>

<svelte:head><title>{m.playlists_title()} · {m.app_title()}</title></svelte:head>

<div class="mx-auto max-w-4xl space-y-6">
	<header>
		<h1 class="text-3xl">{m.playlists_title()}</h1>
		<p class="text-base-content/70 text-pretty">{m.playlists_description()}</p>
	</header>

	{#if !connected}
		<section class="neu p-6" aria-labelledby="connect-heading">
			<h2 id="connect-heading" class="text-lg font-semibold">{m.spotify_section_title()}</h2>
			<p class="mt-1 text-base-content/70 text-pretty">{m.playlists_connect_required()}</p>
			<a href="/settings" class="btn btn-primary mt-4">{m.spotify_connect()}</a>
		</section>
	{:else if $resource.status === 'error'}
		<section class="neu p-6" role="alert">
			<p class="text-pretty">
				{#if $resource.error === 'not_connected'}
					{m.playlists_connect_required()}
				{:else if $resource.error === 'reconnect_required'}
					{m.playlists_reconnect_required()}
				{:else if $resource.error === 'forbidden'}
					{m.playlists_forbidden()}
				{:else if $resource.error === 'not_found'}
					{m.playlists_not_found()}
				{:else if $resource.error === 'rate_limited'}
					{m.playlists_rate_limited()}
				{:else}
					{m.playlists_unavailable()}
				{/if}
			</p>
			<div class="mt-4">
				{#if $resource.error === 'not_connected' || $resource.error === 'reconnect_required'}
					<a href="/settings" class="btn btn-primary">{m.spotify_connect()}</a>
				{:else}
					<button type="button" class="btn" onclick={() => resource.load()}>{m.playlists_retry()}</button>
				{/if}
			</div>
		</section>
	{:else if $resource.status === 'loading'}
		<ul class="grid gap-4" aria-hidden="true">
			{#each Array(4) as _}
				<li class="neu flex items-center gap-4 p-4">
					<div class="neu-sm h-20 w-20 shrink-0 animate-pulse"></div>
					<div class="min-w-0 flex-1 space-y-2">
						<div class="h-5 w-1/2 animate-pulse rounded bg-black/10 dark:bg-white/10"></div>
						<div class="h-4 w-1/3 animate-pulse rounded bg-black/[0.06] dark:bg-white/[0.06]"></div>
					</div>
				</li>
			{/each}
		</ul>
		<p class="sr-only" aria-live="polite">{m.loading()}</p>
	{:else if $resource.items.length === 0}
		<section class="neu p-6">
			<p class="text-base-content/70 text-pretty">{m.playlists_empty()}</p>
		</section>
	{:else}
		<ul class="grid grid-cols-1 gap-4 sm:grid-cols-2">
			<li class="min-w-0">
				<a
					href="/playlists/liked"
					class="neu flex min-w-0 items-center gap-4 p-4 transition-transform touch-manipulation hover:-translate-y-0.5 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-[var(--neu-accent)] motion-reduce:transform-none"
				>
					<span
						class="neu-sm flex h-20 w-20 shrink-0 items-center justify-center text-[var(--neu-accent)]"
						aria-hidden="true"
					>
						<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="currentColor" class="h-8 w-8">
							<path
								d="M11.645 20.91l-.007-.003-.022-.012a15.247 15.247 0 01-.383-.218 25.18 25.18 0 01-4.244-3.17C4.688 15.36 2.25 12.174 2.25 8.25 2.25 5.322 4.714 3 7.688 3A5.5 5.5 0 0112 5.052 5.5 5.5 0 0116.313 3c2.973 0 5.437 2.322 5.437 5.25 0 3.925-2.438 7.111-4.739 9.256a25.175 25.175 0 01-4.244 3.17 15.247 15.247 0 01-.383.219l-.022.012-.007.003-.003.001a.752.752 0 01-.704 0l-.003-.001z"
							/>
						</svg>
					</span>
					<span class="min-w-0">
						<span class="block truncate font-semibold">{m.playlists_liked()}</span>
						<span class="block truncate text-sm text-base-content/70">{m.playlists_liked_description()}</span>
					</span>
				</a>
			</li>
			{#each $resource.items as playlist (playlist.id)}
				<li class="min-w-0">
					{#if playlist.browseable}
						<a
							href={`/playlists/${playlist.id}`}
							class="neu flex min-w-0 items-center gap-4 p-4 transition-transform touch-manipulation hover:-translate-y-0.5 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-[var(--neu-accent)] motion-reduce:transform-none"
						>
							<CoverArt src={playlist.imageUrl} name={playlist.name} />
							<span class="min-w-0">
								<span class="block truncate font-semibold text-wrap-anywhere" title={playlist.name}>{playlist.name}</span>
								<span class="block truncate text-sm text-base-content/70" title={playlist.owner}>{playlist.owner}</span>
								{#if playlist.trackCount > 0}
									<span class="block text-xs text-base-content/60">{trackCount(playlist.trackCount, getLocale())}</span>
								{/if}
							</span>
						</a>
					{:else}
						<div class="neu flex items-center gap-4 p-4 opacity-70">
							<CoverArt src={playlist.imageUrl} name={playlist.name} />
							<span class="min-w-0">
								<span class="block truncate font-semibold text-wrap-anywhere" title={playlist.name}>{playlist.name}</span>
								<span class="block truncate text-sm text-base-content/70">{playlist.owner}</span>
								<span class="block text-xs text-base-content/60">{m.playlists_not_browseable()}</span>
							</span>
						</div>
					{/if}
				</li>
			{/each}
		</ul>

		<div class="flex flex-col items-center gap-3">
			<p class="text-sm text-base-content/60" aria-live="polite">
				{m.playlists_loaded_status({ count: $resource.items.length })}
			</p>
			{#if $resource.next !== null}
				<div bind:this={sentinel} class="h-1 w-full" aria-hidden="true"></div>
				<button
					type="button"
					class="btn touch-manipulation"
					onclick={() => resource.loadMore()}
					disabled={$resource.status === 'loading-more'}
					aria-busy={$resource.status === 'loading-more'}
				>
					{$resource.status === 'loading-more' ? m.playlists_loading_more() : m.playlists_load_more()}
				</button>
			{/if}
		</div>
	{/if}
</div>
