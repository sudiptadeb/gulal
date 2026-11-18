package gulal

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"testing"
)

// The vectors: every call of js/test/cases.mjs, written down by the TypeScript.

type vecEnv struct {
	Seed       string            `json:"seed"`
	Scheme     string            `json:"scheme"`
	Background string            `json:"background"`
	Accent     string            `json:"accent"`
	Contrast   string            `json:"contrast"`
	Text       float64           `json:"text"`
	Hues       [][2]float64      `json:"hues"`
	Avoid      []json.RawMessage `json:"avoid"`
	Spread     float64           `json:"spread"`
	Inks       []string          `json:"inks"`
}

type vecCall struct {
	Call string            `json:"call"`
	Args []json.RawMessage `json:"args"`
	Out  json.RawMessage   `json:"out"`
}

type vectors struct {
	Version int `json:"version"`
	Cases   []struct {
		Env   json.RawMessage `json:"env"`
		Calls []vecCall       `json:"calls"`
	} `json:"cases"`
}

// avoidList reads a JSON list of hex strings and numbers into Go's Avoid strings.
func avoidList(raw []json.RawMessage) []string {
	var out []string
	for _, r := range raw {
		var s string
		if json.Unmarshal(r, &s) == nil {
			out = append(out, s)
			continue
		}
		var f float64
		if err := json.Unmarshal(r, &f); err != nil {
			panic(err)
		}
		out = append(out, strconv.FormatFloat(f, 'f', -1, 64))
	}
	return out
}

func optionsOf(raw json.RawMessage) Options {
	var v vecEnv
	if err := json.Unmarshal(raw, &v); err != nil {
		panic(err)
	}
	return Options{Seed: v.Seed, Scheme: v.Scheme, Background: v.Background, Accent: v.Accent, Contrast: v.Contrast, Text: v.Text,
		Hues: v.Hues, Avoid: avoidList(v.Avoid), Spread: v.Spread, Inks: v.Inks}
}

// The serialisers of cases.mjs: swatchOut, colourOut, paletteOut.

func swatchOut(s *Swatch) map[string]any {
	m := map[string]any{
		"background": s.Background, "border": s.Border, "ink": s.Ink, "muted": s.Muted,
		"tonal": s.Tonal, "complementary": s.Complementary, "harmony": s.Harmony(), "turn75": s.Turn(75),
	}
	if s.Accent != "" {
		m["accent"] = s.Accent
	}
	return m
}

func orNull(s string) any {
	if s == "" {
		return nil
	}
	return s
}

func colourOut(c *Colour) map[string]any {
	return map[string]any{"hex": c.Hex(), "css": c.CSS(), "ink": c.Ink(), "mark": c.Mark(),
		"soft": swatchOut(c.Soft()), "solid": swatchOut(c.Solid()), "replay": orNull(c.Replay)}
}

func paletteOut(p Palette) map[string]any {
	cs := make([]any, len(p.Colours))
	for i, c := range p.Colours {
		cs[i] = colourOut(c)
	}
	return map[string]any{"replay": orNull(p.Replay), "colours": cs}
}

type callOpts struct {
	Avoid       []json.RawMessage `json:"avoid"`
	Spread      float64           `json:"spread"`
	Independent bool              `json:"independent"`
	Arc         float64           `json:"arc"`
	From        string            `json:"from"`
	Around      string            `json:"around"`
	Inks        []string          `json:"inks"`
}

func mustArg(raw json.RawMessage, v any) {
	if err := json.Unmarshal(raw, v); err != nil {
		panic(err)
	}
}

// call makes one recorded call on g and serialises its result as cases.mjs does.
func call(g *Env, c vecCall) any {
	var o callOpts
	if len(c.Args) > 1 && c.Call != "contrast" {
		mustArg(c.Args[1], &o)
	}
	str := func(i int) string { var s string; mustArg(c.Args[i], &s); return s }
	num := func(i int) int { var n int; mustArg(c.Args[i], &n); return n }
	switch c.Call {
	case "fromText":
		return colourOut(g.FromText(str(0)))
	case "random":
		if len(c.Args) == 0 {
			return colourOut(g.Random())
		}
		return paletteOut(g.RandomN(num(0), RandomOptions{Avoid: avoidList(o.Avoid), Spread: o.Spread, Independent: o.Independent}))
	case "fluent":
		return paletteOut(g.Fluent(num(0), FluentOptions{Arc: o.Arc, From: o.From}))
	case "distinct":
		return paletteOut(g.Distinct(num(0), DistinctOptions{From: o.From, Avoid: avoidList(o.Avoid), Spread: o.Spread}))
	case "next":
		var list []string
		mustArg(c.Args[0], &list)
		return colourOut(g.Next(list))
	case "adapt":
		return colourOut(g.Adapt(str(0)))
	case "swatch":
		return swatchOut(g.Swatch(str(0), SwatchOptions{Around: o.Around, Inks: o.Inks}))
	case "contrast":
		v, _ := strconv.ParseFloat(strconv.FormatFloat(g.Contrast(str(0), str(1)), 'f', 6, 64), 64)
		return v
	}
	panic("unknown call " + c.Call)
}

