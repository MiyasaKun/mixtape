<script lang="ts">
	import { m } from '../../paraglide/messages.js';
	import { user } from '#lib/stores/auth.js';

	const connected = $derived($user?.spotifyConnected ?? false);
</script>

<svelte:head><title>{m.dashboard_title()} · {m.app_title()}</title></svelte:head>

<div class="mx-auto max-w-2xl space-y-6 rise">
	<header>
		<h1 class="text-3xl">{m.dashboard_title()}</h1>
		<p class="text-base-content/80 text-pretty">{m.home_intro()}</p>
	</header>

	<section class="neu p-6 md:p-8 space-y-5" aria-labelledby="status-heading">
		<h2 id="status-heading" class="card-title text-xl">{m.dashboard_status_heading()}</h2>

		<div class="space-y-1">
			<p class="flex items-center gap-2 font-medium">
				<span
					class="h-2.5 w-2.5 shrink-0 rounded-full {connected ? 'bg-green-600 dark:bg-green-400' : 'bg-error'}"
					aria-hidden="true"
				></span>
				{connected ? m.spotify_connected() : m.spotify_not_connected()}
			</p>
			<p class="text-sm text-base-content/70 text-pretty">{connected ? m.dashboard_ready() : m.spotify_description()}</p>
		</div>

		<div class="flex flex-wrap items-center gap-4">
			{#if connected}
				<a href="/playlists" class="btn btn-primary">{m.dashboard_action_playlists()}</a>
				<a href="/settings" class="btn btn-ghost">{m.dashboard_action_settings()}</a>
			{:else}
				<a href="/settings" class="btn btn-primary">{m.spotify_connect()}</a>
			{/if}
		</div>
	</section>
</div>
