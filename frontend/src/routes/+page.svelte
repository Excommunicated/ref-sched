<script lang="ts">
	import type { PageData } from './$types';

	export let data: PageData;

	const API_URL = import.meta.env.VITE_API_URL || 'http://localhost:8080';

	function handleGoogleLogin() {
		window.location.href = `${API_URL}/api/auth/google`;
	}
</script>

<svelte:head>
	<title>Referee Scheduler - Login</title>
</svelte:head>

<div class="login-container">
	<div class="login-card card">
		<div class="logo-container">
			<img src="/logo.svg" alt="Referee Scheduler Logo" class="logo" />
		</div>
		<h1>Referee Scheduler</h1>
		<p class="subtitle">Streamline your match assignments</p>

		{#if !data.user}
			<button on:click={handleGoogleLogin} class="google-btn">
				<svg width="18" height="18" viewBox="0 0 18 18">
					<path
						fill="#4285F4"
						d="M17.64 9.2c0-.637-.057-1.251-.164-1.84H9v3.481h4.844c-.209 1.125-.843 2.078-1.796 2.717v2.258h2.908c1.702-1.567 2.684-3.875 2.684-6.615z"
					/>
					<path
						fill="#34A853"
						d="M9 18c2.43 0 4.467-.806 5.956-2.18l-2.908-2.259c-.806.54-1.837.86-3.048.86-2.344 0-4.328-1.584-5.036-3.711H.957v2.332C2.438 15.983 5.482 18 9 18z"
					/>
					<path
						fill="#FBBC05"
						d="M3.964 10.71c-.18-.54-.282-1.117-.282-1.71s.102-1.17.282-1.71V4.958H.957C.347 6.173 0 7.548 0 9s.348 2.827.957 4.042l3.007-2.332z"
					/>
					<path
						fill="#EA4335"
						d="M9 3.58c1.321 0 2.508.454 3.44 1.345l2.582-2.58C13.463.891 11.426 0 9 0 5.482 0 2.438 2.017.957 4.958L3.964 7.29C4.672 5.163 6.656 3.58 9 3.58z"
					/>
				</svg>
				Sign in with Google
			</button>
		{:else}
			<p class="welcome">Welcome, {data.user.name}!</p>
			<p class="role-info">Role: {data.user.role}</p>
		{/if}
	</div>
</div>

<style>
	.login-container {
		display: flex;
		justify-content: center;
		align-items: center;
		min-height: 100vh;
		padding: 1rem;
	}

	.login-card {
		max-width: 400px;
		width: 100%;
		text-align: center;
	}

	.logo-container {
		display: flex;
		justify-content: center;
		margin-bottom: 1.5rem;
	}

	.logo {
		height: 80px;
		width: auto;
	}

	h1 {
		font-size: 2rem;
		font-weight: 700;
		color: var(--text-primary);
		margin-bottom: 0.5rem;
	}

	.subtitle {
		color: var(--text-secondary);
		margin-bottom: 2rem;
	}

	.google-btn {
		display: flex;
		align-items: center;
		justify-content: center;
		gap: 0.75rem;
		width: 100%;
		padding: 0.75rem 1.5rem;
		font-size: 1rem;
		font-weight: 500;
		color: var(--text-primary);
		background-color: white;
		border: 1px solid var(--border-color);
		border-radius: 0.375rem;
		transition: all 0.2s;
		box-shadow: var(--shadow);
	}

	.google-btn:hover {
		background-color: var(--bg-secondary);
		box-shadow: 0 4px 6px -1px rgb(0 0 0 / 0.1), 0 2px 4px -2px rgb(0 0 0 / 0.1);
	}

	.welcome {
		font-size: 1.25rem;
		font-weight: 600;
		color: var(--text-primary);
		margin-bottom: 0.5rem;
	}

	.role-info {
		color: var(--text-secondary);
	}
</style>
