package touchui

import (
	"strings"
	"testing"

	rwp "github.com/SKAARHOJ/rawpanel-lib/ibeam_rawpanel"

	gen "github.com/SKAARHOJ/rawpanel-lib/touchui/gen"
)

func testConfig() *rwp.TouchUIConfig {
	return &rwp.TouchUIConfig{
		Title:      "T",
		ActivePage: 2,
		Pages: []*rwp.TouchUIPage{
			{
				Id: 1, Title: "Main", GridRows: 2, GridCols: 3,
				Widgets: []*rwp.TouchUIWidget{
					{HWCID: 201, Type: rwp.TouchUIWidget_BUTTON, Label: "Cut", Row: 1, Col: 1,
						Options: &rwp.TouchUIWidgetOptions{Color: &rwp.Color{ColorIndex: &rwp.ColorIndex{Index: rwp.ColorIndex_RED}}}},
					{HWCID: 202, Type: rwp.TouchUIWidget_VIDEO, Label: "Cam A", Row: 1, Col: 2, RowSpan: 2},
				},
			},
			{
				Id: 2, Title: "Aux",
				Widgets: []*rwp.TouchUIWidget{
					{HWCID: 203, Type: rwp.TouchUIWidget_VIDEO, Label: "Cam B", X: 0, Y: 0, W: 640, H: 400,
						Options: &rwp.TouchUIWidgetOptions{Source: "mjpg:/dev/video9", HiddenPolicy: rwp.TouchUIWidgetOptions_DISCONNECT}},
				},
			},
		},
	}
}

func testResolver(ordinal int) (string, uint32, uint32, gen.VideoFeed_HiddenPolicy) {
	if ordinal == 1 {
		return "/dev/video0", 640, 480, gen.VideoFeed_KEEP_DECODING
	}
	return "", 0, 0, gen.VideoFeed_KEEP_DECODING
}

func TestConfigToWidgetTree(t *testing.T) {
	tree := ConfigToWidgetTree(testConfig(), 7, testResolver)

	if tree.Epoch != 7 || tree.Title != "T" || len(tree.Pages) != 2 {
		t.Fatalf("tree basics wrong: %+v", tree)
	}
	if tree.ActivePage != 2 {
		t.Errorf("ActivePage should honor the config wish: %d", tree.ActivePage)
	}

	button := tree.Pages[0].Widgets[0]
	if button.AccentRgb != 0xFF0000 {
		t.Errorf("RED index should expand to 0xFF0000, got %06x", button.AccentRgb)
	}

	// VIDEO ordinal 1 (empty Source) resolves to the local default feed.
	camA := tree.Pages[0].Widgets[1]
	if camA.Feed.GetFeedIndex() != 1 || camA.Feed.GetSource() != "/dev/video0" || camA.Feed.GetWidth() != 640 {
		t.Errorf("cam A feed not resolved from local table: %+v", camA.Feed)
	}
	// VIDEO ordinal 2 keeps its explicit source and policy.
	camB := tree.Pages[1].Widgets[0]
	if camB.Feed.GetFeedIndex() != 2 || camB.Feed.GetSource() != "mjpg:/dev/video9" ||
		camB.Feed.GetHiddenPolicy() != gen.VideoFeed_DISCONNECT {
		t.Errorf("cam B feed mangled: %+v", camB.Feed)
	}
}

