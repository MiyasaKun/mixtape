import { m } from '../../paraglide/messages.js';

/**
 * Maps a count to its plural category for `locale` using the same rules the
 * translations follow (`playlists_track_count_{one,other}` in messages/*).
 */
export function pluralCategory(count: number, locale: string): 'other' | 'one' {
	return new Intl.PluralRules(locale).select(count) as 'other' | 'one';
}

/** The localized, correctly pluralized "N tracks" string for a count. */
export function trackCount(count: number, locale: string): string {
	return pluralCategory(count, locale) === 'one'
		? m.playlists_track_count_one({ count })
		: m.playlists_track_count_other({ count });
}
