<script lang="ts">
	import { page } from '$app/state';

	/**
	 * One sidebar row: active state, press feedback and focus ring live here
	 * instead of being hand-duplicated per link.
	 */
	let { href, label, d: iconPath }: { href: string; label: string; d?: string } = $props();

	const isActive = $derived(
		href === '/' ? page.url.pathname === '/' : page.url.pathname.startsWith(href)
	);
</script>

<a
	href={href}
	aria-current={isActive ? 'page' : undefined}
	class="group flex min-h-11 items-center gap-3 rounded-xl px-3 py-2.5 text-sm font-medium transition-colors duration-150 touch-manipulation focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-[var(--neu-accent)] {isActive
		? 'bg-[var(--neu-accent)] text-white shadow-[inset_2px_2px_6px_rgba(0,0,0,0.25)]'
		: 'hover:bg-black/[0.04] dark:hover:bg-white/5'}"
>
	{#if iconPath}
		<svg
			xmlns="http://www.w3.org/2000/svg"
			fill="none"
			viewBox="0 0 24 24"
			stroke-width="1.75"
			stroke="currentColor"
			class="h-5 w-5 shrink-0 opacity-80"
			aria-hidden="true"
		>
			<path stroke-linecap="round" stroke-linejoin="round" d={iconPath} />
		</svg>
	{/if}
	<span class="truncate">{label}</span>
</a>
