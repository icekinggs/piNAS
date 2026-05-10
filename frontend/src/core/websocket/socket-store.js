import { writable } from 'svelte/store';

export const socketState = writable({
	connected: false,
	lastMessage: null
});
