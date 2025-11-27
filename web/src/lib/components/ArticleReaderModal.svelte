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
	showFooter={false}
	fullscreen={true}
	showHeader={false}
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
			<button class="button secondary" on:click={handleClose}>Close</button>
		</div>
	{:else if article}
		<article class="reader-container">
			<header class="reader-header">
				<h1>{article.title}</h1>
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
			</header>

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
		</article>

		<!-- Floating Toolbar -->
		<div class="floating-toolbar">
			<button 
				class="toolbar-button primary"
				on:click={handleOpenOriginal}
				disabled={!article?.url}
				title="Open Original"
			>
				<svg fill="none" viewBox="0 0 24 24" stroke="currentColor">
					<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M10 6H6a2 2 0 00-2 2v10a2 2 0 002 2h10a2 2 0 002-2v-4M14 4h6m0 0v6m0-6L10 14"></path>
				</svg>
			</button>
			<button 
				class="toolbar-button secondary"
				on:click={handleClose}
				title="Close Reader"
			>
				<svg fill="none" viewBox="0 0 24 24" stroke="currentColor">
					<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12"></path>
				</svg>
			</button>
		</div>
	{:else}
		<div class="reader-state">
			<p>Select an article to start reading.</p>
		</div>
	{/if}
</Modal>

<style>
	.reader-state {
		display: flex;
		flex-direction: column;
		align-items: center;
		justify-content: center;
		padding: var(--space-6);
		gap: var(--space-3);
		text-align: center;
		color: var(--text-muted);
		min-height: 50vh;
		width: 100%;
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

	.reader-container {
		max-width: 800px;
		width: 100%;
		margin: 0 auto;
		padding-bottom: 80px; /* Space for floating toolbar */
	}

	.reader-header {
		margin-bottom: var(--space-6);
		text-align: center;
	}

	.reader-header h1 {
		font-size: 2.5rem;
		line-height: 1.3;
		margin-bottom: var(--space-4);
		color: var(--text);
	}

	.reader-metadata {
		display: flex;
		justify-content: center;
		align-items: center;
		gap: var(--space-4);
		font-size: 0.875rem;
		color: var(--text-muted);
	}

	.meta-left {
		display: flex;
		align-items: center;
		gap: var(--space-2);
	}

	.meta-source {
		font-weight: 600;
		color: var(--text);
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
		padding: var(--space-5);
		margin-bottom: var(--space-6);
		font-size: 1.1rem;
		line-height: 1.7;
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
		color: var(--text);
	}

	.reader-body {
		font-size: 1.25rem;
		line-height: 1.8;
		color: var(--text);
		font-family: 'Georgia', var(--font-body); /* Optimize for reading */
	}

	.reader-body.empty {
		color: var(--text-muted);
		text-align: center;
		padding: var(--space-6);
	}

	/* Content Styling */
	.reader-body :global(h1),
	.reader-body :global(h2),
	.reader-body :global(h3) {
		margin-top: 2em;
		margin-bottom: 0.8em;
		line-height: 1.3;
	}

	.reader-body :global(p) {
		margin-bottom: 1.5em;
	}

	.reader-body :global(img) {
		max-width: 100%;
		height: auto;
		display: block;
		margin: var(--space-5) auto;
		border-radius: var(--radius-sm);
	}

	.reader-body :global(pre) {
		background: var(--bg-elev);
		padding: var(--space-4);
		border-radius: var(--radius-sm);
		overflow-x: auto;
		font-family: var(--font-mono);
		font-size: 0.9em;
		margin: var(--space-4) 0;
		border: 1px solid var(--border);
	}

	.reader-body :global(code) {
		font-family: var(--font-mono);
		background: var(--bg-elev);
		padding: 0.2em 0.4em;
		border-radius: 4px;
		font-size: 0.9em;
	}

	.reader-body :global(pre code) {
		background: none;
		padding: 0;
		border-radius: 0;
	}

	.reader-body :global(a) {
		color: var(--primary);
		text-decoration: underline;
		text-underline-offset: 2px;
	}

	.reader-body :global(blockquote) {
		border-left: 4px solid var(--primary);
		margin: var(--space-5) 0;
		padding: var(--space-2) 0 var(--space-2) var(--space-5);
		color: var(--text-muted);
		font-style: italic;
		background: linear-gradient(to right, var(--bg-elev), transparent);
	}

	.reader-body :global(ul),
	.reader-body :global(ol) {
		margin-bottom: 1.5em;
		padding-left: 1.5em;
	}

	.reader-body :global(li) {
		margin-bottom: 0.5em;
	}

	/* Floating Toolbar */
	.floating-toolbar {
		position: fixed;
		bottom: var(--space-6);
		right: var(--space-6);
		display: flex;
		flex-direction: column;
		gap: var(--space-3);
		z-index: 100;
	}

	.toolbar-button {
		width: 56px;
		height: 56px;
		border-radius: 50%;
		display: flex;
		align-items: center;
		justify-content: center;
		border: none;
		cursor: pointer;
		box-shadow: var(--shadow-md);
		transition: transform 0.2s ease, box-shadow 0.2s ease;
		background: var(--bg);
		color: var(--text);
	}

	.toolbar-button:hover {
		transform: translateY(-2px);
		box-shadow: 0 6px 16px rgba(0,0,0,0.2);
	}

	.toolbar-button.primary {
		background: var(--primary);
		color: var(--primary-contrast);
	}

	.toolbar-button.primary:hover {
		background: color-mix(in srgb, var(--primary) 90%, black);
	}

	.toolbar-button svg {
		width: 24px;
		height: 24px;
	}

	@media (max-width: 768px) {
		.reader-header h1 {
			font-size: 1.8rem;
		}

		.reader-body {
			font-size: 1.1rem;
		}

		.floating-toolbar {
			flex-direction: row;
			bottom: var(--space-4);
			right: var(--space-4);
		}

		.toolbar-button {
			width: 48px;
			height: 48px;
		}
	}
</style>
