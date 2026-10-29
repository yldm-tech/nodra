import { Link, getRouteApi } from '@tanstack/react-router';
import { useCallback, useEffect, useRef, useState } from 'react';
import { Mascot } from '../components/Mascot';
import { SiteHeader } from '../components/SiteHeader';
import { INSTALL_CMD, type Lang, REPO_URL, useLang } from '../lib/lang';
import { T } from './copy';
import { Features } from './Features';
import { type FlowModel, flowView, startHeroFlow } from './flow';
import { Hero, ctaPrimary } from './Hero';
import { type HowCursor, STEP_MS, nextCursor } from './how';
import { HowItWorks } from './HowItWorks';
import { sectionsView } from './sections';
import { Command, padX } from './ui';

// Diagrams have a fixed design width; on narrow screens they are zoomed down instead of reflowed.
interface Zoom {
	hero: number;
	how: number;
	sec: number;
}

const homeRoute = getRouteApi('/');

const reducedMotion = () => matchMedia('(prefers-reduced-motion: reduce)').matches;

const inViewport = (selector: string) => {
	for (const el of document.querySelectorAll(selector)) {
		const b = el.getBoundingClientRect();
		if (b.bottom > 0 && b.top < innerHeight) return true;
	}
	return false;
};

function measureZoom(): Zoom {
	const av = Math.max(280, innerWidth - (innerWidth < 600 ? 40 : 72));
	return { hero: Math.min(1, av / 562), how: Math.min(1, av / 722), sec: Math.min(1, av / 522) };
}

