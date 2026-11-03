import type { Lang } from './lang';
import zh from '../locales/zh.json';
import en from '../locales/en.json';
import ja from '../locales/ja.json';
import ko from '../locales/ko.json';
import es from '../locales/es.json';
import fr from '../locales/fr.json';

export const LOCALES = { zh, en, ja, ko, es, fr } as const;
export type Locale = (typeof LOCALES)[Lang];

export function localeFor(lang: Lang): Locale {
	return LOCALES[lang];
}

export function projectRegistrationUrl(lang: Lang): string {
	const t = localeFor(lang).registration;
	return `https://github.com/yldm-tech/nodra/issues/new?title=${encodeURIComponent(t.title)}&body=${encodeURIComponent(t.body)}`;
}