// generic turns a value into what encoding/json gives back for it, for comparing.
func generic(v any) any {
	b, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	var out any
	if err := json.Unmarshal(b, &out); err != nil {
		panic(err)
	}
	return out
}

// diff is the first place two decoded JSON values differ, as a path and both values.
func diff(path string, want, got any) string {
	switch w := want.(type) {
	case map[string]any:
		g, ok := got.(map[string]any)
		if !ok {
			return fmt.Sprintf("%s: want %v, got %v", path, want, got)
		}
		keys := map[string]bool{}
		for k := range w {
			keys[k] = true
		}
		for k := range g {
			keys[k] = true
		}
		for _, k := range sortedKeys(keys) {
			if d := diff(path+"."+k, w[k], g[k]); d != "" {
				return d
			}
		}
		return ""
	case []any:
		g, ok := got.([]any)
		if !ok || len(g) != len(w) {
			return fmt.Sprintf("%s: want %v, got %v", path, want, got)
		}
		for i := range w {
			if d := diff(fmt.Sprintf("%s[%d]", path, i), w[i], g[i]); d != "" {
				return d
			}
		}
		return ""
	}
	if !reflect.DeepEqual(want, got) {
		return fmt.Sprintf("%s: want %v, got %v", path, want, got)
	}
	return ""
}

func sortedKeys(m map[string]bool) []string {
	var out []string
	for k := range m {
		out = append(out, k)
	}
	for i := range out {
		for j := i + 1; j < len(out); j++ {
			if out[j] < out[i] {
				out[i], out[j] = out[j], out[i]
			}
		}
	}
	return out
}

func TestVectors(t *testing.T) {
	b, err := os.ReadFile("vectors/vectors.json")
	if err != nil {
		t.Fatal(err)
	}
	var v vectors
	if err := json.Unmarshal(b, &v); err != nil {
		t.Fatal(err)
	}
	if v.Version != 1 {
		t.Fatalf("vectors version %d, want 1", v.Version)
	}
	n, bad := 0, 0
	for _, c := range v.Cases {
		g := New(optionsOf(c.Env))
		if g == nil {
			t.Fatalf("env %s does not validate: %v", c.Env, optionsOf(c.Env).Validate())
		}
		for _, s := range c.Calls {
			n++
			var want any
			if err := json.Unmarshal(s.Out, &want); err != nil {
				t.Fatal(err)
			}
			got := generic(call(g, s))
			var d string
			if s.Call == "contrast" {
				if w, g := want.(float64), got.(float64); math.Abs(w-g) > 1e-6 {
					d = fmt.Sprintf(": want %v, got %v", w, g)
				}
			} else {
				d = diff("", want, got)
			}
			if d != "" {
				bad++
				if bad <= 8 {
					args, _ := json.Marshal(s.Args)
					t.Errorf("differs: env %s, call %d %s(%s)\n  at out%s", c.Env, n, s.Call, args, d)
				}
			}
		}
	}
	if bad > 0 {
		t.Fatalf("vectors: %d of %d calls differ", bad, n)
	}
	t.Logf("vectors: all %d calls match", n)
}

// What the API promises, beyond the vectors (js/test/api.test.mjs).

func TestSameTextSameColour(t *testing.T) {
	a := New(Options{Seed: "api", Scheme: "light", Accent: "#0faaaa"}).FromText("news.example.com").Hex()
	b := New(Options{Seed: "other", Scheme: "light", Accent: "#0faaaa"}).FromText("news.example.com").Hex()
	if a != b {
		t.Fatalf("same text, different colours: %s %s", a, b)
	}
}

func hexes(p Palette) []string {
	var out []string
	for _, c := range p.Colours {
		out = append(out, c.Hex())
	}
	return out
}

