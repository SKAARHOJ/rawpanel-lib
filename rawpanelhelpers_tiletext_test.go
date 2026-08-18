package rawpanellib

import (
	"image"
	"testing"

	rwp "github.com/SKAARHOJ/rawpanel-lib/ibeam_rawpanel"
	"google.golang.org/protobuf/proto"
)

// WriteDisplayTileNew fills in nil sub-messages on the struct it is handed, so every render
// has to start from its own copy or the second call in a comparison sees different input
// from the first.
func cloneText(txt *rwp.HWCText) *rwp.HWCText {
	return proto.Clone(txt).(*rwp.HWCText)
}

func asRGBA(t *testing.T, img image.Image) *image.RGBA {
	t.Helper()
	rgba, ok := img.(*image.RGBA)
	if !ok {
		t.Fatalf("expected *image.RGBA, got %T", img)
	}
	return rgba
}

// isInk reports whether the pixel carries content. The renderer defaults to white on black,
// so anything not black is something that got drawn.
func isInk(img *image.RGBA, x, y int) bool {
	c := img.RGBAAt(x, y)
	return c.R != 0 || c.G != 0 || c.B != 0
}

func inkCount(img *image.RGBA) int {
	n := 0
	bounds := img.Bounds()
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			if isInk(img, x, y) {
				n++
			}
		}
	}
	return n
}

// tileTextFixtures covers the branches of WriteDisplayTileNew that depend on the tile size,
// so the legacy-equivalence test below exercises them all rather than just the happy path.
func tileTextFixtures() map[string]*rwp.HWCText {
	return map[string]*rwp.HWCText{
		"oneline":   {Formatting: rwp.HWCText_FMT_ONELINE, Title: "One Line"},
		"twolines":  {Formatting: rwp.HWCText_FMT_TWOLINES, Textline1: "Upper", Textline2: "Lower"},
		"titleOnly": {Title: "Titleline"},
		"titleText": {Title: "Titleline", Textline1: "Heeeyl eqrgqrebqe4rgtb4er"},
		"solidBar":  {Title: "Titleline", Textline1: "Body", SolidHeaderBar: true},
		"inverted":  {Title: "Titleline", Textline1: "Body", Inverted: true},
		"value":     {Title: "TestVar", IntegerValue: 335},
		"iconLock":  {Title: "Locked", Textline1: "Body", StateIcon: rwp.HWCText_SI_LOCK},
		"iconFine":  {Title: "Fine", Textline1: "Body", StateIcon: rwp.HWCText_SI_FINE},
		"modifier":  {Title: "Mod", Textline1: "Body", ModifierIcon: rwp.HWCText_MI_CYCLE},
		"pair":      {Title: "Pair", Textline1: "A", Textline2: "B", PairMode: rwp.HWCText_PM_BOTH_MARKED, IntegerValue: 12, IntegerValue2: 34},
		"scale": {
			Title:        "Scaled",
			IntegerValue: 500,
			Scale:        &rwp.HWCText_ScaleM{ScaleType: rwp.HWCText_ScaleM_ST_STRENGTH, RangeLow: 0, RangeHigh: 1000},
		},
	}
}

func TestTextTileBorderPixels(t *testing.T) {
	cases := []struct {
		name              string
		border, borderPct int
		w, h              int
		want              int
	}{
		{"fixed border passes through", 5, 0, 410, 102, 5},
		{"no border at all", 0, 0, 410, 102, 0},
		{"pct of the short side", 0, 8, 410, 102, 8},
		{"pct at half resolution", 0, 8, 205, 51, 4},
		{"pct on the rounded-up half tile", 0, 8, 103, 51, 4},
		{"pct on a square tile", 0, 8, 102, 102, 8},
		{"pct wins over a fixed border", 5, 8, 410, 102, 8},
		{"pct truncates rather than rounds", 0, 8, 100, 49, 3}, // 49*8/100 = 3.92
		{"pct rounds down to nothing on a tiny tile", 0, 8, 10, 10, 0},
		{"pct is clamped at 255", 0, 100, 4000, 4000, 255},
		{"negative border is treated as none", -5, 0, 410, 102, 0},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := TextTileBorderPixels(c.border, c.borderPct, c.w, c.h); got != c.want {
				t.Errorf("TextTileBorderPixels(%d, %d, %d, %d) = %d, want %d", c.border, c.borderPct, c.w, c.h, got, c.want)
			}
		})
	}
}

