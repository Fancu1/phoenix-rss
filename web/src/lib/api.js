// @ts-check
import { get } from 'svelte/store';
import { authStore } from './stores.js';

// API base configuration
const API_BASE = '/api/v1';
const DEFAULT_TIMEOUT = 15000; // 15 seconds

// Custom error class for API errors
export class APIError extends Error {
	constructor(message, code, status) {
		super(message);
		this.name = 'APIError';
		this.code = code;
		this.status = status;
	}
}

// Create fetch wrapper with automatic auth injection and error handling
async function apiFetch(endpoint, options = {}) {
	const controller = new AbortController();
	const timeoutId = setTimeout(() => controller.abort(), DEFAULT_TIMEOUT);

	try {
		// Get current auth state
		const auth = get(authStore);
		
		// Build headers
		const headers = {
			'Content-Type': 'application/json',
			...options.headers
		};

		// Add authorization header if token exists
		if (auth.token) {
			headers.Authorization = `Bearer ${auth.token}`;
		}

		// Make request
		const response = await fetch(`${API_BASE}${endpoint}`, {
			...options,
			headers,
			signal: controller.signal
		});

		clearTimeout(timeoutId);

		// Handle different response types
		if (!response.ok) {
			let errorData;
			try {
				errorData = await response.json();
			} catch {
				// If we can't parse JSON, create a generic error
				errorData = {
					code: response.status,
					message: response.statusText || 'An error occurred'
				};
			}

			// Handle 401 unauthorized - logout user
			if (response.status === 401) {
				authStore.logout();
			}

			throw new APIError(
				errorData.message || 'An error occurred',
				errorData.code || response.status,
				response.status
			);
		}

		// Return parsed JSON
		return await response.json();
	} catch (error) {
		clearTimeout(timeoutId);
		
		// Handle AbortError (timeout)
		if (error.name === 'AbortError') {
			throw new APIError('Request timeout', 408, 408);
		}
		
		// Re-throw API errors
		if (error instanceof APIError) {
			throw error;
		}
		
		// Handle network errors
		throw new APIError('Network error', 0, 0);
	}
}

// API methods for different resources

// Health check
export const health = {
	check: () => apiFetch('/health')
};

// User authentication
export const users = {
	register: (username, password) => 
		apiFetch('/users/register', {
			method: 'POST',
			body: JSON.stringify({ username, password })
		}),
	
	login: (username, password) => 
		apiFetch('/users/login', {
			method: 'POST',
			body: JSON.stringify({ username, password })
		})
};

// Feed management
export const feeds = {
	// Get all user's feeds
	list: () => apiFetch('/feeds'),
	
	// Subscribe to a new feed
	subscribe: (url) => 
		apiFetch('/feeds', {
			method: 'POST',
			body: JSON.stringify({ url })
		}),
	
	// Unsubscribe from a feed
	unsubscribe: (feedId) => 
		apiFetch(`/feeds/${feedId}`, {
			method: 'DELETE'
		}),
	
	// Get articles for a specific feed
	getArticles: (feedId) => 
		apiFetch(`/feeds/${feedId}/articles`),
	
	// Trigger feed fetch
	fetch: (feedId) => 
		apiFetch(`/feeds/${feedId}/fetch`, {
			method: 'POST'
		}),
	
	// Export subscriptions as OPML file
	exportOPML: async () => {
		const auth = get(authStore);
		const response = await fetch(`${API_BASE}/feeds/export`, {
			headers: {
				Authorization: `Bearer ${auth.token}`
			}
		});
		
		if (!response.ok) {
			throw new APIError('Failed to export subscriptions', response.status, response.status);
		}
		
		// Get filename from Content-Disposition header or use default
		const contentDisposition = response.headers.get('Content-Disposition');
		let filename = 'phoenix-rss-subscriptions.opml';
		if (contentDisposition) {
			const match = contentDisposition.match(/filename=(.+)/);
			if (match) {
				filename = match[1];
			}
		}
		
		// Create blob and trigger download
		const blob = await response.blob();
		const url = window.URL.createObjectURL(blob);
		const a = document.createElement('a');
		a.href = url;
		a.download = filename;
		document.body.appendChild(a);
		a.click();
		window.URL.revokeObjectURL(url);
		document.body.removeChild(a);
	},
	
	// Preview OPML import (parse file and check for duplicates)
	previewImport: async (file) => {
		const auth = get(authStore);
		const formData = new FormData();
		formData.append('file', file);
		
		const response = await fetch(`${API_BASE}/feeds/import/preview`, {
			method: 'POST',
			headers: {
				Authorization: `Bearer ${auth.token}`
			},
			body: formData
		});
		
		if (!response.ok) {
			let errorData;
			try {
				errorData = await response.json();
			} catch {
				errorData = { message: 'Failed to preview import' };
			}
			throw new APIError(errorData.message, response.status, response.status);
		}
		
		return await response.json();
	},
	
	// Import feeds from OPML preview
	importFeeds: (feeds) => 
		apiFetch('/feeds/import', {
			method: 'POST',
			body: JSON.stringify({ feeds })
		})
};

export const articles = {
	getById: (articleId) => apiFetch(`/articles/${articleId}`)
};

// Export the base fetch function for custom requests
export { apiFetch };