func TestReplay(t *testing.T) {
	a, b := New(Options{Seed: "s1"}), New(Options{Seed: "s1"})
	one := a.Random()
	b.Random()
	pa, pb := a.RandomN(5, RandomOptions{}), b.RandomN(5, RandomOptions{})
	if !reflect.DeepEqual(hexes(pa), hexes(pb)) {
		t.Fatal("same seed and calls, different colours")
	}
	r, err := Replay(pa.Replay)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(hexes(r), hexes(pa)) {
		t.Fatalf("replay %q gives %v, want %v", pa.Replay, hexes(r), hexes(pa))
	}
	r1, err := Replay(one.Replay)
	if err != nil || len(r1.Colours) != 1 || r1.Colours[0].Hex() != one.Hex() || r1.Replay != one.Replay {
		t.Fatalf("replay of one colour: %v %v", r1, err)
	}
	// an environment with everything set, and a seed that needs escaping
	e := New(Options{Seed: "a;b=c%d", Scheme: "dark", Background: "#1c1d20", Accent: "#faa804", Contrast: "wcag2",
		Hues: [][2]float64{{20, 200.5}}, Avoid: []string{"#d93025", "350"}, Spread: 30, Inks: []string{"#fff4d6", "#14213d"}})
	e.Fluent(2, FluentOptions{From: "random"})
	for _, p := range []Palette{
		e.Distinct(3, DistinctOptions{From: "random", Avoid: []string{"#3470d0"}, Spread: 10}),
		e.Fluent(4, FluentOptions{Arc: 90, From: "random"}),
		e.RandomN(3, RandomOptions{Independent: true, Avoid: []string{"120"}}),
	} {
		r, err := Replay(p.Replay)
		if err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(hexes(r), hexes(p)) || r.Replay != p.Replay {
			t.Fatalf("replay %q gives %v, want %v", p.Replay, hexes(r), hexes(p))
		}
	}
	// negative bands and bare hex round-trip: bands as a..b, colours as #rrggbb, hues as numbers
	n := New(Options{Seed: "neg", Hues: [][2]float64{{-10, 20}, {180, 300.5}}, Avoid: []string{"D93025", "-20", "350.0"}})
	for _, p := range []Palette{
		n.RandomN(4, RandomOptions{Avoid: []string{"3470d0", "abc", "90"}, Spread: 12}),
		n.Distinct(3, DistinctOptions{From: "random", Avoid: []string{"#3470D0"}}),
	} {
		for _, part := range []string{"hues=-10..20,180..300.5", "eavoid=#d93025,-20,350"} {
			if !strings.Contains(p.Replay, part) {
				t.Fatalf("replay %q lacks %q", p.Replay, part)
			}
		}
		r, err := Replay(p.Replay)
		if err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(hexes(r), hexes(p)) || r.Replay != p.Replay {
			t.Fatalf("replay %q gives %v %q, want %v", p.Replay, hexes(r), r.Replay, hexes(p))
		}
	}
	if p := n.RandomN(1, RandomOptions{Avoid: []string{"3470d0", "abc", "90"}}); !strings.Contains(p.Replay, "avoid=#3470d0,#aabbcc,90;") {
		t.Fatalf("avoid written as %q", p.Replay)
	}
	if _, err := Replay("gulal0;random;seed=x"); err == nil {
		t.Fatal("a replay from another version was accepted")
	}
}

func TestPickedSeed(t *testing.T) {
	if s := New(Options{}).Seed(); !regexp.MustCompile(`^[0-9a-f]{8}$`).MatchString(s) {
		t.Fatalf("picked seed %q", s)
	}
}

func TestSwatchReads(t *testing.T) {
	g := New(Options{Seed: "api", Scheme: "light", Accent: "#0faaaa"})
	s := g.Swatch("#5b2a86", SwatchOptions{})
	for _, ink := range []string{s.Ink, s.Muted, s.Tonal, s.Complementary, s.Accent} {
		if g.Contrast(ink, s.Background) < 60 {
			t.Errorf("%s does not read on %s", ink, s.Background)
		}
	}
	h := s.Harmony()
	for _, list := range [][]string{h.Analogous, h.Complementary, h.Split, h.Triadic, h.Square} {
		for _, c := range list {
			if g.Contrast(c, s.Background) < 60 {
				t.Errorf("harmony %s does not read on %s", c, s.Background)
			}
		}
	}
	if ink := g.Swatch("#ffffff", SwatchOptions{Inks: []string{"#eeeeee", "#14213d", "#333333"}}).Ink; ink != "#14213d" {
		t.Errorf("best ink %s", ink)
	}
	if v := s.Vars(""); v["--gulal-accent"] != s.Accent || len(v) != 7 {
		t.Errorf("vars %v", v)
	}
}

func TestBadInput(t *testing.T) {
	if New(Options{Background: "nope"}) != nil {
		t.Error("New took a bad background")
	}
	g := New(Options{Seed: "x"})
	if g.Adapt("#12") != nil || g.Swatch("zz", SwatchOptions{}) != nil || g.Next([]string{"q"}) != nil {
		t.Error("a bad colour was taken")
	}
	if !math.IsNaN(g.Contrast("#fff", "nope")) {
		t.Error("contrast of a bad colour")
	}
	if !strings.HasPrefix(fmt.Sprint(Options{Accent: "x"}.Validate()), "gulal:") {
		t.Error("Validate")
	}
}

func TestConcurrent(t *testing.T) {
	g := New(Options{Seed: "c"})
	done := make(chan bool)
	for i := 0; i < 8; i++ {
		go func() {
			for j := 0; j < 50; j++ {
				g.Random().Hex()
			}
			done <- true
		}()
	}
	for i := 0; i < 8; i++ {
		<-done
	}
	if g.at != 8*50*3 {
		t.Fatalf("stream at %d", g.at)
	}
}
