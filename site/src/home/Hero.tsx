import { Link } from '@tanstack/react-router';
import { Mascot } from '../components/Mascot';
import { INSTALL_CMD, type Lang } from '../lib/lang';
import type { AdoptionStats as Stats } from '../lib/stats';
import { AdoptionStats } from './AdoptionStats';
import type { Copy } from './copy';
import type { flowView } from './flow';
import { Command, Dashes, Packet, kicker, padX } from './ui';

type HeroView = ReturnType<typeof flowView>;

export const ctaPrimary = 'rounded-full bg-lime px-7 py-4 text-base font-bold text-ink shadow-[0_0_40px_-6px_rgba(200,240,90,0.7)] hover:bg-white hover:text-ink';

export function Hero({ t, hv, lang, zoom, copyLabel, onCopy, stats }: { t: Copy; hv: HeroView; lang: Lang; zoom: number; copyLabel: string; onCopy: () => void; stats: Promise<Stats> }) {
	return (
		<section data-shape='0' className={`mx-auto flex min-h-screen max-w-[1312px] flex-wrap items-center gap-10 pt-[110px] pb-12 ${padX}`}>
			<div className='relative flex min-w-0 flex-[1_1_420px] flex-col gap-[26px]'>
				<div className='pointer-events-none absolute -inset-y-[60px] -right-20 -left-[120px] -z-1 bg-[radial-gradient(ellipse_at_35%_50%,rgba(6,7,9,0.92)_0%,rgba(6,7,9,0.75)_45%,rgba(6,7,9,0)_75%)]' />
				<span className={kicker}>{t.kicker}</span>
				<h1 className='m-0 flex flex-col text-[clamp(48px,6.2vw,104px)] leading-none font-bold tracking-[-0.05em]'>
					<span>{t.h1a}</span>
					<span className='text-lime'>{t.h1b}</span>
				</h1>
				<p className='m-0 max-w-[520px] text-[clamp(16px,1.4vw,19px)] leading-[1.65] text-fg-3'>{t.sub}</p>
				<div className='flex flex-wrap items-center gap-3'>
					<Link to='/docs' className={ctaPrimary}>
						{t.ctaDocs}
					</Link>
					<a href='#how' className='rounded-full border border-fg/25 px-6 py-[15px] text-base font-medium text-fg hover:border-lime hover:text-lime'>
						{t.ctaHow}
					</a>
				</div>
				<button type='button' onClick={onCopy} className='flex max-w-full cursor-pointer items-center gap-3 self-start border-0 bg-transparent p-0 text-left font-mono text-sm text-fg-4 hover:text-fg'>
					<span className='text-lime'>$</span>
					<Command text={INSTALL_CMD} />
					<span className='shrink-0 text-xs text-fg-5' aria-live='polite'>
						{copyLabel}
					</span>
				</button>
				<AdoptionStats stats={stats} lang={lang} />
			</div>

			<div className='flex min-w-0 flex-[1.35_1_560px] flex-col gap-3'>
				<div className='relative mt-10 h-24'>
					<div className='absolute -bottom-[30px] left-[calc(46%-120px)] z-5 origin-bottom scale-115'>
						<Mascot mood={hv.mood} ver={hv.ver} lang={lang} />
					</div>
					<div className='absolute bottom-[18px] left-[calc(46%+80px)] flex flex-col gap-0.5 font-mono'>
						<span className='text-[13px] font-bold text-fg'>Dra</span>
						<span className='text-[11px] text-fg-6'>{hv.mascotLine}</span>
					</div>
				</div>
				<div className='overflow-x-auto rounded-3xl border border-fg/10 bg-[rgba(10,11,14,0.82)] backdrop-blur-[6px]'>
					<div aria-hidden='true' className='relative h-[clamp(400px,34vw,480px)] min-w-[560px] font-mono' style={{ zoom }}>
						<Dashes
							lines={[
								[18, 20, 31, 50],
								[18, 50, 31, 50],
								[18, 80, 31, 50],
								[61, 50, 70, 12],
								[61, 50, 70, 37],
								[61, 50, 70, 62],
								[61, 50, 70, 87],
							]}
						/>

						{hv.gors.map((g) => (
							<div
								key={g.name}
								className='absolute left-[2%] flex h-[15%] w-[16%] -translate-y-1/2 flex-col justify-center gap-1 rounded-xl border bg-panel px-2.5 transition-[border-color,box-shadow] duration-300'
								style={{ top: g.top, borderColor: g.border, boxShadow: g.shadow }}
							>
								<span className='text-[11px] text-fg'>{g.name}</span>
								<span className='text-[10px]' style={{ color: g.subColor }}>
									{g.sub}
								</span>
							</div>
						))}

						<div
							className='absolute top-[24%] left-[31%] flex h-[52%] w-[30%] flex-col gap-2 rounded-[18px] border-[1.5px] bg-panel-3 px-4 py-3.5 transition-[border-color,box-shadow] duration-500'
							style={{ borderColor: hv.storeBorder, boxShadow: hv.storeShadow }}
						>
							<div className='flex items-center justify-between text-[11px]'>
								<span className='text-fg-4'>Store[State]</span>
								<span className='rounded-md bg-amber px-[7px] py-[3px] text-[10px] font-bold whitespace-nowrap text-ink transition-opacity duration-250' style={{ opacity: hv.lockOp }}>
									{t.locked}
								</span>
							</div>
							<span className='text-[clamp(28px,2.6vw,38px)] leading-none font-bold tracking-[-0.02em] text-lime'>v{hv.ver}</span>
							<div className='flex flex-col gap-[3px] border-t border-fg/8 pt-2 text-xs text-fg-4'>
								<span>
									count: <span className='font-bold text-fg'>{hv.count}</span>
								</span>
								<span>
									title: <span className='text-amber'>"{hv.title}"</span>
								</span>
							</div>
							<span
								className='mt-auto overflow-hidden rounded-lg border border-dashed px-2 py-[5px] text-[10.5px] text-ellipsis whitespace-nowrap transition-opacity duration-250'
								style={{ borderColor: hv.candColor, color: hv.candColor, opacity: hv.candOp }}
							>
								{hv.cand}
							</span>
						</div>

						{hv.subs.map((s) => (
							<div
								key={s.top}
								className='absolute left-[70%] flex h-[20%] w-[28%] -translate-y-1/2 flex-col justify-center gap-1 rounded-xl border bg-panel px-[11px] py-2 transition-[border-color,box-shadow] duration-350'
								style={{ top: s.top, borderColor: s.border, boxShadow: s.shadow }}
							>
								<span className='text-[10.5px] text-fg-4'>{s.name}</span>
								<span className='overflow-hidden text-sm font-bold text-ellipsis whitespace-nowrap transition-colors duration-300' style={{ color: s.valColor }}>
									{s.val}
								</span>
								{s.hasBar && (
									<span className='block h-[3px] overflow-hidden rounded-[3px] bg-fg/8'>
										<span className='block h-full' style={{ background: s.barColor, width: s.barW, transition: s.barTr }} />
									</span>
								)}
								<span className='overflow-hidden text-[10px] text-ellipsis whitespace-nowrap' style={{ color: s.noteColor, textDecoration: s.deco }}>
									{s.note}
								</span>
							</div>
						))}

						{hv.pk.map((p, i) => (
							<Packet key={i} {...p} />
						))}
					</div>
				</div>
				<div className='flex min-h-[22px] items-center gap-2.5 px-1.5 font-cjk text-sm'>
					<span className='size-2 flex-none rounded-full' style={{ background: hv.capColor, boxShadow: `0 0 10px ${hv.capColor}` }} />
					<span className='text-fg-2'>{hv.cap}</span>
				</div>
			</div>
		</section>
	);
}
