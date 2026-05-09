<script>
	import FileExplorer from '$lib/components/FileExplorer.svelte';
	import { auth } from '$lib/stores/auth.js';

	$: home = $auth?.user?.role === 'admin' ? '/' : ($auth?.user?.home_path || '/');
</script>

<header class="head">
	<div>
		<h1 class="title">Arquivos</h1>
		<p class="sub mono">explorador · armazenamento local</p>
	</div>
</header>

<FileExplorer initialPath={home} />

<style>
	.head {
		display: flex;
		align-items: flex-end;
		justify-content: space-between;
		gap: var(--sp-4);
		margin-bottom: var(--sp-4);
	}
	.title {
		font-family: var(--font-mono);
		font-size: 28px;
		font-weight: 700;
		letter-spacing: -0.03em;
		margin: 0 0 4px;
	}
	.sub { color: var(--fg-2); font-size: 12px; margin: 0; }
</style>
