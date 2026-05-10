<script>
	import Wallpaper from './Wallpaper.svelte';
	import Topbar from './Topbar.svelte';
	import Dock from './Dock.svelte';
</script>

<div class="desktop-shell">
	<Wallpaper />
	<Topbar />

	<div class="desktop-content">
		<slot />
	</div>

	<Dock />
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
