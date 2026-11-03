import { useNavigate, useSearch } from '@tanstack/react-router';
import { useCallback } from 'react';

export type Lang = 'zh' | 'en';

// Chinese is the default and lives at the bare URL; English is `?lang=en`, so every language has a shareable, server-rendered address.
export interface LangSearch {
	lang?: 'en';
}

export const LANG_STORAGE_KEY = 'nodra-lang';

export function validateLangSearch(search: Record<string, unknown>): LangSearch {
	return search.lang === 'en' ? { lang: 'en' } : {};
}

export function useLang(): Lang {
	const search = useSearch({ strict: false }) as LangSearch;
	return search.lang === 'en' ? 'en' : 'zh';
}

export function useSetLang(): (lang: Lang) => void {
	const navigate = useNavigate();
	return useCallback(
		(lang: Lang) => {
			try {
				localStorage.setItem(LANG_STORAGE_KEY, lang);
			} catch {
				// Storage can be unavailable (private mode); the URL still carries the choice.
			}
			void navigate({
				to: '.',
				search: (prev: LangSearch): LangSearch => ({ ...prev, lang: lang === 'en' ? 'en' : undefined }),
				hash: (prev) => prev ?? '',
				replace: true,
				resetScroll: false,
			});
		},
		[navigate],
	);
}

export const INSTALL_CMD = 'go get github.com/yldm-tech/nodra@latest';
export const REPO_URL = 'https://github.com/yldm-tech/nodra';

export function projectRegistrationUrl(lang: Lang): string {
	const title = lang === 'en' ? 'Register a project using Nodra' : '登记使用 Nodra 的项目';
	const body =
		lang === 'en'
			? 'Project name:\nProject URL:\nHow you use Nodra:\nMay we list this project on the Nodra website? (yes/no)\n\nPlease do not include private or sensitive information.'
			: '项目名称：\n项目链接：\n如何使用 Nodra：\n是否同意在 Nodra 官网展示此项目？（是/否）\n\n请勿提交私密或敏感信息。';
	return `${REPO_URL}/issues/new?title=${encodeURIComponent(title)}&body=${encodeURIComponent(body)}`;
}
