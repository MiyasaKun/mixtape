import { writable, derived } from 'svelte/store';
import type { Readable, Writable } from 'svelte/store';
import * as api from '../api/client.js';
import type { AuthUser } from '../api/schemas.js';

/** The signed-in account, or null. */
export const user: Writable<AuthUser | null> = writable(null);

/** True while the initial session check is in flight. */
export const sessionLoading: Writable<boolean> = writable(true);

/** True once the account is known to be signed in. */
export const isAuthenticated: Readable<boolean> = derived(user, ($user) => $user !== null);

/** Loads the current session from `/api/auth/me`. */
export async function loadSession(): Promise<void> {
	sessionLoading.set(true);
	try {
		user.set(await api.getMe());
	} catch {
		user.set(null);
	} finally {
		sessionLoading.set(false);
	}
}

/** Sends the browser to Google's consent screen. */
export async function signInWithGoogle(): Promise<void> {
	const { url } = await api.getGoogleAuthUrl();
	window.location.assign(url);
}

/** Signs out locally and clears in-memory state. */
export async function signOut(): Promise<void> {
	await api.logout();
	user.set(null);
}

/** Marks the account as Spotify-connected after a successful connect. */
export function setSpotifyConnected(connected: boolean): void {
	user.update(($user) => ($user ? { ...$user, spotifyConnected: connected } : $user));
}

/** Applies the saved Spotify autoplay preference to the signed-in account. */
export function setSpotifyAutoplay(enabled: boolean): void {
	user.update(($user) => ($user ? { ...$user, spotifyAutoplay: enabled } : $user));
}
