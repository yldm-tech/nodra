import { Link } from '@tanstack/react-router';
import { REPO_URL, useLang, useSetLang } from '../lib/lang';

const NAV = {
	zh: { home: '首页', docs: '文档', examples: '示例' },
	en: { home: 'Home', docs: 'Docs', examples: 'Examples' },
};

const ear = 'absolute top-2 size-2.5 rounded-full bg-dra-cream';
const earInner = 'absolute top-[3px] left-[3px] size-1 rounded-full bg-dra-blush';
const eye = 'absolute top-5 h-[4.5px] w-[3.5px] rounded-full bg-dra-ink';
const cheek = 'absolute top-[25px] h-[3px] w-[5px] rounded-full bg-[rgba(255,140,130,0.75)]';
const mouth = 'absolute top-[25px] h-[2.5px] w-[3.5px] rounded-b-[3px] border-[1.2px] border-t-0 border-dra-ink';

// The small Dra face used as the wordmark icon.
export function LogoMark() {
	return (
		<span aria-hidden='true' className='relative h-9 w-[38px] flex-none'>
			<span className='absolute top-0 left-[17px] h-2 w-0.5 rounded-[1px] bg-dra-stem' />
			<span className='absolute -top-px left-2.5 h-[5px] w-2 -rotate-[28deg] rounded-full bg-lime' />
			<span className='absolute -top-0.5 left-[18px] h-[5px] w-2 rotate-[28deg] rounded-full bg-lime' />
			<span className={`${ear} left-[3px]`}>
				<span className={earInner} />
			</span>
			<span className={`${ear} left-[23px]`}>
				<span className={earInner} />
			</span>
			<span className='absolute top-2.5 left-px h-[26px] w-[34px] rounded-[50%_50%_46%_46%/60%_60%_40%_40%] bg-dra-cream' />
			<span className={`${eye} left-2.5`} />
			<span className={`${eye} left-[23px]`} />
			<span className={`${cheek} left-1.5`} />
			<span className={`${cheek} left-[26px]`} />
			<span className={`${mouth} left-[15px]`} />
			<span className={`${mouth} left-[18px]`} />
		</span>
	);
}

export function GitHubIcon() {
	return (
		<svg viewBox='0 0 16 16' aria-hidden='true' className='block size-5 fill-current'>
			<path d='M8 0C3.58 0 0 3.58 0 8c0 3.54 2.29 6.53 5.47 7.59.4.07.55-.17.55-.38 0-.19-.01-.82-.01-1.49-2.01.37-2.53-.49-2.69-.94-.09-.23-.48-.94-.82-1.13-.28-.15-.68-.52-.01-.53.63-.01 1.08.58 1.23.82.72 1.21 1.87.87 2.33.66.07-.52.28-.87.51-1.07-1.78-.2-3.64-.89-3.64-3.95 0-.87.31-1.59.82-2.15-.08-.2-.36-1.02.08-2.12 0 0 .67-.21 2.2.82.64-.18 1.32-.27 2-.27.68 0 1.36.09 2 .27 1.53-1.04 2.2-.82 2.2-.82.44 1.1.16 1.92.08 2.12.51.56.82 1.27.82 2.15 0 3.07-1.87 3.75-3.65 3.95.29.25.54.73.54 1.48 0 1.07-.01 1.93-.01 2.2 0 .21.15.46.55.38A8.013 8.013 0 0016 8c0-4.42-3.58-8-8-8z' />
		</svg>
	);
}

const navLink = 'border-b-[1.5px] py-1.5 hover:text-fg';
const navIdle = `${navLink} border-transparent text-fg-4`;
const navActive = `${navLink} border-lime text-fg`;
const langBtn = 'cursor-pointer border-0 px-3 py-1.5 font-mono text-xs';

export function SiteHeader({ active }: { active: 'home' | 'docs' }) {
	const lang = useLang();
	const setLang = useSetLang();
	const t = NAV[lang];

	return (
		<header className='fixed inset-x-0 top-0 z-20 h-[68px] border-b border-fg/8 bg-ink/88 backdrop-blur-md'>
			<div className='mx-auto flex h-full max-w-[1312px] items-center gap-8 px-9 max-sm:gap-2.5 max-sm:px-4'>
				<Link to='/' aria-label='Nodra' className='flex flex-none items-center gap-2.5 pt-1 text-fg hover:text-fg'>
					<LogoMark />
					<span className='text-[22px] font-bold tracking-[-0.035em] max-sm:hidden'>nodra</span>
				</Link>
				<nav className='flex min-w-0 flex-1 gap-[26px] overflow-x-auto font-mono text-[13px] whitespace-nowrap max-sm:gap-3'>
					<Link to='/' className={active === 'home' ? navActive : navIdle}>
						{t.home}
					</Link>
					<Link to='/docs' className={active === 'docs' ? navActive : navIdle}>
						{t.docs}
					</Link>
					<Link to='/docs' hash='store' className={navIdle}>
						API
					</Link>
					<a href={`${REPO_URL}/tree/main/examples`} className={navIdle}>
						{t.examples}
					</a>
				</nav>
				<div role='group' aria-label='Language' className='flex flex-none overflow-hidden rounded-full border border-fg/18 font-mono text-xs'>
					<button type='button' lang='zh-CN' aria-pressed={lang === 'zh'} onClick={() => setLang('zh')} className={`${langBtn} ${lang === 'zh' ? 'bg-fg text-ink' : 'bg-transparent text-fg-4'}`}>
						中
					</button>
					<button type='button' lang='en' aria-pressed={lang === 'en'} onClick={() => setLang('en')} className={`${langBtn} ${lang === 'en' ? 'bg-fg text-ink' : 'bg-transparent text-fg-4'}`}>
						EN
					</button>
				</div>
				<a href={REPO_URL} aria-label='GitHub' title='GitHub' className='flex size-[38px] flex-none items-center justify-center rounded-full border border-fg/18 text-fg hover:border-lime hover:bg-lime hover:text-ink'>
					<GitHubIcon />
				</a>
			</div>
		</header>
	);
}
