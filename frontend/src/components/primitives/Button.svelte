<script>
	export let variant = 'default';
	export let type = 'button';
</script>

<button class={`btn btn-${variant}`} {type}>
	<slot />
</button>
