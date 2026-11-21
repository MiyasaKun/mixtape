<script lang="ts">
	import { goto } from '$app/navigation';
	import { m } from '../../paraglide/messages.js';
	import { setLocale, getLocale } from '../../paraglide/runtime.js';
	import { signOut } from '#lib/stores/auth.js';
	import { theme, toggleTheme } from '#lib/stores/theme.js';
	import type { AuthUser } from '#lib/api/schemas.js';

	let { user }: { user: AuthUser | null } = $props();

	let open = $state(false);
	let avatarFailed = $state(false);
	let root: HTMLElement | undefined;

	const currentLocale = $derived(getLocale());
	const initials = $derived((user?.name || user?.email || '?').trim().charAt(0).toUpperCase());

	// Close on outside click and Escape while the menu is open.
	$effect(() => {
		if (!open) return;

		const onPointerDown = (event: PointerEvent) => {
			if (root && !root.contains(event.target as Node)) open = false;
		};
		const onKeyDown = (event: KeyboardEvent) => {
			if (event.key === 'Escape') open = false;
		};

		document.addEventListener('pointerdown', onPointerDown);
		document.addEventListener('keydown', onKeyDown);
		return () => {
			document.removeEventListener('pointerdown', onPointerDown);
			document.removeEventListener('keydown', onKeyDown);
		};
	});

	async function handleSignOut() {
		open = false;
		await signOut();
		goto('/login');
	}
</script>

<div bind:this={root} class="relative">
	<button
		type="button"
		class="flex w-full items-center gap-3 rounded-xl px-3 py-2 text-left transition-colors touch-manipulation hover:bg-[var(--neu-light)]/60 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-[var(--neu-accent)] dark:hover:bg-[var(--neu-light)]/20"
		aria-haspopup="menu"
		aria-expanded={open}
		aria-label={m.account_menu_label()}
		onclick={() => (open = !open)}
	>
		{#if user?.picture && !avatarFailed}
			<img
				src={user.picture}
				alt=""
				width="40"
				height="40"
				referrerpolicy="no-referrer"
				class="h-10 w-10 shrink-0 rounded-full object-cover"
				onerror={() => (avatarFailed = true)}
			/>
		{:else}
			<span
				class="flex h-10 w-10 shrink-0 items-center justify-center rounded-full bg-[var(--neu-accent)] text-sm font-semibold text-white"
				aria-hidden="true">{initials}</span
			>
		{/if}
		<span class="min-w-0 flex-1">
			<span class="block truncate text-sm font-semibold">{user?.name || '—'}</span>
			<span class="block truncate text-xs opacity-70">{user?.email || ''}</span>
		</span>
		<svg
			xmlns="http://www.w3.org/2000/svg"
			fill="none"
			viewBox="0 0 24 24"
			stroke-width="1.5"
			stroke="currentColor"
			class="h-4 w-4 shrink-0 opacity-60"
			aria-hidden="true"
		>
			<path stroke-linecap="round" stroke-linejoin="round" d="m4.5 15.75 7.5-7.5 7.5 7.5" />
		</svg>
	</button>

	{#if open}
		<ul
			role="menu"
			aria-label={m.account_menu_label()}
			class="absolute bottom-full left-0 right-0 z-50 mb-2 overflow-hidden rounded-xl border border-[var(--neu-dark)]/30 bg-[var(--neu-bg)] py-1 shadow-[8px_8px_16px_var(--neu-dark),-8px_-8px_16px_var(--neu-light)] [&_a]:touch-manipulation [&_button]:touch-manipulation"
		>
			<li role="none">
				<a
					role="menuitem"
					href="/settings"
					class="block px-4 py-2 text-sm hover:bg-[var(--neu-light)]/60 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-[var(--neu-accent)] dark:hover:bg-[var(--neu-light)]/20"
					onclick={() => (open = false)}
				>
					{m.settings_title()}
				</a>
			</li>
			<li role="none">
				<button
					type="button"
					role="menuitem"
					class="flex w-full items-center justify-between px-4 py-2 text-left text-sm hover:bg-[var(--neu-light)]/60 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-[var(--neu-accent)] dark:hover:bg-[var(--neu-light)]/20"
					onclick={toggleTheme}
				>
					<span>{m.theme_label()}</span>
					<span class="opacity-70">{$theme === 'dark' ? m.theme_dark() : m.theme_light()}</span>
				</button>
			</li>
			<li role="none">
				<button
					type="button"
					role="menuitemradio"
					aria-checked={currentLocale === 'en'}
					class="flex w-full items-center justify-between px-4 py-2 text-left text-sm hover:bg-[var(--neu-light)]/60 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-[var(--neu-accent)] dark:hover:bg-[var(--neu-light)]/20"
					onclick={() => setLocale('en')}
				>
					<span>English</span>
					{#if currentLocale === 'en'}<span aria-hidden="true">✓</span>{/if}
				</button>
			</li>
			<li role="none">
				<button
					type="button"
					role="menuitemradio"
					aria-checked={currentLocale === 'de'}
					class="flex w-full items-center justify-between px-4 py-2 text-left text-sm hover:bg-[var(--neu-light)]/60 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-[var(--neu-accent)] dark:hover:bg-[var(--neu-light)]/20"
					onclick={() => setLocale('de')}
				>
					<span>Deutsch</span>
					{#if currentLocale === 'de'}<span aria-hidden="true">✓</span>{/if}
				</button>
			</li>
			<li role="none" class="mt-1 border-t border-[var(--neu-dark)]/30 pt-1">
				<button
					type="button"
					role="menuitem"
					class="block w-full px-4 py-2 text-left text-sm text-red-600 hover:bg-[var(--neu-light)]/60 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-[var(--neu-accent)] dark:text-red-300 dark:hover:bg-[var(--neu-light)]/20"
					onclick={handleSignOut}
				>
					{m.sign_out()}
				</button>
			</li>
		</ul>
	{/if}
</div>