func TestEncoderAndFourWayMapping(t *testing.T) {
	cfg := &rwp.TouchUIConfig{
		Pages: []*rwp.TouchUIPage{{
			Id: 1, Title: "M", GridRows: 1, GridCols: 2,
			Widgets: []*rwp.TouchUIWidget{
				{HWCID: 301, Type: rwp.TouchUIWidget_BUTTON, Label: "Nav", Row: 1, Col: 1,
					Options: &rwp.TouchUIWidgetOptions{FourWay: true}},
				{HWCID: 302, Type: rwp.TouchUIWidget_ENCODER, Label: "Sel", Row: 1, Col: 2},
			},
		}},
	}
	tree := ConfigToWidgetTree(cfg, 1, nil)

	nav := tree.Pages[0].Widgets[0]
	if nav.GetType() != gen.WidgetDef_BUTTON || !nav.GetFourWay() {
		t.Errorf("4-way button not mapped: type=%v fourWay=%v", nav.GetType(), nav.GetFourWay())
	}
	enc := tree.Pages[0].Widgets[1]
	if enc.GetType() != gen.WidgetDef_ENCODER {
		t.Errorf("encoder type not mapped: got %v", enc.GetType())
	}
}

func TestKnobTickCount(t *testing.T) {
	knob := func(o *rwp.TouchUIWidgetOptions) *rwp.TouchUIWidget {
		return &rwp.TouchUIWidget{HWCID: 401, Type: rwp.TouchUIWidget_KNOB, Options: o}
	}
	cases := []struct {
		name   string
		widget *rwp.TouchUIWidget
		want   uint32
	}{
		{"explicit wins over the range", knob(&rwp.TouchUIWidgetOptions{KnobTicks: 7, Min: 0, Max: 100, Step: 1}), 7},
		{"explicit is clamped", knob(&rwp.TouchUIWidgetOptions{KnobTicks: 500}), MaxKnobTicks},
		{"one tick per value", knob(&rwp.TouchUIWidgetOptions{Min: 0, Max: 10, Step: 1}), 11},
		{"step over the implicit 0..1000", knob(&rwp.TouchUIWidgetOptions{Step: 100}), 11},
		{"a dense range is capped", knob(&rwp.TouchUIWidgetOptions{Min: 0, Max: 1000, Step: 1}), MaxKnobTicks},
		{"a step wider than the range still draws two", knob(&rwp.TouchUIWidgetOptions{Min: 0, Max: 5, Step: 50}), MinKnobTicks},
		{"negative ranges count normally", knob(&rwp.TouchUIWidgetOptions{Min: -6, Max: 6, Step: 3}), 5},
		{"no step is left to the renderer", knob(&rwp.TouchUIWidgetOptions{Min: 0, Max: 100}), 0},
		{"an inverted range is left to the renderer", knob(&rwp.TouchUIWidgetOptions{Min: 10, Max: 5, Step: 1}), 0},
		{"no options at all", knob(nil), 0},
		{"non-knobs never carry a ring", &rwp.TouchUIWidget{Type: rwp.TouchUIWidget_SLIDER, Options: &rwp.TouchUIWidgetOptions{Step: 10}}, 0},
	}
	for _, c := range cases {
		if got := KnobTickCount(c.widget); got != c.want {
			t.Errorf("%s: got %d, want %d", c.name, got, c.want)
		}
	}
}

func TestActivePageFallback(t *testing.T) {
	cfg := testConfig()
	cfg.ActivePage = 99 // not a page
	if got := ConfigToWidgetTree(cfg, 1, nil).ActivePage; got != 1 {
		t.Errorf("unknown ActivePage should fall back to the first page, got %d", got)
	}
}

