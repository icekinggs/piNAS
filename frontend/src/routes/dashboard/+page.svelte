<script>
	import { onMount, onDestroy } from 'svelte';
	import { api } from '$lib/api/client.js';
	import { formatBytes, formatDuration } from '$lib/utils/format.js';
	import StatCard from '$lib/components/StatCard.svelte';

	let stats = null;
	let error = '';
	let timer;

	async function refresh() {
		try {
			stats = await api('/system/stats');
			error = '';
		} catch (e) {
			error = e.message;
		}
	}

	onMount(() => {
		refresh();
		timer = setInterval(refresh, 4000);
	});
	onDestroy(() => clearInterval(timer));

	$: cpuTone = !stats ? 'default' : stats.cpu_percent > 85 ? 'err' : stats.cpu_percent > 60 ? 'warn' : 'default';
	$: tempTone = !stats ? 'default' : stats.temp_celsius > 75 ? 'err' : stats.temp_celsius > 65 ? 'warn' : 'default';
	$: memTone = !stats ? 'default' : stats.memory.used_percent > 90 ? 'err' : stats.memory.used_percent > 75 ? 'warn' : 'default';
	$: diskPct = stats?.disk?.total ? (stats.disk.used / stats.disk.total) * 100 : 0;
	$: diskTone = diskPct > 90 ? 'err' : diskPct > 75 ? 'warn' : 'default';
</script>

<header class="head">
	<div>
		<h1 class="title">Dashboard</h1>
		<p class="sub mono">{stats?.hostname || '—'} · {stats?.os || '...'}</p>
	</div>
	<div class="status">
		<span class="pill pill-ok">OPERATIONAL</span>
		<span class="pill mono">UPTIME · {stats ? formatDuration(stats.uptime_sec) : '—'}</span>
	</div>
</header>

{#if error}
	<div class="error mono">erro: {error}</div>
{/if}

<section class="grid">
	<StatCard
		label="CPU"
		value={stats ? stats.cpu_percent.toFixed(1) : '—'}
		unit="%"
		percent={stats?.cpu_percent}
		detail={stats ? `${stats.cpu_cores} núcleos · load ${stats.load_avg[0].toFixed(2)} ${stats.load_avg[1].toFixed(2)} ${stats.load_avg[2].toFixed(2)}` : ''}
		tone={cpuTone}
	/>
	<StatCard
		label="Memória"
		value={stats ? (stats.memory.used / 1073741824).toFixed(2) : '—'}
		unit="GiB"
		percent={stats?.memory.used_percent}
		detail={stats ? `${formatBytes(stats.memory.used)} de ${formatBytes(stats.memory.total)}` : ''}
		tone={memTone}
	/>
	<StatCard
		label="Temperatura"
		value={stats ? stats.temp_celsius.toFixed(1) : '—'}
		unit="°C"
		percent={stats?.temp_celsius ? Math.min(100, (stats.temp_celsius / 85) * 100) : 0}
		detail="thermal_zone0"
		tone={tempTone}
	/>
	<StatCard
		label="Disco"
		value={stats?.disk?.total ? (stats.disk.used / 1073741824).toFixed(1) : '—'}
		unit="GiB"
		percent={diskPct}
		detail={stats?.disk?.total ? `${formatBytes(stats.disk.free)} livres de ${formatBytes(stats.disk.total)}` : ''}
		tone={diskTone}
	/>
</section>

<section class="row-2">
	<div class="card">
		<div class="card-header">
			<div class="card-title">REDE</div>
			<div class="uppercase-tag">interfaces</div>
		</div>
		<div class="card-body">
			{#if stats?.network?.length}
				<table class="net-table">
					<thead>
						<tr>
							<th>iface</th>
							<th>rx</th>
							<th>tx</th>
						</tr>
					</thead>
					<tbody>
						{#each stats.network as n}
							<tr>
								<td class="mono">{n.name}</td>
								<td class="mono dim">{formatBytes(n.bytes_in)}</td>
								<td class="mono dim">{formatBytes(n.bytes_out)}</td>
							</tr>
						{/each}
					</tbody>
				</table>
			{:else}
				<div class="muted">Sem interfaces detectadas.</div>
			{/if}
		</div>
	</div>

	<div class="card">
		<div class="card-header">
			<div class="card-title">SISTEMA</div>
			<div class="uppercase-tag">runtime</div>
		</div>
		<div class="card-body kv">
			<div><span class="k uppercase-tag">hostname</span><span class="v mono">{stats?.hostname || '—'}</span></div>
			<div><span class="k uppercase-tag">os</span><span class="v mono">{stats?.os || '—'}</span></div>
			<div><span class="k uppercase-tag">kernel</span><span class="v mono">{stats?.kernel || '—'}</span></div>
			<div><span class="k uppercase-tag">arch</span><span class="v mono">{stats?.arch || '—'}</span></div>
			<div><span class="k uppercase-tag">uptime</span><span class="v mono">{stats ? formatDuration(stats.uptime_sec) : '—'}</span></div>
		</div>
	</div>
</section>

<style>
	.head {
		display: flex;
		align-items: flex-end;
		justify-content: space-between;
		gap: var(--sp-4);
		margin-bottom: var(--sp-5);
		padding-bottom: var(--sp-4);
		border-bottom: 1px solid var(--line-1);
	}

	.title {
		font-family: var(--font-mono);
		font-size: 28px;
		font-weight: 700;
		letter-spacing: -0.03em;
		margin: 0 0 4px;
	}

	.sub {
		color: var(--fg-2);
		font-size: 12px;
		margin: 0;
	}

	.status {
		display: flex;
		gap: var(--sp-2);
	}

	.error {
		padding: var(--sp-3);
		background: rgba(242, 91, 74, 0.06);
		border: 1px solid rgba(242, 91, 74, 0.3);
		border-radius: var(--radius);
		color: var(--err);
		margin-bottom: var(--sp-4);
		font-size: 13px;
	}

	.grid {
		display: grid;
		grid-template-columns: repeat(auto-fit, minmax(220px, 1fr));
		gap: var(--sp-3);
		margin-bottom: var(--sp-5);
	}

	.row-2 {
		display: grid;
		grid-template-columns: 2fr 1fr;
		gap: var(--sp-3);
	}
	@media (max-width: 900px) {
		.row-2 { grid-template-columns: 1fr; }
	}

	.net-table {
		width: 100%;
		border-collapse: collapse;
		font-size: 13px;
	}
	.net-table th, .net-table td {
		padding: 8px 10px;
		text-align: left;
		border-bottom: 1px solid var(--line-1);
	}
	.net-table th {
		font-family: var(--font-mono);
		font-size: 10px;
		text-transform: uppercase;
		letter-spacing: 0.1em;
		color: var(--fg-3);
		font-weight: 500;
	}

	.kv {
		display: flex;
		flex-direction: column;
		gap: var(--sp-2);
	}

	.kv > div {
		display: flex;
		justify-content: space-between;
		align-items: center;
		padding: 4px 0;
		border-bottom: 1px dashed var(--line-1);
	}
	.kv > div:last-child { border-bottom: 0; }

	.k { font-size: 11px; }
	.v { font-size: 12px; color: var(--fg-1); }
</style>
