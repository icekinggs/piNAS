<script>
	import { appRegistry } from '../../modules/apps/registry.js';
	import { WindowManager } from '../windows/WindowManager.js';
	import { windows } from '../windows/WindowStore.js';
	export let onLauncher = () => {};
	function isOpen(id) { return $windows.some((w) => w.appId === id); }
</script>

<nav class="dock os-glass" aria-label="Aplicativos fixados">
	<button class="launcher" on:click={onLauncher} aria-label="Abrir launcher">⌘</button>
	<div class="sep"></div>
	{#each appRegistry.slice(0, 6) as app}
		<button class:running={isOpen(app.id)} title={app.title} on:click={() => WindowManager.open(app.id)}>
			<span>{app.icon}</span>
		</button>
	{/each}
</nav>

<style>
	.dock { position: fixed; z-index: 1000; left: 50%; bottom: 14px; transform: translateX(-50%); height: var(--os-dock-h); display: flex; align-items: center; gap: 8px; padding: 10px; border-radius: 24px; }
	button { position: relative; width: 48px; height: 48px; display: grid; place-items: center; border-radius: 16px; background: rgba(255,255,255,.06); color: var(--os-text); font-size: 20px; transition: transform 140ms ease, background 140ms; }
	button:hover { transform: translateY(-4px); background: rgba(255,255,255,.12); }
	button.running::after { content: ''; position: absolute; bottom: 4px; width: 5px; height: 5px; border-radius: 50%; background: var(--os-accent); }
	.launcher { color: var(--os-accent); }
	.sep { width: 1px; height: 32px; background: var(--os-border); margin: 0 2px; }
</style>
