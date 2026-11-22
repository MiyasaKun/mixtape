<script lang="ts">
	import { page } from '$app/state';
	import { m } from '../../../paraglide/messages.js';
	import { user } from '#lib/stores/auth.js';
	import { getLikedTracks, getPlaylistTracks, getActiveVoiceGuilds, queueSpotifyTrack, ApiError } from '#lib/api/client.js';
	import type { ActiveGuild } from '#lib/api/client.js';
	import { cachedPlaylist, createPagedResource } from '#lib/stores/playlists.js';
	import CoverArt from '#lib/components/CoverArt.svelte';
	import { spotifyUrl } from '#lib/utils/spotify.js';

	const id = $derived(page.params.id ?? '');
	const isLiked = $derived(id === 'liked');
	const connected = $derived($user?.spotifyConnected ?? false);
	const meta = $derived(isLiked ? undefined : cachedPlaylist(id));

	const resource = createPagedResource((offset) =>
		isLiked ? getLikedTracks(offset, 100) : getPlaylistTracks(id, offset, 100)
	);

	let selectedId: string | null = $state(null);
	const selected = $derived($resource.items.find((track) => track.id === selectedId) ?? null);

	let sentinel = $state<HTMLDivElement | undefined>(undefined);

	// (Re)load when the playlist changes or the connection becomes available.
	$effect(() => {
		id;
		if (!connected) return;
		selectedId = null;
		resource.reset();
		resource.load();
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

	// Guild picker: the song goes to a guild the bot is currently playing in.
	type PickerStatus = 'idle' | 'loading' | 'ready' | 'error';
	let guildPickerOpen = $state(false);
	let guildDialogEl: HTMLDialogElement | undefined;
	let guilds = $state<ActiveGuild[]>([]);
	let guildsStatus = $state<PickerStatus>('idle');
	let guildId = $state('');
	let queueBusy = $state(false);
	let queueMessage = $state('');
	let queueError = $state('');

	// A stale notice from a previous track must never look like it belongs to
	// the newly selected one.
	$effect(() => {
		selectedId;
		queueMessage = '';
	});

	$effect(() => {
		if (!guildDialogEl) return;
		if (guildPickerOpen) {
			guildDialogEl.showModal();
		} else {
			guildDialogEl.close();
		}
	});

	async function openGuildPicker() {
		queueError = '';
		guildId = '';
		guildsStatus = 'loading';
		guildPickerOpen = true;
		try {
			guilds = (await getActiveVoiceGuilds()).guilds;
			guildId = guilds[0]?.id ?? '';
			guildsStatus = 'ready';
		} catch {
			guilds = [];
			guildsStatus = 'error';
		}
	}

	async function sendToDiscord() {
		const track = selected;
		if (!track || !guildId || queueBusy) return;

		queueBusy = true;
		queueError = '';
		try {
			const result = await queueSpotifyTrack(guildId, track.url);
			queueMessage = result.started
				? m.playlists_queue_started({ title: result.title })
				: m.playlists_queue_added({ position: result.position });
			guildPickerOpen = false;
		} catch (error) {
			queueError =
				error instanceof ApiError && error.status === 409
					? m.playlists_bot_not_connected()
					: m.playlists_queue_failed();
		} finally {
			queueBusy = false;
		}
	}

	const heading = $derived(
		isLiked ? m.playlists_liked() : (meta?.name ?? m.playlists_detail_title())
	);

	function formatDuration(ms: number): string {
		const total = Math.round(ms / 1000);
		const minutes = Math.floor(total / 60);
		const seconds = total % 60;
		return `${minutes}:${String(seconds).padStart(2, '0')}`;
	}
</script>

<svelte:head><title>{heading} · {m.app_title()}</title></svelte:head>

<div class="mx-auto max-w-4xl space-y-6">
	<header class="flex flex-col gap-5 sm:flex-row sm:items-end">
		<CoverArt src={meta?.imageUrl ?? $resource.items[0]?.imageUrl} name={heading} size="lg" />
		<div class="min-w-0">
			<a
				href="/playlists"
				class="text-sm text-[var(--neu-accent)] underline-offset-4 hover:underline focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-[var(--neu-accent)]"
				>← {m.playlists_back()}</a
			>
			<h1 class="mt-1 break-words text-3xl">{heading}</h1>
			{#if isLiked}
				<p class="text-base-content/70">{m.playlists_liked_description()}</p>
			{:else if meta}
				<p class="text-base-content/70">
					{meta.owner}{#if meta.trackCount > 0} · {m.playlists_track_count({ count: meta.trackCount })}{/if}
				</p>
			{/if}
		</div>
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
		<p class="text-base-content/70" aria-live="polite">{m.loading()}</p>
	{:else if $resource.items.length === 0}
		<section class="neu p-6">
			<p class="text-base-content/70 text-pretty">{m.playlists_empty_tracks()}</p>
		</section>
	{:else}
		<ul class="space-y-2">
			{#each $resource.items as track, index (track.id)}
				<li>
					<button
						type="button"
						aria-pressed={selectedId === track.id}
						onclick={() => (selectedId = selectedId === track.id ? null : track.id)}
						class="neu-sm flex w-full items-center gap-3 p-2.5 text-left transition-colors touch-manipulation focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-[var(--neu-accent)] {selectedId ===
						track.id
							? 'ring-2 ring-[var(--neu-accent)]'
							: 'hover:bg-[var(--neu-light)]/60 dark:hover:bg-[var(--neu-light)]/20'}"
					>
						<span class="w-6 shrink-0 text-right text-xs text-base-content/60" aria-hidden="true">{index + 1}</span>
						<CoverArt src={track.imageUrl} name={track.title} size="sm" />
						<span class="min-w-0 flex-1">
							<span class="block truncate text-sm font-semibold">{track.title}</span>
							<span class="block truncate text-xs text-base-content/70"
								>{track.artists.join(', ') || m.playlists_unknown_artist()}</span
							>
						</span>
						<span class="shrink-0 text-xs text-base-content/60">{formatDuration(track.durationMs)}</span>
					</button>
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

		{#if selected}
			<section class="neu sticky bottom-4 z-20 p-4" aria-live="polite" aria-label={m.playlists_selected()}>
				<div class="flex items-start gap-4">
					<CoverArt src={selected.imageUrl} name={selected.title} />
					<div class="min-w-0 flex-1">
						<p class="text-xs text-base-content/60">{m.playlists_selected()}</p>
						<p class="truncate text-lg font-semibold">{selected.title}</p>
						<p class="truncate text-sm text-base-content/70">
							{selected.artists.join(', ') || m.playlists_unknown_artist()}
						</p>
						{#if selected.album}
							<p class="truncate text-xs text-base-content/60">{selected.album}</p>
						{/if}
						<div class="mt-3 flex flex-wrap items-center gap-3">
							<button type="button" class="btn btn-primary" onclick={openGuildPicker}>
								{m.playlists_add_to_queue()}
							</button>
							{#if selected.url}
								<a
									href={spotifyUrl(selected.url, $user?.spotifyAutoplay ?? false)}
									target="_blank"
									rel="noopener noreferrer"
									class="link link-primary text-sm"
									>{m.playlists_open_spotify()}</a
								>
							{/if}
							<button type="button" class="btn btn-ghost btn-sm" onclick={() => (selectedId = null)}>
								{m.close()}
							</button>
						</div>
						{#if queueMessage}
							<p class="mt-2 text-sm" role="status">{queueMessage}</p>
						{/if}
					</div>
				</div>
			</section>
		{/if}
	{/if}
</div>

<dialog
	bind:this={guildDialogEl}
	class="modal"
	aria-labelledby="guild-picker-title"
	onclose={() => (guildPickerOpen = false)}
>
	<div class="modal-box">
		<h2 id="guild-picker-title" class="text-lg font-bold">{m.playlists_guild_picker_title()}</h2>
		<p class="py-2 text-base-content/70 text-pretty">{m.playlists_guild_picker_description()}</p>

		{#if guildsStatus === 'loading'}
			<p class="text-base-content/70" aria-live="polite">{m.loading()}</p>
		{:else if guildsStatus === 'error'}
			<p class="text-error" role="alert">{m.playlists_queue_failed()}</p>
		{:else if guilds.length === 0}
			<p class="text-base-content/70" role="alert">{m.playlists_no_active_voice()}</p>
		{:else}
			<fieldset class="mt-2 space-y-2">
				<legend class="sr-only">{m.playlists_guild_picker_legend()}</legend>
				{#each guilds as guild (guild.id)}
					<label
						class="neu-sm flex cursor-pointer items-center gap-3 p-3 transition-colors has-checked:ring-2 has-checked:ring-[var(--neu-accent)]"
					>
						<input type="radio" name="guild-picker" value={guild.id} bind:group={guildId} class="radio radio-primary" />
						<span class="min-w-0">
							<span class="block truncate font-semibold">{guild.name || guild.id}</span>
							{#if guild.name}
								<span class="block truncate text-xs text-base-content/60">{guild.id}</span>
							{/if}
						</span>
					</label>
				{/each}
			</fieldset>
		{/if}

		{#if queueError}
			<p class="text-error mt-3" role="alert">{queueError}</p>
		{/if}

		<div class="modal-action">
			<button type="button" class="btn" onclick={() => (guildPickerOpen = false)} disabled={queueBusy}>
				{m.cancel()}
			</button>
			<button
				type="button"
				class="btn btn-primary"
				onclick={sendToDiscord}
				disabled={queueBusy || guildsStatus !== 'ready' || !guildId}
				aria-busy={queueBusy}
			>
				{queueBusy ? m.playlists_queue_sending() : m.playlists_queue_confirm()}
			</button>
		</div>
	</div>
	<form method="dialog" class="modal-backdrop">
		<button aria-label={m.cancel()}>{m.cancel()}</button>
	</form>
</dialog>
