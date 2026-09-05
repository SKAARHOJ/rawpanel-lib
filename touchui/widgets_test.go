package touchui

import (
	"strconv"
	"strings"
	"testing"

	helpers "github.com/SKAARHOJ/rawpanel-lib"
	rwp "github.com/SKAARHOJ/rawpanel-lib/ibeam_rawpanel"

	gen "github.com/SKAARHOJ/rawpanel-lib/touchui/gen"
)

// EditKind enables and EventMask narrows — a client should never have to set both.
func TestEffectiveEventMask(t *testing.T) {
	label := func(mutate func(*rwp.TouchUIWidget)) *rwp.TouchUIWidget {
		w := &rwp.TouchUIWidget{
			HWCID:   100,
			Type:    rwp.TouchUIWidget_LABEL,
			Options: &rwp.TouchUIWidgetOptions{},
		}
		if mutate != nil {
			mutate(w)
		}
		return w
	}

	cases := []struct {
		name string
		give *rwp.TouchUIWidget
		want uint32
	}{
		{
			"plain label taps only",
			label(nil),
			helpers.TouchUIEventBinary,
		},
		{
			"no-tap label is silent",
			label(func(w *rwp.TouchUIWidget) { w.Options.NoTapEvents = true }),
			0,
		},
		{
			// EditKind alone must turn Text on: this is the whole "the Option enables" rule.
			"editable label taps and commits",
			label(func(w *rwp.TouchUIWidget) { w.Options.EditKind = rwp.TouchUIWidgetOptions_TEXT }),
			helpers.TouchUIEventBinary | helpers.TouchUIEventText,
		},
		{
			// ...and EventMask can take the tap away without disabling the edit.
			"editable label narrowed to commits only",
			label(func(w *rwp.TouchUIWidget) {
				w.Options.EditKind = rwp.TouchUIWidgetOptions_TEXT
				w.EventMask = helpers.TouchUIEventText
			}),
			helpers.TouchUIEventText,
		},
		{
			// A mask can only remove: it must never talk the panel into an ability
			// the type does not have.
			"override cannot add an ability",
			label(func(w *rwp.TouchUIWidget) { w.EventMask = helpers.TouchUIEventAbsolute }),
			0,
		},
		{
			"xypad reports a vector plus touch",
			&rwp.TouchUIWidget{HWCID: 101, Type: rwp.TouchUIWidget_XYPAD},
			helpers.TouchUIEventVector | helpers.TouchUIEventBinary,
		},
		{
			"dropdown reports an index",
			&rwp.TouchUIWidget{HWCID: 102, Type: rwp.TouchUIWidget_DROPDOWN},
			helpers.TouchUIEventAbsolute,
		},
		{
			// The container is passive; its members emit under their own ids.
			"compressor container is silent",
			&rwp.TouchUIWidget{HWCID: 103, Type: rwp.TouchUIWidget_COMPRESSOR},
			0,
		},
		{
			// A video region is pointed at, so it reports WHERE — same as an XYPAD.
			"video reports a vector plus tap",
			&rwp.TouchUIWidget{HWCID: 104, Type: rwp.TouchUIWidget_VIDEO},
			helpers.TouchUIEventVector | helpers.TouchUIEventBinary,
		},
		{
			// NoTapEvents keeps its original meaning — it suppresses the press/release pair
			// — and must not take the position stream with it, or a widget that wants
			// position without taps would have no way to ask for it.
			"no-tap video still reports position",
			&rwp.TouchUIWidget{HWCID: 105, Type: rwp.TouchUIWidget_VIDEO,
				Options: &rwp.TouchUIWidgetOptions{NoTapEvents: true}},
			helpers.TouchUIEventVector,
		},
		{
			"video narrowed to taps only",
			&rwp.TouchUIWidget{HWCID: 106, Type: rwp.TouchUIWidget_VIDEO,
				EventMask: helpers.TouchUIEventBinary},
			helpers.TouchUIEventBinary,
		},
		{
			// An IMAGE is not a video: it stays a plain tap target and NoTapEvents still
			// silences it completely.
			"no-tap image is silent",
			&rwp.TouchUIWidget{HWCID: 107, Type: rwp.TouchUIWidget_IMAGE,
				Options: &rwp.TouchUIWidgetOptions{NoTapEvents: true}},
			0,
		},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			if got := EffectiveEventMask(tt.give); got != tt.want {
				t.Errorf("EffectiveEventMask = %d, want %d", got, tt.want)
			}
		})
	}
}

