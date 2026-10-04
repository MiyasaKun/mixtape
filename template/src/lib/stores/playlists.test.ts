import { describe, expect, it, vi } from 'vitest';
import { get } from 'svelte/store';
import { cachedPlaylist, createPagedResource, rememberPlaylists, spotifyErrorCode } from './playlists.js';

function deferred<T>() {
	let resolve!: (value: T) => void;
	let reject!: (reason?: unknown) => void;
	const promise = new Promise<T>((res, rej) => {
		resolve = res;
		reject = rej;
	});
	return { promise, resolve, reject };
}

describe('createPagedResource', () => {
	it('loads the first page', async () => {
		const fetchPage = vi.fn().mockResolvedValue({ items: ['a', 'b'], next: 2 });
		const resource = createPagedResource(fetchPage);

		await resource.load();

		expect(fetchPage).toHaveBeenCalledWith(0);
		expect(get(resource)).toMatchObject({ items: ['a', 'b'], next: 2, status: 'ready', error: '' });
	});

	it('appends the next page on loadMore and advances the offset', async () => {
		const fetchPage = vi
			.fn()
			.mockResolvedValueOnce({ items: ['a'], next: 1 })
			.mockResolvedValueOnce({ items: ['b'], next: null });
		const resource = createPagedResource(fetchPage);

		await resource.load();
		await resource.loadMore();

		expect(fetchPage).toHaveBeenNthCalledWith(2, 1);
		expect(get(resource)).toMatchObject({ items: ['a', 'b'], next: null, status: 'ready' });
	});

	it('does nothing on loadMore when the list is exhausted', async () => {
		const fetchPage = vi.fn().mockResolvedValue({ items: ['a'], next: null });
		const resource = createPagedResource(fetchPage);

		await resource.load();
		await resource.loadMore();

		expect(fetchPage).toHaveBeenCalledTimes(1);
	});

	it('maps a failed load to an error status with the backend code', async () => {
		const fetchPage = vi.fn().mockRejectedValue(new Error('spotify_not_connected'));
		const resource = createPagedResource(fetchPage);

		await resource.load();

		expect(get(resource)).toMatchObject({ items: [], status: 'error', error: 'not_connected' });
	});

	it('keeps loaded items when loadMore fails', async () => {
		const fetchPage = vi
			.fn()
			.mockResolvedValueOnce({ items: ['a'], next: 1 })
			.mockRejectedValueOnce(new Error('spotify_unavailable'));
		const resource = createPagedResource(fetchPage);

		await resource.load();
		await resource.loadMore();

		expect(get(resource)).toMatchObject({ items: ['a'], status: 'ready', error: 'unavailable' });
	});

	it('exposes a loading state while a page is in flight', async () => {
		const pending = deferred<{ items: string[]; next: number | null }>();
		const resource = createPagedResource(() => pending.promise);

		const load = resource.load();
		expect(get(resource).status).toBe('loading');

		pending.resolve({ items: ['a'], next: null });
		await load;
		expect(get(resource).status).toBe('ready');
	});
});

describe('playlist catalogue', () => {
	it('remembers playlists by id for the detail view', () => {
		rememberPlaylists([
			{
				id: 'pl-1',
				name: 'Deep Focus',
				description: '',
				owner: 'Ada',
				trackCount: 12,
				imageUrl: '',
				url: 'https://open.spotify.com/playlist/pl-1',
				isOwned: true,
				collaborative: false,
				browseable: true
			}
		]);

		expect(cachedPlaylist('pl-1')?.name).toBe('Deep Focus');
		expect(cachedPlaylist('missing')).toBeUndefined();
	});
});

describe('spotifyErrorCode', () => {
	it.each([
		['spotify_not_connected', 'not_connected'],
		['spotify_reconnect_required', 'reconnect_required'],
		['spotify_forbidden', 'forbidden'],
		['spotify_not_found', 'not_found'],
		['spotify_rate_limited', 'rate_limited'],
		['spotify_unavailable', 'unavailable'],
		['something else', 'unavailable']
	])('maps %s to %s', (message, expected) => {
		expect(spotifyErrorCode(new Error(message))).toBe(expected);
	});
});