// TestRenderHWCTextToTileLegacy is the backwards-compatibility gate. Every panel model that
// sets neither TextScale nor TextBorderPct - which today is all of them bar the Flywheel -
// has to render exactly as it did before those settings existed.
func TestRenderHWCTextToTileLegacy(t *testing.T) {
	// Tile sizes chosen to hit the size-dependent branches in WriteDisplayTileNew: the
	// height < 32 mini-tile path, the width == 256 wide-title-bar path, and height >= 48.
	sizes := [][2]int{{64, 32}, {128, 36}, {128, 48}, {256, 32}, {256, 64}, {96, 72}, {410, 102}}

	for name, fixture := range tileTextFixtures() {
		for _, size := range sizes {
			w, h := size[0], size[1]
			for _, shrink := range []int{0, 1, 2, 3} {
				for _, border := range []int{0, 5} {
					for _, scale := range []float64{0, 1} {
						want := renderTextTileImage(cloneText(fixture), w, h, shrink, border)
						got := RenderHWCTextToTile(cloneText(fixture), w, h, TileTextSettings{
							Shrink: shrink,
							Border: border,
							Scale:  scale,
						})

						wantPix := asRGBA(t, want).Pix
						gotPix := asRGBA(t, got).Pix
						if len(gotPix) != len(wantPix) {
							t.Fatalf("%s %dx%d shrink=%d border=%d scale=%v: size %d, want %d", name, w, h, shrink, border, scale, len(gotPix), len(wantPix))
						}
						for i := range wantPix {
							if gotPix[i] != wantPix[i] {
								t.Fatalf("%s %dx%d shrink=%d border=%d scale=%v: pixel data differs at byte %d", name, w, h, shrink, border, scale, i)
							}
						}
					}
				}
			}
		}
	}
}

// TestRenderHWCTextToTileMagnified pins the whole magnified path in one assertion: the
// reduced canvas size, the border recomputed at that size, and the nearest-neighbour
// sampling formula.
func TestRenderHWCTextToTileMagnified(t *testing.T) {
	cases := []struct {
		name           string
		w, h           int
		scale          float64
		borderPct      int
		wantSW, wantSH int
		wantBorder     int
	}{
		// The Flywheel's three curved tile sizes. Note the half-bar rounds UP to 103.
		{"full bar", 410, 102, 2, 8, 205, 51, 4},
		{"half bar rounds up", 205, 102, 2, 8, 103, 51, 4},
		{"quarter", 102, 102, 2, 8, 51, 51, 4},
		{"scale 3", 410, 102, 3, 8, 137, 34, 2},
	}

	fixture := &rwp.HWCText{Title: "Titleline", Textline1: "Heeeyl eqrgqrebqe4rgtb4er"}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := TextTileBorderPixels(0, c.borderPct, c.wantSW, c.wantSH); got != c.wantBorder {
				t.Fatalf("border at reduced size = %d, want %d", got, c.wantBorder)
			}

			small := asRGBA(t, renderTextTileImage(cloneText(fixture), c.wantSW, c.wantSH, 0, c.wantBorder))
			big := asRGBA(t, RenderHWCTextToTile(cloneText(fixture), c.w, c.h, TileTextSettings{
				BorderPct: c.borderPct,
				Scale:     c.scale,
			}))

			if big.Bounds().Dx() != c.w || big.Bounds().Dy() != c.h {
				t.Fatalf("magnified tile is %v, want %dx%d", big.Bounds(), c.w, c.h)
			}

			for y := 0; y < c.h; y++ {
				sy := y * c.wantSH / c.h
				for x := 0; x < c.w; x++ {
					sx := x * c.wantSW / c.w
					if big.RGBAAt(x, y) != small.RGBAAt(sx, sy) {
						t.Fatalf("pixel (%d,%d) does not match source pixel (%d,%d)", x, y, sx, sy)
					}
				}
			}
		})
	}
}

