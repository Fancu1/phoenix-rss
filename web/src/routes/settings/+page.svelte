<script>
	import { goto } from '$app/navigation';
	import { authStore, uiStore, toast } from '$lib/stores.js';
	import { feeds } from '$lib/api.js';
	import NavBar from '$lib/components/NavBar.svelte';
	import SideBar from '$lib/components/SideBar.svelte';
	import AddSubscriptionModal from '$lib/components/AddSubscriptionModal.svelte';
	import ImportModal from '$lib/components/ImportModal.svelte';

	let showAddModal = false;
	let showImportModal = false;
	let isExporting = false;

	// Redirect to login if not authenticated
	$: if ($authStore.status === 'anonymous') {
		goto('/login');
	}

	function handleAddSubscription() {
		showAddModal = true;
	}

	function handleModalClose() {
		showAddModal = false;
	}

	function handleImportModalClose() {
		showImportModal = false;
	}

	function handleThemeChange() {
		uiStore.toggleTheme();
	}

	async function handleExport() {
		if (isExporting) return;
		
		isExporting = true;
		try {
			await feeds.exportOPML();
			toast.success('Subscriptions exported successfully');
		} catch (err) {
			toast.error(err.message || 'Failed to export subscriptions');
		} finally {
			isExporting = false;
		}
	}

	function handleImport() {
		showImportModal = true;
	}
</script>

<svelte:head>
	<title>Settings - Phoenix RSS</title>
</svelte:head>