func TestStateToFrames(t *testing.T) {
	state := &rwp.HWCState{
		HWCIDs:      []uint32{201},
		HWCMode:     &rwp.HWCMode{State: rwp.HWCMode_ON, BlinkPattern: 3},
		HWCColor:    &rwp.HWCColor{ColorIndex: &rwp.ColorIndex{Index: rwp.ColorIndex_CYAN}},
		HWCText:     &rwp.HWCText{IntegerValue: 250, Formatting: rwp.HWCText_FMT_PERCENTAGE, Title: "Lvl"},
		HWCExtended: &rwp.HWCExtended{Interpretation: rwp.HWCExtended_FADER, Value: 700},
		HWCOverlay:  &rwp.HWCOverlay{Boxes: []*rwp.HWCOverlay_Box{{Id: 1, X: 10, Y: 20, W: 30, H: 40, Label: "b"}}},
	}
	frames := StateToFrames(201, state, 5)
	if len(frames) != 1 {
		t.Fatalf("expected 1 combined WidgetState frame, got %d", len(frames))
	}
	ws := frames[0].GetState()
	if ws.GetEpoch() != 5 || ws.GetHwcId() != 201 {
		t.Errorf("frame addressing wrong: %+v", ws)
	}
	if ws.GetMode().GetState() != uint32(rwp.HWCMode_ON) || ws.GetMode().GetBlinkPattern() != 3 {
		t.Errorf("mode digest wrong: %+v", ws.GetMode())
	}
	if ws.ColorRgb == nil || *ws.ColorRgb != 0x00FFFF {
		t.Errorf("CYAN should digest to 0x00FFFF: %v", ws.ColorRgb)
	}
	if ws.GetText().GetLine1() != "250%" || ws.GetText().GetTitle() != "Lvl" {
		t.Errorf("text pre-format wrong: %+v", ws.GetText())
	}
	if ws.GetValue().GetValue() != 700 {
		t.Errorf("extended digest wrong: %+v", ws.GetValue())
	}
	if len(ws.GetOverlay().GetBoxes()) != 1 || ws.GetOverlay().GetBoxes()[0].GetLabel() != "b" {
		t.Errorf("overlay digest wrong: %+v", ws.GetOverlay())
	}

	// A gfx-only state yields only a WidgetGfx frame.
	gfxState := &rwp.HWCState{
		HWCIDs: []uint32{201},
		HWCGfx: &rwp.HWCGfx{ImageType: rwp.HWCGfx_MONO, W: 8, H: 2, ImageData: []byte{0xF0, 0x0F}},
	}
	frames = StateToFrames(201, gfxState, 5)
	if len(frames) != 1 || frames[0].GetGfx() == nil {
		t.Fatalf("expected 1 gfx frame: %v", frames)
	}
	g := frames[0].GetGfx()
	if g.GetW() != 8 || g.GetH() != 2 || len(g.GetRgb565()) != 8*2*2 {
		t.Errorf("gfx frame dims wrong: %dx%d, %d bytes", g.GetW(), g.GetH(), len(g.GetRgb565()))
	}
	// First pixel of 0xF0 row is set => white (0xFFFF LE); last of row 1 unset => 0.
	if g.GetRgb565()[0] != 0xFF || g.GetRgb565()[1] != 0xFF {
		t.Errorf("MONO set pixel should be white: % x", g.GetRgb565()[:2])
	}
	if g.GetRgb565()[14] != 0x00 || g.GetRgb565()[15] != 0x00 {
		t.Errorf("MONO unset pixel should be black: % x", g.GetRgb565()[14:16])
	}
}

func TestGfxScaling(t *testing.T) {
	// The budget is bytes, not a shape: 640x100 is 64000 pixels, inside the 65536
	// the 131072-byte cap buys, so it must now survive at full size. The old fixed
	// 320x200 box halved it for no reason.
	w, h := 640, 100
	data := make([]byte, w*h*2)
	g, err := GfxToWidgetGfx(1, &rwp.HWCGfx{ImageType: rwp.HWCGfx_RGB16bit, W: uint32(w), H: uint32(h), ImageData: data}, 1)
	if err != nil {
		t.Fatal(err)
	}
	if g.GetW() != 640 || g.GetH() != 100 {
		t.Errorf("in-budget gfx was rescaled: %dx%d, want 640x100", g.GetW(), g.GetH())
	}
	if len(g.GetRgb565()) != 640*100*2 {
		t.Errorf("payload size wrong: %d", len(g.GetRgb565()))
	}
}

