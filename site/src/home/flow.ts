import type { Mood } from '../components/Mascot';

// The hero diagram: a looping 17.4 s script of goroutines writing to one Store — a plain update, two contending writers, then a rolled-back UpdateErr.

type Point = [number, number];
type GorState = '' | 'w' | 'q' | 'e';
type SubState = 'idle' | 'hit' | 'skip' | 'none' | 'pend' | 'saved';
type Pair = [zh: string, en: string];

interface Packet {
	x: number;
	y: number;
	op: number;
	tr: string;
	bg: string;
	label: string;
}

export interface FlowModel {
	ver: number;
	count: number;
	title: string;
	win: [number, string];
	hist: number;
	lock: boolean;
	glow: '' | 'ok' | 'err';
	cand: [label: string, color: string] | null;
	gor: GorState[];
	subs: SubState[];
	per: { p: number; tr: string; pend: boolean; saved: number };
	pk: Packet[];
	cap: [Pair, string];
}

const LIME = '#c8f05a';
const AMBER = '#f0c674';
const RED = '#ff6b5a';
const GREY = '#7a7d74';
const LOOP_MS = 17400;

const G: Point[] = [
	[18, 20],
	[18, 50],
	[18, 80],
];
const IN: Point = [31, 50];
const Q: Point = [25, 66];
const OUT: Point = [61, 50];
const SUB: Point[] = [
	[70, 12],
	[70, 37],
	[70, 62],
	[70, 87],
];

const V = (n: number) => `v${String(n).padStart(4, '0')}`;

function fresh(prev?: FlowModel): FlowModel {
	return {
		ver: prev ? prev.ver : 42,
		count: prev ? prev.count : 7,
		title: prev ? prev.title : 'Nodra',
		win: prev ? [prev.win[0], prev.win[1]] : [7, 'Nodra'],
		hist: prev ? prev.hist : 6,
		lock: false,
		glow: '',
		cand: null,
		gor: ['', '', ''],
		subs: ['idle', 'idle', 'idle', 'idle'],
		per: { p: 1, tr: 'none', pend: false, saved: prev ? prev.per.saved : 42 },
		pk: [0, 1, 2, 3, 4, 5].map(() => ({ x: 50, y: 50, op: 0, tr: 'none', bg: LIME, label: '' })),
		cap: [['等待写入…', 'Waiting for a write…'], GREY],
	};
}

function cloneModel(m: FlowModel): FlowModel {
	return { ...m, pk: m.pk.map((p) => ({ ...p })), subs: [...m.subs], gor: [...m.gor], win: [m.win[0], m.win[1]], per: { ...m.per } };
}

