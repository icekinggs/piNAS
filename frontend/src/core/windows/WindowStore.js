import { writable } from 'svelte/store';

export const windows = writable([]);
export const activeWindowId = writable(null);

export const defaultWindow = {
	x: 96,
	y: 86,
	width: 960,
	height: 640,
	minWidth: 520,
	minHeight: 360,
	zIndex: 10,
	state: 'normal'
};
