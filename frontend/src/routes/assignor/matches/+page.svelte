<script lang="ts">
	import { onMount } from 'svelte';
	import { goto } from '$app/navigation';

	const API_URL = import.meta.env.VITE_API_URL || 'http://localhost:8080';

	let loading = true;
	let matches: any[] = [];
	let filteredMatches: any[] = [];
	let error = '';

	// Filters
	let ageGroupFilter = 'all';
	let assignmentStatusFilter = 'all';
	let showCancelled = false;
	let dateFilter = '';

	// Edit modal state
	let editingMatch: any = null;
	let editForm = {
		event_name: '',
		team_name: '',
		age_group: '',
		match_date: '',
		start_time: '',
		end_time: '',
		location: '',
		description: ''
	};
	let saving = false;
	let editError = '';

	// Assignment panel state
	let assignmentMatch: any = null;
	let selectedRole: string | null = null;
	let eligibleReferees: any[] = [];
	let loadingReferees = false;
	let assignmentError = '';
	let assigning = false;

	onMount(async () => {
		await loadMatches();
	});

	async function loadMatches() {
		loading = true;
		error = '';

		try {
			const response = await fetch(`${API_URL}/api/matches`, {
				credentials: 'include'
			});

			if (response.ok) {
				matches = await response.json();
				filterMatches();
			} else {
				error = 'Failed to load matches';
			}
		} catch (err) {
			error = 'Failed to load matches';
		} finally {
			loading = false;
		}
	}

	function filterMatches() {
		let filtered = matches;

		// Filter by cancelled status
		if (!showCancelled) {
			filtered = filtered.filter((m) => m.status !== 'cancelled');
		}

		// Filter by age group
		if (ageGroupFilter !== 'all') {
			filtered = filtered.filter((m) => m.age_group === ageGroupFilter);
		}

		// Filter by assignment status
		if (assignmentStatusFilter !== 'all') {
			filtered = filtered.filter((m) => m.assignment_status === assignmentStatusFilter);
		}

		// Filter by date
		if (dateFilter) {
			filtered = filtered.filter((m) => {
				const matchDate = m.match_date.split('T')[0];
				return matchDate === dateFilter;
			});
		}

		filteredMatches = filtered;
	}

	function getStatusBadge(status: string) {
		const badges = {
			unassigned: { class: 'badge-error', text: 'Unassigned' },
			partial: { class: 'badge-warning', text: 'Partial' },
			full: { class: 'badge-success', text: 'Full' }
		};
		return badges[status] || badges.unassigned;
	}

	function formatDate(dateString: string) {
		// Parse date as Eastern Time (dates are stored without timezone, assume Eastern)
		// Split on 'T' to get just the date part (YYYY-MM-DD) to avoid timezone conversion
		const datePart = dateString.split('T')[0];
		const [year, month, day] = datePart.split('-').map(Number);
		// Create date in local timezone to avoid UTC conversion issues
		const date = new Date(year, month - 1, day);
		return date.toLocaleDateString('en-US', {
			weekday: 'short',
			month: 'short',
			day: 'numeric'
		});
	}

	function formatTime(timeString: string) {
		// Time comes as HH:MM:SS from backend
		const parts = timeString.split(':');
		const hours = parseInt(parts[0]);
		const minutes = parts[1];
		const ampm = hours >= 12 ? 'PM' : 'AM';
		const displayHours = hours % 12 || 12;
		return `${displayHours}:${minutes} ${ampm}`;
	}

	function openEditModal(match: any) {
		editingMatch = match;
		editForm = {
			event_name: match.event_name || '',
			team_name: match.team_name || '',
			age_group: match.age_group || '',
			match_date: match.match_date ? match.match_date.split('T')[0] : '',
			start_time: match.start_time || '',
			end_time: match.end_time || '',
			location: match.location || '',
			description: match.description || ''
		};
		editError = '';
	}

	function closeEditModal() {
		editingMatch = null;
		editError = '';
	}

	async function saveMatchEdit() {
		if (!editingMatch) return;

		saving = true;
		editError = '';

		try {
			const response = await fetch(`${API_URL}/api/matches/${editingMatch.id}`, {
				method: 'PUT',
				credentials: 'include',
				headers: {
					'Content-Type': 'application/json'
				},
				body: JSON.stringify(editForm)
			});

			if (response.ok) {
				await loadMatches();
				closeEditModal();
			} else {
				const text = await response.text();
				editError = text || 'Failed to update match';
			}
		} catch (err) {
			editError = 'Failed to update match';
		} finally {
			saving = false;
		}
	}

	async function toggleCancelMatch(match: any) {
		const newStatus = match.status === 'cancelled' ? 'active' : 'cancelled';
		const confirmMessage =
			newStatus === 'cancelled'
				? 'Are you sure you want to cancel this match?'
				: 'Are you sure you want to un-cancel this match?';

		if (!confirm(confirmMessage)) {
			return;
		}

		try {
			const response = await fetch(`${API_URL}/api/matches/${match.id}`, {
				method: 'PUT',
				credentials: 'include',
				headers: {
					'Content-Type': 'application/json'
				},
				body: JSON.stringify({ status: newStatus })
			});

			if (response.ok) {
				await loadMatches();
			} else {
				const text = await response.text();
				alert(`Failed to update match: ${text}`);
			}
		} catch (err) {
			alert('Failed to update match');
		}
	}

	function openAssignmentPanel(match: any) {
		assignmentMatch = match;
		selectedRole = null;
		eligibleReferees = [];
		assignmentError = '';
	}

	function closeAssignmentPanel() {
		assignmentMatch = null;
		selectedRole = null;
		eligibleReferees = [];
		assignmentError = '';
	}

	async function selectRole(roleType: string) {
		selectedRole = roleType;
		loadingReferees = true;
		assignmentError = '';
		eligibleReferees = [];

		try {
			const response = await fetch(
				`${API_URL}/api/matches/${assignmentMatch.id}/eligible-referees?role=${roleType}`,
				{ credentials: 'include' }
			);

			if (response.ok) {
				const allReferees = await response.json();
				eligibleReferees = allReferees;
			} else {
				assignmentError = 'Failed to load eligible referees';
			}
		} catch (err) {
			assignmentError = 'Failed to load eligible referees';
		} finally {
			loadingReferees = false;
		}
	}

	function closeRolePicker() {
		selectedRole = null;
		eligibleReferees = [];
	}

	async function assignReferee(refereeId: number, refereeName: string) {
		// Check for conflicts first
		const hasConflict = await checkConflict(refereeId);

		if (hasConflict) {
			const confirmed = confirm(
				`${refereeName} is already assigned to another match at this time. Assign anyway?`
			);
			if (!confirmed) {
				return;
			}
		}

		assigning = true;
		assignmentError = '';

		try {
			const response = await fetch(
				`${API_URL}/api/matches/${assignmentMatch.id}/roles/${selectedRole}/assign`,
				{
					method: 'POST',
					credentials: 'include',
					headers: { 'Content-Type': 'application/json' },
					body: JSON.stringify({ referee_id: refereeId })
				}
			);

			if (response.ok) {
				await loadMatches();
				// Update assignmentMatch with the refreshed data
				const refreshedMatch = matches.find(m => m.id === assignmentMatch.id);
				if (refreshedMatch) {
					assignmentMatch = refreshedMatch;
				}
				// Go back to role selection instead of closing the panel
				selectedRole = null;
				eligibleReferees = [];
				assignmentError = '';
			} else {
				const text = await response.text();
				assignmentError = text || 'Failed to assign referee';
			}
		} catch (err) {
			assignmentError = 'Failed to assign referee';
		} finally {
			assigning = false;
		}
	}

	async function removeAssignment(roleType: string) {
		const confirmed = confirm('Remove this assignment?');
		if (!confirmed) return;

		assigning = true;
		assignmentError = '';

		try {
			const response = await fetch(
				`${API_URL}/api/matches/${assignmentMatch.id}/roles/${roleType}/assign`,
				{
					method: 'POST',
					credentials: 'include',
					headers: { 'Content-Type': 'application/json' },
					body: JSON.stringify({ referee_id: null })
				}
			);

			if (response.ok) {
				await loadMatches();
				// Update the local assignmentMatch to reflect the change
				if (assignmentMatch.roles) {
					const role = assignmentMatch.roles.find((r: any) => r.role_type === roleType);
					if (role) {
						role.assigned_referee_id = null;
						role.assigned_referee_name = null;
					}
				}
				assignmentMatch = assignmentMatch; // Trigger reactivity
			} else {
				const text = await response.text();
				assignmentError = text || 'Failed to remove assignment';
			}
		} catch (err) {
			assignmentError = 'Failed to remove assignment';
		} finally {
			assigning = false;
		}
	}

	async function checkConflict(refereeId: number): Promise<boolean> {
		try {
			const response = await fetch(
				`${API_URL}/api/matches/${assignmentMatch.id}/conflicts?referee_id=${refereeId}`,
				{ credentials: 'include' }
			);

			if (response.ok) {
				const data = await response.json();
				return data.has_conflict;
			}
		} catch (err) {
			console.error('Failed to check conflicts', err);
		}
		return false;
	}

	function getRoleName(roleType: string): string {
		const names: Record<string, string> = {
			center: 'Center Referee',
			assistant_1: 'Assistant Referee 1',
			assistant_2: 'Assistant Referee 2'
		};
		return names[roleType] || roleType;
	}

	function getRoleShortName(roleType: string): string {
		const names: Record<string, string> = {
			center: 'CR',
			assistant_1: 'AR1',
			assistant_2: 'AR2'
		};
		return names[roleType] || roleType;
	}

	function countAvailableReferees(match: any): number {
		// This is a placeholder - would need to fetch from backend
		// For now, just return 0
		return 0;
	}

	// Separate referees into available and not available
	$: availableReferees = eligibleReferees.filter((r: any) => r.is_eligible);
	$: unavailableReferees = eligibleReferees.filter((r: any) => !r.is_eligible);

	// Sort roles to display Center Referee first, then AR1, then AR2
	$: sortedRoles = assignmentMatch?.roles ? [...assignmentMatch.roles].sort((a: any, b: any) => {
		const order: Record<string, number> = { center: 1, assistant_1: 2, assistant_2: 3 };
		return (order[a.role_type] || 99) - (order[b.role_type] || 99);
	}) : [];

	// Get unique age groups for filter
	$: uniqueAgeGroups = [...new Set(matches.map((m) => m.age_group).filter(Boolean))].sort();

	$: {
		ageGroupFilter;
		assignmentStatusFilter;
		showCancelled;
		dateFilter;
		filterMatches();
	}
