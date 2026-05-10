<script>
	import { goto } from '$app/navigation';
	import { page } from '$app/stores';
	import { appRegistry } from '../../modules/apps/registry.js';

	function active(route) {
		return $page.url.pathname === route || $page.url.pathname.startsWith(route + '/');
	}
</script>

<aside class="sidenav">
	<div class="brand-block">
		<div class="brand-icon">π</div>
		<div>
			<strong>piNAS</strong>
			<span>Private Cloud</span>
		</div>
	</div>

	<nav>
		{#each appRegistry as app}
			<button class:active={active(app.route)} on:click={() => goto(app.route)}>
				<span class="icon">{app.icon}</span>
				<span>{app.title}</span>
			</button>
		{/each}
	</nav>

	<div class="status-card">
		<span class="status-dot"></span>
		<div>
			<strong>Online</strong>
			<small>Sistema operacional</small>
		</div>
	</div>
</aside>

<style>
	.sidenav { position: fixed; z-index: 900; left: 22px; top: 22px; bottom: 22px; width: 244px; display: flex; flex-direction: column; padding: 18px; border-radius: 30px; background: rgba(10,16,29,.78); border: 1px solid rgba(255,255,255,.1); backdrop-filter: blur(28px) saturate(155%); box-shadow: 0 28px 80px rgba(0,0,0,.34), inset 0 1px 0 rgba(255,255,255,.08); }
	.brand-block { display: flex; align-items: center; gap: 12px; padding: 4px 4px 20px; margin-bottom: 10px; border-bottom: 1px solid rgba(255,255,255,.08); }
	.brand-icon { width: 42px; height: 42px; display: grid; place-items: center; border-radius: 16px; background: linear-gradient(135deg, #7cf25b, #66d9ff); color: #04100a; font-weight: 900; font-size: 20px; }
	strong { display: block; font-size: 14px; }
	span, small { color: var(--os-text-muted); }
	small { font-size: 11px; }
	nav { display: grid; gap: 8px; margin-top: 8px; }
	button { height: 46px; display: flex; align-items: center; gap: 12px; padding: 0 12px; border-radius: 16px; background: transparent; color: var(--os-text-muted); transition: background .16s ease, color .16s ease, transform .16s ease; }
	button:hover { background: rgba(255,255,255,.07); color: var(--os-text); transform: translateX(2px); }
	button.active { background: linear-gradient(135deg, rgba(124,242,91,.18), rgba(102,217,255,.12)); color: var(--os-text); box-shadow: inset 0 0 0 1px rgba(255,255,255,.08); }
	.icon { width: 28px; height: 28px; display: grid; place-items: center; border-radius: 10px; background: rgba(255,255,255,.06); color: var(--os-accent); }
	.status-card { margin-top: auto; display: flex; align-items: center; gap: 10px; padding: 14px; border-radius: 20px; background: rgba(255,255,255,.055); border: 1px solid rgba(255,255,255,.07); }
	.status-dot { width: 10px; height: 10px; border-radius: 50%; background: var(--os-accent); }
	@media (max-width: 920px) { .sidenav { display: none; } }
</style>
