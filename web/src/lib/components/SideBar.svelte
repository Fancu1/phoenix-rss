<script>
	import { createEventDispatcher } from 'svelte';
	import { page } from '$app/stores';
	import { feedsStore, uiStore } from '../stores.js';
	import { feeds as feedsApi } from '../api.js';

	const dispatch = createEventDispatcher();

	$: currentFeedId = $page.params.feed_id ? parseInt($page.params.feed_id) : null;

	let editingFeedId = null;
	let editingTitle = '';

	function getFeedTitle(feed) {
		return feed.custom_title || feed.title || new URL(feed.url).hostname;
	}

	function getFeedDescription(feed) {
		try {
			return new URL(feed.url).hostname;
		} catch {
			return feed.url;
		}
	}

	function handleFeedClick(feedId) {
		if (window.innerWidth <= 768) {
			uiStore.setSidebar(false);
		}
	}

	function handleAddSubscription() {
		dispatch('add-subscription');
	}

	function startEditing(event, feed) {
		event.preventDefault();
		event.stopPropagation();
		editingFeedId = feed.id;
		editingTitle = feed.custom_title || feed.title || '';
	}

	async function saveTitle(feed) {
		if (editingFeedId !== feed.id) return;
		
		try {
			// If title is empty or same as original, clear custom_title
			const newCustomTitle = editingTitle.trim() === '' || editingTitle.trim() === feed.title 
				? null 
				: editingTitle.trim();
			
			await feedsApi.update(feed.id, { custom_title: newCustomTitle });
			feedsStore.update(feeds => 
				feeds.map(f => f.id === feed.id ? { ...f, custom_title: newCustomTitle } : f)
			);
		} catch (error) {
			console.error('Failed to update feed title:', error);
			uiStore.addToast('error', 'Failed to update feed title');
		} finally {
			editingFeedId = null;
			editingTitle = '';
		}
	}

	function cancelEditing() {
		editingFeedId = null;
		editingTitle = '';
	}

	function handleKeydown(event, feed) {
		if (event.key === 'Enter') {
			saveTitle(feed);
		} else if (event.key === 'Escape') {
			cancelEditing();
		}
	}
</script>

