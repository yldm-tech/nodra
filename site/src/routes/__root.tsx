import { HeadContent, Outlet, Scripts, createRootRoute, retainSearchParams, useNavigate } from '@tanstack/react-router';
import { type ReactNode, useEffect } from 'react';
import { NotFound } from '../components/NotFound';
import { LANGS, LANG_STORAGE_KEY, type Lang, type LangSearch, useLang, validateLangSearch } from '../lib/lang';
import styles from '../styles.css?url';

const FONTS = 'https://fonts.googleapis.com/css2?family=Space+Grotesk:wght@400;500;700&family=Noto+Sans+JP:wght@400;500;700;900&family=Noto+Sans+KR:wght@400;500;700;900&family=Noto+Sans+SC:wght@400;500;700;900&family=JetBrains+Mono:wght@400;500;700&display=swap';

export const Route = createRootRoute({
	validateSearch: validateLangSearch,
	search: { middlewares: [retainSearchParams<LangSearch>(['lang'])] },
	head: () => ({
		meta: [{ charSet: 'utf-8' }, { name: 'viewport', content: 'width=device-width, initial-scale=1' }, { name: 'theme-color', content: '#060709' }],
		links: [
			{ rel: 'icon', type: 'image/svg+xml', href: '/favicon.svg' },
			{ rel: 'preconnect', href: 'https://fonts.googleapis.com' },
			{ rel: 'preconnect', href: 'https://fonts.gstatic.com', crossOrigin: 'anonymous' },
			{ rel: 'stylesheet', href: FONTS },
			{ rel: 'stylesheet', href: styles },
		],
	}),
	component: RootComponent,
	notFoundComponent: NotFound,
});

function RootComponent() {
	return (
		<RootDocument>
			<Outlet />
		</RootDocument>
	);
}

function RootDocument({ children }: { children: ReactNode }) {
	const lang = useLang();
	const navigate = useNavigate();

	// A visitor who picked a language before, landing on a bare URL, gets that language.
	useEffect(() => {
		if (lang !== 'zh') return;
		let stored: string | null = null;
		try {
			stored = localStorage.getItem(LANG_STORAGE_KEY);
		} catch {
			return;
		}
		if (stored && LANGS.includes(stored as Lang) && stored !== 'zh') void navigate({ to: '.', search: (prev: LangSearch): LangSearch => ({ ...prev, lang: stored as Exclude<Lang, 'zh'> }), hash: (prev) => prev ?? '', replace: true, resetScroll: false });
		// Only on first load; later switches go through the toggle.
	}, []);

	return (
		<html lang={lang === 'zh' ? 'zh-CN' : lang}>
			<head>
				<HeadContent />
				{/* Cloudflare Web Analytics: cookieless page views; the beacon also records client-side route changes. */}
				<script type='module' src='https://static.cloudflareinsights.com/beacon.min.js' data-cf-beacon='{"token": "1a0753d27c464839a3af309cdc3da281"}' />
			</head>
			<body>
				{children}
				<Scripts />
			</body>
		</html>
	);
}