// The renderer feeds choices straight into lv_roller_set_options, so they must
// arrive newline-joined, capped, and free of embedded newlines (which would split
// one option into two and shift every index after it).
func TestJoinChoices(t *testing.T) {
	if got := joinChoices([]string{"1080p50", "2160p30"}); got != "1080p50\n2160p30" {
		t.Errorf("joined = %q", got)
	}
	if got := joinChoices([]string{"a\nb"}); got != "a b" {
		t.Errorf("embedded newline survived: %q", got)
	}
	if got := joinChoices(strings.Split(strings.Repeat("x,", MaxChoices+5), ",")); strings.Count(got, "\n") >= MaxChoices+5 {
		t.Errorf("choice list was not capped: %d entries", strings.Count(got, "\n")+1)
	}
	long := strings.Repeat("y", MaxChoiceLen+10)
	if got := joinChoices([]string{long}); len(got) != MaxChoiceLen {
		t.Errorf("long choice not truncated: %d bytes", len(got))
	}
}

// 0/0 means "panel default for the role", and resolving it in Go keeps the role
// table out of the renderer.
func TestCompressorRoleDefaults(t *testing.T) {
	out := compressorParams([]*rwp.TouchUICompressorParam{
		{HWCID: 401, Role: rwp.TouchUICompressorParam_THRESHOLD},
		{HWCID: 402, Role: rwp.TouchUICompressorParam_RATIO, Min: 2, Max: 8},
	})
	if len(out) != 2 {
		t.Fatalf("got %d params", len(out))
	}
	if out[0].GetMin() != -60 || out[0].GetMax() != 0 {
		t.Errorf("threshold default = %d..%d, want -60..0", out[0].GetMin(), out[0].GetMax())
	}
	if out[1].GetMin() != 2 || out[1].GetMax() != 8 {
		t.Errorf("explicit range was overwritten: %d..%d", out[1].GetMin(), out[1].GetMax())
	}
}

// An XYPAD's two axes must travel together, and the mode decides which rwp
// message carries them.
func TestEventToRWPVectorAndText(t *testing.T) {
	abs := EventToRWP(&gen.WidgetEvent{
		HwcId: 100,
		Kind:  &gen.WidgetEvent_Vector{Vector: &gen.VectorEv{Value: []int32{250, 750}}},
	})
	if abs.GetAbsoluteVector() == nil || abs.GetSpeedVector() != nil {
		t.Fatalf("absolute mode produced %+v", abs)
	}
	if v := abs.GetAbsoluteVector().GetValue(); len(v) != 2 || v[0] != 250 || v[1] != 750 {
		t.Errorf("absolute vector = %v", v)
	}

	rel := EventToRWP(&gen.WidgetEvent{
		HwcId: 100,
		Kind:  &gen.WidgetEvent_Vector{Vector: &gen.VectorEv{Value: []int32{-3, 4}, Relative: true}},
	})
	if rel.GetSpeedVector() == nil || rel.GetAbsoluteVector() != nil {
		t.Fatalf("relative mode produced %+v", rel)
	}
	if v := rel.GetSpeedVector().GetValue(); len(v) != 2 || v[0] != -3 || v[1] != 4 {
		t.Errorf("speed vector = %v", v)
	}

	// AbsoluteVector is unsigned on the wire, so a stray negative must clamp rather
	// than wrap into a coordinate near 4 billion.
	clamped := EventToRWP(&gen.WidgetEvent{
		HwcId: 100,
		Kind:  &gen.WidgetEvent_Vector{Vector: &gen.VectorEv{Value: []int32{-5, 10}}},
	})
	if v := clamped.GetAbsoluteVector().GetValue(); v[0] != 0 {
		t.Errorf("negative absolute axis = %d, want 0", v[0])
	}

	txt := EventToRWP(&gen.WidgetEvent{
		HwcId: 100,
		Kind:  &gen.WidgetEvent_Text{Text: &gen.TextEv{Value: "CAM 4"}},
	})
	if txt.GetText().GetValue() != "CAM 4" {
		t.Errorf("text event = %+v", txt.GetText())
	}
}