</script>

<svelte:head>
	<title>Match Schedule - Referee Scheduler</title>
</svelte:head>

<div class="container">
	<div class="header">
		<div class="header-left">
			<img src="/logo.svg" alt="Logo" class="header-logo" />
			<h1>Match Schedule</h1>
		</div>
		<div class="header-actions">
			<button on:click={() => goto('/assignor/matches/import')} class="btn btn-primary">
				Import Matches
			</button>
			<button on:click={() => goto('/dashboard')} class="btn btn-secondary">
				Back to Dashboard
			</button>
		</div>
	</div>

	{#if error}
		<div class="alert alert-error">{error}</div>
	{/if}

	<div class="filters card">
		<div class="filter-group">
			<label for="ageGroup">Age Group</label>
			<select id="ageGroup" bind:value={ageGroupFilter}>
				<option value="all">All Age Groups</option>
				{#each uniqueAgeGroups as ageGroup}
					<option value={ageGroup}>{ageGroup}</option>
				{/each}
			</select>
		</div>

		<div class="filter-group">
			<label for="status">Assignment Status</label>
			<select id="status" bind:value={assignmentStatusFilter}>
				<option value="all">All Statuses</option>
				<option value="unassigned">Unassigned</option>
				<option value="partial">Partial</option>
				<option value="full">Full</option>
			</select>
		</div>

		<div class="filter-group">
			<label for="dateFilter">Match Date</label>
			<input
				type="date"
				id="dateFilter"
				bind:value={dateFilter}
				placeholder="Filter by date"
			/>
		</div>

		<div class="filter-group checkbox-filter">
			<label class="checkbox-label">
				<input type="checkbox" bind:checked={showCancelled} />
				<span>Show cancelled</span>
			</label>
		</div>

		<div class="stats">
			<strong>{filteredMatches.length}</strong> match{filteredMatches.length !== 1 ? 'es' : ''} shown
		</div>
	</div>

	{#if loading}
		<div class="card">
			<p>Loading matches...</p>
		</div>
	{:else if filteredMatches.length === 0}
		<div class="card">
			<p>No matches found. {matches.length === 0 ? 'Import a CSV file to get started.' : 'Try adjusting your filters.'}</p>
			{#if matches.length === 0}
				<div class="empty-actions">
					<button on:click={() => goto('/assignor/matches/import')} class="btn btn-primary">
						Import Match Schedule
					</button>
				</div>
			{/if}
		</div>
	{:else}
		<div class="matches-grid">
			{#each filteredMatches as match}
				<div class="match-card card" class:cancelled={match.status === 'cancelled'}>
					<div class="match-header">
						<div class="match-date-time">
							<div class="date">{formatDate(match.match_date)}</div>
							<div class="time">{formatTime(match.start_time)}</div>
						</div>
						<div class="match-info">
							<div class="event-name">{match.event_name}</div>
							<div class="team-name">{match.team_name}</div>
						</div>
						<div class="match-meta">
							{#if match.age_group}
								<span class="age-badge">{match.age_group}</span>
							{/if}
							<span class="badge {getStatusBadge(match.assignment_status).class}">
								{getStatusBadge(match.assignment_status).text}
							</span>
							{#if match.status === 'cancelled'}
								<span class="badge badge-cancelled">Cancelled</span>
							{/if}
							{#if match.has_overdue_ack}
								<span class="badge badge-overdue">⚠ Needs Acknowledgment</span>
							{/if}
						</div>
					</div>

					<div class="match-details">
						<div class="detail-item">
							<span class="detail-label">Location:</span>
							<span class="detail-value">{match.location}</span>
						</div>
						{#if match.description}
							<div class="detail-item">
								<span class="detail-label">Details:</span>
								<span class="detail-value">{match.description}</span>
							</div>
						{/if}
					</div>

					{#if match.roles && match.roles.length > 0}
						{@const sortedMatchRoles = [...match.roles].sort((a, b) => {
							const order = { center: 1, assistant_1: 2, assistant_2: 3 };
							return (order[a.role_type] || 99) - (order[b.role_type] || 99);
						})}
						<div class="roles">
							<div class="roles-label">Assignments:</div>
							<div class="roles-list">
								{#each sortedMatchRoles as role}
									<div class="role-item">
										<span class="role-type">
											{#if role.role_type === 'center'}
												CR
											{:else if role.role_type === 'assistant_1'}
												AR1
											{:else if role.role_type === 'assistant_2'}
												AR2
											{/if}:
										</span>
										<span class="role-referee" class:acknowledged={role.assigned_referee_name && role.acknowledged}>
											{role.assigned_referee_name || '—'}
											{#if role.assigned_referee_name && role.acknowledged}
												<span class="ack-check">✓</span>
											{/if}
										</span>
									</div>
								{/each}
							</div>
						</div>
					{/if}

					<div class="match-actions">
						<button
							class="btn-small btn-primary"
							on:click={() => openAssignmentPanel(match)}
							disabled={match.status === 'cancelled'}
						>
							Assign Referees
						</button>
						<button class="btn-small btn-secondary" on:click={() => openEditModal(match)}>
							Edit Match
						</button>
						{#if match.status === 'cancelled'}
							<button class="btn-small btn-success" on:click={() => toggleCancelMatch(match)}>
								Un-cancel
							</button>
						{:else}
							<button class="btn-small btn-warning" on:click={() => toggleCancelMatch(match)}>
								Cancel Match
							</button>
						{/if}
					</div>
				</div>
			{/each}
		</div>
	{/if}
</div>

	{#if assignmentMatch}
		<div class="modal-overlay" on:click={closeAssignmentPanel}>
			<div class="modal assignment-modal" on:click|stopPropagation>
				<div class="modal-header">
					<div>
						<h2>Assign Referees</h2>
						<div class="match-subtitle">
							{assignmentMatch.event_name} • {assignmentMatch.team_name}
						</div>
						<div class="match-details-small">
							{formatDate(assignmentMatch.match_date)} • {formatTime(assignmentMatch.start_time)}
							• {assignmentMatch.location}
						</div>
					</div>
					<button class="close-btn" on:click={closeAssignmentPanel}>&times;</button>
				</div>

				{#if assignmentError}
					<div class="alert alert-error">{assignmentError}</div>
				{/if}

				<div class="assignment-content">
					{#if !selectedRole}
						<!-- Role selection view -->
						<div class="roles-grid">
							{#if sortedRoles.length > 0}
								{#each sortedRoles as role}
									<div class="role-card">
										<div class="role-card-header">
											<h3>{getRoleName(role.role_type)}</h3>
											{#if role.assigned_referee_name}
												<span class="assigned-badge">Assigned</span>
											{:else}
												<span class="unassigned-badge">Open</span>
											{/if}
										</div>

										<div class="role-card-body">
											{#if role.assigned_referee_name}
												<div class="current-assignment">
													<div class="referee-info-row">
														<div class="referee-name">{role.assigned_referee_name}</div>
														{#if role.acknowledged}
															<span class="ack-badge ack-confirmed">✓ Confirmed</span>
														{:else if role.ack_overdue}
															<span class="ack-badge ack-overdue">⚠ Overdue</span>
														{:else}
															<span class="ack-badge ack-pending">Pending</span>
														{/if}
													</div>
													<div class="assignment-actions">
														<button
															class="btn-small btn-secondary"
															on:click={() => selectRole(role.role_type)}
															disabled={assigning}
														>
															Change
														</button>
														<button
															class="btn-small btn-warning"
															on:click={() => removeAssignment(role.role_type)}
															disabled={assigning}
														>
															Remove
														</button>
													</div>
												</div>
											{:else}
												<button
													class="btn btn-primary full-width"
													on:click={() => selectRole(role.role_type)}
													disabled={assigning}
												>
													Select Referee
												</button>
											{/if}
										</div>
									</div>
								{/each}
							{/if}
						</div>
					{:else}
						<!-- Referee picker view -->
						<div class="picker-header">
							<button class="btn-back" on:click={closeRolePicker}>← Back</button>
							<h3>Select {getRoleName(selectedRole)}</h3>
						</div>

						{#if loadingReferees}
							<div class="loading">Loading referees...</div>
						{:else if eligibleReferees.length === 0}
							<div class="empty-state">
								<p>No eligible referees found for this role.</p>
							</div>
						{:else}
							<div class="referee-list">
								<!-- Eligible referees -->
								{#if availableReferees.length > 0}
									<div class="referee-section">
										<h4 class="section-title">Eligible Referees ({availableReferees.length})</h4>
										{#each availableReferees as referee}
											<button
												class="referee-item"
												class:has-availability={referee.is_available}
												on:click={() => assignReferee(referee.id, `${referee.first_name} ${referee.last_name}`)}
												disabled={assigning}
											>
												<div class="referee-info">
													<div class="referee-name-row">
														{#if referee.is_available}
															<span class="availability-badge" title="Marked as available">★</span>
														{/if}
														<span class="referee-name">
															{referee.first_name}
															{referee.last_name}
														</span>
														{#if referee.grade}
															<span class="grade-badge">{referee.grade}</span>
														{/if}
													</div>
													<div class="referee-details">
														{#if referee.age_at_match}
															<span>Age: {referee.age_at_match}</span>
														{/if}
														{#if referee.certified && selectedRole === 'center'}
															<span class="cert-badge">Certified</span>
														{/if}
														{#if referee.is_available}
															<span class="available-indicator">Available</span>
														{/if}
													</div>
												</div>
												<div class="referee-action">
													<span class="assign-icon">→</span>
												</div>
											</button>
										{/each}
									</div>
								{/if}

								<!-- Ineligible referees -->
								{#if unavailableReferees.length > 0}
									<div class="referee-section">
										<h4 class="section-title ineligible">
											Ineligible Referees ({unavailableReferees.length})
										</h4>
										{#each unavailableReferees as referee}
											<div class="referee-item ineligible">
												<div class="referee-info">
													<div class="referee-name-row">
														<span class="referee-name">
															{referee.first_name}
															{referee.last_name}
														</span>
														{#if referee.grade}
															<span class="grade-badge">{referee.grade}</span>
														{/if}
													</div>
													<div class="referee-details">
														{#if referee.age_at_match}
															<span>Age: {referee.age_at_match}</span>
														{/if}
														{#if referee.ineligible_reason}
															<span class="ineligible-reason">
																{referee.ineligible_reason}
															</span>
														{/if}
													</div>
												</div>
											</div>
										{/each}
									</div>
								{/if}
							</div>
						{/if}
					{/if}
				</div>
			</div>
		</div>
	{/if}

{#if editingMatch}
	<div class="modal-overlay" on:click={closeEditModal}>
		<div class="modal" on:click|stopPropagation>
			<div class="modal-header">
				<h2>Edit Match</h2>
				<button class="close-btn" on:click={closeEditModal}>&times;</button>
			</div>

			{#if editError}
				<div class="alert alert-error">{editError}</div>
			{/if}

			<form on:submit|preventDefault={saveMatchEdit}>
				<div class="form-row">
					<div class="form-group">
						<label for="event_name">Event Name *</label>
						<input
							type="text"
							id="event_name"
							bind:value={editForm.event_name}
							required
						/>
					</div>

					<div class="form-group">
						<label for="team_name">Team Name *</label>
						<input
							type="text"
							id="team_name"
							bind:value={editForm.team_name}
							required
						/>
					</div>
				</div>

				<div class="form-row">
					<div class="form-group">
						<label for="age_group">Age Group *</label>
						<select id="age_group" bind:value={editForm.age_group} required>
							<option value="">Select age group</option>
							<option value="U6">U6</option>
							<option value="U8">U8</option>
							<option value="U10">U10</option>
							<option value="U12">U12</option>
							<option value="U14">U14</option>
							<option value="U16">U16</option>
							<option value="U18">U18</option>
						</select>
						<small class="warning-text">
							⚠️ Changing age group will reconfigure role slots
						</small>
					</div>

					<div class="form-group">
						<label for="match_date">Date *</label>
						<input
							type="date"
							id="match_date"
							bind:value={editForm.match_date}
							required
						/>
					</div>
				</div>

				<div class="form-row">
					<div class="form-group">
						<label for="start_time">Start Time *</label>
						<input
							type="time"
							id="start_time"
							bind:value={editForm.start_time}
							required
						/>
					</div>

					<div class="form-group">
						<label for="end_time">End Time *</label>
						<input
							type="time"
							id="end_time"
							bind:value={editForm.end_time}
							required
						/>
					</div>
				</div>

				<div class="form-group">
					<label for="location">Location *</label>
					<input
						type="text"
						id="location"
						bind:value={editForm.location}
						required
					/>
				</div>

				<div class="form-group">
					<label for="description">Description / Field Info</label>
					<textarea
						id="description"
						bind:value={editForm.description}
						rows="3"
						placeholder="e.g., Field 3 - Meet at 08:45"
					></textarea>
				</div>

				<div class="modal-actions">
					<button type="button" class="btn btn-secondary" on:click={closeEditModal}>
						Cancel
					</button>
					<button type="submit" class="btn btn-primary" disabled={saving}>
						{saving ? 'Saving...' : 'Save Changes'}
					</button>
				</div>
			</form>
		</div>
	</div>
{/if}

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

	.header-actions {
		display: flex;
		gap: 1rem;
		flex-wrap: wrap;
	}

	h1 {
		font-size: 2rem;
		font-weight: 700;
		color: var(--text-primary);
	}

	.filters {
		display: flex;
		gap: 1.5rem;
		align-items: flex-end;
		flex-wrap: wrap;
		margin-bottom: 1.5rem;
	}

	.filter-group {
		flex: 1;
		min-width: 200px;
	}

	.filter-group label {
		display: block;
		margin-bottom: 0.5rem;
		font-weight: 500;
		color: var(--text-primary);
	}

	.filter-group select,
	.filter-group input[type='date'] {
		width: 100%;
		padding: 0.75rem;
		border: 1px solid var(--border-color);
		border-radius: 0.375rem;
		font-size: 1rem;
		font-family: inherit;
	}

	.checkbox-filter {
		display: flex;
		align-items: center;
		padding-top: 1.75rem;
	}

	.checkbox-label {
		display: flex;
		align-items: center;
		gap: 0.5rem;
		cursor: pointer;
		margin: 0;
	}

	.checkbox-label input[type='checkbox'] {
		cursor: pointer;
	}

	.checkbox-label span {
		font-weight: 500;
		color: var(--text-primary);
	}

	.stats {
		color: var(--text-secondary);
		padding: 0.75rem 0;
	}

	.matches-grid {
		display: grid;
		grid-template-columns: repeat(auto-fill, minmax(450px, 1fr));
		gap: 1rem;
	}

	.match-card {
		transition: all 0.2s;
	}

	.match-card:hover {
		box-shadow: 0 4px 6px -1px rgba(0, 0, 0, 0.1), 0 2px 4px -1px rgba(0, 0, 0, 0.06);
	}

	.match-card.cancelled {
		opacity: 0.6;
		background-color: #fafafa;
	}

	.match-header {
		display: flex;
		gap: 1.5rem;
		align-items: flex-start;
		margin-bottom: 1rem;
		padding-bottom: 1rem;
		border-bottom: 1px solid var(--border-color);
	}

	.match-date-time {
		flex-shrink: 0;
		text-align: center;
		min-width: 100px;
	}

	.date {
		font-size: 0.875rem;
		font-weight: 600;
		color: var(--text-primary);
		margin-bottom: 0.25rem;
	}

	.time {
		font-size: 1.25rem;
		font-weight: 700;
		color: var(--primary-color);
	}

	.match-info {
		flex: 1;
	}

	.event-name {
		font-size: 1.125rem;
		font-weight: 600;
		color: var(--text-primary);
		margin-bottom: 0.25rem;
	}

	.team-name {
		color: var(--text-secondary);
		font-size: 0.875rem;
	}

	.match-meta {
		display: flex;
		gap: 0.5rem;
		flex-wrap: wrap;
		align-items: flex-start;
	}

	.age-badge {
		display: inline-block;
		padding: 0.25rem 0.75rem;
		background-color: #dbeafe;
		color: #1e40af;
		border-radius: 0.25rem;
		font-size: 0.875rem;
		font-weight: 600;
	}

	.badge {
		display: inline-block;
		padding: 0.25rem 0.75rem;
		border-radius: 0.25rem;
		font-size: 0.875rem;
		font-weight: 500;
	}

	.badge-success {
		background-color: #d1fae5;
		color: #065f46;
	}

	.badge-warning {
		background-color: #fef3c7;
		color: #92400e;
	}

	.badge-error {
		background-color: #fee2e2;
		color: #991b1b;
	}

	.badge-cancelled {
		background-color: #f3f4f6;
		color: #6b7280;
	}

	.badge-overdue {
		background-color: #fecaca;
		color: #991b1b;
		font-weight: 600;
	}

	.match-details {
		margin-bottom: 1rem;
	}

	.detail-item {
		display: flex;
		gap: 0.5rem;
		margin-bottom: 0.5rem;
		font-size: 0.875rem;
	}

	.detail-label {
		font-weight: 500;
		color: var(--text-secondary);
	}

	.detail-value {
		color: var(--text-primary);
	}

	.roles {
		background-color: var(--bg-secondary);
		padding: 0.75rem;
		border-radius: 0.375rem;
		margin-bottom: 1rem;
	}

	.roles-label {
		font-weight: 600;
		color: var(--text-primary);
		margin-bottom: 0.5rem;
		font-size: 0.875rem;
	}

	.roles-list {
		display: flex;
		gap: 1.5rem;
		flex-wrap: wrap;
	}

	.role-item {
		display: flex;
		gap: 0.5rem;
		font-size: 0.875rem;
	}

	.role-type {
		font-weight: 600;
		color: var(--text-secondary);
	}

	.role-referee {
		color: var(--text-primary);
	}

	.role-referee.acknowledged {
		color: #059669;
		font-weight: 500;
	}

	.ack-check {
		display: inline-block;
		margin-left: 0.25rem;
		color: #059669;
		font-weight: bold;
		font-size: 0.9rem;
	}

	.match-actions {
		display: flex;
		gap: 0.5rem;
		flex-wrap: wrap;
	}

	.btn-small {
		padding: 0.5rem 1rem;
		font-size: 0.875rem;
		border: none;
		border-radius: 0.25rem;
		cursor: pointer;
		font-weight: 500;
		transition: all 0.2s;
	}

	.btn-small:disabled {
		opacity: 0.5;
		cursor: not-allowed;
	}

	.btn-primary {
		background-color: var(--primary-color);
		color: white;
	}

	.btn-primary:hover:not(:disabled) {
		background-color: #1d4ed8;
	}

	.btn-secondary {
		background-color: white;
		color: var(--text-primary);
		border: 1px solid var(--border-color);
	}

	.btn-secondary:hover:not(:disabled) {
		background-color: var(--bg-secondary);
	}

	.btn-warning {
		background-color: #fbbf24;
		color: #78350f;
	}

	.btn-warning:hover:not(:disabled) {
		background-color: #f59e0b;
	}

	.btn-success {
		background-color: var(--success-color);
		color: white;
	}

	.btn-success:hover:not(:disabled) {
		background-color: #15803d;
	}

	.alert {
		padding: 1rem;
		border-radius: 0.375rem;
		margin-bottom: 1.5rem;
	}

	.alert-error {
		background-color: #fee;
		color: var(--error-color);
		border: 1px solid var(--error-color);
	}

	.empty-actions {
		margin-top: 1rem;
	}

	@media (max-width: 768px) {
		.header {
			flex-direction: column;
			align-items: flex-start;
		}

		.header-actions {
			width: 100%;
		}

		.btn {
			width: 100%;
		}

		.filters {
			flex-direction: column;
		}

		.filter-group {
			width: 100%;
		}

		.matches-grid {
			grid-template-columns: 1fr;
		}

		.match-header {
			flex-direction: column;
			gap: 1rem;
		}

		.match-date-time {
			min-width: auto;
		}

		.match-actions {
			flex-direction: column;
		}

		.btn-small {
			width: 100%;
		}
	}

	/* Modal styles */
	.modal-overlay {
		position: fixed;
		top: 0;
		left: 0;
		right: 0;
		bottom: 0;
		background-color: rgba(0, 0, 0, 0.5);
		display: flex;
		justify-content: center;
		align-items: center;
		z-index: 1000;
		padding: 1rem;
	}

	.modal {
		background-color: white;
		border-radius: 0.5rem;
		max-width: 600px;
		width: 100%;
		max-height: 90vh;
		overflow-y: auto;
		box-shadow: 0 20px 25px -5px rgba(0, 0, 0, 0.1), 0 10px 10px -5px rgba(0, 0, 0, 0.04);
	}

	.modal-header {
		display: flex;
		justify-content: space-between;
		align-items: center;
		padding: 1.5rem;
		border-bottom: 1px solid var(--border-color);
	}

	.modal-header h2 {
		font-size: 1.5rem;
		font-weight: 600;
		color: var(--text-primary);
		margin: 0;
	}

	.close-btn {
		background: none;
		border: none;
		font-size: 2rem;
		line-height: 1;
		cursor: pointer;
		color: var(--text-secondary);
		padding: 0;
		width: 2rem;
		height: 2rem;
		display: flex;
		align-items: center;
		justify-content: center;
	}

	.close-btn:hover {
		color: var(--text-primary);
	}

	.modal form {
		padding: 1.5rem;
	}

	.form-row {
		display: grid;
		grid-template-columns: 1fr 1fr;
		gap: 1rem;
		margin-bottom: 1rem;
	}

	.form-group {
		margin-bottom: 1rem;
	}

	.form-group label {
		display: block;
		margin-bottom: 0.5rem;
		font-weight: 500;
		color: var(--text-primary);
	}

	.form-group input,
	.form-group select,
	.form-group textarea {
		width: 100%;
		padding: 0.75rem;
		border: 1px solid var(--border-color);
		border-radius: 0.375rem;
		font-size: 1rem;
		font-family: inherit;
	}

	.form-group input:focus,
	.form-group select:focus,
	.form-group textarea:focus {
		outline: none;
		border-color: var(--primary-color);
		box-shadow: 0 0 0 3px rgba(37, 99, 235, 0.1);
	}

	.form-group small {
		display: block;
		margin-top: 0.25rem;
		color: var(--text-secondary);
		font-size: 0.875rem;
	}

	.warning-text {
		color: #d97706;
	}

	.modal-actions {
		display: flex;
		gap: 1rem;
		justify-content: flex-end;
		padding-top: 1rem;
		border-top: 1px solid var(--border-color);
	}

	@media (max-width: 640px) {
		.form-row {
			grid-template-columns: 1fr;
		}

		.modal-actions {
			flex-direction: column-reverse;
		}

		.modal-actions .btn {
			width: 100%;
		}
	}

		/* Assignment Panel Styles */
		.assignment-modal {
			max-width: 700px;
		}

		.match-subtitle {
			font-size: 0.95rem;
			color: var(--text-secondary);
			margin-top: 0.25rem;
		}

		.match-details-small {
			font-size: 0.85rem;
			color: var(--text-secondary);
			margin-top: 0.25rem;
		}

		.assignment-content {
			padding: 1.5rem;
		}

		.roles-grid {
			display: grid;
			gap: 1rem;
		}

		.role-card {
			border: 2px solid var(--border-color);
			border-radius: 0.5rem;
			overflow: hidden;
		}

		.role-card-header {
			background-color: var(--bg-secondary);
			padding: 1rem;
			display: flex;
			justify-content: space-between;
			align-items: center;
			border-bottom: 1px solid var(--border-color);
		}

		.role-card-header h3 {
			margin: 0;
			font-size: 1.1rem;
			font-weight: 600;
			color: var(--text-primary);
		}

		.assigned-badge {
			background-color: #d1fae5;
			color: #065f46;
			padding: 0.25rem 0.75rem;
			border-radius: 0.25rem;
			font-size: 0.85rem;
			font-weight: 500;
		}

		.unassigned-badge {
			background-color: #fee2e2;
			color: #991b1b;
			padding: 0.25rem 0.75rem;
			border-radius: 0.25rem;
			font-size: 0.85rem;
			font-weight: 500;
		}

		.role-card-body {
			padding: 1rem;
		}

		.current-assignment {
			display: flex;
			flex-direction: column;
			gap: 0.75rem;
		}

		.referee-info-row {
			display: flex;
			align-items: center;
			justify-content: space-between;
			gap: 0.5rem;
			flex-wrap: wrap;
		}

		.current-assignment .referee-name {
			font-size: 1.05rem;
			font-weight: 600;
			color: var(--text-primary);
		}

		.ack-badge {
			padding: 0.25rem 0.75rem;
			border-radius: 0.25rem;
			font-size: 0.85rem;
			font-weight: 500;
			white-space: nowrap;
		}

		.ack-confirmed {
			background-color: #d1fae5;
			color: #065f46;
		}

		.ack-pending {
			background-color: #fef3c7;
			color: #92400e;
		}

		.ack-overdue {
			background-color: #fecaca;
			color: #991b1b;
			font-weight: 600;
			animation: pulse 2s ease-in-out infinite;
		}

		@keyframes pulse {
			0%, 100% {
				opacity: 1;
			}
			50% {
				opacity: 0.7;
			}
		}

		.assignment-actions {
			display: flex;
			gap: 0.5rem;
		}

		.full-width {
			width: 100%;
		}

		.picker-header {
			display: flex;
			align-items: center;
			gap: 1rem;
			margin-bottom: 1.5rem;
			padding-bottom: 1rem;
			border-bottom: 2px solid var(--border-color);
		}

		.picker-header h3 {
			margin: 0;
			font-size: 1.25rem;
			font-weight: 600;
			color: var(--text-primary);
		}

		.btn-back {
			background: none;
			border: 1px solid var(--border-color);
			padding: 0.5rem 1rem;
			border-radius: 0.375rem;
			cursor: pointer;
			font-size: 0.95rem;
			font-weight: 500;
			color: var(--text-primary);
			transition: all 0.2s;
		}

		.btn-back:hover {
			background-color: var(--bg-secondary);
		}

		.referee-list {
			display: flex;
			flex-direction: column;
			gap: 1.5rem;
			max-height: 60vh;
			overflow-y: auto;
		}

		.referee-section {
			display: flex;
			flex-direction: column;
			gap: 0.5rem;
		}

		.section-title {
			font-size: 0.95rem;
			font-weight: 600;
			color: var(--text-primary);
			margin: 0 0 0.5rem 0;
			padding-bottom: 0.5rem;
			border-bottom: 1px solid var(--border-color);
		}

		.section-title.ineligible {
			color: var(--text-secondary);
		}

		.referee-item {
			display: flex;
			justify-content: space-between;
			align-items: center;
			padding: 1rem;
			background-color: white;
			border: 2px solid var(--border-color);
			border-radius: 0.5rem;
			cursor: pointer;
			transition: all 0.2s;
			text-align: left;
			width: 100%;
		}

		.referee-item.has-availability {
			background-color: #f0fdf4;
			border-color: #86efac;
		}

		.referee-item.has-availability:hover {
			border-color: #22c55e;
			background-color: #dcfce7;
		}

		.referee-item:not(.ineligible):not(.has-availability):hover {
			border-color: var(--primary-color);
			background-color: #eff6ff;
		}

		.referee-item:disabled {
			opacity: 0.6;
			cursor: not-allowed;
		}

		.referee-item.ineligible {
			opacity: 0.6;
			cursor: default;
			background-color: #fafafa;
		}

		.referee-info {
			flex: 1;
			display: flex;
			flex-direction: column;
			gap: 0.5rem;
		}

		.referee-name-row {
			display: flex;
			align-items: center;
			gap: 0.75rem;
			flex-wrap: wrap;
		}

		.referee-name {
			font-weight: 600;
			color: var(--text-primary);
			font-size: 1rem;
		}

		.availability-badge {
			color: #22c55e;
			font-size: 1.25rem;
			line-height: 1;
			display: inline-flex;
			align-items: center;
		}

		.grade-badge {
			background-color: #dbeafe;
			color: #1e40af;
			padding: 0.2rem 0.6rem;
			border-radius: 0.25rem;
			font-size: 0.8rem;
			font-weight: 500;
		}

		.available-indicator {
			background-color: #22c55e;
			color: white;
			padding: 0.2rem 0.6rem;
			border-radius: 0.25rem;
			font-size: 0.8rem;
			font-weight: 600;
		}

		.referee-details {
			display: flex;
			gap: 1rem;
			font-size: 0.875rem;
			color: var(--text-secondary);
			flex-wrap: wrap;
		}

		.cert-badge {
			background-color: #d1fae5;
			color: #065f46;
			padding: 0.15rem 0.5rem;
			border-radius: 0.25rem;
			font-weight: 500;
		}

		.ineligible-reason {
			color: #991b1b;
			font-style: italic;
		}

		.referee-action {
			margin-left: 1rem;
		}

		.assign-icon {
			font-size: 1.5rem;
			color: var(--primary-color);
			font-weight: bold;
		}

		.loading,
		.empty-state {
			text-align: center;
			padding: 2rem;
			color: var(--text-secondary);
		}

		@media (max-width: 640px) {
			.assignment-modal {
				max-width: 100%;
				max-height: 95vh;
			}

			.roles-grid {
				grid-template-columns: 1fr;
			}

			.assignment-actions {
				flex-direction: column;
			}

			.assignment-actions .btn-small {
				width: 100%;
			}

			.referee-name-row {
				flex-direction: column;
				align-items: flex-start;
			}
		}
</style>
