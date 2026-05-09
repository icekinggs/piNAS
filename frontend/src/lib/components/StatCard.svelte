<script>
	export let label = '';
	export let value = '—';
	export let unit = '';
	export let detail = '';
	/** 0..100 — barra opcional */
	export let percent = null;
	export let tone = 'default'; // 'default' | 'ok' | 'warn' | 'err'
</script>

<div class="stat" data-tone={tone}>
	<div class="stat-label uppercase-tag">{label}</div>
	<div class="stat-value mono">
		<span class="num">{value}</span>
		{#if unit}<span class="unit">{unit}</span>{/if}
	</div>
	{#if percent !== null}
		<div class="bar">
			<div class="bar-fill" style="width: {Math.max(0, Math.min(100, percent))}%"></div>
		</div>
	{/if}
	{#if detail}
		<div class="stat-detail mono">{detail}</div>
	{/if}
</div>

<style>
	.stat {
		background: var(--bg-2);
		border: 1px solid var(--line-1);
		border-radius: var(--radius);
		padding: var(--sp-4);
		display: flex;
		flex-direction: column;
		gap: var(--sp-2);
		min-height: 110px;
		position: relative;
	}

	.stat::after {
		content: '';
		position: absolute;
		top: 0; left: 0;
		width: 24px;
		height: 2px;
		background: var(--accent);
	}

	.stat[data-tone="warn"]::after { background: var(--warn); }
	.stat[data-tone="err"]::after  { background: var(--err); }

	.stat-value {
		font-size: 28px;
		font-weight: 700;
		letter-spacing: -0.03em;
		line-height: 1;
		display: flex;
		align-items: baseline;
		gap: 6px;
		color: var(--fg-0);
	}

	.unit {
		font-size: 13px;
		color: var(--fg-2);
		font-weight: 500;
	}

	.bar {
		height: 4px;
		background: var(--bg-4);
		border-radius: 2px;
		overflow: hidden;
	}

	.bar-fill {
		height: 100%;
		background: var(--accent);
		transition: width 400ms ease;
	}

	.stat[data-tone="warn"] .bar-fill { background: var(--warn); }
	.stat[data-tone="err"]  .bar-fill { background: var(--err); }

	.stat-detail {
		font-size: 11px;
		color: var(--fg-3);
		margin-top: auto;
	}
</style>
