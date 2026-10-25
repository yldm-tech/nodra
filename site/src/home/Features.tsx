import type { ReactNode } from 'react';
import { highlightGo } from '../lib/highlight';
import { type Copy, MYGO_SNIPPET } from './copy';
import type { sectionsView } from './sections';
import { Chips, Dashes, FeatureText, Packet, VizFrame, WindowDots, padX } from './ui';

type Views = ReturnType<typeof sectionsView>;

const mygoCode = highlightGo(MYGO_SNIPPET);

function Row({ shape, children }: { shape: number; children: ReactNode }) {
	return (
		<section data-shape={shape} className={`mx-auto grid min-h-[92vh] max-w-[1312px] grid-cols-[repeat(auto-fit,minmax(min(100%,520px),1fr))] items-center gap-10 py-[72px] ${padX}`}>
			{children}
		</section>
	);
}

const storeCard = 'absolute flex flex-col gap-1.5 rounded-2xl border-[1.5px] bg-panel-4 px-3.5 py-3 transition-[border-color,box-shadow] duration-400';
const statLabel = 'text-[10px] tracking-[0.12em] text-fg-6';

export function Features({ t, views, zoom }: { t: Copy; views: Views; zoom: number }) {
	const { v1, v2, v3, v4, v5 } = views;
	return (
		<>
			<Row shape={1}>
				<FeatureText kicker={`01 — ${t.s1k}`} title={t.s1t} body={t.s1p}>
					<div className='flex flex-col gap-2 font-mono text-[13px]'>
						{[
							[t.a1, t.a1n],
							[t.a2, t.a2n],
							[t.a3, t.a3n],
						].map(([a, n]) => (
							<span key={a} className='flex justify-between gap-3 border-t border-fg/10 pt-2.5'>
								<span>{a}</span>
								<span className='text-fg-6'>{n}</span>
							</span>
						))}
					</div>
				</FeatureText>
				<VizFrame zoom={zoom} title={v1.title} cap={v1.cap} capColor={v1.capColor}>
					<Dashes
						lines={[
							[24, 26, 44, 50],
							[24, 50, 44, 50],
							[24, 74, 44, 50],
							[68, 50, 74, 50],
						]}
					/>
					{v1.lanes.map((l) => (
						<div key={l.name} className='absolute left-[5%] flex h-[13%] w-[19%] -translate-y-1/2 flex-col justify-center gap-[3px] rounded-[10px] border bg-panel px-2.5 transition-[border-color] duration-300' style={{ top: l.top, borderColor: l.border }}>
							<span className='text-[11px] text-fg'>{l.name}</span>
							<span className='text-[10px]' style={{ color: l.color }}>
								{l.status}
							</span>
						</div>
					))}
					<div className={`${storeCard} top-[32%] left-[44%] h-[36%] w-[24%]`} style={{ borderColor: v1.storeBorder, boxShadow: v1.storeShadow }}>
						<div className='flex items-center justify-between text-[10.5px]'>
							<span className='text-fg-4'>Store[T]</span>
							<span className='rounded-[5px] bg-amber px-1.5 py-0.5 text-[9.5px] font-bold text-ink transition-opacity duration-200' style={{ opacity: v1.lockOp }}>
								{t.locked}
							</span>
						</div>
						<span className='text-[clamp(22px,2.2vw,30px)] font-bold tracking-[-0.02em] text-lime'>{v1.ver}</span>
						<span className='text-[10.5px] text-fg-6'>{v1.owner}</span>
					</div>
					<div className='absolute top-[20%] left-[75%] flex w-[21%] flex-col gap-1.5'>
						<span className={statLabel}>{t.logK}</span>
						{v1.log.map((g, j) => (
							<div key={j} className='flex justify-between rounded-[7px] px-2 py-[5px] text-[11.5px] transition-colors duration-300' style={{ background: g.bg, color: g.color }}>
								<span>{g.v}</span>
								<span>{g.from}</span>
							</div>
						))}
					</div>
					<Packet {...v1.pk} bg='#c8f05a' />
				</VizFrame>
			</Row>

			<Row shape={2}>
				<FeatureText swap kicker={`02 — ${t.s2k}`} title={t.s2t} body={t.s2p}>
					<Chips items={['Undo', 'Redo', 'BeginGroup', 'TrackWith']} />
				</FeatureText>
				<VizFrame zoom={zoom} title={v2.title} cap={v2.cap} capColor={v2.capColor}>
					<span className='absolute top-[52px] left-6 text-[clamp(18px,1.8vw,24px)] font-bold transition-colors duration-300' style={{ color: v2.actColor }}>
						{v2.act}
					</span>
					<div className='absolute inset-x-[12%] top-[52%] h-0.5 bg-fg/12' />
					{v2.dots.map((d) => (
						<div
							key={d.x}
							className='absolute top-[52%] flex size-[34px] items-center justify-center rounded-full border-2 text-[10px] font-bold transition-all duration-500 ease-[cubic-bezier(.2,.7,.2,1)]'
							style={{ left: d.x, transform: `translate(-50%,-50%) scale(${d.scale})`, borderColor: d.border, background: d.bg, opacity: d.op, boxShadow: d.shadow, color: d.color }}
						>
							{d.label}
						</div>
					))}
					<div className='absolute top-[calc(52%-46px)] flex -translate-x-1/2 flex-col items-center gap-0.5 transition-[left] duration-600 ease-[cubic-bezier(.2,.7,.2,1)]' style={{ left: v2.headX }}>
						<span className='text-[10px] font-bold tracking-[0.1em] text-lime'>{t.headK}</span>
						<span className='size-0 border-x-[6px] border-t-[7px] border-x-transparent border-t-lime' />
					</div>
					<div className='absolute top-[calc(52%+30px)] -translate-x-1/2 text-[10.5px] whitespace-nowrap text-coral transition-opacity duration-400' style={{ left: v2.ghostX, opacity: v2.ghostOp }}>
						✗ {v2.ghost}
					</div>
					<div className='absolute inset-x-6 top-[72%] flex flex-wrap gap-6 text-xs'>
						<span className='whitespace-nowrap' style={{ color: v2.undoColor }}>
							CanUndo {v2.canUndo} · ×{v2.undoN}
						</span>
						<span className='whitespace-nowrap' style={{ color: v2.redoColor }}>
							CanRedo {v2.canRedo} · ×{v2.redoN}
						</span>
					</div>
				</VizFrame>
			</Row>

			<Row shape={3}>
				<FeatureText kicker={`03 — ${t.s3k}`} title={t.s3t} body={t.s3p}>
					<div className='flex flex-col gap-1.5 font-mono text-[13px]'>
						<span className='rounded-[10px] bg-lime px-3.5 py-[11px] font-bold text-ink'>WithConflictDetection</span>
						{['FileStorage · Journal · Durable', 'AESGCMCodec', 'VersionedJSONCodec'].map((x) => (
							<span key={x} className='rounded-[10px] border border-fg/14 px-3.5 py-[11px]'>
								{x}
							</span>
						))}
					</div>
				</FeatureText>
				<VizFrame zoom={zoom} title={v3.title} cap={v3.cap} capColor={v3.capColor}>
					{v3.rows.map((r) => (
						<div key={r.name} className='absolute right-[5%] left-[14%] flex h-[11%] items-center justify-between gap-3 rounded-[10px] border px-3 transition-[border-color,background] duration-300' style={{ top: r.top, borderColor: r.border, background: r.bg }}>
							<span className='text-[11.5px] whitespace-nowrap' style={{ color: r.nameColor }}>
								{r.name}
							</span>
							<span className='overflow-hidden text-[11px] text-ellipsis whitespace-nowrap transition-opacity duration-400' style={{ color: r.outColor, opacity: r.outOp }}>
								{r.out}
							</span>
						</div>
					))}
					<span className='absolute left-[7.5%] size-4 -translate-1/2 rounded-full bg-lime shadow-[0_0_16px_#c8f05a] transition-[top,opacity] duration-900 ease-[cubic-bezier(.6,0,.2,1)]' style={{ top: v3.pkY, opacity: v3.pkOp }} />
					<div className='absolute inset-y-[20%] left-[7.5%] w-0.5 -translate-x-1/2 bg-fg/10' />
				</VizFrame>
			</Row>

			<Row shape={4}>
				<FeatureText swap kicker={`04 — ${t.s4k}`} title={t.s4t} body={t.s4p}>
					<Chips items={['LatestWins', 'QueueAll', 'Debounce', 'Retry', 'Scope']} />
				</FeatureText>
				<VizFrame zoom={zoom} title={v4.title} cap={v4.cap} capColor={v4.capColor}>
					{v4.runs.map((r) => (
						<div key={r.name} className='absolute right-[36%] left-6 flex flex-col gap-[7px]' style={{ top: r.top }}>
							<div className='flex justify-between gap-2 text-[11.5px]'>
								<span className='whitespace-pre text-fg'>{r.name}</span>
								<span style={{ color: r.color }}>{r.status}</span>
							</div>
							<div className='h-2.5 overflow-hidden rounded-md bg-fg/7'>
								<div className='h-full rounded-md' style={{ width: r.w, background: r.color, transition: r.tr }} />
							</div>
						</div>
					))}
					<div className={`${storeCard} top-[26%] right-6 h-[40%] w-[28%]`} style={{ borderColor: v4.storeBorder, boxShadow: v4.storeShadow }}>
						<span className='text-[10.5px] text-fg-4'>Store[T]</span>
						<span className='text-[clamp(20px,2vw,26px)] font-bold text-lime'>{v4.ver}</span>
						<span className='text-[11px] text-fg-4'>
							{t.resultsK} <span className='text-amber'>{v4.res}</span>
						</span>
						<span className='text-[10.5px]' style={{ color: v4.noteColor }}>
							{v4.note}
						</span>
					</div>
				</VizFrame>
			</Row>

			<Row shape={5}>
				<FeatureText kicker='05 — MyGo · DevTools' title={t.s5t} body={t.s5p}>
					<pre className='m-0 overflow-x-auto rounded-[14px] border border-fg/8 bg-black/50 px-5 py-[18px] font-mono text-[13px] leading-[1.75] text-fg-2'>{mygoCode}</pre>
				</FeatureText>
				<VizFrame zoom={zoom} title={v5.title} cap={v5.cap} capColor={v5.capColor}>
					<div className='absolute top-[22%] left-[8%] flex h-[52%] w-[46%] flex-col overflow-hidden rounded-[14px] border-[1.5px] bg-panel-2 transition-[border-color,box-shadow] duration-400' style={{ borderColor: v5.winBorder, boxShadow: v5.winShadow }}>
						<div className='flex items-center gap-[5px] border-b border-fg/8 px-2.5 py-2 text-[10.5px] text-fg-6'>
							<WindowDots />
							<span className='pl-1.5'>MyGo · selector(s.Count)</span>
						</div>
						<div className='flex flex-1 flex-col items-center justify-center gap-1.5'>
							<span className={statLabel}>{t.countK}</span>
							<span className='font-display text-[52px] leading-none font-bold transition-colors duration-300' style={{ color: v5.numColor }}>
								{v5.count}
							</span>
						</div>
					</div>
					<div className='absolute top-[22%] right-6 left-[60%] flex flex-col gap-3.5'>
						<div className='flex flex-col gap-[3px]'>
							<span className={statLabel}>{t.commitsK}</span>
							<span className='text-[28px] font-bold text-fg'>{v5.commits}</span>
						</div>
						<div className='flex flex-col gap-[3px]'>
							<span className={statLabel}>{t.redrawsK}</span>
							<span className='text-[28px] font-bold text-lime'>{v5.redraws}</span>
						</div>
						<span className='rounded-lg px-2.5 py-1.5 text-[11.5px] transition-colors duration-300' style={{ background: v5.evBg, color: v5.evColor }}>
							{v5.ev}
						</span>
					</div>
				</VizFrame>
			</Row>
		</>
	);
}
