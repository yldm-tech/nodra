// "How it works": three scripted scenarios (rollback, batch, async LatestWins) stepped through on a fixed diagram.

type Anchor = 'WIN' | 'STORE' | 'CAND' | 'T1' | 'T2' | 'S0' | 'S1' | 'S2' | 'S3';
type Pair = [zh: string, en: string];
type SubKind = 'idle' | 'hit' | 'skip' | 'rec' | 'queued' | 'saved' | 'none';
type TaskKind = 'run' | 'cancel' | 'done' | 'reject';
type PacketSpec = [label: string, from: Anchor, to: Anchor, failed?: 1];

interface Frame {
	ver: number;
	count: number;
	title: string;
	win: number;
	lock: boolean;
	batch: number | null;
	cand: { v: number; hot?: boolean; err?: boolean } | null;
	tasks: { l: string; s: TaskKind }[];
	subs: { s: SubKind; n?: string }[];
	persist: number;
	packets: PacketSpec[];
	err: string | null;
	glow: boolean;
	winFlash: boolean;
	line: number;
}

interface Scenario {
	tab: Pair;
	start: Partial<Frame>;
	code: string[];
	steps: { t: Pair; d: Pair; o: Partial<Frame> }[];
	states: Frame[];
}

export const STEP_MS = 3400;

const ANCH: Record<Anchor, [number, number]> = { WIN: [13.5, 55], STORE: [50, 48], CAND: [50, 86], T1: [39.5, 86], T2: [60.5, 86], S0: [76, 12.5], S1: [76, 35.5], S2: [76, 58.5], S3: [76, 81.5] };
const SUBN = ['Selector · Count', 'Selector · Title', 'History', 'Persistence'];
const SUBNZ = ['选择器 · count', '选择器 · title', '历史记录', '持久化'];
const SUBNOTE: Record<SubKind, Pair> = { idle: ['等待中', 'idle'], hit: ['触发', 'fired'], skip: ['未变化 · 跳过', 'unchanged · skipped'], rec: ['已记录', 'recorded'], queued: ['防抖中…', 'debouncing…'], saved: ['已保存至 state.json', 'saved to state.json'], none: ['未通知', 'not notified'] };
const SUBCOLOR: Record<SubKind, string> = { hit: '#c8f05a', skip: '#7c7f75', rec: '#9ecbff', queued: '#f0c674', saved: '#c8f05a', idle: '#7a7d74', none: '#7a7d74' };
const TASKST: Record<TaskKind, [zh: string, en: string, color: string, deco: string]> = { run: ['运行中…', 'running…', '#f0c674', 'none'], cancel: ['已取消', 'canceled', '#7c7f75', 'line-through'], done: ['完成', 'done', '#c8f05a', 'none'], reject: ['已拒绝', 'rejected', '#ff6b5a', 'line-through'] };
const PACKET_ZH: Record<string, string> = { copy: '复制', commit: '提交', error: '错误', apply: '应用结果', 'stale #1': '过时的 #1' };
const ERR_ZH: Record<string, string> = { 'validate: count < 0': '校验失败：count < 0', '#1 rejected · stale generation': '#1 被拒绝：已被更新的任务取代' };

const IDLE4: Frame['subs'] = [{ s: 'idle' }, { s: 'idle' }, { s: 'idle' }, { s: 'idle' }];
const NONE4: Frame['subs'] = [{ s: 'none' }, { s: 'none' }, { s: 'none' }, { s: 'none' }];
const BASE: Frame = { ver: 42, count: 7, title: 'Nodra', win: 7, lock: false, batch: null, cand: null, tasks: [], subs: IDLE4, persist: 0, packets: [], err: null, glow: false, winFlash: false, line: -1 };

