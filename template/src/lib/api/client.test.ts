import { afterEach, describe, expect, it, vi } from 'vitest';
import {
	ApiError,
	ValidationError,
	getLikedTracks,
	getMe,
	getPlaylistTracks,
	getPlaylists,
	request
} from './client.js';
import { authUserSchema } from './schemas.js';

const validUser = {
	id: 'user-1',
	email: 'ada@example.com',
	name: 'Ada Lovelace',
	picture: 'https://example.com/ada.png',
	locale: 'en',
	spotifyConnected: false
};

function jsonResponse(body: unknown, status = 200) {
	return new Response(JSON.stringify(body), {
		status,
		headers: { 'content-type': 'application/json' }
	});
}

afterEach(() => {
	vi.unstubAllGlobals();
});

describe('getMe', () => {
	it('returns the validated signed-in account', async () => {
		vi.stubGlobal('fetch', vi.fn().mockResolvedValue(jsonResponse(validUser)));

		await expect(getMe()).resolves.toEqual(validUser);
	});

	it('rejects with a ValidationError when a profile field is missing', async () => {
		const malformed = {
			id: validUser.id,
			email: validUser.email,
			locale: validUser.locale,
			spotifyConnected: validUser.spotifyConnected,
			picture: validUser.picture
		};
		vi.stubGlobal('fetch', vi.fn().mockResolvedValue(jsonResponse(malformed)));

		await expect(getMe()).rejects.toBeInstanceOf(ValidationError);
	});
});

describe('request failure mapping', () => {
	it('throws an ApiError carrying the status when the response is not ok', async () => {
		vi.stubGlobal('fetch', vi.fn().mockResolvedValue(jsonResponse({ error: 'unauthorized' }, 401)));

		const error = await getMe().catch((thrown) => thrown);

		expect(error).toBeInstanceOf(ApiError);
		expect(error).toMatchObject({ status: 401, message: 'unauthorized' });
	});
});

describe('outbound validation', () => {
	it('rejects with a ValidationError before calling fetch when the body schema fails', async () => {
		const fetchMock = vi.fn().mockResolvedValue(jsonResponse({ status: 'ok' }));
		vi.stubGlobal('fetch', fetchMock);

		await expect(
			request('/auth/logout', authUserSchema, {
				method: 'POST',
				body: { id: 1 },
				bodySchema: authUserSchema
			})
		).rejects.toBeInstanceOf(ValidationError);

		expect(fetchMock).not.toHaveBeenCalled();
	});
});

const playlistPage = {
	items: [
		{
			id: 'pl-1',
			name: 'Deep Focus',
			description: 'Long sessions',
			owner: 'Ada',
			trackCount: 12,
			imageUrl: 'https://img.example/cover.jpg',
			url: 'https://open.spotify.com/playlist/pl-1',
			isOwned: true,
			collaborative: false,
			browseable: true
		}
	],
	next: null
};

const trackPage = {
	items: [
		{
			id: 'tr-1',
			title: 'Nightcall',
			artists: ['Kavinsky'],
			album: 'OutRun',
			durationMs: 258000,
			imageUrl: 'https://img.example/outrun.jpg',
			url: 'https://open.spotify.com/track/tr-1'
		}
	],
	next: 100
};

describe('getPlaylists', () => {
	it('returns the validated playlist page and requests the right page', async () => {
		const fetchMock = vi.fn().mockResolvedValue(jsonResponse(playlistPage));
		vi.stubGlobal('fetch', fetchMock);

		await expect(getPlaylists()).resolves.toEqual(playlistPage);
		expect(fetchMock.mock.calls[0][0]).toBe('/api/spotify/playlists?offset=0&limit=50');
	});

	it('rejects a page whose playlist is missing required fields', async () => {
		vi.stubGlobal('fetch', vi.fn().mockResolvedValue(jsonResponse({ items: [{ id: 'pl-1' }], next: null })));

		await expect(getPlaylists()).rejects.toBeInstanceOf(ValidationError);
	});
});

describe('getPlaylistTracks', () => {
	it('encodes the playlist id and validates the track page', async () => {
		const fetchMock = vi.fn().mockResolvedValue(jsonResponse(trackPage));
		vi.stubGlobal('fetch', fetchMock);

		await expect(getPlaylistTracks('pl 1')).resolves.toEqual(trackPage);
		expect(fetchMock.mock.calls[0][0]).toBe('/api/spotify/playlists/pl%201/tracks?offset=0&limit=100');
	});
});

describe('getLikedTracks', () => {
	it('validates the liked track page', async () => {
		const fetchMock = vi.fn().mockResolvedValue(jsonResponse(trackPage));
		vi.stubGlobal('fetch', fetchMock);

		await expect(getLikedTracks()).resolves.toEqual(trackPage);
		expect(fetchMock.mock.calls[0][0]).toBe('/api/spotify/liked?offset=0&limit=100');
	});
});
