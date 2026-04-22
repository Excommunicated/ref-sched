<script lang="ts">
	import { onMount } from 'svelte';
	import { goto } from '$app/navigation';
	import type { PageData } from './$types';

	export let data: PageData;

	const API_URL = import.meta.env.VITE_API_URL || 'http://localhost:8080';

	let loading = true;
	let referees: any[] = [];
	let filteredReferees: any[] = [];
	let searchTerm = '';
	let statusFilter = 'all';
	let showAssignors = true;
	let error = '';

	$: currentUserId = data.user?.id;

	onMount(async () => {
		await loadReferees();
	});

	async function loadReferees() {
		loading = true;
		error = '';

		try {
			const response = await fetch(`${API_URL}/api/referees`, {
				credentials: 'include'
			});

			if (response.ok) {
				referees = await response.json();
				filterReferees();
			} else {
				error = 'Failed to load referees';
			}
		} catch (err) {
			error = 'Failed to load referees';
		} finally {
			loading = false;
		}
	}

	function filterReferees() {
		let filtered = referees;

		// Filter assignors if needed
		if (!showAssignors) {
			filtered = filtered.filter((ref) => ref.role !== 'assignor');
		}

		// Filter by status
		if (statusFilter !== 'all') {
			filtered = filtered.filter((ref) => ref.status === statusFilter);
		}

		// Filter by search term
		if (searchTerm.trim()) {
			const term = searchTerm.toLowerCase();
			filtered = filtered.filter(
				(ref) =>
					ref.name?.toLowerCase().includes(term) ||
					ref.email?.toLowerCase().includes(term) ||
					ref.first_name?.toLowerCase().includes(term) ||
					ref.last_name?.toLowerCase().includes(term)
			);
		}

		filteredReferees = filtered;
	}

	async function updateReferee(refereeId: number, updates: any) {
		try {
			const response = await fetch(`${API_URL}/api/referees/${refereeId}`, {
				method: 'PUT',
				credentials: 'include',
				headers: {
					'Content-Type': 'application/json'
				},
				body: JSON.stringify(updates)
			});

			if (response.ok) {
				await loadReferees();
			} else {
				const text = await response.text();
				alert(`Failed to update referee: ${text}`);
			}
		} catch (err) {
			alert('Failed to update referee');
		}
	}

	function handleStatusChange(refereeId: number, newStatus: string) {
		// Prevent self-deactivation
		if (refereeId === currentUserId && (newStatus === 'inactive' || newStatus === 'removed')) {
			alert('You cannot deactivate your own account.');
			return;
		}

		if (
			newStatus === 'removed' &&
			!confirm('Are you sure you want to remove this referee? This cannot be undone.')
		) {
			return;
		}

		if (
			newStatus === 'inactive' &&
			!confirm('Are you sure you want to deactivate this referee?')
		) {
			return;
		}

		updateReferee(refereeId, { status: newStatus });
	}

	function handleRoleChange(refereeId: number, newRole: string) {
		const referee = referees.find(r => r.id === refereeId);
		const oldRole = referee?.role;

		if (oldRole === newRole) {
			return;
		}

		if (newRole === 'assignor') {
			if (!confirm('Are you sure you want to promote this user to Assignor? They will have full access to manage referees and assignments.')) {
				return;
			}
		} else if (oldRole === 'assignor') {
			if (!confirm('Are you sure you want to demote this Assignor to Referee?')) {
				return;
			}
		}

		updateReferee(refereeId, { role: newRole });
	}

	function handleGradeChange(refereeId: number, newGrade: string) {
		updateReferee(refereeId, { grade: newGrade || '' });
	}

	function getCertStatusBadge(certStatus: string) {
		const badges = {
			valid: { class: 'badge-success', text: 'Valid' },
			expiring_soon: { class: 'badge-warning', text: 'Expiring Soon' },
			expired: { class: 'badge-error', text: 'Expired' },
			none: { class: 'badge-secondary', text: 'Not Certified' }
		};
		return badges[certStatus] || badges.none;
	}

	function getStatusBadge(status: string) {
		const badges = {
			pending: { class: 'badge-warning', text: 'Pending' },
			active: { class: 'badge-success', text: 'Active' },
			inactive: { class: 'badge-secondary', text: 'Inactive' },
			removed: { class: 'badge-error', text: 'Removed' }
		};
		return badges[status] || { class: 'badge-secondary', text: status };
	}

	function formatDate(dateString: string | null) {
		if (!dateString) return 'N/A';
		return new Date(dateString).toLocaleDateString();
	}

	function calculateAge(dob: string | null) {
		if (!dob) return 'N/A';
		const today = new Date();
		const birthDate = new Date(dob);
		let age = today.getFullYear() - birthDate.getFullYear();
		const monthDiff = today.getMonth() - birthDate.getMonth();
		if (monthDiff < 0 || (monthDiff === 0 && today.getDate() < birthDate.getDate())) {
			age--;
		}
		return age;
	}

	$: {
		searchTerm;
		statusFilter;
		showAssignors;
		filterReferees();
	}
