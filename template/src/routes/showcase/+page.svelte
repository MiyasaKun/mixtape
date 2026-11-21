<script lang="ts">
	let dark = $state(false);
	let toast = $state('');
	let email = $state('');
	let password = $state('');
	let emailError = $state('');
	let isSubmitting = $state(false);
	let volume = $state(70);
	let query = $state('');
	let activeTab = $state<'Queue' | 'History' | 'Saved'>('Queue');
	let confirmDelete = $state(false);

	function toggleDark() {
		dark = !dark;
		document.documentElement.classList.toggle('dark', dark);
		document.documentElement.style.colorScheme = dark ? 'dark' : 'light';
	}

	function showToast(message: string) {
		toast = message;
		setTimeout(() => (toast = ''), 3200);
	}

	async function handleLogin(event: SubmitEvent) {
		event.preventDefault();
		emailError = '';
		if (!email.includes('@')) {
			emailError = 'Enter a valid email address, like Ada@example.com…';
			document.getElementById('login-email')?.focus();
			return;
		}
		if (!password) {
			document.getElementById('login-password')?.focus();
			return;
		}
		isSubmitting = true;
		await new Promise((r) => setTimeout(r, 900));
		isSubmitting = false;
		showToast('Signed in. Welcome back.');
	}
</script>

<svelte:head>
	<title>Element Gallery — Mixtape Bot</title>
	<meta name="theme-color" content="#e0e5ec" />
</svelte:head>

<a
	href="#gallery-main"
	class="sr-only focus:not-sr-only focus:absolute focus:z-50 focus:m-4 focus:rounded-lg focus:bg-[var(--neu-accent)] focus:px-4 focus:py-2 focus:text-white focus-visible:ring-2 focus-visible:ring-offset-2"
>
	Skip to Gallery Content
</a>

