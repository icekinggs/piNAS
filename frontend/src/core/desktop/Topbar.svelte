<script>
	import { auth } from '$lib/stores/auth.js';
</script>

<header class="topbar">
	<div class="brand">
		<div class="logo">π</div>
		<div>
			<strong>piNAS</strong>
			<span>Personal Cloud OS</span>
		</div>
	</div>

	<div class="search">Buscar apps, arquivos e ações</div>

	<div class="user">
		<span class="dot"></span>
		<span>{$auth?.user?.username || 'local'}</span>
	</div>
</header>

<style>
	.topbar {
		position: fixed;
		z-index: 1000;
		top: 18px;
		left: 24px;
		right: 24px;
		height: 58px;
		display: grid;
		grid-template-columns: 260px minmax(260px, 520px) 260px;
		align-items: center;
		gap: 18px;
		padding: 0 18px;
		border-radius: 24px;
		background: rgba(9, 14, 25, .66);
		border: 1px solid rgba(255,255,255,.11);
		box-shadow: 0 18px 55px rgba(0,0,0,.32), inset 0 1px 0 rgba(255,255,255,.08);
		backdrop-filter: blur(26px) saturate(150%);
	}
	.brand { display: flex; align-items: center; gap: 12px; }
	.logo { width: 36px; height: 36px; display: grid; place-items: center; border-radius: 14px; background: linear-gradient(135deg, rgba(124,242,91,.95), rgba(102,217,255,.75)); color: #061008; font-weight: 900; box-shadow: 0 10px 30px rgba(124,242,91,.18); }
	.brand strong { display: block; font-size: 14px; letter-spacing: -.02em; }
	.brand span { display: block; color: var(--os-text-dim); font-size: 11px; margin-top: -2px; }
	.search { height: 38px; display: flex; align-items: center; justify-content: center; border-radius: 16px; background: rgba(255,255,255,.06); color: var(--os-text-dim); font-size: 12px; border: 1px solid rgba(255,255,255,.07); }
	.user { display: flex; align-items: center; justify-content: flex-end; gap: 8px; color: var(--os-text-muted); font-size: 12px; }
	.dot { width: 8px; height: 8px; border-radius: 50%; background: var(--os-accent); box-shadow: 0 0 16px var(--os-accent); }
	@media (max-width: 860px) { .topbar { grid-template-columns: 1fr auto; left: 12px; right: 12px; } .search { display: none; } }
</style>
