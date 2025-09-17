<script>
	import { createEventDispatcher } from 'svelte';
	import Modal from './Modal.svelte';
	import { feeds } from '../api.js';
	import { feedsStore, toast } from '../stores.js';

	export let open = false;

	const dispatch = createEventDispatcher();

	let url = '';
	let loading = false;
	let error = '';

	// Clear form when modal opens/closes
	$: if (open) {
		url = '';
		error = '';
	}

	function validateUrl(urlString) {
		try {
			const parsedUrl = new URL(urlString);
			return ['http:', 'https:'].includes(parsedUrl.protocol);
		} catch {
			return false;
		}
	}

	async function handleSubmit() {
		// Validate URL
		if (!url.trim()) {
			error = 'URL is required';
			return;
		}

		if (!validateUrl(url.trim())) {
			error = 'Please enter a valid HTTP or HTTPS URL';
			return;
		}

		loading = true;
		error = '';

		try {
			const newFeed = await feeds.subscribe(url.trim());
			
			// Add to feeds store
			feedsStore.addFeed(newFeed);
			
			// Show success message
			toast.success(`Successfully subscribed to ${newFeed.title || 'feed'}`);
			
			// Close modal
			handleClose();
		} catch (err) {
			error = err.message || 'Failed to subscribe to feed';
		} finally {
			loading = false;
		}
	}

	function handleClose() {
		dispatch('close');
	}

	function clearError() {
		if (error) {
			error = '';
		}
	}

	function handleKeydown(event) {
		if (event.key === 'Enter' && !loading) {
			event.preventDefault();
			handleSubmit();
		}
	}
</script>

<Modal {open} title="Add RSS Subscription" showFooter={true} on:close={handleClose}>
	<form on:submit|preventDefault={handleSubmit}>
		<div class="form-group">
			<label for="feed-url" class="form-label">RSS Feed URL</label>
			<input
				id="feed-url"
				type="url"
				class="input {error ? 'error' : ''}"
				bind:value={url}
				on:input={clearError}
				on:keydown={handleKeydown}
				placeholder="https://example.com/feed.xml"
				disabled={loading}
				autocomplete="url"
			/>
			{#if error}
				<div class="form-error">{error}</div>
			{/if}
			<div class="form-hint">
				Enter the URL of an RSS or Atom feed to subscribe to it.
			</div>
		</div>
	</form>

	<div slot="footer" class="modal-actions">
		<button 
			type="button" 
			class="button secondary" 
			on:click={handleClose}
			disabled={loading}
		>
			Cancel
		</button>
		<button 
			type="button" 
			class="button primary {loading ? 'loading' : ''}" 
			on:click={handleSubmit}
			disabled={loading || !url.trim()}
		>
			{loading ? '' : 'Subscribe'}
		</button>
	</div>
</Modal>

<style>
	.form-hint {
		margin-top: var(--space-1);
		font-size: 0.75rem;
		color: var(--text-muted);
		line-height: 1.4;
	}

	.modal-actions {
		display: flex;
		gap: var(--space-2);
		justify-content: flex-end;
	}
</style>
