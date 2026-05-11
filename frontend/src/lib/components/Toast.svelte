<!--
  Toast.svelte — notificação individual.
  Renderizada por ToastContainer; recebe o objeto t (item da fila).
-->
<script>
	import { toast as toastStore } from '$lib/stores/toasts.js';
	export let t;

	const ICONS = {
		success: '✓',
		error:   '!',
		warn:    '⚠',
		info:    'ℹ',
		confirm: '?'
	};
</script>

<div class="toast" data-kind={t.kind} role={t.kind === 'error' ? 'alert' : 'status'}>
	<div class="icon mono" aria-hidden="true">{ICONS[t.kind] || ''}</div>

	<div class="body">
		{#if t.title}
			<div class="title">{t.title}</div>
		{/if}
		{#if t.message}
			<div class="msg">{t.message}</div>
		{/if}

		{#if t.kind === 'confirm'}
			<div class="actions">
				<button class="btn btn-ghost" on:click={t.onCancel}>{t.cancelLabel}</button>
				<button
					class="btn"
					class:btn-danger={t.destructive}
					class:btn-primary={!t.destructive}
					on:click={t.onConfirm}
				>
					{t.confirmLabel}
				</button>
			</div>
		{/if}
	</div>

	{#if t.kind !== 'confirm'}
		<button class="close" aria-label="Fechar" on:click={() => toastStore.dismiss(t.id)}>×</button>
	{/if}
</div>

<style>
	.toast {
		position: relative;
		display: grid;
		grid-template-columns: 28px 1fr auto;
		gap: var(--sp-3);
		align-items: start;
		padding: var(--sp-3) var(--sp-4);
		min-width: 280px;
		max-width: 420px;
		background: var(--bg-2);
		border: 1px solid var(--line-2);
		border-left: 3px solid var(--fg-2);
		border-radius: var(--radius);
		box-shadow: 0 12px 32px -8px rgba(0, 0, 0, 0.4);
		font-size: 13px;
		animation: slide-in 200ms ease;
	}

	.toast[data-kind="success"] { border-left-color: var(--ok);   }
	.toast[data-kind="error"]   { border-left-color: var(--err);  }
	.toast[data-kind="warn"]    { border-left-color: var(--warn); }
	.toast[data-kind="info"]    { border-left-color: var(--info); }
	.toast[data-kind="confirm"] { border-left-color: var(--accent); }

	@keyframes slide-in {
		from {
			transform: translateX(20px);
			opacity: 0;
		}
		to {
			transform: translateX(0);
			opacity: 1;
		}
	}

	.icon {
		display: grid;
		place-items: center;
		width: 28px;
		height: 28px;
		border-radius: 50%;
		font-weight: 700;
		font-size: 14px;
		background: var(--bg-3);
		color: var(--fg-1);
	}

	.toast[data-kind="success"] .icon { background: var(--accent-bg); color: var(--ok); }
	.toast[data-kind="error"]   .icon { background: rgba(242, 91, 74, 0.12); color: var(--err); }
	.toast[data-kind="warn"]    .icon { background: rgba(242, 193, 78, 0.12); color: var(--warn); }
	.toast[data-kind="info"]    .icon { background: rgba(91, 190, 242, 0.12); color: var(--info); }
	.toast[data-kind="confirm"] .icon { background: var(--accent-bg); color: var(--accent); }

	.body {
		min-width: 0;
		display: flex;
		flex-direction: column;
		gap: 4px;
	}

	.title {
		font-weight: 600;
		color: var(--fg-0);
		line-height: 1.3;
		word-wrap: break-word;
	}

	.msg {
		color: var(--fg-2);
		font-size: 12px;
		line-height: 1.45;
		word-wrap: break-word;
	}

	.actions {
		display: flex;
		gap: var(--sp-2);
		margin-top: var(--sp-2);
		justify-content: flex-end;
	}

	.actions .btn {
		padding: 5px 12px;
		font-size: 12px;
	}

	.close {
		background: transparent;
		border: 0;
		color: var(--fg-3);
		font-size: 18px;
		line-height: 1;
		cursor: pointer;
		padding: 0;
		width: 22px;
		height: 22px;
	}
	.close:hover { color: var(--fg-0); }
</style>
