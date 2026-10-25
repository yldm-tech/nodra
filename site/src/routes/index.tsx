import { createFileRoute } from '@tanstack/react-router';
import { HomePage } from '../home/HomePage';
import { SITE_NAME, SITE_URL, SOFTWARE, pageUrl, seoHead } from '../lib/seo';
import { getAdoptionStats } from '../lib/stats';

const META = {
	zh: {
		title: 'Nodra · 为 Go 桌面应用打造的响应式状态管理',
		description: 'Nodra 是为 Go 桌面应用打造的并发安全响应式状态库：带版本号的 Store、选择器订阅、撤销重做、可组合持久化与异步动作，类似 Zustand 的使用体验。',
		ogTitle: 'Nodra · 看得见的状态管理',
		ogDescription: '为 Go 桌面应用打造的并发安全响应式状态库。',
		imageAlt: 'Nodra：看得见的状态管理。吉祥物 Dra 守着一个带版本号的 Store。',
	},
	en: {
		title: 'Nodra · Reactive state management for Go desktop apps',
		description: 'Nodra is a concurrency-safe reactive state store for Go desktop apps: a versioned Store, selector subscriptions, undo/redo, composable persistence and async actions, with Zustand-like ergonomics.',
		ogTitle: 'Nodra · State management, made visible',
		ogDescription: 'A concurrency-safe reactive state store for Go desktop apps.',
		imageAlt: 'Nodra: state management, made visible. Dra the mascot guards a versioned Store.',
	},
};

export const Route = createFileRoute('/')({
	head: ({ match }) => {
		const lang = match.search.lang === 'en' ? 'en' : 'zh';
		const m = META[lang];
		const inLanguage = lang === 'en' ? 'en' : 'zh-CN';
		return seoHead({
			path: '/',
			lang,
			title: m.title,
			description: m.description,
			ogTitle: m.ogTitle,
			ogDescription: m.ogDescription,
			image: 'home',
			imageAlt: m.imageAlt,
			jsonLd: [
				{ '@type': 'WebSite', name: SITE_NAME, url: SITE_URL, inLanguage },
				{ ...SOFTWARE, url: pageUrl('/', lang), description: m.description, inLanguage },
			],
		});
	},
	// Not awaited: the page renders immediately and the numbers stream in when the upstream calls finish.
	loader: () => ({ stats: getAdoptionStats() }),
	staleTime: 10 * 60 * 1000,
	component: HomePage,
});