export function HomePage() {
	const lang = useLang();
	const li = lang === 'en' ? 1 : 0;
	const t = T[lang];
	const { stats } = homeRoute.useLoaderData();

	const [zoom, setZoom] = useState<Zoom>({ hero: 1, how: 1, sec: 1 });
	useEffect(() => {
		let lastWidth = innerWidth;
		const onResize = () => {
			if (innerWidth === lastWidth) return;
			lastWidth = innerWidth;
			setZoom(measureZoom());
		};
		setZoom(measureZoom());
		addEventListener('resize', onResize);
		const ro = new ResizeObserver(onResize);
		ro.observe(document.documentElement);
		return () => {
			removeEventListener('resize', onResize);
			ro.disconnect();
		};
	}, []);

	const barRef = useRef<HTMLDivElement>(null);
	useEffect(() => {
		const onScroll = () => {
			const max = document.documentElement.scrollHeight - innerHeight;
			if (barRef.current) barRef.current.style.width = `${max > 0 ? (scrollY / max) * 100 : 0}%`;
		};
		onScroll();
		addEventListener('scroll', onScroll, { passive: true });
		return () => removeEventListener('scroll', onScroll);
	}, []);

	const [flow, setFlow] = useState<FlowModel | null>(null);
	useEffect(() => startHeroFlow(setFlow, () => !reducedMotion() && !document.hidden && inViewport('[data-shape="0"],[data-shape="6"]')), []);

	const [sec, setSec] = useState<{ step: number; ph: 0 | 1 | 2 }>({ step: 0, ph: 0 });
	useEffect(() => {
		let phases: ReturnType<typeof setTimeout>[] = [];
		const iv = setInterval(() => {
			if (document.hidden || reducedMotion() || !inViewport('[data-shape="1"],[data-shape="2"],[data-shape="3"],[data-shape="4"],[data-shape="5"]')) return;
			setSec((p) => ({ step: p.step + 1, ph: 0 }));
			phases = [setTimeout(() => setSec((p) => ({ ...p, ph: 1 })), 60), setTimeout(() => setSec((p) => ({ ...p, ph: 2 })), 1150)];
		}, 2000);
		return () => {
			clearInterval(iv);
			phases.forEach(clearTimeout);
		};
	}, []);

	const [cursor, setCursor] = useState<HowCursor>({ scen: 0, step: 0, phase: 0 });
	const [playing, setPlaying] = useState(true);
	const [timerEpoch, setTimerEpoch] = useState(0);
	const phaseTimer = useRef<ReturnType<typeof setTimeout>>(undefined);
	const howRef = useRef<HTMLElement>(null);
	const howVisible = useRef(true);

	// Every step renders once at phase 0 (packets at their source), then flips to phase 1 so the CSS transitions run.
	const go = useCallback((target: (c: HowCursor) => Pick<HowCursor, 'scen' | 'step'>) => {
		clearTimeout(phaseTimer.current);
		setCursor((c) => ({ ...target(c), phase: 0 }));
		phaseTimer.current = setTimeout(() => setCursor((c) => ({ ...c, phase: 1 })), 60);
	}, []);

	useEffect(() => {
		if (reducedMotion()) setPlaying(false);
		go(() => ({ scen: 0, step: 0 }));
		const io = new IntersectionObserver(
			(es) => {
				howVisible.current = es[0].isIntersecting;
			},
			{ threshold: 0.05 },
		);
		if (howRef.current) io.observe(howRef.current);
		return () => {
			io.disconnect();
			clearTimeout(phaseTimer.current);
		};
	}, [go]);

	useEffect(() => {
		const iv = setInterval(() => {
			if (playing && howVisible.current && !document.hidden) go((c) => nextCursor(c, 1));
		}, STEP_MS);
		return () => clearInterval(iv);
	}, [playing, timerEpoch, go]);

	const restartTimer = () => setTimerEpoch((n) => n + 1);

	const [copied, setCopied] = useState(false);
	const copyTimer = useRef<ReturnType<typeof setTimeout>>(undefined);
	useEffect(() => () => clearTimeout(copyTimer.current), []);
	const copyInstall = () => {
		void navigator.clipboard?.writeText(INSTALL_CMD);
		setCopied(true);
		clearTimeout(copyTimer.current);
		copyTimer.current = setTimeout(() => setCopied(false), 1600);
	};
	const copyLabel = copied ? t.copied : t.copy;

	const hv = flowView(flow, li);

	return (
		<div className='relative overflow-x-clip bg-ink'>
			<div ref={barRef} className='fixed top-0 left-0 z-30 h-0.5 w-0 bg-lime shadow-[0_0_12px_#c8f05a]' />
			<SiteHeader active='home' />

			<main className='relative z-1'>
				<Hero t={t} hv={hv} lang={lang} zoom={zoom.hero} copyLabel={copyLabel} onCopy={copyInstall} stats={stats} />

				<HowItWorks
					t={t}
					li={li}
					cursor={cursor}
					playing={playing}
					zoom={zoom.how}
					sectionRef={howRef}
					onTab={(scen) => {
						go(() => ({ scen, step: 0 }));
						restartTimer();
					}}
					onStep={(step) => {
						go((c) => ({ scen: c.scen, step }));
						restartTimer();
					}}
					onPrev={() => {
						go((c) => nextCursor(c, -1));
						restartTimer();
					}}
					onNext={() => {
						go((c) => nextCursor(c, 1));
						restartTimer();
					}}
					onTogglePlay={() => {
						setPlaying((p) => !p);
						restartTimer();
					}}
				/>

				<Features t={t} views={sectionsView(sec.step, sec.ph, li)} zoom={zoom.sec} />

				<section data-shape='6' className={`mx-auto flex max-w-[1312px] flex-col items-center justify-end gap-7 pb-[110px] text-center ${padX}`}>
					<div className='flex flex-1 flex-col items-center justify-center gap-7 pt-[110px]'>
						<div className='relative max-w-[420px] rounded-2xl border bg-ink-4 px-[18px] py-2.5 text-center font-cjk text-[15px] text-fg transition-[border-color] duration-300' style={{ borderColor: hv.bubbleBorder }}>
							{hv.mascotLine}
							<span className='absolute -bottom-[7px] left-1/2 size-3 -translate-x-1/2 rotate-45 border-r border-b bg-ink-4' style={{ borderColor: hv.bubbleBorder }} />
						</div>
						<div className='mb-[170px] h-[150px] w-40 origin-top scale-200'>
							<Mascot mood={hv.mood} ver={hv.ver} lang={lang} />
						</div>
						<span className='pb-9 font-mono text-xs tracking-[0.12em] text-fg-6'>{hv.mascotTag}</span>
					</div>
					<h2 className='m-0 max-w-[800px] text-[clamp(34px,4vw,56px)] leading-[1.05] font-bold tracking-[-0.04em]'>{t.endT}</h2>
					<div className='flex flex-wrap justify-center gap-3'>
						<Link to='/docs' className={ctaPrimary}>
							{t.ctaDocs}
						</Link>
						<button type='button' onClick={copyInstall} className='flex max-w-full cursor-pointer items-center gap-3 rounded-full border border-fg/20 bg-ink/40 px-[22px] py-[15px] text-left font-mono text-sm text-fg backdrop-blur-[10px] hover:border-lime'>
							<span className='text-lime'>$</span>
							<Command text='go get github.com/yldm-tech/nodra' />
							<span className='shrink-0 text-xs text-fg-6'>{copyLabel}</span>
						</button>
					</div>
				</section>
			</main>

			<HomeFooter lang={lang} />
		</div>
	);
}

