// Package gulal makes colours made for where they sit: random, seeded or from text, in
// palettes that sit well together, and for any background the border and inks that read on
// it.
//
// It is a port of the TypeScript reference in js/src/gulal.ts and gives the same output,
// checked by vectors/vectors.json.
//
// Bad input: the methods keep the signatures of the TypeScript API and do not panic. Where
// the TypeScript throws for a string that is not a colour, gulal returns a zero value
// instead: New returns nil for Options that do not validate (Options.Validate says why),
// Adapt, Next and Swatch return nil, Contrast returns NaN, and RandomN and Distinct return an
// empty Palette when their Avoid holds a bad colour. Replay returns an error.
package gulal

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"math"
	"net/url"
	"strconv"
	"strings"
	"sync"
)

// Options is an environment: what the colours sit on and beside.
type Options struct {
	// Seed drives all of the environment's randomness. Picked (8 hex digits) when empty.
	Seed string
	// Scheme is "light", "dark" or "": the app's overall scheme.
	Scheme string
	// Background is the surface the colours sit on, as hex, or "".
	Background string
	// Accent is the primary colour they live beside, as hex, or "".
	Accent string
	// Contrast is "apca" (the default, also for "") or "wcag2".
	Contrast string
	// Text is the contrast text reads at: 0 means Lc 60 for APCA, 4.5 for WCAG 2.
	Text float64
	// Hues are the bands, in degrees, hues may fall in. nil means all of them.
	Hues [][2]float64
	// Avoid holds colours (hex, "#" optional) or hues (plain numbers such as "350") to keep
	// away from. An entry that does not start with "#" and reads as a number is a hue.
	Avoid []string
	// Spread is the degrees of hue kept clear either side of each Avoid. 0 means 25.
	Spread float64
	// Inks are the inks a swatch picks its plain ink from: white and near-black when empty.
	Inks []string
}

// RandomOptions are the options of Env.RandomN.
type RandomOptions struct {
	Avoid       []string
	Spread      float64 // 0: the environment's
	Independent bool
}

// FluentOptions are the options of Env.Fluent.
type FluentOptions struct {
	Arc  float64 // 0: min(360, 40n)
	From string  // "accent", "random", or any text to hash for the first hue; "": the default
}

// DistinctOptions are the options of Env.Distinct.
type DistinctOptions struct {
	From   string
	Avoid  []string
	Spread float64
}

// SwatchOptions are the options of Env.Swatch.
type SwatchOptions struct {
	Around string   // what the background sits on: the border shows against it
	Inks   []string // the inks to choose the plain ink from
}

// Palette is a list of colours, with the replay string that makes it again when it was drawn.
type Palette struct {
	Colours []*Colour
	Replay  string
}

type resolved struct {
	scheme   string
	page     *rgb8 // the background, or the scheme's own
	pageL    float64
	pageC    float64
	pageDark bool
	accent   *rgb8
	accentH  float64
	hasHue   bool // accentH is set
	chroma   float64
	hasC     bool // chroma is set
	measure  string
	text     float64 // the level text reads at
	bands    []interval
	avoid    []float64
	spread   float64
	inks     []rgb8
}

var schemeBG = map[string]rgb8{"light": {255, 255, 255}, "dark": {28, 29, 32}}

// Validate reports the first thing in o that New cannot use.
func (o Options) Validate() error {
	_, err := resolve(o)
	return err
}

func resolve(o Options) (*resolved, error) {
	switch o.Scheme {
	case "", "light", "dark":
	default:
		return nil, fmt.Errorf("gulal: not a scheme: %s", o.Scheme)
	}
	switch o.Contrast {
	case "", "apca", "wcag2":
	default:
		return nil, fmt.Errorf("gulal: not a contrast measure: %s", o.Contrast)
	}
	r := &resolved{scheme: o.Scheme, measure: o.Contrast, spread: o.Spread}
	if r.measure == "" {
		r.measure = "apca"
	}
	if r.spread == 0 {
		r.spread = 25
	}
	if o.Text < 0 {
		return nil, fmt.Errorf("gulal: not a contrast level: %v", o.Text)
	}
	r.text = o.Text
	if r.text == 0 {
		r.text = levelsOf(r.measure).ink
	}
	if o.Background != "" {
		c, err := parse(o.Background)
		if err != nil {
			return nil, err
		}
		r.page = &c
	} else if o.Scheme != "" {
		c := schemeBG[o.Scheme]
		r.page = &c
	}
	if r.page != nil {
		r.pageL, r.pageC, _ = lch(*r.page)
		r.pageDark = r.pageL < 0.45 || (r.pageL < 0.8 && o.Scheme == "dark")
	} else {
		r.pageL, r.pageC = 1, 0
		r.pageDark = o.Scheme == "dark"
	}
	if o.Accent != "" {
		c, err := parse(o.Accent)
		if err != nil {
			return nil, err
		}
		r.accent = &c
		_, aC, aH := lch(c)
		r.hasC = true
		if aC < 0.03 {
			r.chroma = 0.06
		} else {
			r.chroma = math.Min(0.15, math.Max(0.04, 0.8*aC))
			r.accentH, r.hasHue = aH, true
		}
	}
	var err error
	if r.avoid, err = huesOf(o.Avoid); err != nil {
		return nil, err
	}
	r.bands = bandsOf(o.Hues)
	for _, s := range o.Inks {
		c, err := parse(s)
		if err != nil {
			return nil, err
		}
		r.inks = append(r.inks, c)
	}
	return r, nil
}

