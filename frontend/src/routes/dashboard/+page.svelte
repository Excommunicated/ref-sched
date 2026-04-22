<script lang="ts">
	import { onMount } from 'svelte';
	import { goto } from '$app/navigation';

	const API_URL = import.meta.env.VITE_API_URL || 'http://localhost:8080';

	interface User {
		id: number;
		email: string;
		name: string;
		role: string;
		first_name?: string;
		last_name?: string;
	}

	interface Match {
		id: number;
		event_name: string;
		team_name: string;
		age_group: string;
		match_date: string;
		start_time: string;
		end_time: string;
		location: string;
		description: string | null;
		status: string;
		is_assigned: boolean;
		assigned_role: string | null;
		is_available: boolean;
		acknowledged: boolean;
	}

	let user: User | null = null;
	let matches: Match[] = [];
	let loading = true;
	let error = '';

	onMount(async () => {
		await loadUser();
		await loadMatches();
	});

	async function loadUser() {
		try {
			const response = await fetch(`${API_URL}/api/auth/me`, {
				credentials: 'include'
			});

			if (response.ok) {
				user = await response.json();
			} else {
				goto('/');
			}
		} catch (err) {
			console.error('Failed to load user:', err);
			goto('/');
		}
	}

	async function loadMatches() {
		loading = true;
		error = '';

		try {
			const response = await fetch(`${API_URL}/api/referee/matches`, {
				credentials: 'include'
			});

			if (response.ok) {
				matches = await response.json();
			} else {
				error = 'Failed to load matches';
			}
		} catch (err) {
			error = 'Failed to load matches';
			console.error(err);
		} finally {
			loading = false;
		}
	}

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

	function formatDate(dateString: string): string {
		const [year, month, day] = dateString.split('-').map(Number);
		const date = new Date(year, month - 1, day);
		return date.toLocaleDateString('en-US', {
			weekday: 'short',
			month: 'short',
			day: 'numeric'
		});
	}

	function formatTime(timeString: string): string {
		const parts = timeString.split(':');
		const hour = parseInt(parts[0]);
		const minute = parts[1];
		const ampm = hour >= 12 ? 'PM' : 'AM';
		const displayHour = hour === 0 ? 12 : hour > 12 ? hour - 12 : hour;
		return `${displayHour}:${minute} ${ampm}`;
	}

	function getRoleName(role: string): string {
		const roleMap: Record<string, string> = {
			center: 'Center Referee',
			assistant_1: 'Assistant Referee 1',
			assistant_2: 'Assistant Referee 2'
		};
		return roleMap[role] || role;
	}

	$: displayName = user?.first_name && user?.last_name
		? `${user.first_name} ${user.last_name}`
		: user?.name || 'User';

	$: assignedMatches = matches.filter((m) => m.is_assigned).slice(0, 5);
	$: availableMatches = matches.filter((m) => !m.is_assigned && m.is_available).slice(0, 5);
	$: upcomingUnmarked = matches.filter((m) => !m.is_assigned && !m.is_available).slice(0, 3);
</script>

<svelte:head>
	<title>Dashboard - Referee Scheduler</title>
</svelte:head>

