import { useNavigate, useSearch } from '@tanstack/react-router';
import { useCallback } from 'react';
import { projectRegistrationUrl } from './i18n';

export type Lang = 'zh' | 'en' | 'ja' | 'ko' | 'es' | 'fr';

export const LANGS: Lang[] = ['zh', 'en', 'ja', 'ko', 'es', 'fr'];

// Chinese is the default and lives at the bare URL; other languages use `?lang=…`, so every language has a shareable, server-rendered address.
export interface LangSearch {
	lang?: Exclude<Lang, 'zh'>;
}

export const LANG_STORAGE_KEY = 'nodra-lang';

export function validateLangSearch(search: Record<string, unknown>): LangSearch {
	return typeof search.lang === 'string' && LANGS.includes(search.lang as Lang) && search.lang !== 'zh' ? { lang: search.lang as Exclude<Lang, 'zh'> } : {};
}

export function useLang(): Lang {
	const search = useSearch({ strict: false }) as LangSearch;
	return search.lang && LANGS.includes(search.lang) ? search.lang : 'zh';
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
				search: (prev: LangSearch): LangSearch => ({ ...prev, lang: lang === 'zh' ? undefined : lang }),
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

export { projectRegistrationUrl };
