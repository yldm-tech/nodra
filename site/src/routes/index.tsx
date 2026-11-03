import { createFileRoute } from '@tanstack/react-router';
import { HomePage } from '../home/HomePage';
import { LOCALES } from '../lib/i18n';
import { SITE_NAME, SITE_URL, SOFTWARE, pageUrl, seoHead } from '../lib/seo';
import { getAdoptionStats } from '../lib/stats';

export const Route = createFileRoute('/')({
	head: ({ match }) => {
		const lang = match.search.lang ?? 'zh';
		const m = LOCALES[lang].seo;
		const inLanguage = LOCALES[lang].html;
		return seoHead({
			path: '/',
			lang,
			title: m.homeTitle,
			description: m.homeDescription,
			ogTitle: m.homeOgTitle,
			ogDescription: m.homeOgDescription,
			image: 'home',
			imageAlt: m.homeImageAlt,
			jsonLd: [
				{ '@type': 'WebSite', name: SITE_NAME, url: SITE_URL, inLanguage },
				{ ...SOFTWARE, url: pageUrl('/', lang), description: m.homeDescription, inLanguage },
			],
		});
	},
	// Not awaited: the page renders immediately and the numbers stream in when the upstream calls finish.
	loader: () => ({ stats: getAdoptionStats() }),
	staleTime: 10 * 60 * 1000,
	component: HomePage,
});
