import { z } from 'zod';

/**
 * Response shapes for the Mixtape `/api/*` backend. Every payload that crosses
 * the network is declared here once; TypeScript types are inferred from the
 * schemas so validation and types can never drift apart.
 */

/** The signed-in account (`GET /api/auth/me`). */
export const authUserSchema = z.object({
	id: z.string(),
	email: z.email(),
	name: z.string(),
	picture: z.string(),
	locale: z.string(),
	spotifyConnected: z.boolean()
});

export type AuthUser = z.infer<typeof authUserSchema>;

/** An OAuth consent URL (`.../google/url`, `.../spotify/url`). */
export const authUrlSchema = z.object({
	url: z.url()
});

export type AuthUrl = z.infer<typeof authUrlSchema>;

/** The generic acknowledgement (`POST /auth/logout`, `DELETE /auth/spotify`). */
export const okSchema = z.object({
	status: z.literal('ok')
});

export type Ok = z.infer<typeof okSchema>;

/** A Spotify playlist summary (`GET /api/spotify/playlists`). */
export const playlistSchema = z.object({
	id: z.string(),
	name: z.string(),
	description: z.string(),
	owner: z.string(),
	trackCount: z.number(),
	imageUrl: z.string(),
	url: z.string(),
	isOwned: z.boolean(),
	collaborative: z.boolean(),
	browseable: z.boolean()
});

export type Playlist = z.infer<typeof playlistSchema>;

/** A single song in a playlist or the Liked Songs library. */
export const playlistTrackSchema = z.object({
	id: z.string(),
	title: z.string(),
	artists: z.array(z.string()),
	album: z.string(),
	durationMs: z.number(),
	imageUrl: z.string(),
	url: z.string()
});

export type PlaylistTrack = z.infer<typeof playlistTrackSchema>;

/** A page of playlists; `next` is the offset of the following page or null. */
export const playlistPageSchema = z.object({
	items: z.array(playlistSchema),
	next: z.number().nullable()
});

export type PlaylistPage = z.infer<typeof playlistPageSchema>;

/** A page of tracks; `next` is the offset of the following page or null. */
export const playlistTrackPageSchema = z.object({
	items: z.array(playlistTrackSchema),
	next: z.number().nullable()
});

export type PlaylistTrackPage = z.infer<typeof playlistTrackPageSchema>;
