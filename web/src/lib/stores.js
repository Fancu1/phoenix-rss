// @ts-check
import { browser } from '$app/environment';
import { writable } from 'svelte/store';

// Initialize auth state from localStorage synchronously at module load time
function getInitialAuthState() {
	if (browser) {
		const token = localStorage.getItem('token');
		if (token) {
			return {
				token,
				user: null,
				status: 'unknown' // Will be validated by first API call
			};
		}
	}
	return {
		token: null,
		user: null,
		status: browser ? 'anonymous' : 'unknown' // Only anonymous if we're sure (browser loaded, no token)
	};
}

// Auth Store - manages authentication state
function createAuthStore() {
	const { subscribe, set, update } = writable(getInitialAuthState());

	return {
		subscribe,
		// Login with token and user data
		login: (token, user) => {
			if (browser) {
				localStorage.setItem('token', token);
			}
			set({
				token,
				user,
				status: 'authenticated'
			});
		},
		// Logout and clear data
		logout: () => {
			if (browser) {
				localStorage.removeItem('token');
			}
			set({
				token: null,
				user: null,
				status: 'anonymous'
			});
		},
		// Initialize from localStorage (kept for backwards compatibility, but now a no-op)
		init: () => {
			// Auth state is now initialized synchronously at module load time
			// This function is kept for backwards compatibility
		},
		// Set status (used after API validation)
		setStatus: (status) => {
			update(state => ({
				...state,
				status
			}));
		}
	};
}

// Feeds Store - manages RSS feed subscriptions
function createFeedsStore() {
	const { subscribe, set, update } = writable([]);

	return {
		subscribe,
		// Set feeds list
		setFeeds: (feeds) => set(feeds),
		// Add a new feed
		addFeed: (feed) => {
			update(feeds => [...feeds, feed]);
		},
		// Remove a feed by id
		removeFeed: (feedId) => {
			update(feeds => feeds.filter(feed => feed.id !== feedId));
		},
		// Update a feed
		updateFeed: (feedId, updates) => {
			update(feeds => feeds.map(feed => 
				feed.id === feedId ? { ...feed, ...updates } : feed
			));
		},
		// Clear all feeds
		clear: () => set([])
	};
}

// UI Store - manages UI state (theme, sidebar, toasts)
function createUIStore() {
	const { subscribe, set, update } = writable({
		sidebarOpen: true,
		theme: 'light', // 'light' | 'dark'
		toasts: []
	});

	return {
		subscribe,
		// Toggle sidebar
		toggleSidebar: () => {
			update(state => ({
				...state,
				sidebarOpen: !state.sidebarOpen
			}));
		},
		// Set sidebar state
		setSidebar: (open) => {
			update(state => ({
				...state,
				sidebarOpen: open
			}));
		},
		// Toggle theme
		toggleTheme: () => {
			update(state => {
				const newTheme = state.theme === 'light' ? 'dark' : 'light';
				if (browser) {
					localStorage.setItem('theme', newTheme);
					document.documentElement.setAttribute('data-theme', newTheme);
				}
				return {
					...state,
					theme: newTheme
				};
			});
		},
		// Set theme
		setTheme: (theme) => {
			update(state => {
				if (browser) {
					localStorage.setItem('theme', theme);
					document.documentElement.setAttribute('data-theme', theme);
				}
				return {
					...state,
					theme
				};
			});
		},
		// Initialize theme from localStorage or system preference
		initTheme: () => {
			if (browser) {
				let theme = localStorage.getItem('theme');
				if (!theme) {
					theme = window.matchMedia('(prefers-color-scheme: dark)').matches ? 'dark' : 'light';
				}
				document.documentElement.setAttribute('data-theme', theme);
				update(state => ({
					...state,
					theme
				}));
			}
		},
		// Add toast
		addToast: (toast) => {
			const id = Math.random().toString(36).substr(2, 9);
			const newToast = { id, ...toast };
			
			update(state => ({
				...state,
				toasts: [...state.toasts, newToast]
			}));

			// Auto remove toast after 3.5 seconds
			setTimeout(() => {
				update(state => ({
					...state,
					toasts: state.toasts.filter(t => t.id !== id)
				}));
			}, 3500);

			return id;
		},
		// Remove toast by id
		removeToast: (id) => {
			update(state => ({
				...state,
				toasts: state.toasts.filter(t => t.id !== id)
			}));
		},
		// Clear all toasts
		clearToasts: () => {
			update(state => ({
				...state,
				toasts: []
			}));
		}
	};
}

// Export store instances
export const authStore = createAuthStore();
export const feedsStore = createFeedsStore();
export const uiStore = createUIStore();

// Helper functions for toasts
export const toast = {
	success: (message) => uiStore.addToast({ type: 'success', message }),
	error: (message) => uiStore.addToast({ type: 'error', message }),
	info: (message) => uiStore.addToast({ type: 'info', message }),
	warning: (message) => uiStore.addToast({ type: 'warning', message })
};
