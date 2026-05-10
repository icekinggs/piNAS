<script>
	import { WindowManager } from './WindowManager.js';
	export let item;
</script>

{#if item.state !== 'minimized'}
	<section class="window os-glass" style={`left:${item.x}px;top:${item.y}px;width:${item.width}px;height:${item.height}px;z-index:${item.zIndex};`}>
		<header class="titlebar" on:mousedown={() => WindowManager.focus(item.id)}>
			<div class="meta">
				<span>{item.icon || '◉'}</span>
				<strong>{item.title}</strong>
			</div>
			<div class="actions">
				<button on:click={() => WindowManager.minimize(item.id)}>—</button>
				<button on:click={() => WindowManager.close(item.id)}>×</button>
			</div>
		</header>
		<div class="content">
			<p>Aplicação: {item.appId}</p>
		</div>
	</section>
{/if}

<style>
	.window { position: absolute; overflow: hidden; border-radius: 22px; }
	.titlebar { height: 52px; display: flex; align-items: center; justify-content: space-between; padding: 0 14px; border-bottom: 1px solid var(--os-border); }
	.meta { display: flex; align-items: center; gap: 10px; }
	.actions { display: flex; gap: 8px; }
	.actions button { width: 30px; height: 30px; border-radius: 50%; background: rgba(255,255,255,.08); }
	.content { padding: 18px; color: var(--os-text-muted); }
</style>
