// Package touchui translates between the Raw Panel TouchUI protocol (rwp.*)
// and the internal UI contract (gen.*). It is pure logic — no sockets, no
// goroutines — so every mapping is unit-testable. The Go side pre-digests all
// RWP state here; the C renderer never sees an HWCState.
package touchui

import (
	"fmt"
	"strings"

	rwp "github.com/SKAARHOJ/rawpanel-lib/ibeam_rawpanel"

	gen "github.com/SKAARHOJ/rawpanel-lib/touchui/gen"
)

// FeedResolver supplies the panel-local default video source for a VIDEO
// widget whose Options.Source is empty. ordinal is 1-based, counted over VIDEO
// widgets in config order. Returning an empty source leaves the widget as a
// placeholder (renders, but no video).
type FeedResolver func(ordinal int) (source string, width, height uint32, policy gen.VideoFeed_HiddenPolicy)

// ConfigToWidgetTree translates an accepted TouchUIConfig into the WidgetTree
// the UI renders. Video sources are resolved here (Options.Source, or the
// local default feed by ordinal) so the UI receives ready-to-open specs.
func ConfigToWidgetTree(cfg *rwp.TouchUIConfig, epoch uint32, resolve FeedResolver) *gen.WidgetTree {
	tree := &gen.WidgetTree{
		Epoch: epoch,
		Title: cfg.GetTitle(),
	}

	videoOrdinal := 0
	for _, page := range cfg.GetPages() {
		pageDef := &gen.PageDef{
			Id:       page.GetId(),
			Title:    page.GetTitle(),
			GridRows: page.GetGridRows(),
			GridCols: page.GetGridCols(),
		}
		for _, widget := range page.GetWidgets() {
			def := widgetToDef(widget)
			if widget.GetType() == rwp.TouchUIWidget_VIDEO {
				videoOrdinal++
				def.Feed = resolveFeed(widget, videoOrdinal, resolve)
			}
			pageDef.Widgets = append(pageDef.Widgets, def)
		}
		tree.Pages = append(tree.Pages, pageDef)
	}

	// Initial page: the config's wish if it names an existing page, else the
	// first page. Stored on the tree; runtime switches ride ActivePage frames.
	tree.ActivePage = defaultPage(cfg)

	// Panel-wide overlays: digest the config's options into the tree so a config
	// change or a renderer reconnect re-applies them. Absent = all off.
	tree.Options = globalOptions(cfg.GetOptions())
	return tree
}

// globalOptions digests rwp TouchUIConfig.Options into the UI contract. A nil
// input yields nil (the renderer treats an absent message as all-off).
func globalOptions(o *rwp.TouchUIGlobalOptions) *gen.GlobalOptions {
	if o == nil {
		return nil
	}
	return &gen.GlobalOptions{
		ShowDebugInfo: o.GetShowDebugInfo(),
		ShowTaps:      uint32(o.GetShowTaps()),
		ShowVideoFps:  o.GetShowVideoFPS(),
	}
}

func defaultPage(cfg *rwp.TouchUIConfig) uint32 {
	pages := cfg.GetPages()
	if len(pages) == 0 {
		return 0
	}
	want := cfg.GetActivePage()
	for _, p := range pages {
		if p.GetId() == want {
			return want
		}
	}
	return pages[0].GetId()
}

func widgetToDef(w *rwp.TouchUIWidget) *gen.WidgetDef {
	opts := w.GetOptions()
	return &gen.WidgetDef{
		HwcId:     w.GetHWCID(),
		Type:      gen.WidgetDef_Type(w.GetType()),
		Label:     w.GetLabel(),
		Row:       w.GetRow(),
		Col:       w.GetCol(),
		RowSpan:   w.GetRowSpan(),
		ColSpan:   w.GetColSpan(),
		X:         w.GetX(),
		Y:         w.GetY(),
		W:         w.GetW(),
		H:         w.GetH(),
		Min:       opts.GetMin(),
		Max:       opts.GetMax(),
		Step:      opts.GetStep(),
		Vertical:  opts.GetVertical(),
		AccentRgb: ColorToRGB(opts.GetColor()),
		NoTap:     opts.GetNoTapEvents(),
		Momentary: opts.GetMomentary(),
		FourWay:   opts.GetFourWay(),

		Choices:      joinChoices(opts.GetChoices()),
		Relative:     opts.GetRelative(),
		CenterReturn: opts.GetCenterReturn(),
		Params:       compressorParams(opts.GetParams()),
		EditKind:     gen.WidgetDef_EditKind(opts.GetEditKind()),
		EditMaxLen:   opts.GetEditMaxLen(),
		EventMask:    EffectiveEventMask(w),
	}
}

