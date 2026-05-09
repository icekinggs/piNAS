<script>
	import { goto } from '$app/navigation';
	import { auth } from '$lib/stores/auth.js';
	import { api } from '$lib/api/client.js';

	let username = '';
	let password = '';
	let loading = false;
	let error = '';

	async function submit() {
		loading = true;
		error = '';
		try {
			const data = await api('/auth/login', {
				method: 'POST',
				body: { username, password }
			});
			auth.set({
				accessToken: data.access_token,
				expiresAt: data.expires_at,
				user: data.user
			});
			goto('/dashboard');
		} catch (e) {
			error = e.message || 'falha no login';
		} finally {
			loading = false;
		}
	}
</script>

<div class="page">
	<div class="bg-grid"></div>

	<form class="login-card" on:submit|preventDefault={submit}>
		<header class="card-head">
			<div class="logo-mini">
				<svg viewBox="0 0 24 24" width="22" height="22" fill="none" stroke="currentColor" stroke-width="2">
					<rect x="3" y="4" width="18" height="5" rx="1" />
					<rect x="3" y="11" width="18" height="5" rx="1" />
					<circle cx="7" cy="6.5" r="0.6" fill="currentColor" stroke="none" />
					<circle cx="7" cy="13.5" r="0.6" fill="currentColor" stroke="none" />
					<line x1="6" y1="20" x2="18" y2="20" />
				</svg>
			</div>
			<div>
				<div class="title">PiNAS</div>
				<div class="sub mono">PRIVATE NETWORK STORAGE · v0.1</div>
			</div>
		</header>

		<div class="field">
			<label class="lbl uppercase-tag" for="u">usuário</label>
			<input id="u" class="input" type="text" autocomplete="username" required bind:value={username} autofocus />
		</div>

		<div class="field">
			<label class="lbl uppercase-tag" for="p">senha</label>
			<input id="p" class="input" type="password" autocomplete="current-password" required bind:value={password} />
		</div>

		{#if error}
			<div class="error mono">{error}</div>
		{/if}

		<button class="btn btn-primary submit" type="submit" disabled={loading}>
			{loading ? 'autenticando…' : '> entrar'}
		</button>

		<div class="footer-line mono">
			<span class="dot"></span>
			conexão local · sem telemetria
		</div>
	</form>
</div>

<style>
	.page {
		min-height: 100vh;
		display: grid;
		place-items: center;
		position: relative;
		overflow: hidden;
	}

	.bg-grid {
		position: absolute;
		inset: 0;
		background-image:
			linear-gradient(var(--line-1) 1px, transparent 1px),
			linear-gradient(90deg, var(--line-1) 1px, transparent 1px);
		background-size: 32px 32px;
		mask-image: radial-gradient(ellipse 60% 60% at 50% 50%, black 30%, transparent 80%);
		opacity: 0.4;
	}

	.login-card {
		position: relative;
		z-index: 1;
		width: 100%;
		max-width: 380px;
		padding: var(--sp-6) var(--sp-5);
		background: var(--bg-2);
		border: 1px solid var(--line-2);
		border-radius: var(--radius);
		display: flex;
		flex-direction: column;
		gap: var(--sp-4);
		box-shadow: 0 30px 80px -20px rgba(0,0,0,0.6);
	}

	.card-head {
		display: flex;
		align-items: center;
		gap: var(--sp-3);
		padding-bottom: var(--sp-4);
		border-bottom: 1px solid var(--line-1);
		margin-bottom: var(--sp-2);
	}

	.logo-mini {
		width: 40px;
		height: 40px;
		display: grid;
		place-items: center;
		background: var(--accent-bg);
		border: 1px solid var(--accent-line);
		border-radius: var(--radius);
		color: var(--accent);
	}

	.title {
		font-family: var(--font-mono);
		font-size: 22px;
		font-weight: 700;
		letter-spacing: -0.02em;
	}

	.sub {
		font-size: 10px;
		color: var(--fg-3);
		letter-spacing: 0.12em;
	}

	.field {
		display: flex;
		flex-direction: column;
		gap: 6px;
	}

	.lbl {
		font-size: 11px;
		letter-spacing: 0.1em;
	}

	.error {
		padding: var(--sp-2) var(--sp-3);
		background: rgba(242, 91, 74, 0.08);
		border: 1px solid rgba(242, 91, 74, 0.3);
		border-radius: var(--radius);
		color: var(--err);
		font-size: 12px;
	}

	.submit {
		width: 100%;
		justify-content: center;
		padding: 11px;
		font-family: var(--font-mono);
		font-weight: 700;
	}

	.footer-line {
		display: flex;
		align-items: center;
		gap: 8px;
		font-size: 11px;
		color: var(--fg-3);
		letter-spacing: 0.06em;
		justify-content: center;
		padding-top: var(--sp-3);
		border-top: 1px solid var(--line-1);
	}

	.dot {
		width: 6px;
		height: 6px;
		border-radius: 50%;
		background: var(--accent);
		box-shadow: 0 0 8px var(--accent);
	}
</style>
