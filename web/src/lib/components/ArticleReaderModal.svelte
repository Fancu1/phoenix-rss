<script>
	import { createEventDispatcher } from 'svelte';
	import Modal from '$lib/components/Modal.svelte';

	export let open = false;
	export let article = null;
	export let loading = false;
	export let error = '';

	const dispatch = createEventDispatcher();

	function handleClose() {
		dispatch('close');
	}

	function handleOpenOriginal() {
		if (article?.url) {
			dispatch('openOriginal');
		}
	}

	function formatDateTime(dateString) {
		if (!dateString) return '';
		try {
			return new Date(dateString).toLocaleString();
		} catch (err) {
			return dateString;
		}
	}

	function getSourceDomain(url) {
		if (!url) return '';
		try {
			return new URL(url).hostname.replace('www.', '');
		} catch (err) {
			return url;
		}
	}
</script>

<Modal 
	open={open} 
	title={article ? article.title : 'Article'} 
	showFooter={!loading && !error && !!article}
	on:close={handleClose}
>
	{#if loading}
		<div class="reader-state">
			<div class="reader-spinner"></div>
			<p>Loading article...</p>
		</div>
	{:else if error}
		<div class="reader-state error">
			<p>{error}</p>
		</div>
	{:else if article}
		<div class="reader-metadata">
			<div class="meta-left">
				{#if article.url}
					<span class="meta-source">{getSourceDomain(article.url)}</span>
				{/if}
				{#if article.published_at}
					<span class="meta-separator">•</span>
					<time datetime={article.published_at}>{formatDateTime(article.published_at)}</time>
				{/if}
			</div>
			{#if article.processing_model}
				<span class="meta-model">AI: {article.processing_model}</span>
			{/if}
		</div>

		{#if article.summary}
			<section class="reader-summary">
				<h4>AI Summary</h4>
				<p>{article.summary}</p>
			</section>
		{/if}

		<div class="reader-body" class:empty={!article.content}>
			{#if article.content}
				<!-- eslint-disable-next-line svelte/no-at-html-tags -->
				{@html article.content}
			{:else}
				<p>No content available for this article.</p>
			{/if}
		</div>
	{:else}
		<div class="reader-state">
			<p>Select an article to start reading.</p>
		</div>
	{/if}

	<svelte:fragment slot="footer">
		{#if article}
			<div class="reader-actions">
				<button 
					class="button primary"
					on:click={handleOpenOriginal}
					disabled={!article?.url}
				>
					Open Original
				</button>
				<div class="reader-action-group">
					<button class="button secondary" disabled title="Coming soon">Mark Read</button>
					<button class="button secondary" disabled title="Coming soon">Star</button>
				</div>
			</div>
		{/if}
	</svelte:fragment>
</Modal>

<style>
	.reader-state {
		display: flex;
		flex-direction: column;
		align-items: center;
		justify-content: center;
		padding: var(--space-5);
		gap: var(--space-3);
		text-align: center;
		color: var(--text-muted);
	}

	.reader-state.error {
		color: var(--danger);
	}

	.reader-spinner {
		width: 28px;
		height: 28px;
		border: 3px solid var(--border);
		border-top: 3px solid var(--primary);
		border-radius: 50%;
		animation: spin 1s linear infinite;
	}

	.reader-metadata {
		display: flex;
		justify-content: space-between;
		align-items: center;
		font-size: 0.875rem;
		color: var(--text-muted);
		margin-bottom: var(--space-4);
	}

	.meta-left {
		display: flex;
		align-items: center;
		gap: var(--space-2);
	}

	.meta-source {
		font-weight: 600;
		color: var(--text-muted);
		text-transform: uppercase;
		letter-spacing: 0.05em;
	}

	.meta-separator {
		color: var(--border-strong);
	}

	.meta-model {
		font-size: 0.75rem;
		background: var(--bg-elev);
		padding: var(--space-1) var(--space-2);
		border-radius: var(--radius-sm);
		border: 1px solid var(--border);
	}

	.reader-summary {
		background: var(--bg-elev);
		border: 1px solid var(--border);
		border-radius: var(--radius-md);
		padding: var(--space-4);
		margin-bottom: var(--space-4);
	}

	.reader-summary h4 {
		margin: 0 0 var(--space-2) 0;
		font-size: 0.875rem;
		letter-spacing: 0.05em;
		text-transform: uppercase;
		color: var(--text-muted);
	}

	.reader-summary p {
		margin: 0;
		line-height: 1.6;
		color: var(--text);
	}

	.reader-body {
		max-height: 60vh;
		overflow-y: auto;
		padding-right: var(--space-2);
		font-size: 1rem;
		line-height: 1.6;
		color: var(--text);
	}

	.reader-body.empty {
		color: var(--text-muted);
	}

	.reader-body :global(img) {
		max-width: 100%;
		height: auto;
		display: block;
		margin: var(--space-3) auto;
	}

	.reader-body :global(pre) {
		background: var(--bg-elev);
		padding: var(--space-3);
		border-radius: var(--radius-sm);
		overflow-x: auto;
		font-family: var(--font-mono);
	}

	.reader-body :global(code) {
		font-family: var(--font-mono);
	}

	.reader-body :global(a) {
		color: var(--primary);
		text-decoration: underline;
		word-break: break-word;
	}

	.reader-body :global(blockquote) {
		border-left: 3px solid var(--border);
		margin: var(--space-3) 0;
		padding: 0 0 0 var(--space-3);
		color: var(--text-muted);
	}

	.reader-actions {
		display: flex;
		justify-content: space-between;
		align-items: center;
		gap: var(--space-3);
	}

	.reader-action-group {
		display: flex;
		gap: var(--space-2);
	}

	@media (max-width: 768px) {
		.reader-body {
			max-height: 50vh;
		}

		.reader-actions {
			flex-direction: column;
			align-items: stretch;
		}

		.reader-action-group {
			justify-content: stretch;
		}
	}
</style>
