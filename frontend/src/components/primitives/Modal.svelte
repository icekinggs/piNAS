<script>
	export let open = false;
	export let title = '';
	export let onClose = function() {};
</script>

{#if open}
	<div class="overlay" role="presentation" on:click={onClose}></div>
	<section class="modal os-glass os-panel" role="dialog" aria-modal="true">
		<header>
			<strong>{title}</strong>
			<button type="button" on:click={onClose}>×</button>
		</header>
		<div class="body"><slot /></div>
	</section>
{/if}

<style>
	.overlay { position: fixed; inset: 0; z-index: 2000; background: rgba(0,0,0,.45); }
	.modal { position: fixed; z-index: 2001; left: 50%; top: 50%; transform: translate(-50%, -50%); width: min(560px, calc(100vw - 32px)); overflow: hidden; }
	header { height: 52px; display: flex; align-items: center; justify-content: space-between; padding: 0 16px; border-bottom: 1px solid var(--os-border); }
	header button { font-size: 22px; color: var(--os-text-muted); }
	.body { padding: 18px; }
</style>
