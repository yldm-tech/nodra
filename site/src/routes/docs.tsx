import { createFileRoute } from '@tanstack/react-router';
import { UI } from '../docs/content';
import { DocsPage } from '../docs/DocsPage';

export const Route = createFileRoute('/docs')({
	head: ({ match }) => {
		const t = UI[match.search.lang === 'en' ? 'en' : 'zh'];
		return {
			meta: [
				{ title: t.metaTitle },
				{ name: 'description', content: t.metaDesc },
				{ property: 'og:title', content: t.metaTitle },
				{ property: 'og:description', content: t.metaDesc },
			],
		};
	},
	component: DocsPage,
});
