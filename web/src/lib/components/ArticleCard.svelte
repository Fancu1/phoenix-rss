<script>
	export let article;
	export let showFeedName = false;

	// Format the published date
	function formatDate(dateString) {
		if (!dateString) return '';
		
		const date = new Date(dateString);
		const now = new Date();
		const diffTime = now - date;
		const diffDays = Math.floor(diffTime / (1000 * 60 * 60 * 24));
		
		if (diffDays === 0) {
			return 'Today';
		} else if (diffDays === 1) {
			return 'Yesterday';
		} else if (diffDays < 7) {
			return `${diffDays} days ago`;
		} else {
			return date.toLocaleDateString('en-US', { 
				year: 'numeric', 
				month: 'short', 
				day: 'numeric' 
			});
		}
	}

	// Get source domain from URL
	function getSourceDomain(url) {
		try {
			return new URL(url).hostname.replace('www.', '');
		} catch {
			return url;
		}
	}

	// Open article in new tab
	function handleClick() {
		if (article.url) {
			window.open(article.url, '_blank', 'noopener,noreferrer');
		}
	}

	// Truncate text with ellipsis
	function truncateText(text, maxLength = 200) {
		if (!text || text.length <= maxLength) return text;
		return text.slice(0, maxLength).trim() + '...';
	}
</script>

<!-- svelte-ignore a11y-no-static-element-interactions -->
<article 
	class="article-card" 
	on:click={handleClick}
	on:keydown={(e) => e.key === 'Enter' && handleClick()}
	tabindex="0"
	role="button"
	aria-label="Open article: {article.title}"
>
	<div class="article-header">
		<h3 class="article-title">{article.title}</h3>
		
		<div class="article-meta">
			<time class="article-date" datetime={article.published_at}>
				{formatDate(article.published_at)}
			</time>
			
			{#if showFeedName && article.feed_name}
				<span class="article-separator">•</span>
				<span class="article-source">{article.feed_name}</span>
			{:else if article.url}
				<span class="article-separator">•</span>
				<span class="article-source">{getSourceDomain(article.url)}</span>
			{/if}
		</div>
	</div>

	{#if article.description}
		<p class="article-description">
			{truncateText(article.description)}
		</p>
	{/if}

	{#if article.summary}
		<div class="ai-summary">
			<div class="ai-badge">
				<svg fill="none" viewBox="0 0 24 24" stroke="currentColor">
					<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9.663 17h4.673M12 3v1m6.364 1.636l-.707.707M21 12h-1M4 12H3m3.343-5.657l-.707-.707m2.828 9.9a5 5 0 117.072 0l-.548.547A3.374 3.374 0 0014 18.469V19a2 2 0 11-4 0v-.531c0-.895-.356-1.754-.988-2.386l-.548-.547z"></path>
				</svg>
				AI Summary
			</div>
			<p class="ai-content">
				{truncateText(article.summary, 150)}
			</p>
		</div>
	{:else if article.processed_at === null}
		<div class="ai-processing">
			<div class="processing-badge">
				<div class="processing-spinner"></div>
				Processing...
			</div>
		</div>
	{/if}

	<!-- Reading status indicators -->
	<div class="article-status">
		{#if article.read}
			<span class="status-indicator read" title="Read">
				<svg fill="none" viewBox="0 0 24 24" stroke="currentColor">
					<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 13l4 4L19 7"></path>
				</svg>
			</span>
		{/if}
		
		{#if article.starred}
			<span class="status-indicator starred" title="Starred">
				<svg fill="currentColor" viewBox="0 0 24 24">
					<path d="M12 2l3.09 6.26L22 9.27l-5 4.87 1.18 6.88L12 17.77l-6.18 3.25L7 14.14 2 9.27l6.91-1.01L12 2z"></path>
				</svg>
			</span>
		{/if}
	</div>
</article>

<style>
	.article-card {
		background: var(--bg-elev);
		border: 1px solid var(--border);
		border-radius: var(--radius-md);
		padding: var(--space-4);
		margin-bottom: var(--space-3);
		cursor: pointer;
		transition: all 0.2s ease;
		position: relative;
		box-shadow: var(--shadow-sm);
	}

	.article-card:hover {
		box-shadow: var(--shadow-md);
		border-color: var(--primary);
		transform: translateY(-1px);
	}

	.article-card:focus {
		outline: 2px solid var(--primary);
		outline-offset: 2px;
	}

	.article-header {
		margin-bottom: var(--space-3);
	}

	.article-title {
		margin: 0 0 var(--space-2) 0;
		font-size: 1.125rem;
		font-weight: 600;
		line-height: 1.4;
		color: var(--text);
		display: -webkit-box;
		-webkit-line-clamp: 2;
		-webkit-box-orient: vertical;
		overflow: hidden;
	}

	.article-meta {
		display: flex;
		align-items: center;
		gap: var(--space-1);
		font-size: 0.75rem;
		color: var(--text-muted);
	}

	.article-separator {
		color: var(--text-muted);
	}

	.article-date {
		font-weight: 500;
	}

	.article-source {
		color: var(--text-muted);
	}

	.article-description {
		margin: 0 0 var(--space-3) 0;
		font-size: 0.875rem;
		line-height: 1.5;
		color: var(--text-muted);
		display: -webkit-box;
		-webkit-line-clamp: 3;
		-webkit-box-orient: vertical;
		overflow: hidden;
	}

	.ai-summary {
		background: linear-gradient(135deg, var(--primary) 0%, color-mix(in srgb, var(--primary) 80%, var(--bg)) 100%);
		border-radius: var(--radius-sm);
		padding: var(--space-3);
		margin-bottom: var(--space-2);
	}

	.ai-badge {
		display: flex;
		align-items: center;
		gap: var(--space-1);
		font-size: 0.625rem;
		font-weight: 600;
		text-transform: uppercase;
		letter-spacing: 0.05em;
		color: var(--primary-contrast);
		margin-bottom: var(--space-2);
	}

	.ai-badge svg {
		width: 12px;
		height: 12px;
	}

	.ai-content {
		margin: 0;
		font-size: 0.875rem;
		line-height: 1.4;
		color: var(--primary-contrast);
		font-weight: 500;
	}

	.ai-processing {
		margin-bottom: var(--space-2);
	}

	.processing-badge {
		display: flex;
		align-items: center;
		gap: var(--space-2);
		font-size: 0.75rem;
		color: var(--text-muted);
		font-weight: 500;
	}

	.processing-spinner {
		width: 12px;
		height: 12px;
		border: 2px solid var(--border);
		border-top: 2px solid var(--primary);
		border-radius: 50%;
		animation: spin 1s linear infinite;
	}

	.article-status {
		position: absolute;
		top: var(--space-3);
		right: var(--space-3);
		display: flex;
		gap: var(--space-1);
	}

	.status-indicator {
		width: 16px;
		height: 16px;
		opacity: 0.6;
	}

	.status-indicator.read {
		color: #10b981;
	}

	.status-indicator.starred {
		color: #f59e0b;
	}

	.status-indicator svg {
		width: 100%;
		height: 100%;
	}

	@media (max-width: 768px) {
		.article-card {
			padding: var(--space-3);
			margin-bottom: var(--space-2);
		}

		.article-title {
			font-size: 1rem;
		}

		.article-description {
			-webkit-line-clamp: 2;
		}
	}
</style>