// joinChoices renders a ROLLER's options as the single '\n'-joined string the renderer wants
// (it feeds lv_roller_set_options directly). Truncation happens here rather than being an
// error, because ConfigToWidgetTree also runs on panel-local configs that never pass through
// Validate — a too-long list should degrade, not produce a broken widget.
func joinChoices(choices []string) string {
	if len(choices) == 0 {
		return ""
	}
	if len(choices) > MaxChoices {
		choices = choices[:MaxChoices]
	}
	out := make([]string, len(choices))
	for i, c := range choices {
		// A newline inside a label would split one option into two and shift every index
		// after it, desyncing the domain the client emits against.
		c = strings.NewReplacer("\r", " ", "\n", " ").Replace(c)
		if len(c) > MaxChoiceLen {
			c = c[:MaxChoiceLen]
		}
		out[i] = c
	}
	return strings.Join(out, "\n")
}

// compressorRoleDefaults is the natural-unit range each role gets when a client leaves
// Min/Max at 0/0. Resolved here, in Go, so the renderer never has to carry a role table and
// the mapping stays in one testable place.
var compressorRoleDefaults = map[rwp.TouchUICompressorParam_RoleE][2]int32{
	rwp.TouchUICompressorParam_THRESHOLD: {-60, 0},  // dB
	rwp.TouchUICompressorParam_RATIO:     {1, 20},   // n:1
	rwp.TouchUICompressorParam_KNEE:      {0, 24},   // dB
	rwp.TouchUICompressorParam_MAKEUP:    {0, 24},   // dB
	rwp.TouchUICompressorParam_ATTACK:    {0, 500},  // ms
	rwp.TouchUICompressorParam_RELEASE:   {0, 5000}, // ms
}

func compressorParams(params []*rwp.TouchUICompressorParam) []*gen.CompressorParam {
	if len(params) == 0 {
		return nil
	}
	if len(params) > MaxCompressorParams {
		params = params[:MaxCompressorParams]
	}
	out := make([]*gen.CompressorParam, 0, len(params))
	for _, p := range params {
		min, max := p.GetMin(), p.GetMax()
		if min == 0 && max == 0 {
			if def, ok := compressorRoleDefaults[p.GetRole()]; ok {
				min, max = def[0], def[1]
			}
		}
		label := p.GetLabel()
		if len(label) > MaxParamLabelLen {
			label = label[:MaxParamLabelLen]
		}
		out = append(out, &gen.CompressorParam{
			HwcId: p.GetHWCID(),
			Role:  gen.CompressorParam_Role(p.GetRole()),
			Min:   min,
			Max:   max,
			Label: label,
		})
	}
	return out
}

func resolveFeed(w *rwp.TouchUIWidget, ordinal int, resolve FeedResolver) *gen.VideoFeed {
	opts := w.GetOptions()
	feed := &gen.VideoFeed{
		FeedIndex:    uint32(ordinal),
		Source:       opts.GetSource(),
		Width:        opts.GetSourceWidth(),
		Height:       opts.GetSourceHeight(),
		HiddenPolicy: gen.VideoFeed_HiddenPolicy(opts.GetHiddenPolicy()),
	}
	if feed.Source == "" && resolve != nil {
		src, w, h, policy := resolve(ordinal)
		feed.Source = src
		if feed.Width == 0 {
			feed.Width = w
		}
		if feed.Height == 0 {
			feed.Height = h
		}
		feed.HiddenPolicy = policy
	}
	return feed
}

