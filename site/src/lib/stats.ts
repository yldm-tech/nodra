import { createServerFn } from '@tanstack/react-start';

// Public adoption numbers shown on the home page. Go modules have no public download counts, so these are the real signals that exist: GitHub stars, the version `go get` resolves, and the pkg.go.dev importer count. A source that fails is reported as null and simply not shown.
export interface AdoptionStats {
	stars: number | null;
	version: string | null;
	importedBy: number | null;
}

const MODULE = 'github.com/yldm-tech/nodra';

// Cloudflare caches successful upstream responses at the edge for an hour, which keeps unauthenticated GitHub API use far below its 60-requests-per-hour limit. Failures are not cached.
type EdgeRequestInit = RequestInit & { cf?: { cacheEverything?: boolean; cacheTtlByStatus?: Record<string, number> } };
const EDGE_CACHE: EdgeRequestInit['cf'] = { cacheEverything: true, cacheTtlByStatus: { '200-299': 3600, '300-599': 0 } };

async function fetchOk(url: string, headers: Record<string, string> = {}): Promise<Response> {
	const res = await fetch(url, { headers: { 'User-Agent': 'nodra-site', ...headers }, cf: EDGE_CACHE, signal: AbortSignal.timeout(3000) } as EdgeRequestInit);
	if (!res.ok) throw new Error(`${url}: HTTP ${res.status}`);
	return res;
}

async function githubStars(): Promise<number> {
	const repo = (await (await fetchOk(`https://api.github.com/repos/yldm-tech/nodra`, { Accept: 'application/vnd.github+json' })).json()) as { stargazers_count: number };
	return repo.stargazers_count;
}

async function latestVersion(): Promise<string> {
	const latest = (await (await fetchOk(`https://proxy.golang.org/${MODULE}/@latest`)).json()) as { Version: string };
	return latest.Version;
}

async function importedBy(): Promise<number> {
	const html = await (await fetchOk(`https://pkg.go.dev/${MODULE}`)).text();
	const m = html.match(/aria-label="Imported By: ([\d,]+)"/);
	if (!m) throw new Error('pkg.go.dev: importer count not found');
	return Number(m[1].replace(/,/g, ''));
}

const valueOrNull = <T,>(r: PromiseSettledResult<T>): T | null => (r.status === 'fulfilled' ? r.value : null);

export const getAdoptionStats = createServerFn({ method: 'GET' }).handler(async (): Promise<AdoptionStats> => {
	const [stars, version, imported] = await Promise.allSettled([githubStars(), latestVersion(), importedBy()]);
	for (const r of [stars, version, imported]) if (r.status === 'rejected') console.warn('adoption stats:', r.reason);
	return { stars: valueOrNull(stars), version: valueOrNull(version), importedBy: valueOrNull(imported) };
});
