import type { LangIndex, Tr } from '../lib/lang';

// "How it works": three scripted scenarios (rollback, batch, async LatestWins) stepped through on a fixed diagram.

type Anchor = 'WIN' | 'STORE' | 'CAND' | 'T1' | 'T2' | 'S0' | 'S1' | 'S2' | 'S3';
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
	subs: { s: SubKind; n?: string | Tr }[];
	persist: number;
	packets: PacketSpec[];
	err: string | null;
	glow: boolean;
	winFlash: boolean;
	line: number;
}

interface Scenario {
	tab: Tr;
	start: Partial<Frame>;
	// A line with a comment is given per language, repeating its Go code unchanged; lines without one are shared.
	code: (string | Tr)[];
	steps: { t: Tr; d: Tr; o: Partial<Frame> }[];
	states: Frame[];
}

export const STEP_MS = 3400;

const ANCH: Record<Anchor, [number, number]> = { WIN: [13.5, 55], STORE: [50, 48], CAND: [50, 86], T1: [39.5, 86], T2: [60.5, 86], S0: [76, 12.5], S1: [76, 35.5], S2: [76, 58.5], S3: [76, 81.5] };
const SUBN: Tr[] = [
	['选择器 · count', 'Selector · Count', 'セレクター · Count', '선택자 · Count', 'Selector · Count', 'Sélecteur · Count'],
	['选择器 · title', 'Selector · Title', 'セレクター · Title', '선택자 · Title', 'Selector · Title', 'Sélecteur · Title'],
	['历史记录', 'History', '履歴', '기록', 'Historial', 'Historique'],
	['持久化', 'Persistence', '永続化', '영속화', 'Persistencia', 'Persistance'],
];
const SUBNOTE: Record<SubKind, Tr> = {
	idle: ['等待中', 'idle', '待機中', '대기 중', 'en espera', 'en attente'],
	hit: ['触发', 'fired', '発火', '실행됨', 'disparado', 'déclenché'],
	skip: ['未变化 · 跳过', 'unchanged · skipped', '変化なし · スキップ', '변화 없음 · 건너뜀', 'sin cambios · omitido', 'inchangé · ignoré'],
	rec: ['已记录', 'recorded', '記録済み', '기록됨', 'registrado', 'enregistré'],
	queued: ['防抖中…', 'debouncing…', 'デバウンス中…', '디바운스 중…', 'en debounce…', 'debounce…'],
	saved: ['已保存至 state.json', 'saved to state.json', 'state.json に保存済み', 'state.json에 저장됨', 'guardado en state.json', 'écrit dans state.json'],
	none: ['未通知', 'not notified', '通知なし', '알림 없음', 'sin aviso', 'non notifié'],
};
const SUBCOLOR: Record<SubKind, string> = { hit: '#c8f05a', skip: '#7c7f75', rec: '#9ecbff', queued: '#f0c674', saved: '#c8f05a', idle: '#7a7d74', none: '#7a7d74' };
const TASKST: Record<TaskKind, [label: Tr, color: string, deco: string]> = {
	run: [['运行中…', 'running…', '実行中…', '실행 중…', 'en curso…', 'en cours…'], '#f0c674', 'none'],
	cancel: [['已取消', 'canceled', 'キャンセル', '취소됨', 'cancelado', 'annulé'], '#7c7f75', 'line-through'],
	done: [['完成', 'done', '完了', '완료', 'listo', 'terminé'], '#c8f05a', 'none'],
	reject: [['已拒绝', 'rejected', '拒否', '거부됨', 'rechazado', 'rejeté'], '#ff6b5a', 'line-through'],
};
// Packet labels and error messages are stored in English; labels missing here (API calls, versions) are shown as is.
const PACKET: Record<string, Tr> = {
	copy: ['复制', 'copy', 'コピー', '복사', 'copia', 'copie'],
	commit: ['提交', 'commit', 'コミット', '커밋', 'commit', 'commit'],
	error: ['错误', 'error', 'エラー', '오류', 'error', 'erreur'],
	apply: ['应用结果', 'apply', '適用', '적용', 'aplicar', 'appliquer'],
	'stale #1': ['过时的 #1', 'stale #1', '古い #1', '오래된 #1', '#1 obsoleto', '#1 obsolète'],
};
const ERR: Record<string, Tr> = {
	'validate: count < 0': ['校验失败：count < 0', 'validate: count < 0', '検証失敗：count < 0', '검증 실패: count < 0', 'validación: count < 0', 'validation : count < 0'],
	'#1 rejected · stale generation': ['#1 被拒绝：已被更新的任务取代', '#1 rejected · stale generation', '#1 を拒否 · 古い世代', '#1 거부 · 오래된 세대', '#1 rechazado · generación obsoleta', '#1 rejeté · génération obsolète'],
};
const UNDO_STACK = (v: string): Tr => [`${v} → 撤销栈`, `${v} → undo stack`, `${v} → Undo スタック`, `${v} → 실행 취소 스택`, `${v} → pila de deshacer`, `${v} → pile d’annulation`];

