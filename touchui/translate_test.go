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
	// 640x100 RGB16 must downscale to 320x50.
	w, h := 640, 100
	data := make([]byte, w*h*2)
	g, err := GfxToWidgetGfx(1, &rwp.HWCGfx{ImageType: rwp.HWCGfx_RGB16bit, W: uint32(w), H: uint32(h), ImageData: data}, 1)
	if err != nil {
		t.Fatal(err)
	}
	if g.GetW() != 320 || g.GetH() != 50 {
		t.Errorf("scaling wrong: %dx%d", g.GetW(), g.GetH())
	}
	if len(g.GetRgb565()) != 320*50*2 {
		t.Errorf("payload size wrong: %d", len(g.GetRgb565()))
	}
}

func TestValidateRejects(t *testing.T) {
	cases := map[string]func(*rwp.TouchUIConfig){
		"no pages":          func(c *rwp.TouchUIConfig) { c.Pages = nil },
		"dup page":          func(c *rwp.TouchUIConfig) { c.Pages[1].Id = 1 },
		"dup hwc":           func(c *rwp.TouchUIConfig) { c.Pages[1].Widgets[0].HWCID = 201 },
		"zero hwc":          func(c *rwp.TouchUIConfig) { c.Pages[0].Widgets[0].HWCID = 0 },
		"outside grid":      func(c *rwp.TouchUIConfig) { c.Pages[0].Widgets[0].Col = 9 },
		"bad active page":   func(c *rwp.TouchUIConfig) { c.ActivePage = 42 },
		"long label":        func(c *rwp.TouchUIConfig) { c.Pages[0].Widgets[0].Label = string(make([]byte, 60)) },
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
