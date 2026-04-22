<script lang="ts">
	import { goto } from '$app/navigation';

	const API_URL = import.meta.env.VITE_API_URL || 'http://localhost:8080';

	let step: 'upload' | 'preview' | 'complete' = 'upload';
	let file: File | null = null;
	let uploading = false;
	let importing = false;
	let error = '';

	// Preview data
	let rows: any[] = [];
	let duplicates: any[] = [];
	let validRows: any[] = [];
	let errorRows: any[] = [];

	// Import results
	let importResult: any = null;

	function handleFileSelect(event: Event) {
		const target = event.target as HTMLInputElement;
		if (target.files && target.files[0]) {
			file = target.files[0];

			// Validate file extension
			if (!file.name.toLowerCase().endsWith('.csv')) {
				error = 'Only .csv files are accepted';
				file = null;
				return;
			}

			error = '';
		}
	}

	async function handleUpload() {
		if (!file) {
			error = 'Please select a file';
			return;
		}

		uploading = true;
		error = '';

		try {
			const formData = new FormData();
			formData.append('file', file);

			const response = await fetch(`${API_URL}/api/matches/import/parse`, {
				method: 'POST',
				credentials: 'include',
				body: formData
			});

			if (response.ok) {
				const data = await response.json();
				rows = data.rows || [];
				duplicates = data.duplicates || [];

				// Separate valid and error rows
				validRows = rows.filter((r) => !r.error);
				errorRows = rows.filter((r) => r.error);

				step = 'preview';
			} else {
				const text = await response.text();
				error = text || 'Failed to parse CSV';
			}
		} catch (err) {
			error = 'Failed to upload file';
		} finally {
			uploading = false;
		}
	}

	async function handleConfirmImport() {
		importing = true;
		error = '';

		try {
			const response = await fetch(`${API_URL}/api/matches/import/confirm`, {
				method: 'POST',
				credentials: 'include',
				headers: {
					'Content-Type': 'application/json'
				},
				body: JSON.stringify({
					rows: validRows,
					resolutions: {} // TODO: Story 3.2 will handle duplicate resolution
				})
			});

			if (response.ok) {
				importResult = await response.json();
				step = 'complete';
			} else {
				const text = await response.text();
				error = text || 'Failed to import matches';
			}
		} catch (err) {
			error = 'Failed to import matches';
		} finally {
			importing = false;
		}
	}

	function handleStartOver() {
		step = 'upload';
		file = null;
		rows = [];
		duplicates = [];
		validRows = [];
		errorRows = [];
		importResult = null;
		error = '';
	}
</script>

<svelte:head>
	<title>Import Match Schedule - Referee Scheduler</title>
</svelte:head>

