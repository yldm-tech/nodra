import type { Mood } from '../components/Mascot';
import type { LangIndex, Tr } from '../lib/lang';

// The hero diagram: a looping 17.4 s script of goroutines writing to one Store — a plain update, two contending writers, then a rolled-back UpdateErr.

type Point = [number, number];
type GorState = '' | 'w' | 'q' | 'e';
type SubState = 'idle' | 'hit' | 'skip' | 'none' | 'pend' | 'saved';

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
	cap: [Tr, string];
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
		cap: [['等待写入…', 'Waiting for a write…', '書き込み待ち…', '쓰기 대기 중…', 'Esperando una escritura…', 'En attente d’une écriture…'], GREY],
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
	const cap = (t: Tr, c = LIME) => {
		M.cap = [t, c];
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
		[0, () => { place(0, G[0], LIME, 'Update · count++'); M.gor[0] = 'w'; cap(['goroutine A 调用 store.Update(fn)', 'goroutine A calls store.Update(fn)', 'goroutine A が store.Update(fn) を呼ぶ', 'goroutine A가 store.Update(fn) 호출', 'goroutine A llama a store.Update(fn)', 'goroutine A appelle store.Update(fn)']); }],
		[60, () => move(0, IN, 900)],
		[980, () => { hide(0); M.lock = true; M.cand = [`candidate · count ${M.count} → ${M.count + 1}`, LIME]; cap([`已加写锁，回调只修改候选值；此时所有读取仍看到 ${V(M.ver)}`, `Write lock held; the callback edits a candidate — readers still see ${V(M.ver)}`, `書き込みロック中。コールバックは候補値だけを変更し、読み取りには ${V(M.ver)} が見える`, `쓰기 잠금 중: 콜백은 후보 값만 수정하고, 읽기는 여전히 ${V(M.ver)}`, `Bloqueo tomado; el callback edita un candidato y las lecturas ven ${V(M.ver)}`, `Verrou pris ; le callback modifie un candidat, les lectures voient ${V(M.ver)}`], AMBER); }],
		[2000, () => { M.lock = false; M.cand = null; M.ver++; M.count++; M.glow = 'ok'; M.gor[0] = ''; cap([`提交 → ${V(M.ver)}，释放锁后再通知订阅者`, `Commit → ${V(M.ver)}, unlock, then notify outside the lock`, `コミット → ${V(M.ver)}、ロックを解放してから通知`, `커밋 → ${V(M.ver)}, 잠금을 푼 뒤 알림`, `Commit → ${V(M.ver)}, se libera el bloqueo y luego se avisa`, `Commit → ${V(M.ver)}, verrou relâché, puis notification`]); }],
		...notify([false, true, false, false], 2100),
		[3000, () => cap(['count 变了，窗口重绘；title 没变，选择器跳过；历史记录已保存；落盘进入防抖', 'Count changed → redraw; Title unchanged → skipped; History records; save is debounced', 'count が変化 → 再描画、title は不変 → スキップ、履歴を記録、保存はデバウンス', 'count 변경 → 다시 그리기, title 그대로 → 건너뜀, 기록 저장, 저장은 디바운스', 'count cambió → redibujo; title igual → omitido; Historial registra; guardado con debounce', 'count change → rendu ; title identique → ignoré ; Historique note ; sauvegarde en debounce'])],
		[4300, () => { M.per = { ...M.per, pend: false, saved: M.ver }; M.subs[3] = 'saved'; }],
		[5400, () => { M.subs = ['idle', 'idle', 'idle', 'idle']; place(0, G[1], LIME, 'Update · title'); place(5, G[2], LIME, 'Update · count++'); M.gor[1] = 'w'; M.gor[2] = 'w'; cap(['goroutine B 和 C 同时写入', 'goroutines B and C write at the same time', 'goroutine B と C が同時に書き込む', 'goroutine B와 C가 동시에 쓰기', 'Las goroutines B y C escriben a la vez', 'Les goroutines B et C écrivent en même temps']); }],
		[5460, () => { move(0, IN, 900); move(5, Q, 820); }],
		[6380, () => { hide(0); M.lock = true; M.cand = [`candidate · title "${other()}"`, LIME]; M.pk[5] = { ...M.pk[5], bg: AMBER, label: 'C · waiting for lock' }; M.gor[2] = 'q'; cap(['B 拿到写锁；C 在锁外排队，不会读到 B 的中间状态', 'B holds the lock; C queues outside and never sees a half-done value', 'B がロックを保持。C は外で待ち、途中の値は見えない', 'B가 잠금을 잡음: C는 밖에서 기다리며 중간 값을 보지 못함', 'B tiene el bloqueo; C espera fuera y nunca ve un valor a medias', 'B tient le verrou ; C attend dehors et ne voit aucune valeur partielle'], AMBER); }],
		[7400, () => { M.lock = false; M.cand = null; M.ver++; M.title = other(); M.glow = 'ok'; M.gor[1] = ''; cap([`B 提交 ${V(M.ver)}`, `B commits ${V(M.ver)}`, `B が ${V(M.ver)} をコミット`, `B가 ${V(M.ver)} 커밋`, `B hace commit de ${V(M.ver)}`, `B commite ${V(M.ver)}`]); }],
		...notify([true, false, false, false], 7500),
		[7600, () => { M.pk[5] = { ...M.pk[5], bg: LIME, label: 'Update · count++' }; move(5, IN, 520); M.gor[2] = 'w'; }],
		[8160, () => { hide(5); M.lock = true; M.cand = [`candidate · count ${M.count} → ${M.count + 1}`, LIME]; cap(['锁释放后 C 立即拿到锁', 'C takes the lock as soon as it is released', 'ロックが解放されるとすぐ C が取得', '잠금이 풀리자마자 C가 획득', 'C toma el bloqueo en cuanto se libera', 'C prend le verrou dès qu’il est relâché'], AMBER); }],
		[9200, () => { M.lock = false; M.cand = null; M.ver++; M.count++; M.glow = 'ok'; M.gor[2] = ''; cap([`C 提交 ${V(M.ver)}：两次写入严格串行，版本号连续`, `C commits ${V(M.ver)} — writes are serialized and versions stay contiguous`, `C が ${V(M.ver)} をコミット。書き込みは直列化され、バージョンは連続`, `C가 ${V(M.ver)} 커밋: 쓰기는 직렬화되고 버전은 연속`, `C hace commit de ${V(M.ver)}: escrituras en serie, versiones contiguas`, `C commite ${V(M.ver)} : écritures sérialisées, versions contiguës`]); }],
		...notify([false, true, false, false], 9300),
		[10200, () => cap(['持久化的防抖把两次提交合并成一次磁盘写入', 'Persistence debounce merged both commits into one disk write', '永続化のデバウンスで 2 回のコミットを 1 回の書き込みに', '영속화 디바운스가 두 커밋을 한 번의 디스크 쓰기로 합침', 'El debounce de persistencia juntó ambos commits en una escritura', 'Le debounce de persistance a fusionné les deux commits en une écriture'])],
		[11500, () => { M.per = { ...M.per, pend: false, saved: M.ver }; M.subs[3] = 'saved'; }],
		[12600, () => { M.subs = ['idle', 'idle', 'idle', 'idle']; place(0, G[0], LIME, 'UpdateErr · count = -1'); M.gor[0] = 'w'; cap(['goroutine A 调用 UpdateErr，试图写入非法值', 'goroutine A calls UpdateErr with an invalid value', 'goroutine A が不正な値で UpdateErr を呼ぶ', 'goroutine A가 잘못된 값으로 UpdateErr 호출', 'goroutine A llama a UpdateErr con un valor inválido', 'goroutine A appelle UpdateErr avec une valeur invalide']); }],
		[12660, () => move(0, IN, 900)],
		[13580, () => { hide(0); M.lock = true; M.cand = ['candidate · count -1  ✗ validate', RED]; cap(['回调返回错误（panic 也一样）', 'The callback returns an error — a panic behaves the same', 'コールバックがエラーを返す（panic でも同じ）', '콜백이 오류를 반환 (panic도 동일)', 'El callback devuelve un error; un panic hace lo mismo', 'Le callback renvoie une erreur ; un panic fait pareil'], AMBER); }],
		[14600, () => { M.lock = false; M.cand = null; M.glow = 'err'; M.subs = ['none', 'none', 'none', 'none']; place(0, IN, RED, '✗ error'); cap([`整体回滚：${V(M.ver)} 保持不变，没有订阅者被通知，错误返回给调用方`, `Rolled back: ${V(M.ver)} unchanged, 0 subscribers notified, error returned`, `ロールバック：${V(M.ver)} は不変、通知した購読者 0、エラーを返す`, `롤백: ${V(M.ver)} 그대로, 알림받은 구독자 0, 오류 반환`, `Rollback: ${V(M.ver)} sin cambios, 0 suscriptores avisados, error devuelto`, `Rollback : ${V(M.ver)} inchangée, 0 abonné notifié, erreur renvoyée`], RED); }],
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

const GSUB: Record<GorState, [label: Tr, color: string]> = {
	'': [['空闲', 'idle', 'アイドル', '유휴', 'inactivo', 'inactif'], '#878a81'],
	w: [['写入中…', 'writing…', '書き込み中…', '쓰는 중…', 'escribe…', 'écrit…'], '#c8f05a'],
	q: [['等待锁', 'waiting', 'ロック待ち', '잠금 대기', 'esperando', 'en attente'], '#f0c674'],
	e: [['收到错误', 'got error', 'エラー受信', '오류 받음', 'con error', 'en erreur'], '#ff6b5a'],
};
const GBORDER: Record<GorState, string> = { '': 'rgba(238,240,234,0.1)', w: '#c8f05a', q: '#f0c674', e: '#ff6b5a' };
const DIM = 'rgba(238,240,234,0.1)';

export function flowView(M: FlowModel | null, li: LangIndex) {
	if (!M) M = fresh();
	const L = (...t: Tr) => t[li];
	const pad = (n: number) => String(n).padStart(4, '0');
	const names = [
		L('窗口 · count', 'Window · Count', 'ウィンドウ · Count', '창 · Count', 'Ventana · Count', 'Fenêtre · Count'),
		L('窗口 · title', 'Window · Title', 'ウィンドウ · Title', '창 · Title', 'Ventana · Title', 'Fenêtre · Title'),
		L('历史记录', 'History', '履歴', '기록', 'Historial', 'Historique'),
		L('持久化', 'Persistence', '永続化', '영속화', 'Persistencia', 'Persistance'),
	];
	const subs = M.subs.map((st, k) => {
		let val: string;
		let note: string;
		let noteColor = '#7a7d74';
		let border = DIM;
		let valColor = '#eef0ea';
		let deco = 'none';
		if (k === 0) val = String(M.win[0]);
		else if (k === 1) val = `"${M.win[1]}"`;
		else if (k === 2) val = L('可撤销 ×', 'undo ×', 'Undo ×', '실행 취소 ×', 'deshacer ×', 'annuler ×') + M.hist;
		else val = M.per.pend ? L('防抖 100 毫秒…', 'debounce 100ms…', 'デバウンス 100ms…', '디바운스 100ms…', 'debounce 100ms…', 'debounce 100ms…') : L('已保存 v', 'saved v', '保存済み v', '저장됨 v', 'guardado v', 'sauvé v') + pad(M.per.saved);
		if (st === 'hit') {
			note = k === 2 ? L('已记录快照', 'snapshot recorded', 'スナップショット記録', '스냅숏 기록됨', 'instantánea guardada', 'instantané noté') : L('变化 → 重绘', 'changed → redraw', '変化 → 再描画', '변경 → 다시 그리기', 'cambió → redibujo', 'changé → rendu');
			noteColor = border = valColor = '#c8f05a';
		} else if (st === 'skip') {
			note = L('Δ = 0 · 跳过', 'Δ = 0 · skipped', 'Δ = 0 · スキップ', 'Δ = 0 · 건너뜀', 'Δ = 0 · omitido', 'Δ = 0 · ignoré');
			noteColor = '#9ea197';
		} else if (st === 'none') {
			note = L('未通知', 'not notified', '通知なし', '알림 없음', 'sin aviso', 'non notifié');
			deco = 'line-through';
		} else if (st === 'pend') {
			note = L('合并写入中…', 'merging writes…', '書き込みを統合中…', '쓰기 병합 중…', 'uniendo escrituras…', 'fusion des écritures…');
			noteColor = border = valColor = '#f0c674';
		} else if (st === 'saved') {
			note = '→ state.json';
			noteColor = border = valColor = '#c8f05a';
		} else note = ['selector(s.Count)', 'selector(s.Title)', 'Track(store)', 'AutoSaveAsync · 100ms'][k];
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
		gors: M.gor.map((s, k) => ({ name: `goroutine ${'ABC'[k]}`, top: `${[20, 50, 80][k]}%`, sub: GSUB[s][0][li], subColor: GSUB[s][1], border: GBORDER[s], shadow: s ? `0 0 20px -8px ${GBORDER[s]}` : 'none' })),
		ver: pad(M.ver),
		count: M.count,
		title: M.title,
		lockOp: M.lock ? 1 : 0,
		storeBorder: M.lock ? '#f0c674' : g === 'ok' ? '#c8f05a' : g === 'err' ? '#ff6b5a' : 'rgba(238,240,234,0.18)',
		storeShadow: M.lock ? '0 0 40px -12px rgba(240,198,116,0.6)' : g === 'ok' ? '0 0 56px -10px rgba(200,240,90,0.6)' : g === 'err' ? '0 0 56px -10px rgba(255,107,90,0.6)' : '0 20px 50px -24px rgba(0,0,0,0.7)',
		cand: M.cand ? M.cand[0].replace('candidate', L('候选值', 'candidate', '候補値', '후보 값', 'candidato', 'candidat')).replace('✗ validate', L('✗ 校验失败', '✗ validate', '✗ 検証失敗', '✗ 검증 실패', '✗ validación', '✗ validation')) : '',
		candColor: M.cand ? M.cand[1] : 'transparent',
		candOp: M.cand ? 1 : 0,
		subs,
		pk: M.pk.map((p) => ({
			...p,
			x: `${p.x}%`,
			y: `${p.y}%`,
			label: p.label.replace('C · waiting for lock', L('C · 等待锁', 'C · waiting for lock', 'C · ロック待ち', 'C · 잠금 대기', 'C · espera el bloqueo', 'C · attend le verrou')).replace('✗ error', L('✗ 错误', '✗ error', '✗ エラー', '✗ 오류', '✗ error', '✗ erreur')),
		})),
		cap: M.cap[0][li],
		capColor: M.cap[1],
		mood,
		bubbleBorder: g === 'err' ? '#ff6b5a' : M.lock ? '#f0c674' : g === 'ok' ? '#c8f05a' : 'rgba(238,240,234,0.14)',
		mascotTag: L('Dra · 守护 {v}', 'DRA · KEEPER OF {v}', 'Dra · {v} の守り手', 'Dra · {v}의 수호자', 'DRA · GUARDIÁN DE {v}', 'DRA · GARDIEN DE {v}').replace('{v}', `v${pad(M.ver)}`),
		mascotLine:
			g === 'err'
				? L('回滚了，什么都没泄露', 'rolled back, nothing leaked', 'ロールバックしたよ、何も漏れてない', '롤백했어, 아무것도 안 샜어', 'rollback hecho, nada se filtró', 'rollback fait, rien n’a fui')
				: M.lock
					? L('我拿着锁呢…', 'holding the lock…', 'ロックを持ってるよ…', '잠금을 잡고 있어…', 'tengo el bloqueo…', 'je tiens le verrou…')
					: g === 'ok'
						? L('新版本！通知大家', 'new version! telling everyone', '新しいバージョン！みんなに知らせるね', '새 버전! 모두에게 알릴게', '¡versión nueva! aviso a todos', 'nouvelle version ! je préviens tout le monde')
						: L('你的状态守护者', 'keeper of your state', 'きみの状態の守り手', '너의 상태를 지키는 수호자', 'guardián de tu estado', 'gardien de ton état'),
	};
}