// Runs the script, calling `emit` with a fresh copy after every beat. While `canPlay` is false the loop idles and polls. Returns a stop function.
export function startHeroFlow(emit: (m: FlowModel) => void, canPlay: () => boolean): () => void {
	let M = fresh();
	let timers: ReturnType<typeof setTimeout>[] = [];
	let first = true;
	const push = () => emit(cloneModel(M));

	const place = (i: number, at: Point, bg: string, label: string) => {
		M.pk[i] = { x: at[0], y: at[1], op: 1, tr: 'none', bg, label };
	};
	const move = (i: number, at: Point, d: number) => {
		M.pk[i] = { ...M.pk[i], x: at[0], y: at[1], tr: `left ${d}ms cubic-bezier(.6,0,.2,1), top ${d}ms cubic-bezier(.6,0,.2,1), opacity .25s` };
	};
	const hide = (...ix: number[]) =>
		ix.forEach((i) => {
			M.pk[i] = { ...M.pk[i], op: 0, tr: 'opacity .2s' };
		});
	const cap = (zh: string, en: string, c = LIME) => {
		M.cap = [[zh, en], c];
	};
	const other = () => (M.title === 'Nodra' ? 'Hi' : 'Nodra');
	const notify = (skip: boolean[], t0: number): [number, () => void][] => [
		[
			t0,
			() => {
				[0, 1, 2, 3].forEach((k) => place(k + 1, OUT, skip[k] ? GREY : LIME, skip[k] ? 'Δ=0' : V(M.ver)));
			},
		],
		[
			t0 + 60,
			() => {
				[0, 1, 2, 3].forEach((k) => move(k + 1, SUB[k], 760));
			},
		],
		[
			t0 + 860,
			() => {
				hide(1, 2, 3, 4);
				M.subs = skip.map((s, k) => (k === 3 ? 'pend' : k === 2 ? 'hit' : s ? 'skip' : 'hit'));
				if (!skip[0]) M.win[0] = M.count;
				if (!skip[1]) M.win[1] = M.title;
				M.hist++;
				M.per = { ...M.per, p: 0, tr: 'none', pend: true };
				M.glow = '';
			},
		],
		[
			t0 + 920,
			() => {
				M.per = { ...M.per, p: 1, tr: 'width 1300ms linear' };
			},
		],
	];

	const script: [number, () => void][] = [
		[0, () => { place(0, G[0], LIME, 'Update · count++'); M.gor[0] = 'w'; cap('goroutine A 调用 store.Update(fn)', 'goroutine A calls store.Update(fn)'); }],
		[60, () => move(0, IN, 900)],
		[980, () => { hide(0); M.lock = true; M.cand = [`candidate · count ${M.count} → ${M.count + 1}`, LIME]; cap(`已加写锁，回调只修改候选值；此时所有读取仍看到 ${V(M.ver)}`, `Write lock held; the callback edits a candidate — readers still see ${V(M.ver)}`, AMBER); }],
		[2000, () => { M.lock = false; M.cand = null; M.ver++; M.count++; M.glow = 'ok'; M.gor[0] = ''; cap(`提交 → ${V(M.ver)}，释放锁后再通知订阅者`, `Commit → ${V(M.ver)}, unlock, then notify outside the lock`); }],
		...notify([false, true, false, false], 2100),
		[3000, () => cap('count 变了，窗口重绘；title 没变，选择器跳过；历史记录已保存；落盘进入防抖', 'Count changed → redraw; Title unchanged → skipped; History records; save is debounced')],
		[4300, () => { M.per = { ...M.per, pend: false, saved: M.ver }; M.subs[3] = 'saved'; }],
		[5400, () => { M.subs = ['idle', 'idle', 'idle', 'idle']; place(0, G[1], LIME, 'Update · title'); place(5, G[2], LIME, 'Update · count++'); M.gor[1] = 'w'; M.gor[2] = 'w'; cap('goroutine B 和 C 同时写入', 'goroutines B and C write at the same time'); }],
		[5460, () => { move(0, IN, 900); move(5, Q, 820); }],
		[6380, () => { hide(0); M.lock = true; M.cand = [`candidate · title "${other()}"`, LIME]; M.pk[5] = { ...M.pk[5], bg: AMBER, label: 'C · waiting for lock' }; M.gor[2] = 'q'; cap('B 拿到写锁；C 在锁外排队，不会读到 B 的中间状态', 'B holds the lock; C queues outside and never sees a half-done value', AMBER); }],
		[7400, () => { M.lock = false; M.cand = null; M.ver++; M.title = other(); M.glow = 'ok'; M.gor[1] = ''; cap(`B 提交 ${V(M.ver)}`, `B commits ${V(M.ver)}`); }],
		...notify([true, false, false, false], 7500),
		[7600, () => { M.pk[5] = { ...M.pk[5], bg: LIME, label: 'Update · count++' }; move(5, IN, 520); M.gor[2] = 'w'; }],
		[8160, () => { hide(5); M.lock = true; M.cand = [`candidate · count ${M.count} → ${M.count + 1}`, LIME]; cap('锁释放后 C 立即拿到锁', 'C takes the lock as soon as it is released', AMBER); }],
		[9200, () => { M.lock = false; M.cand = null; M.ver++; M.count++; M.glow = 'ok'; M.gor[2] = ''; cap(`C 提交 ${V(M.ver)}：两次写入严格串行，版本号连续`, `C commits ${V(M.ver)} — writes are serialized and versions stay contiguous`); }],
		...notify([false, true, false, false], 9300),
		[10200, () => cap('持久化的防抖把两次提交合并成一次磁盘写入', 'Persistence debounce merged both commits into one disk write')],
		[11500, () => { M.per = { ...M.per, pend: false, saved: M.ver }; M.subs[3] = 'saved'; }],
		[12600, () => { M.subs = ['idle', 'idle', 'idle', 'idle']; place(0, G[0], LIME, 'UpdateErr · count = -1'); M.gor[0] = 'w'; cap('goroutine A 调用 UpdateErr，试图写入非法值', 'goroutine A calls UpdateErr with an invalid value'); }],
		[12660, () => move(0, IN, 900)],
		[13580, () => { hide(0); M.lock = true; M.cand = ['candidate · count -1  ✗ validate', RED]; cap('回调返回错误（panic 也一样）', 'The callback returns an error — a panic behaves the same', AMBER); }],
		[14600, () => { M.lock = false; M.cand = null; M.glow = 'err'; M.subs = ['none', 'none', 'none', 'none']; place(0, IN, RED, '✗ error'); cap(`整体回滚：${V(M.ver)} 保持不变，没有订阅者被通知，错误返回给调用方`, `Rolled back: ${V(M.ver)} unchanged, 0 subscribers notified, error returned`, RED); }],
		[14660, () => move(0, G[0], 820)],
		[15600, () => { hide(0); M.gor[0] = 'e'; M.glow = ''; }],
		[16600, () => { M.gor[0] = ''; }],
	];

	const run = () => {
		if (!canPlay()) {
			if (first) {
				M = fresh();
				first = false;
				push();
			}
			timers = [setTimeout(run, 700)];
			return;
		}
		M = first ? fresh() : fresh(M);
		first = false;
		push();
		timers = script.map(([at, fn]) =>
			setTimeout(() => {
				fn();
				push();
			}, at + 300),
		);
		timers.push(setTimeout(run, LOOP_MS));
	};
	run();
	return () => timers.forEach(clearTimeout);
}

