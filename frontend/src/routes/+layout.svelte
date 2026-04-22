<script lang="ts">
	import '../app.css';
	import { page } from '$app/stores';
	import { onMount } from 'svelte';
	import { goto } from '$app/navigation';
	import type { LayoutData } from './$types';

	export let data: LayoutData;

	let user = data.user;

	onMount(() => {
		// Check if user needs to be redirected based on role
		if (user && $page.url.pathname === '/') {
			redirectBasedOnRole(user.role);
		}
	});

	function redirectBasedOnRole(role: string) {
		if (role === 'assignor') {
			goto('/assignor');
		} else if (role === 'referee') {
			goto('/referee');
		} else if (role === 'pending_referee') {
			goto('/pending');
		}
	}
</script>

<slot />
