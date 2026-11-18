// What the API promises, beyond the vectors.
import assert from 'node:assert/strict';
import { env, replay } from '../dist/gulal.js';

const gulal = env({ seed: 'api', scheme: 'light', accent: '#0faaaa' });

// the same text, the same colour, whatever the seed
assert.equal(gulal.fromText('news.example.com').hex, env({ seed: 'other', scheme: 'light', accent: '#0faaaa' }).fromText('news.example.com').hex);

// the same seed and calls, the same colours; a replay gives them again without the calls before
const a = env({ seed: 's1' }), b = env({ seed: 's1' });
a.random(); b.random();
const pa = a.random(5), pb = b.random(5);
assert.deepEqual(pa.map(String), pb.map(String));
assert.deepEqual(replay(pa.replay).map(String), pa.map(String));

// no seed: one is picked and shown
const e = env();
assert.match(e.seed, /^[0-9a-f]{8}$/);

// a picked colour reads on its swatch
const s = gulal.swatch('#5b2a86');
for (const k of ['ink', 'muted', 'tonal', 'complementary', 'accent']) assert.ok(gulal.contrast(s[k], s.background) >= 60, k);
for (const [, list] of Object.entries(s.harmony)) for (const c of list) assert.ok(gulal.contrast(c, s.background) >= 60);

// the best of the given inks
assert.equal(gulal.swatch('#ffffff', { inks: ['#eeeeee', '#14213d', '#333333'] }).ink, '#14213d');

// palettes are arrays
assert.equal(gulal.fluent(4).length, 4);
assert.ok(Array.isArray(gulal.distinct(3)));

// a palette kept apart: no two hues closer than independent draws would allow on average
const hues = env({ seed: 'apart' }).random(6).map((c) => c.oklch.h);
for (let i = 0; i < hues.length; i++) for (let j = i + 1; j < hues.length; j++) {
  const d = Math.abs(hues[i] - hues[j]);
  assert.ok(Math.min(d, 360 - d) > 25, 'kept apart');
}

console.log('api: all checks pass');