const footHead = 'font-mono text-[11px] tracking-[0.14em] text-fg-5';
const footCol = 'flex flex-col gap-2.5 text-sm';
const footLink = 'text-fg-3 hover:text-fg';
const exampleLink = `${footLink} font-mono text-[13px]`;

function HomeFooter({ lang }: { lang: Lang }) {
	const t = T[lang];
	return (
		<footer className='relative z-1 border-t border-fg/8 bg-ink'>
			<div className={`mx-auto grid max-w-[1312px] grid-cols-[repeat(auto-fit,minmax(180px,1fr))] gap-9 pt-14 pb-10 ${padX}`}>
				<div className='col-span-2 flex min-w-0 flex-col gap-3.5'>
					<span className='text-[22px] font-bold tracking-[-0.035em]'>nodra</span>
					<span className='max-w-[360px] text-sm leading-[1.7] text-fg-4'>{t.ftTag}</span>
					<span className='font-mono text-xs text-fg-5'>Go 1.27.3+ · MyGo v0.7.2 · MIT</span>
				</div>
				<nav aria-label={t.ftDocs} className={footCol}>
					<span className={footHead}>{t.ftDocs}</span>
					<Link to='/docs' hash='quickstart' className={footLink}>
						{t.ftQuick}
					</Link>
					<Link to='/docs' hash='store' className={footLink}>
						API
					</Link>
					<Link to='/docs' hash='persistence' className={footLink}>
						{t.ftPersist}
					</Link>
					<Link to='/docs' hash='mapping' className={footLink}>
						{t.ftMigrate}
					</Link>
				</nav>
				<nav aria-label={t.ftEx} className={footCol}>
					<span className={footHead}>{t.ftEx}</span>
					<a href={`${REPO_URL}/blob/main/examples/workbench/README.md`} className={exampleLink}>
						workbench
					</a>
					<a href={`${REPO_URL}/blob/main/examples/mygo/main.go`} className={exampleLink}>
						mygo
					</a>
					<a href={`${REPO_URL}/blob/main/examples/persistent/main.go`} className={exampleLink}>
						persistent
					</a>
					<a href={`${REPO_URL}/blob/main/examples/devtools/main.go`} className={exampleLink}>
						devtools
					</a>
				</nav>
				<nav aria-label={t.ftProj} className={footCol}>
					<span className={footHead}>{t.ftProj}</span>
					<a href={REPO_URL} className={footLink}>
						GitHub
					</a>
					<a href={`${REPO_URL}/blob/main/CHANGELOG.md`} className={footLink}>
						{t.ftChange}
					</a>
					<a href={`${REPO_URL}/blob/main/CONTRIBUTING.md`} className={footLink}>
						{t.ftContrib}
					</a>
					<a href={`${REPO_URL}/blob/main/SECURITY.md`} className={footLink}>
						{t.ftSec}
					</a>
				</nav>
			</div>
			<div className={`mx-auto flex max-w-[1312px] flex-wrap justify-between gap-4 border-t border-fg/6 pt-[18px] pb-7 text-xs text-fg-5 ${padX}`}>
				<span>© 2026 yldm-tech · MIT License</span>
				<span>{t.ftMade}</span>
			</div>
		</footer>
	);
}
