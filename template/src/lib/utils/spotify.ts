/**
 * Builds the href for an "Open in Spotify" link. When the account-level
 * autoplay preference is enabled the link asks Spotify's web player to start
 * playing immediately.
 */
export function spotifyUrl(url: string, autoplay: boolean): string {
	if (!autoplay || !url) return url;
	try {
		const parsed = new URL(url);
		if (!parsed.searchParams.has('autoplay')) {
			parsed.searchParams.set('autoplay', 'true');
		}
		return parsed.toString();
	} catch {
		// Not a parseable URL (defensive: backend URLs are absolute); leave as-is.
		return url;
	}
}
