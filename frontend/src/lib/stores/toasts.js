// src/lib/stores/toasts.js
//
// Sistema de notificações tipo "toast". Substitui alert()/confirm() do navegador
// por uma fila visual que aparece no canto inferior direito.
//
// Uso:
//   import { toast } from '$lib/stores/toasts.js';
//   toast.success('Salvo!');
//   toast.error('Falha ao salvar', 'detalhes do erro...');
//   toast.warn('Disco com 90% de uso');
//   toast.info('Backup iniciado');
//
//   // Confirmação (substitui confirm() do browser):
//   const ok = await toast.confirm('Excluir arquivo?', { destructive: true });
//   if (ok) doIt();
import { writable } from 'svelte/store';

let nextId = 1;

function createToastStore() {
	const { subscribe, update } = writable([]);

	function push(toast) {
		const id = nextId++;
		const t = {
			id,
			kind: 'info',     // 'success' | 'error' | 'warn' | 'info' | 'confirm'
			title: '',
			message: '',
			timeoutMs: 4500,
			...toast
		};
		update((arr) => [...arr, t]);
		// Auto-dismiss exceto pra confirm (que aguarda interação).
		if (t.kind !== 'confirm' && t.timeoutMs > 0) {
			setTimeout(() => dismiss(id), t.timeoutMs);
		}
		return id;
	}

	function dismiss(id) {
		update((arr) => arr.filter((t) => t.id !== id));
	}

	return {
		subscribe,
		dismiss,

		success: (title, message = '', opts = {}) =>
			push({ kind: 'success', title, message, ...opts }),

		error: (title, message = '', opts = {}) =>
			push({ kind: 'error', title, message, timeoutMs: 7000, ...opts }),

		warn: (title, message = '', opts = {}) =>
			push({ kind: 'warn', title, message, ...opts }),

		info: (title, message = '', opts = {}) =>
			push({ kind: 'info', title, message, ...opts }),

		// confirm devolve uma Promise<boolean>.
		// Uso: const ok = await toast.confirm('Apagar?', { destructive: true });
		confirm: (title, opts = {}) => {
			return new Promise((resolve) => {
				const id = push({
					kind: 'confirm',
					title,
					message: opts.message || '',
					destructive: opts.destructive || false,
					confirmLabel: opts.confirmLabel || (opts.destructive ? 'Excluir' : 'Confirmar'),
					cancelLabel: opts.cancelLabel || 'Cancelar',
					timeoutMs: 0,
					onConfirm: () => { dismiss(id); resolve(true); },
					onCancel: () => { dismiss(id); resolve(false); }
				});
			});
		}
	};
}

export const toast = createToastStore();