const RAW: Omit<Scenario, 'states'>[] = [
	{
		tab: ['失败回滚', 'UpdateErr rollback'],
		start: { ver: 43, count: 8, win: 8 },
		code: ['err := store.UpdateErr(func(s *State) error {', '    s.Count = -1', '    return validate(*s) // count < 0', '})', '// err != nil：状态、版本、订阅者均未改变'],
		steps: [
			{ t: ['调用 UpdateErr', 'Call UpdateErr'], d: ['这次回调可以返回错误。', 'This time the callback may return an error.'], o: { winFlash: true, packets: [['UpdateErr(fn)', 'WIN', 'STORE']], line: 0 } },
			{ t: ['加锁，准备候选', 'Lock & prepare a candidate'], d: ['与 Update 相同：加锁并生成候选值。', 'Same as Update: lock and prepare a candidate.'], o: { lock: true, cand: { v: 8 }, packets: [['copy', 'STORE', 'CAND']], line: 0 } },
			{ t: ['写入非法值', 'Write an invalid value'], d: ['回调把 count 改成 -1，而 Store 中仍然是 8。', 'The callback sets count to -1. The Store itself still holds 8.'], o: { cand: { v: -1, hot: true }, line: 1 } },
			{ t: ['校验失败', 'Validation fails'], d: ['校验失败，回调返回错误（panic 的效果完全相同）。', 'The callback returns an error — a panic behaves exactly the same.'], o: { cand: { v: -1, err: true }, err: 'validate: count < 0', line: 2 } },
			{ t: ['丢弃候选，什么都不发布', 'Discard — publish nothing'], d: ['候选被丢弃：状态仍是 8，版本仍是 v43，没有任何订阅者被打扰。错误交还给调用方。', 'The candidate is discarded: state stays 8, version stays v43, no subscriber is disturbed. The error returns to the caller.'], o: { cand: null, lock: false, packets: [['error', 'STORE', 'WIN', 1]], subs: NONE4, line: 4 } },
		],
	},
	{
		tab: ['批量合并', 'Batch'],
		start: { ver: 43, count: 8, win: 8 },
		code: ['store.Batch(func() {', '    store.Update(func(s *State) { s.Count++ })', '    store.Update(func(s *State) { s.Title = "Hi" })', '    store.Update(func(s *State) { s.Count++ })', '}) // ← 只在这里发布一次'],
		steps: [
			{ t: ['开启 Batch', 'Open a Batch'], d: ['Batch 内的写入照常提交，但通知会暂存到最外层结束时统一发出。', 'Writes inside a Batch commit normally, but notifications are held until the outermost boundary.'], o: { batch: 0, line: 0 } },
			{ t: ['第 1 次写入：count = 9', 'Write 1: count = 9'], d: ['版本 v44，通知暂存。', 'Version v44; the notification is held.'], o: { batch: 1, ver: 44, count: 9, glow: true, packets: [['Update', 'WIN', 'STORE']], line: 1 } },
			{ t: ['第 2 次写入：title = "Hi"', 'Write 2: title = "Hi"'], d: ['版本 v45，订阅者仍然什么都没收到。', 'Version v45; subscribers still receive nothing.'], o: { batch: 2, ver: 45, title: 'Hi', glow: true, packets: [['Update', 'WIN', 'STORE']], line: 2 } },
			{ t: ['第 3 次写入：count = 10', 'Write 3: count = 10'], d: ['三次写入、三个版本、零次通知。', 'Three writes, three versions, zero notifications.'], o: { batch: 3, ver: 46, count: 10, glow: true, packets: [['Update', 'WIN', 'STORE']], line: 3 } },
			{
				t: ['关闭：只发布一次', 'Close: publish once'],
				d: ['在最外层边界一次性发布最终快照，每个订阅者只回调一次。', 'The final snapshot is published once at the outermost boundary — one callback per subscriber.'],
				o: { batch: null, packets: [['v46', 'STORE', 'S0'], ['v46', 'STORE', 'S1'], ['v46', 'STORE', 'S2'], ['v46', 'STORE', 'S3']], subs: [{ s: 'hit', n: '8 → 10' }, { s: 'hit', n: '"Nodra" → "Hi"' }, { s: 'rec', n: 'v43 → undo stack' }, { s: 'queued' }], persist: 0.5, line: 4 },
			},
			{
				t: ['一次重绘', 'One redraw'],
				d: ['窗口只重绘一次，直接从 8 跳到 10。', 'The window redraws once, jumping straight from 8 to 10.'],
				o: { win: 10, winFlash: true, packets: [['Invalidate()', 'S0', 'WIN']], subs: [{ s: 'hit', n: '8 → 10' }, { s: 'hit', n: '"Nodra" → "Hi"' }, { s: 'rec', n: 'v43 → undo stack' }, { s: 'saved' }], persist: 1, line: 4 },
			},
		],
	},
	{
		tab: ['异步 · 最新优先', 'Async LatestWins'],
		start: { ver: 46, count: 10, win: 10, title: 'Hi' },
		code: ['search := nodra.NewAsyncAction(store, work, apply)', 'search.Run(ctx, "no")     // #1', 'search.Run(ctx, "nodra")  // #2 取代 #1', '// #2 先完成 → 加锁、校验代际、提交', '// #1 迟到 → 被拒绝，绝不覆盖新结果'],
		steps: [
			{ t: ['启动 #1', 'Start run #1'], d: ['异步动作在所有锁之外执行耗时工作，例如一次网络搜索。', 'An AsyncAction runs work outside every lock — a network search, say.'], o: { tasks: [{ l: '#1 "no"', s: 'run' }], packets: [['Run #1', 'WIN', 'T1']], line: 1 } },
			{ t: ['启动 #2，取代 #1', '#2 supersedes #1'], d: ['最新优先：新任务启动时，旧任务的上下文会被取消。', 'LatestWins: a newer run starts and the older run’s context is canceled.'], o: { tasks: [{ l: '#1 "no"', s: 'cancel' }, { l: '#2 "nodra"', s: 'run' }], packets: [['Run #2', 'WIN', 'T2']], line: 2 } },
			{ t: ['#2 完成 → 提交', '#2 finishes → commit'], d: ['#2 获取写锁与代际锁，确认自己仍是最新，然后应用结果：v47。', '#2 takes the write and generation locks, confirms it is still the latest, and applies: v47.'], o: { tasks: [{ l: '#1 "no"', s: 'cancel' }, { l: '#2 "nodra"', s: 'done' }], ver: 47, title: 'nodra ×12', glow: true, packets: [['apply', 'T2', 'STORE']], line: 3 } },
			{ t: ['通知订阅者', 'Notify subscribers'], d: ['与普通 Update 一样在锁外通知；只有关注 title 的选择器被触发。', 'Notified outside the lock, like any Update — only the Title selector fires.'], o: { packets: [['v47', 'STORE', 'S1'], ['v47', 'STORE', 'S2']], subs: [{ s: 'skip' }, { s: 'hit', n: '"Hi" → "nodra ×12"' }, { s: 'rec', n: 'v46 → undo stack' }, { s: 'idle' }], line: 3 } },
			{ t: ['迟到的 #1 被拒绝', 'The late #1 is rejected'], d: ['#1 终于返回，但它无法越过 #2 已接受的提交边界，旧结果永远不会覆盖新结果。', '#1 finally returns, but it cannot pass #2’s accepted commit boundary — stale results never overwrite newer ones.'], o: { tasks: [{ l: '#1 "no"', s: 'reject' }, { l: '#2 "nodra"', s: 'done' }], packets: [['stale #1', 'T1', 'STORE', 1]], err: '#1 rejected · stale generation', line: 4 } },
		],
	},
];

