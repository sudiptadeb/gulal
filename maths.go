package gulal

// Colour maths, contrast and hashing: a line-by-line port of js/src/gulal.ts.
//
// Every product that is added to or subtracted from something is wrapped in float64(...).
// Go may fuse a*b+c into one FMA instruction (it does on arm64), which rounds once instead of
// twice and so gives a different last bit from JavaScript; an explicit conversion forbids the
// fusion. Constants that JavaScript works out at run time (1/2.4, 1+1e-4) are package
// variables here, so that Go does not fold them exactly at compile time.

import (
	"fmt"
	"math"
	"strconv"
	"strings"
)

type rgb8 [3]int

var (
	white     = rgb8{255, 255, 255}
	nearBlack = rgb8{31, 31, 31}
)

// Run-time constants (see above).
var (
	two4      = 2.4
	invTwo4   = 1 / two4
	gamutEps  = 1e-4
	gamutHigh = 1 + gamutEps
)

func isHexDigit(c byte) bool {
	return (c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')
}

// parse reads #rgb or #rrggbb (the # optional, any case, surrounding space ignored).
func parse(hex string) (rgb8, error) {
	h := strings.TrimSpace(hex)
	h = strings.TrimPrefix(h, "#")
	ok := len(h) == 3 || len(h) == 6
	for i := 0; ok && i < len(h); i++ {
		ok = isHexDigit(h[i])
	}
	if !ok {
		return rgb8{}, fmt.Errorf("gulal: not a colour: %s", hex)
	}
	if len(h) == 3 {
		h = string([]byte{h[0], h[0], h[1], h[1], h[2], h[2]})
	}
	var c rgb8
	for i := 0; i < 3; i++ {
		v, _ := strconv.ParseUint(h[2*i:2*i+2], 16, 8)
		c[i] = int(v)
	}
	return c, nil
}

func hexOf(c rgb8) string {
	return fmt.Sprintf("#%02x%02x%02x", c[0], c[1], c[2])
}

func toLinear(v8 int) float64 {
	v := float64(v8) / 255
	if v <= 0.04045 {
		return v / 12.92
	}
	return math.Pow((v+0.055)/1.055, two4)
}

func fromLinear(v float64) int {
	x := math.Min(1, math.Max(0, v))
	var s float64
	if x <= 0.0031308 {
		s = 12.92 * x
	} else {
		s = float64(1.055*math.Pow(x, invTwo4)) - 0.055
	}
	return int(math.Floor(float64(s*255) + 0.5))
}

// lch is the OKLCH of an 8-bit colour: L, C, H. A colour with almost no chroma has hue 0.
func lch(c rgb8) (float64, float64, float64) {
	r, g, b := toLinear(c[0]), toLinear(c[1]), toLinear(c[2])
	l := math.Cbrt(float64(float64(0.4122214708*r)+float64(0.5363325363*g)) + float64(0.0514459929*b))
	m := math.Cbrt(float64(float64(0.2119034982*r)+float64(0.6806995451*g)) + float64(0.1073969566*b))
	s := math.Cbrt(float64(float64(0.0883024619*r)+float64(0.2817188376*g)) + float64(0.6299787005*b))
	L := float64(float64(0.2104542553*l)+float64(0.793617785*m)) - float64(0.0040720420*s)
	A := float64(float64(1.9779984951*l)-float64(2.428592205*m)) + float64(0.4505937099*s)
	B := float64(float64(0.0259040371*l)+float64(0.7827717662*m)) - float64(0.808675766*s)
	C := math.Sqrt(float64(A*A) + float64(B*B))
	if C < 1e-4 {
		return L, C, 0
	}
	H := float64(math.Atan2(B, A)*180) / math.Pi
	if H < 0 {
		H += 360
	}
	return L, C, H
}

func linear(L, C, H float64) [3]float64 {
	hr := float64(H*math.Pi) / 180
	a, b := float64(C*math.Cos(hr)), float64(C*math.Sin(hr))
	l_ := float64(L+float64(0.3963377774*a)) + float64(0.2158037573*b)
	m_ := float64(L-float64(0.1055613458*a)) - float64(0.0638541728*b)
	s_ := float64(L-float64(0.0894841775*a)) - float64(1.291485548*b)
	l := float64(float64(l_*l_) * l_)
	m := float64(float64(m_*m_) * m_)
	s := float64(float64(s_*s_) * s_)
	return [3]float64{
		float64(float64(4.0767416621*l)-float64(3.3077115913*m)) + float64(0.2309699292*s),
		float64(float64(-1.2684380046*l)+float64(2.6097574011*m)) - float64(0.3413193965*s),
		float64(float64(-0.0041960863*l)-float64(0.7034186147*m)) + float64(1.707614701*s),
	}
}

func inGamut(v [3]float64) bool {
	for _, x := range v {
		if !(x >= -gamutEps && x <= gamutHigh) {
			return false
		}
	}
	return true
}

// rgb is an OKLCH colour as 8-bit sRGB: lightness and hue kept, chroma lowered until it fits.
func rgb(L, C, H float64) rgb8 {
	L = math.Min(1, math.Max(0, L))
	C = math.Max(0, C)
	v := linear(L, C, H)
	if !inGamut(v) {
		lo, hi := 0.0, C
		for i := 0; i < 16; i++ {
			mid := (lo + hi) / 2
			if inGamut(linear(L, mid, H)) {
				lo = mid
			} else {
				hi = mid
			}
		}
		v = linear(L, lo, H)
	}
	return rgb8{fromLinear(v[0]), fromLinear(v[1]), fromLinear(v[2])}
}

// apart is the OKLab distance of two colours.
func apart(x, y rgb8) float64 {
	p0, p1, p2 := lch(x)
	q0, q1, q2 := lch(y)
	pa, pb := float64(p1*math.Cos(float64(p2*math.Pi)/180)), float64(p1*math.Sin(float64(p2*math.Pi)/180))
	qa, qb := float64(q1*math.Cos(float64(q2*math.Pi)/180)), float64(q1*math.Sin(float64(q2*math.Pi)/180))
	d0, d1, d2 := p0-q0, pa-qa, pb-qb
	return math.Sqrt(float64(float64(d0*d0)+float64(d1*d1)) + float64(d2*d2))
}

// ---- contrast: APCA 0.0.98G's lightness contrast (a magnitude), or WCAG 2's ratio.

func apcaY(c rgb8) float64 {
	y := float64(float64(0.2126729*math.Pow(float64(c[0])/255, two4))+float64(0.7151522*math.Pow(float64(c[1])/255, two4))) +
		float64(0.072175*math.Pow(float64(c[2])/255, two4))
	if y < 0.022 {
		y += math.Pow(0.022-y, 1.414)
	}
	return y
}

func apca(text, bg rgb8) float64 {
	t, b := apcaY(text), apcaY(bg)
	if b > t {
		s := float64((math.Pow(b, 0.56) - math.Pow(t, 0.57)) * 1.14)
		if s < 0.1 {
			return 0
		}
		return (s - 0.027) * 100
	}
	s := float64((math.Pow(b, 0.65) - math.Pow(t, 0.62)) * 1.14)
	if s > -0.1 {
		return 0
	}
	return -(s + 0.027) * 100
}

func wcagY(c rgb8) float64 {
	return float64(float64(0.2126*toLinear(c[0]))+float64(0.7152*toLinear(c[1]))) + float64(0.0722*toLinear(c[2]))
}

func wcag(a, b rgb8) float64 {
	x, z := wcagY(a), wcagY(b)
	return (math.Max(x, z) + 0.05) / (math.Min(x, z) + 0.05)
}

type levels struct{ ink, border, mark float64 }

func levelsOf(m string) levels {
	if m == "wcag2" {
		return levels{4.5, 1.3, 3}
	}
	return levels{60, 15, 30}
}

type scoreFn func(a, b rgb8) float64

func measureOf(m string) scoreFn {
	if m == "wcag2" {
		return wcag
	}
	return apca
}

// ---- hashing and the seeded stream

// hash is FNV-1a over the UTF-8 bytes (and one extra byte when extra >= 0), then murmur3's
// finaliser.
func hash(text string, extra int) uint32 {
	h := uint32(0x811c9dc5)
	bytes := []byte(text)
	if extra >= 0 {
		bytes = append(bytes, byte(extra))
	}
	for _, b := range bytes {
		h = (h ^ uint32(b)) * 0x01000193
	}
	h ^= h >> 16
	h *= 0x85ebca6b
	h ^= h >> 13
	h *= 0xc2b2ae35
	h ^= h >> 16
	return h
}

func unit(n uint32) float64 { return float64(n) / 4294967296 }

const golden = 0x9e3779b9

// splitmix is SplitMix32's output: murmur3's finaliser of the state.
func splitmix(z uint32) uint32 {
	z = (z ^ (z >> 16)) * 0x85ebca6b
	z = (z ^ (z >> 13)) * 0xc2b2ae35
	return z ^ (z >> 16)
}