// TestGfxBudgetFit pins the rule that replaced the 320x200 box: scale to the byte
// cap, keeping aspect. The 1280x400 row is the panel-sized background that drove
// the change — under the old box it arrived as 320x100, a sixteenth of the pixels
// for half the cap.
func TestGfxBudgetFit(t *testing.T) {
	for _, tc := range []struct{ w, h int }{
		{1280, 400}, {1920, 1080}, {800, 480}, {4000, 60}, {64, 32}, {1, 1},
	} {
		outW, outH := fitBudget(tc.w, tc.h, maxGfxPixels)

		if outW*outH > maxGfxPixels {
			t.Errorf("%dx%d -> %dx%d: %d pixels exceeds the %d-pixel cap",
				tc.w, tc.h, outW, outH, outW*outH, maxGfxPixels)
		}
		if outW > tc.w || outH > tc.h {
			t.Errorf("%dx%d -> %dx%d: upscaled", tc.w, tc.h, outW, outH)
		}
		if tc.w*tc.h <= maxGfxPixels && (outW != tc.w || outH != tc.h) {
			t.Errorf("%dx%d fits the budget but was rescaled to %dx%d", tc.w, tc.h, outW, outH)
		}
		// Aspect within a pixel of the source, and the budget actually spent: a
		// downscaled result should land close to the cap, not at half of it.
		if tc.w*tc.h > maxGfxPixels {
			want, got := float64(tc.w)/float64(tc.h), float64(outW)/float64(outH)
			if want/got > 1.02 || got/want > 1.02 {
				t.Errorf("%dx%d -> %dx%d: aspect drifted %.3f -> %.3f", tc.w, tc.h, outW, outH, want, got)
			}
			if outW*outH < maxGfxPixels*9/10 {
				t.Errorf("%dx%d -> %dx%d: only %d of %d pixels used",
					tc.w, tc.h, outW, outH, outW*outH, maxGfxPixels)
			}
		}
	}
}

// TestBoxDownscaleAverages: the reduction must average its source rect, not pick
// one pixel out of it. Nearest-neighbour on this input returns a source pixel
// (0x000000 or 0xFFFFFF); averaging returns the mid grey between them.
func TestBoxDownscaleAverages(t *testing.T) {
	pix := []uint32{0x000000, 0xFFFFFF, 0x000000, 0xFFFFFF} // 2x2 checker of columns
	got := boxDownscale(pix, 2, 2, 1, 1)
	if len(got) != 1 {
		t.Fatalf("want 1 pixel, got %d", len(got))
	}
	if got[0] != 0x7F7F7F {
		t.Errorf("box filter returned %06X, want 7F7F7F (a point sample would give 000000 or FFFFFF)", got[0])
	}
}

// TestPackRGB565Rounds: quantisation rounds to the nearest level. Truncation —
// what the old (r>>3) did — sends every value below 8 to zero, so a dark channel
// lost detail and the whole image skewed dark.
func TestPackRGB565Rounds(t *testing.T) {
	// 7/255 is 0.85 of a 5-bit level: rounds to 1, truncates to 0.
	if got := packRGB565(0x070000) >> 11 & 0x1F; got != 1 {
		t.Errorf("red 7 packed to level %d, want 1 (truncation would give 0)", got)
	}
	if got := packRGB565(0xFFFFFF); got != 0xFFFF {
		t.Errorf("white packed to %04X, want FFFF", got)
	}
	if got := packRGB565(0x000000); got != 0x0000 {
		t.Errorf("black packed to %04X, want 0000", got)
	}
}

// TestPackRGB565Roundtrip: every level the renderer can display must survive
// expand-then-requantise unchanged. Without this, rounding could shift a pixel
// that arrived already quantised — which is exactly what an RGB16bit HWCGfx is.
func TestPackRGB565Roundtrip(t *testing.T) {
	for bits := uint(5); bits <= 6; bits++ {
		for q := uint32(0); q <= 1<<bits-1; q++ {
			if got := quantiseChannel(int32(expandChannel(q, bits)), bits); got != q {
				t.Errorf("%d-bit level %d expanded to %d and came back as %d",
					bits, q, expandChannel(q, bits), got)
			}
		}
	}
}

