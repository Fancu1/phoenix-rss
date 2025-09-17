<script>
	import { goto } from '$app/navigation';
	import { authStore, toast } from '$lib/stores.js';
	import { users } from '$lib/api.js';

	let username = '';
	let password = '';
	let loading = false;
	let errors = {};

	// Validate form
	function validateForm() {
		const newErrors = {};
		
		if (!username.trim()) {
			newErrors.username = 'Username is required';
		} else if (username.length < 3) {
			newErrors.username = 'Username must be at least 3 characters';
		}
		
		if (!password) {
			newErrors.password = 'Password is required';
		} else if (password.length < 6) {
			newErrors.password = 'Password must be at least 6 characters';
		}
		
		errors = newErrors;
		return Object.keys(newErrors).length === 0;
	}

	// Handle form submission
	async function handleSubmit() {
		if (!validateForm()) return;
		
		loading = true;
		errors = {};

		try {
			const response = await users.login(username, password);
			
			// Store auth data
			authStore.login(response.token, response.user);
			
			toast.success('Login successful!');
			goto('/');
		} catch (error) {
			toast.error(error.message || 'Login failed');
		} finally {
			loading = false;
		}
	}

	// Clear field errors on input
	function clearError(field) {
		if (errors[field]) {
			errors = { ...errors, [field]: '' };
		}
	}
</script>

<svelte:head>
	<title>Login - Phoenix RSS</title>
</svelte:head>

<div class="login-page">
	<div class="login-container">
		<div class="login-header">
			<h1>Welcome Back</h1>
			<p class="text-muted">Sign in to your RSS reader account</p>
		</div>

		<form on:submit|preventDefault={handleSubmit} class="login-form">
			<div class="form-group">
				<label for="username" class="form-label">Username</label>
				<input
					id="username"
					type="text"
					class="input {errors.username ? 'error' : ''}"
					bind:value={username}
					on:input={() => clearError('username')}
					placeholder="Enter your username"
					autocomplete="username"
					disabled={loading}
				/>
				{#if errors.username}
					<div class="form-error">{errors.username}</div>
				{/if}
			</div>

			<div class="form-group">
				<label for="password" class="form-label">Password</label>
				<input
					id="password"
					type="password"
					class="input {errors.password ? 'error' : ''}"
					bind:value={password}
					on:input={() => clearError('password')}
					placeholder="Enter your password"
					autocomplete="current-password"
					disabled={loading}
				/>
				{#if errors.password}
					<div class="form-error">{errors.password}</div>
				{/if}
			</div>

			<button 
				type="submit" 
				class="button primary {loading ? 'loading' : ''}"
				disabled={loading}
			>
				{loading ? '' : 'Sign In'}
			</button>
		</form>

		<div class="login-footer">
			<p class="text-muted text-sm">
				Don't have an account? 
				<a href="/register">Create one here</a>
			</p>
		</div>
	</div>
</div>

<style>
	.login-page {
		min-height: 100vh;
		display: flex;
		align-items: center;
		justify-content: center;
		padding: var(--space-4);
		background: var(--bg);
	}

	.login-container {
		width: 100%;
		max-width: 400px;
		background: var(--bg-elev);
		border: 1px solid var(--border);
		border-radius: var(--radius-lg);
		box-shadow: var(--shadow-md);
		padding: var(--space-6);
	}

	.login-header {
		text-align: center;
		margin-bottom: var(--space-6);
	}

	.login-header h1 {
		margin-bottom: var(--space-1);
		color: var(--text);
	}

	.login-form {
		margin-bottom: var(--space-4);
	}

	.login-form .button {
		width: 100%;
		margin-top: var(--space-2);
	}

	.login-footer {
		text-align: center;
		padding-top: var(--space-4);
		border-top: 1px solid var(--border);
	}

	@media (max-width: 480px) {
		.login-container {
			margin: var(--space-2);
			padding: var(--space-4);
		}
	}
</style>