// The new option fields must actually reach the renderer — widgetToDef silently
// dropped every one of them before this change.
func TestWidgetOptionsReachTheRenderer(t *testing.T) {
	tree := ConfigToWidgetTree(&rwp.TouchUIConfig{
		Pages: []*rwp.TouchUIPage{{
			Id: 1,
			Widgets: []*rwp.TouchUIWidget{
				{HWCID: 100, Type: rwp.TouchUIWidget_DROPDOWN,
					Options: &rwp.TouchUIWidgetOptions{Choices: []string{"a", "b"}}},
				{HWCID: 101, Type: rwp.TouchUIWidget_XYPAD,
					Options: &rwp.TouchUIWidgetOptions{Relative: true}},
				{HWCID: 102, Type: rwp.TouchUIWidget_LABEL,
					Options: &rwp.TouchUIWidgetOptions{
						EditKind: rwp.TouchUIWidgetOptions_PASSWORD, EditMaxLen: 32}},
			},
		}},
	}, 1, nil)

	defs := tree.GetPages()[0].GetWidgets()
	if defs[0].GetChoices() != "a\nb" {
		t.Errorf("choices = %q", defs[0].GetChoices())
	}
	if !defs[1].GetRelative() {
		t.Error("relative was dropped")
	}
	if defs[2].GetEditKind() != gen.WidgetDef_EDIT_PASSWORD || defs[2].GetEditMaxLen() != 32 {
		t.Errorf("edit options = %v/%d", defs[2].GetEditKind(), defs[2].GetEditMaxLen())
	}
	if defs[2].GetEventMask()&helpers.TouchUIEventText == 0 {
		t.Error("editable label did not advertise Text in its event mask")
	}
}

// joinDomain's whole job is that the two lists stay index-parallel: entry N of choices must
// describe entry N of values no matter where either was cut, because a pick reports one and
// is indexed by the other.
func TestJoinDomain(t *testing.T) {
	c, v := joinDomain([]string{"Black", "Input 1"}, []string{"0", "1"})
	if c != "Black\nInput 1" || v != "0\n1" {
		t.Errorf("plain join = %q / %q", c, v)
	}

	// No values at all, or a short list: index-only rather than reporting some picks and
	// not others.
	if _, v := joinDomain([]string{"a", "b"}, nil); v != "" {
		t.Errorf("valueless domain should join to no values, got %q", v)
	}
	if _, v := joinDomain([]string{"a", "b", "c"}, []string{"1", "2"}); v != "" {
		t.Errorf("short value list should join to no values, got %q", v)
	}

	// The choices overflow first: long labels against short values. Both lists must lose
	// the same entries.
	long := make([]string, MaxChoices)
	vals := make([]string, MaxChoices)
	for i := range long {
		long[i] = strings.Repeat("x", MaxChoiceLen)
		vals[i] = strconv.Itoa(i)
	}
	c, v = joinDomain(long, vals)
	if got, want := len(strings.Split(c, "\n")), len(strings.Split(v, "\n")); got != want {
		t.Errorf("choices/values desynced when choices overflowed: %d vs %d entries", got, want)
	}
	if len(c) > MaxChoicesJoined {
		t.Errorf("joined choices %d bytes exceeds %d", len(c), MaxChoicesJoined)
	}

	// And the other way round: short labels against long values, so the VALUES hit the
	// budget first. A naive implementation that only measures choices desyncs here.
	shortC := make([]string, MaxChoices)
	longV := make([]string, MaxChoices)
	for i := range shortC {
		shortC[i] = "x"
		longV[i] = strings.Repeat("9", MaxChoiceLen)
	}
	c, v = joinDomain(shortC, longV)
	if got, want := len(strings.Split(c, "\n")), len(strings.Split(v, "\n")); got != want {
		t.Errorf("choices/values desynced when values overflowed: %d vs %d entries", got, want)
	}
	if len(v) > MaxChoicesJoined {
		t.Errorf("joined values %d bytes exceeds %d", len(v), MaxChoicesJoined)
	}

	// A newline inside a value would split it in two exactly as it would a label.
	if _, v := joinDomain([]string{"a"}, []string{"1\n2"}); v != "1 2" {
		t.Errorf("newline in a value not neutralized: %q", v)
	}
}
