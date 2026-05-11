<!--
  ToastContainer.svelte — fixo no canto inferior direito, renderiza a fila.
  Adicione UMA ÚNICA VEZ no +layout.svelte raiz.
-->
<script>
	import { toast } from '$lib/stores/toasts.js';
	import Toast from './Toast.svelte';
</script>

<div class="container" aria-live="polite" aria-atomic="false">
	{#each $toast as t (t.id)}
		<Toast {t} />
	{/each}
</div>

<style>
	.container {
		position: fixed;
		bottom: var(--sp-4);
		right: var(--sp-4);
		z-index: 9999;
		display: flex;
		flex-direction: column;
		gap: var(--sp-2);
		pointer-events: none;
	}

	/* Toasts individuais re-habilitam pointer-events */
	.container :global(.toast) {
		pointer-events: auto;
	}

	@media (max-width: 600px) {
		.container {
			left: var(--sp-3);
			right: var(--sp-3);
			bottom: var(--sp-3);
		}
		.container :global(.toast) {
			max-width: 100%;
		}
	}
</style>