{#if $authStore.status === 'authenticated'}
	<div class="main-layout">
		<NavBar />
		
		<div class="content-layout">
			<SideBar on:add-subscription={handleAddSubscription} />
			
			<main class="main-content" class:sidebar-open={$uiStore.sidebarOpen}>
				<div class="settings-page">
					<header class="settings-header">
						<h1>Settings</h1>
						<p class="text-muted">Customize your RSS reading experience</p>
					</header>

					<div class="settings-sections">
						<!-- Appearance Section -->
						<section class="settings-section">
							<div class="section-header">
								<h2>Appearance</h2>
								<p class="text-muted">Customize the look and feel of Phoenix RSS</p>
							</div>
							
							<div class="setting-item">
								<div class="setting-info">
									<label for="theme-toggle" class="setting-label">Theme</label>
									<p class="setting-description">Choose between light and dark mode</p>
								</div>
								<div class="setting-control">
									<button 
										id="theme-toggle"
										class="theme-toggle-button"
										class:dark={$uiStore.theme === 'dark'}
										on:click={handleThemeChange}
									>
										<div class="toggle-slider">
											{#if $uiStore.theme === 'dark'}
												<svg fill="none" viewBox="0 0 24 24" stroke="currentColor">
													<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M20.354 15.354A9 9 0 018.646 3.646 9.003 9.003 0 0012 21a9.003 9.003 0 008.354-5.646z"></path>
												</svg>
											{:else}
												<svg fill="none" viewBox="0 0 24 24" stroke="currentColor">
													<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 3v1m0 16v1m9-9h-1M4 12H3m15.364 6.364l-.707-.707M6.343 6.343l-.707-.707m12.728 0l-.707.707M6.343 17.657l-.707.707M16 12a4 4 0 11-8 0 4 4 0 018 0z"></path>
												</svg>
											{/if}
										</div>
										<span class="theme-label">
											{$uiStore.theme === 'dark' ? 'Dark' : 'Light'}
										</span>
									</button>
								</div>
							</div>
						</section>

						<!-- Account Section -->
						<section class="settings-section">
							<div class="section-header">
								<h2>Account</h2>
								<p class="text-muted">Manage your account settings</p>
							</div>
							
							<div class="setting-item">
								<div class="setting-info">
									<span class="setting-label">Username</span>
									<p class="setting-description">Your account username</p>
								</div>
								<div class="setting-control">
									<span class="username-display">{$authStore.user?.username || 'Unknown'}</span>
								</div>
							</div>

							<div class="setting-item">
								<div class="setting-info">
									<span class="setting-label">Sign Out</span>
									<p class="setting-description">Sign out from your account</p>
								</div>
								<div class="setting-control">
									<button 
										class="button danger"
										on:click={() => authStore.logout()}
									>
										<svg fill="none" viewBox="0 0 24 24" stroke="currentColor">
											<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M17 16l4-4m0 0l-4-4m4 4H7m6 4v1a3 3 0 01-3 3H6a3 3 0 01-3-3V7a3 3 0 013-3h4a3 3 0 013 3v1"></path>
										</svg>
										Sign Out
									</button>
								</div>
							</div>
						</section>

						<!-- Data Management Section -->
						<section class="settings-section">
							<div class="section-header">
								<h2>Data Management</h2>
								<p class="text-muted">Import or export your RSS subscriptions</p>
							</div>
							
							<div class="setting-item">
								<div class="setting-info">
									<span class="setting-label">Import Subscriptions</span>
									<p class="setting-description">Import feeds from an OPML file exported from another RSS reader</p>
								</div>
								<div class="setting-control">
									<button 
										class="button secondary"
										on:click={handleImport}
									>
										<svg fill="none" viewBox="0 0 24 24" stroke="currentColor">
											<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 16v1a3 3 0 003 3h10a3 3 0 003-3v-1m-4-8l-4-4m0 0L8 8m4-4v12"></path>
										</svg>
										Import OPML
									</button>
								</div>
							</div>

							<div class="setting-item">
								<div class="setting-info">
									<span class="setting-label">Export Subscriptions</span>
									<p class="setting-description">Download all your subscriptions as an OPML file</p>
								</div>
								<div class="setting-control">
									<button 
										class="button secondary"
										class:loading={isExporting}
										on:click={handleExport}
										disabled={isExporting}
									>
										{#if isExporting}
											<span class="spinner"></span>
										{:else}
											<svg fill="none" viewBox="0 0 24 24" stroke="currentColor">
												<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 16v1a3 3 0 003 3h10a3 3 0 003-3v-1m-4-4l-4 4m0 0l-4-4m4 4V4"></path>
											</svg>
										{/if}
										Export OPML
									</button>
								</div>
							</div>
						</section>

						<!-- About Section -->
						<section class="settings-section">
							<div class="section-header">
								<h2>About Phoenix RSS</h2>
								<p class="text-muted">Information about this application</p>
							</div>
							
							<div class="setting-item">
								<div class="setting-info">
									<span class="setting-label">Version</span>
									<p class="setting-description">Current application version</p>
								</div>
								<div class="setting-control">
									<span class="version-display">1.0.0</span>
								</div>
							</div>

							<div class="setting-item">
								<div class="setting-info">
									<span class="setting-label">Learn More</span>
									<p class="setting-description">Get more information about Phoenix RSS</p>
								</div>
								<div class="setting-control">
									<a href="/about" class="button secondary">
										<svg fill="none" viewBox="0 0 24 24" stroke="currentColor">
											<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M13 16h-1v-4h-1m1-4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z"></path>
										</svg>
										About Page
									</a>
								</div>
							</div>
						</section>
					</div>
				</div>
			</main>
		</div>
	</div>

	<!-- Add Subscription Modal -->
	<AddSubscriptionModal 
		open={showAddModal} 
		on:close={handleModalClose}
	/>

	<!-- Import Modal -->
	<ImportModal 
		open={showImportModal} 
		on:close={handleImportModalClose}
	/>
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

	.settings-page {
		max-width: 1200px;
		margin: 0;
		padding: var(--space-6) var(--space-6);
	}

	.settings-header {
		margin-bottom: var(--space-6);
	}

	.settings-header h1 {
		margin: 0 0 var(--space-2) 0;
		color: var(--text);
	}

	.settings-sections {
		display: flex;
		flex-direction: column;
		gap: var(--space-6);
		max-width: 800px;
	}

	.settings-section {
		background: var(--bg-elev);
		border: 1px solid var(--border);
		border-radius: var(--radius-lg);
		overflow: hidden;
	}

	.section-header {
		padding: var(--space-5);
		border-bottom: 1px solid var(--border);
		background: var(--bg);
	}

	.section-header h2 {
		margin: 0 0 var(--space-1) 0;
		font-size: 1.25rem;
		color: var(--text);
	}

	.setting-item {
		display: flex;
		align-items: center;
		justify-content: space-between;
		padding: var(--space-5);
		border-bottom: 1px solid var(--border);
	}

	.setting-item:last-child {
		border-bottom: none;
	}

	.setting-info {
		flex: 1;
		min-width: 0;
		margin-right: var(--space-4);
	}

	.setting-label {
		display: block;
		font-weight: 600;
		margin-bottom: var(--space-1);
		color: var(--text);
	}

	.setting-description {
		margin: 0;
		font-size: 0.875rem;
		color: var(--text-muted);
		line-height: 1.4;
	}

	.setting-control {
		flex-shrink: 0;
	}

	.theme-toggle-button {
		display: flex;
		align-items: center;
		gap: var(--space-2);
		background: var(--bg);
		border: 1px solid var(--border);
		border-radius: var(--radius-md);
		padding: var(--space-2) var(--space-3);
		cursor: pointer;
		transition: all 0.2s ease;
		color: var(--text);
		font-family: inherit;
	}

	.theme-toggle-button:hover {
		border-color: var(--primary);
	}

	.theme-toggle-button.dark {
		background: var(--primary);
		color: var(--primary-contrast);
		border-color: var(--primary);
	}

	.toggle-slider {
		width: 24px;
		height: 24px;
		display: flex;
		align-items: center;
		justify-content: center;
	}

	.toggle-slider svg {
		width: 16px;
		height: 16px;
	}

	.theme-label {
		font-weight: 500;
		font-size: 0.875rem;
	}

	.toggle-button {
		display: flex;
		align-items: center;
		gap: var(--space-2);
		background: none;
		border: none;
		cursor: pointer;
		color: var(--text);
		font-family: inherit;
		font-size: 0.875rem;
	}

	.toggle-track {
		width: 44px;
		height: 24px;
		background: var(--border);
		border-radius: 12px;
		position: relative;
		transition: background-color 0.2s ease;
	}

	.toggle-button.active .toggle-track {
		background: var(--primary);
	}

	.toggle-thumb {
		width: 20px;
		height: 20px;
		background: var(--primary-contrast);
		border-radius: 50%;
		position: absolute;
		top: 2px;
		left: 2px;
		transition: transform 0.2s ease;
		box-shadow: var(--shadow-sm);
	}

	.toggle-button.active .toggle-thumb {
		transform: translateX(20px);
	}

	.username-display,
	.version-display {
		font-weight: 500;
		color: var(--text);
		background: var(--bg);
		padding: var(--space-2) var(--space-3);
		border-radius: var(--radius-sm);
		border: 1px solid var(--border);
	}

	.button {
		gap: var(--space-1);
	}

	.button svg {
		width: 16px;
		height: 16px;
	}

	.button.loading {
		opacity: 0.7;
		cursor: not-allowed;
	}

	.spinner {
		width: 16px;
		height: 16px;
		border: 2px solid var(--border);
		border-top-color: var(--primary);
		border-radius: 50%;
		animation: spin 0.8s linear infinite;
	}

	@keyframes spin {
		to {
			transform: rotate(360deg);
		}
	}

	@media (max-width: 768px) {
		.main-content {
			margin-left: 0;
		}

		.main-content.sidebar-open {
			margin-left: 0;
		}

		.settings-page {
			padding: var(--space-4) var(--space-3);
		}

		.setting-item {
			flex-direction: column;
			align-items: stretch;
			gap: var(--space-3);
		}

		.setting-info {
			margin-right: 0;
		}

		.setting-control {
			align-self: flex-start;
		}
	}
</style>
