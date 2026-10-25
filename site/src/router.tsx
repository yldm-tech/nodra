import { createRouter } from '@tanstack/react-router';
import { routeTree } from './routeTree.gen';

export function getRouter() {
	let firstRender = true;
	return createRouter({
		routeTree,
		defaultPreload: 'intent',
		// The first render after SSR would otherwise scroll the window back to the top. A visitor who already scrolled the server-rendered page while the client bundle loaded, or who arrived on a #hash the browser has already jumped to, keeps that position; later navigations restore or reset scroll as usual.
		scrollRestoration: () => {
			if (!firstRender) return true;
			firstRender = false;
			return window.scrollY === 0;
		},
	});
}
