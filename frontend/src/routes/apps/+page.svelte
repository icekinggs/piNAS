<script>
	const apps = [
		{ name: 'Jellyfin', category: 'Media' },
		{ name: 'Nextcloud', category: 'Cloud' },
		{ name: 'Immich', category: 'Photos' },
		{ name: 'Grafana', category: 'Monitoring' }
	];
</script>

<svelte:head>
	<title>Apps • PiNAS</title>
</svelte:head>

<div class="page">
	<header class="hero os-glass">
		<div>
			<div class="tag">docker apps</div>
			<h1>Loja Docker</h1>
			<p>Aplicações para instalar no servidor.</p>
		</div>
	</header>

	<div class="grid">
		{#each apps as app}
			<article class="card app-card os-glass os-panel">
				<div class="icon">⬢</div>
				<div class="content">
					<h3>{app.name}</h3>
					<span>{app.category}</span>
					<button class="btn btn-primary">Instalar</button>
				</div>
			</article>
		{/each}
	</div>
</div>

<style>
	.page { padding: 92px 28px 120px; }
	.hero { padding: 28px; border-radius: 28px; margin-bottom: 24px; }
	.tag { color: var(--os-accent); text-transform: uppercase; letter-spacing: .12em; font-size: 11px; margin-bottom: 10px; }
	h1 { margin: 0 0 10px; font-size: 42px; }
	p, span { color: var(--os-text-muted); }
	.grid { display: grid; grid-template-columns: repeat(auto-fill, minmax(280px, 1fr)); gap: 18px; }
	.app-card { padding: 20px; display: flex; gap: 18px; border-radius: 24px; }
	.icon { width: 72px; height: 72px; border-radius: 22px; display: grid; place-items: center; font-size: 28px; background: rgba(124,242,91,.12); color: var(--os-accent); }
	.content { display: grid; gap: 8px; }
</style>
