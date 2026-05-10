<script>
	import { windows } from '../windows/WindowStore.js';
	import Wallpaper from './Wallpaper.svelte';
	import Topbar from './Topbar.svelte';
	import Dock from './Dock.svelte';
	import Launcher from './Launcher.svelte';
	import Window from '../windows/Window.svelte';

	let launcherOpen = false;
</script>

<div class="desktop-shell">
	<Wallpaper />
	<Topbar />

	<div class="desktop-content">
		<slot />
	</div>

	{#each $windows as item (item.id)}
		<Window item={item} />
	{/each}

	<Launcher open={launcherOpen} onClose={() => launcherOpen = false} />
	<Dock onLauncher={() => launcherOpen = !launcherOpen} />
</div>

<style>
	.desktop-shell {
		position: relative;
		width: 100%;
		height: 100vh;
		overflow: hidden;
		background: var(--os-bg);
	}

	.desktop-content {
		position: relative;
		z-index: 2;
		height: 100%;
		padding-top: var(--os-topbar-h);
		padding-bottom: var(--os-dock-h);
		overflow: auto;
	}
</style>