// TestPageGfxDithers: a background gets error diffusion, a widget icon does not.
// A flat fill that lands between two 5-bit levels stipples in the background and
// stays solid in the widget.
func TestPageGfxDithers(t *testing.T) {
	const w, h = 16, 16
	pix := make([]uint32, w*h)
	for i := range pix {
		pix[i] = 0x0C0C0C // between 5-bit levels 1 and 2
	}

	distinct := func(b []byte) int {
		seen := map[uint16]bool{}
		for i := 0; i < len(b); i += 2 {
			seen[uint16(b[i])|uint16(b[i+1])<<8] = true
		}
		return len(seen)
	}

	if n := distinct(packBuffer(pix, w, h, false)); n != 1 {
		t.Errorf("undithered flat fill produced %d distinct values, want 1", n)
	}
	if n := distinct(packBuffer(pix, w, h, true)); n < 2 {
		t.Errorf("dithered flat fill produced %d distinct values, want at least 2", n)
	}
}

func TestValidateRejects(t *testing.T) {
	cases := map[string]func(*rwp.TouchUIConfig){
		"no pages":        func(c *rwp.TouchUIConfig) { c.Pages = nil },
		"dup page":        func(c *rwp.TouchUIConfig) { c.Pages[1].Id = 1 },
		"dup hwc":         func(c *rwp.TouchUIConfig) { c.Pages[1].Widgets[0].HWCID = 201 },
		"zero hwc":        func(c *rwp.TouchUIConfig) { c.Pages[0].Widgets[0].HWCID = 0 },
		"outside grid":    func(c *rwp.TouchUIConfig) { c.Pages[0].Widgets[0].Col = 9 },
		"bad active page": func(c *rwp.TouchUIConfig) { c.ActivePage = 42 },
		"long label":      func(c *rwp.TouchUIConfig) { c.Pages[0].Widgets[0].Label = string(make([]byte, 60)) },
		"too many widgets": func(c *rwp.TouchUIConfig) {
			for i := 0; i < 30; i++ {
				c.Pages[1].Widgets = append(c.Pages[1].Widgets,
					&rwp.TouchUIWidget{HWCID: uint32(300 + i), Type: rwp.TouchUIWidget_LABEL})
			}
		},

		"dropdown without choices": func(c *rwp.TouchUIConfig) {
			c.Pages[0].Widgets[0].Type = rwp.TouchUIWidget_DROPDOWN
		},
		"choice with a newline": func(c *rwp.TouchUIConfig) {
			c.Pages[0].Widgets[0].Type = rwp.TouchUIWidget_DROPDOWN
			c.Pages[0].Widgets[0].Options.Choices = []string{"a\nb"}
		},
		"choices that join past the panel's byte cap": func(c *rwp.TouchUIConfig) {
			c.Pages[0].Widgets[0].Type = rwp.TouchUIWidget_DROPDOWN
			// 16 x 23 bytes + 15 separators = 383, over MaxChoicesJoined but within both
			// MaxChoices and MaxChoiceLen — so only the joined check can catch it.
			ch := make([]string, 16)
			for i := range ch {
				ch[i] = strings.Repeat("x", MaxChoiceLen)
			}
			c.Pages[0].Widgets[0].Options.Choices = ch
		},
		"relative xypad that also center-returns": func(c *rwp.TouchUIConfig) {
			c.Pages[0].Widgets[0].Type = rwp.TouchUIWidget_XYPAD
			c.Pages[0].Widgets[0].Options.Relative = true
			c.Pages[0].Widgets[0].Options.CenterReturn = true
		},
		"compressor without params": func(c *rwp.TouchUIConfig) {
			c.Pages[0].Widgets[0].Type = rwp.TouchUIWidget_COMPRESSOR
		},
		"compressor with duplicate roles": func(c *rwp.TouchUIConfig) {
			c.Pages[0].Widgets[0].Type = rwp.TouchUIWidget_COMPRESSOR
			c.Pages[0].Widgets[0].Options.Params = []*rwp.TouchUICompressorParam{
				{HWCID: 401, Role: rwp.TouchUICompressorParam_THRESHOLD},
				{HWCID: 402, Role: rwp.TouchUICompressorParam_THRESHOLD},
			}
		},
		// The important one: a member id sharing the widget id space would make the
		// renderer's lookup and the core's state store ambiguous.
		"compressor member colliding with a widget": func(c *rwp.TouchUIConfig) {
			c.Pages[0].Widgets[0].Type = rwp.TouchUIWidget_COMPRESSOR
			c.Pages[0].Widgets[0].Options.Params = []*rwp.TouchUICompressorParam{
				{HWCID: 203, Role: rwp.TouchUICompressorParam_THRESHOLD}, // 203 is a VIDEO widget
			}
		},
		"compressor members colliding with each other": func(c *rwp.TouchUIConfig) {
			c.Pages[0].Widgets[0].Type = rwp.TouchUIWidget_COMPRESSOR
			c.Pages[0].Widgets[0].Options.Params = []*rwp.TouchUICompressorParam{
				{HWCID: 401, Role: rwp.TouchUICompressorParam_THRESHOLD},
				{HWCID: 401, Role: rwp.TouchUICompressorParam_RATIO},
			}
		},
		"editkind on a non-label": func(c *rwp.TouchUIConfig) {
			c.Pages[0].Widgets[0].Options.EditKind = rwp.TouchUIWidgetOptions_TEXT
		},
		"edit length beyond the cap": func(c *rwp.TouchUIConfig) {
			c.Pages[0].Widgets[0].Type = rwp.TouchUIWidget_LABEL
			c.Pages[0].Widgets[0].Options.EditKind = rwp.TouchUIWidgetOptions_TEXT
			c.Pages[0].Widgets[0].Options.EditMaxLen = MaxEditLen + 1
		},
	}
	for name, mutate := range cases {
		cfg := testConfig()
		mutate(cfg)
		if err := Validate(cfg); err == nil {
			t.Errorf("%s: expected a validation error", name)
		}
	}
	if err := Validate(testConfig()); err != nil {
		t.Errorf("pristine config should validate: %v", err)
	}
}