// Env is an environment; everything else is a method on it. It is safe for concurrent use:
// the seeded stream is guarded, and a call that draws holds it for the whole call.
type Env struct {
	seed string
	o    Options
	r    *resolved
	mu   sync.Mutex
	at   uint32
}

func pickSeed() string {
	b := make([]byte, 4)
	if _, err := rand.Read(b); err != nil {
		panic("gulal: no random source: " + err.Error())
	}
	return hex.EncodeToString(b)
}

// New makes an environment, or returns nil when o does not validate (see Options.Validate).
func New(o Options) *Env {
	r, err := resolve(o)
	if err != nil {
		return nil
	}
	if o.Seed == "" {
		o.Seed = pickSeed()
	}
	return &Env{seed: o.Seed, o: o, r: r}
}

// Seed is the environment's seed, given or picked.
func (e *Env) Seed() string { return e.seed }

// ---- the stream (callers hold e.mu)

func (e *Env) draw() float64 {
	e.at++
	return unit(splitmix(hash(e.seed, -1) + e.at*golden))
}

// jsNum writes a number as JavaScript's String(n) does, for the values gulal writes.
func jsNum(f float64) string {
	if f == 0 {
		return "0"
	}
	if a := math.Abs(f); a >= 1e21 || a < 1e-6 {
		s := strconv.FormatFloat(f, 'e', -1, 64)
		mant, exp, _ := strings.Cut(s, "e")
		sign := exp[0]
		exp = strings.TrimLeft(exp[1:], "0")
		return mant + "e" + string(sign) + exp
	}
	return strconv.FormatFloat(f, 'f', -1, 64)
}

// escapeReplay escapes only what would break a replay string: %, ; and =.
func escapeReplay(s string) string {
	var b strings.Builder
	for i := 0; i < len(s); i++ {
		switch c := s[i]; c {
		case '%', ';', '=':
			fmt.Fprintf(&b, "%%%02X", c)
		default:
			b.WriteByte(c)
		}
	}
	return b.String()
}

type kv struct{ k, v string }

// avoidReplay is an avoid list as a replay writes it: colours as #rrggbb, hues as plain
// numbers. Its entries have been checked already (resolve, huesOf).
func avoidReplay(list []string) string {
	out := make([]string, len(list))
	for i, x := range list {
		if f, ok := isHue(x); ok {
			out[i] = jsNum(f)
		} else {
			c, _ := parse(x)
			out[i] = hexOf(c)
		}
	}
	return strings.Join(out, ",")
}

func (e *Env) replayOf(method string, args []kv, at uint32) string {
	parts := []string{"gulal1", method}
	put := func(k, v string) {
		if v != "" {
			parts = append(parts, k+"="+escapeReplay(v))
		}
	}
	for _, a := range args {
		put(a.k, a.v)
	}
	o := e.o
	put("scheme", o.Scheme)
	put("bg", o.Background)
	put("accent", o.Accent)
	put("contrast", o.Contrast)
	if o.Text != 0 {
		put("text", jsNum(o.Text))
	}
	if o.Hues != nil {
		hs := make([]string, len(o.Hues))
		for i, p := range o.Hues {
			hs[i] = jsNum(p[0]) + ".." + jsNum(p[1])
		}
		put("hues", strings.Join(hs, ","))
	}
	put("eavoid", avoidReplay(o.Avoid))
	if o.Spread != 0 {
		put("espread", jsNum(o.Spread))
	}
	put("inks", strings.Join(o.Inks, ","))
	put("seed", e.seed)
	put("at", strconv.FormatUint(uint64(at), 10))
	return strings.Join(parts, ";")
}

