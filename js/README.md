# gulal

Colours made for where they sit: random, seeded or from text, palettes that sit well
together, and for any background the border and inks that read on it. The same output as the
Go package, checked by shared test vectors.

```ts
import { env } from 'gulal';

const gulal = env({ scheme: 'light', background: '#ffffff', accent: '#0faaaa' });
gulal.fromText('news.example.com').soft; // { background, border, ink, muted, tonal, … }
gulal.swatch('#5b2a86').ink;             // '#ffffff'
gulal.random(5);                         // five colours kept apart
```

The full README, spec and playground: https://github.com/sudiptadeb/gulal ·
https://debkosh.com/gulal
