<script>
	import { onMount } from 'svelte';
	import { goto } from '$app/navigation';
	import { auth } from '$lib/stores/auth.js';
	import { sambaApi } from '$lib/api/samba.js';
	import { formatDate } from '$lib/utils/format.js';
	import { toast } from '$lib/stores/toasts.js';
	import ShareForm from '$lib/components/ShareForm.svelte';

	let tab = 'shares'; // 'shares' | 'users' | 'status'

	let shares = [];
	let users = [];
	let status = null;

	let loading = false;
	let error = '';

	// modal de criar/editar share
	let editingShare = null;        // null | object (existing) | 'new'
	// modal de criar usuário
	let creatingUser = false;
	let newUsername = '';
	let newPassword = '';
	let userBusy = false;

	async function loadAll() {
		loading = true;
		error = '';
		try {
			const state = await sambaApi.getState();
			shares = state.shares || [];
			users  = state.users  || [];
			// pega status separado (arquivo gerado pelo sync script)
			try {
				const statusFile = await fetch('/api/v1/samba/state', { credentials: 'include' });
			} catch {}
		} catch (e) {
			error = e.message;
		} finally {
			loading = false;
		}
	}

	onMount(() => {
		if ($auth?.user?.role !== 'admin') {
			goto('/dashboard');
			return;
		}
		loadAll();
	});

	// ----- shares -----

	function openNewShare() { editingShare = 'new'; }
	function openEditShare(sh) { editingShare = { ...sh }; }
	function closeShareModal() { editingShare = null; }

	async function onShareSubmit(ev) {
		const payload = ev.detail;
		try {
			if (editingShare === 'new') {
				await sambaApi.createShare(payload);
				toast.success('Share criado', `"${payload.name}" foi criado e está sendo aplicado no Samba.`);
			} else {
				await sambaApi.updateShare(editingShare.name, payload);
				toast.success('Share atualizado', `"${payload.name}" foi salvo.`);
			}
			editingShare = null;
			await loadAll();
		} catch (e) {
			toast.error('Falha ao salvar share', e.message);
		}
	}

	async function deleteShare(sh) {
		const ok = await toast.confirm(`Excluir o share "${sh.name}"?`, {
			message: `A pasta ${sh.path} NÃO será apagada — só o compartilhamento Samba.`,
			destructive: true,
			confirmLabel: 'Excluir share'
		});
		if (!ok) return;
		try {
			await sambaApi.deleteShare(sh.name);
			toast.success('Share removido', `"${sh.name}" foi excluído.`);
			await loadAll();
		} catch (e) { toast.error('Falha ao excluir', e.message); }
	}

	// ----- users -----

	async function createUser() {
		if (!newUsername || newPassword.length < 6) {
			toast.warn('Dados incompletos', 'Username obrigatório, senha mínima 6 caracteres.');
			return;
		}
		userBusy = true;
		try {
			await sambaApi.createUser(newUsername, newPassword);
			toast.success('Usuário SMB criado', `"${newUsername}" pode acessar shares agora.`);
			newUsername = '';
			newPassword = '';
			creatingUser = false;
			await loadAll();
		} catch (e) {
			toast.error('Falha ao criar usuário', e.message);
		} finally {
			userBusy = false;
		}
	}

	async function changeUserPassword(u) {
		// prompt() ainda é nativo — substituir requer um modal de input próprio.
		// Por enquanto mantemos o prompt; toast só pra resposta.
		const p = prompt(`Nova senha SMB para ${u.username}:`);
		if (!p) return;
		if (p.length < 6) {
			toast.warn('Senha curta', 'Mínimo 6 caracteres.');
			return;
		}
		try {
			await sambaApi.setUserPassword(u.username, p);
			toast.success('Senha atualizada', 'O Samba vai aplicar em alguns segundos.');
		} catch (e) { toast.error('Falha ao atualizar senha', e.message); }
	}

	async function toggleUserDisabled(u) {
		try {
			await sambaApi.setUserDisabled(u.username, !u.disabled);
			toast.success(
				u.disabled ? 'Usuário ativado' : 'Usuário desativado',
				`"${u.username}" agora está ${u.disabled ? 'ativo' : 'inativo'}.`
			);
			await loadAll();
		} catch (e) { toast.error('Falha', e.message); }
	}

	async function deleteUser(u) {
		const ok = await toast.confirm(`Excluir usuário SMB "${u.username}"?`, {
			message: 'Não apaga arquivos do disco — apenas remove o usuário do Samba.',
			destructive: true,
			confirmLabel: 'Excluir usuário'
		});
		if (!ok) return;
		try {
			await sambaApi.deleteUser(u.username);
			toast.success('Usuário removido', `"${u.username}" foi excluído.`);
			await loadAll();
		} catch (e) { toast.error('Falha ao excluir', e.message); }
	}

	$: usernameList = users.map((u) => u.username);
