# gulal spec, version 1

This is the contract. The Go package and the TypeScript package implement it, and
`vectors/vectors.json` pins it: for the same inputs both give the same hex strings, bit for
bit. Where this page and the vectors disagree, the vectors win and this page is fixed.

Contents: 1 the model · 2 the environment · 3 making colours · 4 swatches · 5 algorithms ·
6 contrast · 7 the seed and replay · 8 the Go package · 9 versions.

## 1. The model

A colour is three numbers in OKLCH: lightness `L` (0–1), chroma `C` (strength) and hue `H`
(degrees). gulal decides them separately:

- **Hue** comes from text (hashed) or from the environment's seed (drawn), within the bands
  allowed, then leans toward the accent's hue.
- **Chroma** comes from the accent, or is drawn when there is none, and is never below the
  background's own.
- **Lightness** comes from where the colour sits: a step off the background, or drawn when
  there is neither background nor scheme. Inks then move only as far as they must to read.

A made colour is an identity (a hue and a chroma, and a lightness when drawn); it gives a
swatch for each look, soft and solid. A swatch is any background with what goes on it.

## 2. The environment

```ts
const gulal = env({ seed, scheme, background, accent, contrast, text, hues, avoid, spread, inks })
```

| Option | Default | Meaning |
|---|---|---|
| `seed` | 8 hex characters from a secure random source | all of the environment's randomness |
| `scheme` | none | `'light'` or `'dark'` |
| `background` | none | the surface the colours sit on |
| `accent` | none | the colour they live beside |
| `contrast` | `'apca'` | `'apca'` or `'wcag2'` |
| `text` | Lc 60, or 4.5 under `wcag2` | the contrast text reads at (§6); a number above 0 |
| `hues` | `[[0, 360]]` | bands `[from, to]` in degrees; a band with `from > to` wraps through 0 |
| `avoid` | none | colours (hex) or hues (numbers) to keep away from; a colour with chroma below 0.02 is ignored |
| `spread` | 25 | degrees kept clear either side of each `avoid` |
| `inks` | `#ffffff`, `#1f1f1f` | the inks a swatch picks its plain ink from |

Resolved values:

- **page:** the background; else the scheme's own, `#ffffff` for light and `#1c1d20` for dark;
  else none (swatches and inks then read on white).
- **pageL, pageC:** the page's OKLCH lightness and chroma (1 and 0 with no page).
- **pageDark:** with a page, `pageL < 0.45`, or `pageL < 0.8` in a dark scheme; with none,
  the scheme is dark.
- **chroma:** with an accent of chroma `aC`: 0.06 if `aC < 0.03`, else `clamp(0.8 × aC, 0.04,
  0.15)`. With no accent: drawn (§5.6).
- **accent hue:** the accent's hue, only when `aC ≥ 0.03`.
- **allowed hues:** the bands less `avoid ± spread` (§5.4). If nothing is left, the bands.

## 3. Making colours

| Method | Returns | Random |
|---|---|---|
| `random()` | a Colour | yes |
| `random(n, { avoid, spread, independent })` | a Palette of `n` | yes |
| `fromText(text)` | a Colour | no |
| `fluent(n, { arc, from })` | a Palette of `n` | when it starts at random |
| `distinct(n, { from, avoid, spread })` | a Palette of `n` | when `from` is `'random'` |
| `next(existing)` | a Colour | no |
| `adapt(hex)` | a Colour | no |
| `swatch(background, { around, inks })` | a Swatch | no |
| `contrast(ink, background)` | a number | no |
| `replay(string)` | a Colour or a Palette | — |

A **Palette** is an array of Colours with a `replay` string when it was drawn at random.

### 3.1 random()

`H` = the allowed hues at a draw (§5.4); `C` from §5.6, drawing if there is no accent; `L`
drawn as `0.45 + u × 0.3` if there is no page. Then the lean (§5.5).

### 3.2 random(n)

