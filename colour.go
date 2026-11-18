package gulal

import (
	"math"
	"math/big"
	"sync"
)

// Colour is a made colour: a hue and a chroma, with its fill, inks and swatches worked out
// when first asked for and kept.
type Colour struct {
	// Replay makes the colour again, when it was drawn at random; "" otherwise.
	Replay string

	r       *resolved
	hue     float64
	chroma  float64
	free    float64 // the drawn lightness, when hasFree
	hasFree bool
	shift   float64 // lightness offset for fluent palettes that alternate

	hexOnce, solidOnce, softOnce, inkOnce, markOnce sync.Once
	hex, ink, mark                                  string
	solid, soft                                     *Swatch
}

type ctx struct {
	score     scoreFn
	prefersUp bool
}

func (c *Colour) ctx() ctx {
	return ctx{measureOf(c.r.measure), c.r.pageDark}
}

func (c *Colour) page() rgb8 {
	if c.r.page != nil {
		return *c.r.page
	}
	return white
}

// Hex is the colour as a solid fill.
func (c *Colour) Hex() string {
	c.hexOnce.Do(func() { c.hex = hexOf(c.fill()) })
	return c.hex
}

// String is Hex.
func (c *Colour) String() string { return c.Hex() }

func (c *Colour) fill() rgb8 {
	r := c.r
	var L0 float64
	if c.hasFree || r.page == nil {
		L0 = 0.6
		if c.hasFree {
			L0 = c.free
		}
	} else {
		L0 = 0.6
		if r.pageDark {
			L0 = 0.62
		}
		if math.Abs(L0-r.pageL) < 0.15 {
			if r.pageL >= 0.5 {
				L0 = r.pageL - 0.15
			} else {
				L0 = r.pageL + 0.15
			}
		}
	}
	L0 += c.shift
	fill := rgb(L0, c.chroma, c.hue)
	inks := r.inks
	if len(inks) == 0 {
		inks = []rgb8{white, nearBlack}
	}
	score := c.ctx().score
	ink := best(inks, fill, score)
	level := levelsOf(r.measure).ink
	if score(ink, fill) >= level {
		return fill
	}
	// move the fill away from its ink until the ink reads
	iL, _, _ := lch(ink)
	end := 1.0
	if iL > L0 {
		end = 0
	}
	reads := func(L float64) bool { return score(ink, rgb(L, c.chroma, c.hue)) >= level }
	if !reads(end) {
		return rgb(end, c.chroma, c.hue)
	}
	lo, hi := L0, end
	for i := 0; i < 20; i++ {
		mid := (lo + hi) / 2
		if reads(mid) {
			hi = mid
		} else {
			lo = mid
		}
	}
	return rgb(hi, c.chroma, c.hue)
}

// OKLCH is the fill's lightness, chroma and hue.
func (c *Colour) OKLCH() (l, ch, h float64) {
	v, _ := parse(c.Hex())
	return lch(v)
}

// CSS is the fill as a CSS oklch() colour.
func (c *Colour) CSS() string {
	l, ch, h := c.OKLCH()
	return "oklch(" + toFixed(l, 3) + " " + toFixed(ch, 3) + " " + toFixed(h, 1) + ")"
}

// Solid is the swatch on the colour's solid fill.
func (c *Colour) Solid() *Swatch {
	c.solidOnce.Do(func() {
		v, _ := parse(c.Hex())
		c.solid = newSwatch(c.r, v, c.r.page, c.r.inks)
	})
	return c.solid
}

// Soft is the swatch on the colour's soft tint, a step off the background.
func (c *Colour) Soft() *Swatch {
	c.softOnce.Do(func() { c.soft = c.makeSoft() })
	return c.soft
}

