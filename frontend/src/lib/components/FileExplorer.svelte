<script>
	import { onMount } from 'svelte';
	import { api } from '$lib/api/client.js';
	import { formatBytes, formatDate, pathParts, joinPath } from '$lib/utils/format.js';
	import { toast } from '$lib/stores/toasts.js';

	export let initialPath = '/';

	let cwd = initialPath;
	let entries = [];
	let loading = false;
	let error = '';
	let selected = new Set();
	let dragOver = false;
	let uploads = []; // { name, progress, ok, error }

	$: crumbs = (() => {
		const parts = pathParts(cwd);
		const out = [{ name: 'root', path: '/' }];
		let acc = '';
		for (const p of parts) {
			acc += '/' + p;
			out.push({ name: p, path: acc });
		}
		return out;
	})();

	async function load(path) {
		loading = true;
		error = '';
		try {
			const data = await api('/files', { query: { path } });
			cwd = data.path;
			entries = (data.entries || []).sort((a, b) => {
				if (a.is_dir !== b.is_dir) return a.is_dir ? -1 : 1;
				return a.name.localeCompare(b.name);
			});
			selected = new Set();
		} catch (e) {
			error = e.message;
			entries = [];
		} finally {
			loading = false;
		}
	}

	onMount(() => load(cwd));

	function navigate(path) {
		load(path);
	}

	function toggleSelect(p) {
		const next = new Set(selected);
		if (next.has(p)) next.delete(p);
		else next.add(p);
		selected = next;
	}

	async function deleteEntry(path) {
		const ok = await toast.confirm(`Excluir ${path}?`, { destructive: true });
		if (!ok) return;
		try {
			await api('/files', { method: 'DELETE', query: { path } });
			toast.success('Excluído');
			await load(cwd);
		} catch (e) {
			toast.error('Falha ao excluir', e.message);
		}
	}

	async function deleteSelected() {
		if (selected.size === 0) return;
		const n = selected.size;
		const ok = await toast.confirm(`Excluir ${n} item(ns)?`, { destructive: true });
		if (!ok) return;
		let failures = 0;
		for (const p of selected) {
			try { await api('/files', { method: 'DELETE', query: { path: p } }); }
			catch { failures++; }
		}
		if (failures === 0) {
			toast.success(`${n} item(ns) excluído(s)`);
		} else {
			toast.warn(`${n - failures} excluído(s)`, `${failures} falha(s)`);
		}
		await load(cwd);
	}

	async function newFolder() {
		const name = prompt('Nome da pasta:');
		if (!name) return;
		try {
			await api('/files/folder', {
				method: 'POST',
				body: { path: joinPath(cwd, name) }
			});
			toast.success('Pasta criada', name);
			await load(cwd);
		} catch (e) {
			toast.error('Falha ao criar pasta', e.message);
		}
	}

	async function renameEntry(path, oldName) {
		const newName = prompt('Novo nome:', oldName);
		if (!newName || newName === oldName) return;
		try {
			await api('/files/rename', {
				method: 'PATCH',
				body: { path, new_name: newName }
			});
			toast.success('Renomeado');
			await load(cwd);
		} catch (e) {
			toast.error('Falha ao renomear', e.message);
		}
	}

	function downloadEntry(path) {
		(async () => {
			try {
				const res = await api('/files/download', { query: { path }, raw: true });
				if (!res.ok) {
					toast.error('Falha no download');
					return;
				}
				const blob = await res.blob();
				const url = URL.createObjectURL(blob);
				const a = document.createElement('a');
				a.href = url;
				a.download = path.split('/').pop();
				a.click();
				URL.revokeObjectURL(url);
			} catch (e) {
				toast.error('Falha no download', e.message);
			}
		})();
	}
			const a = document.createElement('a');
			a.href = url;
			a.download = path.split('/').pop();
			a.click();
			URL.revokeObjectURL(url);
		})();
	}

	async function handleFiles(fileList) {
		const formData = new FormData();
		formData.append('path', cwd);
		for (const f of fileList) formData.append('file', f);
		const localId = Date.now();
		uploads = [...uploads, { id: localId, name: `${fileList.length} arquivo(s)`, progress: 0 }];
		try {
			await api('/files/upload', { method: 'POST', formData });
			uploads = uploads.map((u) => u.id === localId ? { ...u, progress: 100, ok: true } : u);
			setTimeout(() => uploads = uploads.filter((u) => u.id !== localId), 1500);
			await load(cwd);
		} catch (e) {
			uploads = uploads.map((u) => u.id === localId ? { ...u, error: e.message } : u);
		}
	}

	function onDrop(ev) {
		ev.preventDefault();
		dragOver = false;
		if (ev.dataTransfer?.files?.length) handleFiles(ev.dataTransfer.files);
	}

	function onPickFiles(ev) {
		if (ev.target.files?.length) handleFiles(ev.target.files);
		ev.target.value = '';
	}

	function iconFor(entry) {
		if (entry.is_dir) return '📁';
		const ext = entry.name.split('.').pop().toLowerCase();
		if (['jpg','jpeg','png','gif','webp','svg'].includes(ext)) return '🖼';
		if (['mp4','mkv','mov','avi','webm'].includes(ext)) return '🎬';
		if (['mp3','wav','flac','ogg','m4a'].includes(ext)) return '🎵';
		if (['pdf'].includes(ext)) return '📕';
		if (['zip','tar','gz','7z','rar'].includes(ext)) return '🗜';
		if (['md','txt','log'].includes(ext)) return '📄';
		return '📦';
	}