// TestRenderHWCTextToTileMagnifiedInset checks the border actually keeps content away from
// the edge, and that something was drawn at all - a blank tile would otherwise pass.
func TestRenderHWCTextToTileMagnifiedInset(t *testing.T) {
	const w, h = 410, 102

	fixture := &rwp.HWCText{Title: "Titleline", Textline1: "Heeeyl eqrgqrebqe4rgtb4er"}
	img := asRGBA(t, RenderHWCTextToTile(cloneText(fixture), w, h, TileTextSettings{BorderPct: 8, Scale: 2}))

	// The 205x51 render insets by 4 and so draws within x=[4,201), y=[4,47); doubled that is
	// x=[8,402), y=[8,94).
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			outside := x < 8 || x >= 402 || y < 8 || y >= 94
			if outside && isInk(img, x, y) {
				t.Fatalf("ink at (%d,%d), outside the 8px inset", x, y)
			}
		}
	}

	if n := inkCount(img); n == 0 {
		t.Fatal("nothing was drawn inside the inset")
	}
}

// TestRenderHWCTextToTileMagnifiedIsBigger is the property the whole change exists for: at
// scale 2 the same text covers materially more of the tile.
func TestRenderHWCTextToTileMagnifiedIsBigger(t *testing.T) {
	const w, h = 410, 102

	fixture := &rwp.HWCText{Title: "Titleline", Textline1: "Heeeyl eqrgqrebqe4rgtb4er"}

	plain := asRGBA(t, RenderHWCTextToTile(cloneText(fixture), w, h, TileTextSettings{}))
	scaled := asRGBA(t, RenderHWCTextToTile(cloneText(fixture), w, h, TileTextSettings{BorderPct: 8, Scale: 2}))

	plainInk, scaledInk := inkCount(plain), inkCount(scaled)
	if scaledInk <= plainInk*2 {
		t.Errorf("scaled ink %d should be well above twice the unscaled %d", scaledInk, plainInk)
	}
}

// TestRenderHWCTextToTileMagnifiedInverted guards the ordering: inversion is baked into the
// bits by the inner rasteriser, so it has to survive magnification untouched. The border
// ring, black when upright, must come out lit.
func TestRenderHWCTextToTileMagnifiedInverted(t *testing.T) {
	const w, h = 410, 102

	fixture := &rwp.HWCText{Title: "Titleline", Textline1: "Body", Inverted: true}
	img := asRGBA(t, RenderHWCTextToTile(cloneText(fixture), w, h, TileTextSettings{BorderPct: 8, Scale: 2}))

	for _, p := range [][2]int{{0, 0}, {w - 1, 0}, {0, h - 1}, {w - 1, h - 1}, {4, h / 2}} {
		if !isInk(img, p[0], p[1]) {
			t.Errorf("inverted tile should have a lit border, but (%d,%d) is black", p[0], p[1])
		}
	}
}

// TestRenderHWCTextToTileFractionalScale covers the bilinear branch. Nothing ships a
// fractional TextScale today, but the panel supports it and so must we.
func TestRenderHWCTextToTileFractionalScale(t *testing.T) {
	const w, h = 410, 102

	if isIntegerTextScale(1.5) {
		t.Fatal("1.5 should not count as an integer scale")
	}
	if !isIntegerTextScale(2.0) || !isIntegerTextScale(1.999) {
		t.Fatal("2.0 and 1.999 should both count as integer scales")
	}

	fixture := &rwp.HWCText{Title: "Titleline", Textline1: "Heeeyl eqrgqrebqe4rgtb4er"}
	smooth := asRGBA(t, RenderHWCTextToTile(cloneText(fixture), w, h, TileTextSettings{BorderPct: 8, Scale: 1.5}))

	if smooth.Bounds().Dx() != w || smooth.Bounds().Dy() != h {
		t.Fatalf("tile is %v, want %dx%d", smooth.Bounds(), w, h)
	}

	// Bilinear blends between the pixel and background colours, so there must be pixels that
	// are neither pure black nor pure white. Nearest-neighbour could never produce those.
	blended := 0
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			c := smooth.RGBAAt(x, y)
			if (c.R != 0 || c.G != 0 || c.B != 0) && (c.R != 0xFF || c.G != 0xFF || c.B != 0xFF) {
				blended++
			}
		}
	}
	if blended == 0 {
		t.Error("fractional scale produced no blended pixels, so it did not resample bilinearly")
	}
}

func TestRenderHWCTextToTileDegenerateSize(t *testing.T) {
	fixture := &rwp.HWCText{Title: "x"}
	for _, size := range [][2]int{{0, 10}, {10, 0}, {-1, -1}} {
		img := RenderHWCTextToTile(cloneText(fixture), size[0], size[1], TileTextSettings{Scale: 2})
		if img == nil {
			t.Fatalf("%dx%d returned nil", size[0], size[1])
		}
		if !img.Bounds().Empty() {
			t.Errorf("%dx%d should give an empty image, got %v", size[0], size[1], img.Bounds())
		}
	}
}

