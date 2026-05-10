<script>
	import { auth } from '$lib/stores/auth.js';
</script>

<header class="topbar os-glass">
	<div class="brand"><strong>piNAS</strong><span>Web OS</span></div>
	<div class="user">{$auth?.user?.username || 'local'}</div>
</header>

<style>
	.topbar { position: fixed; z-index: 1000; top: 10px; left: 12px; right: 12px; height: var(--os-topbar-h); border-radius: 999px; display: flex; align-items: center; justify-content: space-between; padding: 0 14px; }
	.brand { display: flex; align-items: center; gap: 9px; font-size: 12px; }
	.brand span, .user { color: var(--os-text-muted); font-size: 12px; }
</style>