// StateToFrames digests one HWCState addressed to hwc into zero or more UI
// frames (WidgetState and/or WidgetGfx). Aspects absent from the input yield
// no output — the UI applies deltas only.
func StateToFrames(hwc uint32, state *rwp.HWCState, epoch uint32) []*gen.ServerMessage {
	var frames []*gen.ServerMessage

	ws := &gen.WidgetState{Epoch: epoch, HwcId: hwc}
	touched := false

	if m := state.GetHWCMode(); m != nil {
		ws.Mode = &gen.ModeState{
			State:        uint32(m.GetState()),
			BlinkPattern: m.GetBlinkPattern(),
			Output:       m.GetOutput(),
		}
		touched = true
	}
	if c := state.GetHWCColor(); c != nil {
		rgb := colorRGBFrom(c)
		ws.ColorRgb = &rgb
		touched = true
	}
	if t := state.GetHWCText(); t != nil {
		ws.Text = textStateFrom(t)
		touched = true
	}
	if x := state.GetHWCExtended(); x != nil {
		ws.Value = &gen.ValueState{
			Interpretation: uint32(x.GetInterpretation()),
			Value:          int32(x.GetValue()),
		}
		touched = true
	}
	if o := state.GetHWCOverlay(); o != nil {
		ws.Overlay = overlayStateFrom(o)
		touched = true
	}
	if touched {
		frames = append(frames, &gen.ServerMessage{Kind: &gen.ServerMessage_State{State: ws}})
	}

	if g := state.GetHWCGfx(); g != nil {
		if gfx, err := GfxToWidgetGfx(hwc, g, epoch); err == nil {
			frames = append(frames, &gen.ServerMessage{Kind: &gen.ServerMessage_Gfx{Gfx: gfx}})
		}
	}

	return frames
}

// EventToRWP maps a renderer WidgetEvent to the RawPanel HWCEvent it represents
// — the inverse direction of the state digest: a tap/turn in the UI becomes a
// panel event. Returns nil if the event carries no recognized kind. Shared so
// both the native touch host and Reactor's browser bridge inject events the same
// way.
func EventToRWP(ev *gen.WidgetEvent) *rwp.HWCEvent {
	out := &rwp.HWCEvent{HWCID: ev.GetHwcId()}
	switch kind := ev.GetKind().(type) {
	case *gen.WidgetEvent_Binary:
		out.Binary = &rwp.BinaryEvent{
			Pressed: kind.Binary.GetPressed(),
			Edge:    rwp.BinaryEvent_EdgeID(kind.Binary.GetEdge()), // 4-way / encoder edge (0=none)
		}
	case *gen.WidgetEvent_Pulsed:
		out.Pulsed = &rwp.PulsedEvent{Value: kind.Pulsed.GetValue()}
	case *gen.WidgetEvent_Absolute:
		out.Absolute = &rwp.AbsoluteEvent{Value: uint32(kind.Absolute.GetValue())}
	case *gen.WidgetEvent_Vector:
		// One renderer arm, two RWP messages: an XYPAD in relative mode reports movement
		// (SpeedVector), otherwise position (AbsoluteVector). The absolute form is unsigned
		// on the wire, so clamp — a negative here would wrap into a huge coordinate.
		vals := kind.Vector.GetValue()
		if kind.Vector.GetRelative() {
			out.SpeedVector = &rwp.SpeedVectorEvent{Value: append([]int32(nil), vals...)}
		} else {
			abs := make([]uint32, len(vals))
			for i, v := range vals {
				if v < 0 {
					v = 0
				}
				abs[i] = uint32(v)
			}
			out.AbsoluteVector = &rwp.AbsoluteVectorEvent{Value: abs}
		}
	case *gen.WidgetEvent_Text:
		out.Text = &rwp.TextEvent{Value: kind.Text.GetValue()}
	default:
		return nil
	}
	return out
}

func overlayStateFrom(o *rwp.HWCOverlay) *gen.OverlayState {
	out := &gen.OverlayState{}
	for _, b := range o.GetBoxes() {
		out.Boxes = append(out.Boxes, &gen.OverlayBox{
			Id:    b.GetId(),
			X:     b.GetX(),
			Y:     b.GetY(),
			W:     b.GetW(),
			H:     b.GetH(),
			Rgb:   ColorToRGB(b.GetColor()),
			Label: b.GetLabel(),
		})
	}
	return out
}

