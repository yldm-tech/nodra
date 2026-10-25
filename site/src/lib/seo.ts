import type { JSX } from 'react';
import type { Lang } from './lang';
import { REPO_URL } from './lang';

export const SITE_URL = 'https://nodra.yldm.ai';
export const SITE_NAME = 'Nodra';

const OG_SIZE = { width: 1200, height: 630 };
const LOCALE: Record<Lang, string> = { zh: 'zh_CN', en: 'en_US' };

// TanStack renders { 'script:ld+json': data } entries as an escaped <script type="application/ld+json"> (router-core types it), but react-router narrows head meta to <meta> props, hence the cast.
const ldJson = (data: Record<string, unknown>) => ({ 'script:ld+json': { '@context': 'https://schema.org', ...data } }) as unknown as JSX.IntrinsicElements['meta'];

// Chinese lives at the bare path and English at ?lang=en; any other query string (utm tags and the like) is dropped from canonical URLs.
export function pageUrl(path: string, lang: Lang): string {
	return `${SITE_URL}${path}${lang === 'en' ? '?lang=en' : ''}`;
}

interface PageSeo {
	path: '/' | '/docs';
	lang: Lang;
	title: string;
	description: string;
	ogTitle?: string;
	ogDescription?: string;
	// Base name of the 1200x630 card in public/og/, rendered per language by scripts/og-image.sh.
	image: 'home' | 'docs';
	imageAlt: string;
	type?: 'website' | 'article';
	jsonLd?: Record<string, unknown>[];
}

export function seoHead(p: PageSeo) {
	const url = pageUrl(p.path, p.lang);
	const image = `${SITE_URL}/og/${p.image}-${p.lang}.png`;
	const ogTitle = p.ogTitle ?? p.title;
	const ogDescription = p.ogDescription ?? p.description;
	const other: Lang = p.lang === 'en' ? 'zh' : 'en';
	return {
		meta: [
			{ title: p.title },
			{ name: 'description', content: p.description },
			{ name: 'robots', content: 'index, follow, max-image-preview:large' },
			{ property: 'og:type', content: p.type ?? 'website' },
			{ property: 'og:site_name', content: SITE_NAME },
			{ property: 'og:title', content: ogTitle },
			{ property: 'og:description', content: ogDescription },
			{ property: 'og:url', content: url },
			{ property: 'og:locale', content: LOCALE[p.lang] },
			{ property: 'og:locale:alternate', content: LOCALE[other] },
			{ property: 'og:image', content: image },
			{ property: 'og:image:type', content: 'image/png' },
			{ property: 'og:image:width', content: String(OG_SIZE.width) },
			{ property: 'og:image:height', content: String(OG_SIZE.height) },
			{ property: 'og:image:alt', content: p.imageAlt },
			{ name: 'twitter:card', content: 'summary_large_image' },
			{ name: 'twitter:title', content: ogTitle },
			{ name: 'twitter:description', content: ogDescription },
			{ name: 'twitter:image', content: image },
			{ name: 'twitter:image:alt', content: p.imageAlt },
			...(p.jsonLd ?? []).map(ldJson),
		],
		links: [
			{ rel: 'canonical', href: url },
			{ rel: 'alternate', hrefLang: 'zh-CN', href: pageUrl(p.path, 'zh') },
			{ rel: 'alternate', hrefLang: 'en', href: pageUrl(p.path, 'en') },
			{ rel: 'alternate', hrefLang: 'x-default', href: pageUrl(p.path, 'zh') },
		],
	};
}

export const SOFTWARE = {
	'@type': 'SoftwareSourceCode',
	name: SITE_NAME,
	codeRepository: REPO_URL,
	programmingLanguage: { '@type': 'ComputerLanguage', name: 'Go', url: 'https://go.dev' },
	runtimePlatform: 'Go 1.27.3+',
	license: 'https://opensource.org/licenses/MIT',
	author: { '@type': 'Organization', name: 'yldm-tech', url: 'https://github.com/yldm-tech' },
};