</script>

<svelte:head>
	<title>Manage Referees - Referee Scheduler</title>
</svelte:head>

<div class="container">
	<div class="header">
		<div class="header-left">
			<img src="/logo.svg" alt="Logo" class="header-logo" />
			<h1>Manage Referees</h1>
		</div>
		<button on:click={() => goto('/dashboard')} class="btn btn-secondary">Back to Dashboard</button
		>
	</div>

	{#if error}
		<div class="alert alert-error">{error}</div>
	{/if}

	<div class="filters card">
		<div class="filter-group">
			<label for="search">Search</label>
			<input
				type="text"
				id="search"
				bind:value={searchTerm}
				placeholder="Search by name or email..."
			/>
		</div>

		<div class="filter-group">
			<label for="status">Filter by Status</label>
			<select id="status" bind:value={statusFilter}>
				<option value="all">All</option>
				<option value="pending">Pending</option>
				<option value="active">Active</option>
				<option value="inactive">Inactive</option>
			</select>
		</div>

		<div class="filter-group checkbox-filter">
			<label class="checkbox-label">
				<input type="checkbox" bind:checked={showAssignors} />
				<span>Show assignors</span>
			</label>
		</div>

		<div class="stats">
			<strong>{filteredReferees.length}</strong> referee{filteredReferees.length !== 1
				? 's'
				: ''} shown
		</div>
	</div>

	{#if loading}
		<div class="card">
			<p>Loading referees...</p>
		</div>
	{:else if filteredReferees.length === 0}
		<div class="card">
			<p>No referees found matching your filters.</p>
		</div>
	{:else}
		<div class="table-container card">
			<table>
				<thead>
					<tr>
						<th>Name</th>
						<th>Email</th>
						<th>Age</th>
						<th>DOB</th>
						<th>Certification</th>
						<th>Status</th>
						<th>Role</th>
						<th>Grade</th>
						<th>Actions</th>
					</tr>
				</thead>
				<tbody>
					{#each filteredReferees as referee}
						<tr class:pending-row={referee.status === 'pending'}>
							<td>
								<div class="name-cell">
									{#if referee.first_name && referee.last_name}
										<strong>{referee.first_name} {referee.last_name}</strong>
										<small>{referee.name}</small>
									{:else}
										<strong>{referee.name}</strong>
										<small class="incomplete">Profile incomplete</small>
									{/if}
								</div>
							</td>
							<td>{referee.email}</td>
							<td>{calculateAge(referee.date_of_birth)}</td>
							<td>{formatDate(referee.date_of_birth)}</td>
							<td>
								<span class="badge {getCertStatusBadge(referee.cert_status).class}">
									{getCertStatusBadge(referee.cert_status).text}
								</span>
								{#if referee.certified && referee.cert_expiry}
									<small>Expires: {formatDate(referee.cert_expiry)}</small>
								{/if}
							</td>
							<td>
								{#if referee.role === 'assignor'}
									<span class="badge badge-assignor">Assignor</span>
								{:else}
									<span class="badge {getStatusBadge(referee.status).class}">
										{getStatusBadge(referee.status).text}
									</span>
								{/if}
							</td>
							<td>
								{#if referee.id === currentUserId}
									<span class="text-muted">You</span>
								{:else}
									<select
										value={referee.role}
										on:change={(e) => handleRoleChange(referee.id, e.currentTarget.value)}
										disabled={referee.status === 'removed'}
									>
										<option value="referee">Referee</option>
										<option value="assignor">Assignor</option>
									</select>
								{/if}
							</td>
							<td>
								{#if referee.role === 'assignor' && referee.id !== currentUserId}
									<span class="text-muted">N/A</span>
								{:else}
									<select
										value={referee.grade || ''}
										on:change={(e) => handleGradeChange(referee.id, e.currentTarget.value)}
										disabled={referee.status === 'removed'}
									>
										<option value="">No Grade</option>
										<option value="Junior">Junior</option>
										<option value="Mid">Mid</option>
										<option value="Senior">Senior</option>
									</select>
								{/if}
							</td>
							<td>
								{#if referee.role === 'assignor' && referee.id !== currentUserId}
									<span class="text-muted">—</span>
								{:else if referee.id === currentUserId}
									<span class="text-muted">You</span>
								{:else}
									<div class="action-buttons">
										{#if referee.status === 'pending'}
											<button
												class="btn-small btn-success"
												on:click={() => handleStatusChange(referee.id, 'active')}
											>
												Activate
											</button>
										{:else if referee.status === 'active'}
											<button
												class="btn-small btn-secondary"
												on:click={() => handleStatusChange(referee.id, 'inactive')}
											>
												Deactivate
											</button>
										{:else if referee.status === 'inactive'}
											<button
												class="btn-small btn-success"
												on:click={() => handleStatusChange(referee.id, 'active')}
											>
												Activate
											</button>
										{/if}
										{#if referee.status !== 'removed'}
											<button
												class="btn-small btn-error"
												on:click={() => handleStatusChange(referee.id, 'removed')}
											>
												Remove
											</button>
										{/if}
									</div>
								{/if}
							</td>
						</tr>
					{/each}
				</tbody>
			</table>
		</div>
	{/if}
</div>

<style>
	.container {
		max-width: 1400px;
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

	.filter-group input,
	.filter-group select {
		width: 100%;
		padding: 0.75rem;
		border: 1px solid var(--border-color);
		border-radius: 0.375rem;
		font-size: 1rem;
		font-family: inherit;
	}

	.stats {
		color: var(--text-secondary);
		padding: 0.75rem 0;
	}

	.table-container {
		overflow-x: auto;
	}

	table {
		width: 100%;
		border-collapse: collapse;
	}

	th {
		text-align: left;
		padding: 0.75rem;
		border-bottom: 2px solid var(--border-color);
		font-weight: 600;
		color: var(--text-primary);
		background-color: var(--bg-secondary);
	}

	td {
		padding: 0.75rem;
		border-bottom: 1px solid var(--border-color);
	}

	tr:hover {
		background-color: var(--bg-secondary);
	}

	.pending-row {
		background-color: #fffbeb;
	}

	.pending-row:hover {
		background-color: #fef3c7;
	}

	.name-cell {
		display: flex;
		flex-direction: column;
		gap: 0.25rem;
	}

	.name-cell small {
		color: var(--text-secondary);
		font-size: 0.875rem;
	}

	.name-cell .incomplete {
		color: var(--warning-color);
		font-style: italic;
	}

	.badge {
		display: inline-block;
		padding: 0.25rem 0.5rem;
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

	.badge-secondary {
		background-color: var(--bg-secondary);
		color: var(--text-secondary);
	}

	.badge-assignor {
		background-color: #dbeafe;
		color: #1e40af;
		font-weight: 600;
	}

	.text-muted {
		color: var(--text-secondary);
		font-style: italic;
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

	.action-buttons {
		display: flex;
		gap: 0.5rem;
		flex-wrap: wrap;
	}

	.btn-small {
		padding: 0.375rem 0.75rem;
		font-size: 0.875rem;
		border: none;
		border-radius: 0.25rem;
		cursor: pointer;
		font-weight: 500;
		transition: all 0.2s;
	}

	.btn-success {
		background-color: var(--success-color);
		color: white;
	}

	.btn-success:hover {
		background-color: #15803d;
	}

	.btn-error {
		background-color: var(--error-color);
		color: white;
	}

	.btn-error:hover {
		background-color: #b91c1c;
	}

	.btn-secondary {
		background-color: white;
		color: var(--text-primary);
		border: 1px solid var(--border-color);
	}

	.btn-secondary:hover {
		background-color: var(--bg-secondary);
	}

	select {
		padding: 0.375rem;
		border: 1px solid var(--border-color);
		border-radius: 0.25rem;
		font-size: 0.875rem;
	}

	select:disabled {
		opacity: 0.5;
		cursor: not-allowed;
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

	@media (max-width: 768px) {
		.filters {
			flex-direction: column;
			align-items: stretch;
		}

		.filter-group {
			width: 100%;
		}

		table {
			font-size: 0.875rem;
		}

		th,
		td {
			padding: 0.5rem;
		}

		.action-buttons {
			flex-direction: column;
		}

		.btn-small {
			width: 100%;
		}
	}
</style>
