import { createFileRoute } from '@tanstack/react-router';
import { UI } from '../docs/content';
import { DocsPage } from '../docs/DocsPage';
import { LOCALES } from '../lib/i18n';
import { SITE_NAME, SOFTWARE, pageUrl, seoHead } from '../lib/seo';

export const Route = createFileRoute('/docs')({
	head: ({ match }) => {
		const lang = match.search.lang ?? 'zh';
		const t = UI[lang];
		return seoHead({
			path: '/docs',
			lang,
			title: t.metaTitle,
			description: t.metaDesc,
			image: 'docs',
			imageAlt: t.ogImageAlt,
			type: 'article',
			jsonLd: [
				{
					'@type': 'TechArticle',
					headline: t.title,
					description: t.metaDesc,
					url: pageUrl('/docs', lang),
					inLanguage: LOCALES[lang].html,
					isPartOf: { '@type': 'WebSite', name: SITE_NAME, url: pageUrl('/', lang) },
					about: { ...SOFTWARE },
				},
			],
		});
	},
	component: DocsPage,
});