func numArg(k string, f float64) kv {
	if f == 0 {
		return kv{k, ""}
	}
	return kv{k, jsNum(f)}
}

// ---- making colours

// chromaFor is the strength of a colour: the accent's, or drawn; never below the background's.
func (e *Env) chromaFor(u float64) float64 {
	C := e.r.chroma
	if !e.r.hasC {
		C = float64(u*0.09) + 0.06
	}
	if e.r.page != nil {
		C = math.Max(C, 0.9*e.r.pageC)
	}
	return math.Min(0.16, C)
}

func (e *Env) lean(H float64) float64 {
	if !e.r.hasHue {
		return H
	}
	d := wrap(e.r.accentH - H)
	diff := d
	if d > 180 {
		diff = 360 - d
	}
	turn := math.Min(diff/2, 15)
	if d > 180 {
		return wrap(H + -turn)
	}
	return wrap(H + turn)
}

// freeL is a drawn lightness, used only when there is no page.
func (e *Env) freeL(u func() float64) (float64, bool) {
	if e.r.page != nil {
		return 0, false
	}
	return float64(u()*0.3) + 0.45, true
}

func (e *Env) colour(H, C, L float64, free bool) *Colour {
	return &Colour{r: e.r, hue: H, chroma: C, free: L, hasFree: free}
}

// drawnChroma draws a chroma only when the accent does not set one.
func (e *Env) drawnChroma() float64 {
	if e.r.hasC {
		return e.chromaFor(0)
	}
	return e.chromaFor(e.draw())
}

func (e *Env) fixedChroma() float64 { return e.chromaFor(0.5) }

func (e *Env) drawn(bands []interval) *Colour {
	set := allowed(bands, e.r.avoid, e.r.spread)
	H := along(set, e.draw())
	C := e.drawnChroma()
	L, free := e.freeL(e.draw)
	return e.colour(e.lean(H), C, L, free)
}

// Random is one colour from the environment's seed.
func (e *Env) Random() *Colour {
	e.mu.Lock()
	defer e.mu.Unlock()
	at := e.at
	c := e.drawn(e.r.bands)
	c.Replay = e.replayOf("random", nil, at)
	return c
}

