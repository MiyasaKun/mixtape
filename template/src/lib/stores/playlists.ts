import { writable, get } from 'svelte/store';
import type { Readable } from 'svelte/store';
import type { Playlist } from '../api/schemas.js';

/**
 * Pagination state shared by the Playlists list, a playlist's tracks, and
 * Liked Songs. The backend pages by offset and reports `next` as the offset of
 * the following page (or null when exhausted).
 */

export type PagedStatus = 'idle' | 'loading' | 'loading-more' | 'ready' | 'error';

export interface Page<T> {
	items: T[];
	next: number | null;
}

export interface PagedState<T> {
	items: T[];
	next: number | null;
	status: PagedStatus;
	error: string;
}

export interface PagedResource<T> extends Readable<PagedState<T>> {
	/** Loads the first page, replacing any current items. */
	load(): Promise<void>;
	/** Appends the next page when one exists and the resource is ready. */
	loadMore(): Promise<void>;
	/** Returns the resource to its initial empty state. */
	reset(): void;
}

const empty = <T>(): PagedState<T> => ({ items: [], next: null, status: 'idle', error: '' });

export function createPagedResource<T>(fetchPage: (offset: number) => Promise<Page<T>>): PagedResource<T> {
	const store = writable<PagedState<T>>(empty<T>());

	async function load(): Promise<void> {
		store.set({ items: [], next: null, status: 'loading', error: '' });
		try {
			const page = await fetchPage(0);
			store.set({ items: page.items, next: page.next, status: 'ready', error: '' });
		} catch (error) {
			store.set({ items: [], next: null, status: 'error', error: spotifyErrorCode(error) });
		}
	}

	async function loadMore(): Promise<void> {
		const state = get(store);
		if (state.status !== 'ready' || state.next === null) return;

		const offset = state.next;
		store.set({ ...state, status: 'loading-more', error: '' });
		try {
			const page = await fetchPage(offset);
			store.update((value) => ({
				items: [...value.items, ...page.items],
				next: page.next,
				status: 'ready',
				error: ''
			}));
		} catch (error) {
			store.update((value) => ({ ...value, status: 'ready', error: spotifyErrorCode(error) }));
		}
	}

	function reset(): void {
		store.set(empty<T>());
	}

	return { subscribe: store.subscribe, load, loadMore, reset };
}

const catalog = new Map<string, Playlist>();

/**
 * Remembers playlist summaries so the detail view can still show the cover and
 * name after a client-side navigation. A direct deep link has no cached entry
 * and falls back to a generic header.
 */
export function rememberPlaylists(items: Playlist[]): void {
	for (const item of items) catalog.set(item.id, item);
}

/** Returns a previously remembered playlist summary, if any. */
export function cachedPlaylist(id: string): Playlist | undefined {
	return catalog.get(id);
}

/** Maps a backend `ApiError` message to a small, translatable UI state. */
export function spotifyErrorCode(error: unknown): string {
	const message = error instanceof Error ? error.message : '';
	switch (message) {
		case 'spotify_not_connected':
			return 'not_connected';
		case 'spotify_reconnect_required':
			return 'reconnect_required';
		case 'spotify_forbidden':
			return 'forbidden';
		case 'spotify_not_found':
			return 'not_found';
		case 'spotify_rate_limited':
			return 'rate_limited';
		default:
			return 'unavailable';
	}
}
