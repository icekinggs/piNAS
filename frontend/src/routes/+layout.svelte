<script>
	import '../app.css';
	import { onMount } from 'svelte';
	import { page } from '$app/stores';
	import { goto } from '$app/navigation';
	import { auth } from '$lib/stores/auth.js';
	import { api } from '$lib/api/client.js';
	import Desktop from '../core/desktop/Desktop.svelte';

	let booting = true;

	onMount(async () => {
		if ($page.url.pathname === '/login') {
			booting = false;
			return;
		}
		try {
			const me = await api('/auth/me');
			auth.set({ ...($auth || {}), user: me });
		} catch (_) {
			goto('/login');
		} finally {
			booting = false;
		}
	});

	$: isLogin = $page.url.pathname === '/login';
</script>

<svelte:head>
	<title>PiNAS Web OS</title>
</svelte:head>

{#if booting && !isLogin}
	<div class="boot mono pulse">PiNAS · iniciando web os…</div>
{:else if isLogin}
	<slot />
{:else}
	<Desktop>
		<slot />
	</Desktop>
{/if}

<style>
	.boot {
		display: grid;
		place-items: center;
		min-height: 100vh;
		font-size: 13px;
		letter-spacing: 0.12em;
		color: var(--fg-2);
	}
</style>
