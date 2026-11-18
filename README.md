# gulal

**Colours made for where they sit.**

gulal makes up the colours an interface needs and has no designer for: a site's letter tile,
a tag, a tab group, a chart's series, an avatar. You tell it where they will sit, a light or
dark scheme, a background and an accent, and every colour it makes fits that place: as
strong as the theme, deep or pale enough for its background, and with inks that read on it.

- **Random, seeded or from text.** One seed drives all the randomness, so tests and
  screenshots repeat. Any text gives the same colour every time, in Go and in TypeScript.
- **Palettes that work.** Random ones kept apart instead of clumped, fluent ones that sit well
  side by side, distinct ones as different as the place allows.
- **Swatches for any background,** yours or gulal's: a border, an ink, a muted ink, tonal and
  complementary ink, the accent, and every colour-wheel relation, each held to APCA Lc 60.
- **Two reference implementations,** Go and TypeScript, giving the same hex strings, checked
  by one shared file of test vectors. No dependencies.

Try it: **[debkosh.com/gulal](https://debkosh.com/gulal)**.

Gulal is the coloured powder of Holi.

## A look

```ts
import { env } from '@debkosh/gulal';

const gulal = env({ scheme: 'light', background: '#ffffff', accent: '#0faaaa' });

gulal.fromText('news.example.com').hex;  // '#aa714a', every time, everywhere
gulal.fromText('news.example.com').soft; // a pale tile with its border and inks

const card = gulal.swatch('#5b2a86');    // a background you already have
card.ink;                                // '#ffffff'  plain ink, the best of white and near-black
card.muted;                              // '#c5c5c5'  for a second line
card.tonal;                              // '#d8b7ff'  its own hue
card.complementary;                      // '#abd25e'  the opposite hue
card.harmony.triadic;                    // ['#ffb56a', '#00dbca']
card.vars('card');                       // { '--card-background': '#5b2a86', '--card-border': … }

gulal.random(5);                         // five colours kept apart, with a replay string
gulal.fluent(6);                         // six that sit well in a row
gulal.distinct(8);                       // eight as different as this place allows
```

```go
import "github.com/sudiptadeb/gulal"

g := gulal.New(gulal.Options{Scheme: "light", Background: "#ffffff", Accent: "#0faaaa"})

g.FromText("news.example.com").Hex()   // "#aa714a", the same as in TypeScript
card := g.Swatch("#5b2a86", gulal.SwatchOptions{})
card.Ink, card.Muted, card.Tonal, card.Complementary
card.Harmony().Triadic
```

## Install

```sh
npm install @debkosh/gulal
go get github.com/sudiptadeb/gulal
```

## The environment

Everything is a method on an environment, made once for a place in your interface.

```ts
const gulal = env({
  seed: '9f2c41d0',          // one seed for all its randomness; picked and shown when left out
  scheme: 'light',           // 'light' or 'dark': the intent; stands in for a background
  background: '#ffffff',     // the surface the colours sit on
  accent: '#0faaaa',         // the colour they live beside: sets their strength, leans their hues
  contrast: 'apca',          // or 'wcag2'
  hues: [[0, 360]],          // bands hues may fall in
  avoid: ['#d93025'],        // colours or hues to keep away from, such as an error red
  spread: 25,                // degrees to keep clear either side of each
  inks: ['#ffffff', '#1f1f1f'], // the inks a swatch picks its plain ink from
});
```

All of it is optional. What you leave out widens the range: with no accent, strength is
drawn too; with no background or scheme, lightness is drawn too. `env()` alone makes
colours from the whole range.

How it decides:

- **Hue** comes from the text (hashed) or the seed (drawn), then leans toward the accent's
  hue, half the gap and at most 15°, as Material's `harmonize` does, so a set looks of one
  family.
- **Strength** (OKLCH chroma) is the accent's, 80% of it, never weaker than the background's
  own: a muted theme gets muted colours, a vivid one vivid. Saturation is what makes colours
  fit a theme.
- **Lightness** is only for reading: a step off the background, and inks moved as little as
  they must to pass.

All of it is worked in OKLCH, where equal steps of hue look equally different. In HSL a third
of all hashed colours come out green or purple.

## Making colours

| Call | What you get |
|---|---|
| `random()` | one colour, the next draw from the seed |
| `random(n, { avoid, spread, independent })` | `n` kept apart: a random start, then each the farthest from those taken |
| `fromText(text)` | the same colour for the same text, in every environment with the same inputs, whatever its seed |
| `fluent(n, { arc, from })` | one strength and lightness, hues in even steps through an arc, for lists and legends |
| `distinct(n, { from, avoid })` | the `n` most different, the same every time |
| `next(existing)` | the one colour farthest from those in use |
| `adapt(hex)` | a colour you have, its hue kept, made at home here |

A palette is an array of colours. A colour has:

```ts
c.hex      // its solid fill
c.soft     // a swatch on its soft tint, a step off the background
c.solid    // a swatch on its solid fill
c.ink      // its colour as ink on the environment's background
c.mark     // its colour as a dot or a line there
c.oklch    // { l, c, h }
c.css      // 'oklch(0.600 0.090 54.5)'
c.replay   // for random ones: the call that makes it again
```

To colour one id two ways, put the use in the text: `fromText('tab-group:' + id)`. Seed from
what never changes, an id rather than a name, or a rename changes the colour.

## Swatches

`swatch(background, { around, inks })` takes any background and gives what goes on it:

| Key | What it is |
|---|---|
| `background` | the background |
| `border` | its hue, a step away from what is around it, until it shows |
| `ink` | the best of your `inks`, or of white and near-black |
| `muted` | the ink moved toward the background until it only just reads |
| `tonal` | ink in the background's own hue |
| `complementary` | ink in the opposite hue |
| `accent` | ink in the environment's accent, when it has one |
| `harmony` | ink in every relation: `analogous` (±50°), `complementary`, `split` (±150°), `triadic` (±120°), `square` (90° steps). Made when first read |
| `turn(degrees)` | ink in any other hue |
| `vars(prefix)` | all of it as CSS custom properties |

`gulal.contrast(ink, background)` measures any pair the environment's way.

## Seeds and replay

One stream per environment, seeded by its `seed`; each random call takes the next draws, so
the same seed and the same calls give the same colours. Every random result carries a replay
string, such as `gulal1;random;n=5;scheme=light;accent=#0faaaa;seed=9f2c41d0;at=12`, and
`gulal.replay(s)` makes those colours again without the calls before it: keep the palette
someone liked as one short string.

## The contract

[spec.md](spec.md) is normative: the hash, the seeded stream, every rule and constant, and
how colours are rounded. [vectors/vectors.json](vectors/vectors.json) holds hundreds of calls
over a dozen environments with everything they return; the Go and TypeScript tests replay
every one. A change that moves any output is a major version.

## Develop

```sh
go test ./...                        # Go, against the vectors
cd js && npm install && npm test     # TypeScript, against the vectors
cd js && npm run vectors             # rewrite the vectors from the TypeScript
python3 -m http.server               # then open /playground/ (after npm run build)
```

To release, set the version in `js/package.json`, tag `vX.Y.Z` and push it (the Go module is
the tag), and `.github/workflows/npm.yml` publishes the npm package from that tag.

## Licence

MIT.