`avoid` and `spread` here add to the environment's (`spread` defaults to the environment's).
With `independent`, `n` × random() over those allowed hues. Otherwise: the first hue is
the allowed hues at a draw; each next one is the farthest (§5.7) from those taken and those
this call avoids. `C` and `L` are drawn per colour as in random(), the lean applied to each.

### 3.3 fromText(text)

As random(), with draws from the text instead of the seed: `u1 = hash(text)`,
`u2 = hash(text, 0x01)`, `u3 = hash(text, 0x02)` (§5.1) for hue, chroma and lightness.

### 3.4 fluent(n)

- start `s`: `from` is `'accent'` (the accent hue, or 0), `'random'` (a draw × 360), or any
  other text (`unit(hash(text)) × 360`); `'accent'`, `'random'` and `'zero'` are words, not
  text to hash; with no `from`, the accent hue if there is one,
  else a draw.
- `arc = options.arc ?? min(360, 40n)`; `step = 360/n` when `arc ≥ 360`, else
  `arc/(n − 1)` (0 for one colour). If `n > 1` and `step < 18`: `arc = min(360, 18n)` and
  the step again; if still under 18, colours alternate lightness `+0.06, −0.06, …`.
- hue `i` is `s + i × step`; no lean; `C` from §5.6 with the middle draw 0.5; `L` 0.6 when
  there is no page.

### 3.5 distinct(n)

The start as fluent, but with no `from` and no accent it is 0. If the start is not allowed,
the first is the farthest allowed hue from the start's opposite (or from the avoided ones).
Then each next one is the farthest from those taken and avoided. No lean; `C` as fluent;
`L` 0.6 when there is no page.

### 3.6 next(existing)

The farthest allowed hue from the hues of `existing` and the environment's avoided ones (or
the first allowed hue if `existing` is empty). No lean; `C` as fluent; `L` 0.6 with no page.

### 3.7 adapt(hex)

Its hue, leaned. `C` is the environment's, or the colour's own clamped to 0.04–0.15 when
there is no accent, then the page floor (§5.6). `L` 0.6 when there is no page.

### 3.8 A Colour

| Field | What it is |
|---|---|
| `hex` | the solid fill (§5.9) |
| `soft` | the Swatch on the soft tint (§5.10), around the page |
| `solid` | the Swatch on `hex`, around the page |
| `ink` | ink of its hue on the page (white with none): reach from 0.5 (light page) or 0.8 (dark), chroma `min(max(1.3C, 0.06), 0.15)`, at the ink level |
| `mark` | the same at the mark level, from 0.62 (light) or 0.66 (dark) |
| `oklch`, `css` | the OKLCH of `hex`; `css` is `oklch(L C H)` to 3, 3 and 1 decimals |
| `replay` | for random ones |

## 4. Swatches

`swatch(background, { around, inks })`: `around` defaults to the page; `inks` to the
environment's (white and near-black if none).

- `light`: near-black reads at least as well as white on the background.
- `base hue`: the background's hue, or, when its chroma is under 0.02, the accent hue (or 0).
- **border:** hue `H`, chroma `min(1.1C, 0.37)`. It goes down in lightness when `around` is
  more than 0.02 lighter than the background, up when more than 0.02 darker, and otherwise
  down if `light`. Start 0.12 from the background's lightness, then 0.005 at a time (at most
  200 steps, within 0–1) until it shows against `around` at the border level, measured both
  ways round, the larger. With no `around`, the start.
- **ink:** of the inks, the one with the most contrast on the background; ties go to the
  earlier. Used even if it falls short.
- **muted:** if the ink reads, the ink's lightness moved toward the background's,
  `L = Li + (Lb − Li) × t`, at the ink's own chroma and hue, with the largest `t` in [0, 1]
  that still reads (bisection, 20 rounds, keeping the side that reads); else the ink.
- **tonal:** reach from `start` (0.5 if `light`, else 0.8), chroma `C` if `C < 0.02` else
  `min(max(1.3C, 0.06), 0.15)`, hue `H`, at the ink level.
- **complementary:** `turn(180)`.
- **accent:** with an accent hue, reach from `start` at the accent's chroma clamped to
  0.06–0.15, at the accent hue.
