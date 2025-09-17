<script>
	import { authStore, uiStore } from '$lib/stores.js';
	import NavBar from '$lib/components/NavBar.svelte';
	import SideBar from '$lib/components/SideBar.svelte';
	import AddSubscriptionModal from '$lib/components/AddSubscriptionModal.svelte';

	let showAddModal = false;

	// Check if user is authenticated to show the app layout
	$: isAuthenticated = $authStore.status === 'authenticated';

	function handleAddSubscription() {
		showAddModal = true;
	}

	function handleModalClose() {
		showAddModal = false;
	}
</script>

<svelte:head>
	<title>About - Phoenix RSS</title>
</svelte:head>

{#if isAuthenticated}
	<div class="main-layout">
		<NavBar on:add-subscription={handleAddSubscription} />
		
		<div class="content-layout">
			<SideBar />
			
			<main class="main-content" class:sidebar-open={$uiStore.sidebarOpen}>
				<div class="about-page">
					<div class="about-content">
						<header class="about-header">
							<div class="logo-section">
								<div class="logo">
									<svg fill="none" viewBox="0 0 24 24" stroke="currentColor">
										<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 5c7.18 0 13 5.82 13 13M6 11a7 7 0 017 7m-6 0a1 1 0 11-2 0 1 1 0 012 0z"></path>
									</svg>
								</div>
								<h1>Phoenix RSS</h1>
								<p class="tagline">A modern, AI-powered RSS aggregator</p>
							</div>
						</header>

						<div class="content-sections">
							<section class="info-section">
								<h2>About Phoenix RSS</h2>
								<p>
									Phoenix RSS is a modern, AI-powered RSS aggregator built on a scalable microservice 
									architecture. It's designed to showcase production-ready patterns for distributed 
									systems while providing an elegant reading experience.
								</p>
								<p>
									Built with Go microservices on the backend and a fast SvelteKit frontend, Phoenix RSS 
									delivers high performance with intelligent content processing through AI-powered summaries.
								</p>
							</section>

							<section class="features-section">
								<h2>Core Features</h2>
								<div class="features-grid">
									<div class="feature-card">
										<div class="feature-icon">
											<svg fill="none" viewBox="0 0 24 24" stroke="currentColor">
												<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 11H5m14 0a2 2 0 012 2v6a2 2 0 01-2 2H5a2 2 0 01-2-2v-6a2 2 0 012-2m14 0V9a2 2 0 00-2-2M5 9a2 2 0 012-2m0 0V5a2 2 0 012-2h6a2 2 0 012 2v2M7 7h10"></path>
											</svg>
										</div>
										<h3>Microservice Architecture</h3>
										<p>Independent services communicating over gRPC for high performance and scalability.</p>
									</div>

									<div class="feature-card">
										<div class="feature-icon">
											<svg fill="none" viewBox="0 0 24 24" stroke="currentColor">
												<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9.663 17h4.673M12 3v1m6.364 1.636l-.707.707M21 12h-1M4 12H3m3.343-5.657l-.707-.707m2.828 9.9a5 5 0 117.072 0l-.548.547A3.374 3.374 0 0014 18.469V19a2 2 0 11-4 0v-.531c0-.895-.356-1.754-.988-2.386l-.548-.547z"></path>
											</svg>
										</div>
										<h3>AI-Powered Processing</h3>
										<p>Automatic article summarization using Large Language Models for better discoverability.</p>
									</div>

									<div class="feature-card">
										<div class="feature-icon">
											<svg fill="none" viewBox="0 0 24 24" stroke="currentColor">
												<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M13 10V3L4 14h7v7l9-11h-7z"></path>
											</svg>
										</div>
										<h3>Event-Driven Core</h3>
										<p>Kafka-based messaging ensures resilience and scalable asynchronous processing.</p>
									</div>

									<div class="feature-card">
										<div class="feature-icon">
											<svg fill="none" viewBox="0 0 24 24" stroke="currentColor">
												<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M20 7l-8-4-8 4m16 0l-8 4m8-4v10l-8 4m0-10L4 7m8 4v10M4 7v10l8 4"></path>
											</svg>
										</div>
										<h3>Containerized & Production-Ready</h3>
										<p>Full Docker containerization with streamlined deployment and health monitoring.</p>
									</div>
								</div>
							</section>

							<section class="tech-section">
								<h2>Technology Stack</h2>
								<div class="tech-grid">
									<div class="tech-category">
										<h3>Backend</h3>
										<ul>
											<li>Go</li>
											<li>Gin (HTTP Framework)</li>
											<li>gRPC + Protocol Buffers</li>
											<li>PostgreSQL</li>
											<li>Kafka</li>
										</ul>
									</div>
									<div class="tech-category">
										<h3>Frontend</h3>
										<ul>
											<li>SvelteKit</li>
											<li>Static Site Generation</li>
											<li>CSS Variables</li>
											<li>Responsive Design</li>
										</ul>
									</div>
									<div class="tech-category">
										<h3>Infrastructure</h3>
										<ul>
											<li>Docker & Docker Compose</li>
											<li>Multi-stage Builds</li>
											<li>Health Checks</li>
											<li>Go Binary Embedding</li>
										</ul>
									</div>
								</div>
							</section>

							<section class="architecture-section">
								<h2>System Architecture</h2>
								<p>
									Phoenix RSS follows a distributed microservice pattern with clear service boundaries 
									and event-driven communication. The system is designed for high availability, 
									scalability, and maintainability.
								</p>
								<div class="architecture-diagram">
									<div class="service-node api">API Gateway</div>
									<div class="service-node user">User Service</div>
									<div class="service-node feed">Feed Service</div>
									<div class="service-node ai">AI Service</div>
									<div class="service-node scheduler">Scheduler</div>
									<div class="service-node db">PostgreSQL</div>
									<div class="service-node kafka">Kafka</div>
								</div>
							</section>
						</div>
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
{:else}
	<!-- Public about page for non-authenticated users -->
	<div class="public-about">
		<div class="public-content">
			<header class="public-header">
				<div class="logo-section">
					<div class="logo">
						<svg fill="none" viewBox="0 0 24 24" stroke="currentColor">
							<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 5c7.18 0 13 5.82 13 13M6 11a7 7 0 017 7m-6 0a1 1 0 11-2 0 1 1 0 012 0z"></path>
						</svg>
					</div>
					<h1>Phoenix RSS</h1>
					<p class="tagline">A modern, AI-powered RSS aggregator</p>
				</div>
				<div class="auth-links">
					<a href="/login" class="button primary">Sign In</a>
					<a href="/register" class="button secondary">Create Account</a>
				</div>
			</header>

			<section class="hero-section">
				<h2>Intelligent RSS Reading</h2>
				<p>
					Experience RSS feeds like never before with AI-powered summaries, 
					real-time updates, and a beautiful, responsive interface.
				</p>
			</section>
		</div>
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

	.about-page {
		max-width: 900px;
		margin: 0 auto;
		padding: var(--space-6) var(--space-4);
	}

	.about-content {
		/* Content styling */
	}

	.about-header,
	.public-header {
		text-align: center;
		margin-bottom: var(--space-6);
		padding-bottom: var(--space-4);
		border-bottom: 1px solid var(--border);
	}

	.public-header {
		display: flex;
		justify-content: space-between;
		align-items: center;
		text-align: left;
	}

	.logo-section {
		display: flex;
		flex-direction: column;
		align-items: center;
		gap: var(--space-2);
	}

	.public-header .logo-section {
		align-items: flex-start;
		flex: 1;
	}

	.logo {
		width: 64px;
		height: 64px;
		background: var(--primary);
		color: var(--primary-contrast);
		border-radius: var(--radius-lg);
		display: flex;
		align-items: center;
		justify-content: center;
		margin-bottom: var(--space-2);
	}

	.logo svg {
		width: 32px;
		height: 32px;
	}

	.about-header h1,
	.public-header h1 {
		margin: 0;
		font-size: 2.5rem;
		font-weight: 700;
		color: var(--text);
	}

	.tagline {
		margin: 0;
		font-size: 1.125rem;
		color: var(--text-muted);
		font-weight: 500;
	}

	.auth-links {
		display: flex;
		gap: var(--space-2);
	}

	.content-sections {
		display: flex;
		flex-direction: column;
		gap: var(--space-6);
	}

	.info-section,
	.features-section,
	.tech-section,
	.architecture-section {
		background: var(--bg-elev);
		border: 1px solid var(--border);
		border-radius: var(--radius-lg);
		padding: var(--space-5);
	}

	.info-section h2,
	.features-section h2,
	.tech-section h2,
	.architecture-section h2 {
		margin: 0 0 var(--space-4) 0;
		color: var(--text);
	}

	.info-section p,
	.architecture-section p {
		margin: 0 0 var(--space-3) 0;
		line-height: 1.6;
		color: var(--text-muted);
	}

	.info-section p:last-child,
	.architecture-section p:last-child {
		margin-bottom: 0;
	}

	.features-grid {
		display: grid;
		grid-template-columns: repeat(auto-fit, minmax(280px, 1fr));
		gap: var(--space-4);
	}

	.feature-card {
		background: var(--bg);
		border: 1px solid var(--border);
		border-radius: var(--radius-md);
		padding: var(--space-4);
	}

	.feature-icon {
		width: 40px;
		height: 40px;
		background: var(--primary);
		color: var(--primary-contrast);
		border-radius: var(--radius-sm);
		display: flex;
		align-items: center;
		justify-content: center;
		margin-bottom: var(--space-3);
	}

	.feature-icon svg {
		width: 20px;
		height: 20px;
	}

	.feature-card h3 {
		margin: 0 0 var(--space-2) 0;
		font-size: 1.125rem;
		color: var(--text);
	}

	.feature-card p {
		margin: 0;
		font-size: 0.875rem;
		line-height: 1.5;
		color: var(--text-muted);
	}

	.tech-grid {
		display: grid;
		grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
		gap: var(--space-4);
	}

	.tech-category {
		background: var(--bg);
		border: 1px solid var(--border);
		border-radius: var(--radius-md);
		padding: var(--space-4);
	}

	.tech-category h3 {
		margin: 0 0 var(--space-3) 0;
		color: var(--text);
		font-size: 1.125rem;
	}

	.tech-category ul {
		margin: 0;
		padding: 0;
		list-style: none;
	}

	.tech-category li {
		padding: var(--space-1) 0;
		color: var(--text-muted);
		font-size: 0.875rem;
		border-bottom: 1px solid var(--border);
	}

	.tech-category li:last-child {
		border-bottom: none;
	}

	.architecture-diagram {
		display: grid;
		grid-template-columns: repeat(auto-fit, minmax(120px, 1fr));
		gap: var(--space-3);
		margin-top: var(--space-4);
	}

	.service-node {
		background: var(--bg);
		border: 2px solid var(--border);
		border-radius: var(--radius-md);
		padding: var(--space-3);
		text-align: center;
		font-weight: 600;
		font-size: 0.875rem;
		color: var(--text);
		transition: all 0.2s ease;
	}

	.service-node:hover {
		border-color: var(--primary);
		transform: translateY(-2px);
		box-shadow: var(--shadow-md);
	}

	.service-node.api { border-color: #3b82f6; }
	.service-node.user { border-color: #10b981; }
	.service-node.feed { border-color: #f59e0b; }
	.service-node.ai { border-color: #8b5cf6; }
	.service-node.scheduler { border-color: #ef4444; }
	.service-node.db { border-color: #6b7280; }
	.service-node.kafka { border-color: #f97316; }

	.public-about {
		min-height: 100vh;
		background: var(--bg);
		padding: var(--space-6) var(--space-4);
	}

	.public-content {
		max-width: 900px;
		margin: 0 auto;
	}

	.hero-section {
		text-align: center;
		padding: var(--space-6) 0;
	}

	.hero-section h2 {
		margin: 0 0 var(--space-4) 0;
		font-size: 2rem;
		color: var(--text);
	}

	.hero-section p {
		margin: 0;
		font-size: 1.125rem;
		line-height: 1.6;
		color: var(--text-muted);
		max-width: 600px;
		margin: 0 auto;
	}

	@media (max-width: 768px) {
		.main-content {
			margin-left: 0;
		}

		.main-content.sidebar-open {
			margin-left: 0;
		}

		.about-page {
			padding: var(--space-4) var(--space-3);
		}

		.public-header {
			flex-direction: column;
			gap: var(--space-4);
			text-align: center;
		}

		.public-header .logo-section {
			align-items: center;
		}

		.about-header h1,
		.public-header h1 {
			font-size: 2rem;
		}

		.features-grid {
			grid-template-columns: 1fr;
		}

		.tech-grid {
			grid-template-columns: 1fr;
		}

		.architecture-diagram {
			grid-template-columns: repeat(2, 1fr);
		}
	}
</style>
