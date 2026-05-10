<script>
	import { goto } from '$app/navigation';
	import { page } from '$app/stores';
	import { appRegistry } from '../../modules/apps/registry.js';
	export let onLauncher = () => {};
	function isActive(route) { return $page.url.pathname === route || $page.url.pathname.startsWith(route + '/'); }
</script>

<nav class="dock" aria-label="Aplicativos fixados">
	<button class="launcher" on:click={onLauncher} aria-label="Abrir launcher">⌘</button>
	<div class="sep"></div>
	{#each appRegistry.slice(0, 6) as app}
		<button class="app-btn" class:running={isActive(app.route)} title={app.title} on:click={() => goto(app.route)}>
			<span>{app.icon}</span>
		</button>
	{/each}
</nav>

<style>
	.dock {
		position: fixed;
		z-index: 1000;
		left: 50%;
		bottom: 22px;
		transform: translateX(-50%);
		display: flex;
		align-items: center;
		gap: 10px;
		padding: 12px;
		border-radius: 30px;
		background: rgba(9,14,25,.72);
		border: 1px solid rgba(255,255,255,.11);
		backdrop-filter: blur(30px) saturate(160%);
		box-shadow: 0 18px 55px rgba(0,0,0,.38), inset 0 1px 0 rgba(255,255,255,.08);
	}

	.app-btn, .launcher {
		position: relative;
		width: 56px;
		height: 56px;
		display: grid;
		place-items: center;
		border-radius: 20px;
		background: rgba(255,255,255,.05);
		color: var(--os-text);
		font-size: 22px;
		transition: transform .18s cubic-bezier(.2,.8,.2,1), background .18s ease;
	}

	.app-btn:hover, .launcher:hover {
		transform: translateY(-8px) scale(1.08);
		background: rgba(255,255,255,.12);
	}

	.launcher {
		background: linear-gradient(135deg, rgba(124,242,91,.18), rgba(102,217,255,.14));
		color: var(--os-accent);
	}

	.running::after {
		content: '';
		position: absolute;
		bottom: 6px;
		width: 6px;
		height: 6px;
		border-radius: 50%;
		background: var(--os-accent);
		box-shadow: 0 0 14px var(--os-accent);
	}

	.sep {
		width: 1px;
		height: 38px;
		background: rgba(255,255,255,.08);
		margin: 0 4px;
	}
</style>
