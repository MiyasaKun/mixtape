import { writable } from 'svelte/store';
import type { Writable } from 'svelte/store';

export type Theme = 'light' | 'dark';

const STORAGE_KEY = 'mixtape-theme';

/** Current theme: 'light' or 'dark'. */
export const theme: Writable<Theme> = writable('light');

/** Uses the saved choice when it names a real theme, else the system preference. */
export function resolveTheme(saved: string | null, prefersDark: boolean): Theme {
	return saved === 'dark' || saved === 'light' ? saved : prefersDark ? 'dark' : 'light';
}

/** Reads the saved or system theme and applies it. */
export function initTheme(): Theme {
	let saved: string | null = null;
	try {
		saved = localStorage.getItem(STORAGE_KEY);
	} catch {
		saved = null;
	}

	const prefersDark = window.matchMedia('(prefers-color-scheme: dark)').matches;
	const value = resolveTheme(saved, prefersDark);

	apply(value);
	return value;
}

/** Persists and applies the chosen theme. */
export function toggleTheme(): void {
	let next: Theme = 'light';
	theme.update((current) => {
		next = current === 'dark' ? 'light' : 'dark';
		return next;
	});
	try {
		localStorage.setItem(STORAGE_KEY, next);
	} catch {
		// Storage can be unavailable in private mode; the theme still applies.
	}
	apply(next);
}

function apply(value: Theme): void {
	const dark = value === 'dark';
	document.documentElement.classList.toggle('dark', dark);
	document.documentElement.style.colorScheme = dark ? 'dark' : 'light';
	const meta = document.querySelector('meta[name="theme-color"]');
	if (meta) meta.setAttribute('content', dark ? '#1e2230' : '#e0e5ec');
	theme.set(value);
}