// The RGB16bit wire format packs (b<<11)|(g<<5)|r, so a colour must survive the trip to the
// renderer's RGB565 frame with red and blue where they started. Dark blue read as plain
// RGB565 comes back dark red, which is the failure this pins down.
func TestGfxRGB16ChannelOrder(t *testing.T) {
	darkBlue := uint16((17 & 0x1F) << 11) // b=17, g=0, r=0 in the wire's b-g-r order
	g, err := GfxToWidgetGfx(1, &rwp.HWCGfx{
		ImageType: rwp.HWCGfx_RGB16bit, W: 1, H: 1,
		ImageData: []byte{byte(darkBlue >> 8), byte(darkBlue)},
	}, 1)
	if err != nil {
		t.Fatalf("GfxToWidgetGfx: %v", err)
	}

	// Frame is little-endian RGB565 for LVGL: red in the top bits, blue in the bottom.
	out := uint16(g.GetRgb565()[0]) | uint16(g.GetRgb565()[1])<<8
	red := (out >> 11) & 0x1F
	blue := out & 0x1F
	if blue != 17 || red != 0 {
		t.Errorf("dark blue came back as r=%d b=%d, want r=0 b=17 (channels swapped)", red, blue)
	}
}

// joinChoices degrades instead of erroring, because ConfigToWidgetTree also runs on
// panel-local configs that never pass through Validate. What it must never do is hand the
// renderer a list whose surviving indices mean something other than what the client sent.
func TestJoinChoicesDropsWholeOptions(t *testing.T) {
	long := strings.Repeat("x", MaxChoiceLen)
	choices := make([]string, MaxChoices)
	for i := range choices {
		choices[i] = long
	}

	got := joinChoices(choices)

	if len(got) > MaxChoicesJoined {
		t.Fatalf("joined to %d bytes, over the %d cap", len(got), MaxChoicesJoined)
	}
	if strings.HasSuffix(got, "\n") {
		t.Error("trailing separator: nanopb would read a phantom empty final option")
	}
	// Every surviving option must be whole, and a prefix of the input — a half-written
	// label, or a dropped option from the middle, would shift the index domain.
	parts := strings.Split(got, "\n")
	for i, p := range parts {
		if p != long {
			t.Fatalf("option %d is %q, not the whole label it was cut from", i, p)
		}
	}
	if want := (MaxChoicesJoined + 1) / (MaxChoiceLen + 1); len(parts) != want {
		t.Errorf("kept %d options, want %d", len(parts), want)
	}
}

