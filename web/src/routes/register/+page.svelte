<script>
	import { goto } from '$app/navigation';
	import { authStore, toast } from '$lib/stores.js';
	import { users } from '$lib/api.js';

	let username = '';
	let password = '';
	let confirmPassword = '';
	let loading = false;
	let errors = {};

	// Validate form
	function validateForm() {
		const newErrors = {};
		
		if (!username.trim()) {
			newErrors.username = 'Username is required';
		} else if (username.length < 3) {
			newErrors.username = 'Username must be at least 3 characters';
		} else if (username.length > 50) {
			newErrors.username = 'Username must be less than 50 characters';
		}
		
		if (!password) {
			newErrors.password = 'Password is required';
		} else if (password.length < 6) {
			newErrors.password = 'Password must be at least 6 characters';
		}
		
		if (!confirmPassword) {
			newErrors.confirmPassword = 'Please confirm your password';
		} else if (password !== confirmPassword) {
			newErrors.confirmPassword = 'Passwords do not match';
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
			const response = await users.register(username, password);
			
			// Store auth data
			authStore.login(response.token, response.user);
			
			toast.success('Account created successfully!');
			goto('/');
		} catch (error) {
			toast.error(error.message || 'Registration failed');
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

	// Real-time password confirmation validation
	function validatePasswordMatch() {
		if (confirmPassword && password !== confirmPassword) {
			errors = { ...errors, confirmPassword: 'Passwords do not match' };
		} else {
			clearError('confirmPassword');
		}
	}
</script>

<svelte:head>
	<title>Register - Phoenix RSS</title>
</svelte:head>

<div class="register-page">
	<div class="register-container">
		<div class="register-header">
			<h1>Create Account</h1>
			<p class="text-muted">Join Phoenix RSS to start reading feeds</p>
		</div>

		<form on:submit|preventDefault={handleSubmit} class="register-form">
			<div class="form-group">
				<label for="username" class="form-label">Username</label>
				<input
					id="username"
					type="text"
					class="input {errors.username ? 'error' : ''}"
					bind:value={username}
					on:input={() => clearError('username')}
					placeholder="Choose a username"
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
					on:input={() => {
						clearError('password');
						if (confirmPassword) validatePasswordMatch();
					}}
					placeholder="Create a password"
					autocomplete="new-password"
					disabled={loading}
				/>
				{#if errors.password}
					<div class="form-error">{errors.password}</div>
				{/if}
			</div>

			<div class="form-group">
				<label for="confirmPassword" class="form-label">Confirm Password</label>
				<input
					id="confirmPassword"
					type="password"
					class="input {errors.confirmPassword ? 'error' : ''}"
					bind:value={confirmPassword}
					on:input={() => {
						clearError('confirmPassword');
						validatePasswordMatch();
					}}
					placeholder="Confirm your password"
					autocomplete="new-password"
					disabled={loading}
				/>
				{#if errors.confirmPassword}
					<div class="form-error">{errors.confirmPassword}</div>
				{/if}
			</div>

			<button 
				type="submit" 
				class="button primary {loading ? 'loading' : ''}"
				disabled={loading}
			>
				{loading ? '' : 'Create Account'}
			</button>
		</form>

		<div class="register-footer">
			<p class="text-muted text-sm">
				Already have an account? 
				<a href="/login">Sign in here</a>
			</p>
		</div>
	</div>
</div>

<style>
	.register-page {
		min-height: 100vh;
		display: flex;
		align-items: center;
		justify-content: center;
		padding: var(--space-4);
		background: var(--bg);
	}

	.register-container {
		width: 100%;
		max-width: 400px;
		background: var(--bg-elev);
		border: 1px solid var(--border);
		border-radius: var(--radius-lg);
		box-shadow: var(--shadow-md);
		padding: var(--space-6);
	}

	.register-header {
		text-align: center;
		margin-bottom: var(--space-6);
	}

	.register-header h1 {
		margin-bottom: var(--space-1);
		color: var(--text);
	}

	.register-form {
		margin-bottom: var(--space-4);
	}

	.register-form .button {
		width: 100%;
		margin-top: var(--space-2);
	}

	.register-footer {
		text-align: center;
		padding-top: var(--space-4);
		border-top: 1px solid var(--border);
	}

	@media (max-width: 480px) {
		.register-container {
			margin: var(--space-2);
			padding: var(--space-4);
		}
	}
</style>
