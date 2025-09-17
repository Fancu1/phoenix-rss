<script>
	import { uiStore } from '../stores.js';

	// SVG icons for different toast types
	const icons = {
		success: `<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 13l4 4L19 7"></path>`,
		error: `<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12"></path>`,
		warning: `<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-2.5L13.732 4c-.77-.833-1.964-.833-2.732 0L4.082 16.5c-.77.833.192 2.5 1.732 2.5z"></path>`,
		info: `<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M13 16h-1v-4h-1m1-4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z"></path>`
	};

	function closeToast(id) {
		uiStore.removeToast(id);
	}
</script>

<div class="toast-container">
	{#each $uiStore.toasts as toast (toast.id)}
		<div class="toast {toast.type}">
			<svg class="toast-icon" fill="none" viewBox="0 0 24 24" stroke="currentColor">
				{@html icons[toast.type]}
			</svg>
			<span class="toast-message">{toast.message}</span>
			<button 
				class="toast-close" 
				on:click={() => closeToast(toast.id)}
				aria-label="Close notification"
			>
				<svg fill="none" viewBox="0 0 24 24" stroke="currentColor">
					<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12"></path>
				</svg>
			</button>
		</div>
	{/each}
</div>

<style>
	.toast-icon {
		width: 20px;
		height: 20px;
		flex-shrink: 0;
	}

	.toast-message {
		flex: 1;
		font-size: 0.875rem;
	}

	.toast-close {
		background: none;
		border: none;
		color: inherit;
		cursor: pointer;
		padding: 0;
		width: 20px;
		height: 20px;
		flex-shrink: 0;
		opacity: 0.7;
		transition: opacity 0.2s ease;
	}

	.toast-close:hover {
		opacity: 1;
	}

	.toast-close svg {
		width: 16px;
		height: 16px;
	}
</style>