func TestTileTextSettingsFromDisp(t *testing.T) {
	if got := TileTextSettingsFromDisp(nil); got != (TileTextSettings{}) {
		t.Errorf("nil display should give the zero value, got %+v", got)
	}
}

// The tests below were moved here from hardware-manager-go's internal/display package, which
// used to carry its own copy of this renderer. They are retargeted at RenderHWCTextToTile
// with default settings, which is the same rasterisation path WriteDisplayTileNew gives.

func rowInkCount(img *image.RGBA, y int) int {
	n := 0
	for x := img.Bounds().Min.X; x < img.Bounds().Max.X; x++ {
		if isInk(img, x, y) {
			n++
		}
	}
	return n
}

// regionInkCount counts lit pixels inside the rectangle [x0,x1) x [y0,y1).
func regionInkCount(img *image.RGBA, x0, y0, x1, y1 int) int {
	n := 0
	for y := y0; y < y1; y++ {
		for x := x0; x < x1; x++ {
			if isInk(img, x, y) {
				n++
			}
		}
	}
	return n
}

func renderPlain(t *testing.T, txt *rwp.HWCText, w, h int) *image.RGBA {
	t.Helper()
	return asRGBA(t, RenderHWCTextToTile(cloneText(txt), w, h, TileTextSettings{}))
}

// TestRenderTextTwoLines: both lines must produce pixels, each in its own vertical half.
func TestRenderTextTwoLines(t *testing.T) {
	const w, h = 128, 36
	img := renderPlain(t, &rwp.HWCText{
		Formatting: rwp.HWCText_FMT_TWOLINES,
		Textline1:  "AAAA",
		Textline2:  "BBBB",
	}, w, h)

	if inkCount(img) == 0 {
		t.Fatal("two-line render produced an all-blank buffer")
	}

	top, bottom := 0, 0
	for y := 0; y < h; y++ {
		if y < h/2 {
			top += rowInkCount(img, y)
		} else {
			bottom += rowInkCount(img, y)
		}
	}
	if top == 0 {
		t.Error("line1 produced no pixels in the top half")
	}
	if bottom == 0 {
		t.Error("line2 produced no pixels in the bottom half")
	}
}

// TestRenderTextOneLine renders a centred single line and asserts both side margins stay
// blank.
func TestRenderTextOneLine(t *testing.T) {
	const w, h = 128, 36
	img := renderPlain(t, &rwp.HWCText{Formatting: rwp.HWCText_FMT_ONELINE, Title: "HELLO"}, w, h)

	if inkCount(img) == 0 {
		t.Fatal("one-line render produced an all-blank buffer")
	}
	if left := regionInkCount(img, 0, 0, 1, h); left != 0 {
		t.Errorf("one-line text not centred: leftmost column has %d lit pixels", left)
	}
	if right := regionInkCount(img, w-1, 0, w, h); right != 0 {
		t.Errorf("one-line text not centred: rightmost column has %d lit pixels", right)
	}
}

// TestRenderTextSolidHeaderBar: a solid bar fills the top rows, an underlined title only
// draws a thin rule.
func TestRenderTextSolidHeaderBar(t *testing.T) {
	const w, h = 128, 36

	solid := renderPlain(t, &rwp.HWCText{Formatting: rwp.HWCText_FMT_HIDE, Title: "TITLE", SolidHeaderBar: true}, w, h)
	if got := rowInkCount(solid, 0); got < w/2 {
		t.Errorf("solid header bar top row only %d/%d lit", got, w)
	}

	underlined := renderPlain(t, &rwp.HWCText{Formatting: rwp.HWCText_FMT_HIDE, Title: "TITLE"}, w, h)
	if got := rowInkCount(underlined, 0); got > w/4 {
		t.Errorf("underlined title top row too full: %d lit", got)
	}
	if inkCount(solid) <= inkCount(underlined) {
		t.Error("solid header bar should light more pixels than an underlined title")
	}
}