<div class="container">
	<header class="header">
		<div class="header-left">
			<img src="/logo.svg" alt="Logo" class="header-logo" />
			<h1>Dashboard</h1>
		</div>
		<button on:click={handleLogout} class="btn btn-secondary">Sign Out</button>
	</header>

	{#if user}
		<div class="welcome-section">
			<h2>Welcome back, {displayName}!</h2>
			<p class="role-badge">
				{#if user.role === 'assignor'}
					<span class="badge badge-assignor">Assignor</span>
				{:else if user.role === 'referee'}
					<span class="badge badge-referee">Referee</span>
				{/if}
			</p>
		</div>

		<div class="navigation-section">
			<h3>Quick Actions</h3>
			<div class="nav-grid">
				{#if user.role === 'assignor'}
					<a href="/assignor/matches" class="nav-card">
						<div class="nav-icon">📋</div>
						<div class="nav-title">Match Schedule</div>
						<div class="nav-description">View and manage all matches</div>
					</a>
					<a href="/assignor/referees" class="nav-card">
						<div class="nav-icon">👥</div>
						<div class="nav-title">Manage Referees</div>
						<div class="nav-description">View and approve referees</div>
					</a>
					<a href="/assignor/matches/import" class="nav-card">
						<div class="nav-icon">📥</div>
						<div class="nav-title">Import Matches</div>
						<div class="nav-description">Upload match schedule</div>
					</a>
					<a href="/referee/matches" class="nav-card">
						<div class="nav-icon">✅</div>
						<div class="nav-title">My Availability</div>
						<div class="nav-description">Mark your availability</div>
					</a>
					<a href="/referee/profile" class="nav-card">
						<div class="nav-icon">👤</div>
						<div class="nav-title">My Profile</div>
						<div class="nav-description">Update your information</div>
					</a>
				{:else}
					<a href="/referee/matches" class="nav-card">
						<div class="nav-icon">⚽</div>
						<div class="nav-title">My Matches</div>
						<div class="nav-description">View and mark availability</div>
					</a>
					<a href="/referee/profile" class="nav-card">
						<div class="nav-icon">👤</div>
						<div class="nav-title">My Profile</div>
						<div class="nav-description">Update your information</div>
					</a>
				{/if}
			</div>
		</div>

		<div class="matches-section">
			<h3>Upcoming Matches</h3>

			{#if loading}
				<p class="loading-text">Loading matches...</p>
			{:else if error}
				<div class="error-box">
					<p>{error}</p>
				</div>
			{:else if assignedMatches.length === 0 && availableMatches.length === 0 && upcomingUnmarked.length === 0}
				<div class="info-box">
					<p>No upcoming matches at this time.</p>
					{#if user.role === 'referee'}
						<p>Check back later or contact your assignor.</p>
					{/if}
				</div>
			{:else}
				{#if assignedMatches.length > 0}
					<div class="match-group">
						<h4>My Assignments ({assignedMatches.length})</h4>
						<div class="match-list">
							{#each assignedMatches as match}
								<div class="match-item assigned">
									<div class="match-header">
										<span class="match-title">{match.event_name}</span>
										<span class="match-role">{getRoleName(match.assigned_role || '')}</span>
										{#if match.acknowledged}
											<span class="ack-status confirmed">✓</span>
										{:else}
											<span class="ack-status pending">!</span>
										{/if}
									</div>
									<div class="match-details">
										<span class="match-date">📅 {formatDate(match.match_date)}</span>
										<span class="match-time">🕐 {formatTime(match.start_time)}</span>
										<span class="match-location">📍 {match.location}</span>
									</div>
									<div class="match-info">
										<span class="age-badge">{match.age_group}</span>
										<span class="team-name">{match.team_name}</span>
									</div>
								</div>
							{/each}
						</div>
						{#if matches.filter((m) => m.is_assigned).length > 5}
							<a href="/referee/matches" class="view-all-link">View all assignments →</a>
						{/if}
					</div>
				{/if}

				{#if availableMatches.length > 0}
					<div class="match-group">
						<h4>Marked Available ({availableMatches.length})</h4>
						<div class="match-list">
							{#each availableMatches as match}
								<div class="match-item available">
									<div class="match-header">
										<span class="match-title">{match.event_name}</span>
									</div>
									<div class="match-details">
										<span class="match-date">📅 {formatDate(match.match_date)}</span>
										<span class="match-time">🕐 {formatTime(match.start_time)}</span>
										<span class="match-location">📍 {match.location}</span>
									</div>
									<div class="match-info">
										<span class="age-badge">{match.age_group}</span>
										<span class="team-name">{match.team_name}</span>
									</div>
								</div>
							{/each}
						</div>
						{#if matches.filter((m) => !m.is_assigned && m.is_available).length > 5}
							<a href="/referee/matches" class="view-all-link">View all available matches →</a>
						{/if}
					</div>
				{/if}

				{#if upcomingUnmarked.length > 0}
					<div class="match-group">
						<h4>Action Needed</h4>
						<p class="group-description">Mark your availability for these matches</p>
						<div class="match-list">
							{#each upcomingUnmarked as match}
								<div class="match-item unmarked">
									<div class="match-header">
										<span class="match-title">{match.event_name}</span>
									</div>
									<div class="match-details">
										<span class="match-date">📅 {formatDate(match.match_date)}</span>
										<span class="match-time">🕐 {formatTime(match.start_time)}</span>
										<span class="match-location">📍 {match.location}</span>
									</div>
									<div class="match-info">
										<span class="age-badge">{match.age_group}</span>
										<span class="team-name">{match.team_name}</span>
									</div>
								</div>
							{/each}
						</div>
						<a href="/referee/matches" class="view-all-link">Mark availability →</a>
					</div>
				{/if}
			{/if}
		</div>
	{/if}
</div>

<style>
	.container {
		max-width: 1200px;
		margin: 0 auto;
		padding: 2rem 1rem;
	}

	.header {
		display: flex;
		justify-content: space-between;
		align-items: center;
		margin-bottom: 2rem;
		flex-wrap: wrap;
		gap: 1rem;
	}

	.header-left {
		display: flex;
		align-items: center;
		gap: 1rem;
	}

	.header-logo {
		height: 40px;
		width: auto;
	}

	h1 {
		font-size: 2rem;
		font-weight: 700;
		color: var(--text-primary);
		margin: 0;
	}

	h2 {
		font-size: 1.75rem;
		font-weight: 600;
		color: var(--text-primary);
		margin: 0;
	}

	h3 {
		font-size: 1.5rem;
		font-weight: 600;
		color: var(--text-primary);
		margin-bottom: 1rem;
	}

	h4 {
		font-size: 1.25rem;
		font-weight: 600;
		color: var(--text-primary);
		margin-bottom: 0.5rem;
	}

	.welcome-section {
		margin-bottom: 2rem;
	}

	.role-badge {
		margin-top: 0.5rem;
	}

	.badge {
		display: inline-block;
		padding: 0.375rem 0.75rem;
		border-radius: 0.375rem;
		font-size: 0.875rem;
		font-weight: 600;
	}

	.badge-assignor {
		background-color: #dbeafe;
		color: #1e40af;
	}

	.badge-referee {
		background-color: #d1fae5;
		color: #065f46;
	}

	.navigation-section {
		margin-bottom: 3rem;
	}

	.nav-grid {
		display: grid;
		grid-template-columns: repeat(auto-fill, minmax(200px, 1fr));
		gap: 1rem;
	}

	.nav-card {
		display: block;
		background: white;
		border: 2px solid var(--border-color);
		border-radius: 0.5rem;
		padding: 1.5rem;
		text-decoration: none;
		transition: all 0.2s;
		text-align: center;
	}

	.nav-card:hover {
		border-color: var(--primary-color);
		box-shadow: 0 4px 6px rgba(0, 0, 0, 0.1);
		transform: translateY(-2px);
	}

	.nav-icon {
		font-size: 2.5rem;
		margin-bottom: 0.75rem;
	}

	.nav-title {
		font-size: 1rem;
		font-weight: 600;
		color: var(--text-primary);
		margin-bottom: 0.5rem;
	}

	.nav-description {
		font-size: 0.875rem;
		color: var(--text-secondary);
	}

	.matches-section {
		margin-bottom: 2rem;
	}

	.match-group {
		margin-bottom: 2rem;
	}

	.group-description {
		color: var(--text-secondary);
		margin-bottom: 1rem;
		font-size: 0.875rem;
	}

	.match-list {
		display: grid;
		gap: 1rem;
	}

	.match-item {
		background: white;
		border: 2px solid var(--border-color);
		border-radius: 0.5rem;
		padding: 1rem;
		transition: all 0.2s;
	}

	.match-item:hover {
		box-shadow: 0 2px 4px rgba(0, 0, 0, 0.1);
	}

	.match-item.assigned {
		border-color: #3b82f6;
		background-color: #eff6ff;
	}

	.match-item.available {
		border-color: #10b981;
		background-color: #f0fdf4;
	}

	.match-item.unmarked {
		border-color: #f59e0b;
		background-color: #fffbeb;
	}

	.match-header {
		display: flex;
		align-items: center;
		gap: 0.75rem;
		margin-bottom: 0.75rem;
		flex-wrap: wrap;
	}

	.match-title {
		font-weight: 600;
		color: var(--text-primary);
		flex: 1;
		min-width: 200px;
	}

	.match-role {
		padding: 0.25rem 0.75rem;
		background-color: #3b82f6;
		color: white;
		border-radius: 0.375rem;
		font-size: 0.875rem;
		font-weight: 600;
	}

	.ack-status {
		padding: 0.25rem 0.5rem;
		border-radius: 0.375rem;
		font-size: 0.875rem;
		font-weight: 600;
	}

	.ack-status.confirmed {
		background-color: #d1fae5;
		color: #065f46;
	}

	.ack-status.pending {
		background-color: #fef3c7;
		color: #92400e;
	}

	.match-details {
		display: flex;
		gap: 1.5rem;
		flex-wrap: wrap;
		font-size: 0.875rem;
		color: var(--text-secondary);
		margin-bottom: 0.75rem;
	}

	.match-info {
		display: flex;
		gap: 1rem;
		align-items: center;
	}

	.age-badge {
		padding: 0.25rem 0.5rem;
		background-color: #3b82f6;
		color: white;
		border-radius: 0.375rem;
		font-size: 0.875rem;
		font-weight: 600;
	}

	.team-name {
		color: var(--text-secondary);
		font-size: 0.875rem;
	}

	.view-all-link {
		display: inline-block;
		margin-top: 0.75rem;
		color: var(--primary-color);
		text-decoration: none;
		font-weight: 500;
		font-size: 0.875rem;
	}

	.view-all-link:hover {
		text-decoration: underline;
	}

	.loading-text {
		color: var(--text-secondary);
		text-align: center;
		padding: 2rem;
	}

	.error-box {
		background-color: #fef2f2;
		border: 1px solid #fecaca;
		border-radius: 0.5rem;
		padding: 1rem;
		color: #991b1b;
	}

	.info-box {
		background-color: #f3f4f6;
		border: 1px solid #d1d5db;
		border-radius: 0.5rem;
		padding: 1.5rem;
		text-align: center;
		color: var(--text-secondary);
	}

	.btn-secondary {
		background-color: white;
		color: var(--text-primary);
		border: 1px solid var(--border-color);
		padding: 0.5rem 1rem;
		border-radius: 0.375rem;
		cursor: pointer;
		font-weight: 500;
		transition: all 0.2s;
	}

	.btn-secondary:hover {
		background-color: var(--bg-secondary);
	}

	@media (max-width: 768px) {
		.container {
			padding: 1rem 0.5rem;
		}

		h1 {
			font-size: 1.5rem;
		}

		.nav-grid {
			grid-template-columns: repeat(auto-fill, minmax(150px, 1fr));
		}

		.match-header {
			flex-direction: column;
			align-items: flex-start;
		}

		.match-details {
			flex-direction: column;
			gap: 0.5rem;
		}
	}
</style>
