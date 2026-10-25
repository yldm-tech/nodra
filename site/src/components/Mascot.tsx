import type { Lang } from '../lib/lang';

export type Mood = 'idle' | 'focus' | 'happy' | 'sad';

const BODY_ANIM: Record<Mood, string> = { idle: 'animate-dra-bob', focus: 'animate-dra-bob', happy: 'animate-dra-hop', sad: 'animate-dra-shake' };
const LEAF: Record<Mood, string> = { idle: 'bg-[#b5dd5a]', focus: 'bg-[#e8e08a]', happy: 'bg-lime shadow-[0_0_14px_#c8f05a]', sad: 'bg-[#b9c7a0]' };
const TAG: Record<Mood, string> = { idle: 'border-[#3a3d35] text-fg-4', focus: 'border-amber text-amber', happy: 'border-lime text-lime', sad: 'border-coral text-coral' };

const leaf = 'absolute h-3 w-5 rounded-full border-[2.5px] border-dra-ink transition-[background,box-shadow] duration-300';
const paw = 'absolute rounded-full border-3 border-dra-ink bg-dra-cream';
const pawThin = 'absolute size-3.5 rounded-full border-[2.5px] border-dra-ink bg-dra-cream';
const eye = 'absolute top-[72px] h-3.5 w-[11px] animate-dra-blink rounded-full bg-dra-ink transition-[left] duration-300';
const glint = 'absolute top-0.5 left-0.5 size-1 rounded-full bg-white';
const lidUp = 'absolute h-[9px] w-[15px] rounded-t-[10px] border-3 border-b-0 border-dra-ink';
const lidDown = 'absolute h-[9px] w-[15px] rounded-b-[10px] border-3 border-t-0 border-dra-ink';
const smile = 'absolute top-[86px] h-1.5 w-[9px] rounded-b-md border-[2.5px] border-t-0 border-dra-ink';
const blush = 'absolute top-[88px] h-[9px] w-4 rounded-full bg-[rgba(255,140,130,0.6)]';
const tear = 'absolute rounded-[50%_50%_50%_50%/60%_60%_40%_40%] bg-dra-tear';

// Dra, the Nodra mascot: a 160x150 figure whose face and version tag follow the store's mood.
export function Mascot({ mood = 'idle', ver = '0042', lang = 'zh' }: { mood?: Mood; ver?: string; lang?: Lang }) {
	const happy = mood === 'happy';
	const sad = mood === 'sad';
	const focus = mood === 'focus';
	const calm = !happy && !sad;

	return (
		<div aria-hidden='true' className='relative h-[150px] w-40 font-mono'>
			<div key={mood} className={`absolute inset-0 origin-bottom ${BODY_ANIM[mood]}`}>
				<div className='absolute top-1 left-[78px] h-[30px] w-1.5 origin-bottom animate-dra-sway'>
					<div className='absolute top-2 left-px h-[22px] w-1 rounded-xs bg-dra-stem' />
					<div className={`${leaf} ${LEAF[mood]} top-0.5 -left-[17px] -rotate-[28deg]`} />
					<div className={`${leaf} ${LEAF[mood]} -top-0.5 left-[3px] rotate-[28deg]`} />
				</div>

				<div className={`${paw} top-[34px] left-[30px] size-7`}>
					<span className='absolute top-1.5 left-1.5 size-2.5 rounded-full bg-dra-blush' />
				</div>
				<div className={`${paw} top-[34px] left-[102px] size-7`}>
					<span className='absolute top-1.5 left-1.5 size-2.5 rounded-full bg-dra-blush' />
				</div>
				<div className={`${paw} top-[102px] left-[124px] size-[18px]`} />

				<div className='absolute top-10 left-6 h-[94px] w-[114px] rounded-[50%_50%_46%_46%/60%_60%_40%_40%] border-3 border-dra-ink bg-dra-cream shadow-[inset_-8px_-7px_0_rgba(200,190,160,0.35)]' />
				<div className='absolute top-[98px] left-[54px] h-[30px] w-[54px] rounded-full bg-dra-belly' />
				<div className={`${paw} top-[122px] left-12 h-[13px] w-6`} />
				<div className={`${paw} top-[122px] left-[90px] h-[13px] w-6`} />

				{calm && (
					<>
						<div className={`${eye} ${focus ? 'left-[59px]' : 'left-[55px]'}`}>
							<span className={glint} />
						</div>
						<div className={`${eye} ${focus ? 'left-[99px]' : 'left-[95px]'}`}>
							<span className={glint} />
						</div>
					</>
				)}
				{happy && (
					<>
						<div className={`${lidUp} top-[74px] left-[53px]`} />
						<div className={`${lidUp} top-[74px] left-[93px]`} />
					</>
				)}
				{sad && (
					<>
						<div className={`${lidDown} top-[76px] left-[53px]`} />
						<div className={`${lidDown} top-[76px] left-[93px]`} />
						<div className={`${tear} top-[86px] left-[55px] h-2.5 w-[7px] animate-dra-drop`} />
					</>
				)}

				<div className={`${blush} left-[42px]`} />
				<div className={`${blush} left-[104px]`} />

				{calm && !focus && (
					<>
						<div className={`${smile} left-[73px]`} />
						<div className={`${smile} left-20`} />
					</>
				)}
				{happy && <div className='absolute top-[85px] left-[74px] h-2.5 w-3.5 rounded-[2px_2px_8px_8px] border-[2.5px] border-dra-ink bg-[#e8665c]' />}
				{focus && <div className='absolute top-[87px] left-[77px] size-2 rounded-full border-[2.5px] border-dra-ink' />}
				{sad && <div className='absolute top-[89px] left-[75px] h-1.5 w-3 rounded-t-md border-[2.5px] border-b-0 border-dra-ink' />}
				{focus && <div className={`${tear} top-[58px] left-[122px] h-3 w-2 animate-dra-sweat`} />}

				<div className={`absolute top-[104px] left-[57px] flex h-5 w-12 items-center justify-center rounded-md border-2 bg-ink-4 text-[9.5px] font-bold transition-colors duration-300 ${TAG[mood]}`}>v{ver}</div>
				<div className={`${pawThin} top-[104px] left-[50px]`} />
				<div className={`${pawThin} top-[104px] left-[98px]`} />
			</div>

			{(happy || sad) && (
				<div key={`bubble-${mood}`} className={`absolute top-1.5 left-28 animate-dra-pop rounded-full px-[9px] py-1 text-[11px] font-bold whitespace-nowrap text-ink ${happy ? 'bg-lime' : 'bg-coral-soft'}`}>
					{happy ? '+1 ✓' : lang === 'en' ? 'oops…' : '哎呀…'}
				</div>
			)}
		</div>
	);
}