<div class="container">
	<div class="header">
		<div class="header-left">
			<img src="/logo.svg" alt="Logo" class="header-logo" />
			<h1>Import Match Schedule</h1>
		</div>
		<button on:click={() => goto('/dashboard')} class="btn btn-secondary">Back to Dashboard</button>
	</div>

	{#if error}
		<div class="alert alert-error">{error}</div>
	{/if}

	{#if step === 'upload'}
		<div class="card">
			<h2>Upload CSV File</h2>
			<p class="instructions">
				Upload a CSV export from Stack Team App. The file must include columns: event_name,
				team_name, start_date, start_time, end_time, location.
			</p>

			<div class="upload-section">
				<input
					type="file"
					accept=".csv"
					on:change={handleFileSelect}
					id="csvFile"
					class="file-input"
				/>
				<label for="csvFile" class="file-label">
					{file ? file.name : 'Choose CSV file...'}
				</label>
			</div>

			{#if file}
				<div class="actions">
					<button on:click={handleUpload} class="btn btn-primary" disabled={uploading}>
						{uploading ? 'Parsing...' : 'Parse CSV'}
					</button>
				</div>
			{/if}
		</div>
	{:else if step === 'preview'}
		<div class="card">
			<h2>Import Preview</h2>
			<p class="summary">
				<strong>{validRows.length}</strong> matches ready to import •
				<strong class="error-count">{errorRows.length}</strong> rows with errors •
				<strong class="duplicate-count">{duplicates.length}</strong> duplicate groups
			</p>

			{#if duplicates.length > 0}
				<div class="alert alert-warning">
					<strong>⚠️ Duplicates detected</strong>
					<p>
						{duplicates.length} duplicate match group(s) found. Duplicate resolution will be implemented
						in Story 3.2. For now, all rows will be imported (duplicates included).
					</p>
				</div>
			{/if}

			{#if errorRows.length > 0}
				<div class="section">
					<h3>Rows with Errors ({errorRows.length})</h3>
					<p class="section-info">These rows will be skipped:</p>
					<div class="table-container">
						<table class="preview-table">
							<thead>
								<tr>
									<th>Row</th>
									<th>Team</th>
									<th>Date</th>
									<th>Time</th>
									<th>Age Group</th>
									<th>Error</th>
								</tr>
							</thead>
							<tbody>
								{#each errorRows as row}
									<tr class="error-row">
										<td>{row.row_number}</td>
										<td>{row.team_name}</td>
										<td>{row.start_date}</td>
										<td>{row.start_time}</td>
										<td>
											{#if row.age_group}
												{row.age_group}
											{:else}
												<span class="text-muted">—</span>
											{/if}
										</td>
										<td class="error-cell">{row.error}</td>
									</tr>
								{/each}
							</tbody>
						</table>
					</div>
				</div>
			{/if}

			<div class="section">
				<h3>Valid Matches ({validRows.length})</h3>
				<div class="table-container">
					<table class="preview-table">
						<thead>
							<tr>
								<th>Row</th>
								<th>Event Name</th>
								<th>Team</th>
								<th>Age Group</th>
								<th>Date</th>
								<th>Time</th>
								<th>Location</th>
							</tr>
						</thead>
						<tbody>
							{#each validRows as row}
								<tr>
									<td>{row.row_number}</td>
									<td>{row.event_name}</td>
									<td>{row.team_name}</td>
									<td>{row.age_group || '—'}</td>
									<td>{row.start_date}</td>
									<td>{row.start_time} - {row.end_time}</td>
									<td>{row.location}</td>
								</tr>
							{/each}
						</tbody>
					</table>
				</div>
			</div>

			<div class="actions">
				<button on:click={handleStartOver} class="btn btn-secondary">Cancel</button>
				<button
					on:click={handleConfirmImport}
					class="btn btn-primary"
					disabled={importing || validRows.length === 0}
				>
					{importing ? 'Importing...' : `Import ${validRows.length} Matches`}
				</button>
			</div>
		</div>
	{:else if step === 'complete'}
		<div class="card">
			<h2>✅ Import Complete</h2>

			<div class="result-summary">
				<div class="result-item success">
					<span class="result-label">Imported:</span>
					<span class="result-value">{importResult.imported}</span>
				</div>
				<div class="result-item">
					<span class="result-label">Skipped:</span>
					<span class="result-value">{importResult.skipped}</span>
				</div>
			</div>

			{#if importResult.errors && importResult.errors.length > 0}
				<div class="alert alert-warning">
					<strong>Import Errors:</strong>
					<ul>
						{#each importResult.errors as err}
							<li>{err}</li>
						{/each}
					</ul>
				</div>
			{/if}

			<div class="actions">
				<button on:click={handleStartOver} class="btn btn-secondary">Import Another File</button>
				<button on:click={() => goto('/assignor')} class="btn btn-primary">
					View Schedule
				</button>
			</div>
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

	h2 {
		font-size: 1.5rem;
		font-weight: 600;
		margin-bottom: 1rem;
		color: var(--text-primary);
	}

	h3 {
		font-size: 1.25rem;
		font-weight: 600;
		margin-bottom: 0.5rem;
		color: var(--text-primary);
	}

	.instructions {
		color: var(--text-secondary);
		margin-bottom: 1.5rem;
		line-height: 1.6;
	}

	.upload-section {
		margin-bottom: 1.5rem;
	}

	.file-input {
		display: none;
	}

	.file-label {
		display: inline-block;
		padding: 0.75rem 1.5rem;
		background-color: white;
		border: 2px dashed var(--border-color);
		border-radius: 0.375rem;
		cursor: pointer;
		color: var(--text-secondary);
		transition: all 0.2s;
	}

	.file-label:hover {
		border-color: var(--primary-color);
		color: var(--primary-color);
	}

	.summary {
		padding: 1rem;
		background-color: var(--bg-secondary);
		border-radius: 0.375rem;
		margin-bottom: 1.5rem;
	}

	.error-count {
		color: var(--error-color);
	}

	.duplicate-count {
		color: #d97706;
	}

	.section {
		margin-bottom: 2rem;
	}

	.section-info {
		color: var(--text-secondary);
		margin-bottom: 1rem;
		font-size: 0.875rem;
	}

	.table-container {
		overflow-x: auto;
		margin-bottom: 1rem;
	}

	.preview-table {
		width: 100%;
		border-collapse: collapse;
		font-size: 0.875rem;
	}

	.preview-table th {
		text-align: left;
		padding: 0.75rem;
		border-bottom: 2px solid var(--border-color);
		font-weight: 600;
		color: var(--text-primary);
		background-color: var(--bg-secondary);
	}

	.preview-table td {
		padding: 0.75rem;
		border-bottom: 1px solid var(--border-color);
	}

	.preview-table tr:hover {
		background-color: var(--bg-secondary);
	}

	.error-row {
		background-color: #fee;
	}

	.error-row:hover {
		background-color: #fdd;
	}

	.error-cell {
		color: var(--error-color);
		font-weight: 500;
	}

	.text-muted {
		color: var(--text-secondary);
		font-style: italic;
	}

	.actions {
		display: flex;
		gap: 1rem;
		flex-wrap: wrap;
		margin-top: 1.5rem;
	}

	.result-summary {
		display: flex;
		gap: 2rem;
		margin: 2rem 0;
	}

	.result-item {
		display: flex;
		flex-direction: column;
		gap: 0.5rem;
	}

	.result-item.success .result-value {
		color: var(--success-color);
	}

	.result-label {
		font-weight: 600;
		color: var(--text-primary);
	}

	.result-value {
		font-size: 2rem;
		font-weight: 700;
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

	.alert-warning {
		background-color: #fffbeb;
		color: #92400e;
		border: 1px solid #fbbf24;
	}

	.alert ul {
		margin-top: 0.5rem;
		margin-left: 1.5rem;
	}

	.alert li {
		margin-bottom: 0.25rem;
	}

	.btn-secondary {
		background-color: white;
		color: var(--text-primary);
		border: 1px solid var(--border-color);
	}

	.btn-secondary:hover {
		background-color: var(--bg-secondary);
	}

	@media (max-width: 768px) {
		.header {
			flex-direction: column;
			align-items: flex-start;
		}

		.preview-table {
			font-size: 0.75rem;
		}

		.preview-table th,
		.preview-table td {
			padding: 0.5rem;
		}

		.result-summary {
			flex-direction: column;
			gap: 1rem;
		}

		.actions {
			flex-direction: column;
		}

		.btn {
			width: 100%;
		}
	}
</style>
