<script>
	import { page } from '$app/stores';
	import { auth } from '$lib/stores/auth.js';
	import { goto } from '$app/navigation';
	import { api } from '$lib/api/client.js';

	const items = [
		{ href: '/dashboard', label: 'Dashboard', code: '01' },
		{ href: '/files',     label: 'Arquivos',  code: '02' },
		{ href: '/samba',     label: 'Samba',     code: '03', adminOnly: true },
		{ href: '/users',     label: 'Usuários',  code: '04', adminOnly: true },
		{ href: '/system',    label: 'Sistema',   code: '05' },
		{ href: '/settings',  label: 'Ajustes',   code: '06' }
	];

	async function logout() {
		try { await api('/auth/logout', { method: 'POST' }); } catch {}
		auth.clear();
		goto('/login');
	}

	$: role = $auth?.user?.role;
	$: visibleItems = items.filter((i) => !i.adminOnly || role === 'admin');
</script>

<aside class="sidebar">
	<div class="brand">
		<div class="logo">
			<svg viewBox="0 0 24 24" width="22" height="22" fill="none" stroke="currentColor" stroke-width="2">
				<rect x="3" y="4" width="18" height="5" rx="1" />
				<rect x="3" y="11" width="18" height="5" rx="1" />
				<circle cx="7" cy="6.5" r="0.6" fill="currentColor" stroke="none" />
				<circle cx="7" cy="13.5" r="0.6" fill="currentColor" stroke="none" />
				<line x1="6" y1="20" x2="18" y2="20" />
			</svg>
		</div>
		<div class="brand-text">
			<div class="brand-title">PiNAS</div>
			<div class="brand-sub">v0.1 · arm64</div>
		</div>
	</div>

	<nav class="nav">
		{#each visibleItems as item}
			<a
				href={item.href}
				class="nav-item"
				class:active={$page.url.pathname.startsWith(item.href)}
			>
				<span class="nav-code">{item.code}</span>
				<span class="nav-label">{item.label}</span>
			</a>
		{/each}
	</nav>

	<div class="footer">
		{#if $auth?.user}
			<div class="user-block">
				<div class="user-name">{$auth.user.username}</div>
				<div class="user-role mono">{$auth.user.role}</div>
			</div>
			<button class="btn btn-ghost btn-logout" on:click={logout} title="Sair">
				<svg viewBox="0 0 24 24" width="16" height="16" fill="none" stroke="currentColor" stroke-width="2">
					<path d="M9 21H5a2 2 0 01-2-2V5a2 2 0 012-2h4" />
					<polyline points="16 17 21 12 16 7" />
					<line x1="21" y1="12" x2="9" y2="12" />
				</svg>
			</button>
		{/if}
	</div>
</aside>

<style>
	.sidebar {
		grid-area: sidebar;
		display: flex;
		flex-direction: column;
		background: var(--bg-1);
		border-right: 1px solid var(--line-1);
		padding: var(--sp-4) 0;
	}

	.brand {
		display: flex;
		align-items: center;
		gap: var(--sp-3);
		padding: 0 var(--sp-4) var(--sp-4);
		border-bottom: 1px solid var(--line-1);
		margin-bottom: var(--sp-3);
	}

	.logo {
		width: 38px;
		height: 38px;
		display: grid;
		place-items: center;
		background: var(--accent-bg);
		border: 1px solid var(--accent-line);
		border-radius: var(--radius);
		color: var(--accent);
	}

	.brand-title {
		font-family: var(--font-mono);
		font-weight: 700;
		letter-spacing: -0.02em;
		font-size: 16px;
	}

	.brand-sub {
		font-family: var(--font-mono);
		font-size: 10px;
		color: var(--fg-3);
		letter-spacing: 0.08em;
		text-transform: uppercase;
	}

	.nav {
		flex: 1;
		display: flex;
		flex-direction: column;
		gap: 2px;
		padding: 0 var(--sp-2);
	}

	.nav-item {
		display: flex;
		align-items: center;
		gap: var(--sp-3);
		padding: 10px var(--sp-3);
		border-radius: var(--radius);
		color: var(--fg-2);
		font-size: 14px;
		transition: all 100ms ease;
		border: 1px solid transparent;
	}

	.nav-item:hover {
		background: var(--bg-2);
		color: var(--fg-0);
	}

	.nav-item.active {
		background: var(--bg-3);
		color: var(--fg-0);
		border-color: var(--line-2);
	}

	.nav-item.active::before {
		content: '';
		position: absolute;
		left: 0;
		width: 3px;
		height: 18px;
		background: var(--accent);
		margin-left: -10px;
	}

	.nav-code {
		font-family: var(--font-mono);
		font-size: 11px;
		font-weight: 500;
		color: var(--fg-3);
		min-width: 20px;
	}

	.nav-item.active .nav-code {
		color: var(--accent);
	}

	.nav-label {
		font-weight: 500;
	}

	.footer {
		display: flex;
		align-items: center;
		justify-content: space-between;
		gap: var(--sp-2);
		padding: var(--sp-3) var(--sp-4);
		border-top: 1px solid var(--line-1);
		margin-top: var(--sp-3);
	}

	.user-block {
		min-width: 0;
	}

	.user-name {
		font-size: 13px;
		font-weight: 500;
		color: var(--fg-0);
		overflow: hidden;
		text-overflow: ellipsis;
		white-space: nowrap;
	}

	.user-role {
		font-size: 10px;
		color: var(--fg-3);
		text-transform: uppercase;
		letter-spacing: 0.08em;
	}

	.btn-logout {
		padding: 6px;
		color: var(--fg-2);
	}
	.btn-logout:hover { color: var(--err); }
</style>
