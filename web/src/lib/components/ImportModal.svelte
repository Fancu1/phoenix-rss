<script>
	import { createEventDispatcher } from 'svelte';
	import Modal from './Modal.svelte';
	import { feeds } from '../api.js';
	import { feedsStore, toast } from '../stores.js';

	export let open = false;

	const dispatch = createEventDispatcher();

	// State machine: 'upload' | 'preview' | 'importing' | 'result'
	let step = 'upload';
	let loading = false;
	let error = '';
	
	// Preview data
	let previewData = null;
	let selectedFeeds = [];
	
	// Import result
	let importResult = null;

	// Reset state when modal opens/closes
	$: if (open) {
		resetState();
	}

	function resetState() {
		step = 'upload';
		loading = false;
		error = '';
		previewData = null;
		selectedFeeds = [];
		importResult = null;
	}

	async function handleFileSelect(event) {
		const file = event.target.files?.[0];
		if (!file) return;

		// Validate file type
		if (!file.name.endsWith('.opml') && !file.name.endsWith('.xml')) {
			error = 'Please select an OPML or XML file';
			return;
		}

		loading = true;
		error = '';

		try {
			previewData = await feeds.previewImport(file);
			selectedFeeds = previewData.to_import.map((_, i) => i);
			step = 'preview';
		} catch (err) {
			error = err.message || 'Failed to parse OPML file';
		} finally {
			loading = false;
		}
	}

	function toggleFeed(index) {
		if (selectedFeeds.includes(index)) {
			selectedFeeds = selectedFeeds.filter(i => i !== index);
		} else {
			selectedFeeds = [...selectedFeeds, index];
		}
	}

	function selectAll() {
		selectedFeeds = previewData.to_import.map((_, i) => i);
	}

	function deselectAll() {
		selectedFeeds = [];
	}

	async function handleImport() {
		if (selectedFeeds.length === 0) {
			error = 'Please select at least one feed to import';
			return;
		}

		loading = true;
		error = '';
		step = 'importing';

		try {
			const feedsToImport = selectedFeeds.map(i => previewData.to_import[i]);
			importResult = await feeds.importFeeds(feedsToImport);
			step = 'result';
			
			// Refresh the feeds list
			const updatedFeeds = await feeds.list();
			feedsStore.setFeeds(updatedFeeds);
		} catch (err) {
			error = err.message || 'Failed to import feeds';
			step = 'preview';
		} finally {
			loading = false;
		}
	}

	function handleClose() {
		if (step === 'result' && importResult?.imported > 0) {
			toast.success(`Successfully imported ${importResult.imported} feed(s)`);
		}
		dispatch('close');
	}

	function handleBackToUpload() {
		resetState();
	}
</script>

