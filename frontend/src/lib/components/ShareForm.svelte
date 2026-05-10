<!--
  ShareForm.svelte — formulário reutilizável de share Samba.
  Usado em modo "criar" (sem prop initial) ou "editar" (com initial).
-->
<script>
	import { createEventDispatcher } from 'svelte';

	/** @type {import('$lib/types').Share | null} */
	export let initial = null;
	export let availableUsers = []; // array de strings (usernames SMB)

	const dispatch = createEventDispatcher();

	let name        = initial?.name        ?? '';
	let path        = initial?.path        ?? '/srv/pinas/data/';
	let comment     = initial?.comment     ?? '';
	let browseable  = initial?.browseable  ?? true;
	let readOnly    = initial?.read_only   ?? false;
	let guestOk     = initial?.guest_ok    ?? false;
	let validUsers  = new Set(initial?.valid_users ?? []);
	let createMask  = initial?.create_mask     ?? '0664';
	let directoryMask = initial?.directory_mask ?? '0775';
	let forceUser   = initial?.force_user  ?? '';

	let saving = false;
	let error = '';

	$: isEdit = initial !== null;

	function toggleUser(u) {
		const next = new Set(validUsers);
		if (next.has(u)) next.delete(u);
		else next.add(u);
		validUsers = next;
	}

	async function submit() {
		error = '';
		if (!/^[a-zA-Z0-9_-]{1,32}$/.test(name)) {
			error = 'Nome inválido (letras, números, _, -, máximo 32)';
			return;
		}
		if (!path.startsWith('/')) {
			error = 'Path deve ser absoluto (começar com /)';
			return;
		}

		saving = true;
		const payload = {
			name,
			path,
			comment,
			browseable,
			read_only: readOnly,
			guest_ok: guestOk,
			valid_users: Array.from(validUsers),
			create_mask: createMask,
			directory_mask: directoryMask,
			force_user: forceUser || undefined
		};
		try {
			dispatch('submit', payload);
		} finally {
			saving = false;
		}
	}

	function cancel() {
		dispatch('cancel');
	}
</script>