</script>

<div class="explorer"
	on:dragover|preventDefault={() => dragOver = true}
	on:dragleave|preventDefault={() => dragOver = false}
	on:drop={onDrop}
	role="region"
	aria-label="Explorador de arquivos"
>
	<header class="toolbar">
		<nav class="crumbs">
			{#each crumbs as c, i}
				{#if i > 0}<span class="sep">/</span>{/if}
				<button class="crumb" on:click={() => navigate(c.path)}>{c.name}</button>
			{/each}
		</nav>
		<div class="actions">
			<label class="btn">
				<input type="file" hidden multiple on:change={onPickFiles} />
				Upload
			</label>
			<button class="btn" on:click={newFolder}>Nova pasta</button>
			<button class="btn btn-danger" disabled={selected.size === 0} on:click={deleteSelected}>
				Excluir {selected.size > 0 ? `(${selected.size})` : ''}
			</button>
		</div>
	</header>

	{#if dragOver}
		<div class="drop-overlay">
			<div class="drop-msg mono">SOLTE PARA ENVIAR PARA {cwd}</div>
		</div>
	{/if}

	{#if uploads.length > 0}
		<div class="uploads">
			{#each uploads as u}
				<div class="upload-row" class:ok={u.ok} class:err={u.error}>
					<span class="mono">{u.name}</span>
					<span class="upload-status mono">
						{u.error ? `ERR: ${u.error}` : u.ok ? 'OK' : 'enviando…'}
					</span>
				</div>
			{/each}
		</div>
	{/if}

	{#if error}
		<div class="error mono">{error}</div>
	{:else if loading}
		<div class="loading mono pulse">carregando…</div>
	{:else if entries.length === 0}
		<div class="empty">
			<div class="empty-title mono">DIRETÓRIO VAZIO</div>
			<div class="empty-sub">Arraste arquivos para enviar ou crie uma nova pasta.</div>
		</div>
	{:else}
		<div class="list">
			<div class="list-head mono">
				<span class="col-check"></span>
				<span class="col-name">NAME</span>
				<span class="col-size">SIZE</span>
				<span class="col-mtime">MODIFIED</span>
				<span class="col-actions"></span>
			</div>
			{#each entries as e}
				<div class="row" class:selected={selected.has(e.path)}>
					<input
						type="checkbox"
						class="col-check"
						checked={selected.has(e.path)}
						on:change={() => toggleSelect(e.path)}
					/>
					<button class="col-name name-btn"
						on:click={() => e.is_dir ? navigate(e.path) : downloadEntry(e.path)}
					>
						<span class="icon">{iconFor(e)}</span>
						<span class="name">{e.name}</span>
					</button>
					<span class="col-size mono">{e.is_dir ? '—' : formatBytes(e.size)}</span>
					<span class="col-mtime mono">{formatDate(e.mtime)}</span>
					<span class="col-actions">
						<button class="btn btn-ghost" on:click={() => renameEntry(e.path, e.name)} title="Renomear">
							✎
						</button>
						<button class="btn btn-ghost" on:click={() => deleteEntry(e.path)} title="Excluir">
							✕
						</button>
					</span>
				</div>
			{/each}
		</div>
	{/if}
</div>

<style>
	.explorer {
		position: relative;
		background: var(--bg-2);
		border: 1px solid var(--line-1);
		border-radius: var(--radius);
		overflow: hidden;
	}

	.toolbar {
		display: flex;
		align-items: center;
		justify-content: space-between;
		gap: var(--sp-3);
		padding: var(--sp-3) var(--sp-4);
		border-bottom: 1px solid var(--line-1);
		background: var(--bg-1);
	}

	.crumbs {
		display: flex;
		align-items: center;
		gap: 6px;
		font-family: var(--font-mono);
		font-size: 13px;
		flex-wrap: wrap;
	}

	.crumb {
		background: none;
		color: var(--fg-1);
		padding: 2px 6px;
		border-radius: 3px;
		cursor: pointer;
	}
	.crumb:hover { background: var(--bg-3); color: var(--fg-0); }

	.sep { color: var(--fg-3); }

	.actions {
		display: flex;
		gap: var(--sp-2);
	}

	.uploads {
		padding: var(--sp-2) var(--sp-4);
		border-bottom: 1px solid var(--line-1);
		background: var(--bg-1);
		display: flex;
		flex-direction: column;
		gap: 4px;
	}

	.upload-row {
		display: flex;
		justify-content: space-between;
		font-size: 12px;
		color: var(--fg-2);
	}
	.upload-row.ok { color: var(--ok); }
	.upload-row.err { color: var(--err); }

	.upload-status { font-size: 11px; }

	.list {
		display: flex;
		flex-direction: column;
	}

	.list-head, .row {
		display: grid;
		grid-template-columns: 32px 1fr 110px 160px 90px;
		align-items: center;
		gap: var(--sp-3);
		padding: 0 var(--sp-4);
	}

	.list-head {
		height: 36px;
		background: var(--bg-1);
		border-bottom: 1px solid var(--line-1);
		font-size: 11px;
		color: var(--fg-3);
		letter-spacing: 0.1em;
	}

	.row {
		height: 44px;
		border-bottom: 1px solid var(--line-1);
		transition: background 80ms ease;
	}
	.row:hover { background: var(--bg-3); }
	.row.selected { background: rgba(124, 242, 91, 0.06); }
	.row:last-child { border-bottom: 0; }

	.col-check { display: flex; align-items: center; }

	.name-btn {
		display: flex;
		align-items: center;
		gap: var(--sp-3);
		min-width: 0;
		padding: 0;
		text-align: left;
	}
	.name-btn .icon {
		font-size: 16px;
		flex-shrink: 0;
	}
	.name-btn .name {
		font-size: 14px;
		color: var(--fg-0);
		overflow: hidden;
		text-overflow: ellipsis;
		white-space: nowrap;
	}
	.name-btn:hover .name {
		color: var(--accent);
	}

	.col-size, .col-mtime {
		font-size: 12px;
		color: var(--fg-2);
	}

	.col-actions {
		display: flex;
		gap: 4px;
		justify-content: flex-end;
	}
	.col-actions .btn {
		padding: 4px 8px;
		font-size: 13px;
	}

	.error {
		padding: var(--sp-5);
		color: var(--err);
		text-align: center;
	}

	.loading, .empty {
		padding: var(--sp-7);
		text-align: center;
		color: var(--fg-3);
	}

	.empty-title {
		font-size: 13px;
		letter-spacing: 0.12em;
		color: var(--fg-2);
		margin-bottom: 8px;
	}

	.empty-sub {
		font-size: 13px;
		color: var(--fg-3);
	}

	.drop-overlay {
		position: absolute;
		inset: 0;
		background: rgba(20, 32, 19, 0.85);
		border: 2px dashed var(--accent);
		display: grid;
		place-items: center;
		z-index: 10;
		pointer-events: none;
	}

	.drop-msg {
		font-size: 14px;
		letter-spacing: 0.12em;
		color: var(--accent);
		font-weight: 700;
	}
</style>
