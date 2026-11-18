// Writes ../vectors/vectors.json from this implementation: every call, in order, per
// environment, with everything it returns. The Go tests and test/vectors.test.mjs check it.
import { writeFileSync } from 'node:fs';
import { cases, run } from './cases.mjs';

const out = cases.map((c) => ({ env: c.env, calls: run(c) }));
writeFileSync(new URL('../../vectors/vectors.json', import.meta.url), JSON.stringify({ version: 1, cases: out }, null, 1) + '\n');
console.log(`wrote ${out.reduce((n, c) => n + c.calls.length, 0)} calls over ${out.length} environments`);
