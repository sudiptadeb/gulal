// Every call in vectors/vectors.json, made again, must give exactly what is written there.
import { readFileSync } from 'node:fs';
import { env } from '../dist/gulal.js';
import { call } from './cases.mjs';

const v = JSON.parse(readFileSync(new URL('../../vectors/vectors.json', import.meta.url), 'utf8'));
let n = 0, bad = 0;
for (const c of v.cases) {
  const g = env(c.env);
  for (const s of c.calls) {
    n++;
    const got = JSON.stringify(call(g, s.call, s.args));
    if (got !== JSON.stringify(s.out)) {
      bad++;
      if (bad <= 5) console.error(`differs: ${JSON.stringify(c.env)} ${s.call}(${JSON.stringify(s.args)})\n  want ${JSON.stringify(s.out).slice(0, 300)}\n  got  ${got.slice(0, 300)}`);
    }
  }
}
if (bad) {
  console.error(`vectors: ${bad} of ${n} calls differ`);
  process.exit(1);
}
console.log(`vectors: all ${n} calls match`);
