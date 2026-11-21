import { describe, expect, it, vi } from 'vitest';

vi.mock('../../paraglide/messages.js', () => ({
	m: {
		playlists_track_count_one: () => 'ONE',
		playlists_track_count_other: (params: { count: number }) => `OTHER:${params.count}`
	}
}));

const { pluralCategory, trackCount } = await import('./plural.js');




describe('pluralCategory', () => {
	it('selects "one" for 1 in English', () => {
		expect(pluralCategory(1, 'en')).toBe('one');
	});

	it('selects "other" for 0 and counts above one in English', () => {
		expect(pluralCategory(0, 'en')).toBe('other');
		expect(pluralCategory(2, 'en')).toBe('other');
		expect(pluralCategory(1284, 'en')).toBe('other');
	});

	it('follows the German rule set', () => {
		expect(pluralCategory(1, 'de')).toBe('one');
		expect(pluralCategory(1.5, 'de')).toBe('other');
	});
});

describe('trackCount', () => {
	it('renders the singular message for exactly 1', () => {
		expect(trackCount(1, 'en')).toBe('ONE');
	});

	it('renders the plural message for 0 and any count above one', () => {
		expect(trackCount(0, 'en')).toBe('OTHER:0');
		expect(trackCount(234, 'en')).toBe('OTHER:234');
	});
});