// RandomN is a palette of n colours kept apart (or drawn each on its own, when Independent).
func (e *Env) RandomN(n int, o RandomOptions) Palette {
	extra, err := huesOf(o.Avoid)
	if err != nil {
		return Palette{}
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	at := e.at
	spread := o.Spread
	if spread == 0 {
		spread = e.r.spread
	}
	set := allowed(allowed(e.r.bands, e.r.avoid, e.r.spread), extra, spread)
	out := make([]*Colour, 0, n)
	if o.Independent {
		for i := 0; i < n; i++ {
			out = append(out, e.drawn(set))
		}
	} else {
		var taken []float64
		for i := 0; i < n; i++ {
			var H float64
			if i == 0 {
				H = along(set, e.draw())
			} else {
				H = farthest(set, append(append([]float64(nil), taken...), extra...))
			}
			taken = append(taken, H)
			lh := e.lean(H)
			C := e.drawnChroma()
			L, free := e.freeL(e.draw)
			out = append(out, e.colour(lh, C, L, free))
		}
	}
	args := []kv{{"n", strconv.Itoa(n)}, {"avoid", avoidReplay(o.Avoid)}, numArg("spread", o.Spread)}
	if o.Independent {
		args = append(args, kv{"independent", "1"})
	}
	return Palette{Colours: out, Replay: e.replayOf("random", args, at)}
}

// FromText is the same colour for the same text, every time, whatever the seed.
func (e *Env) FromText(text string) *Colour {
	set := allowed(e.r.bands, e.r.avoid, e.r.spread)
	H := along(set, unit(hash(text, -1)))
	C := e.chromaFor(unit(hash(text, 1)))
	L, free := e.freeL(func() float64 { return unit(hash(text, 2)) })
	return e.colour(e.lean(H), C, L, free)
}

// start is the first hue of a palette, and whether it drew from the stream.
func (e *Env) start(from, fallback string) (float64, bool) {
	f := from
	if f == "" {
		f = fallback
		if e.r.hasHue {
			f = "accent"
		}
	}
	switch f {
	case "accent":
		if e.r.hasHue {
			return e.r.accentH, false
		}
		return 0, false
	case "zero":
		return 0, false
	case "random":
		return e.draw() * 360, true
	}
	return unit(hash(f, -1)) * 360, false
}

// Fluent is n colours that sit well side by side: one strength and lightness, hues in even
// steps.
func (e *Env) Fluent(n int, o FluentOptions) Palette {
	e.mu.Lock()
	defer e.mu.Unlock()
	at := e.at
	s, drew := e.start(o.From, "random")
	arc := o.Arc
	if arc == 0 {
		arc = math.Min(360, float64(40*n))
	}
	stepOf := func(a float64) float64 {
		if a >= 360 {
			return 360 / float64(n)
		}
		if n > 1 {
			return a / float64(n-1)
		}
		return 0
	}
	step := stepOf(arc)
	if n > 1 && step < 18 {
		arc = math.Min(360, float64(18*n))
		step = stepOf(arc)
	}
	alternate := n > 1 && step < 18
	C := e.fixedChroma()
	out := make([]*Colour, 0, n)
	for i := 0; i < n; i++ {
		c := e.colour(wrap(float64(float64(i)*step)+s), C, 0.6, e.r.page == nil)
		if alternate {
			if i%2 == 0 {
				c.shift = 0.06
			} else {
				c.shift = -0.06
			}
		}
		out = append(out, c)
	}
	p := Palette{Colours: out}
	if drew {
		p.Replay = e.replayOf("fluent", []kv{{"n", strconv.Itoa(n)}, numArg("arc", o.Arc), {"from", o.From}}, at)
	}
	return p
}

// Distinct is the n most different colours in this environment, the same every time.
func (e *Env) Distinct(n int, o DistinctOptions) Palette {
	extra, err := huesOf(o.Avoid)
	if err != nil {
		return Palette{}
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	at := e.at
	spread := o.Spread
	if spread == 0 {
		spread = e.r.spread
	}
	set := allowed(allowed(e.r.bands, e.r.avoid, e.r.spread), extra, spread)
	s, drew := e.start(o.From, "zero")
	var first float64
	if inSet(set, wrap(s)) {
		first = wrap(s)
	} else if len(extra) > 0 {
		first = farthest(set, extra)
	} else {
		first = farthest(set, []float64{wrap(s) + 180})
	}
	taken := []float64{first}
	for i := 1; i < n; i++ {
		taken = append(taken, farthest(set, append(append([]float64(nil), taken...), extra...)))
	}
	C := e.fixedChroma()
	out := make([]*Colour, len(taken))
	for i, H := range taken {
		out[i] = e.colour(H, C, 0.6, e.r.page == nil)
	}
	p := Palette{Colours: out}
	if drew {
		p.Replay = e.replayOf("distinct", []kv{{"n", strconv.Itoa(n)}, {"from", o.From}, {"avoid", avoidReplay(o.Avoid)}, numArg("spread", o.Spread)}, at)
	}
	return p
}

// Next is the one colour most different from those in use (hex colours). It returns nil when
// one of them is not a colour.
func (e *Env) Next(existing []string) *Colour {
	var hues []float64
	for _, x := range existing {
		c, err := parse(x)
		if err != nil {
			return nil
		}
		if _, C, H := lch(c); C >= 0.02 {
			hues = append(hues, H)
		}
	}
	set := allowed(e.r.bands, e.r.avoid, e.r.spread)
	var H float64
	if len(hues) > 0 {
		H = farthest(set, append(hues, e.r.avoid...))
	} else {
		H = along(set, 0)
	}
	return e.colour(H, e.fixedChroma(), 0.6, e.r.page == nil)
}

// Hue is a colour at hue h, as Distinct's and Next's are made: for a hue kept from one of
// them. It returns nil when h is not a number.
func (e *Env) Hue(h float64) *Colour {
	if math.IsNaN(h) || math.IsInf(h, 0) {
		return nil
	}
	return e.colour(wrap(h), e.fixedChroma(), 0.6, e.r.page == nil)
}

// Adapt is a colour you already have, its hue kept, made at home in this environment. It
// returns nil when hex is not a colour.
func (e *Env) Adapt(hex string) *Colour {
	c, err := parse(hex)
	if err != nil {
		return nil
	}
	_, C0, H0 := lch(c)
	C := e.r.chroma
	if !e.r.hasC {
		C = math.Min(0.15, math.Max(0.04, C0))
	}
	if e.r.page != nil {
		C = math.Min(0.16, math.Max(C, 0.9*e.r.pageC))
	}
	return e.colour(e.lean(H0), C, 0.6, e.r.page == nil)
}

// Swatch is any background with its border and the inks that read on it. It returns nil when
// a colour given is not one.
func (e *Env) Swatch(bg string, o SwatchOptions) *Swatch {
	b, err := parse(bg)
	if err != nil {
		return nil
	}
	around := e.r.page
	if o.Around != "" {
		a, err := parse(o.Around)
		if err != nil {
			return nil
		}
		around = &a
	}
	inks := e.r.inks
	if o.Inks != nil {
		inks = nil
		for _, s := range o.Inks {
			c, err := parse(s)
			if err != nil {
				return nil
			}
			inks = append(inks, c)
		}
	}
	return newSwatch(e.r, b, around, inks)
}

// Contrast is the contrast of an ink on a background by this environment's measure, or NaN
// when either is not a colour.
func (e *Env) Contrast(ink, bg string) float64 {
	a, err1 := parse(ink)
	b, err2 := parse(bg)
	if err1 != nil || err2 != nil {
		return math.NaN()
	}
	return measureOf(e.r.measure)(a, b)
}

// Replay makes the colours a replay string made, again. A single random colour comes back as
// a palette of one.
func Replay(s string) (Palette, error) {
	parts := strings.Split(s, ";")
	if parts[0] != "gulal1" {
		return Palette{}, fmt.Errorf("gulal: a replay from another version: %s", parts[0])
	}
	if len(parts) < 2 {
		return Palette{}, errors.New("gulal: nothing to replay")
	}
	method := parts[1]
	kvs := map[string]string{}
	for _, p := range parts[2:] {
		k, v, _ := strings.Cut(p, "=")
		d, err := url.PathUnescape(v)
		if err != nil {
			return Palette{}, fmt.Errorf("gulal: a bad replay value %q: %w", v, err)
		}
		kvs[k] = d
	}
	num := func(k string) (float64, error) {
		if kvs[k] == "" {
			return 0, nil
		}
		return strconv.ParseFloat(kvs[k], 64)
	}
	list := func(k string) []string {
		if kvs[k] == "" {
			return nil
		}
		return strings.Split(kvs[k], ",")
	}
	// in an avoid list a colour is written with its #, a hue as a plain number
	avoid := func(k string) ([]string, error) {
		l := list(k)
		for _, x := range l {
			if _, ok := isHue(x); !ok && !strings.HasPrefix(x, "#") {
				return nil, fmt.Errorf("gulal: a bad avoid entry in a replay: %s", x)
			}
		}
		return l, nil
	}
	eavoid, err := avoid("eavoid")
	if err != nil {
		return Palette{}, err
	}
	xavoid, err := avoid("avoid")
	if err != nil {
		return Palette{}, err
	}
	o := Options{Seed: kvs["seed"], Scheme: kvs["scheme"], Background: kvs["bg"], Accent: kvs["accent"], Contrast: kvs["contrast"], Avoid: eavoid, Inks: list("inks")}
	for _, p := range list("hues") {
		a, b, ok := strings.Cut(p, "..")
		fa, err1 := strconv.ParseFloat(a, 64)
		fb, err2 := strconv.ParseFloat(b, 64)
		if !ok || err1 != nil || err2 != nil {
			return Palette{}, fmt.Errorf("gulal: a bad hue band in a replay: %s", p)
		}
		o.Hues = append(o.Hues, [2]float64{fa, fb})
	}
	if o.Spread, err = num("espread"); err != nil {
		return Palette{}, err
	}
	if o.Text, err = num("text"); err != nil {
		return Palette{}, err
	}
	if o.Seed == "" {
		return Palette{}, errors.New("gulal: a replay with no seed")
	}
	e := New(o)
	if e == nil {
		return Palette{}, fmt.Errorf("gulal: a replay of a bad environment: %w", o.Validate())
	}
	if kvs["at"] != "" {
		at, err := strconv.ParseUint(kvs["at"], 10, 32)
		if err != nil {
			return Palette{}, err
		}
		e.at = uint32(at)
	}
	n := 1
	hasN := kvs["n"] != ""
	if hasN {
		if n, err = strconv.Atoi(kvs["n"]); err != nil {
			return Palette{}, err
		}
	}
	spread, err := num("spread")
	if err != nil {
		return Palette{}, err
	}
	switch method {
	case "random":
		if !hasN {
			c := e.Random()
			return Palette{Colours: []*Colour{c}, Replay: c.Replay}, nil
		}
		return e.RandomN(n, RandomOptions{Avoid: xavoid, Spread: spread, Independent: kvs["independent"] == "1"}), nil
	case "fluent":
		arc, err := num("arc")
		if err != nil {
			return Palette{}, err
		}
		return e.Fluent(n, FluentOptions{Arc: arc, From: kvs["from"]}), nil
	case "distinct":
		return e.Distinct(n, DistinctOptions{From: kvs["from"], Avoid: xavoid, Spread: spread}), nil
	}
	return Palette{}, fmt.Errorf("gulal: nothing to replay for %s", method)
}