<form class="form" on:submit|preventDefault={submit}>
	<div class="row">
		<div class="field">
			<label class="lbl uppercase-tag" for="sh-name">nome do share</label>
			<input
				id="sh-name"
				class="input"
				type="text"
				required
				maxlength="32"
				bind:value={name}
				disabled={isEdit}
				placeholder="ex: familia"
			/>
			<span class="hint">Aparece como \\\\pinas\\<strong>{name || '...'}</strong></span>
		</div>

		<div class="field">
			<label class="lbl uppercase-tag" for="sh-path">path no host</label>
			<input
				id="sh-path"
				class="input"
				type="text"
				required
				bind:value={path}
				placeholder="/home/gustavo/Documents"
			/>
			<span class="hint">Caminho absoluto. Será criado se não existir.</span>
		</div>
	</div>

	<div class="field">
		<label class="lbl uppercase-tag" for="sh-comment">descrição</label>
		<input id="sh-comment" class="input" type="text" maxlength="100" bind:value={comment} placeholder="(opcional)" />
	</div>

	<fieldset class="checkboxes">
		<legend class="lbl uppercase-tag">flags</legend>
		<label class="check">
			<input type="checkbox" bind:checked={browseable} />
			<span>Visível na rede (browseable)</span>
		</label>
		<label class="check">
			<input type="checkbox" bind:checked={readOnly} />
			<span>Somente leitura</span>
		</label>
		<label class="check warn">
			<input type="checkbox" bind:checked={guestOk} />
			<span>Permitir acesso anônimo (não recomendado)</span>
		</label>
	</fieldset>

	<fieldset class="users-fs">
		<legend class="lbl uppercase-tag">usuários permitidos</legend>
		{#if availableUsers.length === 0}
			<p class="empty mono">
				Nenhum usuário SMB cadastrado.
				Crie usuários na aba "Usuários" antes de restringir o share.
			</p>
		{:else}
			<div class="user-grid">
				{#each availableUsers as u}
					<label class="check user-check">
						<input
							type="checkbox"
							checked={validUsers.has(u)}
							on:change={() => toggleUser(u)}
						/>
						<span class="mono">{u}</span>
					</label>
				{/each}
			</div>
			<p class="hint">
				Se vazio, qualquer usuário SMB cadastrado pode acessar (sujeito ao 'guest ok' acima).
			</p>
		{/if}
	</fieldset>

	<details>
		<summary class="lbl uppercase-tag">avançado</summary>
		<div class="row">
			<div class="field">
				<label class="lbl uppercase-tag" for="sh-cmask">máscara de arquivos</label>
				<input id="sh-cmask" class="input mono" type="text" pattern="0[0-7]{'{3}'}" bind:value={createMask} />
			</div>
			<div class="field">
				<label class="lbl uppercase-tag" for="sh-dmask">máscara de pastas</label>
				<input id="sh-dmask" class="input mono" type="text" pattern="0[0-7]{'{3}'}" bind:value={directoryMask} />
			</div>
			<div class="field">
				<label class="lbl uppercase-tag" for="sh-force">force user (opcional)</label>
				<input id="sh-force" class="input mono" type="text" bind:value={forceUser} placeholder="(deixe vazio)" />
			</div>
		</div>
	</details>

	{#if error}
		<div class="error mono">{error}</div>
	{/if}

	<div class="actions">
		<button type="button" class="btn btn-ghost" on:click={cancel}>Cancelar</button>
		<button type="submit" class="btn btn-primary" disabled={saving}>
			{saving ? 'salvando…' : isEdit ? '> salvar' : '> criar share'}
		</button>
	</div>
</form>

<style>
	.form {
		display: flex;
		flex-direction: column;
		gap: var(--sp-4);
	}

	.row {
		display: grid;
		grid-template-columns: 1fr 1fr;
		gap: var(--sp-3);
	}
	@media (max-width: 700px) {
		.row { grid-template-columns: 1fr; }
	}

	.field {
		display: flex;
		flex-direction: column;
		gap: 6px;
	}

	.lbl { font-size: 11px; }

	.hint {
		font-size: 11px;
		color: var(--fg-3);
		font-family: var(--font-mono);
	}

	.checkboxes,
	.users-fs {
		border: 1px solid var(--line-1);
		border-radius: var(--radius);
		padding: var(--sp-3);
		display: flex;
		flex-direction: column;
		gap: var(--sp-2);
	}
	.checkboxes legend,
	.users-fs legend {
		padding: 0 6px;
		color: var(--fg-2);
	}

	.check {
		display: flex;
		align-items: center;
		gap: 8px;
		cursor: pointer;
		font-size: 13px;
		user-select: none;
	}
	.check input {
		accent-color: var(--accent);
		cursor: pointer;
	}
	.check.warn span { color: var(--warn); }

	.user-grid {
		display: grid;
		grid-template-columns: repeat(auto-fill, minmax(160px, 1fr));
		gap: var(--sp-2);
	}

	.user-check {
		padding: 4px 8px;
		border-radius: var(--radius);
		background: var(--bg-3);
	}
	.user-check:hover { background: var(--bg-4); }

	.empty {
		font-size: 12px;
		color: var(--fg-3);
		padding: var(--sp-3);
		text-align: center;
	}

	details {
		background: var(--bg-1);
		border: 1px solid var(--line-1);
		border-radius: var(--radius);
		padding: var(--sp-3);
	}
	details > summary {
		cursor: pointer;
		padding: 4px 0;
		list-style: none;
		display: flex;
		align-items: center;
		gap: 6px;
	}
	details > summary::before {
		content: '▶';
		font-size: 9px;
		color: var(--fg-3);
		transition: transform 120ms;
	}
	details[open] > summary::before {
		transform: rotate(90deg);
	}
	details > .row { margin-top: var(--sp-3); }

	.error {
		padding: var(--sp-2) var(--sp-3);
		background: rgba(242, 91, 74, 0.06);
		border: 1px solid rgba(242, 91, 74, 0.3);
		border-radius: var(--radius);
		color: var(--err);
		font-size: 12px;
	}

	.actions {
		display: flex;
		gap: var(--sp-2);
		justify-content: flex-end;
		padding-top: var(--sp-2);
		border-top: 1px solid var(--line-1);
	}
</style>
