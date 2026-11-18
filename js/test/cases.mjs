// The calls the vectors hold, and how each result is written down.
import { env } from '../dist/gulal.js';

const envs = [
  {},
  { scheme: 'light' },
  { scheme: 'dark' },
  { background: '#ffffff', accent: '#0faaaa' },
  { background: '#1c1d20', accent: '#faa804', scheme: 'dark' },
  { background: '#608a82', accent: '#104746' },
  { background: '#00ffcc', accent: '#034948' },
  { background: '#f6efe3', accent: '#1d3557', avoid: ['#d93025'], spread: 30 },
  { scheme: 'light', accent: '#888888' },
  { contrast: 'wcag2', background: '#ffffff', accent: '#5b7fd6' },
  { hues: [[180, 300]], scheme: 'dark' },
  { inks: ['#fff4d6', '#14213d'], background: '#ffffff', accent: '#d82b1f' },
  { text: 75, scheme: 'light', accent: '#0faaaa' },
  { text: 75, scheme: 'dark', accent: '#faa804' },
  { text: 7, contrast: 'wcag2', background: '#f6efe3' },
];

export const cases = envs.map((e) => ({ env: { seed: 'vectors', ...e } }));

const texts = ['news.example.com', 'recipes.example.net', 'tab-group:tg_81f2', 'café', '日本', ''];
const backgrounds = ['#5b2a86', '#ffffff', '#000000', '#608a82', '#00ffcc', '#faa804', '#1c1d20', '#dcdee2'];

export function swatchOut(s) {
  return { ...s.toJSON(), turn75: s.turn(75) };
}

export function colourOut(c) {
  return { hex: c.hex, css: c.css, ink: c.ink, mark: c.mark, soft: swatchOut(c.soft), solid: swatchOut(c.solid), replay: c.replay ?? null };
}

function paletteOut(p) {
  return { replay: p.replay ?? null, colours: p.map(colourOut) };
}

/** Every call for one environment, in order (the stream depends on it). */
export function steps() {
  const s = [];
  for (const t of texts) s.push(['fromText', [t]]);
  s.push(['random', []], ['random', []], ['random', [5]], ['random', [4, { avoid: ['#3470d0'], spread: 30 }]], ['random', [3, { independent: true }]]);
  s.push(['fluent', [6]], ['fluent', [4, { arc: 90, from: 'project-x' }]], ['fluent', [30]], ['fluent', [3, { from: 'random' }]]);
  s.push(['distinct', [5]], ['distinct', [3, { from: 'random' }]], ['next', [['#3470d0', '#d03434']]], ['adapt', ['#3470d0']], ['adapt', ['#777777']]);
  for (const b of backgrounds) s.push(['swatch', [b]]);
  s.push(['swatch', ['#5b2a86', { around: '#5b2a86' }]], ['swatch', ['#f6efe3', { inks: ['#14213d', '#fff4d6'] }]]);
  for (const [a, b] of [['#ffffff', '#5b2a86'], ['#1f1f1f', '#ffffff'], ['#777777', '#888888']]) s.push(['contrast', [a, b]]);
  return s;
}

export function call(g, name, args) {
  const r = g[name](...args);
  if (name === 'contrast') return Number(r.toFixed(6));
  if (name === 'swatch') return swatchOut(r);
  if (Array.isArray(r)) return paletteOut(r);
  return colourOut(r);
}

export function run(c) {
  const g = env(c.env);
  return steps().map(([name, args]) => ({ call: name, args, out: call(g, name, args) }));
}
