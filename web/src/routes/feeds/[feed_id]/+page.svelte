<script>
	import { goto } from '$app/navigation';
	import { page } from '$app/stores';
	import { articles as articleApi, feeds } from '$lib/api.js';
	import AddSubscriptionModal from '$lib/components/AddSubscriptionModal.svelte';
	import ArticleCard from '$lib/components/ArticleCard.svelte';
	import ArticleReaderModal from '$lib/components/ArticleReaderModal.svelte';
	import Modal from '$lib/components/Modal.svelte';
	import NavBar from '$lib/components/NavBar.svelte';
	import Pagination from '$lib/components/Pagination.svelte';
	import SideBar from '$lib/components/SideBar.svelte';
	import { authStore, feedsStore, toast, uiStore } from '$lib/stores.js';
	import { onMount } from 'svelte';

	const PAGE_SIZE = 8;

	let showAddModal = false;
	let showUnsubscribeModal = false;
	let loading = true;
	let feedArticles = [];
	let currentFeed = null;
	let fetchingArticles = false;
	let readerOpen = false;
	let readerLoading = false;
	let readerError = '';
	let readerArticle = null;
	let readerArticleId = null;

	// Pagination state
	let pagination = {
		page: 1,
		pageSize: PAGE_SIZE,
		total: 0,
		totalPages: 0
	};

	$: feedId = parseInt($page.params.feed_id);

	// Derive current page from URL for refresh persistence
	$: currentPage = parseInt($page.url.searchParams.get('page')) || 1;

	// Track previous feedId to detect feed changes
	let prevFeedId = null;

	// Find current feed from the store
	$: currentFeed = $feedsStore.find(feed => feed.id === feedId);

	onMount(async () => {
		// Auth state is now initialized synchronously from localStorage at module load
		// If we have a token (status === 'unknown'), validate it and load data
		if ($authStore.status === 'unknown' && $authStore.token) {
			try {
				await loadFeedsAndArticles();
				authStore.setStatus('authenticated');
			} catch (error) {
				// Only logout if it's an auth error (401)
				// Global auth guard in +layout.svelte will handle the redirect
				if (error.status === 401) {
					authStore.logout();
					return;
				}
				// For other errors, still mark as authenticated
				authStore.setStatus('authenticated');
			}
		} else if ($authStore.status === 'authenticated') {
			await loadFeedsAndArticles();
		}
		loading = false;
	});

	// Watch for feed or page changes - reset page when feed changes
	$: if (feedId && $authStore.status === 'authenticated' && !loading) {
		// Detect feed change: reset to page 1 and load articles
		if (prevFeedId !== null && prevFeedId !== feedId) {
			// Reset URL to page 1 without triggering another load
			const currentUrl = new URL($page.url);
			currentUrl.searchParams.delete('page');
			goto(currentUrl.pathname + currentUrl.search, { replaceState: true, noScroll: true });
			// Load articles for the new feed
			loadArticles(1);
		} else {
			loadArticles(currentPage);
		}
		prevFeedId = feedId;
	}

	async function loadFeedsAndArticles() {
		try {
			// Load feeds if not already loaded
			if ($feedsStore.length === 0) {
				const feedList = await feeds.list();
				feedsStore.setFeeds(feedList);
			}
			
			// Load articles for this feed with current page from URL
			await loadArticles(currentPage);
		} catch (error) {
			console.error('Failed to load data:', error);
			if (error.status === 404) {
				toast.error('Feed not found');
				goto('/');
			}
		}
	}

	async function loadArticles(page = 1) {
		if (!feedId) return;
		
		try {
			fetchingArticles = true;
			const response = await feeds.getArticles(feedId, { page, pageSize: PAGE_SIZE });
			feedArticles = response.items;
			// Map snake_case API response to camelCase frontend state
			pagination = {
				page: response.pagination.page,
				pageSize: response.pagination.page_size,
				total: response.pagination.total,
				totalPages: response.pagination.total_pages
			};
		} catch (error) {
			console.error('Failed to load articles:', error);
			if (error.status === 404) {
				toast.error('Feed not found');
				goto('/');
			} else {
				toast.error('Failed to load articles');
			}
		} finally {
			fetchingArticles = false;
		}
	}

	function goToPage(targetPage) {
		// Update URL which triggers reactive load via currentPage
		const currentUrl = new URL($page.url);
		if (targetPage === 1) {
			currentUrl.searchParams.delete('page');
		} else {
			currentUrl.searchParams.set('page', targetPage.toString());
		}
		goto(currentUrl.pathname + currentUrl.search, { replaceState: false, noScroll: true });
	}

	function handlePageChange(event) {
		goToPage(event.detail.page);
	}

	async function openArticleReader(event) {
		const article = event.detail?.article;
		if (!article) return;

		readerArticleId = article.id;
		readerOpen = true;
		readerLoading = true;
		readerError = '';
		readerArticle = null;

		try {
			const detail = await articleApi.getById(article.id);
			readerArticle = detail;
		} catch (error) {
			console.error('Failed to load article detail:', error);
			if (error.status === 404) {
				toast.error('Article not found');
			} else if (error.status === 403) {
				toast.error('You are not subscribed to this feed');
			} else if (error.status === 401) {
				toast.error('Please log in again');
				readerOpen = false;
				handleReaderClose();
				return;
			} else {
				toast.error('Failed to open article');
			}
			readerError = error.message ?? 'Failed to load article';
		} finally {
			readerLoading = false;
		}
	}

	function handleReaderClose() {
		readerOpen = false;
		readerLoading = false;
		readerError = '';
		readerArticle = null;
		readerArticleId = null;
	}

	function handleOpenOriginal() {
		if (readerArticle?.url) {
			window.open(readerArticle.url, '_blank', 'noopener,noreferrer');
		}
	}

	async function handleFetchNow() {
		if (fetchingArticles) return;
		
		try {
			await feeds.fetch(feedId);
			toast.success('Feed fetch requested. Articles will update shortly.');
			
			// Auto-refresh articles after a short delay, go to first page to see new articles
			setTimeout(async () => {
				goToPage(1);
			}, 3000);
		} catch (error) {
			toast.error('Failed to fetch feed');
		}
	}

	async function handleUnsubscribe() {
		try {
			await feeds.unsubscribe(feedId);
			feedsStore.removeFeed(feedId);
			toast.success('Successfully unsubscribed from feed');
			showUnsubscribeModal = false;
			goto('/');
		} catch (error) {
			toast.error('Failed to unsubscribe from feed');
		}
	}

	function handleAddSubscription() {
		showAddModal = true;
	}

	function handleModalClose() {
		showAddModal = false;
	}

	function getFeedTitle(feed) {
		if (!feed) return 'Loading...';
		return feed.title || new URL(feed.url).hostname;
	}

	function getFeedDescription(feed) {
		if (!feed) return '';
		return feed.description || `RSS feed from ${new URL(feed.url).hostname}`;
	}