<Modal {open} title="Import Subscriptions" showFooter={true} on:close={handleClose}>
	{#if step === 'upload'}
		<div class="upload-section">
			<div class="upload-info">
				<svg class="upload-icon" fill="none" viewBox="0 0 24 24" stroke="currentColor">
					<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M7 16a4 4 0 01-.88-7.903A5 5 0 1115.9 6L16 6a5 5 0 011 9.9M15 13l-3-3m0 0l-3 3m3-3v12"></path>
				</svg>
				<h4>Upload OPML File</h4>
				<p class="text-muted">Select an OPML file exported from another RSS reader</p>
			</div>
			
			<label class="file-input-label" class:disabled={loading}>
				<input
					type="file"
					accept=".opml,.xml"
					on:change={handleFileSelect}
					disabled={loading}
				/>
				<span class="file-input-button">
					{#if loading}
						<span class="spinner"></span>
						Parsing...
					{:else}
						Choose File
					{/if}
				</span>
			</label>

			{#if error}
				<div class="form-error">{error}</div>
			{/if}

			<div class="supported-formats">
				<span class="text-muted">Supported formats: OPML, XML</span>
			</div>
		</div>

	{:else if step === 'preview'}
		<div class="preview-section">
			<div class="preview-summary">
				<div class="summary-item">
					<span class="summary-number">{previewData.to_import.length}</span>
					<span class="summary-label">New feeds</span>
				</div>
				<div class="summary-item duplicates">
					<span class="summary-number">{previewData.duplicates.length}</span>
					<span class="summary-label">Already subscribed</span>
				</div>
			</div>

			{#if previewData.to_import.length > 0}
				<div class="feed-list-header">
					<span>Feeds to import ({selectedFeeds.length} selected)</span>
					<div class="selection-controls">
						<button type="button" class="link-button" on:click={selectAll}>Select all</button>
						<span class="separator">|</span>
						<button type="button" class="link-button" on:click={deselectAll}>Deselect all</button>
					</div>
				</div>
				
				<div class="feed-list">
					{#each previewData.to_import as feed, index}
						<label class="feed-item">
							<input
								type="checkbox"
								checked={selectedFeeds.includes(index)}
								on:change={() => toggleFeed(index)}
							/>
							<div class="feed-info">
								<span class="feed-title">{feed.title || 'Untitled'}</span>
								<span class="feed-url">{feed.url}</span>
							</div>
						</label>
					{/each}
				</div>
			{:else}
				<div class="empty-state">
					<p>No new feeds to import. All feeds in this file are already in your subscriptions.</p>
				</div>
			{/if}

			{#if previewData.duplicates.length > 0}
				<details class="duplicates-section">
					<summary>
						{previewData.duplicates.length} duplicate(s) will be skipped
					</summary>
					<div class="duplicate-list">
						{#each previewData.duplicates as feed}
							<div class="duplicate-item">
								<span class="feed-title">{feed.title || 'Untitled'}</span>
								<span class="feed-url">{feed.url}</span>
							</div>
						{/each}
					</div>
				</details>
			{/if}

			{#if error}
				<div class="form-error">{error}</div>
			{/if}
		</div>

	{:else if step === 'importing'}
		<div class="importing-section">
			<div class="spinner large"></div>
			<p>Importing {selectedFeeds.length} feed(s)...</p>
			<p class="text-muted">This may take a moment</p>
		</div>

	{:else if step === 'result'}
		<div class="result-section">
			<div class="result-icon success">
				<svg fill="none" viewBox="0 0 24 24" stroke="currentColor">
					<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 13l4 4L19 7"></path>
				</svg>
			</div>
			
			<h4>Import Complete</h4>
			
			<div class="result-stats">
				<div class="stat-item success">
					<span class="stat-number">{importResult.imported}</span>
					<span class="stat-label">Imported</span>
				</div>
				{#if importResult.failed > 0}
					<div class="stat-item error">
						<span class="stat-number">{importResult.failed}</span>
						<span class="stat-label">Failed</span>
					</div>
				{/if}
			</div>

			{#if importResult.failed_urls?.length > 0}
				<details class="failed-section">
					<summary>View failed imports</summary>
					<div class="failed-list">
						{#each importResult.failed_urls as url}
							<div class="failed-item">{url}</div>
						{/each}
					</div>
				</details>
			{/if}
		</div>
	{/if}

	<div slot="footer" class="modal-actions">
		{#if step === 'upload'}
			<button type="button" class="button secondary" on:click={handleClose}>
				Cancel
			</button>
		{:else if step === 'preview'}
			<button type="button" class="button secondary" on:click={handleBackToUpload}>
				Back
			</button>
			<button 
				type="button" 
				class="button primary"
				on:click={handleImport}
				disabled={selectedFeeds.length === 0}
			>
				Import {selectedFeeds.length} Feed(s)
			</button>
		{:else if step === 'result'}
			<button type="button" class="button primary" on:click={handleClose}>
				Done
			</button>
		{/if}
	</div>
</Modal>

<style>
	.upload-section {
		display: flex;
		flex-direction: column;
		align-items: center;
		gap: var(--space-4);
		padding: var(--space-4) 0;
	}

	.upload-info {
		text-align: center;
	}

	.upload-icon {
		width: 48px;
		height: 48px;
		color: var(--primary);
		margin-bottom: var(--space-2);
	}

	.upload-info h4 {
		margin: 0 0 var(--space-1) 0;
		color: var(--text);
	}

	.upload-info p {
		margin: 0;
		font-size: 0.875rem;
	}

	.file-input-label {
		cursor: pointer;
	}

	.file-input-label.disabled {
		cursor: not-allowed;
		opacity: 0.6;
	}

	.file-input-label input {
		display: none;
	}

	.file-input-button {
		display: inline-flex;
		align-items: center;
		gap: var(--space-2);
		padding: var(--space-3) var(--space-5);
		background: var(--primary);
		color: var(--primary-contrast);
		border-radius: var(--radius-md);
		font-weight: 500;
		transition: background-color 0.2s ease;
	}

	.file-input-label:not(.disabled):hover .file-input-button {
		background: var(--primary-hover);
	}

	.supported-formats {
		font-size: 0.75rem;
	}

	/* Preview Section */
	.preview-section {
		display: flex;
		flex-direction: column;
		gap: var(--space-4);
	}

	.preview-summary {
		display: flex;
		gap: var(--space-4);
		padding: var(--space-3);
		background: var(--bg);
		border-radius: var(--radius-md);
	}

	.summary-item {
		display: flex;
		flex-direction: column;
		align-items: center;
		flex: 1;
	}

	.summary-number {
		font-size: 1.5rem;
		font-weight: 700;
		color: var(--primary);
	}

	.summary-item.duplicates .summary-number {
		color: var(--text-muted);
	}

	.summary-label {
		font-size: 0.75rem;
		color: var(--text-muted);
	}

	.feed-list-header {
		display: flex;
		justify-content: space-between;
		align-items: center;
		font-size: 0.875rem;
		color: var(--text-muted);
	}

	.selection-controls {
		display: flex;
		gap: var(--space-2);
		align-items: center;
	}

	.link-button {
		background: none;
		border: none;
		color: var(--primary);
		cursor: pointer;
		font-size: 0.75rem;
		padding: 0;
	}

	.link-button:hover {
		text-decoration: underline;
	}

	.separator {
		color: var(--border);
	}

	.feed-list {
		max-height: 300px;
		overflow-y: auto;
		border: 1px solid var(--border);
		border-radius: var(--radius-md);
	}

	.feed-item {
		display: flex;
		align-items: flex-start;
		gap: var(--space-3);
		padding: var(--space-3);
		cursor: pointer;
		border-bottom: 1px solid var(--border);
		transition: background-color 0.15s ease;
	}

	.feed-item:last-child {
		border-bottom: none;
	}

	.feed-item:hover {
		background: var(--bg);
	}

	.feed-item input[type="checkbox"] {
		margin-top: 2px;
		flex-shrink: 0;
	}

	.feed-info {
		display: flex;
		flex-direction: column;
		gap: 2px;
		min-width: 0;
	}

	.feed-title {
		font-weight: 500;
		color: var(--text);
		white-space: nowrap;
		overflow: hidden;
		text-overflow: ellipsis;
	}

	.feed-url {
		font-size: 0.75rem;
		color: var(--text-muted);
		white-space: nowrap;
		overflow: hidden;
		text-overflow: ellipsis;
	}

	.empty-state {
		text-align: center;
		padding: var(--space-6);
		color: var(--text-muted);
	}

	.duplicates-section {
		font-size: 0.875rem;
	}

	.duplicates-section summary {
		cursor: pointer;
		color: var(--text-muted);
		padding: var(--space-2) 0;
	}

	.duplicate-list {
		padding: var(--space-2);
		background: var(--bg);
		border-radius: var(--radius-sm);
		max-height: 150px;
		overflow-y: auto;
	}

	.duplicate-item {
		display: flex;
		flex-direction: column;
		gap: 2px;
		padding: var(--space-2);
		border-bottom: 1px solid var(--border);
	}

	.duplicate-item:last-child {
		border-bottom: none;
	}

	/* Importing Section */
	.importing-section {
		display: flex;
		flex-direction: column;
		align-items: center;
		gap: var(--space-3);
		padding: var(--space-6);
	}

	/* Result Section */
	.result-section {
		display: flex;
		flex-direction: column;
		align-items: center;
		gap: var(--space-4);
		padding: var(--space-4);
	}

	.result-icon {
		width: 64px;
		height: 64px;
		border-radius: 50%;
		display: flex;
		align-items: center;
		justify-content: center;
	}

	.result-icon.success {
		background: rgba(34, 197, 94, 0.1);
		color: rgb(34, 197, 94);
	}

	.result-icon svg {
		width: 32px;
		height: 32px;
	}

	.result-section h4 {
		margin: 0;
		color: var(--text);
	}

	.result-stats {
		display: flex;
		gap: var(--space-6);
	}

	.stat-item {
		display: flex;
		flex-direction: column;
		align-items: center;
	}

	.stat-number {
		font-size: 1.5rem;
		font-weight: 700;
	}

	.stat-item.success .stat-number {
		color: rgb(34, 197, 94);
	}

	.stat-item.error .stat-number {
		color: rgb(239, 68, 68);
	}

	.stat-label {
		font-size: 0.75rem;
		color: var(--text-muted);
	}

	.failed-section {
		width: 100%;
		font-size: 0.875rem;
	}

	.failed-section summary {
		cursor: pointer;
		color: var(--text-muted);
	}

	.failed-list {
		padding: var(--space-2);
		background: var(--bg);
		border-radius: var(--radius-sm);
		max-height: 100px;
		overflow-y: auto;
	}

	.failed-item {
		padding: var(--space-1);
		font-size: 0.75rem;
		color: rgb(239, 68, 68);
		word-break: break-all;
	}

	/* Spinner */
	.spinner {
		width: 16px;
		height: 16px;
		border: 2px solid var(--border);
		border-top-color: var(--primary);
		border-radius: 50%;
		animation: spin 0.8s linear infinite;
	}

	.spinner.large {
		width: 40px;
		height: 40px;
		border-width: 3px;
	}

	@keyframes spin {
		to {
			transform: rotate(360deg);
		}
	}

	/* Modal Actions */
	.modal-actions {
		display: flex;
		gap: var(--space-2);
		justify-content: flex-end;
	}
</style>

