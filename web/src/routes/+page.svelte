<script>
	import { onMount } from 'svelte';
	import { goto } from '$app/navigation';
	import { authStore, feedsStore, uiStore } from '$lib/stores.js';
	import { feeds } from '$lib/api.js';
	import NavBar from '$lib/components/NavBar.svelte';
	import SideBar from '$lib/components/SideBar.svelte';
	import AddSubscriptionModal from '$lib/components/AddSubscriptionModal.svelte';

	let showAddModal = false;
	let loading = true;

	// Redirect to login if not authenticated
	$: if ($authStore.status === 'anonymous') {
		goto('/login');
	}

	onMount(async () => {
		// Auth state is now initialized synchronously from localStorage at module load
		// If we have a token (status === 'unknown'), validate it with an API call
		if ($authStore.status === 'unknown' && $authStore.token) {
			try {
				await loadFeeds();
				authStore.setStatus('authenticated');
			} catch (error) {
				// Only logout if it's an auth error (401)
				if (error.status === 401) {
					authStore.logout();
					goto('/login');
					return;
				}
				// For other errors, still mark as authenticated (token might be valid)
				authStore.setStatus('authenticated');
			}
		} else if ($authStore.status === 'authenticated') {
			await loadFeeds();
		}
		loading = false;
	});

	async function loadFeeds() {
		try {
			const feedList = await feeds.list();
			feedsStore.setFeeds(feedList);
		} catch (error) {
			console.error('Failed to load feeds:', error);
			// If it's an auth error, it will be handled by the API layer
		}
	}

	function handleAddSubscription() {
		showAddModal = true;
	}

	function handleModalClose() {
		showAddModal = false;
	}
</script>

<svelte:head>
	<title>Phoenix RSS</title>
</svelte:head>

{#if $authStore.status === 'authenticated'}
	<div class="main-layout">
		<NavBar />
		
		<div class="content-layout">
			<SideBar on:add-subscription={handleAddSubscription} />
			
			<main class="main-content" class:sidebar-open={$uiStore.sidebarOpen}>
				<div class="dashboard">
					<div class="dashboard-header">
						<h1>Welcome to Phoenix RSS</h1>
						<p class="text-muted">
							{#if $feedsStore.length === 0}
								Get started by adding your first RSS feed subscription.
							{:else}
								You have {$feedsStore.length} feed{$feedsStore.length === 1 ? '' : 's'} subscribed.
							{/if}
						</p>
					</div>

					{#if $feedsStore.length === 0}
						<div class="welcome-card">
							<div class="welcome-icon">
								<svg fill="none" viewBox="0 0 24 24" stroke="currentColor">
									<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 5c7.18 0 13 5.82 13 13M6 11a7 7 0 017 7m-6 0a1 1 0 11-2 0 1 1 0 012 0z"></path>
								</svg>
							</div>
							<h3>Start Building Your Feed Library</h3>
							<p class="text-muted">
								Add RSS feeds from your favorite websites, blogs, and news sources. 
								Phoenix RSS will automatically fetch and organize articles for you.
							</p>
							<button 
								class="button primary"
								on:click={handleAddSubscription}
							>
								<svg fill="none" viewBox="0 0 24 24" stroke="currentColor">
									<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 6v6m0 0v6m0-6h6m-6 0H6"></path>
								</svg>
								Add Your First Feed
							</button>
						</div>
					{:else}
						<div class="recent-activity">
							<h2>Recent Activity</h2>
							<p class="text-muted">
								Select a feed from the sidebar to view articles, or add a new subscription to expand your library.
							</p>
						</div>
					{/if}
				</div>
			</main>
		</div>
	</div>

	<!-- Add Subscription Modal -->
	<AddSubscriptionModal 
		open={showAddModal} 
		on:close={handleModalClose}
	/>
{:else if loading}
	<div class="loading-screen">
		<div class="loading-spinner"></div>
		<p>Loading...</p>
	</div>
{/if}

<style>
	.main-layout {
		min-height: 100vh;
		display: flex;
		flex-direction: column;
	}

	.content-layout {
		flex: 1;
		display: flex;
		overflow: hidden;
	}

	.main-content {
		flex: 1;
		overflow-y: auto;
		margin-left: 0;
		transition: margin-left 0.3s ease;
	}

	.main-content.sidebar-open {
		margin-left: 280px;
	}

	.dashboard {
		max-width: 1200px;
		margin: 0;
		padding: var(--space-6) var(--space-6);
	}

	.dashboard-header {
		text-align: left;
		margin-bottom: var(--space-6);
	}

	.dashboard-header h1 {
		margin-bottom: var(--space-2);
		color: var(--text);
	}

	.welcome-card {
		background: var(--bg-elev);
		border: 1px solid var(--border);
		border-radius: var(--radius-lg);
		padding: var(--space-6);
		text-align: center;
		box-shadow: var(--shadow-sm);
		max-width: 600px;
	}

	.welcome-icon {
		width: 64px;
		height: 64px;
		margin: 0 auto var(--space-4);
		color: var(--primary);
	}

	.welcome-icon svg {
		width: 100%;
		height: 100%;
	}

	.welcome-card h3 {
		margin-bottom: var(--space-3);
		color: var(--text);
	}

	.welcome-card p {
		margin-bottom: var(--space-4);
		line-height: 1.6;
		max-width: 400px;
		margin-left: auto;
		margin-right: auto;
	}

	.welcome-card .button {
		gap: var(--space-2);
	}

	.welcome-card .button svg {
		width: 20px;
		height: 20px;
	}

	.recent-activity {
		background: var(--bg-elev);
		border: 1px solid var(--border);
		border-radius: var(--radius-lg);
		padding: var(--space-5);
		text-align: left;
	}

	.recent-activity h2 {
		margin-bottom: var(--space-2);
		color: var(--text);
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

	.loading-spinner {
		width: 32px;
		height: 32px;
		border: 3px solid var(--border);
		border-top: 3px solid var(--primary);
		border-radius: 50%;
		animation: spin 1s linear infinite;
	}

	@media (max-width: 768px) {
		.main-content {
			margin-left: 0;
		}

		.main-content.sidebar-open {
			margin-left: 0;
		}

		.dashboard {
			padding: var(--space-4) var(--space-3);
		}

		.welcome-card {
			padding: var(--space-4);
		}
	}
</style>
