import { Link } from '@tanstack/react-router';
import { useLang } from '../lib/lang';
import { Mascot } from './Mascot';
import { SiteHeader } from './SiteHeader';

const COPY = {
	zh: { title: '这个版本不存在', body: '找不到这个页面，可能链接已经过期。', home: '回到首页' },
	en: { title: 'No such version', body: 'This page does not exist; the link may be out of date.', home: 'Back home' },
};

export function NotFound() {
	const lang = useLang();
	const t = COPY[lang];
	return (
		<div className='min-h-screen bg-ink'>
			<SiteHeader active='home' />
			<main className='flex min-h-screen flex-col items-center justify-center gap-6 px-5 text-center'>
				<Mascot mood='sad' ver='404' lang={lang} />
				<h1 className='m-0 text-[clamp(32px,4vw,48px)] font-bold tracking-[-0.04em]'>{t.title}</h1>
				<p className='m-0 text-fg-3'>{t.body}</p>
				<Link to='/' className='rounded-full bg-lime px-7 py-4 font-bold text-ink hover:bg-white hover:text-ink'>
					{t.home}
				</Link>
			</main>
		</div>
	);
}
