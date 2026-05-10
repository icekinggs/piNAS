<div class="wallpaper">
	<div class="glow glow-a"></div>
	<div class="glow glow-b"></div>
	<div class="grid"></div>
</div>

<style>
	.wallpaper {
		position: fixed;
		inset: 0;
		overflow: hidden;
		background: radial-gradient(circle at top, #11182d 0%, #06080d 48%, #040507 100%);
		z-index: 0;
	}

	.glow {
		position: absolute;
		border-radius: 999px;
		filter: blur(90px);
		opacity: .55;
	}

	.glow-a {
		width: 520px;
		height: 520px;
		left: -120px;
		top: -160px;
		background: rgba(102,217,255,.22);
	}

	.glow-b {
		width: 460px;
		height: 460px;
		right: -120px;
		bottom: -140px;
		background: rgba(124,242,91,.16);
	}

	.grid {
		position: absolute;
		inset: 0;
		background-image:
			linear-gradient(rgba(255,255,255,.03) 1px, transparent 1px),
			linear-gradient(90deg, rgba(255,255,255,.03) 1px, transparent 1px);
		background-size: 42px 42px;
		mask-image: radial-gradient(circle at center, black, transparent 85%);
	}
</style>
