<script>
	import { createEventDispatcher } from 'svelte';
	
	export let open = false;
	export let title = '';
	export let showHeader = true;
	export let showFooter = false;
	export let fullscreen = false;
	
	const dispatch = createEventDispatcher();

	function handleClose() {
		dispatch('close');
	}

	function handleKeydown(event) {
		if (event.key === 'Escape') {
			handleClose();
		}
	}

	function handleOverlayClick(event) {
		if (event.target === event.currentTarget) {
			handleClose();
		}
	}
</script>

{#if open}
	<!-- svelte-ignore a11y-no-noninteractive-element-interactions -->
	<div 
		class="modal-overlay" 
		class:fullscreen
		on:click={handleOverlayClick}
		on:keydown={handleKeydown}
		role="dialog"
		aria-modal="true"
		aria-labelledby={title ? 'modal-title' : undefined}
	>
		<div class="modal-container" class:fullscreen>
			{#if showHeader}
				<div class="modal-header">
					<h3 id="modal-title" class="modal-title">{title}</h3>
					<button 
						class="modal-close"
						on:click={handleClose}
						aria-label="Close modal"
					>
						<svg fill="none" viewBox="0 0 24 24" stroke="currentColor">
							<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12"></path>
						</svg>
					</button>
				</div>
			{/if}
			
			<div class="modal-content">
				<slot />
			</div>
			
			{#if showFooter}
				<div class="modal-footer">
					<slot name="footer" />
				</div>
			{/if}
		</div>
	</div>
{/if}

<style>
	.modal-title {
		font-size: 1.125rem;
		font-weight: 600;
		margin: 0;
	}

	.modal-close {
		background: none;
		border: none;
		color: var(--text-muted);
		cursor: pointer;
		padding: var(--space-1);
		border-radius: var(--radius-sm);
		transition: color 0.2s ease, background-color 0.2s ease;
	}

	.modal-close:hover {
		color: var(--text);
		background: var(--bg-elev);
	}

	.modal-close svg {
		width: 20px;
		height: 20px;
	}
</style>