const GSUB: Record<GorState, [zh: string, en: string, color: string]> = { '': ['空闲', 'idle', '#878a81'], w: ['写入中…', 'writing…', '#c8f05a'], q: ['等待锁', 'waiting', '#f0c674'], e: ['收到错误', 'got error', '#ff6b5a'] };
const GBORDER: Record<GorState, string> = { '': 'rgba(238,240,234,0.1)', w: '#c8f05a', q: '#f0c674', e: '#ff6b5a' };
const DIM = 'rgba(238,240,234,0.1)';

export function flowView(M: FlowModel | null, li: 0 | 1) {
	if (!M) M = fresh();
	const pad = (n: number) => String(n).padStart(4, '0');
	const names = li ? ['Window · Count', 'Window · Title', 'History', 'Persistence'] : ['窗口 · count', '窗口 · title', '历史记录', '持久化'];
	const subs = M.subs.map((st, k) => {
		let val: string;
		let note: string;
		let noteColor = '#7a7d74';
		let border = DIM;
		let valColor = '#eef0ea';
		let deco = 'none';
		if (k === 0) val = String(M.win[0]);
		else if (k === 1) val = `"${M.win[1]}"`;
		else if (k === 2) val = (li ? 'undo ×' : '可撤销 ×') + M.hist;
		else val = M.per.pend ? (li ? 'debounce 100ms…' : '防抖 100 毫秒…') : (li ? 'saved v' : '已保存 v') + pad(M.per.saved);
		if (st === 'hit') {
			note = k === 2 ? (li ? 'snapshot recorded' : '已记录快照') : li ? 'changed → redraw' : '变化 → 重绘';
			noteColor = border = valColor = '#c8f05a';
		} else if (st === 'skip') {
			note = li ? 'Δ = 0 · skipped' : 'Δ = 0 · 跳过';
			noteColor = '#9ea197';
		} else if (st === 'none') {
			note = li ? 'not notified' : '未通知';
			deco = 'line-through';
		} else if (st === 'pend') {
			note = li ? 'merging writes…' : '合并写入中…';
			noteColor = border = valColor = '#f0c674';
		} else if (st === 'saved') {
			note = '→ state.json';
			noteColor = border = valColor = '#c8f05a';
		} else note = ['selector(s.Count)', 'selector(s.Title)', 'Track(store)', 'AutoSaveAsync'][k];
		return {
			name: names[k],
			top: `${[12, 37, 62, 87][k]}%`,
			val,
			note,
			noteColor,
			valColor,
			border,
			deco,
			shadow: border === DIM ? 'none' : `0 0 22px -8px ${border}`,
			hasBar: k === 3,
			barW: `${k === 3 ? M.per.p * 100 : 0}%`,
			barTr: M.per.tr,
			barColor: M.per.pend ? '#f0c674' : '#c8f05a',
		};
	});
	const g = M.glow;
	const mood: Mood = g === 'err' ? 'sad' : M.lock ? 'focus' : g === 'ok' ? 'happy' : 'idle';
	return {
		gors: M.gor.map((s, k) => ({ name: `goroutine ${'ABC'[k]}`, top: `${[20, 50, 80][k]}%`, sub: GSUB[s][li], subColor: GSUB[s][2], border: GBORDER[s], shadow: s ? `0 0 20px -8px ${GBORDER[s]}` : 'none' })),
		ver: pad(M.ver),
		count: M.count,
		title: M.title,
		lockOp: M.lock ? 1 : 0,
		storeBorder: M.lock ? '#f0c674' : g === 'ok' ? '#c8f05a' : g === 'err' ? '#ff6b5a' : 'rgba(238,240,234,0.18)',
		storeShadow: M.lock ? '0 0 40px -12px rgba(240,198,116,0.6)' : g === 'ok' ? '0 0 56px -10px rgba(200,240,90,0.6)' : g === 'err' ? '0 0 56px -10px rgba(255,107,90,0.6)' : '0 20px 50px -24px rgba(0,0,0,0.7)',
		cand: M.cand ? (li ? M.cand[0] : M.cand[0].replace('candidate', '候选值').replace('✗ validate', '✗ 校验失败')) : '',
		candColor: M.cand ? M.cand[1] : 'transparent',
		candOp: M.cand ? 1 : 0,
		subs,
		pk: M.pk.map((p) => ({ ...p, x: `${p.x}%`, y: `${p.y}%`, label: li ? p.label : p.label.replace('C · waiting for lock', 'C · 等待锁').replace('✗ error', '✗ 错误') })),
		cap: M.cap[0][li],
		capColor: M.cap[1],
		mood,
		bubbleBorder: g === 'err' ? '#ff6b5a' : M.lock ? '#f0c674' : g === 'ok' ? '#c8f05a' : 'rgba(238,240,234,0.14)',
		mascotTag: `${li ? 'DRA · KEEPER OF ' : 'Dra · 守护 '}v${pad(M.ver)}`,
		mascotLine: g === 'err' ? (li ? 'rolled back, nothing leaked' : '回滚了，什么都没泄露') : M.lock ? (li ? 'holding the lock…' : '我拿着锁呢…') : g === 'ok' ? (li ? 'new version! telling everyone' : '新版本！通知大家') : li ? 'keeper of your state' : '你的状态守护者',
	};
}