<aside class="sidebar" class:open={$uiStore.sidebarOpen}>
	<div class="sidebar-header">
		<h3>Subscriptions</h3>
		<button 
			class="add-feed-button" 
			on:click={handleAddSubscription}
			title="Add Subscription"
		>
			<svg fill="none" viewBox="0 0 24 24" stroke="currentColor">
				<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 6v6m0 0v6m0-6h6m-6 0H6"></path>
			</svg>
		</button>
	</div>

	<div class="sidebar-content">
		{#if $feedsStore.length === 0}
			<div class="empty-state">
				<div class="empty-icon">
					<svg fill="none" viewBox="0 0 24 24" stroke="currentColor">
						<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 5c7.18 0 13 5.82 13 13M6 11a7 7 0 017 7m-6 0a1 1 0 11-2 0 1 1 0 012 0z"></path>
					</svg>
				</div>
				<p class="empty-text">No feeds yet</p>
				<p class="empty-subtext">Click the + button to add your first RSS feed</p>
			</div>
		{:else}
			<div class="feed-list">
				{#each $feedsStore as feed (feed.id)}
					<a 
						href="/feeds/{feed.id}" 
						class="feed-item" 
						class:active={currentFeedId === feed.id}
						on:click={() => handleFeedClick(feed.id)}
					>
						<div class="feed-info">
							{#if editingFeedId === feed.id}
								<!-- svelte-ignore a11y-autofocus -->
								<input
									type="text"
									class="edit-title-input"
									bind:value={editingTitle}
									on:blur={() => saveTitle(feed)}
									on:keydown={(e) => handleKeydown(e, feed)}
									on:click|stopPropagation
									autofocus
								/>
							{:else}
								<div class="feed-title">{getFeedTitle(feed)}</div>
							{/if}
							<div class="feed-description">{getFeedDescription(feed)}</div>
						</div>

						{#if editingFeedId !== feed.id}
							<button 
								class="edit-button"
								on:click={(e) => startEditing(e, feed)}
								title="Edit feed name"
							>
								<svg fill="none" viewBox="0 0 24 24" stroke="currentColor" width="14" height="14">
									<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15.232 5.232l3.536 3.536m-2.036-5.036a2.5 2.5 0 113.536 3.536L6.5 21.036H3v-3.572L16.732 3.732z" />
								</svg>
							</button>
						{/if}
					</a>
				{/each}
			</div>
		{/if}
	</div>
</aside>

<!-- Mobile overlay -->
{#if $uiStore.sidebarOpen}
	<!-- svelte-ignore a11y-no-static-element-interactions -->
	<div 
		class="sidebar-overlay"
		on:click={() => uiStore.setSidebar(false)}
		on:keydown={(e) => e.key === 'Escape' && uiStore.setSidebar(false)}
	></div>
{/if}

<style>
	.sidebar {
		position: fixed;
		top: 56px;
		left: 0;
		bottom: 0;
		width: 280px;
		background: var(--bg-elev);
		border-right: 1px solid var(--border);
		z-index: 200;
		transform: translateX(-100%);
		transition: transform 0.3s ease;
		display: flex;
		flex-direction: column;
	}

	.sidebar.open {
		transform: translateX(0);
	}

	.sidebar-header {
		display: flex;
		align-items: center;
		justify-content: space-between;
		padding: var(--space-4);
		border-bottom: 1px solid var(--border);
	}

	.sidebar-header h3 {
		margin: 0;
		font-size: 1rem;
		font-weight: 600;
		color: var(--text);
	}

	.add-feed-button {
		background: none;
		border: none;
		color: var(--text-muted);
		cursor: pointer;
		padding: var(--space-1);
		border-radius: var(--radius-sm);
		transition: color 0.2s ease, background-color 0.2s ease;
		display: flex;
		align-items: center;
		justify-content: center;
	}

	.add-feed-button:hover {
		color: var(--primary);
		background: var(--bg);
	}

	.add-feed-button svg {
		width: 20px;
		height: 20px;
	}

	.feed-count {
		background: var(--bg);
		color: var(--text-muted);
		padding: var(--space-1) var(--space-2);
		border-radius: var(--radius-sm);
		font-size: 0.75rem;
		font-weight: 500;
	}

	.sidebar-content {
		flex: 1;
		overflow-y: auto;
	}

	.empty-state {
		display: flex;
		flex-direction: column;
		align-items: center;
		justify-content: center;
		padding: var(--space-6);
		text-align: center;
		min-height: 200px;
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

	.empty-text {
		margin: 0 0 var(--space-1) 0;
		font-weight: 500;
		color: var(--text);
	}

	.empty-subtext {
		margin: 0;
		font-size: 0.875rem;
		color: var(--text-muted);
		line-height: 1.4;
	}

	.feed-list {
		padding: var(--space-2) 0;
	}

	.feed-item {
		display: flex;
		align-items: center;
		justify-content: space-between;
		padding: var(--space-3) var(--space-4);
		text-decoration: none;
		color: var(--text);
		transition: background-color 0.2s ease;
		position: relative;
	}

	.feed-item:hover {
		background: var(--bg);
	}

	.feed-item.active {
		background: var(--bg);
		border-right: 3px solid var(--primary);
	}

	.feed-item.active::before {
		content: '';
		position: absolute;
		left: 0;
		top: 0;
		bottom: 0;
		width: 3px;
		background: var(--primary);
	}

	.feed-info {
		flex: 1;
		min-width: 0;
	}

	.feed-title {
		font-weight: 500;
		font-size: 0.875rem;
		margin-bottom: var(--space-1);
		white-space: nowrap;
		overflow: hidden;
		text-overflow: ellipsis;
		line-height: 1.2;
	}

	.feed-description {
		font-size: 0.75rem;
		color: var(--text-muted);
		white-space: nowrap;
		overflow: hidden;
		text-overflow: ellipsis;
	}

	.unread-badge {
		background: var(--primary);
		color: var(--primary-contrast);
		padding: 2px var(--space-1);
		border-radius: 10px;
		font-size: 0.625rem;
		font-weight: 600;
		min-width: 18px;
		text-align: center;
		margin-left: var(--space-2);
	}

	.edit-button {
		background: none;
		border: none;
		color: var(--text-muted);
		cursor: pointer;
		padding: var(--space-1);
		border-radius: var(--radius-sm);
		opacity: 0;
		transition: opacity 0.2s ease, color 0.2s ease;
		display: flex;
		align-items: center;
		justify-content: center;
		flex-shrink: 0;
	}

	.feed-item:hover .edit-button {
		opacity: 1;
	}

	.edit-button:hover {
		color: var(--primary);
	}

	.edit-title-input {
		width: 100%;
		padding: var(--space-1) var(--space-2);
		font-size: 0.875rem;
		font-weight: 500;
		border: 1px solid var(--primary);
		border-radius: var(--radius-sm);
		background: var(--bg);
		color: var(--text);
		outline: none;
	}

	.edit-title-input:focus {
		box-shadow: 0 0 0 2px var(--primary-alpha);
	}

	.sidebar-overlay {
		position: fixed;
		top: 56px;
		left: 0;
		right: 0;
		bottom: 0;
		background: rgba(0, 0, 0, 0.3);
		z-index: 150;
		display: none;
	}

	@media (max-width: 768px) {
		.sidebar {
			width: 280px;
		}

		.sidebar-overlay {
			display: block;
		}
	}

	@media (min-width: 769px) {
		.sidebar {
			position: relative;
			top: 0;
			transform: none;
		}

		.sidebar.open {
			transform: none;
		}
	}
</style>
