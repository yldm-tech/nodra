import { createFileRoute } from '@tanstack/react-router';
import { HomePage } from '../home/HomePage';

const META = {
	zh: {
		title: 'Nodra · 为 Go 桌面应用打造的响应式状态管理',
		description: 'Nodra 是为 Go 桌面应用打造的并发安全响应式状态库：带版本号的 Store、选择器订阅、撤销重做、可组合持久化与异步动作，类似 Zustand 的使用体验。',
		ogTitle: 'Nodra · 看得见的状态管理',
		ogDescription: '为 Go 桌面应用打造的并发安全响应式状态库。',
	},
	en: {
		title: 'Nodra · Reactive state management for Go desktop apps',
		description: 'Nodra is a concurrency-safe reactive state store for Go desktop apps: a versioned Store, selector subscriptions, undo/redo, composable persistence and async actions, with Zustand-like ergonomics.',
		ogTitle: 'Nodra · State management, made visible',
		ogDescription: 'A concurrency-safe reactive state store for Go desktop apps.',
	},
};

export const Route = createFileRoute('/')({
	head: ({ match }) => {
		const m = META[match.search.lang === 'en' ? 'en' : 'zh'];
		return {
			meta: [
				{ title: m.title },
				{ name: 'description', content: m.description },
				{ property: 'og:title', content: m.ogTitle },
				{ property: 'og:description', content: m.ogDescription },
				{ property: 'og:type', content: 'website' },
			],
		};
	},
	component: HomePage,
});