// The whole reactor -> renderer path for a domain: an HWCDomain on an outbound state has to
// come out the other side as a DomainState the renderer can hand straight to
// lv_dropdown_set_options, with the values still index-parallel to the labels.
func TestStateToFramesDomain(t *testing.T) {
	state := &rwp.HWCState{
		HWCIDs: []uint32{112},
		HWCDomain: &rwp.HWCDomain{
			Choices: []string{"Black", "Input 1", "Media Player 1"},
			Values:  []string{"0", "1", "3010"},
		},
		HWCExtended: &rwp.HWCExtended{Interpretation: rwp.HWCExtended_STEPS, Value: 2},
	}
	frames := StateToFrames(112, state, 9)
	if len(frames) != 1 {
		t.Fatalf("expected 1 WidgetState frame, got %d", len(frames))
	}
	d := frames[0].GetState().GetDomain()
	if d == nil {
		t.Fatal("no domain in the digested frame")
	}
	if d.GetChoices() != "Black\nInput 1\nMedia Player 1" {
		t.Errorf("choices = %q", d.GetChoices())
	}
	if d.GetValues() != "0\n1\n3010" {
		t.Errorf("values = %q", d.GetValues())
	}
	// The domain and the selection ride the same frame, so the renderer can install the list
	// before it reads the index against it.
	if frames[0].GetState().GetValue().GetValue() != 2 {
		t.Errorf("the selection should travel with its domain: %+v", frames[0].GetState().GetValue())
	}

	// A state with no domain must add none — the renderer applies deltas, and an empty
	// DomainState would read as "replace the list with nothing".
	plain := &rwp.HWCState{HWCIDs: []uint32{112}, HWCExtended: &rwp.HWCExtended{Value: 1}}
	if d := StateToFrames(112, plain, 9)[0].GetState().GetDomain(); d != nil {
		t.Errorf("a domainless state produced a domain: %+v", d)
	}
}

// EventToRWP has to carry a dropdown pick's value up alongside its index, in ONE HWCEvent —
// rwp.HWCEvent has plain fields rather than a oneof precisely so both arms fit.
func TestEventToRWPDomainValue(t *testing.T) {
	ev := &gen.WidgetEvent{
		HwcId: 112,
		Kind:  &gen.WidgetEvent_Absolute{Absolute: &gen.AbsoluteEv{Value: 2, DomainValue: "3010"}},
	}
	out := EventToRWP(ev)
	if out.GetAbsolute().GetValue() != 2 {
		t.Errorf("index lost: %+v", out.GetAbsolute())
	}
	if out.GetText().GetValue() != "3010" {
		t.Errorf("picked value lost: %+v", out.GetText())
	}

	// Without a domain value the event is exactly what it always was: no Text arm, so a
	// behavior bound the historical way sees no change at all.
	plain := &gen.WidgetEvent{HwcId: 112, Kind: &gen.WidgetEvent_Absolute{Absolute: &gen.AbsoluteEv{Value: 2}}}
	if out := EventToRWP(plain); out.GetText() != nil {
		t.Errorf("a plain pick grew a Text arm: %+v", out.GetText())
	}
}
