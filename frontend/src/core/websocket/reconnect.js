export function reconnectSocket(factory, delay) {
	const timeout = delay || 3000;
	setTimeout(function() {
		factory();
	}, timeout);
}
