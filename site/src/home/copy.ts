import type { Lang } from '../lib/lang';
import { LOCALES } from '../lib/i18n';

export const T = Object.fromEntries(Object.entries(LOCALES).map(([lang, locale]) => [lang, locale.home])) as Record<Lang, (typeof LOCALES)['zh']['home']>;
export type Copy = (typeof T)[Lang];

export const MYGO_SNIPPET = `mygo.App.WhenReady(func() {
\twindow := mygo.NewWindow(options)
\t// redraws until the window closes
\tnodra.InvalidateSelector(store, window,
\t\tfunc(s State) int { return s.Count },
\t)
})`;