func (c *Colour) makeSoft() *Swatch {
	r := c.r
	page := c.page()
	pL, pC, dark := 1.0, 0.0, r.scheme == "dark"
	if r.page != nil {
		pL, pC, dark = r.pageL, r.pageC, r.pageDark
	}
	k := 0.45
	if dark {
		k = 0.55
	}
	C := math.Max(c.chroma*k, 0.85*pC)
	var L float64
	switch {
	case pL < 0.45:
		L = math.Max(0.3, pL+0.09)
	case pL < 0.8:
		L = 0.93
		if dark {
			L = 0.3
		}
	default:
		L = math.Min(0.93, pL-0.05)
	}
	away := -0.005
	if L > pL {
		away = 0.005
	}
	need := float64(0.65*pC) + 0.05
	fill := rgb(L, C, c.hue)
	for i := 0; i < 200 && apart(fill, page) < need && L > 0 && L < 1; i++ {
		L += away
		fill = rgb(L, C, c.hue)
	}
	return newSwatch(r, fill, r.page, r.inks)
}

func (c *Colour) inkC() float64 {
	return math.Min(math.Max(c.chroma*1.3, 0.06), 0.15)
}

// Ink is the colour as ink on the environment's background.
func (c *Colour) Ink() string {
	c.inkOnce.Do(func() {
		L0 := 0.8
		if !c.r.pageDark {
			L0 = 0.5
		}
		c.ink = hexOf(reach(L0, c.inkC(), c.hue, c.page(), levelsOf(c.r.measure).ink, c.ctx()))
	})
	return c.ink
}

// Mark is the colour as a dot or a line on the environment's background.
func (c *Colour) Mark() string {
	c.markOnce.Do(func() {
		L0 := 0.66
		if !c.r.pageDark {
			L0 = 0.62
		}
		c.mark = hexOf(reach(L0, c.inkC(), c.hue, c.page(), levelsOf(c.r.measure).mark, c.ctx()))
	})
	return c.mark
}

func best(inks []rgb8, bg rgb8, score scoreFn) rgb8 {
	pick, top := inks[0], score(inks[0], bg)
	for _, c := range inks[1:] {
		if s := score(c, bg); s > top {
			top, pick = s, c
		}
	}
	return pick
}

// reach is the colour of hue H and chroma C, lightness moved from L0 as little as it must for
// it to read on against at level: the least move up past the background's lightness, or
// down, each by bisection; the smaller wins.
func reach(L0, C, H float64, against rgb8, level float64, x ctx) rgb8 {
	bgL, _, _ := lch(against)
	reads := func(L float64) bool { return x.score(rgb(L, C, H), against) >= level }
	side := func(from, to float64) (float64, bool) {
		if !reads(to) {
			return 0, false
		}
		if reads(from) {
			return from, true
		}
		lo, hi := from, to // lo does not read, hi does
		for i := 0; i < 20; i++ {
			mid := (lo + hi) / 2
			if reads(mid) {
				hi = mid
			} else {
				lo = mid
			}
		}
		return hi, true
	}
	up, okUp := side(math.Max(L0, bgL), 1)
	down, okDown := side(math.Min(L0, bgL), 0)
	var L float64
	switch {
	case !okUp && !okDown:
		L = 0
		if x.score(rgb(1, C, H), against) >= x.score(rgb(0, C, H), against) {
			L = 1
		}
	case !okUp:
		L = down
	case !okDown:
		L = up
	default:
		du, dd := up-L0, L0-down
		switch {
		case math.Abs(du-dd) < 1e-9:
			L = down
			if x.prefersUp {
				L = up
			}
		case du < dd:
			L = up
		default:
			L = down
		}
	}
	return rgb(L, C, H)
}

// toFixed is JavaScript's Number.prototype.toFixed for finite numbers below 1e21: the exact
// value rounded to d decimals, a tie away from zero (Go's 'f' format rounds a tie to even).
func toFixed(x float64, d int) string {
	neg := x < 0
	r := new(big.Rat).SetFloat64(math.Abs(x))
	scale := new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(d)), nil)
	r.Mul(r, new(big.Rat).SetInt(scale))
	r.Add(r, big.NewRat(1, 2))
	n := new(big.Int).Quo(r.Num(), r.Denom()) // floor, as r >= 0
	s := n.String()
	for len(s) <= d {
		s = "0" + s
	}
	if d > 0 {
		s = s[:len(s)-d] + "." + s[len(s)-d:]
	}
	if neg { // JavaScript keeps the sign of a negative that rounds to zero
		s = "-" + s
	}
	return s
}
