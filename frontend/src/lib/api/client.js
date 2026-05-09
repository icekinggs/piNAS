// src/lib/api/client.js
//
// Cliente HTTP que:
// - injeta access token em Authorization
// - tenta refresh automático em 401
// - redireciona pro login se refresh também falhar
import { auth } from '$lib/stores/auth.js';
import { get } from 'svelte/store';
import { goto } from '$app/navigation';

const BASE = '/api/v1';

let refreshPromise = null;

async function tryRefresh() {
	if (refreshPromise) return refreshPromise;
	refreshPromise = (async () => {
		const res = await fetch(`${BASE}/auth/refresh`, {
			method: 'POST',
			credentials: 'include'
		});
		if (!res.ok) {
			auth.clear();
			throw new Error('refresh failed');
		}
		const data = await res.json();
		auth.set({
			accessToken: data.access_token,
			expiresAt: data.expires_at,
			user: data.user
		});
		return data.access_token;
	})().finally(() => {
		refreshPromise = null;
	});
	return refreshPromise;
}

/**
 * api(path, options) — fetch wrapper.
 * options: { method, body (object — JSON), query, raw (não JSON, retorna Response) }
 */
export async function api(path, options = {}) {
	const { method = 'GET', body, query, raw, headers = {}, signal, formData } = options;

	let url = BASE + path;
	if (query) {
		const qs = new URLSearchParams(query).toString();
		if (qs) url += '?' + qs;
	}

	const doFetch = async () => {
		const tok = get(auth)?.accessToken;
		const h = { ...headers };
		if (tok) h['Authorization'] = `Bearer ${tok}`;

		const init = { method, credentials: 'include', headers: h, signal };
		if (formData) {
			init.body = formData; // browser seta Content-Type com boundary
		} else if (body !== undefined) {
			h['Content-Type'] = 'application/json';
			init.body = JSON.stringify(body);
		}
		return fetch(url, init);
	};

	let res = await doFetch();

	// Tentativa de refresh em 401, exceto no próprio endpoint de auth.
	if (res.status === 401 && !path.startsWith('/auth/')) {
		try {
			await tryRefresh();
			res = await doFetch();
		} catch {
			goto('/login');
			throw new Error('não autenticado');
		}
	}

	if (raw) return res;

	if (!res.ok) {
		let msg = res.statusText;
		try {
			const j = await res.json();
			msg = j.error || msg;
		} catch (_) {}
		const err = new Error(msg);
		err.status = res.status;
		throw err;
	}
	if (res.status === 204) return null;
	const ct = res.headers.get('content-type') || '';
	if (ct.includes('application/json')) return res.json();
	return res.text();
}
