<script>
	import { onMount } from 'svelte';
	import { page } from '$app/stores';
	import { authStore, uiStore } from '$lib/stores.js';
	import Toast from '$lib/components/Toast.svelte';
	import '../app.css';

	// Initialize stores on mount
	onMount(() => {
		authStore.init();
		uiStore.initTheme();
	});

	// Check if current route is public (doesn't require auth)
	$: isPublicRoute = ['/login', '/register', '/about'].includes($page.route.id);
	$: isAuthRoute = ['/login', '/register'].includes($page.route.id);
</script>

<!-- Show auth layout for login/register -->
{#if isAuthRoute}
	<slot />
{:else}
	<!-- Main app layout -->
	<div class="app-layout">
		<slot />
	</div>
{/if}

<!-- Global toast container -->
<Toast />

<style>
	.app-layout {
		min-height: 100vh;
		background: var(--bg);
	}
</style>