// TestRenderTextInverted: inversion swaps lit and unlit, so the two renders are exact
// complements of each other.
func TestRenderTextInverted(t *testing.T) {
	const w, h = 128, 36
	mk := func(inv bool) *rwp.HWCText {
		return &rwp.HWCText{Formatting: rwp.HWCText_FMT_TWOLINES, Textline1: "AB", Textline2: "CD", Inverted: inv}
	}

	normal := inkCount(renderPlain(t, mk(false), w, h))
	inverted := inkCount(renderPlain(t, mk(true), w, h))

	if normal == 0 {
		t.Fatal("normal render blank")
	}
	if inverted <= normal {
		t.Errorf("inverted (%d) should light more pixels than normal (%d)", inverted, normal)
	}
	if normal+inverted != w*h {
		t.Errorf("inverted is not the complement of normal: %d + %d != %d", normal, inverted, w*h)
	}
}

// TestRenderTextIntegerValue: a label plus a right-aligned value, with the value's pixels
// in the right portion of the tile.
func TestRenderTextIntegerValue(t *testing.T) {
	const w, h = 128, 36
	img := renderPlain(t, &rwp.HWCText{
		Formatting:   rwp.HWCText_FMT_INTEGER,
		Title:        "GAIN",
		Textline1:    "L",
		IntegerValue: 42,
	}, w, h)

	if inkCount(img) == 0 {
		t.Fatal("integer-value render blank")
	}
	if regionInkCount(img, w*3/4, 0, w, h) == 0 {
		t.Error("integer value did not render in the right portion of the tile")
	}
}

// TestRenderTextStateIcons: each state icon lights pixels in its corner, over and above the
// same tile without it.
func TestRenderTextStateIcons(t *testing.T) {
	const w, h = 128, 48

	plain := renderPlain(t, &rwp.HWCText{Formatting: rwp.HWCText_FMT_HIDE, Title: "T"}, w, h)

	lock := renderPlain(t, &rwp.HWCText{Formatting: rwp.HWCText_FMT_HIDE, Title: "T", StateIcon: rwp.HWCText_SI_LOCK}, w, h)
	if regionInkCount(lock, w-8, 0, w, 12) <= regionInkCount(plain, w-8, 0, w, 12) {
		t.Error("SI_LOCK did not add pixels in the top-right of the header")
	}

	noacc := renderPlain(t, &rwp.HWCText{Formatting: rwp.HWCText_FMT_HIDE, Title: "T", StateIcon: rwp.HWCText_SI_NOACCESS}, w, h)
	if regionInkCount(noacc, w-8, h-8, w, h) == 0 {
		t.Error("SI_NOACCESS produced no pixels in the bottom-right corner")
	}
}

// TestRenderTextModifierIcon: a modifier icon lights pixels in the top-right.
func TestRenderTextModifierIcon(t *testing.T) {
	const w, h = 128, 48

	plain := renderPlain(t, &rwp.HWCText{Formatting: rwp.HWCText_FMT_HIDE, Title: "T"}, w, h)
	cycle := renderPlain(t, &rwp.HWCText{Formatting: rwp.HWCText_FMT_HIDE, Title: "T", ModifierIcon: rwp.HWCText_MI_CYCLE}, w, h)

	if regionInkCount(cycle, w-8, 0, w, 24) <= regionInkCount(plain, w-8, 0, w, 24) {
		t.Error("MI_CYCLE did not add pixels in the top-right")
	}
}

// TestRenderTextScaleStrength: an ST_STRENGTH bar fills the bottom rows from the left in
// proportion to the value, so a half value must not reach the right half.
func TestRenderTextScaleStrength(t *testing.T) {
	const w, h = 128, 48
	img := renderPlain(t, &rwp.HWCText{
		Formatting:   rwp.HWCText_FMT_HIDE,
		IntegerValue: 50,
		Scale: &rwp.HWCText_ScaleM{
			ScaleType: rwp.HWCText_ScaleM_ST_STRENGTH,
			RangeLow:  0, RangeHigh: 100,
			LimitLow: 0, LimitHigh: 100,
		},
	}, w, h)

	// Only the fill rows: the base line at h-1 spans the full width and would mask this.
	left := regionInkCount(img, 0, h-3, w/2, h-1)
	right := regionInkCount(img, w/2, h-3, w, h-1)
	if left == 0 {
		t.Error("ST_STRENGTH bar lit no pixels in the left (filled) half")
	}
	if right != 0 {
		t.Errorf("ST_STRENGTH bar bled into the right half: left=%d right=%d", left, right)
	}
}
