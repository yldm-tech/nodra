import type { LangIndex, Tr } from '../lib/lang';

// The five feature diagrams share one clock: `step` advances every 2 s and `ph` walks 0 → 1 → 2 within each step.

const LIME = '#c8f05a';
const AMB = '#f0c674';
const RED = '#ff6b5a';
const DIM = 'rgba(238,240,234,0.1)';
const VV = (n: number) => `v${String(n).padStart(4, '0')}`;

export function sectionsView(step: number, ph: 0 | 1 | 2, li: LangIndex) {
	const s = step;
	const L = (...t: Tr) => t[li];

	// 1 — one owner
	const lane = s % 3;
	const ys = [26, 50, 74];
	const cur = 100 + s - (ph === 2 ? 0 : 1);
	const waiting = (s + 1) % 3;
	const v1 = {
		title: L('一个所有者 · 并发写入被串行化', 'ONE OWNER · CONCURRENT WRITES SERIALIZED', '単一の所有者 · 並行書き込みを直列化', '단일 소유자 · 동시 쓰기를 직렬화', 'UN SOLO DUEÑO · ESCRITURAS EN SERIE', 'UN PROPRIÉTAIRE · ÉCRITURES SÉRIALISÉES'),
		lanes: ['goroutine A', 'goroutine B', 'goroutine C'].map((name, k) => ({
			name,
			top: `${ys[k]}%`,
			status:
				k === lane
					? ph === 2
						? L('已提交', 'committed', 'コミット済み', '커밋됨', 'commit hecho', 'commité')
						: L('写入中…', 'writing…', '書き込み中…', '쓰는 중…', 'escribe…', 'écrit…')
					: k === waiting && ph < 2
						? L('等待锁', 'waiting', 'ロック待ち', '잠금 대기', 'esperando', 'en attente')
						: L('空闲', 'idle', 'アイドル', '유휴', 'inactivo', 'inactif'),
			color: k === lane ? LIME : k === waiting && ph < 2 ? AMB : '#878a81',
			border: k === lane ? LIME : DIM,
		})),
		ver: VV(cur),
		lockOp: ph === 1 ? 1 : 0,
		storeBorder: ph === 1 ? AMB : ph === 2 ? LIME : 'rgba(238,240,234,0.18)',
		storeShadow: ph === 2 ? '0 0 40px -10px rgba(200,240,90,0.6)' : 'none',
		owner: L('唯一的状态所有者', 'single owner', '唯一の所有者', '단일 소유자', 'dueño único', 'seul propriétaire'),
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
				? L(
						`goroutine ${'ABC'[lane]} 提交 ${VV(cur)}：版本连续，没有并发写丢失`,
						`goroutine ${'ABC'[lane]} commits ${VV(cur)} — contiguous versions, no lost writes`,
						`goroutine ${'ABC'[lane]} が ${VV(cur)} をコミット：バージョンは連続、書き込みの消失なし`,
						`goroutine ${'ABC'[lane]}가 ${VV(cur)} 커밋: 버전은 연속, 유실된 쓰기 없음`,
						`goroutine ${'ABC'[lane]} hace commit de ${VV(cur)}: versiones contiguas, nada se pierde`,
						`goroutine ${'ABC'[lane]} commite ${VV(cur)} : versions contiguës, rien de perdu`,
					)
				: L(
						'多个 goroutine 同时调用 Update，Store 依次加锁处理',
						'Many goroutines call Update; the Store takes them one at a time',
						'複数の goroutine が Update を呼び、Store が 1 つずつ処理',
						'여러 goroutine이 Update를 호출하고 Store가 하나씩 처리',
						'Varias goroutines llaman a Update; el Store las atiende de una en una',
						'Plusieurs goroutines appellent Update ; le Store les traite une à une',
					),
		capColor: ph === 2 ? LIME : '#b9bcb2',
	};

	// 2 — history
	const F = [
		{ len: 4, head: 3, a: 'Update → v4' },
		{ len: 5, head: 4, a: 'Update → v5' },
		{ len: 5, head: 3, a: 'Undo() ↶' },
		{ len: 5, head: 2, a: 'Undo() ↶' },
		{ len: 5, head: 3, a: 'Redo() ↷' },
		{
			len: 5,
			head: 4,
			drop: true,
			a: L('Update：产生新版本，旧的可重做记录被丢弃', 'Update → new version, old redo dropped', 'Update → 新バージョン、古い Redo は破棄', 'Update → 새 버전, 이전 Redo는 폐기', 'Update → versión nueva, sin redo', 'Update → nouvelle version, redo perdu'),
		},
	];
	const f = F[s % F.length];
	const X = (i: number) => `${12 + i * 19}%`;
	const v2 = {
		title: L('撤销与重做 · Track(store)', 'UNDO / REDO · Track(store)', 'Undo / Redo · Track(store)', '실행 취소 / 다시 실행 · Track(store)', 'DESHACER / REHACER · Track(store)', 'ANNULER / RÉTABLIR · Track(store)'),
		act: f.a,
		actColor: f.drop ? AMB : f.a.startsWith('U') && f.a.includes('do') ? '#9ecbff' : LIME,
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
		ghost: L('旧的 v5 已丢弃', 'old v5 dropped', '古い v5 は破棄', '이전 v5 폐기', 'v5 antigua descartada', 'ancienne v5 jetée'),
		canUndo: f.head > 0 ? '✓' : '✗',
		canRedo: f.head < f.len - 1 ? '✓' : '✗',
		undoN: f.head,
		redoN: f.len - 1 - f.head,
		undoColor: f.head > 0 ? LIME : '#878a81',
		redoColor: f.head < f.len - 1 ? '#9ecbff' : '#878a81',
		cap: f.drop
			? L(
					'撤销后再提交会开启新分支，可重做记录随之清空',
					'Committing after undo starts a new branch and clears redo',
					'Undo 後にコミットすると新しい分岐になり、Redo は消える',
					'실행 취소 후 커밋하면 새 분기가 시작되고 다시 실행이 비워짐',
					'Un commit tras deshacer abre una rama nueva y borra el redo',
					'Un commit après annulation ouvre une branche et vide le redo',
				)
			: L(
					'当前版本左侧是可撤销的历史，右侧是可重做的版本',
					'Left of HEAD: undo history. Right: redo-able versions',
					'現在の左は Undo できる履歴、右は Redo できるバージョン',
					'현재 위치 왼쪽은 실행 취소 기록, 오른쪽은 다시 실행할 버전',
					'A la izquierda, lo que se puede deshacer; a la derecha, rehacer',
					'À gauche, ce qu’on peut annuler ; à droite, rétablir',
				),
		capColor: f.drop ? AMB : '#b9bcb2',
	};

	// 3 — persistence
	const k3 = s % 6;
	const rowsD = [
		['Store[T]', '{count: 9}'],
		['VersionedJSONCodec', '{"schemaVersion":2,"state":{…}}'],
		['AESGCMCodec', L('密钥 k1 · 随机数 · 9f3a…c71e', 'kid=k1 · nonce · 9f3a…c71e', '鍵 k1 · nonce · 9f3a…c71e', '키 k1 · nonce · 9f3a…c71e', 'clave k1 · nonce · 9f3a…c71e', 'clé k1 · nonce · 9f3a…c71e')],
		['FileStorage · Journal · Durable', L('写日志 → 刷盘 → 原子替换', 'journal → fsync → rename', 'journal → fsync → rename', '저널 → fsync → rename', 'journal → fsync → rename', 'journal → fsync → rename')],
		['state.json', L('✓ 已写入 · 备份已更新', '✓ saved · backup updated', '✓ 保存済み · バックアップ更新', '✓ 저장됨 · 백업 갱신', '✓ guardado · copia al día', '✓ enregistré · sauvegarde à jour')],
	];
	const tops = [16, 31, 46, 61, 76];
	const v3 = {
		title: L('持久化栈 · 一次写入的旅程', 'PERSISTENCE STACK · ONE SAVE', '永続化スタック · 1 回の保存', '영속화 스택 · 한 번의 저장', 'PILA DE PERSISTENCIA · UN GUARDADO', 'PILE DE PERSISTANCE · UNE SAUVEGARDE'),
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
			L('防抖结束，取最新快照', 'Debounce ends — take the latest snapshot', 'デバウンス終了、最新のスナップショットを取得', '디바운스가 끝나면 최신 스냅숏을 가져옴', 'Acaba el debounce: se toma la última instantánea', 'Fin du debounce : on prend le dernier instantané'),
			L('写入 schemaVersion，方便日后做版本迁移', 'Wrap with schemaVersion for future migrations', '将来のマイグレーションに備えて schemaVersion で包む', '향후 마이그레이션을 위해 schemaVersion으로 감쌈', 'Se envuelve con schemaVersion para migrar después', 'Enveloppé avec schemaVersion pour migrer plus tard'),
			L('AES-GCM 认证加密，只保存密钥 ID 和密文', 'AES-GCM authenticated encryption — key ID + ciphertext only', 'AES-GCM 認証付き暗号化。保存するのは鍵 ID と暗号文だけ', 'AES-GCM 인증 암호화: 키 ID와 암호문만 저장', 'Cifrado autenticado AES-GCM: solo ID de clave y texto cifrado', 'Chiffrement authentifié AES-GCM : ID de clé et chiffré seuls'),
			L('先写日志再原子替换，崩溃也能恢复', 'Journal first, then atomic replace — crash-safe', '先にジャーナル、次にアトミックに置換。クラッシュしても安全', '저널을 먼저 쓰고 원자적으로 교체: 충돌에도 안전', 'Primero el journal, luego reemplazo atómico: resiste fallos', 'Journal d’abord, puis remplacement atomique : résiste aux crashs'),
			L('落盘完成，同时刷新备份', 'On disk, backup refreshed', 'ディスクに保存し、バックアップも更新', '디스크에 저장하고 백업도 갱신', 'En disco, con la copia renovada', 'Sur disque, sauvegarde rafraîchie'),
			L('状态与字节分离：每一层都可替换', 'State apart from bytes — every layer is replaceable', '状態とバイト列を分離。どの層も差し替え可能', '상태와 바이트를 분리: 모든 계층은 교체 가능', 'Estado separado de bytes: cada capa se puede cambiar', 'L’état séparé des octets : chaque couche se remplace'),
		][k3],
		capColor: k3 >= 4 ? LIME : '#b9bcb2',
	};

	// 4 — async latest wins
	const f4 = s % 5;
	const base4 = 46 + Math.floor(s / 5);
	const canceled = L('已取消（仍在返回）', 'canceled (still returning)', 'キャンセル（戻り待ち）', '취소됨 (아직 반환 중)', 'cancelado (aún vuelve)', 'annulé (revient encore)');
	const running = L('运行中…', 'running…', '実行中…', '실행 중…', 'en curso…', 'en cours…');
	const done = L('完成', 'done', '完了', '완료', 'listo', 'terminé');
	const run1 = [
		{ w: ph ? '42%' : '0%', c: AMB, st: running, tr: 'width 1.6s linear' },
		{ w: '55%', c: '#7c7f75', st: canceled, tr: 'width 1.6s linear' },
		{ w: '78%', c: '#7c7f75', st: canceled, tr: 'width 1.6s linear' },
		{ w: '100%', c: RED, st: L('迟到 → 被拒绝', 'late → rejected', '遅延 → 拒否', '늦음 → 거부', 'tarde → rechazado', 'en retard → rejeté'), tr: 'width .6s ease-out' },
		{ w: '100%', c: RED, st: L('被拒绝', 'rejected', '拒否', '거부됨', 'rechazado', 'rejeté'), tr: 'none' },
	][f4];
	const run2 = [
		{ w: '0%', c: '#878a81', st: L('未开始', 'not started', '未開始', '시작 전', 'sin iniciar', 'pas lancé'), tr: 'none' },
		{ w: ph ? '70%' : '0%', c: LIME, st: running, tr: 'width 1.6s linear' },
		{ w: '100%', c: LIME, st: L('完成 → 提交', 'done → commit', '完了 → コミット', '완료 → 커밋', 'listo → commit', 'terminé → commit'), tr: 'width .5s ease-out' },
		{ w: '100%', c: LIME, st: done, tr: 'none' },
		{ w: '100%', c: LIME, st: done, tr: 'none' },
	][f4];
	const applied = f4 >= 3 || (f4 === 2 && ph === 2);
	const v4 = {
		title: L('异步动作 · 最新优先', 'ASYNC ACTION · LATESTWINS', '非同期アクション · LatestWins', '비동기 작업 · LatestWins', 'ACCIÓN ASÍNCRONA · LATESTWINS', 'ACTION ASYNCHRONE · LATESTWINS'),
		runs: [
			{ name: 'Run #1  search("no")', top: '24%', w: run1.w, color: run1.c, status: run1.st, tr: run1.tr },
			{ name: 'Run #2  search("nodra")', top: '46%', w: run2.w, color: run2.c, status: run2.st, tr: run2.tr },
		],
		ver: VV(applied ? base4 + 1 : base4),
		res: applied ? '"nodra" ×12' : '—',
		storeBorder: f4 === 3 ? RED : applied && f4 === 2 ? LIME : 'rgba(238,240,234,0.18)',
		storeShadow: f4 === 3 ? '0 0 40px -10px rgba(255,107,90,0.6)' : applied && f4 === 2 ? '0 0 40px -10px rgba(200,240,90,0.6)' : 'none',
		note:
			f4 === 3
				? L('拒绝过时的 #1', 'stale #1 rejected', '古い #1 を拒否', '오래된 #1 거부', '#1 obsoleto rechazado', '#1 obsolète rejeté')
				: applied
					? L('#2 的结果', 'from #2', '#2 の結果', '#2의 결과', 'de #2', 'de #2')
					: L('等待结果', 'waiting', '結果待ち', '결과 대기', 'esperando', 'en attente'),
		noteColor: f4 === 3 ? RED : applied ? LIME : '#7c7f75',
		cap: [
			L('#1 开始搜索', '#1 starts searching', '#1 が検索を開始', '#1이 검색 시작', '#1 empieza a buscar', '#1 lance la recherche'),
			L('#2 启动，#1 的上下文被取消', '#2 starts; #1’s context is canceled', '#2 が開始し、#1 のコンテキストはキャンセル', '#2 시작, #1의 컨텍스트는 취소됨', 'Empieza #2; se cancela el contexto de #1', '#2 démarre ; le contexte de #1 est annulé'),
			L('#2 先完成，加锁、校验代际、提交', '#2 finishes first — lock, check generation, commit', '#2 が先に完了。ロック、世代を確認、コミット', '#2가 먼저 완료: 잠금, 세대 확인, 커밋', '#2 acaba primero: bloqueo, generación, commit', '#2 finit en premier : verrou, génération, commit'),
			L(
				'#1 迟到返回：越不过 #2 的提交边界，被丢弃',
				'#1 returns late — it can’t pass #2’s commit, so it’s dropped',
				'#1 が遅れて戻るが、#2 のコミットを越えられず破棄',
				'#1이 늦게 반환: #2의 커밋을 넘지 못해 버려짐',
				'#1 vuelve tarde: no pasa el commit de #2 y se descarta',
				'#1 revient en retard : il ne passe pas le commit de #2, jeté',
			),
			L('新结果永远不会被旧结果覆盖', 'Newer results are never overwritten by older ones', '新しい結果が古い結果で上書きされることはない', '새 결과는 이전 결과로 덮어쓰이지 않음', 'Un resultado nuevo nunca lo pisa uno viejo', 'Un résultat récent n’est jamais écrasé par un ancien'),
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
		title: L('MyGo · 只在关心的值变化时重绘', 'MYGO · REDRAW ONLY WHEN IT MATTERS', 'MyGo · 必要なときだけ再描画', 'MyGo · 필요할 때만 다시 그리기', 'MYGO · REDIBUJA SOLO SI IMPORTA', 'MYGO · REDESSINER SI NÉCESSAIRE'),
		count: cnt,
		commits: 20 + s + (ph === 2 ? 1 : 0),
		redraws: rd,
		winBorder: hit ? LIME : 'rgba(238,240,234,0.14)',
		winShadow: hit ? '0 0 40px -10px rgba(200,240,90,0.6)' : 'none',
		numColor: hit ? LIME : '#eef0ea',
		ev:
			ph < 2
				? ev === 'count'
					? 'Update · count++'
					: 'Update · title = …'
				: hit
					? L('count 变了，请求重绘', 'count changed → Invalidate()', 'count が変化 → Invalidate()', 'count 변경 → Invalidate()', 'count cambió → Invalidate()', 'count a changé → Invalidate()')
					: L('title 变了，选择器判定无变化，不重绘', 'title changed · selector Δ=0 → no redraw', 'title が変化 · セレクター Δ=0 → 再描画なし', 'title 변경 · 선택자 Δ=0 → 다시 그리지 않음', 'title cambió · selector Δ=0 → no redibuja', 'title a changé · sélecteur Δ=0 → pas de rendu'),
		evBg: hit ? 'rgba(200,240,90,0.14)' : skip ? 'rgba(238,240,234,0.06)' : 'rgba(240,198,116,0.12)',
		evColor: hit ? LIME : skip ? '#9ea197' : AMB,
		cap: L(
			'窗口只订阅 s.Count：无论 title 如何变化，都不会触发重绘',
			'The window selects s.Count — title changes never trigger a redraw',
			'ウィンドウは s.Count だけを選択。title が変わっても再描画しない',
			'창은 s.Count만 선택: title이 바뀌어도 다시 그리지 않음',
			'La ventana selecciona s.Count: un cambio de title no redibuja',
			'La fenêtre sélectionne s.Count : changer title ne redessine pas',
		),
		capColor: '#b9bcb2',
	};

	return { v1, v2, v3, v4, v5 };
}
