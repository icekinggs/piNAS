<script>
	import { onMount, onDestroy } from 'svelte';
	import { api } from '$lib/api/client.js';
	import { formatBytes, formatDuration, formatDate } from '$lib/utils/format.js';

	let stats = null;
	let history = []; // últimos pontos para sparkline
	let timer;

	async function tick() {
		try {
			stats = await api('/system/stats');
			history = [...history, { t: Date.now(), cpu: stats.cpu_percent, mem: stats.memory.used_percent }].slice(-60);
		} catch {}
	}

	onMount(() => { tick(); timer = setInterval(tick, 2000); });
	onDestroy(() => clearInterval(timer));

	function sparkline(values, key, w = 200, h = 40) {
		if (!values.length) return '';
		const xs = values.map(v => v[key]);
		const max = Math.max(100, ...xs);
		const step = w / Math.max(1, values.length - 1);
		return xs.map((y, i) => `${i === 0 ? 'M' : 'L'}${(i * step).toFixed(1)},${(h - (y / max) * h).toFixed(1)}`).join(' ');
	}
</script>

<header class="head">
	<div>
		<h1 class="title">Sistema</h1>
		<p class="sub mono">monitoramento em tempo real · 2s</p>
	</div>
</header>

{#if stats}
	<section class="grid">
		<div class="card spark-card">
			<div class="card-header">
				<div class="card-title">CPU</div>
				<div class="mono big">{stats.cpu_percent.toFixed(1)}%</div>
			</div>
			<div class="card-body">
				<svg viewBox="0 0 200 40" class="spark">
					<path d={sparkline(history, 'cpu')} fill="none" stroke="var(--accent)" stroke-width="1.5" />
				</svg>
				<div class="kv">
					<div><span class="k uppercase-tag">cores</span><span class="v mono">{stats.cpu_cores}</span></div>
					<div><span class="k uppercase-tag">load 1m</span><span class="v mono">{stats.load_avg[0].toFixed(2)}</span></div>
					<div><span class="k uppercase-tag">load 5m</span><span class="v mono">{stats.load_avg[1].toFixed(2)}</span></div>
					<div><span class="k uppercase-tag">load 15m</span><span class="v mono">{stats.load_avg[2].toFixed(2)}</span></div>
				</div>
			</div>
		</div>

		<div class="card spark-card">
			<div class="card-header">
				<div class="card-title">MEMÓRIA</div>
				<div class="mono big">{stats.memory.used_percent.toFixed(1)}%</div>
			</div>
			<div class="card-body">
				<svg viewBox="0 0 200 40" class="spark">
					<path d={sparkline(history, 'mem')} fill="none" stroke="#5bbef2" stroke-width="1.5" />
				</svg>
				<div class="kv">
					<div><span class="k uppercase-tag">total</span><span class="v mono">{formatBytes(stats.memory.total)}</span></div>
					<div><span class="k uppercase-tag">usado</span><span class="v mono">{formatBytes(stats.memory.used)}</span></div>
					<div><span class="k uppercase-tag">livre</span><span class="v mono">{formatBytes(stats.memory.free)}</span></div>
					<div><span class="k uppercase-tag">disp.</span><span class="v mono">{formatBytes(stats.memory.available)}</span></div>
				</div>
			</div>
		</div>

		<div class="card">
			<div class="card-header">
				<div class="card-title">TÉRMICO</div>
				<div class="mono big">{stats.temp_celsius.toFixed(1)}°C</div>
			</div>
			<div class="card-body">
				<div class="thermo">
					<div class="thermo-bar">
						<div class="thermo-fill" style="width: {Math.min(100, (stats.temp_celsius / 85) * 100)}%"></div>
						<div class="thermo-mark" style="left: 70%">throttle</div>
					</div>
					<div class="thermo-scale mono">
						<span>0</span><span>40</span><span>60</span><span>80</span><span>85°C</span>
					</div>
				</div>
			</div>
		</div>

		<div class="card">
			<div class="card-header">
				<div class="card-title">DISCO</div>
				<div class="mono big">{((stats.disk.used / stats.disk.total) * 100).toFixed(1)}%</div>
			</div>
			<div class="card-body">
				<div class="kv">
					<div><span class="k uppercase-tag">total</span><span class="v mono">{formatBytes(stats.disk.total)}</span></div>
					<div><span class="k uppercase-tag">usado</span><span class="v mono">{formatBytes(stats.disk.used)}</span></div>
					<div><span class="k uppercase-tag">livre</span><span class="v mono">{formatBytes(stats.disk.free)}</span></div>
				</div>
			</div>
		</div>
	</section>

	<section class="row-2">
		<div class="card">
			<div class="card-header"><div class="card-title">REDE</div></div>
			<div class="card-body">
				{#if stats.network?.length}
					<table class="t">
						<thead><tr><th>iface</th><th>recebido</th><th>enviado</th></tr></thead>
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
					<div class="muted">Sem interfaces.</div>
				{/if}
			</div>
		</div>

		<div class="card">
			<div class="card-header"><div class="card-title">HOST</div></div>
			<div class="card-body kv vert">
				<div><span class="k uppercase-tag">hostname</span><span class="v mono">{stats.hostname}</span></div>
				<div><span class="k uppercase-tag">os</span><span class="v mono">{stats.os}</span></div>
				<div><span class="k uppercase-tag">kernel</span><span class="v mono">{stats.kernel}</span></div>
				<div><span class="k uppercase-tag">arch</span><span class="v mono">{stats.arch}</span></div>
				<div><span class="k uppercase-tag">uptime</span><span class="v mono">{formatDuration(stats.uptime_sec)}</span></div>
				<div><span class="k uppercase-tag">snapshot</span><span class="v mono">{formatDate(stats.timestamp)}</span></div>
			</div>
		</div>
	</section>
{:else}
	<div class="loading mono pulse">conectando ao agente…</div>
{/if}

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

	.grid {
		display: grid;
		grid-template-columns: repeat(auto-fit, minmax(280px, 1fr));
		gap: var(--sp-3);
		margin-bottom: var(--sp-4);
	}

	.row-2 {
		display: grid;
		grid-template-columns: 2fr 1fr;
		gap: var(--sp-3);
	}
	@media (max-width: 900px) { .row-2 { grid-template-columns: 1fr; } }

	.big {
		font-size: 18px;
		font-weight: 700;
		letter-spacing: -0.02em;
	}

	.spark { width: 100%; height: 40px; margin-bottom: var(--sp-3); }

	.kv {
		display: grid;
		grid-template-columns: repeat(2, 1fr);
		gap: var(--sp-2) var(--sp-4);
	}
	.kv.vert { grid-template-columns: 1fr; gap: var(--sp-2); }
	.kv > div {
		display: flex;
		justify-content: space-between;
		font-size: 12px;
		padding: 3px 0;
		border-bottom: 1px dashed var(--line-1);
	}
	.k { font-size: 10px; }
	.v { color: var(--fg-1); }

	.thermo {
		display: flex;
		flex-direction: column;
		gap: 6px;
	}
	.thermo-bar {
		position: relative;
		height: 24px;
		background: linear-gradient(90deg, #5bbef2 0%, var(--accent) 50%, var(--warn) 78%, var(--err) 100%);
		border-radius: var(--radius);
		opacity: 0.25;
		overflow: hidden;
	}
	.thermo-fill {
		position: absolute;
		inset: 0 auto 0 0;
		background: linear-gradient(90deg, #5bbef2 0%, var(--accent) 50%, var(--warn) 78%, var(--err) 100%);
		opacity: 1;
		mix-blend-mode: normal;
	}
	.thermo-mark {
		position: absolute;
		top: -16px;
		font-family: var(--font-mono);
		font-size: 9px;
		color: var(--warn);
		text-transform: uppercase;
		letter-spacing: 0.1em;
		transform: translateX(-50%);
	}
	.thermo-scale {
		display: flex;
		justify-content: space-between;
		font-size: 10px;
		color: var(--fg-3);
	}

	.t { width: 100%; border-collapse: collapse; font-size: 13px; }
	.t th, .t td { padding: 7px 10px; text-align: left; border-bottom: 1px solid var(--line-1); }
	.t th { font-family: var(--font-mono); font-size: 10px; text-transform: uppercase; letter-spacing: 0.1em; color: var(--fg-3); font-weight: 500; }

	.loading {
		text-align: center;
		padding: var(--sp-7);
		color: var(--fg-3);
		letter-spacing: 0.1em;
	}
</style>