- **turn(d):** reach from `start`, chroma 0.15, hue `base hue + d`.
- **harmony** (made when first read): `analogous` turns 50, −50; `complementary` 180;
  `split` 150, −150; `triadic` 120, −120; `square` 90, 180, −90.
- **vars(prefix = 'gulal'):** `--prefix-background`, `-border`, `-ink`, `-muted`, `-tonal`,
  `-complementary`, and `-accent` when there is one.

Ties in reach (§5.8) go up in a dark scheme, down in a light one, and with no scheme up when
the background is not `light`.

## 5. Algorithms

### 5.1 Hash and unit draws

FNV-1a, 32-bit, over the UTF-8 bytes of the text (and one extra byte when given), then
murmur3's finaliser:

```
h = 0x811c9dc5; for each byte b: h = (h xor b) × 0x01000193  (mod 2^32)
h ^= h >> 16; h ×= 0x85ebca6b; h ^= h >> 13; h ×= 0xc2b2ae35; h ^= h >> 16
```

`unit(h) = h / 2^32`, in [0, 1).

### 5.2 Colour conversion

8-bit sRGB to linear: `v/255`, then `v/12.92` if `≤ 0.04045`, else `((v + 0.055)/1.055)^2.4`.
To OKLab with Björn Ottosson's matrices, to the digits in the source; `L, a, b` to `C =
sqrt(a² + b²)`, `H = atan2(b, a)` in degrees in [0, 360); `H = 0` when `C < 1e-4`.

OKLCH to 8-bit: OKLab, cubes as `x·x·x`, linear sRGB by the inverse matrix. If any channel is
outside `[−1e-4, 1 + 1e-4]`, chroma is lowered by bisection over `[0, C]`, 16 rounds, keeping
the highest in range; lightness and hue are kept. Each channel is clamped to [0, 1], encoded
(`12.92v` if `≤ 0.0031308`, else `1.055 v^(1/2.4) − 0.055`) and rounded as
`floor(255 × v + 0.5)`. `L` is clamped to [0, 1] and `C` to ≥ 0 first.

Implementations must not fuse a multiply and an add (Go on arm64 does unless each product is
converted explicitly); the vectors catch it.

### 5.3 OKLab distance

The Euclidean distance of two 8-bit colours' `L, a, b`, with `a, b` from their `C, H`.

### 5.4 Allowed hues

Bands as intervals in [0, 360). Each avoided hue removes `[h − spread, h + spread]`, also
shifted by ±360. Laid end to end, a draw `u` maps to the point `u × total` along them.

### 5.5 Lean

With an accent hue `a`: `d = wrap(a − H)`; `diff = d > 180 ? 360 − d : d`;
`turn = min(diff/2, 15)`; `H += d > 180 ? −turn : turn`. As Material's `Blend.harmonize`.

### 5.6 Chroma

`C` = the accent's (§2); with no accent, `0.06 + u × 0.09` for a draw `u` (0.5 where the
method says the middle). With a page, `C = max(C, 0.9 × pageC)`. Then `min(C, 0.16)`.

### 5.7 Farthest

Over the allowed hues on a 0.5° grid (`k × 0.5`, k = 0…719, inside an interval `[a, b)`), the
one whose least circular distance to the given hues is largest; ties go to the smaller hue.

### 5.8 Reach

A colour of hue `H` and chroma `C`, its lightness moved from `L0` as little as it must to
read on `against` at `level`:

- `bgL` = the lightness of `against`.
- **up:** if it does not read at 1, none. If it reads at `max(L0, bgL)`, that. Else bisection
  between it and 1, 20 rounds, keeping the end that reads.
- **down:** the same toward 0 from `min(L0, bgL)`.
- Neither: 1 if it scores at least as well at 1 as at 0, else 0. One: that one. Both: the
  smaller move from `L0`; within 1e-9, the tie rule of §4.

### 5.9 The solid fill (`hex`)

`L0`: drawn or fixed `L` when the method gave one (0.6 with no page); else 0.62 on a dark
page or 0.6 on a light one, and if that is within 0.15 of `pageL`, `pageL − 0.15` (when
`pageL ≥ 0.5`) or `pageL + 0.15`. Plus a fluent alternation. The fill is that colour; its ink
is the best of the environment's inks (white and near-black by default). If the ink does not
read, the fill moves away from it (down if the ink is lighter than `L0`, else up): if it reads
at the far end, bisection between `L0` and that end, 20 rounds, keeping the end that reads;
else the far end.

### 5.10 The soft tint

`dark` = pageDark. Chroma `max(C × (dark ? 0.55 : 0.45), 0.85 × pageC)`. Lightness: `pageL <
0.45` → `max(0.3, pageL + 0.09)`; `pageL < 0.8` → 0.3 if dark, else 0.93; else `min(0.93,
pageL − 0.05)`. Then 0.005 at a time away from the page's lightness (at most 200 steps, while
within 0–1) until its OKLab distance to the page is at least `0.05 + 0.65 × pageC`.

### 5.11 The seeded stream

SplitMix32 over the hash of the seed: the `k`-th draw (k = 1, 2, …) is
`unit(fmix(hash(seed) + k × 0x9e3779b9 mod 2^32))`, where `fmix` is murmur3's finaliser as in
§5.1. An environment counts its draws; each random call takes the next ones.

## 6. Contrast

| Use | APCA (default) | WCAG 2 |
|---|---|---|
| ink: a swatch's inks, `turn()`, `colour.ink`, a fill's ink | Lc 60 | 4.5 |

The ink level is the environment's `text` when given (Lc 75 suits letters under 24px).
| border against `around` | Lc 15 | 1.3 |
| `colour.mark` | Lc 30 | 3 |

APCA is 0.0.98G as a magnitude: `Y = 0.2126729 r^2.4 + 0.7151522 g^2.4 + 0.0721750 b^2.4`
on 0–1 sRGB; below 0.022, `Y += (0.022 − Y)^1.414`. Dark ink on light: `S = (Yb^0.56 −
Yt^0.57) × 1.14`, 0 if under 0.1, else `(S − 0.027) × 100`. Light on dark: `S = (Yb^0.65 −
Yt^0.62) × 1.14`, 0 if over −0.1, else `−(S + 0.027) × 100`. WCAG 2 is the usual ratio.

## 7. The seed and replay

The seed is any text. Each random result carries a replay string:

```
gulal1;<method>;<call options>;<environment>;seed=<seed>;at=<draws before the call>
```

Call options: `n`, `avoid`, `spread`, `independent=1`, `arc`, `from`. Environment:
`scheme`, `bg`, `accent`, `contrast`, `text`, `hues` (`a..b,a..b`), `eavoid`, `espread`, `inks`.
Avoid lists are comma-separated, colours as `#rrggbb` and hues as plain numbers; an entry
that starts with `#` is a colour. Absent values are left out. In values, only `%`, `;` and `=` are escaped, as `%`
and two upper-case hex digits. `replay(s)` rebuilds the environment, sets its count to `at`
and makes the call again. A string that does not start with `gulal1` is refused.

## 8. The Go package

The same methods, in Go's style: `New(Options) *Env` and `Env.RandomN`, `FromText`, `Fluent`,
`Distinct`, `Next`, `Adapt`, `Swatch`, `Contrast`; `Replay(s) (Palette, error)`, where a
single colour replays as a palette of one. Go has no union types, so an avoid entry is a
colour when it starts with `#`, else a hue; zero values mean the defaults. Bad input gives
nil (`New` when `Options.Validate()` fails, `Adapt`, `Next`, `Swatch`), NaN from `Contrast`,
or an empty palette, never a panic.

Go's maths functions and JavaScript's can differ in the last bit. The rules above round to
8-bit and compare against thresholds, which absorbs it for every vector; an input that lands
exactly on a rounding edge could still come out one step apart in the other language.

## 9. Versions

The `1` in `gulal1` is this spec's major version. A change that moves any output in the
vectors is a new major version, as stored or remembered colours would shift. New methods or
options that leave every output alone are minor versions.
