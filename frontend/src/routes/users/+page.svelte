<script>
	import { onMount } from 'svelte';
	import { api } from '$lib/api/client.js';
	import { formatBytes, formatDate } from '$lib/utils/format.js';
	import { auth } from '$lib/stores/auth.js';
	import { toast } from '$lib/stores/toasts.js';
	import { goto } from '$app/navigation';

	let users = [];
	let error = '';
	let loading = false;
	let creating = false;

	let newUser = { username: '', password: '', role: 'user', quota_bytes: 0 };

	async function load() {
		loading = true;
		try {
			const data = await api('/users');
			users = data.users || [];
			error = '';
		} catch (e) { error = e.message; }
		finally { loading = false; }
	}

	async function create() {
		creating = true;
		try {
			await api('/users', { method: 'POST', body: newUser });
			toast.success('Usuário criado', `"${newUser.username}" foi adicionado.`);
			newUser = { username: '', password: '', role: 'user', quota_bytes: 0 };
			await load();
		} catch (e) { toast.error('Falha ao criar usuário', e.message); }
		finally { creating = false; }
	}

	async function toggleDisabled(u) {
		try {
			await api(`/users/${u.id}`, { method: 'PATCH', body: { disabled: !u.disabled } });
			toast.success(
				u.disabled ? 'Usuário ativado' : 'Usuário desativado',
				`"${u.username}" agora está ${u.disabled ? 'ativo' : 'inativo'}.`
			);
			await load();
		} catch (e) { toast.error('Falha', e.message); }
	}

	async function delUser(u) {
		const ok = await toast.confirm(`Excluir usuário "${u.username}"?`, {
			message: 'Esta ação não pode ser desfeita. Os arquivos do usuário não serão apagados.',
			destructive: true,
			confirmLabel: 'Excluir'
		});
		if (!ok) return;
		try {
			await api(`/users/${u.id}`, { method: 'DELETE' });
			toast.success('Usuário excluído', `"${u.username}" foi removido.`);
			await load();
		} catch (e) { toast.error('Falha ao excluir', e.message); }
	}

	async function changePass(u) {
		const p = prompt(`Nova senha para ${u.username}:`);
		if (!p) return;
		if (p.length < 8) {
			toast.warn('Senha curta', 'Mínimo 8 caracteres.');
			return;
		}
		try {
			await api(`/users/${u.id}/password`, { method: 'POST', body: { new_password: p } });
			toast.success('Senha alterada', `Nova senha de "${u.username}" foi salva.`);
		} catch (e) { toast.error('Falha ao alterar senha', e.message); }
	}

	onMount(() => {
		if ($auth?.user?.role !== 'admin') {
			goto('/dashboard');
			return;
		}
		load();
	});
</script>

<header class="head">
	<div>
		<h1 class="title">Usuários</h1>
		<p class="sub mono">gerenciamento · acesso local</p>
	</div>
</header>

<section class="card create">
	<div class="card-header">
		<div class="card-title">+ NOVO USUÁRIO</div>
	</div>
	<form class="card-body create-form" on:submit|preventDefault={create}>
		<div class="field">
			<label class="lbl uppercase-tag" for="cu">usuário</label>
			<input id="cu" class="input" type="text" required minlength="3" maxlength="32"
				bind:value={newUser.username} placeholder="ex: maria" />
		</div>
		<div class="field">
			<label class="lbl uppercase-tag" for="cp">senha</label>
			<input id="cp" class="input" type="password" required minlength="8" bind:value={newUser.password} />
		</div>
		<div class="field">
			<label class="lbl uppercase-tag" for="cr">papel</label>
			<select id="cr" class="input" bind:value={newUser.role}>
				<option value="user">user</option>
				<option value="admin">admin</option>
			</select>
		</div>
		<div class="field">
			<label class="lbl uppercase-tag" for="cq">quota (B, 0 = ∞)</label>
			<input id="cq" class="input" type="number" min="0" bind:value={newUser.quota_bytes} />
		</div>
		<button class="btn btn-primary" type="submit" disabled={creating}>
			{creating ? 'criando…' : '> criar'}
		</button>
	</form>
</section>

{#if error}
	<div class="error mono">{error}</div>
{/if}

<section class="card">
	<div class="card-header">
		<div class="card-title">USUÁRIOS</div>
		<div class="uppercase-tag">{users.length} total</div>
	</div>

	{#if loading}
		<div class="empty mono pulse">carregando…</div>
	{:else if users.length === 0}
		<div class="empty">nenhum usuário ainda</div>
	{:else}
		<table class="t">
			<thead>
				<tr>
					<th>USERNAME</th>
					<th>ROLE</th>
					<th>HOME</th>
					<th>QUOTA</th>
					<th>STATUS</th>
					<th>CRIADO</th>
					<th></th>
				</tr>
			</thead>
			<tbody>
				{#each users as u}
					<tr class:disabled={u.disabled}>
						<td class="mono">{u.username}</td>
						<td>
							<span class="pill" class:pill-ok={u.role === 'admin'}>{u.role}</span>
						</td>
						<td class="mono dim">{u.home_path}</td>
						<td class="mono dim">{u.quota_bytes ? formatBytes(u.quota_bytes) : '∞'}</td>
						<td>
							<span class="pill" class:pill-ok={!u.disabled} class:pill-err={u.disabled}>
								{u.disabled ? 'disabled' : 'active'}
							</span>
						</td>
						<td class="mono dim">{formatDate(u.created_at)}</td>
						<td class="actions">
							<button class="btn btn-ghost" on:click={() => changePass(u)}>senha</button>
							<button class="btn btn-ghost" on:click={() => toggleDisabled(u)}>
								{u.disabled ? 'ativar' : 'desativar'}
							</button>
							<button class="btn btn-danger" on:click={() => delUser(u)}>✕</button>
						</td>
					</tr>
				{/each}
			</tbody>
		</table>
	{/if}
</section>

<style>
	.head { margin-bottom: var(--sp-4); }
	.title {
		font-family: var(--font-mono);
		font-size: 28px;
		font-weight: 700;
		letter-spacing: -0.03em;
		margin: 0 0 4px;
	}
	.sub { color: var(--fg-2); font-size: 12px; margin: 0; }

	.create { margin-bottom: var(--sp-4); }
	.create-form {
		display: grid;
		grid-template-columns: 1.2fr 1.2fr 0.8fr 1fr auto;
		gap: var(--sp-3);
		align-items: end;
	}
	@media (max-width: 800px) {
		.create-form { grid-template-columns: 1fr 1fr; }
	}
	.field { display: flex; flex-direction: column; gap: 6px; }
	.lbl { font-size: 11px; }

	.error {
		padding: var(--sp-3);
		background: rgba(242, 91, 74, 0.06);
		border: 1px solid rgba(242, 91, 74, 0.3);
		border-radius: var(--radius);
		color: var(--err);
		margin-bottom: var(--sp-4);
	}

	.t {
		width: 100%;
		border-collapse: collapse;
		font-size: 13px;
	}
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
	.t tr.disabled { opacity: 0.55; }

	.actions {
		display: flex;
		gap: 4px;
		justify-content: flex-end;
	}
	.actions .btn { padding: 4px 8px; font-size: 12px; }

	.empty { padding: var(--sp-7); text-align: center; color: var(--fg-3); font-size: 13px; }
</style>
