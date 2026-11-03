import { Await } from '@tanstack/react-router';
import { LOCALES } from '../lib/i18n';
import { projectRegistrationUrl, type Lang, REPO_URL } from '../lib/lang';
import type { AdoptionStats as Stats } from '../lib/stats';

const PKG = 'https://pkg.go.dev/github.com/yldm-tech/nodra';

const item = 'text-fg-5 hover:text-fg';

// Live adoption numbers under the install command. The promise streams in after the first paint; a number that could not be fetched is left out rather than guessed.
export function AdoptionStats({ stats, lang }: { stats: Promise<Stats>; lang: Lang }) {
	const t = LOCALES[lang].adoption;
	return (
		<div className='min-h-[18px] font-mono text-xs'>
			<Await promise={stats} fallback={null}>
				{(s) => {
					const parts = [
						s.stars !== null && (
							<a key='stars' href={`${REPO_URL}/stargazers`} className={item}>
								{t.stars.replace('{n}', s.stars.toLocaleString('en-US'))}
							</a>
						),
						s.version !== null && (
							<a key='version' href={`${PKG}?tab=versions`} className={item}>
								{s.version}
							</a>
						),
						s.importedBy !== null && (
							<a key='imported' href={`${PKG}?tab=importedby`} title={t.importedTitle} className={item}>
								{t.imported.replace('{n}', s.importedBy.toLocaleString('en-US'))}
							</a>
						),
					].filter(Boolean);
					if (parts.length === 0) return null;
					return (
						<p aria-label={t.aria} className='m-0 flex flex-wrap items-center gap-x-2.5 gap-y-1 text-fg-6'>
							{parts.flatMap((p, i) => (i === 0 ? [p] : [<span key={`sep-${i}`} aria-hidden='true'>·</span>, p]))}
						</p>
					);
				}}
			</Await>
			<a href={projectRegistrationUrl(lang)} className='mt-1 inline-block text-fg-5 underline decoration-fg/25 underline-offset-4 hover:text-lime hover:decoration-lime'>
				{t.register}
			</a>
		</div>
	);
}
