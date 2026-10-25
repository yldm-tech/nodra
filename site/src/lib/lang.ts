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
