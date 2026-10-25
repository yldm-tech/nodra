import { Link } from '@tanstack/react-router';
import { useEffect, useMemo, useRef, useState } from 'react';
import { Mascot } from '../components/Mascot';
import { SiteHeader } from '../components/SiteHeader';
import { highlightGo } from '../lib/highlight';
import { INSTALL_CMD, REPO_URL, useLang } from '../lib/lang';
import { GROUPS, SECTIONS, UI } from './content';

const anchorOf = (sectionId: string, method: string) => `${sectionId}-${method.replace(/[^A-Za-z0-9]+/g, '-')}`;
const highlighted = Object.fromEntries(SECTIONS.filter((s) => s.code).map((s) => [s.id, highlightGo(s.code!)]));
const eyebrow = 'font-mono text-[11px] tracking-[0.14em]';

export function DocsPage() {
	const lang = useLang();
	const li = lang === 'en' ? 1 : 0;
	const t = UI[lang];

	const [query, setQuery] = useState('');
	const [active, setActive] = useState(SECTIONS[0].id);
	const [navOpen, setNavOpen] = useState(false);
	const [copied, setCopied] = useState<string | null>(null);
	const copyTimer = useRef<ReturnType<typeof setTimeout>>(undefined);

	useEffect(() => {
		const onScroll = () => {
			let cur = SECTIONS[0].id;
			for (const el of document.querySelectorAll<HTMLElement>('[data-doc-sec]')) {
				if (el.getBoundingClientRect().top < 140) cur = el.dataset.docSec ?? cur;
			}
			setActive(cur);
		};
		onScroll();
		addEventListener('scroll', onScroll, { passive: true });
		return () => {
			removeEventListener('scroll', onScroll);
			clearTimeout(copyTimer.current);
		};
	}, []);

	const copy = (id: string, code: string) => {
		void navigator.clipboard?.writeText(code);
		setCopied(id);
		clearTimeout(copyTimer.current);
		copyTimer.current = setTimeout(() => setCopied(null), 1500);
	};

	// Sidebar groups, filtered by the search box: a section stays when its title or id matches, or when any method name does (those methods are listed under it).
	const navGroups = useMemo(() => {
		const q = query.trim().toLowerCase();
		const groups: { key: string; label: string; items: { id: string; title: string; hits: { name: string; href: string }[] }[] }[] = [];
		for (const s of SECTIONS) {
			const title = s.title[li];
			const titleHit = !q || title.toLowerCase().includes(q) || s.id.includes(q);
			const hits = q ? (s.methods ?? []).filter((m) => m[0].toLowerCase().includes(q)).map((m) => ({ name: m[0], href: `#${anchorOf(s.id, m[0])}` })) : [];
			if (!titleHit && hits.length === 0) continue;
			let g = groups.find((x) => x.key === s.group);
			if (!g) {
				g = { key: s.group, label: GROUPS[s.group][li], items: [] };
				groups.push(g);
			}
			g.items.push({ id: s.id, title, hits });
		}
		return groups;
	}, [query, li]);

	const activeTitle = (SECTIONS.find((s) => s.id === active) ?? SECTIONS[0]).title[li];
	const closeNav = () => setNavOpen(false);
	const navVisible = navOpen || query !== '';

	return (
		<div className='min-h-screen bg-ink font-sans text-fg'>
			<SiteHeader active='docs' />

			<div className='mx-auto flex max-w-[1312px] flex-wrap gap-x-16 px-9 pt-[68px] max-[819px]:px-5'>
				<aside className='sticky top-[68px] flex max-h-[calc(100vh-68px)] max-w-[280px] flex-[1_1_220px] flex-col gap-6 self-start overflow-y-auto pt-8 pb-10 [scrollbar-width:none] max-[819px]:static max-[819px]:max-h-none max-[819px]:max-w-full max-[819px]:basis-full'>
					<label className='flex items-center rounded-xl border border-fg/14 bg-ink-4'>
						<span className='pl-3.5 font-mono text-xs text-lime' aria-hidden='true'>
							/
						</span>
						<span className='sr-only'>{t.search}</span>
						<input
							type='search'
							value={query}
							onChange={(e) => setQuery(e.target.value)}
							placeholder={t.search}
							className='min-w-0 flex-1 border-0 bg-transparent px-3 py-[11px] font-sans text-[13px] text-fg outline-none placeholder:text-fg-5'
						/>
					</label>
					<button
						type='button'
						aria-expanded={navVisible}
						onClick={() => setNavOpen((o) => !o)}
						className='hidden cursor-pointer items-center justify-between rounded-xl border border-fg/14 bg-ink-4 px-3.5 py-3 font-sans text-sm text-fg max-[819px]:flex'
					>
						<span>
							{t.toc} · {activeTitle}
						</span>
						<span className='font-mono text-lime'>{navOpen ? '▴' : '▾'}</span>
					</button>
					<nav aria-label={t.toc} className={`flex flex-col gap-6 ${navVisible ? '' : 'max-[819px]:hidden'}`}>
						{navGroups.map((g) => (
							<div key={g.key} className='flex flex-col gap-0.5'>
								<span className={`${eyebrow} pb-2 pl-3.5 text-fg-5 uppercase`}>{g.label}</span>
								{g.items.map((it) => (
									<div key={it.id} className='flex flex-col'>
										<a href={`#${it.id}`} onClick={closeNav} aria-current={active === it.id ? 'location' : undefined} className={`px-3.5 py-1.5 text-sm transition-colors duration-200 hover:text-fg ${active === it.id ? 'font-semibold text-lime' : 'font-normal text-fg-4'}`}>
											{it.title}
										</a>
										{it.hits.map((h) => (
											<a key={h.href} href={h.href} onClick={closeNav} className='py-1 pr-3.5 pl-7 font-mono text-xs text-lime'>
												{h.name}
											</a>
										))}
									</div>
								))}
							</div>
						))}
					</nav>
					{navGroups.length === 0 && (
						<div className='flex flex-col items-center gap-1.5 py-2'>
							<div className='h-[150px] w-40 [zoom:0.7]'>
								<Mascot mood='sad' ver='····' lang={lang} />
							</div>
							<span className='text-[13px] text-fg-4'>{t.noResults}</span>
						</div>
					)}
				</aside>

				<main className='flex max-w-[860px] min-w-0 flex-[999_1_480px] flex-col pt-16 pb-[120px]'>
					<div className='flex flex-col gap-[18px] border-b border-fg/10 pb-11'>
						<span className='font-mono text-xs tracking-[0.14em] text-lime'>{t.kicker}</span>
						<h1 className='m-0 text-[clamp(40px,5vw,64px)] leading-[1.02] font-bold tracking-[-0.045em]'>{t.title}</h1>
						<p className='m-0 max-w-[680px] text-[17px] leading-[1.75] text-fg-3'>{t.intro}</p>
						<div className='box-border flex max-w-full items-center gap-3 self-start overflow-x-auto rounded-full border border-fg/14 bg-ink-4 px-[18px] py-3 font-mono text-[13.5px]'>
							<span className='text-lime'>$</span>
							<span className='whitespace-nowrap'>{INSTALL_CMD}</span>
						</div>
					</div>

					{SECTIONS.map((s) => (
						<section key={s.id} id={s.id} data-doc-sec={s.id} className='flex flex-col gap-6 border-b border-fg/8 pt-16 pb-10'>
							<div className='flex flex-col gap-3'>
								<span className={`${eyebrow} text-lime uppercase`}>{GROUPS[s.group][li]}</span>
								<h2 className='m-0 text-[clamp(28px,3vw,38px)] leading-[1.1] font-bold tracking-[-0.035em]'>{s.title[li]}</h2>
								<p className='m-0 text-base leading-[1.8] text-fg-3'>{s.lead[li]}</p>
							</div>

							{s.code && (
								<div className='min-w-0 overflow-hidden rounded-2xl border border-fg/10 bg-ink-3'>
									<div className='flex items-center justify-between border-b border-fg/8 py-[9px] pr-2.5 pl-[18px] font-mono text-xs text-fg-6'>
										<span className='flex items-center gap-2'>
											<span className='size-2 rounded-full bg-lime' />
											{s.file}
										</span>
										<button
											type='button'
											onClick={() => copy(s.id, s.code!)}
											className='cursor-pointer rounded-full border border-fg/16 bg-transparent px-3 py-[5px] font-mono text-[11px] text-fg-4 hover:border-lime hover:bg-lime hover:text-ink'
										>
											{copied === s.id ? t.copied : t.copy}
										</button>
									</div>
									<pre className='m-0 overflow-x-auto px-[22px] pt-5 pb-[22px] font-mono text-[13px] leading-[1.8] text-fg-2'>{highlighted[s.id]}</pre>
								</div>
							)}

							{s.methods && (
								<div className='flex flex-col overflow-hidden rounded-2xl border border-fg/10'>
									{s.methods.map((m) => (
										<div key={m[0]} id={anchorOf(s.id, m[0])} className='grid grid-cols-[minmax(140px,210px)_minmax(0,1fr)] gap-x-6 gap-y-2 border-b border-fg/6 bg-ink-3 px-5 py-4 hover:bg-panel max-sm:grid-cols-[minmax(0,1fr)] max-sm:px-4 max-sm:py-3.5'>
											<span className='min-w-0 font-mono text-sm font-medium break-words text-fg'>{m[0]}</span>
											<div className='flex min-w-0 flex-col gap-2'>
												<code className='max-w-full self-start overflow-x-auto rounded-lg border border-lime/15 bg-lime/7 px-[9px] py-[5px] font-mono text-[12.5px] whitespace-pre text-lime'>{m[1]}</code>
												<span className='text-[14.5px] leading-[1.7] text-pretty text-fg-3'>{m[2 + li]}</span>
											</div>
										</div>
									))}
								</div>
							)}

							{s.contract && (
								<div className='flex flex-col gap-3'>
									<span className={`${eyebrow} text-fg-6`}>{t.contract}</span>
									{s.contract.map((c) => (
										<div key={c[0]} className='grid grid-cols-[18px_minmax(0,1fr)] gap-2.5 text-[15px] leading-[1.75] text-fg-2'>
											<span className='mt-[11px] size-1.5 rounded-full bg-lime' />
											<span className='text-pretty'>{c[li]}</span>
										</div>
									))}
								</div>
							)}

							{s.concurrency && (
								<div className='flex flex-col gap-2 rounded-2xl border border-amber/22 bg-amber/6 px-5 py-[18px]'>
									<span className={`${eyebrow} text-amber`}>⚡ {t.concurrency}</span>
									<p className='m-0 text-[14.5px] leading-[1.75] text-fg-2'>{s.concurrency[li]}</p>
								</div>
							)}
						</section>
					))}

					<div className='flex flex-wrap justify-between gap-4 pt-11 text-[13px] text-fg-6'>
						<span>{t.footnote}</span>
						<a href={`${REPO_URL}/blob/main/docs/public-api.md`}>docs/public-api.md ↗</a>
					</div>
					<Link to='/' hash='how' className='mt-10 flex items-center justify-between gap-5 rounded-[18px] border border-lime/30 bg-lime/5 px-[26px] py-[22px] text-fg hover:bg-lime/10 hover:text-fg'>
						<span className='flex flex-col gap-1'>
							<span className={`${eyebrow} text-lime`}>{t.seeK}</span>
							<span className='text-lg font-bold tracking-[-0.02em]'>{t.seeT}</span>
						</span>
						<span className='text-[22px] text-lime'>→</span>
					</Link>
				</main>
			</div>

			<footer className='border-t border-fg/8'>
				<div className='mx-auto flex max-w-[1312px] flex-wrap justify-between gap-4 px-8 pt-[22px] pb-[30px] text-xs text-fg-5'>
					<span>© 2026 yldm-tech · MIT License</span>
					<div className='flex flex-wrap gap-5'>
						<a href={`${REPO_URL}/blob/main/CHANGELOG.md`} className='text-fg-6 hover:text-fg'>
							{t.ftChange}
						</a>
						<a href={`${REPO_URL}/blob/main/CONTRIBUTING.md`} className='text-fg-6 hover:text-fg'>
							{t.ftContrib}
						</a>
						<a href={`${REPO_URL}/blob/main/SECURITY.md`} className='text-fg-6 hover:text-fg'>
							{t.ftSec}
						</a>
					</div>
				</div>
			</footer>
		</div>
	);
}
