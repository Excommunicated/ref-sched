import type { LayoutLoad } from './$types';

const API_URL = import.meta.env.VITE_API_URL || 'http://localhost:8080';

export const load: LayoutLoad = async ({ fetch }) => {
	try {
		const response = await fetch(`${API_URL}/api/auth/me`, {
			credentials: 'include'
		});

		if (response.ok) {
			const user = await response.json();
			return { user };
		}
	} catch (error) {
		console.error('Failed to fetch user:', error);
	}

	return { user: null };
};

export const ssr = false;
