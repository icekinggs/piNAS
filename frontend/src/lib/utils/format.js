// src/lib/utils/format.js

export function formatBytes(b) {
	if (b == null || isNaN(b)) return '—';
	if (b < 1024) return `${b} B`;
	const units = ['KiB', 'MiB', 'GiB', 'TiB', 'PiB'];
	let val = b / 1024, i = 0;
	while (val >= 1024 && i < units.length - 1) {
		val /= 1024;
		i++;
	}
	return `${val.toFixed(val < 10 ? 2 : val < 100 ? 1 : 0)} ${units[i]}`;
}

export function formatDuration(secs) {
	if (!secs) return '0s';
	const d = Math.floor(secs / 86400);
	const h = Math.floor((secs % 86400) / 3600);
	const m = Math.floor((secs % 3600) / 60);
	const s = secs % 60;
	const parts = [];
	if (d) parts.push(`${d}d`);
	if (h) parts.push(`${h}h`);
	if (m) parts.push(`${m}m`);
	if (!d && !h) parts.push(`${s}s`);
	return parts.join(' ');
}

export function formatDate(unixSec) {
	if (!unixSec) return '—';
	const d = new Date(unixSec * 1000);
	return d.toLocaleString('pt-BR', {
		year: 'numeric', month: '2-digit', day: '2-digit',
		hour: '2-digit', minute: '2-digit'
	});
}

export function pathParts(p) {
	if (!p || p === '/') return [];
	return p.replace(/^\/+|\/+$/g, '').split('/');
}

export function joinPath(...parts) {
	return '/' + parts
		.flatMap((p) => (p || '').split('/'))
		.filter((s) => s && s !== '.')
		.join('/');
}
