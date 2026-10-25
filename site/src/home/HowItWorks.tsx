import type { Ref } from 'react';
import type { Copy } from './copy';
import { type HowCursor, STEP_MS, howView } from './how';
import { Dashes, Packet, WindowDots, kicker } from './ui';

interface Props {
	t: Copy;
	li: 0 | 1;
	cursor: HowCursor;
	playing: boolean;
	zoom: number;
	sectionRef: Ref<HTMLElement>;
	onTab: (scen: number) => void;
	onStep: (step: number) => void;
	onPrev: () => void;
	onNext: () => void;
	onTogglePlay: () => void;
}

const roundBtn = 'cursor-pointer rounded-full border py-[9px] font-mono text-[13px]';

export function HowItWorks({ t, li, cursor, playing, zoom, sectionRef, onTab, onStep, onPrev, onNext, onTogglePlay }: Props) {
	const { v, cur, tabs, steps, codeLines } = howView(cursor, li);
	const running = playing && cursor.phase === 1;

	return (
		<section
			id='how'
			ref={sectionRef}
			className='relative flex flex-col gap-10 border-y border-fg/8 bg-ink-2 px-[max(36px,calc((100%-1240px)/2))] py-[110px] max-[600px]:px-5'
		>
			<div className='flex flex-wrap items-end justify-between gap-7'>
				<div className='flex max-w-[760px] flex-col gap-3.5'>
					<span className={kicker}>{t.howK}</span>
					<h2 className='m-0 text-[clamp(36px,4.6vw,68px)] leading-none font-bold tracking-[-0.045em]'>{t.howTitle}</h2>
				</div>
				<div role='tablist' className='flex flex-wrap gap-1.5 rounded-full border border-fg/12 bg-white/2 p-[5px]'>
					{tabs.map((tb, k) => (
						<button
							key={tb.label}
							type='button'
							role='tab'
							aria-selected={tb.active}
							onClick={() => onTab(k)}
							className={`cursor-pointer rounded-full border-0 px-[18px] py-2.5 font-mono text-[13px] transition-colors duration-250 ${tb.active ? 'bg-lime text-ink' : 'bg-transparent text-fg-4'}`}
						>
							{tb.label}
						</button>
					))}
				</div>
			</div>

			<div className='flex flex-wrap items-start gap-8'>
				<div className='order-2 flex max-w-[400px] flex-[1_1_300px] flex-col gap-1'>
					{steps.map((st, k) => (
						<button
							key={st.num}
							type='button'
							aria-current={st.active ? 'step' : undefined}
							onClick={() => onStep(k)}
							className={`grid cursor-pointer grid-cols-[40px_minmax(0,1fr)] gap-3 rounded-[14px] border px-4 py-3.5 text-left font-sans text-fg transition-[background,border-color] duration-300 ${st.active ? 'border-lime/35 bg-lime/6' : 'border-transparent bg-transparent'}`}
						>
							<span className={`pt-[3px] font-mono text-[13px] ${st.active ? 'text-lime' : st.done ? 'text-fg-6' : 'text-[#7a7d74]'}`}>{st.num}</span>
							<span className='flex flex-col gap-2'>
								<span className={`text-lg font-bold tracking-[-0.015em] ${st.active ? 'text-fg' : st.done ? 'text-fg-4' : 'text-fg-5'}`}>{st.title}</span>
								{st.active && (
									<>
										<span className='text-[14.5px] leading-[1.65] text-pretty text-fg-3'>{st.desc}</span>
										<span className='block h-0.5 overflow-hidden rounded-xs bg-fg/10'>
											<span className='block h-full bg-lime' style={{ width: running ? '100%' : '0%', transition: running ? `width ${STEP_MS - 100}ms linear` : 'none' }} />
										</span>
									</>
								)}
							</span>
						</button>
					))}
					<div className='flex gap-2 px-4 pt-3.5 font-mono text-[13px]'>
						<button type='button' aria-label='Previous step' onClick={onPrev} className={`${roundBtn} px-3.5 border-fg/18 bg-transparent text-fg hover:border-lime`}>
							←
						</button>
						<button type='button' onClick={onTogglePlay} className={`${roundBtn} border-lime/50 bg-lime/10 px-[18px] text-lime hover:bg-lime hover:text-ink`}>
							{playing ? t.pause : t.play}
						</button>
						<button type='button' aria-label='Next step' onClick={onNext} className={`${roundBtn} px-3.5 border-fg/18 bg-transparent text-fg hover:border-lime`}>
							→
						</button>
					</div>
				</div>

				<div className='order-1 flex min-w-0 flex-[999_1_620px] flex-col gap-3.5'>
					<div className='overflow-x-auto rounded-[22px] border border-fg/10 bg-[radial-gradient(ellipse_at_50%_45%,#12141a,#0a0b0e_70%)]'>
						<div className='relative h-[clamp(460px,38vw,600px)] min-w-[720px] font-mono' style={{ zoom }}>
							<Dashes
								dash='4 4'
								color='rgba(238,240,234,0.16)'
								lines={[
									[24, 55, 37, 50],
									[50, 70, 50, 78],
									[63, 48, 75, 12.5],
									[63, 48, 75, 35.5],
									[63, 48, 75, 58.5],
									[63, 48, 75, 81.5],
								]}
							/>

							<div className='absolute top-[4%] left-[3%] z-2 flex w-[32%] flex-col gap-1.5'>
								<span className='text-[11px] tracking-[0.12em] text-lime'>
									{t.stepK} {cur.num} / {cur.total}
								</span>
								<span className='font-sans text-xl font-bold text-fg'>{cur.title}</span>
								<span className='font-cjk text-[13px] leading-[1.55] text-fg-4'>{cur.desc}</span>
							</div>

							<div aria-hidden='true'>
								<div className='absolute top-[36%] left-[3%] flex h-[38%] w-[21%] flex-col overflow-hidden rounded-[14px] border bg-panel-2 transition-[border-color,box-shadow] duration-400' style={{ borderColor: v.winBorder, boxShadow: v.winShadow }}>
									<div className='flex items-center gap-[5px] border-b border-fg/8 px-[11px] py-[9px] text-[11px] text-fg-6'>
										<WindowDots />
										<span className='pl-1.5'>{t.mygoWin}</span>
									</div>
									<div className='flex flex-1 flex-col items-center justify-center gap-2.5'>
										<span className='text-[11px] tracking-[0.1em] text-fg-6'>{t.countK}</span>
										<span className='font-display text-[44px] leading-none font-bold text-fg'>{v.win}</span>
										<span className='rounded-lg px-3.5 py-[5px] text-xs transition-colors duration-300' style={{ background: v.btnBg, color: v.btnFg }}>
											+1
										</span>
									</div>
								</div>

								<div
									className='absolute top-[26%] left-[37%] flex h-[44%] w-[26%] flex-col gap-2.5 rounded-[18px] border bg-panel-4 px-[18px] py-4 transition-[border-color,box-shadow] duration-600'
									style={{ borderColor: v.storeBorder, boxShadow: v.storeShadow }}
								>
									<div className='flex items-center justify-between text-xs'>
										<span className='text-fg-4'>Store[State]</span>
										<span className='rounded-md bg-amber px-[7px] py-[3px] text-[10px] tracking-[0.1em] whitespace-nowrap text-ink transition-opacity duration-300' style={{ opacity: v.lockOp }}>
											🔒 {t.locked}
										</span>
									</div>
									<span className='text-[42px] leading-none font-bold tracking-[-0.02em] text-lime'>v{v.ver}</span>
									<div className='flex flex-col gap-1 border-t border-fg/8 pt-2.5 text-[13px] text-fg-2'>
										<span>
											count: <span className='font-bold text-fg'>{v.count}</span>
										</span>
										<span>
											title: <span className='text-amber'>"{v.title}"</span>
										</span>
									</div>
									<span className='self-start rounded-md border border-dashed border-sky px-2 py-[3px] text-[10px] tracking-[0.1em] text-sky transition-opacity duration-300' style={{ opacity: v.batchOp }}>
										{t.batchK} {v.batch}
									</span>
								</div>

								<div
									className='absolute top-[78%] left-[40%] flex h-[16%] w-[20%] flex-col items-center justify-center gap-1 rounded-xl border border-dashed transition-[opacity,transform,border-color,background] duration-400'
									style={{ borderColor: v.candBorder, background: v.candBg, opacity: v.candOp, transform: `scale(${v.candScale})` }}
								>
									<span className='text-[10px] tracking-[0.1em] text-fg-4'>{t.candK}</span>
									<span className='text-[15px] font-bold' style={{ color: v.candColor }}>
										count: {v.cand}
									</span>
								</div>

								{v.tasks.map((tk) => (
									<div
										key={tk.label}
										className='absolute top-[78%] flex h-[16%] w-[17%] flex-col items-center justify-center gap-1 rounded-xl border bg-panel-2 transition-[opacity,border-color] duration-400'
										style={{ left: tk.left, borderColor: tk.color }}
									>
										<span className='text-xs text-fg' style={{ textDecoration: tk.deco }}>
											{tk.label}
										</span>
										<span className='text-[11px]' style={{ color: tk.color }}>
											{tk.state}
										</span>
									</div>
								))}

								{v.subs.map((sb) => (
									<div
										key={sb.top}
										className='absolute left-[75%] flex h-[17%] w-[22%] flex-col justify-center gap-1.5 rounded-xl border bg-panel-2 px-3 py-2.5 transition-[border-color,background] duration-400'
										style={{ top: sb.top, borderColor: sb.border }}
									>
										<span className='flex items-center gap-[7px] text-xs text-fg-2'>
											<span className='size-[7px] rounded-full transition-colors duration-300' style={{ background: sb.dot, boxShadow: `0 0 8px ${sb.dot}` }} />
											{sb.name}
										</span>
										<span className='text-[11.5px]' style={{ color: sb.noteColor, textDecoration: sb.deco }}>
											{sb.note}
										</span>
										{sb.hasBar && (
											<span className='block h-[3px] overflow-hidden rounded-[3px] bg-fg/8'>
												<span className='block h-full bg-lime transition-[width] duration-1600 ease-linear' style={{ width: sb.barW }} />
											</span>
										)}
									</div>
								))}

								{v.packets.map((pk, i) => (
									<Packet key={i} {...pk} large />
								))}

								<div className='absolute top-[6%] left-1/2 -translate-x-1/2 rounded-[10px] border border-coral bg-coral/12 px-4 py-[9px] text-[13px] whitespace-nowrap text-coral-soft transition-opacity duration-300' style={{ opacity: v.errOp }}>
									✗ {v.err}
								</div>
							</div>
						</div>
					</div>

					<div className='overflow-x-auto rounded-2xl border border-fg/10 bg-ink-3 py-3.5 font-mono text-[13px] leading-normal'>
						{codeLines.map((cl) => (
							<div key={cl.n} className={`grid grid-cols-[40px_minmax(0,1fr)] border-l-2 py-[3px] transition-colors duration-300 ${cl.active ? 'border-lime bg-lime/8' : 'border-transparent bg-transparent'}`}>
								<span className='pr-3.5 text-right text-fg-7'>{cl.n}</span>
								<span className={`whitespace-pre transition-colors duration-300 ${cl.active ? 'text-fg' : cl.comment ? 'text-fg-5' : 'text-[#8a8d84]'}`}>{cl.text}</span>
							</div>
						))}
					</div>
				</div>
			</div>
		</section>
	);
}
