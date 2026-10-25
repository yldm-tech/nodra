import { createFileRoute } from '@tanstack/react-router';
import { UI } from '../docs/content';
import { DocsPage } from '../docs/DocsPage';
import { SITE_NAME, SOFTWARE, pageUrl, seoHead } from '../lib/seo';

export const Route = createFileRoute('/docs')({
	head: ({ match }) => {
		const lang = match.search.lang === 'en' ? 'en' : 'zh';
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
					inLanguage: lang === 'en' ? 'en' : 'zh-CN',
					isPartOf: { '@type': 'WebSite', name: SITE_NAME, url: pageUrl('/', lang) },
					about: { ...SOFTWARE },
				},
			],
		});
	},
	component: DocsPage,
});
