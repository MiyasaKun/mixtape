import { describe, expect, it } from 'vitest';
import { resolveTheme } from './theme.js';

describe('resolveTheme', () => {
	it('prefers the saved choice over the system preference', () => {
		expect(resolveTheme('light', true)).toBe('light');
		expect(resolveTheme('dark', false)).toBe('dark');
	});

	it('falls back to the system preference when nothing is saved', () => {
		expect(resolveTheme(null, true)).toBe('dark');
		expect(resolveTheme(null, false)).toBe('light');
	});

	it('ignores saved values that are neither theme', () => {
		expect(resolveTheme('midnight', true)).toBe('dark');
	});
});
