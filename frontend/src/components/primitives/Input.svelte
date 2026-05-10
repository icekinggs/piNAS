<script>
	export let value = '';
	export let placeholder = '';
	export let type = 'text';
</script>

<input class="input" bind:value {placeholder} {type} />
