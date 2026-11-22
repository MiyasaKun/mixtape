import { describe, expect, it } from 'vitest';
import { spotifyUrl } from './spotify.js';

describe('spotifyUrl', () => {
	it('returns the URL untouched when autoplay is off', () => {
		const url = 'https://open.spotify.com/track/abc?si=xyz';
		expect(spotifyUrl(url, false)).toBe(url);
	});

	it('appends autoplay=true when enabled', () => {
		expect(spotifyUrl('https://open.spotify.com/track/abc', true)).toBe(
			'https://open.spotify.com/track/abc?autoplay=true'
		);
	});

	it('keeps existing query parameters when enabling autoplay', () => {
		const url = spotifyUrl('https://open.spotify.com/track/abc?si=xyz', true);
		const parsed = new URL(url);
		expect(parsed.searchParams.get('si')).toBe('xyz');
		expect(parsed.searchParams.get('autoplay')).toBe('true');
	});

	it('does not duplicate an existing autoplay parameter', () => {
		expect(spotifyUrl('https://open.spotify.com/track/abc?autoplay=true', true)).toBe(
			'https://open.spotify.com/track/abc?autoplay=true'
		);
	});

	it('returns empty and invalid URLs unchanged', () => {
		expect(spotifyUrl('', true)).toBe('');
		expect(spotifyUrl('not a url', true)).toBe('not a url');
	});
});
