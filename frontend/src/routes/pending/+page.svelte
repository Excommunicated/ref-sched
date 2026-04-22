<script lang="ts">
	import { goto } from '$app/navigation';
	import type { PageData } from './$types';

	export let data: PageData;

	const API_URL = import.meta.env.VITE_API_URL || 'http://localhost:8080';

	async function handleLogout() {
		try {
			await fetch(`${API_URL}/api/auth/logout`, {
				method: 'POST',
				credentials: 'include'
			});
			goto('/');
		} catch (error) {
			console.error('Logout error:', error);
		}
	}
</script>

<svelte:head>
	<title>Pending Activation - Referee Scheduler</title>
</svelte:head>

<div class="container">
	<div class="pending-card card">
		<h1>Account Pending Activation</h1>
		<p class="message">
			Your account has been created successfully! An assignor needs to verify and activate your
			account before you can view matches and mark your availability.
		</p>

		<div class="info-box">
			<h2>What's next?</h2>
			<ol>
				<li>Complete your profile with your date of birth and certification details</li>
				<li>Wait for an assignor to activate your account</li>
				<li>Once activated, you'll be able to view and mark your availability for matches</li>
			</ol>
		</div>

		<div class="actions">
			<a href="/referee/profile" class="btn btn-primary">Edit Profile</a>
			<button on:click={handleLogout} class="btn btn-secondary">Sign Out</button>
		</div>
	</div>
</div>

<style>
	.container {
		display: flex;
		justify-content: center;
		align-items: center;
		min-height: 100vh;
		padding: 1rem;
	}

	.pending-card {
		max-width: 600px;
		width: 100%;
	}

	h1 {
		font-size: 1.875rem;
		font-weight: 700;
		color: var(--text-primary);
		margin-bottom: 1rem;
	}

	.message {
		color: var(--text-secondary);
		margin-bottom: 1.5rem;
		line-height: 1.6;
	}

	.info-box {
		background-color: var(--bg-secondary);
		border-left: 4px solid var(--primary-color);
		padding: 1rem;
		margin-bottom: 1.5rem;
		border-radius: 0.25rem;
	}

	.info-box h2 {
		font-size: 1.125rem;
		font-weight: 600;
		margin-bottom: 0.75rem;
		color: var(--text-primary);
	}

	.info-box ol {
		margin-left: 1.25rem;
		color: var(--text-secondary);
	}

	.info-box li {
		margin-bottom: 0.5rem;
	}

	.actions {
		display: flex;
		gap: 1rem;
		flex-wrap: wrap;
	}

	.btn-secondary {
		background-color: var(--bg-secondary);
		color: var(--text-primary);
		border: 1px solid var(--border-color);
	}

	.btn-secondary:hover {
		background-color: #f3f4f6;
	}

	@media (max-width: 640px) {
		.actions {
			flex-direction: column;
		}

		.btn {
			width: 100%;
		}
	}
</style>
