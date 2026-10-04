import { z } from 'zod';
import {
	authUserSchema,
	authUrlSchema,
	okSchema,
	playlistPageSchema,
	playlistTrackPageSchema
} from './schemas.js';
import type { AuthUser, AuthUrl, Ok, PlaylistPage, PlaylistTrackPage } from './schemas.js';

export type { AuthUser, AuthUrl, Ok, Playlist, PlaylistTrack, PlaylistPage, PlaylistTrackPage } from './schemas.js';

/**
 * Typed fetch wrapper for the Mixtape `/api/*` backend. All requests send the
 * session cookie and every payload in either direction is validated against a
 * Zod schema, so nothing crosses the boundary unchecked. Nothing here talks to
 * third-party music APIs directly.
 */

const API_BASE = '/api';

/** An HTTP-level failure: the backend answered, but not with a success status. */
export class ApiError extends Error {
	readonly status: number;

	constructor(status: number, message: string) {
		super(message);
		this.name = 'ApiError';
		this.status = status;
	}
}

/** A shape failure: data sent or returned did not match its schema. */
export class ValidationError extends Error {
	readonly issues: unknown;

	constructor(message: string, issues?: unknown) {
		super(message);
		this.name = 'ValidationError';
		this.issues = issues;
	}
}

export interface RequestOptions {
	method?: string;
	/** Outbound payload; requires `bodySchema`. */
	body?: unknown;
	/** Schema the outbound `body` is validated against before sending. */
	bodySchema?: z.ZodType;
}

/**
 * Performs a request and returns the validated response body.
 * @throws {ValidationError} When the outbound body or response fails its schema.
 * @throws {ApiError} When the response status is not ok.
 */
export async function request<TResponse>(
	path: string,
	schema: z.ZodType<TResponse>,
	options: RequestOptions = {}
): Promise<TResponse> {
	const { method = 'GET', body, bodySchema } = options;
	const headers = new Headers();
	let payload: string | undefined;

	if (body !== undefined) {
		if (!bodySchema) {
			throw new ValidationError(`${path} sends a body without a schema`);
		}
		const sent = bodySchema.safeParse(body);
		if (!sent.success) {
			throw new ValidationError(`${path} sent an invalid body`, sent.error);
		}
		headers.set('Content-Type', 'application/json');
		payload = JSON.stringify(sent.data);
	}

	const response = await fetch(`${API_BASE}${path}`, {
		method,
		credentials: 'include',
		headers,
		body: payload
	});

	const isJson = response.headers.get('content-type')?.includes('application/json') ?? false;
	const decoded: unknown = isJson ? await response.json() : null;

	if (!response.ok) {
		const message =
			decoded && typeof decoded === 'object' && 'error' in decoded
				? String((decoded as { error: unknown }).error)
				: response.statusText;
		throw new ApiError(response.status, message);
	}

	const parsed = schema.safeParse(decoded);
	if (!parsed.success) {
		throw new ValidationError(`${path} returned an invalid response`, parsed.error);
	}
	return parsed.data;
}

/** Returns the signed-in account, or throws `ApiError(401)` when signed out. */
export function getMe(): Promise<AuthUser> {
	return request('/auth/me', authUserSchema);
}

/** @returns the Google consent URL. */
export function getGoogleAuthUrl(): Promise<AuthUrl> {
	return request('/auth/google/url', authUrlSchema);
}

/** Revokes the current session and clears the cookie. */
export function logout(): Promise<Ok> {
	return request('/auth/logout', okSchema, { method: 'POST' });
}

/** @returns the Spotify consent URL. */
export function getSpotifyAuthUrl(): Promise<AuthUrl> {
	return request('/auth/spotify/url', authUrlSchema);
}

/** Removes the current user's Spotify connection. */
export function disconnectSpotify(): Promise<Ok> {
	return request('/auth/spotify', okSchema, { method: 'DELETE' });
}

/** Returns a page of the signed-in user's Spotify playlists. */
export function getPlaylists(offset = 0, limit = 50): Promise<PlaylistPage> {
	return request(`/spotify/playlists?offset=${offset}&limit=${limit}`, playlistPageSchema);
}

/** Returns a page of tracks in a single Spotify playlist. */
export function getPlaylistTracks(id: string, offset = 0, limit = 100): Promise<PlaylistTrackPage> {
	const encoded = encodeURIComponent(id);
	return request(`/spotify/playlists/${encoded}/tracks?offset=${offset}&limit=${limit}`, playlistTrackPageSchema);
}

/** Returns a page of the signed-in user's Liked Songs. */
export function getLikedTracks(offset = 0, limit = 100): Promise<PlaylistTrackPage> {
	return request(`/spotify/liked?offset=${offset}&limit=${limit}`, playlistTrackPageSchema);
}
