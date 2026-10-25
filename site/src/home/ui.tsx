import type { ReactNode } from 'react';

export const kicker = 'font-mono text-xs tracking-[0.16em] text-lime';
export const padX = 'px-9 max-[600px]:px-5';

// Dashed connector lines drawn in a 0–100 viewBox stretched over the diagram.
export function Dashes({ lines, dash = '3 5', color = 'rgba(238,240,234,0.14)' }: { lines: [number, number, number, number][]; dash?: string; color?: string }) {
	return (
		<svg viewBox='0 0 100 100' preserveAspectRatio='none' aria-hidden='true' className='absolute inset-0 size-full'>
			{lines.map(([x1, y1, x2, y2]) => (
				<line key={`${x1}-${y1}-${x2}-${y2}`} x1={x1} y1={y1} x2={x2} y2={y2} stroke={color} strokeWidth='1' vectorEffect='non-scaling-stroke' strokeDasharray={dash} />
			))}
		</svg>
	);
}

// A message travelling between diagram nodes; position, colour and transition change every frame, so they stay inline.
export function Packet({ x, y, op, tr, bg, label, large = false }: { x: string; y: string; op: number; tr: string; bg: string; label: string; large?: boolean }) {
	return (
		<span
			className={`absolute z-3 -translate-1/2 rounded-full font-bold whitespace-nowrap text-ink ${large ? 'px-[11px] py-1.5 text-xs' : 'px-2.5 py-[5px] text-[10.5px]'}`}
			style={{ left: x, top: y, opacity: op, transition: tr, background: bg, boxShadow: `0 0 ${large ? 22 : 18}px ${bg}` }}
		>
			{label}
		</span>
	);
}

// The 400px illustration frame used by every feature section: a caption on top, a status line at the bottom.
export function VizFrame({ zoom, title, cap, capColor, children }: { zoom: number; title: string; cap: string; capColor: string; children: ReactNode }) {
	return (
		<div role='img' aria-label={`${title}. ${cap}`} className='relative h-[400px] min-w-[520px] overflow-hidden rounded-3xl border border-fg/10 bg-[radial-gradient(ellipse_at_50%_40%,#12151b,#0a0b0e_75%)] font-mono' style={{ zoom }}>
			<span className='absolute top-5 left-6 text-[11px] tracking-[0.14em] text-fg-6'>{title}</span>
			<span className='absolute right-6 bottom-[18px] left-6 font-cjk text-[13.5px] transition-colors duration-300' style={{ color: capColor }}>
				{cap}
			</span>
			{children}
		</div>
	);
}

export function FeatureText({ swap = false, kicker: k, title, body, children }: { swap?: boolean; kicker: string; title: string; body: string; children: ReactNode }) {
	return (
		<div className={`flex min-w-0 flex-col gap-[22px] rounded-3xl border border-fg/10 bg-ink-3 p-9 ${swap ? 'min-[1152px]:order-2' : ''}`}>
			<span className={kicker}>{k}</span>
			<h2 className='m-0 text-[clamp(38px,4.4vw,64px)] leading-none font-bold tracking-[-0.04em]'>{title}</h2>
			<p className='m-0 text-base leading-[1.7] text-fg-3'>{body}</p>
			{children}
		</div>
	);
}

export function Chips({ items }: { items: string[] }) {
	return (
		<div className='flex flex-wrap gap-2 font-mono text-[13px]'>
			{items.map((item, i) => (
				<span key={item} className={`rounded-full border px-3 py-[7px] ${i === 0 ? 'border-lime/50 text-lime' : 'border-fg/18'}`}>
					{item}
				</span>
			))}
		</div>
	);
}

export function WindowDots() {
	return (
		<>
			<span className='size-[7px] rounded-full bg-coral' />
			<span className='size-[7px] rounded-full bg-amber' />
			<span className='size-[7px] rounded-full bg-fg-6' />
		</>
	);
}

// A shell command that may wrap only after a slash, so narrow screens never split a word.
export function Command({ text }: { text: string }) {
	const parts = text.split('/');
	return (
		<span className='min-w-0'>
			{parts.map((p, i) => (
				<span key={i}>
					{p}
					{i < parts.length - 1 && (
						<>
							/<wbr />
						</>
					)}
				</span>
			))}
		</span>
	);
}
