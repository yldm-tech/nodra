import { Await } from '@tanstack/react-router';
import type { Lang } from '../lib/lang';
import { REPO_URL } from '../lib/lang';
import type { AdoptionStats as Stats } from '../lib/stats';

const PKG = 'https://pkg.go.dev/github.com/yldm-tech/nodra';

const LABEL = {
	zh: { stars: (n: number) => `★ ${n} stars`, imported: (n: number) => `被 ${n.toLocaleString('en-US')} 个包导入`, aria: '项目数据' },
	en: { stars: (n: number) => `★ ${n} stars`, imported: (n: number) => `imported by ${n.toLocaleString('en-US')}`, aria: 'Project stats' },
};

const item = 'text-fg-5 hover:text-fg';

// Live adoption numbers under the install command. The promise streams in after the first paint; a number that could not be fetched is left out rather than guessed.
export function AdoptionStats({ stats, lang }: { stats: Promise<Stats>; lang: Lang }) {
	const t = LABEL[lang];
	return (
		<div className='min-h-[18px] font-mono text-xs'>
			<Await promise={stats} fallback={null}>
				{(s) => {
					const parts = [
						s.stars !== null && (
							<a key='stars' href={`${REPO_URL}/stargazers`} className={item}>
								{t.stars(s.stars)}
							</a>
						),
						s.version !== null && (
							<a key='version' href={`${PKG}?tab=versions`} className={item}>
								{s.version}
							</a>
						),
						s.importedBy !== null && (
							<a key='imported' href={`${PKG}?tab=importedby`} className={item}>
								{t.imported(s.importedBy)}
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
		</div>
	);
}
