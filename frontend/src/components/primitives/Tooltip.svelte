<script>
	export let text = '';
</script>

<span class="tooltip-wrapper">
	<slot />
	<span class="tooltip">{text}</span>
</span>

<style>
	.tooltip-wrapper { position: relative; display: inline-flex; }
	.tooltip { position: absolute; left: 50%; bottom: calc(100% + 8px); transform: translateX(-50%); padding: 6px 10px; border-radius: 10px; background: rgba(0,0,0,.85); color: white; font-size: 11px; white-space: nowrap; opacity: 0; pointer-events: none; transition: opacity 120ms ease; }
	.tooltip-wrapper:hover .tooltip { opacity: 1; }
</style>