export const SCENARIOS: Scenario[] = RAW.map((sc) => {
	let cur: Frame = { ...BASE, ...sc.start };
	const states = sc.steps.map((st) => (cur = { ...cur, packets: [], err: null, glow: false, winFlash: false, ...st.o }));
	return { ...sc, states };
});

export interface HowCursor {
	scen: number;
	step: number;
	phase: 0 | 1;
}

export function nextCursor(c: HowCursor, delta: number): Pick<HowCursor, 'scen' | 'step'> {
	const n = SCENARIOS[c.scen].steps.length;
	let step = c.step + delta;
	let scen = c.scen;
	if (step >= n) {
		scen = (scen + 1) % SCENARIOS.length;
		step = 0;
	}
	if (step < 0) step = 0;
	return { scen, step };
}

const pct = (n: number) => `${n}%`;

export function howView(c: HowCursor, li: 0 | 1) {
	const sc = SCENARIOS[c.scen];
	const i = c.step;
	const s = sc.states[i];
	const ph = c.phase;

	const packets = [0, 1, 2, 3].map((k) => {
		const p = s.packets[k];
		if (!p) return { x: '50%', y: '48%', op: 0, tr: 'opacity .2s', bg: '#c8f05a', label: '' };
		const at = ph ? ANCH[p[2]] : ANCH[p[1]];
		return {
			x: pct(at[0]),
			y: pct(at[1]),
			op: 1,
			label: li ? p[0] : (PACKET_ZH[p[0]] ?? p[0]),
			bg: p[3] ? '#ff6b5a' : '#c8f05a',
			tr: ph ? `left 1.1s cubic-bezier(.6,0,.2,1) ${k * 0.08}s, top 1.1s cubic-bezier(.6,0,.2,1) ${k * 0.08}s, opacity .2s` : 'none',
		};
	});

	const subs = s.subs.map((sb, k) => {
		const st = sb.s;
		const note = sb.n ? (st === 'hit' ? `${sb.n} · ${SUBNOTE.hit[li]}` : sb.n) : SUBNOTE[st][li];
		const lit = ph === 1 && s.packets.some((p) => p[2] === `S${k}`);
		const col = SUBCOLOR[st];
		const prevPersist = (sc.states[i - 1] ?? BASE).persist;
		return {
			name: (li ? SUBN : SUBNZ)[k],
			top: `${[4, 27, 50, 73][k]}%`,
			note,
			dot: col,
			noteColor: st === 'idle' || st === 'none' ? '#878a81' : col,
			deco: st === 'skip' || st === 'none' ? 'line-through' : 'none',
			border: lit ? col : 'rgba(238,240,234,0.1)',
			hasBar: k === 3,
			barW: k === 3 ? `${(ph ? s.persist : prevPersist) * 100}%` : '0%',
		};
	});

	const cand = s.cand;
	const v = {
		ver: String(s.ver).padStart(4, '0'),
		count: s.count,
		title: s.title,
		win: s.win,
		winBorder: s.winFlash ? '#c8f05a' : 'rgba(238,240,234,0.12)',
		winShadow: s.winFlash ? '0 0 30px -6px rgba(200,240,90,0.6)' : 'none',
		btnBg: s.winFlash ? '#c8f05a' : 'rgba(238,240,234,0.08)',
		btnFg: s.winFlash ? '#060709' : '#9ea197',
		storeBorder: s.lock ? '#f0c674' : s.glow ? '#c8f05a' : 'rgba(238,240,234,0.16)',
		storeShadow: s.glow ? '0 0 60px -10px rgba(200,240,90,0.55)' : s.lock ? '0 0 40px -12px rgba(240,198,116,0.5)' : '0 20px 50px -20px rgba(0,0,0,0.6)',
		lockOp: s.lock ? 1 : 0,
		batchOp: s.batch === null ? 0 : 1,
		batch: s.batch ?? 0,
		candOp: cand ? 1 : 0,
		candScale: cand ? 1 : 0.85,
		cand: cand ? cand.v : '',
		candColor: cand?.err ? '#ff6b5a' : cand?.hot ? '#c8f05a' : '#eef0ea',
		candBorder: cand?.err ? '#ff6b5a' : cand?.hot ? '#c8f05a' : 'rgba(238,240,234,0.3)',
		candBg: cand?.err ? 'rgba(255,107,90,0.08)' : '#111318',
		tasks: s.tasks.map((tk, k) => {
			const st = TASKST[tk.s];
			return { left: k ? '52%' : '31%', label: tk.l, state: st[li], color: st[2], deco: st[3] };
		}),
		subs,
		packets,
		err: li ? (s.err ?? '') : s.err ? (ERR_ZH[s.err] ?? s.err) : '',
		errOp: s.err ? 1 : 0,
	};

	return {
		v,
		cur: { num: String(i + 1).padStart(2, '0'), total: String(sc.steps.length).padStart(2, '0'), title: sc.steps[i].t[li], desc: sc.steps[i].d[li] },
		tabs: SCENARIOS.map((x, k) => ({ label: x.tab[li], active: k === c.scen })),
		steps: sc.steps.map((st, k) => ({ num: String(k + 1).padStart(2, '0'), title: st.t[li], desc: st.d[li], active: k === i, done: k < i })),
		codeLines: sc.code.map((ln, k) => ({ n: k + 1, text: ln, active: k === s.line, comment: ln.trim().startsWith('//') })),
	};
}
