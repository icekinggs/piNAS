<script>
	import { onMount } from 'svelte';
	import { goto } from '$app/navigation';
	import { auth } from '$lib/stores/auth.js';

	onMount(() => {
		goto($auth?.accessToken ? '/dashboard' : '/login');
	});
</script>

<div class="root mono pulse">redirecionando…</div>

<style>
	.root {
		display: grid;
		place-items: center;
		min-height: 100vh;
		color: var(--fg-3);
		letter-spacing: 0.1em;
	}
</style>
