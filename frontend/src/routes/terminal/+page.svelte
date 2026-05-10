<svelte:head>
	<title>Terminal • PiNAS</title>
</svelte:head>

<script>
	let lines = [
		'PiNAS shell initialized',
		'Server connected',
		'Docker services online'
	];
</script>

<div class="terminal-page">
	<header class="hero os-glass">
		<div>
			<div class="tag">server shell</div>
			<h1>Terminal</h1>
			<p>Console web para gerenciamento do servidor.</p>
		</div>
	</header>

	<section class="terminal os-glass os-panel">
		{#each lines as line}
			<div class="line mono">{line}</div>
		{/each}
	</section>
</div>

<style>
	.terminal-page { padding: 92px 28px 120px; }
	.hero { padding: 28px; border-radius: 28px; margin-bottom: 24px; }
	.tag { color: var(--os-accent); text-transform: uppercase; letter-spacing: .12em; font-size: 11px; margin-bottom: 10px; }
	h1 { margin: 0 0 10px; font-size: 42px; }
	p { color: var(--os-text-muted); }
	.terminal { min-height: 520px; padding: 22px; border-radius: 28px; background: rgba(5,8,12,.88); }
	.line { line-height: 1.7; color: #9ef58a; }
</style>
