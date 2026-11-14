// Checks that src/docs/translations/<lang>.json has a translation for every English string the docs page shows, and no key the page no longer uses. With --list <file> it writes those English strings to <file> as a JSON array instead, for translating.
import { readFileSync, writeFileSync } from 'node:fs';
import { createServer } from 'vite';

const LANGS = ['ja', 'ko', 'es', 'fr'];

// content.ts imports JSON and TypeScript modules, so Vite loads it; the site's own config is skipped because the Cloudflare plugin is not needed to read data.
const server = await createServer({ configFile: false, logLevel: 'error', server: { middlewareMode: true, hmr: false }, appType: 'custom', optimizeDeps: { noDiscovery: true } });
let texts;
try {
	const content = await server.ssrLoadModule('/src/docs/content.ts');
	texts = content.englishTexts();
} finally {
	await server.close();
}

const listAt = process.argv.indexOf('--list');
if (listAt !== -1) {
	writeFileSync(process.argv[listAt + 1], JSON.stringify(texts, null, '\t') + '\n');
	console.log(`wrote ${texts.length} English strings`);
	process.exit(0);
}

const wanted = new Set(texts);
let failed = false;
for (const lang of LANGS) {
	const file = `src/docs/translations/${lang}.json`;
	const table = JSON.parse(readFileSync(file, 'utf8'));
	const missing = texts.filter((t) => typeof table[t] !== 'string' || table[t].trim() === '');
	const unused = Object.keys(table).filter((k) => !wanted.has(k));
	if (missing.length || unused.length) failed = true;
	console.log(`${lang}: ${texts.length - missing.length}/${texts.length} translated, ${unused.length} unused`);
	for (const t of missing) console.log(`  missing: ${t}`);
	for (const k of unused) console.log(`  unused: ${k}`);
}
if (failed) {
	console.error('Update src/docs/translations: every English string in src/docs/content.ts needs a translation in each file, and keys the page no longer shows must be removed.');
	process.exit(1);
}
