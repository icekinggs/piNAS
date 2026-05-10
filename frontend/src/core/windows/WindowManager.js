import { windows, activeWindowId } from './WindowStore.js';

let nextWindowId = 1;
let nextWindowZ = 20;

export const WindowManager = {
	open(appId) {
		const id = 'window-' + nextWindowId;
		nextWindowId = nextWindowId + 1;
		nextWindowZ = nextWindowZ + 1;

		const win = {
			id,
			appId,
			title: appId,
			x: 96,
			y: 86,
			width: 920,
			height: 620,
			zIndex: nextWindowZ,
			state: 'normal'
		};

		windows.update(function(items) {
			return items.concat(win);
		});
		activeWindowId.set(id);
		return win;
	},

	close(id) {
		windows.update(function(items) {
			return items.filter(function(item) { return item.id !== id; });
		});
	},

	focus(id) {
		nextWindowZ = nextWindowZ + 1;
		windows.update(function(items) {
			return items.map(function(item) {
				if (item.id === id) return Object.assign({}, item, { zIndex: nextWindowZ });
				return item;
			});
		});
		activeWindowId.set(id);
	},

	minimize(id) {
		windows.update(function(items) {
			return items.map(function(item) {
				if (item.id === id) return Object.assign({}, item, { state: 'minimized' });
				return item;
			});
		});
	},

	restore(id) {
		nextWindowZ = nextWindowZ + 1;
		windows.update(function(items) {
			return items.map(function(item) {
				if (item.id === id) return Object.assign({}, item, { state: 'normal', zIndex: nextWindowZ });
				return item;
			});
		});
		activeWindowId.set(id);
	}
};
