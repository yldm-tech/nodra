import type { ReactNode } from 'react';

// Token colours for the Go snippets. Group order matches the capture groups in GO_TOKENS: comment, string, keyword, exported identifier, number.
const COLORS = ['#6b6e66', '#f0c674', '#c8f05a', '#9ecbff', '#ff9a8a'];
const GO_TOKENS =
	/(\/\/[^\n]*|#[^\n]*)|("(?:[^"\\\n]|\\.)*"|`[^`]*`)|\b(package|import|func|type|struct|return|defer|if|else|for|range|var|const|go|chan|map|any|interface|nil|true|false|err|int)\b|\b([A-Z][A-Za-z0-9_]*)\b|\b(\d[\d_.]*)\b/g;

export function highlightGo(src: string): ReactNode {
	const out: ReactNode[] = [];
	const re = new RegExp(GO_TOKENS.source, 'g');
	let last = 0;
	let m: RegExpExecArray | null;
	while ((m = re.exec(src))) {
		if (m.index > last) out.push(src.slice(last, m.index));
		const group = [1, 2, 3, 4, 5].find((i) => m![i] !== undefined) ?? 1;
		out.push(
			<span key={m.index} style={{ color: COLORS[group - 1], fontStyle: group === 1 ? 'italic' : 'normal' }}>
				{m[0]}
			</span>,
		);
		last = re.lastIndex;
	}
	out.push(src.slice(last));
	return <code>{out}</code>;
}
