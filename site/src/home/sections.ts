// The five feature diagrams share one clock: `step` advances every 2 s and `ph` walks 0 → 1 → 2 within each step.

const LIME = '#c8f05a';
const AMB = '#f0c674';
const RED = '#ff6b5a';
const DIM = 'rgba(238,240,234,0.1)';
const VV = (n: number) => `v${String(n).padStart(4, '0')}`;

export function sectionsView(step: number, ph: 0 | 1 | 2, li: 0 | 1) {
	const s = step;
	const L = (zh: string, en: string) => (li ? en : zh);

	// 1 — one owner
	const lane = s % 3;
	const ys = [26, 50, 74];
	const cur = 100 + s - (ph === 2 ? 0 : 1);
	const waiting = (s + 1) % 3;
	const v1 = {
		title: L('一个所有者 · 并发写入被串行化', 'ONE OWNER · CONCURRENT WRITES SERIALIZED'),
		lanes: ['goroutine A', 'goroutine B', 'goroutine C'].map((name, k) => ({
			name,
			top: `${ys[k]}%`,
			status: k === lane ? (ph === 2 ? L('已提交', 'committed') : L('写入中…', 'writing…')) : k === waiting && ph < 2 ? L('等待锁', 'waiting') : L('空闲', 'idle'),
			color: k === lane ? LIME : k === waiting && ph < 2 ? AMB : '#878a81',
			border: k === lane ? LIME : DIM,
		})),
		ver: VV(cur),
		lockOp: ph === 1 ? 1 : 0,
		storeBorder: ph === 1 ? AMB : ph === 2 ? LIME : 'rgba(238,240,234,0.18)',
		storeShadow: ph === 2 ? '0 0 40px -10px rgba(200,240,90,0.6)' : 'none',
		owner: L('唯一的状态所有者', 'single owner'),
		log: [0, 1, 2, 3].map((j) => ({
			v: VV(cur - j),
			from: `← ${'ABC'[(((s - j - (ph === 2 ? 0 : 1)) % 3) + 3) % 3]}`,
			bg: j === 0 && ph === 2 ? 'rgba(200,240,90,0.14)' : 'rgba(238,240,234,0.04)',
			color: j === 0 ? '#eef0ea' : '#7c7f75',
		})),
		pk: {
			x: ph === 0 ? '24%' : '44%',
			y: ph === 0 ? `${ys[lane]}%` : '50%',
			op: ph === 2 ? 0 : 1,
			tr: ph === 1 ? 'left 1s cubic-bezier(.6,0,.2,1), top 1s cubic-bezier(.6,0,.2,1), opacity .2s' : 'opacity .2s',
			label: 'Update(fn)',
		},
		cap:
			ph === 2
				? L(`goroutine ${'ABC'[lane]} 提交 ${VV(cur)}：版本连续，没有并发写丢失`, `goroutine ${'ABC'[lane]} commits ${VV(cur)} — contiguous versions, no lost writes`)
				: L('多个 goroutine 同时调用 Update，Store 依次加锁处理', 'Many goroutines call Update; the Store takes them one at a time'),
		capColor: ph === 2 ? LIME : '#b9bcb2',
	};

	// 2 — history
	const F = [
		{ len: 4, head: 3, a: ['Update → v4', 'Update → v4'] },
		{ len: 5, head: 4, a: ['Update → v5', 'Update → v5'] },
		{ len: 5, head: 3, a: ['Undo() ↶', 'Undo() ↶'] },
		{ len: 5, head: 2, a: ['Undo() ↶', 'Undo() ↶'] },
		{ len: 5, head: 3, a: ['Redo() ↷', 'Redo() ↷'] },
		{ len: 5, head: 4, drop: true, a: ['Update：产生新版本，旧的可重做记录被丢弃', 'Update → new version, old redo dropped'] },
	];
	const f = F[s % F.length];
	const X = (i: number) => `${12 + i * 19}%`;
	const v2 = {
		title: L('撤销与重做 · Track(store)', 'UNDO / REDO · Track(store)'),
		act: f.a[li],
		actColor: f.drop ? AMB : f.a[0].startsWith('U') && f.a[0].includes('do') ? '#9ecbff' : LIME,
		dots: [0, 1, 2, 3, 4].map((i) => {
			const head = i === f.head;
			const redo = i > f.head;
			return {
				x: X(i),
				op: i < f.len ? 1 : 0,
				scale: head ? 1.15 : 1,
				label: `v${i + 1}${f.drop && i === 4 ? '′' : ''}`,
				border: head ? LIME : redo ? '#4a4d45' : '#9ea197',
				bg: head ? LIME : redo ? 'transparent' : '#1d2016',
				color: head ? '#060709' : redo ? '#878a81' : '#dfe2da',
				shadow: head ? '0 0 20px rgba(200,240,90,0.6)' : 'none',
			};
		}),
		headX: X(f.head),
		ghostX: X(4),
		ghostOp: f.drop ? 1 : 0,
		ghost: L('旧的 v5 已丢弃', 'old v5 dropped'),
		canUndo: f.head > 0 ? '✓' : '✗',
		canRedo: f.head < f.len - 1 ? '✓' : '✗',
		undoN: f.head,
		redoN: f.len - 1 - f.head,
		undoColor: f.head > 0 ? LIME : '#878a81',
		redoColor: f.head < f.len - 1 ? '#9ecbff' : '#878a81',
		cap: f.drop ? L('撤销后再提交会开启新分支，可重做记录随之清空', 'Committing after undo starts a new branch and clears redo') : L('当前版本左侧是可撤销的历史，右侧是可重做的版本', 'Left of HEAD: undo history. Right: redo-able versions'),
		capColor: f.drop ? AMB : '#b9bcb2',
	};

	// 3 — persistence
	const k3 = s % 6;
	const rowsD = [
		['Store[T]', '{count: 9}'],
		['VersionedJSONCodec', '{"schemaVersion":2,"state":{…}}'],
		['AESGCMCodec', L('密钥 k1 · 随机数 · 9f3a…c71e', 'kid=k1 · nonce · 9f3a…c71e')],
		['FileStorage · Journal · Durable', L('写日志 → 刷盘 → 原子替换', 'journal → fsync → rename')],
		['state.json', L('✓ 已写入 · 备份已更新', '✓ saved · backup updated')],
	];
	const tops = [16, 31, 46, 61, 76];
	const v3 = {
		title: L('持久化栈 · 一次写入的旅程', 'PERSISTENCE STACK · ONE SAVE'),
		rows: rowsD.map((r, i) => {
			const act = i === k3;
			const done = i < k3 || k3 === 5;
			return {
				name: r[0],
				out: r[1],
				top: `${tops[i]}%`,
				outOp: act || done ? 1 : 0,
				border: act ? LIME : done ? 'rgba(200,240,90,0.3)' : DIM,
				bg: act ? 'rgba(200,240,90,0.07)' : '#101216',
				nameColor: act ? '#eef0ea' : done ? '#b9bcb2' : '#7c7f75',
				outColor: i === 4 ? LIME : act ? '#f0c674' : '#9ea197',
			};
		}),
		pkY: `${tops[Math.min(k3, 4)] + 5.5}%`,
		pkOp: k3 === 5 ? 0 : 1,
		cap: [
			L('防抖结束，取最新快照', 'Debounce ends — take the latest snapshot'),
			L('写入 schemaVersion，方便日后做版本迁移', 'Wrap with schemaVersion for future migrations'),
			L('AES-GCM 认证加密，只保存密钥 ID 和密文', 'AES-GCM authenticated encryption — key ID + ciphertext only'),
			L('先写日志再原子替换，崩溃也能恢复', 'Journal first, then atomic replace — crash-safe'),
			L('落盘完成，同时刷新备份', 'On disk, backup refreshed'),
			L('状态与字节分离：每一层都可替换', 'State apart from bytes — every layer is replaceable'),
		][k3],
		capColor: k3 >= 4 ? LIME : '#b9bcb2',
	};

	// 4 — async latest wins
	const f4 = s % 5;
	const base4 = 46 + Math.floor(s / 5);
	const canceled = L('已取消（仍在返回）', 'canceled (still returning)');
	const run1 = [
		{ w: ph ? '42%' : '0%', c: AMB, st: L('运行中…', 'running…'), tr: 'width 1.6s linear' },
		{ w: '55%', c: '#7c7f75', st: canceled, tr: 'width 1.6s linear' },
		{ w: '78%', c: '#7c7f75', st: canceled, tr: 'width 1.6s linear' },
		{ w: '100%', c: RED, st: L('迟到 → 被拒绝', 'late → rejected'), tr: 'width .6s ease-out' },
		{ w: '100%', c: RED, st: L('被拒绝', 'rejected'), tr: 'none' },
	][f4];
	const run2 = [
		{ w: '0%', c: '#878a81', st: L('未开始', 'not started'), tr: 'none' },
		{ w: ph ? '70%' : '0%', c: LIME, st: L('运行中…', 'running…'), tr: 'width 1.6s linear' },
		{ w: '100%', c: LIME, st: L('完成 → 提交', 'done → commit'), tr: 'width .5s ease-out' },
		{ w: '100%', c: LIME, st: L('完成', 'done'), tr: 'none' },
		{ w: '100%', c: LIME, st: L('完成', 'done'), tr: 'none' },
	][f4];
	const applied = f4 >= 3 || (f4 === 2 && ph === 2);
	const v4 = {
		title: L('异步动作 · 最新优先', 'ASYNC ACTION · LATESTWINS'),
		runs: [
			{ name: 'Run #1  search("no")', top: '24%', w: run1.w, color: run1.c, status: run1.st, tr: run1.tr },
			{ name: 'Run #2  search("nodra")', top: '46%', w: run2.w, color: run2.c, status: run2.st, tr: run2.tr },
		],
		ver: VV(applied ? base4 + 1 : base4),
		res: applied ? '"nodra" ×12' : '—',
		storeBorder: f4 === 3 ? RED : applied && f4 === 2 ? LIME : 'rgba(238,240,234,0.18)',
		storeShadow: f4 === 3 ? '0 0 40px -10px rgba(255,107,90,0.6)' : applied && f4 === 2 ? '0 0 40px -10px rgba(200,240,90,0.6)' : 'none',
		note: f4 === 3 ? L('拒绝过时的 #1', 'stale #1 rejected') : applied ? L('#2 的结果', 'from #2') : L('等待结果', 'waiting'),
		noteColor: f4 === 3 ? RED : applied ? LIME : '#7c7f75',
		cap: [
			L('#1 开始搜索', '#1 starts searching'),
			L('#2 启动，#1 的上下文被取消', '#2 starts; #1’s context is canceled'),
			L('#2 先完成，加锁、校验代际、提交', '#2 finishes first — lock, check generation, commit'),
			L('#1 迟到返回：越不过 #2 的提交边界，被丢弃', '#1 returns late — it can’t pass #2’s commit, so it’s dropped'),
			L('新结果永远不会被旧结果覆盖', 'Newer results are never overwritten by older ones'),
		][f4],
		capColor: f4 === 3 ? RED : f4 === 2 ? LIME : '#b9bcb2',
	};

	// 5 — MyGo selective redraw
	const seq = ['count', 'title', 'count', 'title', 'title', 'count'];
	const done5 = (s % 6) + (ph === 2 ? 1 : 0);
	const loops = Math.floor(s / 6);
	let cnt = 7 + loops * 3;
	let rd = loops * 3;
	for (let i = 0; i < done5; i++) {
		if (seq[i % 6] === 'count') {
			cnt++;
			rd++;
		}
	}
	const ev = seq[s % 6];
	const hit = ph === 2 && ev === 'count';
	const skip = ph === 2 && ev === 'title';
	const v5 = {
		title: L('MyGo · 只在关心的值变化时重绘', 'MYGO · REDRAW ONLY WHEN IT MATTERS'),
		count: cnt,
		commits: 20 + s + (ph === 2 ? 1 : 0),
		redraws: rd,
		winBorder: hit ? LIME : 'rgba(238,240,234,0.14)',
		winShadow: hit ? '0 0 40px -10px rgba(200,240,90,0.6)' : 'none',
		numColor: hit ? LIME : '#eef0ea',
		ev: ph < 2 ? (ev === 'count' ? 'Update · count++' : 'Update · title = …') : hit ? L('count 变了，请求重绘', 'count changed → Invalidate()') : L('title 变了，选择器判定无变化，不重绘', 'title changed · selector Δ=0 → no redraw'),
		evBg: hit ? 'rgba(200,240,90,0.14)' : skip ? 'rgba(238,240,234,0.06)' : 'rgba(240,198,116,0.12)',
		evColor: hit ? LIME : skip ? '#9ea197' : AMB,
		cap: L('窗口只订阅 s.Count：无论 title 如何变化，都不会触发重绘', 'The window selects s.Count — title changes never trigger a redraw'),
		capColor: '#b9bcb2',
	};

	return { v1, v2, v3, v4, v5 };
}