</script>

<header class="head">
	<div>
		<h1 class="title">Samba</h1>
		<p class="sub mono">compartilhamentos de rede · SMB / CIFS</p>
	</div>
	<div class="status">
		<span class="pill pill-ok">{shares.length} {shares.length === 1 ? 'share' : 'shares'}</span>
		<span class="pill mono">{users.length} usuário(s)</span>
	</div>
</header>

<nav class="tabs">
	<button class="tab" class:active={tab === 'shares'} on:click={() => tab = 'shares'}>
		<span class="tab-num mono">01</span>
		<span>Compartilhamentos</span>
	</button>
	<button class="tab" class:active={tab === 'users'} on:click={() => tab = 'users'}>
		<span class="tab-num mono">02</span>
		<span>Usuários SMB</span>
	</button>
	<button class="tab" class:active={tab === 'help'} on:click={() => tab = 'help'}>
		<span class="tab-num mono">03</span>
		<span>Como acessar</span>
	</button>
</nav>

{#if error}
	<div class="error mono">erro: {error}</div>
{/if}

{#if loading && !shares.length && !users.length}
	<div class="loading mono pulse">carregando…</div>
{:else if tab === 'shares'}
	<!-- ============ SHARES ============ -->
	{#if editingShare}
		<section class="card form-card">
			<div class="card-header">
				<div class="card-title">
					{editingShare === 'new' ? '+ NOVO COMPARTILHAMENTO' : `EDITAR · ${editingShare.name}`}
				</div>
			</div>
			<div class="card-body">
				<ShareForm
					initial={editingShare === 'new' ? null : editingShare}
					availableUsers={usernameList}
					on:submit={onShareSubmit}
					on:cancel={closeShareModal}
				/>
			</div>
		</section>
	{:else}
		<div class="actions-row">
			<button class="btn btn-primary" on:click={openNewShare}>+ Novo compartilhamento</button>
		</div>

		{#if shares.length === 0}
			<section class="card empty-state">
				<div class="empty-title mono">NENHUM SHARE AINDA</div>
				<div class="empty-sub">
					Crie seu primeiro compartilhamento Samba pra acessar arquivos
					do seu Pi pelo Windows, macOS ou outros Linux na rede local.
				</div>
				<button class="btn btn-primary" on:click={openNewShare}>+ Criar share</button>
			</section>
		{:else}
			<section class="card">
				<table class="t">
					<thead>
						<tr>
							<th>NOME</th>
							<th>PATH</th>
							<th>FLAGS</th>
							<th>USUÁRIOS</th>
							<th></th>
						</tr>
					</thead>
					<tbody>
						{#each shares as sh}
							<tr>
								<td class="mono name-cell">
									<span class="share-icon">📁</span>
									<strong>{sh.name}</strong>
									{#if sh.comment}<div class="comment dim">{sh.comment}</div>{/if}
								</td>
								<td class="mono dim">{sh.path}</td>
								<td>
									<div class="flag-row">
										{#if sh.read_only}<span class="pill">RO</span>{:else}<span class="pill pill-ok">RW</span>{/if}
										{#if sh.browseable}<span class="pill">visível</span>{/if}
										{#if sh.guest_ok}<span class="pill pill-warn">guest</span>{/if}
									</div>
								</td>
								<td class="mono dim">
									{#if sh.valid_users?.length}
										{sh.valid_users.join(', ')}
									{:else}
										<em>todos</em>
									{/if}
								</td>
								<td class="actions">
									<button class="btn btn-ghost" on:click={() => openEditShare(sh)}>editar</button>
									<button class="btn btn-danger" on:click={() => deleteShare(sh)}>✕</button>
								</td>
							</tr>
						{/each}
					</tbody>
				</table>
			</section>
		{/if}
	{/if}

{:else if tab === 'users'}
	<!-- ============ USERS ============ -->
	{#if creatingUser}
		<section class="card form-card">
			<div class="card-header">
				<div class="card-title">+ NOVO USUÁRIO SMB</div>
			</div>
			<form class="card-body create-user" on:submit|preventDefault={createUser}>
				<div class="field">
					<label class="lbl uppercase-tag" for="u-name">username</label>
					<input id="u-name" class="input" type="text" required minlength="1" maxlength="32"
						bind:value={newUsername} placeholder="ex: maria" />
					<span class="hint">Será criado também como usuário Linux (system, sem shell).</span>
				</div>
				<div class="field">
					<label class="lbl uppercase-tag" for="u-pass">senha</label>
					<input id="u-pass" class="input" type="password" required minlength="6" bind:value={newPassword} />
					<span class="hint">Esta é a senha SMB — pode ser diferente da senha do PiNAS.</span>
				</div>
				<div class="actions">
					<button type="button" class="btn btn-ghost" on:click={() => creatingUser = false}>Cancelar</button>
					<button type="submit" class="btn btn-primary" disabled={userBusy}>
						{userBusy ? 'criando…' : '> criar'}
					</button>
				</div>
			</form>
		</section>
	{:else}
		<div class="actions-row">
			<button class="btn btn-primary" on:click={() => creatingUser = true}>+ Novo usuário SMB</button>
		</div>

		{#if users.length === 0}
			<section class="card empty-state">
				<div class="empty-title mono">NENHUM USUÁRIO SMB AINDA</div>
				<div class="empty-sub">
					Usuários SMB são <strong>diferentes</strong> dos usuários do painel PiNAS.
					Crie um usuário aqui para usar no Windows/macOS quando conectar nos shares.
				</div>
				<button class="btn btn-primary" on:click={() => creatingUser = true}>+ Criar usuário</button>
			</section>
		{:else}
			<section class="card">
				<table class="t">
					<thead>
						<tr>
							<th>USERNAME</th>
							<th>STATUS</th>
							<th></th>
						</tr>
					</thead>
					<tbody>
						{#each users as u}
							<tr class:row-disabled={u.disabled}>
								<td class="mono"><strong>{u.username}</strong></td>
								<td>
									<span class="pill" class:pill-ok={!u.disabled} class:pill-err={u.disabled}>
										{u.disabled ? 'desabilitado' : 'ativo'}
									</span>
								</td>
								<td class="actions">
									<button class="btn btn-ghost" on:click={() => changeUserPassword(u)}>senha</button>
									<button class="btn btn-ghost" on:click={() => toggleUserDisabled(u)}>
										{u.disabled ? 'ativar' : 'desativar'}
									</button>
									<button class="btn btn-danger" on:click={() => deleteUser(u)}>✕</button>
								</td>
							</tr>
						{/each}
					</tbody>
				</table>
			</section>
		{/if}
	{/if}

{:else if tab === 'help'}
	<!-- ============ HELP / COMO ACESSAR ============ -->
	<section class="card help">
		<div class="card-header"><div class="card-title">COMO CONECTAR</div></div>
		<div class="card-body help-body">
			{#each shares as sh}
				<div class="share-help">
					<div class="share-help-name mono">📁 {sh.name}</div>
					<div class="share-help-grid">
						<div>
							<div class="lbl uppercase-tag">windows</div>
							<code class="mono">\\pinas.local\{sh.name}</code>
						</div>
						<div>
							<div class="lbl uppercase-tag">macos (finder · cmd+k)</div>
							<code class="mono">smb://pinas.local/{sh.name}</code>
						</div>
						<div>
							<div class="lbl uppercase-tag">linux (gnome/nautilus)</div>
							<code class="mono">smb://pinas.local/{sh.name}</code>
						</div>
						<div>
							<div class="lbl uppercase-tag">cli</div>
							<code class="mono">smbclient -U &lt;user&gt; //pinas.local/{sh.name}</code>
						</div>
					</div>
				</div>
			{/each}

			{#if shares.length === 0}
				<div class="empty">Nenhum share criado ainda.</div>
			{/if}

			<div class="hr"></div>

			<h3 class="help-h">Notas importantes</h3>
			<ul class="notes">
				<li>
					Use os <strong>usuários SMB</strong> da aba acima — não os usuários do painel PiNAS.
					Eles são bancos de dados separados.
				</li>
				<li>
					Se sua rede não resolve <code>pinas.local</code> (mDNS), use o IP do Pi.
				</li>
				<li>
					Mudanças nesta página são aplicadas pelo Samba em ~1-2 segundos
					(via watcher inotify + smbpasswd no host).
				</li>
				<li>
					Os arquivos NÃO ficam dentro do PiNAS — eles ficam no path indicado em cada share.
					Apagar um share pelo painel NÃO apaga arquivos.
				</li>
			</ul>
		</div>
	</section>
{/if}

<style>
	.head {
		display: flex;
		align-items: flex-end;
		justify-content: space-between;
		gap: var(--sp-4);
		margin-bottom: var(--sp-4);
		padding-bottom: var(--sp-4);
		border-bottom: 1px solid var(--line-1);
	}
	.title {
		font-family: var(--font-mono);
		font-size: 28px;
		font-weight: 700;
		letter-spacing: -0.03em;
		margin: 0 0 4px;
	}
	.sub { color: var(--fg-2); font-size: 12px; margin: 0; }
	.status { display: flex; gap: var(--sp-2); }

	.tabs {
		display: flex;
		gap: 0;
		border-bottom: 1px solid var(--line-1);
		margin-bottom: var(--sp-4);
	}
	.tab {
		display: flex;
		align-items: center;
		gap: var(--sp-2);
		padding: 10px 16px;
		color: var(--fg-2);
		font-size: 13px;
		font-weight: 500;
		border-bottom: 2px solid transparent;
		margin-bottom: -1px;
	}
	.tab:hover { color: var(--fg-0); }
	.tab.active {
		color: var(--accent);
		border-bottom-color: var(--accent);
	}
	.tab-num {
		font-size: 10px;
		color: var(--fg-3);
		letter-spacing: 0.1em;
	}
	.tab.active .tab-num { color: var(--accent-soft); }

	.actions-row {
		display: flex;
		justify-content: flex-end;
		margin-bottom: var(--sp-3);
	}

	.error {
		padding: var(--sp-3);
		background: rgba(242, 91, 74, 0.06);
		border: 1px solid rgba(242, 91, 74, 0.3);
		border-radius: var(--radius);
		color: var(--err);
		margin-bottom: var(--sp-3);
	}

	.loading {
		text-align: center;
		padding: var(--sp-7);
		color: var(--fg-3);
		letter-spacing: 0.1em;
	}

	.empty-state {
		padding: var(--sp-6) var(--sp-5);
		text-align: center;
		display: flex;
		flex-direction: column;
		gap: var(--sp-3);
		align-items: center;
	}
	.empty-title {
		font-size: 13px;
		letter-spacing: 0.12em;
		color: var(--fg-2);
	}
	.empty-sub {
		font-size: 13px;
		color: var(--fg-3);
		max-width: 420px;
		line-height: 1.5;
	}

	.t { width: 100%; border-collapse: collapse; font-size: 13px; }
	.t thead th {
		text-align: left;
		padding: 10px var(--sp-4);
		font-family: var(--font-mono);
		font-size: 10px;
		font-weight: 500;
		letter-spacing: 0.1em;
		color: var(--fg-3);
		background: var(--bg-1);
		border-bottom: 1px solid var(--line-1);
	}
	.t tbody td {
		padding: 11px var(--sp-4);
		border-bottom: 1px solid var(--line-1);
		vertical-align: middle;
	}
	.t tbody tr:last-child td { border-bottom: 0; }
	.t tr.row-disabled { opacity: 0.5; }

	.name-cell { display: flex; flex-direction: column; gap: 2px; line-height: 1.3; }
	.name-cell strong { font-size: 13px; }
	.share-icon { font-size: 14px; margin-right: 6px; }
	.comment { font-size: 11px; }

	.flag-row { display: flex; gap: 4px; flex-wrap: wrap; }

	.actions { display: flex; gap: 4px; justify-content: flex-end; }
	.actions .btn { padding: 4px 10px; font-size: 12px; }

	.form-card { margin-bottom: var(--sp-4); }

	.create-user {
		display: flex;
		flex-direction: column;
		gap: var(--sp-3);
		max-width: 420px;
	}
	.field { display: flex; flex-direction: column; gap: 6px; }
	.lbl { font-size: 11px; }
	.hint { font-size: 11px; color: var(--fg-3); font-family: var(--font-mono); }

	.help-body { display: flex; flex-direction: column; gap: var(--sp-4); }
	.share-help {
		padding: var(--sp-3);
		background: var(--bg-1);
		border: 1px solid var(--line-1);
		border-radius: var(--radius);
	}
	.share-help-name {
		font-size: 14px;
		font-weight: 600;
		margin-bottom: var(--sp-3);
	}
	.share-help-grid {
		display: grid;
		grid-template-columns: repeat(auto-fit, minmax(280px, 1fr));
		gap: var(--sp-3);
	}
	.share-help-grid > div {
		display: flex;
		flex-direction: column;
		gap: 4px;
	}
	.share-help-grid code {
		display: block;
		padding: 6px 10px;
		background: var(--bg-3);
		border-radius: var(--radius);
		font-size: 12px;
		color: var(--accent);
		word-break: break-all;
	}
	.help-h {
		font-family: var(--font-mono);
		font-size: 12px;
		text-transform: uppercase;
		letter-spacing: 0.12em;
		color: var(--fg-1);
		margin: 0 0 var(--sp-2);
	}
	.notes {
		margin: 0;
		padding-left: 18px;
		display: flex;
		flex-direction: column;
		gap: 8px;
		font-size: 13px;
		color: var(--fg-1);
		line-height: 1.5;
	}
	.notes code {
		font-family: var(--font-mono);
		font-size: 12px;
		background: var(--bg-3);
		padding: 1px 6px;
		border-radius: 3px;
		color: var(--accent);
	}
	.empty {
		text-align: center;
		padding: var(--sp-5);
		color: var(--fg-3);
	}
</style>
