<script>
	import { createEventDispatcher } from 'svelte';

	export let page = 1;
	export let totalPages = 1;
	export let total = 0;
	export let pageSize = 8;

	const dispatch = createEventDispatcher();

	// Compute visible page numbers with ellipsis logic
	$: visiblePages = computeVisiblePages(page, totalPages);

	function computeVisiblePages(current, total) {
		if (total <= 7) {
			// Show all pages if 7 or fewer
			return Array.from({ length: total }, (_, i) => i + 1);
		}

		const pages = [];
		const showEllipsisBefore = current > 4;
		const showEllipsisAfter = current < total - 3;

		// Always show first page
		pages.push(1);

		if (showEllipsisBefore) {
			pages.push('...');
		}

		// Calculate middle range
		let start = showEllipsisBefore ? current - 1 : 2;
		let end = showEllipsisAfter ? current + 1 : total - 1;

		// Ensure we show at least 3 numbers in the middle
		if (!showEllipsisBefore) {
			end = Math.min(5, total - 1);
		}
		if (!showEllipsisAfter) {
			start = Math.max(total - 4, 2);
		}

		for (let i = start; i <= end; i++) {
			if (i > 1 && i < total) {
				pages.push(i);
			}
		}

		if (showEllipsisAfter) {
			pages.push('...');
		}

		// Always show last page
		if (total > 1) {
			pages.push(total);
		}

		return pages;
	}

	function goToPage(newPage) {
		if (newPage < 1 || newPage > totalPages || newPage === page) return;
		dispatch('pageChange', { page: newPage });
	}

	function handlePrevious() {
		if (page > 1) {
			goToPage(page - 1);
		}
	}

	function handleNext() {
		if (page < totalPages) {
			goToPage(page + 1);
		}
	}

	// Calculate display range for "Showing X-Y of Z"
	$: startItem = total === 0 ? 0 : (page - 1) * pageSize + 1;
	$: endItem = Math.min(page * pageSize, total);
</script>

{#if totalPages > 1}
	<nav class="pagination" aria-label="Pagination">
		<div class="pagination-info">
			<span class="text-muted">
				Showing {startItem}–{endItem} of {total}
			</span>
		</div>

		<div class="pagination-controls">
			<button
				class="pagination-btn prev"
				on:click={handlePrevious}
				disabled={page <= 1}
				aria-label="Previous page"
			>
				<svg fill="none" viewBox="0 0 24 24" stroke="currentColor">
					<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 19l-7-7 7-7" />
				</svg>
				<span class="btn-text">Previous</span>
			</button>

			<div class="pagination-pages">
				{#each visiblePages as pageNum}
					{#if pageNum === '...'}
						<span class="pagination-ellipsis">…</span>
					{:else}
						<button
							class="pagination-page"
							class:active={pageNum === page}
							on:click={() => goToPage(pageNum)}
							aria-label="Page {pageNum}"
							aria-current={pageNum === page ? 'page' : undefined}
						>
							{pageNum}
						</button>
					{/if}
				{/each}
			</div>

			<button
				class="pagination-btn next"
				on:click={handleNext}
				disabled={page >= totalPages}
				aria-label="Next page"
			>
				<span class="btn-text">Next</span>
				<svg fill="none" viewBox="0 0 24 24" stroke="currentColor">
					<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 5l7 7-7 7" />
				</svg>
			</button>
		</div>
	</nav>
{/if}

<style>
	.pagination {
		display: flex;
		flex-direction: column;
		align-items: center;
		gap: var(--space-3);
		padding: var(--space-4) 0;
		margin-top: var(--space-4);
		border-top: 1px solid var(--border);
	}

	.pagination-info {
		font-size: 0.875rem;
	}

	.pagination-controls {
		display: flex;
		align-items: center;
		gap: var(--space-2);
	}

	.pagination-btn {
		display: flex;
		align-items: center;
		gap: var(--space-1);
		padding: var(--space-2) var(--space-3);
		font-size: 0.875rem;
		font-weight: 500;
		color: var(--text);
		background: var(--bg-elev);
		border: 1px solid var(--border);
		border-radius: var(--radius-md);
		cursor: pointer;
		transition: all 0.15s ease;
	}

	.pagination-btn:hover:not(:disabled) {
		background: var(--bg-hover);
		border-color: var(--primary);
	}

	.pagination-btn:disabled {
		opacity: 0.5;
		cursor: not-allowed;
	}

	.pagination-btn svg {
		width: 16px;
		height: 16px;
	}

	.pagination-pages {
		display: flex;
		align-items: center;
		gap: var(--space-1);
	}

	.pagination-page {
		min-width: 36px;
		height: 36px;
		display: flex;
		align-items: center;
		justify-content: center;
		font-size: 0.875rem;
		font-weight: 500;
		color: var(--text);
		background: transparent;
		border: 1px solid transparent;
		border-radius: var(--radius-md);
		cursor: pointer;
		transition: all 0.15s ease;
	}

	.pagination-page:hover:not(.active) {
		background: var(--bg-hover);
		border-color: var(--border);
	}

	.pagination-page.active {
		color: white;
		background: var(--primary);
		border-color: var(--primary);
	}

	.pagination-ellipsis {
		min-width: 36px;
		height: 36px;
		display: flex;
		align-items: center;
		justify-content: center;
		color: var(--text-muted);
		font-weight: 500;
	}

	@media (max-width: 640px) {
		.pagination-controls {
			width: 100%;
			justify-content: space-between;
		}

		.btn-text {
			display: none;
		}

		.pagination-btn {
			padding: var(--space-2);
		}

		.pagination-pages {
			gap: 0;
		}

		.pagination-page {
			min-width: 32px;
			height: 32px;
			font-size: 0.8rem;
		}

		.pagination-ellipsis {
			min-width: 24px;
		}
	}
</style>

