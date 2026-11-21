<script lang="ts">
	import { m } from '../paraglide/messages.js';
	import { getLocale } from '../paraglide/runtime.js';
	import { user } from '#lib/stores/auth.js';
	import { getSpotifyAuthUrl } from '#lib/api/client.js';

	let busy = $state(false);
	let error = $state('');

	const connected = $derived($user?.spotifyConnected ?? false);
	const greeting = $derived((connected ? m.home_browse_playlists() : m.home_connect_to_continue()));

	async function connectSpotify() {
		busy = true;
		error = '';
		try {
			const { url } = await getSpotifyAuthUrl();
			window.location.assign(url);
		} catch {
			error = m.spotify_connect_error();
			busy = false;
		}
	}
</script>

<svelte:head><title>{m.app_title()}</title></svelte:head>

<div class="mx-auto max-w-2xl space-y-10 rise">
	<header class="space-y-3 pt-4 md:pt-8">
		<h1 class="text-4xl md:text-5xl">{m.app_title()}</h1>
		<p class="text-lg text-base-content/80 text-pretty">{m.home_intro()}</p>
	</header>

	<section class="neu p-6 md:p-8 space-y-4" aria-labelledby="status-heading">
		<h2 id="status-heading" class="card-title text-xl">{m.spotify_section_title()}</h2>

		<p class="flex items-center gap-2 font-medium" aria-live="polite">
			<span
				class="h-2.5 w-2.5 shrink-0 rounded-full {connected ? 'bg-green-600 dark:bg-green-400' : 'bg-error'}"
				aria-hidden="true"
			></span>
			{connected ? m.spotify_connected() : m.spotify_not_connected()}
		</p>

		<div class="flex flex-wrap items-center gap-4">
			{#if connected}
				<a href="/playlists" class="btn btn-primary">{greeting}</a>
				<a href="/settings" class="btn btn-ghost">{m.dashboard_action_settings()}</a>
			{:else}
				<button class="btn btn-primary" onclick={connectSpotify} disabled={busy} aria-busy={busy}>
					{m.spotify_connect()}
				</button>
			{/if}
		</div>

		{#if error}
			<p class="text-error" role="alert" aria-live="polite">{error}</p>
		{/if}
	</section>

	{#if $user}
		<footer class="text-sm text-base-content/60 flex items-center gap-2">
			<img
				src={$user.picture}
				alt=""
				width="20"
				height="20"
				referrerpolicy="no-referrer"
				class="h-5 w-5 rounded-full object-cover"
				onerror={(e) => (e.currentTarget as HTMLElement).style.display = 'none'}
			/>
			{m.home_signed_in_as({ name: $user.name })}
		</footer>
	{/if}
</div>
