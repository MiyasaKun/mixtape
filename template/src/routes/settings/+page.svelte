<script lang="ts">
	import { m } from '../../paraglide/messages.js';
	import { user, setSpotifyConnected, setSpotifyAutoplay } from '#lib/stores/auth.js';
	import { getSpotifyAuthUrl, disconnectSpotify, updateSpotifyAutoplay } from '#lib/api/client.js';

	let busy = $state(false);
	let error = $state('');
	let confirmOpen = $state(false);
	let dialogEl: HTMLDialogElement | undefined;

	let autoplayBusy = $state(false);
	let autoplayError = $state('');

	const connected = $derived($user?.spotifyConnected ?? false);
	const autoplay = $derived($user?.spotifyAutoplay ?? false);

	$effect(() => {
		if (!dialogEl) return;
		if (confirmOpen) {
			dialogEl.showModal();
		} else {
			dialogEl.close();
		}
	});

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

	async function confirmDisconnect() {
		confirmOpen = false;
		busy = true;
		error = '';
		try {
			await disconnectSpotify();
			setSpotifyConnected(false);
		} catch {
			error = m.spotify_disconnect_error();
		} finally {
			busy = false;
		}
	}

	async function toggleAutoplay(event: Event) {
		if (autoplayBusy) return;
		const input = event.currentTarget as HTMLInputElement;
		autoplayBusy = true;
		autoplayError = '';
		try {
			const updated = await updateSpotifyAutoplay(!autoplay);
			setSpotifyAutoplay(updated.spotifyAutoplay);
		} catch {
			autoplayError = m.spotify_autoplay_error();
			// The checkbox flipped natively on click; snap it back to the
			// persisted state so it never lies about what was saved.
			input.checked = autoplay;
		} finally {
			autoplayBusy = false;
		}
	}
</script>

<svelte:head><title>{m.settings_title()} · {m.app_title()}</title></svelte:head>

<div class="mx-auto max-w-2xl space-y-6">
	<h1 class="text-3xl">{m.settings_title()}</h1>

	<section class="card bg-base-100 shadow-xl" aria-labelledby="spotify-heading">
		<div class="card-body">
			<h2 id="spotify-heading" class="card-title">{m.spotify_section_title()}</h2>
			<p class="text-base-content/70 text-pretty">{m.spotify_description()}</p>

			<p class="mt-1 font-medium" aria-live="polite">
				{connected ? m.spotify_connected() : m.spotify_not_connected()}
			</p>

			<div class="card-actions mt-4">
				{#if connected}
					<button class="btn btn-error btn-outline" onclick={() => (confirmOpen = true)} disabled={busy}>
						{m.spotify_disconnect()}
					</button>
				{:else}
					<button class="btn btn-primary" onclick={connectSpotify} disabled={busy} aria-busy={busy}>
						{m.spotify_connect()}
					</button>
				{/if}
			</div>

			{#if error}
				<p class="text-error mt-3" role="alert" aria-live="polite">{error}</p>
			{/if}
		</div>
	</section>

	<section class="card bg-base-100 shadow-xl" aria-labelledby="playback-heading">
		<div class="card-body">
			<h2 id="playback-heading" class="card-title">{m.playback_section_title()}</h2>
			<label class="mt-1 flex cursor-pointer items-center gap-3" for="spotify-autoplay">
				<input
					id="spotify-autoplay"
					type="checkbox"
					class="toggle toggle-primary"
					checked={autoplay}
					onchange={toggleAutoplay}
					disabled={autoplayBusy}
					aria-busy={autoplayBusy}
				/>
				<span class="font-medium">{m.spotify_autoplay_label()}</span>
			</label>
			<p class="text-base-content/70 text-pretty">{m.spotify_autoplay_description()}</p>

			{#if autoplayError}
				<p class="text-error mt-3" role="alert" aria-live="polite">{autoplayError}</p>
			{/if}
		</div>
	</section>
</div>

<dialog bind:this={dialogEl} class="modal" aria-labelledby="spotify-disconnect-title" onclose={() => (confirmOpen = false)}>
	<div class="modal-box">
		<h2 id="spotify-disconnect-title" class="text-lg font-bold">{m.spotify_disconnect_confirm_title()}</h2>
		<p class="py-4 text-base-content/70">{m.spotify_disconnect_confirm_body()}</p>
		<div class="modal-action">
			<button class="btn" onclick={() => (confirmOpen = false)}>{m.cancel()}</button>
			<button class="btn btn-error" onclick={confirmDisconnect} disabled={busy}>
				{m.confirm_disconnect()}
			</button>
		</div>
	</div>
	<form method="dialog" class="modal-backdrop">
		<button aria-label={m.cancel()}>{m.cancel()}</button>
	</form>
</dialog>