const IDLE4: Frame['subs'] = [{ s: 'idle' }, { s: 'idle' }, { s: 'idle' }, { s: 'idle' }];
const NONE4: Frame['subs'] = [{ s: 'none' }, { s: 'none' }, { s: 'none' }, { s: 'none' }];
const BASE: Frame = { ver: 42, count: 7, title: 'Nodra', win: 7, lock: false, batch: null, cand: null, tasks: [], subs: IDLE4, persist: 0, packets: [], err: null, glow: false, winFlash: false, line: -1 };

const RAW: Omit<Scenario, 'states'>[] = [
	{
		tab: ['失败回滚', 'UpdateErr rollback', 'UpdateErr ロールバック', 'UpdateErr 롤백', 'Rollback de UpdateErr', 'Rollback UpdateErr'],
		start: { ver: 43, count: 8, win: 8 },
		code: [
			'err := store.UpdateErr(func(s *State) error {',
			'    s.Count = -1',
			'    return validate(*s) // count < 0',
			'})',
			['// err != nil：状态、版本、订阅者均未改变', '// err != nil: state, version and subscribers unchanged', '// err != nil：状態・バージョン・購読者はすべて不変', '// err != nil: 상태, 버전, 구독자 모두 그대로', '// err != nil: estado, versión y suscriptores sin cambios', '// err != nil : état, version et abonnés inchangés'],
		],
		steps: [
			{
				t: ['调用 UpdateErr', 'Call UpdateErr', 'UpdateErr を呼ぶ', 'UpdateErr 호출', 'Llamar a UpdateErr', 'Appeler UpdateErr'],
				d: ['这次回调可以返回错误。', 'This time the callback may return an error.', '今回はコールバックがエラーを返せます。', '이번에는 콜백이 오류를 반환할 수 있습니다.', 'Esta vez el callback puede devolver un error.', 'Cette fois, le callback peut renvoyer une erreur.'],
				o: { winFlash: true, packets: [['UpdateErr(fn)', 'WIN', 'STORE']], line: 0 },
			},
			{
				t: ['加锁，准备候选', 'Lock & prepare a candidate', 'ロックして候補値を用意', '잠그고 후보 값 준비', 'Bloquear y preparar un candidato', 'Verrouiller, préparer un candidat'],
				d: ['与 Update 相同：加锁并生成候选值。', 'Same as Update: lock and prepare a candidate.', 'Update と同じく、ロックして候補値を作ります。', 'Update와 같습니다. 잠그고 후보 값을 만듭니다.', 'Igual que Update: bloquea y prepara un candidato.', 'Comme Update : verrouille et prépare un candidat.'],
				o: { lock: true, cand: { v: 8 }, packets: [['copy', 'STORE', 'CAND']], line: 0 },
			},
			{
				t: ['写入非法值', 'Write an invalid value', '不正な値を書き込む', '잘못된 값 쓰기', 'Escribir un valor inválido', 'Écrire une valeur invalide'],
				d: ['回调把 count 改成 -1，而 Store 中仍然是 8。', 'The callback sets count to -1. The Store itself still holds 8.', 'コールバックは count を -1 にしますが、Store 自体はまだ 8 です。', '콜백이 count를 -1로 바꾸지만 Store에는 여전히 8이 있습니다.', 'El callback pone count a -1. El Store sigue teniendo 8.', 'Le callback met count à -1. Le Store contient toujours 8.'],
				o: { cand: { v: -1, hot: true }, line: 1 },
			},
			{
				t: ['校验失败', 'Validation fails', '検証に失敗', '검증 실패', 'Falla la validación', 'La validation échoue'],
				d: ['校验失败，回调返回错误（panic 的效果完全相同）。', 'The callback returns an error — a panic behaves exactly the same.', 'コールバックがエラーを返します（panic でもまったく同じです）。', '콜백이 오류를 반환합니다. panic도 똑같이 동작합니다.', 'El callback devuelve un error; un panic se comporta exactamente igual.', 'Le callback renvoie une erreur ; un panic se comporte exactement pareil.'],
				o: { cand: { v: -1, err: true }, err: 'validate: count < 0', line: 2 },
			},
			{
				t: ['丢弃候选，什么都不发布', 'Discard — publish nothing', '破棄し、何も公開しない', '버리고 아무것도 공개하지 않음', 'Descartar sin publicar nada', 'Jeter sans rien publier'],
				d: [
					'候选被丢弃：状态仍是 8，版本仍是 v43，没有任何订阅者被打扰。错误交还给调用方。',
					'The candidate is discarded: state stays 8, version stays v43, no subscriber is disturbed. The error returns to the caller.',
					'候補値は破棄されます。状態は 8、バージョンは v43 のままで、どの購読者にも影響しません。エラーは呼び出し元に返ります。',
					'후보 값은 버려집니다. 상태는 8, 버전은 v43 그대로이고 어떤 구독자도 방해받지 않습니다. 오류는 호출자에게 돌아갑니다.',
					'El candidato se descarta: el estado sigue en 8, la versión en v43 y ningún suscriptor se entera. El error vuelve a quien llamó.',
					'Le candidat est jeté : l’état reste à 8, la version reste v43, aucun abonné n’est dérangé. L’erreur revient à l’appelant.',
				],
				o: { cand: null, lock: false, packets: [['error', 'STORE', 'WIN', 1]], subs: NONE4, line: 4 },
			},
		],
	},
	{
		tab: ['批量合并', 'Batch', 'バッチ', '배치', 'Lotes', 'Lots'],
		start: { ver: 43, count: 8, win: 8 },
		code: [
			'store.Batch(func() {',
			'    store.Update(func(s *State) { s.Count++ })',
			'    store.Update(func(s *State) { s.Title = "Hi" })',
			'    store.Update(func(s *State) { s.Count++ })',
			['}) // ← 只在这里发布一次', '}) // ← published once, only here', '}) // ← ここで 1 回だけ公開', '}) // ← 여기서 한 번만 공개', '}) // ← se publica una vez, solo aquí', '}) // ← publié une fois, ici seulement'],
		],
		steps: [
			{
				t: ['开启 Batch', 'Open a Batch', 'Batch を開始', 'Batch 시작', 'Abrir un Batch', 'Ouvrir un Batch'],
				d: [
					'Batch 内的写入照常提交，但通知会暂存到最外层结束时统一发出。',
					'Writes inside a Batch commit normally, but notifications are held until the outermost boundary.',
					'Batch 内の書き込みは通常どおりコミットされますが、通知は最も外側の境界まで保留されます。',
					'Batch 안의 쓰기는 평소처럼 커밋되지만, 알림은 가장 바깥 경계까지 보류됩니다.',
					'Dentro de un Batch las escrituras hacen commit con normalidad, pero los avisos esperan al límite más externo.',
					'Dans un Batch, les écritures sont commitées normalement, mais les notifications attendent la frontière la plus externe.',
				],
				o: { batch: 0, line: 0 },
			},
			{
				t: ['第 1 次写入：count = 9', 'Write 1: count = 9', '書き込み 1：count = 9', '쓰기 1: count = 9', 'Escritura 1: count = 9', 'Écriture 1 : count = 9'],
				d: ['版本 v44，通知暂存。', 'Version v44; the notification is held.', 'バージョン v44。通知は保留されます。', '버전 v44. 알림은 보류됩니다.', 'Versión v44; el aviso queda en espera.', 'Version v44 ; la notification est retenue.'],
				o: { batch: 1, ver: 44, count: 9, glow: true, packets: [['Update', 'WIN', 'STORE']], line: 1 },
			},
			{
				t: ['第 2 次写入：title = "Hi"', 'Write 2: title = "Hi"', '書き込み 2：title = "Hi"', '쓰기 2: title = "Hi"', 'Escritura 2: title = "Hi"', 'Écriture 2 : title = "Hi"'],
				d: ['版本 v45，订阅者仍然什么都没收到。', 'Version v45; subscribers still receive nothing.', 'バージョン v45。購読者にはまだ何も届きません。', '버전 v45. 구독자는 아직 아무것도 받지 못합니다.', 'Versión v45; los suscriptores aún no reciben nada.', 'Version v45 ; les abonnés ne reçoivent toujours rien.'],
				o: { batch: 2, ver: 45, title: 'Hi', glow: true, packets: [['Update', 'WIN', 'STORE']], line: 2 },
			},
			{
				t: ['第 3 次写入：count = 10', 'Write 3: count = 10', '書き込み 3：count = 10', '쓰기 3: count = 10', 'Escritura 3: count = 10', 'Écriture 3 : count = 10'],
				d: ['三次写入、三个版本、零次通知。', 'Three writes, three versions, zero notifications.', '3 回の書き込み、3 つのバージョン、通知は 0 回。', '쓰기 세 번, 버전 세 개, 알림 0회.', 'Tres escrituras, tres versiones, cero avisos.', 'Trois écritures, trois versions, zéro notification.'],
				o: { batch: 3, ver: 46, count: 10, glow: true, packets: [['Update', 'WIN', 'STORE']], line: 3 },
			},
			{
				t: ['关闭：只发布一次', 'Close: publish once', '終了：1 回だけ公開', '종료: 한 번만 공개', 'Cerrar: publicar una vez', 'Fermer : publier une fois'],
				d: [
					'在最外层边界一次性发布最终快照，每个订阅者只回调一次。',
					'The final snapshot is published once at the outermost boundary — one callback per subscriber.',
					'最も外側の境界で最終スナップショットを 1 回だけ公開します。コールバックは購読者ごとに 1 回です。',
					'가장 바깥 경계에서 최종 스냅숏을 한 번만 공개합니다. 콜백은 구독자마다 한 번입니다.',
					'La instantánea final se publica una sola vez en el límite más externo: un callback por suscriptor.',
					'L’instantané final est publié une seule fois à la frontière externe : un callback par abonné.',
				],
				o: { batch: null, packets: [['v46', 'STORE', 'S0'], ['v46', 'STORE', 'S1'], ['v46', 'STORE', 'S2'], ['v46', 'STORE', 'S3']], subs: [{ s: 'hit', n: '8 → 10' }, { s: 'hit', n: '"Nodra" → "Hi"' }, { s: 'rec', n: UNDO_STACK('v43') }, { s: 'queued' }], persist: 0.5, line: 4 },
			},
			{
				t: ['一次重绘', 'One redraw', '再描画は 1 回', '다시 그리기 한 번', 'Un solo redibujo', 'Un seul rendu'],
				d: ['窗口只重绘一次，直接从 8 跳到 10。', 'The window redraws once, jumping straight from 8 to 10.', 'ウィンドウは 1 回だけ再描画し、8 から 10 へ一気に変わります。', '창은 한 번만 다시 그려지며 8에서 바로 10이 됩니다.', 'La ventana se redibuja una vez y salta directamente de 8 a 10.', 'La fenêtre se redessine une fois et passe directement de 8 à 10.'],
				o: { win: 10, winFlash: true, packets: [['Invalidate()', 'S0', 'WIN']], subs: [{ s: 'hit', n: '8 → 10' }, { s: 'hit', n: '"Nodra" → "Hi"' }, { s: 'rec', n: UNDO_STACK('v43') }, { s: 'saved' }], persist: 1, line: 4 },
			},
		],
	},
	{
		tab: ['异步 · 最新优先', 'Async LatestWins', '非同期 LatestWins', '비동기 LatestWins', 'Async LatestWins', 'Async LatestWins'],
		start: { ver: 46, count: 10, win: 10, title: 'Hi' },
		code: [
			'search, _ := nodra.NewAsyncAction(store, "search", work, apply)',
			'search.Run(ctx, "no")     // #1',
			['search.Run(ctx, "nodra")  // #2 取代 #1', 'search.Run(ctx, "nodra")  // #2 supersedes #1', 'search.Run(ctx, "nodra")  // #2 が #1 を置き換える', 'search.Run(ctx, "nodra")  // #2가 #1을 대체', 'search.Run(ctx, "nodra")  // #2 reemplaza a #1', 'search.Run(ctx, "nodra")  // #2 remplace #1'],
			['// #2 先完成 → 加锁、校验代际、提交', '// #2 finishes first → lock, check generation, commit', '// #2 が先に完了 → ロック、世代を確認、コミット', '// #2가 먼저 완료 → 잠금, 세대 확인, 커밋', '// #2 termina antes → bloqueo, generación, commit', '// #2 finit d’abord → verrou, génération, commit'],
			['// #1 迟到 → 被拒绝，绝不覆盖新结果', '// #1 arrives late → rejected, never overwrites the newer result', '// #1 は遅れて到着 → 拒否、新しい結果を上書きしない', '// #1 늦게 도착 → 거부, 새 결과를 덮어쓰지 않음', '// #1 llega tarde → rechazado, nunca pisa el resultado nuevo', '// #1 arrive en retard → rejeté, n’écrase jamais le résultat récent'],
		],
		steps: [
			{
				t: ['启动 #1', 'Start run #1', '#1 を開始', '#1 시작', 'Lanzar #1', 'Lancer #1'],
				d: ['异步动作在所有锁之外执行耗时工作，例如一次网络搜索。', 'An AsyncAction runs work outside every lock — a network search, say.', 'AsyncAction は時間のかかる処理をすべてのロックの外で実行します。たとえばネットワーク検索です。', 'AsyncAction은 오래 걸리는 작업을 모든 잠금 밖에서 실행합니다. 예를 들면 네트워크 검색입니다.', 'Una AsyncAction hace el trabajo fuera de todo bloqueo, por ejemplo una búsqueda en red.', 'Une AsyncAction exécute le travail hors de tout verrou, par exemple une recherche réseau.'],
				o: { tasks: [{ l: '#1 "no"', s: 'run' }], packets: [['Run #1', 'WIN', 'T1']], line: 1 },
			},
			{
				t: ['启动 #2，取代 #1', '#2 supersedes #1', '#2 が #1 を置き換える', '#2가 #1을 대체', '#2 reemplaza a #1', '#2 remplace #1'],
				d: ['最新优先：新任务启动时，旧任务的上下文会被取消。', 'LatestWins: a newer run starts and the older run’s context is canceled.', 'LatestWins：新しい実行が始まると、古い実行のコンテキストはキャンセルされます。', 'LatestWins: 새 실행이 시작되면 이전 실행의 컨텍스트가 취소됩니다.', 'LatestWins: empieza una ejecución nueva y se cancela el contexto de la anterior.', 'LatestWins : une exécution plus récente démarre et le contexte de l’ancienne est annulé.'],
				o: { tasks: [{ l: '#1 "no"', s: 'cancel' }, { l: '#2 "nodra"', s: 'run' }], packets: [['Run #2', 'WIN', 'T2']], line: 2 },
			},
			{
				t: ['#2 完成 → 提交', '#2 finishes → commit', '#2 完了 → コミット', '#2 완료 → 커밋', '#2 termina → commit', '#2 finit → commit'],
				d: [
					'#2 获取写锁与代际锁，确认自己仍是最新，然后应用结果：v47。',
					'#2 takes the write and generation locks, confirms it is still the latest, and applies: v47.',
					'#2 は書き込みロックと世代ロックを取り、まだ最新であることを確かめてから適用します：v47。',
					'#2는 쓰기 잠금과 세대 잠금을 잡고 여전히 최신인지 확인한 뒤 적용합니다: v47.',
					'#2 toma los bloqueos de escritura y de generación, confirma que sigue siendo el último y aplica: v47.',
					'#2 prend les verrous d’écriture et de génération, vérifie qu’il est toujours le plus récent et applique : v47.',
				],
				o: { tasks: [{ l: '#1 "no"', s: 'cancel' }, { l: '#2 "nodra"', s: 'done' }], ver: 47, title: 'nodra ×12', glow: true, packets: [['apply', 'T2', 'STORE']], line: 3 },
			},
			{
				t: ['通知订阅者', 'Notify subscribers', '購読者に通知', '구독자에게 알림', 'Avisar a los suscriptores', 'Notifier les abonnés'],
				d: ['与普通 Update 一样在锁外通知；只有关注 title 的选择器被触发。', 'Notified outside the lock, like any Update — only the Title selector fires.', '通常の Update と同じくロックの外で通知します。発火するのは title のセレクターだけです。', '일반 Update처럼 잠금 밖에서 알립니다. title 선택자만 실행됩니다.', 'Se avisa fuera del bloqueo, como en cualquier Update; solo se dispara el selector de Title.', 'Notification hors du verrou, comme pour tout Update ; seul le sélecteur Title se déclenche.'],
				o: { packets: [['v47', 'STORE', 'S1'], ['v47', 'STORE', 'S2']], subs: [{ s: 'skip' }, { s: 'hit', n: '"Hi" → "nodra ×12"' }, { s: 'rec', n: UNDO_STACK('v46') }, { s: 'idle' }], line: 3 },
			},
			{
				t: ['迟到的 #1 被拒绝', 'The late #1 is rejected', '遅れた #1 は拒否', '늦은 #1은 거부', 'El #1 tardío se rechaza', 'Le #1 en retard est rejeté'],
				d: [
					'#1 终于返回，但它无法越过 #2 已接受的提交边界，旧结果永远不会覆盖新结果。',
					'#1 finally returns, but it cannot pass #2’s accepted commit boundary — stale results never overwrite newer ones.',
					'#1 がようやく戻りますが、#2 が受理されたコミット境界を越えられません。古い結果が新しい結果を上書きすることはありません。',
					'#1이 마침내 반환되지만 #2가 이미 통과한 커밋 경계를 넘을 수 없습니다. 오래된 결과는 새 결과를 덮어쓰지 않습니다.',
					'#1 vuelve por fin, pero no puede pasar el límite de commit ya aceptado de #2: un resultado obsoleto nunca pisa uno más nuevo.',
					'#1 revient enfin, mais ne peut franchir le commit déjà accepté de #2 : un résultat obsolète n’écrase jamais un plus récent.',
				],
				o: { tasks: [{ l: '#1 "no"', s: 'reject' }, { l: '#2 "nodra"', s: 'done' }], packets: [['stale #1', 'T1', 'STORE', 1]], err: '#1 rejected · stale generation', line: 4 },
			},
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

export function howView(c: HowCursor, li: LangIndex) {
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
			label: PACKET[p[0]]?.[li] ?? p[0],
			bg: p[3] ? '#ff6b5a' : '#c8f05a',
			tr: ph ? `left 1.1s cubic-bezier(.6,0,.2,1) ${k * 0.08}s, top 1.1s cubic-bezier(.6,0,.2,1) ${k * 0.08}s, opacity .2s` : 'none',
		};
	});

	const subs = s.subs.map((sb, k) => {
		const st = sb.s;
		const n = typeof sb.n === 'string' ? sb.n : sb.n?.[li];
		const note = n ? (st === 'hit' ? `${n} · ${SUBNOTE.hit[li]}` : n) : SUBNOTE[st][li];
		const lit = ph === 1 && s.packets.some((p) => p[2] === `S${k}`);
		const col = SUBCOLOR[st];
		const prevPersist = (sc.states[i - 1] ?? BASE).persist;
		return {
			name: SUBN[k][li],
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
			return { left: k ? '52%' : '31%', label: tk.l, state: st[0][li], color: st[1], deco: st[2] };
		}),
		subs,
		packets,
		err: s.err ? (ERR[s.err]?.[li] ?? s.err) : '',
		errOp: s.err ? 1 : 0,
	};

	return {
		v,
		cur: { num: String(i + 1).padStart(2, '0'), total: String(sc.steps.length).padStart(2, '0'), title: sc.steps[i].t[li], desc: sc.steps[i].d[li] },
		tabs: SCENARIOS.map((x, k) => ({ label: x.tab[li], active: k === c.scen })),
		steps: sc.steps.map((st, k) => ({ num: String(k + 1).padStart(2, '0'), title: st.t[li], desc: st.d[li], active: k === i, done: k < i })),
		codeLines: sc.code.map((line, k) => {
			const ln = typeof line === 'string' ? line : line[li];
			return { n: k + 1, text: ln, active: k === s.line, comment: ln.trim().startsWith('//') };
		}),
	};
}
