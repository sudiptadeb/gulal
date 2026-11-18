package gulal

// Hues: bands, less what is avoided, laid end to end.

import (
	"math"
	"strconv"
	"strings"
)

type interval [2]float64

// wrap brings a hue into [0, 360). math.Mod keeps the dividend's sign, as JavaScript's % does.
func wrap(h float64) float64 {
	h = math.Mod(h, 360)
	if h < 0 {
		return h + 360
	}
	return h
}

func circular(a, b float64) float64 {
	d := math.Abs(wrap(a) - wrap(b))
	return math.Min(d, 360-d)
}

func bandsOf(hues [][2]float64) []interval {
	if hues == nil {
		hues = [][2]float64{{0, 360}}
	}
	var out []interval
	for _, p := range hues {
		a0, b0 := p[0], p[1]
		if b0-a0 >= 360 {
			return []interval{{0, 360}}
		}
		a, b := wrap(a0), wrap(b0)
		if a < b {
			out = append(out, interval{a, b})
		} else if a > b {
			out = append(out, interval{a, 360}, interval{0, b})
		}
	}
	return out
}

func subtract(set []interval, cut interval) []interval {
	a, b := cut[0], cut[1]
	var out []interval
	for _, s := range set {
		x, y := s[0], s[1]
		if b <= x || a >= y {
			out = append(out, s)
		} else {
			if a > x {
				out = append(out, interval{x, a})
			}
			if b < y {
				out = append(out, interval{b, y})
			}
		}
	}
	return out
}

func totalOf(set []interval) float64 {
	n := 0.0
	for _, s := range set {
		n = (n + s[1]) - s[0]
	}
	return n
}

func allowed(bands []interval, avoid []float64, spread float64) []interval {
	set := append([]interval(nil), bands...)
	for _, h := range avoid {
		a, b := h-spread, h+spread
		for _, piece := range []interval{{a, b}, {a - 360, b - 360}, {a + 360, b + 360}} {
			set = subtract(set, piece)
		}
	}
	if totalOf(set) > 0 {
		return set
	}
	return bands
}

func along(set []interval, u float64) float64 {
	x := u * totalOf(set)
	for _, s := range set {
		if x < s[1]-s[0] {
			return s[0] + x
		}
		x -= s[1] - s[0]
	}
	return set[len(set)-1][1] - 1e-9
}

func inSet(set []interval, h float64) bool {
	for _, s := range set {
		if h >= s[0] && h < s[1] {
			return true
		}
	}
	return false
}

// farthest is the hue farthest from every hue in from, on a 0.5° grid of the allowed hues.
func farthest(set []interval, from []float64) float64 {
	best, bestScore := -1.0, -1.0
	for k := 0; k < 720; k++ {
		h := float64(k) * 0.5
		if !inSet(set, h) {
			continue
		}
		score := 360.0
		for _, f := range from {
			score = math.Min(score, circular(h, f))
		}
		if score > bestScore {
			bestScore = score
			best = h
		}
	}
	if best < 0 {
		return 0
	}
	return best
}

// isHue says whether an avoid entry is a hue in degrees rather than a colour: it does not
// start with '#' and reads as a number.
func isHue(x string) (float64, bool) {
	s := strings.TrimSpace(x)
	if strings.HasPrefix(s, "#") {
		return 0, false
	}
	f, err := strconv.ParseFloat(s, 64)
	return f, err == nil
}

// hueOf is the hue an avoid entry names, ok false for a colour with too little chroma.
func hueOf(x string) (float64, bool, error) {
	if f, ok := isHue(x); ok {
		return wrap(f), true, nil
	}
	c, err := parse(x)
	if err != nil {
		return 0, false, err
	}
	_, C, H := lch(c)
	if C < 0.02 {
		return 0, false, nil
	}
	return H, true, nil
}

func huesOf(list []string) ([]float64, error) {
	var out []float64
	for _, a := range list {
		h, ok, err := hueOf(a)
		if err != nil {
			return nil, err
		}
		if ok {
			out = append(out, h)
		}
	}
	return out, nil
}