// textStateFrom pre-formats an HWCText for the renderer: the integer-value
// formats become ready strings, colors become RGB. Unsupported niceties
// (icons, scale bars, pair mode) degrade to plain text.
func textStateFrom(t *rwp.HWCText) *gen.TextState {
	out := &gen.TextState{
		Title:       t.GetTitle(),
		Line1:       t.GetTextline1(),
		Line2:       t.GetTextline2(),
		SolidHeader: t.GetSolidHeaderBar(),
		PixelRgb:    ColorToRGB(t.GetPixelColor()),
		BgRgb:       ColorToRGB(t.GetBackgroundColor()),
	}
	// FMT_HIDE (7) hides the value; FMT_ONELINE/TWOLINES (10/11) are pure
	// text-line modes. Any other format with no explicit text lines renders
	// the formatted value as line 1.
	fmtE := t.GetFormatting()
	if out.Line1 == "" && fmtE != rwp.HWCText_FMT_HIDE &&
		fmtE != rwp.HWCText_FMT_ONELINE && fmtE != rwp.HWCText_FMT_TWOLINES {
		out.Line1 = formatValue(t.GetIntegerValue(), fmtE)
	}
	return out
}

// formatValue renders HWCText.IntegerValue per HWCText.FormattingE.
func formatValue(v int32, formatting rwp.HWCText_FormattingE) string {
	switch formatting {
	case rwp.HWCText_FMT_FLOAT_2DEZ:
		return fmt.Sprintf("%.2f", float64(v)/100)
	case rwp.HWCText_FMT_PERCENTAGE:
		return fmt.Sprintf("%d%%", v)
	case rwp.HWCText_FMT_DB:
		return fmt.Sprintf("%ddB", v)
	case rwp.HWCText_FMT_FRAMES:
		return fmt.Sprintf("%df", v)
	case rwp.HWCText_FMT_ONEOVERX:
		return fmt.Sprintf("1/%d", v)
	case rwp.HWCText_FMT_KELVIN:
		return fmt.Sprintf("%dK", v)
	case rwp.HWCText_FMT_FLOAT_X_XXX:
		return fmt.Sprintf("%.3f", float64(v)/1000)
	case rwp.HWCText_FMT_FLOAT_XX_XX:
		return fmt.Sprintf("%.2f", float64(v)/100)
	case rwp.HWCText_FMT_FLOAT_XXX_X:
		return fmt.Sprintf("%.1f", float64(v)/10)
	default: // FMT_INTEGER and unknown formats
		return fmt.Sprintf("%d", v)
	}
}

// colorRGBFrom converts an HWCColor (same RGB-or-index shape as rwp.Color) to
// 0xRRGGBB, 0 = theme default.
func colorRGBFrom(c *rwp.HWCColor) uint32 {
	if c == nil {
		return 0
	}
	return ColorToRGB(&rwp.Color{ColorRGB: c.GetColorRGB(), ColorIndex: c.GetColorIndex()})
}

// SKAARHOJ color-index palette (2 bit/channel in the protocol), expanded to
// 8 bit/channel. Index order matches rwp.ColorIndex_Colors.
var indexRGB = []uint32{
	0xFFFFFF, // 0 Default
	0x000000, // 1 Off
	0xFFFFFF, // 2 White
	0xFFFF55, // 3 Warm White
	0xFF0000, // 4 Red
	0xFF5555, // 5 Rose
	0xFF55FF, // 6 Pink
	0x5500FF, // 7 Purple
	0xFF5500, // 8 Amber
	0xFFFF00, // 9 Yellow
	0x0000FF, // 10 Dark Blue
	0x0055FF, // 11 Blue
	0x55AAFF, // 12 Ice
	0x00FFFF, // 13 Cyan
	0x55FF00, // 14 Spring
	0x00FF00, // 15 Green
	0x00FF55, // 16 Mint
	0xAAAAAA, // 17 Light Gray
	0x555555, // 18 Dark Gray
}

// ColorToRGB converts an rwp.Color (RGB or indexed) to 0xRRGGBB. nil or the
// "default" index yield 0, which the UI treats as "theme default".
func ColorToRGB(c *rwp.Color) uint32 {
	if c == nil {
		return 0
	}
	if rgb := c.GetColorRGB(); rgb != nil {
		return (rgb.GetRed()&0xFF)<<16 | (rgb.GetGreen()&0xFF)<<8 | rgb.GetBlue()&0xFF
	}
	if idx := c.GetColorIndex(); idx != nil {
		i := int(idx.GetIndex())
		if i == 0 {
			return 0 // default => theme decides
		}
		if i < len(indexRGB) {
			return indexRGB[i]
		}
	}
	return 0
}