<div id="gallery-main" class="mx-auto w-full max-w-5xl scroll-mt-24 space-y-12 pb-20">
	<!-- Header -->
	<header class="neu-lg p-8 md:p-10">
		<p class="text-sm text-[var(--neu-text)] opacity-70">Mixtape Bot &middot; Design Reference</p>
		<h1 class="mt-2 text-4xl text-balance">Element Gallery</h1>
		<p class="mt-3 max-w-[62ch] text-base leading-relaxed opacity-80">
			Every control the bot uses on stage: playback, queue, and sign in. Flip the theme to check
			contrast in light and dark.
		</p>
		<div class="mt-6 flex flex-wrap items-center gap-3">
			<button
				type="button"
				onclick={toggleDark}
				aria-pressed={dark}
				class="neu-sm touch-manipulation px-5 py-2.5 font-semibold transition-colors duration-150 hover:brightness-105 focus-visible:ring-2 focus-visible:ring-[var(--neu-accent)] focus-visible:ring-offset-2 focus-visible:ring-offset-[var(--neu-bg)] active:brightness-95"
			>
				{dark ? 'Use Light Theme' : 'Use Dark Theme'}
			</button>
			<a
				href="/dashboard"
				class="neu-sm touch-manipulation px-5 py-2.5 underline-offset-4 transition-colors duration-150 hover:underline focus-visible:ring-2 focus-visible:ring-[var(--neu-accent)] focus-visible:ring-offset-2"
			>
				Open Dashboard
			</a>
			<span class="text-sm tabular-nums opacity-70" aria-live="polite">
				{volume}&#37; monitor volume
			</span>
		</div>
	</header>

	<!-- Buttons -->
	<section aria-labelledby="h-buttons" class="neu scroll-mt-24 p-6 md:p-8">
		<h2 id="h-buttons" class="text-2xl text-balance">Buttons</h2>
		<p class="mt-1 text-sm opacity-70">Raised for primary actions, inset for pressed state.</p>
		<div class="mt-5 flex flex-wrap items-center gap-3">
			<button
				type="button"
				onclick={() => showToast('Playing “Midnight Loop”.')}
				class="touch-manipulation rounded-xl bg-[var(--neu-accent)] px-5 py-2.5 font-semibold text-white shadow-[4px_4px_8px_var(--neu-dark),-4px_-4px_8px_var(--neu-light)] transition-colors duration-150 hover:brightness-110 focus-visible:ring-2 focus-visible:ring-[var(--neu-accent)] focus-visible:ring-offset-2 active:brightness-95"
			>
				Play Now
			</button>
			<button
				type="button"
				onclick={() => showToast('Paused.')}
				class="neu-sm touch-manipulation px-5 py-2.5 font-semibold transition-colors duration-150 hover:brightness-105 focus-visible:ring-2 focus-visible:ring-[var(--neu-accent)] focus-visible:ring-offset-2 active:brightness-95"
			>
				Pause
			</button>
			<button
				type="button"
				onclick={() => showToast('Skipped to next track.')}
				class="neu-sm touch-manipulation px-5 py-2.5 opacity-90 transition-colors duration-150 hover:opacity-100 focus-visible:ring-2 focus-visible:ring-[var(--neu-accent)] focus-visible:ring-offset-2"
			>
				Skip Track
			</button>
			<button
				type="button"
				disabled
				class="neu-sm cursor-not-allowed px-5 py-2.5 opacity-50"
			>
				Connect Voice
			</button>
			<button
				type="button"
				onclick={() => showToast('Shuffle on.')}
				aria-label="Shuffle queue"
				class="neu-sm touch-manipulation grid size-11 place-items-center transition-colors duration-150 hover:brightness-105 focus-visible:ring-2 focus-visible:ring-[var(--neu-accent)] focus-visible:ring-offset-2"
			>
				<span aria-hidden="true">⤨</span>
			</button>
			<button
				type="button"
				onclick={() => showToast('Muted.')}
				aria-label="Mute volume"
				class="neu-sm touch-manipulation grid size-11 place-items-center transition-colors duration-150 hover:brightness-105 focus-visible:ring-2 focus-visible:ring-[var(--neu-accent)] focus-visible:ring-offset-2"
			>
				<span aria-hidden="true">♪</span>
			</button>
		</div>
		<div class="mt-4 flex flex-wrap gap-3">
			<button
				type="button"
				class="neu-inset touch-manipulation px-5 py-2.5 opacity-80 transition-colors duration-150 hover:opacity-100 focus-visible:ring-2 focus-visible:ring-[var(--neu-accent)] focus-visible:ring-offset-2"
			>
				Pressed State
			</button>
			<button
				type="button"
				onclick={() => (confirmDelete = true)}
				class="touch-manipulation rounded-xl border-2 border-red-500/60 px-5 py-2.5 font-semibold text-red-600 transition-colors duration-150 hover:bg-red-500 hover:text-white focus-visible:ring-2 focus-visible:ring-red-500 focus-visible:ring-offset-2 dark:text-red-300"
			>
				Delete Queue…
			</button>
		</div>
		{#if confirmDelete}
			<div
				role="alertdialog"
				aria-modal="true"
				aria-label="Confirm delete"
				class="neu-sm mt-4 max-w-md overscroll-contain p-5"
			>
				<p class="font-semibold">Clear all 8 tracks?</p>
				<p class="mt-1 text-sm opacity-75">You can undo for 10 seconds after clearing.</p>
				<div class="mt-4 flex gap-3">
					<button
						type="button"
						onclick={() => {
							confirmDelete = false;
							showToast('Queue cleared. Undo available.');
						}}
						class="touch-manipulation rounded-xl bg-red-500 px-4 py-2 font-semibold text-white transition-colors duration-150 hover:bg-red-600 focus-visible:ring-2 focus-visible:ring-red-500 focus-visible:ring-offset-2"
					>
						Clear Queue
					</button>
					<button
						type="button"
						onclick={() => (confirmDelete = false)}
						class="neu-sm touch-manipulation px-4 py-2 transition-colors duration-150 hover:brightness-105 focus-visible:ring-2 focus-visible:ring-[var(--neu-accent)] focus-visible:ring-offset-2"
					>
						Keep Music
					</button>
				</div>
			</div>
		{/if}
	</section>

	<!-- Cards -->
	<section aria-labelledby="h-cards" class="scroll-mt-24">
		<h2 id="h-cards" class="text-2xl text-balance">Cards</h2>
		<div class="mt-4 grid gap-5 md:grid-cols-3">
			<article class="neu flex min-w-0 flex-col p-6">
				<h3 class="text-lg text-balance">Now Playing</h3>
				<p class="mt-1 min-w-0 truncate text-sm opacity-70" title="Midnight Loop — Glass Animals">
					Midnight Loop — Glass Animals
				</p>
				<div class="neu-inset mt-4 grid h-28 place-items-center text-4xl" aria-hidden="true">
					♪
				</div>
				<div class="mt-4 flex gap-2">
					<button
						type="button"
						onclick={() => showToast('Paused.')}
						class="neu-sm flex-1 touch-manipulation px-3 py-2 text-sm font-semibold transition-colors duration-150 hover:brightness-105 focus-visible:ring-2 focus-visible:ring-[var(--neu-accent)] focus-visible:ring-offset-2"
					>
						Pause
					</button>
					<button
						type="button"
						onclick={() => showToast('Skipped.')}
						class="neu-sm flex-1 touch-manipulation px-3 py-2 text-sm font-semibold transition-colors duration-150 hover:brightness-105 focus-visible:ring-2 focus-visible:ring-[var(--neu-accent)] focus-visible:ring-offset-2"
					>
						Skip
					</button>
				</div>
			</article>
			<article class="neu-sm flex min-w-0 flex-col p-6">
				<h3 class="text-lg text-balance">Up Next</h3>
				<ul class="mt-2 space-y-2 text-sm">
					<li class="flex items-center justify-between gap-2">
						<span class="min-w-0 truncate">Solar Drift — Nujabes Tribute Band With a Very Long Name</span>
						<span class="shrink-0 tabular-nums opacity-60">3:42</span>
					</li>
					<li class="flex items-center justify-between gap-2">
						<span class="min-w-0 truncate">Bassline Study No. 4</span>
						<span class="shrink-0 tabular-nums opacity-60">2:58</span>
					</li>
					<li class="flex items-center justify-between gap-2">
						<span class="min-w-0 truncate">Quiet Server Room</span>
						<span class="shrink-0 tabular-nums opacity-60">4:11</span>
					</li>
				</ul>
				<a
					href="/dashboard"
					class="mt-4 text-sm font-semibold text-[var(--neu-accent)] underline-offset-4 hover:underline focus-visible:ring-2 focus-visible:ring-[var(--neu-accent)]"
				>
					Manage Queue
				</a>
			</article>
			<article class="neu-lg flex min-w-0 flex-col p-6">
				<h3 class="text-lg text-balance">Voice Status</h3>
				<p class="mt-1 text-sm opacity-70">Connected to Chill Room.</p>
				<div class="mt-4 space-y-2 text-sm" aria-live="polite">
					<div class="flex justify-between gap-2">
						<span>Latency</span><span class="tabular-nums">42 ms</span>
					</div>
					<div class="flex justify-between gap-2">
						<span>Listeners</span><span class="tabular-nums">12</span>
					</div>
				</div>
				<span
					class="mt-4 inline-flex w-fit items-center gap-2 rounded-full bg-green-500/15 px-3 py-1 text-xs font-semibold text-green-700 dark:text-green-300"
				>
					<span aria-hidden="true" class="size-2 rounded-full bg-green-500"></span> Live
				</span>
			</article>
		</div>
	</section>

	<!-- Inputs -->
	<section aria-labelledby="h-inputs" class="neu scroll-mt-24 p-6 md:p-8">
		<h2 id="h-inputs" class="text-2xl text-balance">Inputs</h2>
		<div class="mt-5 grid gap-6 md:grid-cols-2">
			<div>
				<label for="demo-search" class="mb-1.5 block text-sm font-semibold">Search Tracks</label>
				<input
					id="demo-search"
					name="search"
					type="search"
					autocomplete="off"
					placeholder="Midnight Loop…"
					bind:value={query}
					class="neu-inset w-full touch-manipulation px-4 py-2.5 text-sm outline-none transition-shadow duration-150 placeholder:opacity-50 focus-visible:ring-2 focus-visible:ring-[var(--neu-accent)]"
				/>
				<p class="mt-1.5 text-xs opacity-60">Try a title, artist, or pasted link.</p>
			</div>
			<div>
				<label for="demo-volume" class="mb-1.5 block text-sm font-semibold">
					Volume: <span class="tabular-nums">{volume}</span>
				</label>
				<input
					id="demo-volume"
					name="volume"
					type="range"
					min="0"
					max="100"
					bind:value={volume}
					class="w-full accent-[#6d5ef0] focus-visible:ring-2 focus-visible:ring-[var(--neu-accent)]"
				/>
			</div>
			<div>
				<label for="demo-source" class="mb-1.5 block text-sm font-semibold">Audio Source</label>
				<select
					id="demo-source"
					name="source"
					class="neu-inset w-full touch-manipulation bg-[var(--neu-bg)] px-4 py-2.5 text-sm text-[var(--neu-text)] outline-none focus-visible:ring-2 focus-visible:ring-[var(--neu-accent)]"
				>
					<option>YouTube</option>
					<option>Spotify</option>
					<option>SoundCloud</option>
				</select>
			</div>
			<div>
				<label for="demo-notes" class="mb-1.5 block text-sm font-semibold">DJ Notes</label>
				<textarea
					id="demo-notes"
					name="notes"
					rows="3"
					autocomplete="off"
					placeholder="Keep transitions under 6 seconds…"
					class="neu-inset w-full touch-manipulation resize-y break-words px-4 py-2.5 text-sm outline-none transition-shadow duration-150 placeholder:opacity-50 focus-visible:ring-2 focus-visible:ring-[var(--neu-accent)]"
				></textarea>
			</div>
		</div>
		<fieldset class="mt-6">
			<legend class="text-sm font-semibold">Playback Options</legend>
			<div class="mt-2 flex flex-wrap gap-5">
				<label class="inline-flex cursor-pointer touch-manipulation items-center gap-2 text-sm">
					<input
						type="checkbox"
						name="autoplay"
						checked
						class="size-5 accent-[#6d5ef0] focus-visible:ring-2 focus-visible:ring-[var(--neu-accent)]"
					/> Autoplay
				</label>
				<label class="inline-flex cursor-pointer touch-manipulation items-center gap-2 text-sm">
					<input
						type="checkbox"
						name="crossfade"
						class="size-5 accent-[#6d5ef0] focus-visible:ring-2 focus-visible:ring-[var(--neu-accent)]"
					/> Crossfade
				</label>
				<label class="inline-flex cursor-pointer touch-manipulation items-center gap-2 text-sm">
					<input
						type="radio"
						name="loop"
						checked
						class="size-5 accent-[#6d5ef0] focus-visible:ring-2 focus-visible:ring-[var(--neu-accent)]"
					/> Loop All
				</label>
				<label class="inline-flex cursor-pointer touch-manipulation items-center gap-2 text-sm">
					<input
						type="radio"
						name="loop"
						class="size-5 accent-[#6d5ef0] focus-visible:ring-2 focus-visible:ring-[var(--neu-accent)]"
					/> Loop Off
				</label>
			</div>
		</fieldset>
	</section>

	<!-- Login -->
	<section aria-labelledby="h-login" class="scroll-mt-24">
		<h2 id="h-login" class="text-2xl text-balance">Sign In</h2>
		<form
			onsubmit={handleLogin}
			novalidate={false}
			class="neu-lg mx-auto mt-4 max-w-md space-y-4 p-6 md:p-8"
		>
			<div>
				<label for="login-email" class="mb-1.5 block text-sm font-semibold">Email</label>
				<input
					id="login-email"
					name="email"
					type="email"
					inputmode="email"
					autocomplete="email"
					spellcheck={false}
					required
					placeholder="Ada@example.com…"
					bind:value={email}
					aria-invalid={emailError ? 'true' : undefined}
					aria-describedby={emailError ? 'login-email-error' : undefined}
					class="neu-inset w-full touch-manipulation px-4 py-2.5 text-sm outline-none transition-shadow duration-150 placeholder:opacity-50 focus-visible:ring-2 focus-visible:ring-[var(--neu-accent)]"
				/>
				{#if emailError}
					<p id="login-email-error" role="alert" class="mt-1.5 text-sm text-red-600 dark:text-red-300">
						{emailError}
					</p>
				{/if}
			</div>
			<div>
				<label for="login-password" class="mb-1.5 block text-sm font-semibold">Password</label>
				<input
					id="login-password"
					name="password"
					type="password"
					autocomplete="current-password"
					required
					placeholder="Your password…"
					bind:value={password}
					class="neu-inset w-full touch-manipulation px-4 py-2.5 text-sm outline-none transition-shadow duration-150 placeholder:opacity-50 focus-visible:ring-2 focus-visible:ring-[var(--neu-accent)]"
				/>
			</div>
			<button
				type="submit"
				disabled={isSubmitting}
				class="touch-manipulation w-full rounded-xl bg-[var(--neu-accent)] px-5 py-2.5 font-semibold text-white transition-colors duration-150 hover:brightness-110 focus-visible:ring-2 focus-visible:ring-[var(--neu-accent)] focus-visible:ring-offset-2 disabled:opacity-70"
			>
				{isSubmitting ? 'Signing In…' : 'Sign In'}
			</button>
			<p class="text-center text-sm opacity-70">
				New here? <a
					href="/dashboard"
					class="font-semibold text-[var(--neu-accent)] underline-offset-4 hover:underline focus-visible:ring-2 focus-visible:ring-[var(--neu-accent)]"
					>Create Account</a
				>
			</p>
		</form>
	</section>

	<!-- Feedback + data -->
	<section aria-labelledby="h-feedback" class="grid gap-5 md:grid-cols-2">
		<div class="neu p-6">
			<h2 id="h-feedback" class="text-xl text-balance">Status</h2>
			<div role="status" class="mt-3 rounded-xl border-l-4 border-green-500 bg-green-500/10 p-4 text-sm">
				Connected. You control 2 voice channels.
			</div>
			<div role="alert" class="mt-3 rounded-xl border-l-4 border-amber-500 bg-amber-500/10 p-4 text-sm">
				Spotify token expires in 3 days. Reconnect to keep playlists.
			</div>
			<div class="mt-4">
				<div class="flex justify-between text-sm tabular-nums"><span>Buffering</span><span>68&#37;</span></div>
				<div
					role="progressbar"
					aria-valuenow={68}
					aria-valuemin={0}
					aria-valuemax={100}
					aria-label="Buffering progress"
					class="neu-inset mt-2 h-3 overflow-hidden"
				>
					<div class="h-full w-[68%] rounded-full bg-[var(--neu-accent)] motion-safe:transition-[width] motion-safe:duration-300"></div>
				</div>
			</div>
			<div class="mt-4 flex flex-wrap gap-2">
				<span class="rounded-full bg-[var(--neu-accent)] px-3 py-1 text-xs font-semibold text-white">House</span>
				<span class="neu-sm px-3 py-1 text-xs font-semibold">Lo-Fi</span>
				<span class="neu-sm px-3 py-1 text-xs font-semibold">8 Tracks</span>
			</div>
		</div>
		<div class="neu p-6">
			<h2 class="text-xl text-balance">Queue Tabs</h2>
			<div role="tablist" aria-label="Queue views" class="neu-inset mt-3 flex gap-1 p-1">
				{#each (['Queue', 'History', 'Saved'] as const) as tab}
					<button
						type="button"
						role="tab"
						aria-selected={activeTab === tab}
						onclick={() => (activeTab = tab)}
						class="flex-1 touch-manipulation rounded-lg px-3 py-2 text-sm font-semibold transition-colors duration-150 hover:brightness-105 focus-visible:ring-2 focus-visible:ring-[var(--neu-accent)] {activeTab === tab
							? 'bg-[var(--neu-accent)] text-white'
							: ''}"
					>
						{tab}
					</button>
				{/each}
			</div>
			<p class="mt-3 min-w-0 break-words text-sm opacity-75">
				{#if activeTab === 'Queue'}
					8 tracks, 31 minutes. Drag is not needed — use the buttons below.
				{:else if activeTab === 'History'}
					You played 24 tracks this week.
				{:else}
					No saved playlists yet. Save one to see it here.
				{/if}
			</p>
			<nav aria-label="Breadcrumb" class="mt-4 text-sm">
				<ol class="flex min-w-0 flex-wrap items-center gap-1.5">
					<li class="min-w-0"><a href="/" class="underline-offset-4 hover:underline focus-visible:ring-2 focus-visible:ring-[var(--neu-accent)]">Home</a></li>
					<li aria-hidden="true">/</li>
					<li class="min-w-0"><a href="/dashboard" class="underline-offset-4 hover:underline focus-visible:ring-2 focus-visible:ring-[var(--neu-accent)]">Dashboard</a></li>
					<li aria-hidden="true">/</li>
					<li aria-current="page" class="min-w-0 truncate font-semibold">Gallery</li>
				</ol>
			</nav>
		</div>
	</section>
</div>

<div aria-live="polite" class="pointer-events-none fixed inset-x-0 bottom-6 flex justify-center px-4">
	{#if toast}
		<p
			class="neu-sm motion-safe:transition-opacity motion-safe:duration-200 bg-[var(--neu-bg)] px-5 py-3 text-sm font-semibold"
		>
			{toast}
		</p>
	{/if}
</div>

<style>
	@media (prefers-reduced-motion: reduce) {
		* {
			transition-duration: 0.01ms !important;
			animation-duration: 0.01ms !important;
		}
	}
</style>
