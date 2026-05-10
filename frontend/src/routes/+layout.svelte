<script>
	import '../app.css';
	import { onMount } from 'svelte';
	import { page } from '$app/stores';
	import { goto } from '$app/navigation';
	import { auth } from '$lib/stores/auth.js';
	import { api } from '$lib/api/client.js';
	import Sidebar from '$lib/components/Sidebar.svelte';

	let booting = true;

	onMount(async () => {
		// Tenta restaurar sessão chamando /me; se falhar via fetch, o cliente
		// vai tentar refresh automaticamente.
		if ($page.url.pathname === '/login') {
			booting = false;
			return;
		}
		try {
			const me = await api('/auth/me');
			auth.set({
				...($auth || {}),
				user: me
			});
		} catch (_) {
			goto('/login');
		} finally {
			booting = false;
		}
	});

	$: isLogin = $page.url.pathname === '/login';
</script>

<svelte:head>
	<title>PiNAS</title>
</svelte:head>

{#if booting && !isLogin}
	<div class="boot mono pulse">PiNAS · iniciando…</div>
{:else if isLogin}
	<slot />
{:else}
	<div class="shell">
		<Sidebar />
		<main class="main">
			<slot />
		</main>
	</div>
{/if}

<style>
	.shell {
		display: grid;
		grid-template-columns: var(--sidebar-w) 1fr;
		grid-template-areas: 'sidebar main';
		min-height: 100vh;
	}

	.main {
		grid-area: main;
		padding: var(--sp-5) var(--sp-6);
		overflow-x: auto;
	}

	.boot {
		display: grid;
		place-items: center;
		min-height: 100vh;
		font-size: 13px;
		letter-spacing: 0.12em;
		color: var(--fg-2);
	}

	@media (max-width: 720px) {
		.shell {
			grid-template-columns: 1fr;
			grid-template-areas: 'main';
		}
		:global(.sidebar) {
			display: none !important;
		}
		.main { padding: var(--sp-3); }
	}
</style>
