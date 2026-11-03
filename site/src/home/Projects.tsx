import { LOCALES } from '../lib/i18n';
import { projectRegistrationUrl, type Lang } from '../lib/lang';
import projects from '../data/projects.json';

const card = 'group flex min-h-[170px] flex-col justify-between rounded-2xl border border-fg/10 bg-ink-4 p-5 transition-colors hover:border-lime/50 hover:bg-panel';

export function Projects({ lang }: { lang: Lang }) {
	const t = LOCALES[lang].projects;
	return (
		<section id='projects' className='mx-auto max-w-[1312px] px-9 pt-[110px] pb-12 max-sm:px-5'>
			<div className='mb-8 flex flex-col gap-3'>
				<span className='font-mono text-xs tracking-[0.14em] text-lime'>{t.kicker}</span>
				<h2 className='m-0 max-w-[760px] text-[clamp(34px,4vw,56px)] leading-[1.05] font-bold tracking-[-0.04em]'>{t.title}</h2>
				<p className='m-0 max-w-[720px] text-base leading-[1.75] text-fg-3'>{t.lead}</p>
			</div>
			<div className='grid grid-cols-[repeat(auto-fit,minmax(220px,1fr))] gap-3'>
				{projects.map((project) => (
					<a key={project.id} href={project.url} className={card}>
						<div className='flex flex-col gap-3'>
							<span className='font-mono text-[11px] tracking-[0.12em] text-fg-5'>{t.official}</span>
							<h3 className='m-0 text-xl font-semibold tracking-[-0.02em] text-fg group-hover:text-lime'>{project.name}</h3>
							<p className='m-0 text-sm leading-[1.65] text-fg-4'>{project.description[lang] ?? project.description.en}</p>
						</div>
						<span className='mt-6 font-mono text-xs text-fg-5 group-hover:text-lime'>{t.view}</span>
					</a>
				))}
				<a href={projectRegistrationUrl(lang)} className={`${card} border-dashed border-lime/35 bg-lime/5`}>
					<div className='flex flex-col gap-3'>
						<span className='font-mono text-[11px] tracking-[0.12em] text-lime'>NODRA · COMMUNITY</span>
						<h3 className='m-0 text-xl font-semibold tracking-[-0.02em] text-fg'>{t.register}</h3>
						<p className='m-0 text-sm leading-[1.65] text-fg-4'>{t.lead}</p>
					</div>
				</a>
			</div>
		</section>
	);
}
