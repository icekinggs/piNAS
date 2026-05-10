<script>
	import { appRegistry } from '../../modules/apps/registry.js';
	import { WindowManager } from '../windows/WindowManager.js';
	export let open = false;
	export let onClose = function() {};

	function launch(id) {
		WindowManager.open(id);
		onClose();
	}
</script>

{#if open}
	<div class="launcher os-glass">
		<div class="header">
			<strong>Aplicativos</strong>
			<button on:click={onClose}>Fechar</button>
		</div>
		<div class="grid">
			{#each appRegistry as app}
				<button class="app" on:click={() => launch(app.id)}>
					<span>{app.icon}</span>
					<small>{app.title}</small>
				</button>
			{/each}
		</div>
	</div>
{/if}

<style>
	.launcher { position: fixed; z-index: 1100; left: 50%; bottom: 96px; transform: translateX(-50%); width: min(680px, calc(100vw - 32px)); padding: 18px; border-radius: 24px; }
	.header { display: flex; align-items: center; justify-content: space-between; margin-bottom: 16px; }
	.header button { color: var(--os-text-muted); font-size: 12px; }
	.grid { display: grid; grid-template-columns: repeat(auto-fill, minmax(96px, 1fr)); gap: 12px; }
	.app { display: grid; gap: 8px; place-items: center; min-height: 92px; border-radius: 18px; background: rgba(255,255,255,.06); }
	.app:hover { background: rgba(255,255,255,.12); }
	.app span { font-size: 28px; }
	.app small { color: var(--os-text-muted); }
</style>
