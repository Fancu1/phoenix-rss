<script>
	import { createEventDispatcher } from 'svelte';
	import { authStore, uiStore } from '../stores.js';

	const dispatch = createEventDispatcher();

	let showUserMenu = false;

	function toggleSidebar() {
		uiStore.toggleSidebar();
	}

	function toggleTheme() {
		uiStore.toggleTheme();
	}

	function handleLogout() {
		authStore.logout();
		showUserMenu = false;
	}

	function toggleUserMenu() {
		showUserMenu = !showUserMenu;
	}

	// Close user menu when clicking outside
	function handleClickOutside(event) {
		if (!event.target.closest('.user-menu-container')) {
			showUserMenu = false;
		}
	}
</script>

<svelte:window on:click={handleClickOutside} />

<nav class="navbar">
	<div class="navbar-left">
		<button 
			class="navbar-button sidebar-toggle"
			on:click={toggleSidebar}
			aria-label="Toggle sidebar"
		>
			<svg fill="none" viewBox="0 0 24 24" stroke="currentColor">
				<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 6h16M4 12h16M4 18h16"></path>
			</svg>
		</button>

		<div class="navbar-brand">
			<h2>Phoenix RSS</h2>
		</div>
	</div>

	<div class="navbar-center">
		<div class="search-container">
			<svg class="search-icon" fill="none" viewBox="0 0 24 24" stroke="currentColor">
				<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z"></path>
			</svg>
			<input 
				type="text" 
				class="search-input" 
				placeholder="Search articles..."
				disabled
			/>
		</div>
	</div>

	<div class="navbar-right">
		<button 
			class="navbar-button theme-toggle"
			on:click={toggleTheme}
			title="Toggle theme"
		>
			{#if $uiStore.theme === 'dark'}
				<svg fill="none" viewBox="0 0 24 24" stroke="currentColor">
					<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 3v1m0 16v1m9-9h-1M4 12H3m15.364 6.364l-.707-.707M6.343 6.343l-.707-.707m12.728 0l-.707.707M6.343 17.657l-.707.707M16 12a4 4 0 11-8 0 4 4 0 018 0z"></path>
				</svg>
			{:else}
				<svg fill="none" viewBox="0 0 24 24" stroke="currentColor">
					<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M20.354 15.354A9 9 0 018.646 3.646 9.003 9.003 0 0012 21a9.003 9.003 0 008.354-5.646z"></path>
				</svg>
			{/if}
		</button>

		<div class="user-menu-container">
			<button 
				class="navbar-button user-avatar"
				on:click={toggleUserMenu}
				aria-label="User menu"
			>
				<div class="avatar">
					{$authStore.user?.username?.charAt(0).toUpperCase() || 'U'}
				</div>
			</button>

			{#if showUserMenu}
				<div class="user-menu">
					<div class="user-info">
						<div class="username">{$authStore.user?.username || 'User'}</div>
					</div>
					<div class="menu-divider"></div>
					<a href="/settings" class="menu-item">
						<svg fill="none" viewBox="0 0 24 24" stroke="currentColor">
							<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M10.325 4.317c.426-1.756 2.924-1.756 3.35 0a1.724 1.724 0 002.573 1.066c1.543-.94 3.31.826 2.37 2.37a1.724 1.724 0 001.065 2.572c1.756.426 1.756 2.924 0 3.35a1.724 1.724 0 00-1.066 2.573c.94 1.543-.826 3.31-2.37 2.37a1.724 1.724 0 00-2.572 1.065c-.426 1.756-2.924 1.756-3.35 0a1.724 1.724 0 00-2.573-1.066c-1.543.94-3.31-.826-2.37-2.37a1.724 1.724 0 00-1.065-2.572c-1.756-.426-1.756-2.924 0-3.35a1.724 1.724 0 001.066-2.573c-.94-1.543.826-3.31 2.37-2.37.996.608 2.296.07 2.572-1.065z"></path>
							<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 12a3 3 0 11-6 0 3 3 0 016 0z"></path>
						</svg>
						Settings
					</a>
					<a href="/about" class="menu-item">
						<svg fill="none" viewBox="0 0 24 24" stroke="currentColor">
							<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M13 16h-1v-4h-1m1-4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z"></path>
						</svg>
						About
					</a>
					<div class="menu-divider"></div>
					<button class="menu-item logout" on:click={handleLogout}>
						<svg fill="none" viewBox="0 0 24 24" stroke="currentColor">
							<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M17 16l4-4m0 0l-4-4m4 4H7m6 4v1a3 3 0 01-3 3H6a3 3 0 01-3-3V7a3 3 0 013-3h4a3 3 0 013 3v1"></path>
						</svg>
						Sign Out
					</button>
				</div>
			{/if}
		</div>
	</div>
</nav>

<style>
	.navbar {
		display: flex;
		align-items: center;
		justify-content: space-between;
		height: 56px;
		padding: 0 var(--space-4);
		background: var(--bg-elev);
		border-bottom: 1px solid var(--border);
		position: sticky;
		top: 0;
		z-index: 100;
		backdrop-filter: blur(8px);
	}

	.navbar-left {
		display: flex;
		align-items: center;
		gap: var(--space-3);
	}

	.navbar-center {
		flex: 1;
		max-width: 360px;
		margin: 0 var(--space-4);
	}

	.navbar-right {
		display: flex;
		align-items: center;
		gap: var(--space-2);
	}

	.navbar-button {
		background: none;
		border: none;
		color: var(--text-muted);
		cursor: pointer;
		padding: var(--space-2);
		border-radius: var(--radius-sm);
		transition: color 0.2s ease, background-color 0.2s ease;
		display: flex;
		align-items: center;
		gap: var(--space-1);
	}

	.navbar-button:hover {
		color: var(--text);
		background: var(--bg);
	}

	.navbar-button svg {
		width: 20px;
		height: 20px;
	}

	.sidebar-toggle {
		display: none;
	}

	.navbar-brand h2 {
		margin: 0;
		font-size: 1.25rem;
		font-weight: 600;
		color: var(--text);
	}

	.search-container {
		position: relative;
		width: 100%;
	}

	.search-icon {
		position: absolute;
		left: var(--space-3);
		top: 50%;
		transform: translateY(-50%);
		width: 16px;
		height: 16px;
		color: var(--text-muted);
		pointer-events: none;
	}

	.search-input {
		width: 100%;
		padding: var(--space-2) var(--space-3) var(--space-2) 36px;
		background: var(--bg);
		border: 1px solid var(--border);
		border-radius: var(--radius-sm);
		color: var(--text);
		font-size: 0.875rem;
		transition: border-color 0.2s ease;
	}

	.search-input:focus {
		outline: none;
		border-color: var(--primary);
	}

	.search-input:disabled {
		opacity: 0.6;
		cursor: not-allowed;
	}

	.user-menu-container {
		position: relative;
	}

	.avatar {
		width: 32px;
		height: 32px;
		border-radius: 50%;
		background: var(--primary);
		color: var(--primary-contrast);
		display: flex;
		align-items: center;
		justify-content: center;
		font-weight: 600;
		font-size: 0.875rem;
	}

	.user-menu {
		position: absolute;
		top: calc(100% + var(--space-1));
		right: 0;
		min-width: 200px;
		background: var(--bg-elev);
		border: 1px solid var(--border);
		border-radius: var(--radius-md);
		box-shadow: var(--shadow-md);
		padding: var(--space-2);
		z-index: 1000;
	}

	.user-info {
		padding: var(--space-2) var(--space-3);
	}

	.username {
		font-weight: 500;
		color: var(--text);
	}

	.menu-divider {
		height: 1px;
		background: var(--border);
		margin: var(--space-2) 0;
	}

	.menu-item {
		display: flex;
		align-items: center;
		gap: var(--space-2);
		width: 100%;
		padding: var(--space-2) var(--space-3);
		background: none;
		border: none;
		color: var(--text);
		text-decoration: none;
		font-size: 0.875rem;
		border-radius: var(--radius-sm);
		cursor: pointer;
		transition: background-color 0.2s ease;
		text-align: left;
		font-family: inherit;
	}

	.menu-item:hover {
		background: var(--bg);
	}

	.menu-item svg {
		width: 16px;
		height: 16px;
		color: var(--text-muted);
	}

	.menu-item.logout {
		color: var(--danger);
	}

	.menu-item.logout svg {
		color: var(--danger);
	}

	@media (max-width: 768px) {
		.sidebar-toggle {
			display: flex;
		}

		.navbar-center {
			display: none;
		}

		.button-text {
			display: none;
		}

		.navbar {
			padding: 0 var(--space-2);
		}
	}
</style>