</script>

<svelte:head>
	<title>{getFeedTitle(currentFeed)} - Phoenix RSS</title>
</svelte:head>

{#if $authStore.status === 'authenticated'}
	<div class="main-layout">
		<NavBar />
		
		<div class="content-layout">
			<SideBar on:add-subscription={handleAddSubscription} />
			
			<main class="main-content" class:sidebar-open={$uiStore.sidebarOpen}>
				{#if loading}
					<div class="loading-container">
						<div class="loading-spinner"></div>
						<p>Loading feed...</p>
					</div>
				{:else if currentFeed}
					<div class="feed-page">
						<!-- Feed Header -->
						<header class="feed-header">
							<div class="feed-info">
								<h1 class="feed-title">{getFeedTitle(currentFeed)}</h1>
								<p class="feed-description text-muted">
									{getFeedDescription(currentFeed)}
								</p>
								<div class="feed-meta">
									<span class="article-count">
									{pagination.total} article{pagination.total === 1 ? '' : 's'}
									</span>
									{#if currentFeed.updated_at}
										<span class="feed-separator">•</span>
										<span class="last-updated">
											Updated {new Date(currentFeed.updated_at).toLocaleDateString()}
										</span>
									{/if}
								</div>
							</div>
							
							<div class="feed-actions">
								<button 
									class="button secondary"
									on:click={handleFetchNow}
									disabled={fetchingArticles}
									title="Fetch latest articles"
								>
									{#if fetchingArticles}
										<div class="button-spinner"></div>
									{:else}
										<svg fill="none" viewBox="0 0 24 24" stroke="currentColor">
											<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15"></path>
										</svg>
									{/if}
									Fetch Now
								</button>
								
								<button 
									class="button danger"
									on:click={() => showUnsubscribeModal = true}
									title="Unsubscribe from this feed"
								>
									<svg fill="none" viewBox="0 0 24 24" stroke="currentColor">
										<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16"></path>
									</svg>
									Unsubscribe
								</button>
							</div>
						</header>

						<!-- Articles List -->
						<div class="articles-section">
			{#if fetchingArticles && feedArticles.length === 0}
								<div class="loading-container">
									<div class="loading-spinner"></div>
									<p>Loading articles...</p>
								</div>
			{:else if feedArticles.length === 0}
								<div class="empty-state">
									<div class="empty-icon">
										<svg fill="none" viewBox="0 0 24 24" stroke="currentColor">
											<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 6.253v13m0-13C10.832 5.477 9.246 5 7.5 5S4.168 5.477 3 6.253v13C4.168 18.477 5.754 18 7.5 18s3.332.477 4.5 1.253m0-13C13.168 5.477 14.754 5 16.5 5c1.746 0 3.332.477 4.5 1.253v13C19.832 18.477 18.246 18 16.5 18c-1.746 0-3.332.477-4.5 1.253"></path>
										</svg>
									</div>
									<h3>No Articles Yet</h3>
									<p class="text-muted">
										This feed doesn't have any articles yet. Try clicking "Fetch Now" to load the latest content.
									</p>
									<button class="button primary" on:click={handleFetchNow} disabled={fetchingArticles}>
										{#if fetchingArticles}
											<div class="button-spinner"></div>
										{:else}
											<svg fill="none" viewBox="0 0 24 24" stroke="currentColor">
												<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15"></path>
											</svg>
										{/if}
										Fetch Articles
									</button>
								</div>
							{:else}
				<div class="articles-list">
					{#each feedArticles as article (article.id)}
									<ArticleCard {article} on:open={openArticleReader} />
									{/each}
								</div>

								<!-- Pagination -->
								<Pagination
									page={pagination.page}
									totalPages={pagination.totalPages}
									total={pagination.total}
									pageSize={pagination.pageSize}
									on:pageChange={handlePageChange}
								/>
							{/if}
						</div>
					</div>
				{:else}
					<div class="error-state">
						<h2>Feed Not Found</h2>
						<p class="text-muted">The requested feed could not be found.</p>
						<a href="/" class="button primary">Back to Dashboard</a>
					</div>
				{/if}
			</main>
		</div>
	</div>

	<!-- Modals -->
	<ArticleReaderModal
		open={readerOpen}
		article={readerArticle}
		loading={readerLoading}
		error={readerError}
		on:close={handleReaderClose}
		on:openOriginal={handleOpenOriginal}
	/>

	<AddSubscriptionModal 
		open={showAddModal} 
		on:close={handleModalClose}
	/>

	<Modal 
		open={showUnsubscribeModal} 
		title="Unsubscribe from Feed" 
		showFooter={true}
		on:close={() => showUnsubscribeModal = false}
	>
		<p>Are you sure you want to unsubscribe from <strong>{getFeedTitle(currentFeed)}</strong>?</p>
		<p class="text-muted text-sm">This action cannot be undone. You will need to re-subscribe to access this feed again.</p>
		
		<div slot="footer" class="modal-actions">
			<button 
				class="button secondary" 
				on:click={() => showUnsubscribeModal = false}
			>
				Cancel
			</button>
			<button 
				class="button danger" 
				on:click={handleUnsubscribe}
			>
				Unsubscribe
			</button>
		</div>
	</Modal>
{:else if loading}
	<div class="loading-screen">
		<div class="loading-spinner"></div>
		<p>Loading...</p>
	</div>
{/if}

<style>
	.main-layout {
		height: 100vh;
		display: flex;
		flex-direction: column;
		overflow: hidden;
	}

	.content-layout {
		flex: 1;
		display: flex;
		height: calc(100vh - 56px);
		overflow: hidden;
		position: relative;
	}

	.main-content {
		flex: 1;
		height: 100%;
		overflow-y: auto;
	}

	.main-content.sidebar-open {
		/* No margin needed, sidebar is in flexbox flow */
	}

	.feed-page {
		max-width: 1200px;
		margin: 0;
		padding: var(--space-6);
	}

	.feed-header {
		display: flex;
		align-items: flex-start;
		justify-content: space-between;
		gap: var(--space-4);
		margin-bottom: var(--space-6);
		padding-bottom: var(--space-4);
		border-bottom: 1px solid var(--border);
	}

	.feed-info {
		flex: 1;
		min-width: 0;
	}

	.feed-title {
		margin: 0 0 var(--space-2) 0;
		font-size: 1.875rem;
		font-weight: 700;
		color: var(--text);
		line-height: 1.2;
	}

	.feed-description {
		margin: 0 0 var(--space-3) 0;
		font-size: 1rem;
		line-height: 1.5;
	}

	.feed-meta {
		display: flex;
		align-items: center;
		gap: var(--space-1);
		font-size: 0.875rem;
		color: var(--text-muted);
	}

	.feed-separator {
		color: var(--text-muted);
	}

	.feed-actions {
		display: flex;
		gap: var(--space-2);
		flex-shrink: 0;
	}

	.feed-actions .button {
		gap: var(--space-1);
	}

	.feed-actions .button svg {
		width: 16px;
		height: 16px;
	}

	.button-spinner {
		width: 16px;
		height: 16px;
		border: 2px solid transparent;
		border-top: 2px solid currentColor;
		border-radius: 50%;
		animation: spin 1s linear infinite;
	}

	.articles-section {
		min-height: 400px;
	}

	.articles-list {
		/* Articles will be styled by ArticleCard component */
	}

	.loading-container {
		display: flex;
		flex-direction: column;
		align-items: center;
		justify-content: center;
		padding: var(--space-6);
		gap: var(--space-3);
		color: var(--text-muted);
	}

	.loading-spinner {
		width: 32px;
		height: 32px;
		border: 3px solid var(--border);
		border-top: 3px solid var(--primary);
		border-radius: 50%;
		animation: spin 1s linear infinite;
	}

	.loading-screen {
		display: flex;
		flex-direction: column;
		align-items: center;
		justify-content: center;
		min-height: 100vh;
		gap: var(--space-4);
		color: var(--text-muted);
	}

	.empty-state {
		display: flex;
		flex-direction: column;
		align-items: center;
		justify-content: center;
		padding: var(--space-6);
		text-align: center;
		background: var(--bg-elev);
		border: 1px solid var(--border);
		border-radius: var(--radius-lg);
		margin: var(--space-4) 0;
	}

	.empty-icon {
		width: 48px;
		height: 48px;
		color: var(--text-muted);
		margin-bottom: var(--space-4);
	}

	.empty-icon svg {
		width: 100%;
		height: 100%;
	}

	.empty-state h3 {
		margin: 0 0 var(--space-2) 0;
		color: var(--text);
	}

	.empty-state p {
		margin: 0 0 var(--space-4) 0;
		max-width: 400px;
	}

	.empty-state .button {
		gap: var(--space-1);
	}

	.empty-state .button svg {
		width: 16px;
		height: 16px;
	}

	.error-state {
		display: flex;
		flex-direction: column;
		align-items: center;
		justify-content: center;
		min-height: 60vh;
		text-align: center;
		padding: var(--space-6);
	}

	.error-state h2 {
		margin: 0 0 var(--space-2) 0;
		color: var(--text);
	}

	.error-state p {
		margin: 0 0 var(--space-4) 0;
	}

	.modal-actions {
		display: flex;
		gap: var(--space-2);
		justify-content: flex-end;
	}

	@media (max-width: 768px) {
		.feed-page {
			padding: var(--space-3);
		}

		.feed-header {
			flex-direction: column;
			align-items: stretch;
			gap: var(--space-3);
		}

		.feed-actions {
			justify-content: flex-start;
		}

		.feed-title {
			font-size: 1.5rem;
		}
	}
</style>
