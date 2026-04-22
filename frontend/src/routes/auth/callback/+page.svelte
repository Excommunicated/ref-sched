<script lang="ts">
	import { onMount } from 'svelte';
	import { goto } from '$app/navigation';

	const API_URL = import.meta.env.VITE_API_URL || 'http://localhost:8080';

	onMount(async () => {
		// Fetch user info to determine role
		try {
			const response = await fetch(`${API_URL}/api/auth/me`, {
				credentials: 'include'
			});

			if (response.ok) {
				const user = await response.json();

				// Redirect based on role
				if (user.role === 'pending_referee') {
					goto('/pending');
				} else {
					goto('/dashboard');
				}
			} else {
				goto('/');
			}
		} catch (error) {
			console.error('Authentication error:', error);
			goto('/');
		}
	});
</script>

<svelte:head>
	<title>Authenticating...</title>
</svelte:head>

<div class="loading-container">
	<div class="spinner"></div>
	<p>Signing you in...</p>
</div>

<style>
	.loading-container {
		display: flex;
		flex-direction: column;
		justify-content: center;
		align-items: center;
		min-height: 100vh;
		gap: 1rem;
	}

	.spinner {
		width: 40px;
		height: 40px;
		border: 4px solid var(--border-color);
		border-top-color: var(--primary-color);
		border-radius: 50%;
		animation: spin 0.8s linear infinite;
	}

	@keyframes spin {
		to {
			transform: rotate(360deg);
		}
	}

	p {
		color: var(--text-secondary);
	}
</style>
