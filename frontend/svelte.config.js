import adapter from '@sveltejs/adapter-static';
import { vitePreprocess } from '@sveltejs/vite-plugin-svelte';

/** @type {import('@sveltejs/kit').Config} */
const config = {
	preprocess: vitePreprocess(),
	kit: {
		// Build estático servido pelo Caddy.
		// fallback: 'index.html' faz funcionar como SPA (todas as rotas caem no client router).
		adapter: adapter({
			pages: 'build',
			assets: 'build',
			fallback: 'index.html',
			precompress: false,
			strict: false
		}),
		alias: {
			$lib: 'src/lib'
		}
	}
};

export default config;
