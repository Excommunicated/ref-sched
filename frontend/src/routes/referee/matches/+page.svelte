<script lang="ts">
	import { onMount } from 'svelte';
	import type { PageData } from './$types';

	export let data: PageData;

	const API_URL = import.meta.env.VITE_API_URL || 'http://localhost:8080';

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
		eligible_roles: string[];
		is_available: boolean;
		is_assigned: boolean;
		assigned_role: string | null;
		acknowledged: boolean;
		acknowledged_at: string | null;
	}

	interface GroupedMatches {
		[date: string]: Match[];
	}

	let matches: Match[] = [];
	let loading = true;
	let error = '';
	let hasProfile = true;
	let dateFilter = '';
	let acknowledging = false;
	let unavailableDays: Set<string> = new Set();
	let togglingDayAvailability = false;

	onMount(() => {
		loadMatches();
		loadUnavailableDays();
	});

	async function loadMatches() {
		loading = true;
		error = '';

		try {
			const res = await fetch(`${API_URL}/api/referee/matches`, {
				credentials: 'include'
			});

			if (res.ok) {
				matches = await res.json();
				// If matches is empty, check if profile exists
				if (matches.length === 0) {
					const profileRes = await fetch(`${API_URL}/api/profile`, { credentials: 'include' });
					if (profileRes.ok) {
						const profile = await profileRes.json();
						hasProfile = !!(profile.first_name && profile.last_name && profile.date_of_birth);
					}
				}
			} else {
				error = 'Failed to load matches';
			}
		} catch (e) {
			error = 'Network error';
			console.error(e);
		} finally {
			loading = false;
		}
	}

	async function loadUnavailableDays() {
		try {
			const res = await fetch(`${API_URL}/api/referee/day-unavailability`, {
				credentials: 'include'
			});

			if (res.ok) {
				const days = await res.json();
				unavailableDays = new Set(days.map((d: any) => d.unavailable_date));
			}
		} catch (e) {
			console.error('Failed to load unavailable days:', e);
		}
	}

	async function toggleDayAvailability(date: string) {
		const isCurrentlyUnavailable = unavailableDays.has(date);
		const newState = !isCurrentlyUnavailable;

		if (newState) {
			// Marking as unavailable
			if (!confirm(`Mark ${formatDate(date)} as unavailable? This will clear any individual match availability for that day.`)) {
				return;
			}
		}

		togglingDayAvailability = true;

		try {
			const res = await fetch(`${API_URL}/api/referee/day-unavailability/${date}`, {
				method: 'POST',
				headers: { 'Content-Type': 'application/json' },
				credentials: 'include',
				body: JSON.stringify({ unavailable: newState })
			});

			if (res.ok) {
				if (newState) {
					unavailableDays.add(date);
				} else {
					unavailableDays.delete(date);
				}
				unavailableDays = unavailableDays; // Trigger reactivity
				await loadMatches(); // Reload matches to reflect the change
			} else {
				alert('Failed to update day availability');
			}
		} catch (e) {
			console.error(e);
			alert('Network error');
		} finally {
			togglingDayAvailability = false;
		}
	}

	async function toggleAvailability(match: Match) {
		const newAvailability = !match.is_available;

		try {
			const res = await fetch(`${API_URL}/api/referee/matches/${match.id}/availability`, {
				method: 'POST',
				headers: { 'Content-Type': 'application/json' },
				credentials: 'include',
				body: JSON.stringify({ available: newAvailability })
			});

			if (res.ok) {
				// Update local state
				match.is_available = newAvailability;
				matches = matches; // Trigger reactivity
			} else {
				alert('Failed to update availability');
			}
		} catch (e) {
			console.error(e);
			alert('Network error');
		}
	}

	async function acknowledgeAssignment(match: Match) {
		acknowledging = true;

		try {
			const res = await fetch(`${API_URL}/api/referee/matches/${match.id}/acknowledge`, {
				method: 'POST',
				headers: { 'Content-Type': 'application/json' },
				credentials: 'include'
			});

			if (res.ok) {
				// Update local state
				const data = await res.json();
				match.acknowledged = true;
				match.acknowledged_at = data.acknowledged_at;
				matches = matches; // Trigger reactivity
			} else {
				alert('Failed to acknowledge assignment');
			}
		} catch (e) {
			console.error(e);
			alert('Network error');
		} finally {
			acknowledging = false;
		}
	}

	function formatDate(dateString: string): string {
		const [year, month, day] = dateString.split('-').map(Number);
		const date = new Date(year, month - 1, day);
		return date.toLocaleDateString('en-US', {
			weekday: 'long',
			month: 'long',
			day: 'numeric',
			year: 'numeric'
		});
	}

	function formatShortDate(dateString: string): string {
		const [year, month, day] = dateString.split('-').map(Number);
		const date = new Date(year, month - 1, day);
		return date.toLocaleDateString('en-US', {
			weekday: 'short',
			month: 'short',
			day: 'numeric'
		});
	}

	function formatTime(timeString: string): string {
		// timeString is in format like "08:30:00" or "09:00"
		const parts = timeString.split(':');
		const hour = parseInt(parts[0]);
		const minute = parts[1];
		const ampm = hour >= 12 ? 'PM' : 'AM';
		const displayHour = hour === 0 ? 12 : hour > 12 ? hour - 12 : hour;
		return `${displayHour}:${minute} ${ampm}`;
	}

	function extractMeetingTime(description: string | null): string | null {
		if (!description) return null;

		// Look for patterns like "Meet: 8:30 AM" or "Meeting time: 9:00AM"
		const meetingPattern = /(?:meet(?:ing)?[:\s]+)(\d{1,2}:\d{2}\s*(?:AM|PM))/i;
		const match = description.match(meetingPattern);
		return match ? match[1] : null;
	}

	function extractField(description: string | null): string | null {
		if (!description) return null;

		// Look for patterns like "Field: 3" or "Field 5"
		const fieldPattern = /field[:\s]+(\w+)/i;
		const match = description.match(fieldPattern);
		return match ? `Field ${match[1]}` : null;
	}

	// Filter matches by date if filter is set
	$: filteredMatches = dateFilter
		? matches.filter(m => m.match_date.split('T')[0] === dateFilter)
		: matches;

	// Group matches by date, separate assigned from available
	$: groupedMatches = filteredMatches.reduce((acc: GroupedMatches, match: Match) => {
		if (!match.is_assigned) {
			const date = match.match_date;
			if (!acc[date]) {
				acc[date] = [];
			}
			acc[date].push(match);
		}
		return acc;
	}, {});

	$: assignedMatches = filteredMatches.filter(m => m.is_assigned);
	$: sortedDates = Object.keys(groupedMatches).sort();
