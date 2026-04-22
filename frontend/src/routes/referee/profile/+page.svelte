<script lang="ts">
	import { goto } from '$app/navigation';
	import { onMount } from 'svelte';
	import type { PageData } from './$types';

	export let data: PageData;

	const API_URL = import.meta.env.VITE_API_URL || 'http://localhost:8080';

	let loading = true;
	let saving = false;
	let error = '';
	let success = '';

	// Profile data
	let firstName = '';
	let lastName = '';
	let dateOfBirth = '';
	let certified = false;
	let certExpiry = '';

	// Fetch current profile
	onMount(async () => {
		try {
			const response = await fetch(`${API_URL}/api/profile`, {
				credentials: 'include'
			});

			if (response.ok) {
				const profile = await response.json();
				firstName = profile.first_name || '';
				lastName = profile.last_name || '';
				dateOfBirth = profile.date_of_birth ? profile.date_of_birth.split('T')[0] : '';
				certified = profile.certified || false;
				certExpiry = profile.cert_expiry ? profile.cert_expiry.split('T')[0] : '';
			} else {
				error = 'Failed to load profile';
			}
		} catch (err) {
			error = 'Failed to load profile';
		} finally {
			loading = false;
		}
	});

	async function handleSubmit() {
		error = '';
		success = '';
		saving = true;

		// Validation
		if (!firstName.trim() || !lastName.trim()) {
			error = 'First name and last name are required';
			saving = false;
			return;
		}

		if (!dateOfBirth) {
			error = 'Date of birth is required';
			saving = false;
			return;
		}

		if (certified && !certExpiry) {
			error = 'Certification expiry date is required when certified';
			saving = false;
			return;
		}

		try {
			const response = await fetch(`${API_URL}/api/profile`, {
				method: 'PUT',
				credentials: 'include',
				headers: {
					'Content-Type': 'application/json'
				},
				body: JSON.stringify({
					first_name: firstName.trim(),
					last_name: lastName.trim(),
					date_of_birth: dateOfBirth || null,
					certified: certified,
					cert_expiry: certified ? certExpiry : null
				})
			});

			if (response.ok) {
				success = 'Profile updated successfully!';
				setTimeout(() => {
					success = '';
				}, 3000);
			} else {
				const data = await response.text();
				error = data || 'Failed to update profile';
			}
		} catch (err) {
			error = 'Failed to update profile';
		} finally {
			saving = false;
		}
	}

	function handleBack() {
		// Route back to dashboard
		goto('/dashboard');
	}
</script>

<svelte:head>
	<title>My Profile - Referee Scheduler</title>
</svelte:head>

<div class="container">
	<div class="header">
		<div class="header-left">
			<img src="/logo.svg" alt="Logo" class="header-logo" />
			<h1>My Profile</h1>
		</div>
		<button on:click={handleBack} class="btn btn-secondary">Back to Dashboard</button>
	</div>

	{#if loading}
		<div class="card">
			<p>Loading profile...</p>
		</div>
	{:else}
		<div class="card">
			{#if error}
				<div class="alert alert-error">{error}</div>
			{/if}

			{#if success}
				<div class="alert alert-success">{success}</div>
			{/if}

			<form on:submit|preventDefault={handleSubmit}>
				<div class="form-group">
					<label for="firstName">First Name *</label>
					<input
						type="text"
						id="firstName"
						bind:value={firstName}
						required
						placeholder="Enter your first name"
					/>
				</div>

				<div class="form-group">
					<label for="lastName">Last Name *</label>
					<input
						type="text"
						id="lastName"
						bind:value={lastName}
						required
						placeholder="Enter your last name"
					/>
				</div>

				<div class="form-group">
					<label for="dateOfBirth">Date of Birth *</label>
					<input type="date" id="dateOfBirth" bind:value={dateOfBirth} required max={new Date().toISOString().split('T')[0]} />
					<small>Required for age-based eligibility calculations</small>
				</div>

				<div class="form-group">
					<label class="checkbox-label">
						<input type="checkbox" bind:checked={certified} />
						<span>I am certified</span>
					</label>
					<small>Required for center referee assignments on U12+ matches</small>
				</div>

				{#if certified}
					<div class="form-group">
						<label for="certExpiry">Certification Expiry Date *</label>
						<input
							type="date"
							id="certExpiry"
							bind:value={certExpiry}
							required
							min={new Date().toISOString().split('T')[0]}
						/>
						<small>Certification must be valid on the match date</small>
					</div>
				{/if}

				<div class="form-actions">
					<button type="submit" class="btn btn-primary" disabled={saving}>
						{saving ? 'Saving...' : 'Save Profile'}
					</button>
				</div>
			</form>
		</div>
	{/if}
</div>

<style>
	.container {
		max-width: 800px;
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

	.form-group {
		margin-bottom: 1.5rem;
	}

	label {
		display: block;
		margin-bottom: 0.5rem;
		font-weight: 500;
		color: var(--text-primary);
	}

	input[type='text'],
	input[type='date'] {
		width: 100%;
		padding: 0.75rem;
		border: 1px solid var(--border-color);
		border-radius: 0.375rem;
		font-size: 1rem;
		font-family: inherit;
	}

	input[type='text']:focus,
	input[type='date']:focus {
		outline: none;
		border-color: var(--primary-color);
		box-shadow: 0 0 0 3px rgba(37, 99, 235, 0.1);
	}

	.checkbox-label {
		display: flex;
		align-items: center;
		gap: 0.5rem;
		cursor: pointer;
	}

	.checkbox-label input[type='checkbox'] {
		width: auto;
		cursor: pointer;
	}

	.checkbox-label span {
		font-weight: 500;
	}

	small {
		display: block;
		margin-top: 0.25rem;
		color: var(--text-secondary);
		font-size: 0.875rem;
	}

	.form-actions {
		margin-top: 2rem;
		display: flex;
		gap: 1rem;
	}

	.btn-secondary {
		background-color: white;
		color: var(--text-primary);
		border: 1px solid var(--border-color);
	}

	.btn-secondary:hover {
		background-color: var(--bg-secondary);
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

	.alert-success {
		background-color: #efe;
		color: var(--success-color);
		border: 1px solid var(--success-color);
	}

	@media (max-width: 640px) {
		.header {
			flex-direction: column;
			align-items: flex-start;
		}

		.btn {
			width: 100%;
		}
	}
</style>
