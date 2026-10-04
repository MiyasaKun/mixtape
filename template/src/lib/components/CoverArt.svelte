<script lang="ts">
	/**
	 * Square artwork with a graceful fallback: when Spotify has no image (or it
	 * fails to load) the first letter of the name stands in. Purely decorative,
	 * so it is hidden from assistive tech; the accessible name lives next to it.
	 */
	let {
		src,
		name,
		size = 'md'
	}: { src?: string; name: string; size?: 'sm' | 'md' | 'lg' } = $props();

	let failed = $state(false);

	const sizeClass = $derived(
		{
			sm: 'neu-sm h-12 w-12 text-base',
			md: 'neu-sm h-20 w-20 text-2xl',
			lg: 'neu h-40 w-40 text-5xl md:h-52 md:w-52'
		}[size]
	);
	const showImage = $derived(Boolean(src) && !failed);
	const initial = $derived(name.trim().charAt(0).toUpperCase() || '♪');
</script>

<div class="{sizeClass} flex shrink-0 items-center justify-center overflow-hidden" aria-hidden="true">
	{#if showImage}
		<img src={src} alt="" loading="lazy" class="h-full w-full object-cover" onerror={() => (failed = true)} />
	{:else}
		<span class="font-semibold text-[var(--neu-accent)]">{initial}</span>
	{/if}
</div>