</script>

<svelte:head>
	<title>My Matches - Referee Scheduler</title>
</svelte:head>

<div class="container">
	<div class="header">
		<div class="header-left">
			<img src="/logo.svg" alt="Logo" class="header-logo" />
			<h1>My Matches</h1>
		</div>
		<a href="/dashboard" class="btn-secondary">← Back to Dashboard</a>
	</div>

	{#if !loading && !error && hasProfile}
		<div class="filters">
			<div class="filter-group">
				<label for="dateFilter">Filter by Date</label>
				<input
					type="date"
					id="dateFilter"
					bind:value={dateFilter}
					placeholder="All dates"
				/>
			</div>
			{#if dateFilter}
				<button class="btn-clear" on:click={() => (dateFilter = '')}>Clear Filter</button>
			{/if}
		</div>
	{/if}

	{#if loading}
		<p>Loading matches...</p>
	{:else if error}
		<div class="error">
			<p>{error}</p>
		</div>
	{:else if !hasProfile}
		<div class="info-box">
			<h2>Complete Your Profile</h2>
			<p>You need to complete your profile before you can view available matches.</p>
			<a href="/referee/profile" class="btn-primary">Go to Profile</a>
		</div>
	{:else}
		<!-- Assigned Matches Section -->
		{#if assignedMatches.length > 0}
			<section class="assigned-section">
				<h2>My Assignments ({assignedMatches.length})</h2>
				<p class="section-description">Matches you've been assigned to</p>

				<div class="matches-grid">
					{#each assignedMatches as match}
						<div class="match-card assigned">
							<div class="match-header">
								<div class="match-title">
									<h3>{match.event_name}</h3>
									<span class="age-badge">{match.age_group}</span>
									<span class="role-badge assigned-badge">
										{#if match.assigned_role === 'center'}
											Center Referee
										{:else if match.assigned_role === 'assistant_1'}
											Assistant Referee 1
										{:else if match.assigned_role === 'assistant_2'}
											Assistant Referee 2
										{/if}
									</span>
								</div>
							</div>

							<div class="match-details">
								<div class="detail-row">
									<span class="icon">📅</span>
									<span>{formatShortDate(match.match_date)}</span>
								</div>
								<div class="detail-row">
									<span class="icon">🕐</span>
									<span>{formatTime(match.start_time)}</span>
									{#if extractMeetingTime(match.description)}
										<span class="meeting-time">
											(Meet: {extractMeetingTime(match.description)})
										</span>
									{/if}
								</div>
								<div class="detail-row">
									<span class="icon">📍</span>
									<span>{match.location}</span>
									{#if extractField(match.description)}
										<span class="field">• {extractField(match.description)}</span>
									{/if}
								</div>
								<div class="detail-row">
									<span class="icon">⚽</span>
									<span class="team-name">{match.team_name}</span>
								</div>
							</div>

							<!-- Acknowledgment section -->
							<div class="acknowledgment-section">
								{#if match.acknowledged}
									<div class="acknowledged-indicator">
										<span class="check-icon">✓</span>
										<span>Confirmed</span>
									</div>
								{:else}
									<button
										class="btn-acknowledge"
										on:click={() => acknowledgeAssignment(match)}
										disabled={acknowledging}
									>
										{acknowledging ? 'Acknowledging...' : 'Acknowledge Assignment'}
									</button>
								{/if}
							</div>
						</div>
					{/each}
				</div>
			</section>
		{/if}

		<!-- Available Matches Section -->
		<section class="available-section">
			<h2>Available Matches</h2>
			<p class="section-description">
				Mark your availability for upcoming matches you're eligible to referee
			</p>

			{#if sortedDates.length === 0}
				<div class="info-box">
					<p>No upcoming matches available at this time.</p>
					{#if assignedMatches.length === 0}
						<p>Check back later for new match assignments.</p>
					{/if}
				</div>
			{:else}
				{#each sortedDates as date}
					<div class="date-group">
						<div class="date-header-row">
							<h3 class="date-header">{formatDate(date)}</h3>
							<button
								class="btn-day-toggle"
								on:click={() => toggleDayAvailability(date)}
								disabled={togglingDayAvailability}
							>
								Mark Entire Day Unavailable
							</button>
						</div>

						<div class="matches-grid">
							{#each groupedMatches[date] as match}
								<div class="match-card" class:available={match.is_available}>
									<div class="match-header">
										<div class="match-title">
											<h4>{match.event_name}</h4>
											<span class="age-badge">{match.age_group}</span>
										</div>
										<button
											class="availability-toggle"
											class:active={match.is_available}
											on:click={() => toggleAvailability(match)}
										>
											{#if match.is_available}
												<span class="check">✓</span> Available
											{:else}
												Mark Available
											{/if}
										</button>
									</div>

									<div class="match-details">
										<div class="detail-row">
											<span class="icon">🕐</span>
											<span>{formatTime(match.start_time)}</span>
											{#if extractMeetingTime(match.description)}
												<span class="meeting-time">
													(Meet: {extractMeetingTime(match.description)})
												</span>
											{/if}
										</div>
										<div class="detail-row">
											<span class="icon">📍</span>
											<span>{match.location}</span>
											{#if extractField(match.description)}
												<span class="field">• {extractField(match.description)}</span>
											{/if}
										</div>
										<div class="detail-row">
											<span class="icon">⚽</span>
											<span class="team-name">{match.team_name}</span>
										</div>
										<div class="detail-row eligible-roles">
											<span class="icon">✓</span>
											<span class="small-text">
												Eligible for:
												{#if match.eligible_roles.includes('center')}
													Center Referee
												{/if}
												{#if match.eligible_roles.includes('center') && match.eligible_roles.includes('assistant')}
													,
												{/if}
												{#if match.eligible_roles.includes('assistant')}
													Assistant Referee
												{/if}
											</span>
										</div>
									</div>
								</div>
							{/each}
						</div>
					</div>
				{/each}
			{/if}
		</section>
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

	.filters {
		display: flex;
		gap: 1rem;
		align-items: flex-end;
		margin-bottom: 2rem;
		flex-wrap: wrap;
	}

	.filter-group {
		flex: 1;
		min-width: 200px;
	}

	.filter-group label {
		display: block;
		margin-bottom: 0.5rem;
		font-weight: 500;
		color: #374151;
	}

	.filter-group input[type='date'] {
		width: 100%;
		padding: 0.75rem;
		border: 1px solid #d1d5db;
		border-radius: 0.375rem;
		font-size: 1rem;
		font-family: inherit;
	}

	.btn-clear {
		padding: 0.75rem 1.5rem;
		background-color: #6b7280;
		color: white;
		border: none;
		border-radius: 0.375rem;
		cursor: pointer;
		font-weight: 500;
		transition: all 0.2s;
	}

	.btn-clear:hover {
		background-color: #4b5563;
	}

	h1 {
		margin: 0;
		font-size: 2rem;
	}

	h2 {
		font-size: 1.5rem;
		margin-bottom: 0.5rem;
	}

	.section-description {
		color: #666;
		margin-bottom: 1.5rem;
	}

	.assigned-section,
	.available-section {
		margin-bottom: 3rem;
	}

	.date-group {
		margin-bottom: 2rem;
	}

	.date-header-row {
		display: flex;
		justify-content: space-between;
		align-items: center;
		margin-bottom: 1rem;
		padding-bottom: 0.5rem;
		border-bottom: 2px solid #e2e8f0;
		gap: 1rem;
		flex-wrap: wrap;
	}

	.date-header {
		font-size: 1.25rem;
		font-weight: 600;
		color: #2c3e50;
		margin: 0;
	}

	.btn-day-toggle {
		padding: 0.5rem 1rem;
		background-color: #f3f4f6;
		color: #374151;
		border: 1px solid #d1d5db;
		border-radius: 0.375rem;
		font-size: 0.875rem;
		font-weight: 500;
		cursor: pointer;
		transition: all 0.2s;
		white-space: nowrap;
	}

	.btn-day-toggle:hover:not(:disabled) {
		background-color: #e5e7eb;
		border-color: #9ca3af;
	}

	.btn-day-toggle:disabled {
		opacity: 0.5;
		cursor: not-allowed;
	}

	.matches-grid {
		display: grid;
		grid-template-columns: repeat(auto-fill, minmax(320px, 1fr));
		gap: 1rem;
	}

	.match-card {
		background: white;
		border: 2px solid #e2e8f0;
		border-radius: 8px;
		padding: 1rem;
		transition: all 0.2s;
	}

	.match-card.available {
		border-color: #10b981;
		background-color: #f0fdf4;
	}

	.match-card.assigned {
		border-color: #3b82f6;
		background-color: #eff6ff;
	}

	.match-header {
		display: flex;
		justify-content: space-between;
		align-items: flex-start;
		margin-bottom: 1rem;
		gap: 1rem;
	}

	.match-title {
		flex: 1;
		min-width: 0;
	}

	.match-title h3,
	.match-title h4 {
		margin: 0 0 0.5rem 0;
		font-size: 1.1rem;
		overflow-wrap: break-word;
	}

	.age-badge {
		display: inline-block;
		background: #3b82f6;
		color: white;
		padding: 0.25rem 0.5rem;
		border-radius: 4px;
		font-size: 0.85rem;
		font-weight: 600;
		margin-right: 0.5rem;
	}

	.role-badge {
		display: inline-block;
		padding: 0.25rem 0.5rem;
		border-radius: 4px;
		font-size: 0.85rem;
		font-weight: 600;
	}

	.assigned-badge {
		background: #3b82f6;
		color: white;
	}

	.availability-toggle {
		background: white;
		border: 2px solid #cbd5e1;
		border-radius: 6px;
		padding: 0.5rem 1rem;
		font-size: 0.9rem;
		font-weight: 600;
		cursor: pointer;
		transition: all 0.2s;
		white-space: nowrap;
	}

	.availability-toggle:hover {
		border-color: #10b981;
		color: #10b981;
	}

	.availability-toggle.active {
		background: #10b981;
		color: white;
		border-color: #10b981;
	}

	.availability-toggle .check {
		font-size: 1rem;
	}

	.match-details {
		display: flex;
		flex-direction: column;
		gap: 0.5rem;
	}

	.detail-row {
		display: flex;
		align-items: center;
		gap: 0.5rem;
		font-size: 0.95rem;
		flex-wrap: wrap;
	}

	.detail-row .icon {
		font-size: 1rem;
		min-width: 1.5rem;
	}

	.meeting-time {
		color: #059669;
		font-weight: 500;
		font-size: 0.9rem;
	}

	.field {
		color: #666;
		font-size: 0.9rem;
	}

	.team-name {
		color: #374151;
		font-weight: 500;
	}

	.eligible-roles {
		margin-top: 0.25rem;
		padding-top: 0.5rem;
		border-top: 1px solid #e2e8f0;
	}

	.small-text {
		font-size: 0.85rem;
		color: #6b7280;
	}

	.info-box {
		background: #f3f4f6;
		border: 1px solid #d1d5db;
		border-radius: 8px;
		padding: 1.5rem;
		text-align: center;
	}

	.info-box h2 {
		margin-top: 0;
	}

	.error {
		background: #fef2f2;
		border: 1px solid #fecaca;
		border-radius: 8px;
		padding: 1rem;
		color: #991b1b;
	}

	.btn-primary,
	.btn-secondary {
		display: inline-block;
		padding: 0.5rem 1rem;
		border-radius: 6px;
		text-decoration: none;
		font-weight: 500;
		cursor: pointer;
		transition: all 0.2s;
	}

	.btn-primary {
		background: #3b82f6;
		color: white;
		border: none;
	}

	.btn-primary:hover {
		background: #2563eb;
	}

	.btn-secondary {
		background: white;
		color: #3b82f6;
		border: 2px solid #3b82f6;
	}

	.btn-secondary:hover {
		background: #eff6ff;
	}

	.acknowledgment-section {
		margin-top: 1rem;
		padding-top: 1rem;
		border-top: 1px solid #e2e8f0;
	}

	.btn-acknowledge {
		width: 100%;
		padding: 0.75rem 1rem;
		background: #3b82f6;
		color: white;
		border: none;
		border-radius: 6px;
		font-weight: 600;
		cursor: pointer;
		transition: all 0.2s;
	}

	.btn-acknowledge:hover:not(:disabled) {
		background: #2563eb;
	}

	.btn-acknowledge:disabled {
		opacity: 0.6;
		cursor: not-allowed;
	}

	.acknowledged-indicator {
		display: flex;
		align-items: center;
		justify-content: center;
		gap: 0.5rem;
		padding: 0.75rem;
		background: #d1fae5;
		color: #065f46;
		border-radius: 6px;
		font-weight: 600;
	}

	.check-icon {
		font-size: 1.25rem;
		font-weight: bold;
	}

	@media (max-width: 768px) {
		.container {
			padding: 1rem 0.5rem;
		}

		h1 {
			font-size: 1.5rem;
		}

		.matches-grid {
			grid-template-columns: 1fr;
		}

		.header {
			flex-direction: column;
			align-items: flex-start;
		}
	}
</style>
