// src/lib/stores/auth.js
import { writable } from 'svelte/store';
import { browser } from '$app/environment';

const STORAGE_KEY = 'pinas.auth';

function load() {
	if (!browser) return null;
	try {
		const raw = sessionStorage.getItem(STORAGE_KEY);
		return raw ? JSON.parse(raw) : null;
	} catch {
		return null;
	}
}

function save(value) {
	if (!browser) return;
	if (value) sessionStorage.setItem(STORAGE_KEY, JSON.stringify(value));
	else sessionStorage.removeItem(STORAGE_KEY);
}

function createAuth() {
	const initial = load();
	const { subscribe, set, update } = writable(initial);

	return {
		subscribe,
		set: (v) => {
			save(v);
			set(v);
		},
		clear: () => {
			save(null);
			set(null);
		}
	};
}

export const auth = createAuth();
