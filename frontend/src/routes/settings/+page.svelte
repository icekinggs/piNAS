<script>
	import { auth } from '$lib/stores/auth.js';
	import { api } from '$lib/api/client.js';

	let oldPass = '';
	let newPass = '';
	let confirmPass = '';
	let busy = false;
	let msg = '';
	let err = '';

	async function changePassword() {
		err = ''; msg = '';
		if (newPass.length < 8) { err = 'senha mínima de 8 caracteres'; return; }
		if (newPass !== confirmPass) { err = 'senhas não conferem'; return; }
		busy = true;
		try {
			await api(`/users/${$auth.user.id}/password`, {
				method: 'POST',
				body: { new_password: newPass }
			});
			msg = 'senha atualizada';
			oldPass = newPass = confirmPass = '';
		} catch (e) { err = e.message; }
		finally { busy = false; }
	}
</script>

<header class="head">
	<div>
		<h1 class="title">Ajustes</h1>
		<p class="sub mono">conta · preferências</p>
	</div>
</header>

<section class="card">
	<div class="card-header">
		<div class="card-title">CONTA</div>
	</div>
	<div class="card-body kv">
		<div><span class="k uppercase-tag">usuário</span><span class="v mono">{$auth?.user?.username}</span></div>
		<div><span class="k uppercase-tag">papel</span><span class="v mono">{$auth?.user?.role}</span></div>
		<div><span class="k uppercase-tag">home</span><span class="v mono">{$auth?.user?.home_path}</span></div>
	</div>
</section>

<section class="card">
	<div class="card-header">
		<div class="card-title">ALTERAR SENHA</div>
	</div>
	<form class="card-body form" on:submit|preventDefault={changePassword}>
		<div class="field">
			<label class="lbl uppercase-tag" for="np">nova senha</label>
			<input id="np" class="input" type="password" required minlength="8" bind:value={newPass} />
		</div>
		<div class="field">
			<label class="lbl uppercase-tag" for="cp">confirmar</label>
			<input id="cp" class="input" type="password" required minlength="8" bind:value={confirmPass} />
		</div>
		{#if err}<div class="error mono">{err}</div>{/if}
		{#if msg}<div class="ok mono">{msg}</div>{/if}
		<button class="btn btn-primary" type="submit" disabled={busy}>
			{busy ? 'salvando…' : '> salvar'}
		</button>
	</form>
</section>

<section class="card">
	<div class="card-header">
		<div class="card-title">SOBRE</div>
	</div>
	<div class="card-body about">
		<p><strong>PiNAS</strong> v0.1 · Private NAS for Raspberry Pi.</p>
		<p class="muted">Sistema self-hosted, offline-first, zero-telemetria.</p>
		<p class="muted mono small">backend: Go · frontend: SvelteKit · proxy: Caddy · banco: SQLite</p>
	</div>
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

	.card { margin-bottom: var(--sp-3); }

	.kv { display: flex; flex-direction: column; gap: var(--sp-2); }
	.kv > div {
		display: flex; justify-content: space-between;
		padding: 4px 0; border-bottom: 1px dashed var(--line-1);
	}
	.k { font-size: 11px; }
	.v { color: var(--fg-1); font-size: 13px; }

	.form { display: flex; flex-direction: column; gap: var(--sp-3); max-width: 400px; }
	.field { display: flex; flex-direction: column; gap: 6px; }
	.lbl { font-size: 11px; }

	.error {
		color: var(--err);
		font-size: 12px;
		padding: 6px 10px;
		background: rgba(242, 91, 74, 0.06);
		border: 1px solid rgba(242, 91, 74, 0.3);
		border-radius: var(--radius);
	}
	.ok {
		color: var(--accent);
		font-size: 12px;
		padding: 6px 10px;
		background: var(--accent-bg);
		border: 1px solid var(--accent-line);
		border-radius: var(--radius);
	}

	.about p { margin: 0 0 6px; font-size: 13px; }
	.about .small { font-size: 11px; }
</style>
