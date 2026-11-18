package gulal

import (
	"math"
	"sync"
)

// Harmony is ink in every relation on the colour wheel.
type Harmony struct {
	Analogous     []string `json:"analogous"`
	Complementary []string `json:"complementary"`
	Split         []string `json:"split"`
	Triadic       []string `json:"triadic"`
	Square        []string `json:"square"`
}

// Swatch is a background with what goes on it.
type Swatch struct {
	Background    string
	Border        string
	Ink           string
	Muted         string
	Tonal         string
	Complementary string
	Accent        string // "" when the environment has no accent with a hue

	bg      rgb8
	baseH   float64
	light   bool
	c       ctx
	level   float64
	hOnce   sync.Once
	harmony Harmony
}

func newSwatch(r *resolved, bg rgb8, around *rgb8, inks []rgb8) *Swatch {
	score := measureOf(r.measure)
	L, C, H := lch(bg)
	s := &Swatch{bg: bg}
	s.light = score(nearBlack, bg) >= score(white, bg)
	s.c = ctx{score: score, prefersUp: !s.light}
	if r.scheme != "" {
		s.c.prefersUp = r.scheme == "dark"
	}
	s.level = r.text
	s.baseH = H
	if C < 0.02 {
		s.baseH = 0
		if r.hasHue {
			s.baseH = r.accentH
		}
	}
	s.Background = hexOf(bg)

	// border: its hue, away from what is around it (or toward the ink's side), until it shows
	bc := math.Min(C*1.1, 0.37)
	aL := L
	if around != nil {
		aL, _, _ = lch(*around)
	}
	down := s.light
	if around != nil && math.Abs(aL-L) > 0.02 {
		down = aL > L
	}
	step, bstep := 0.12, 0.005
	if down {
		step, bstep = -0.12, -0.005
	}
	bl := L + step
	border := rgb(bl, bc, H)
	borderLevel := levelsOf(r.measure).border
	shows := func(x rgb8) bool {
		return around == nil || math.Max(score(x, *around), score(*around, x)) >= borderLevel
	}
	for i := 0; i < 200 && !shows(border) && bl > 0 && bl < 1; i++ {
		bl += bstep
		border = rgb(bl, bc, H)
	}
	s.Border = hexOf(border)

	// ink: the best of the inks given, else white or near-black
	if len(inks) == 0 {
		inks = []rgb8{white, nearBlack}
	}
	ink := best(inks, bg, score)
	s.Ink = hexOf(ink)

	// muted: the ink moved toward the background until it only just reads
	muted := ink
	if score(ink, bg) >= s.level {
		iL, iC, iH := lch(ink)
		at := func(t float64) rgb8 { return rgb(float64((L-iL)*t)+iL, iC, iH) }
		lo, hi := 0.0, 1.0 // lo reads, hi does not
		for i := 0; i < 20; i++ {
			mid := (lo + hi) / 2
			if score(at(mid), bg) >= s.level {
				lo = mid
			} else {
				hi = mid
			}
		}
		muted = at(lo)
	}
	s.Muted = hexOf(muted)

	tc := C
	if C >= 0.02 {
		tc = math.Min(math.Max(C*1.3, 0.06), 0.15)
	}
	s.Tonal = hexOf(reach(s.start(), tc, H, bg, s.level, s.c))
	s.Complementary = s.Turn(180)
	if r.accent != nil && r.hasHue {
		_, aC, _ := lch(*r.accent)
		s.Accent = hexOf(reach(s.start(), math.Min(math.Max(aC, 0.06), 0.15), r.accentH, bg, s.level, s.c))
	}
	return s
}

func (s *Swatch) start() float64 {
	if s.light {
		return 0.5
	}
	return 0.8
}

// Turn is ink in the background's hue turned by deg degrees, made to read.
func (s *Swatch) Turn(deg float64) string {
	return hexOf(reach(s.start(), 0.15, wrap(s.baseH+deg), s.bg, s.level, s.c))
}

func (s *Swatch) turns(angles ...float64) []string {
	out := make([]string, len(angles))
	for i, a := range angles {
		out[i] = s.Turn(a)
	}
	return out
}

// Harmony is ink in every relation on the colour wheel, made when first asked for.
func (s *Swatch) Harmony() Harmony {
	s.hOnce.Do(func() {
		s.harmony = Harmony{
			Analogous:     s.turns(50, -50),
			Complementary: s.turns(180),
			Split:         s.turns(150, -150),
			Triadic:       s.turns(120, -120),
			Square:        s.turns(90, 180, -90),
		}
	})
	return s.harmony
}

// Vars is the swatch as CSS custom properties: --<prefix>-background, -border, -ink, ...
// An empty prefix means "gulal".
func (s *Swatch) Vars(prefix string) map[string]string {
	if prefix == "" {
		prefix = "gulal"
	}
	p := "--" + prefix + "-"
	out := map[string]string{
		p + "background":    s.Background,
		p + "border":        s.Border,
		p + "ink":           s.Ink,
		p + "muted":         s.Muted,
		p + "tonal":         s.Tonal,
		p + "complementary": s.Complementary,
	}
	if s.Accent != "" {
		out[p+"accent"] = s.Accent
	}
	return out
}
