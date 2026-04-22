<script lang="ts">
	import { onMount } from 'svelte';
	import { goto } from '$app/navigation';

	onMount(() => {
		goto('/dashboard');
	});
</script>

<svelte:head>
	<title>Redirecting...</title>
</svelte:head>

<div class="loading-container">
	<p>Redirecting to dashboard...</p>
</div>

<style>
	.loading-container {
		display: flex;
		justify-content: center;
		align-items: center;
		min-height: 100vh;
	}

	p {
		color: var(--text-secondary);
	}
</style>
