package rawpanellib

import (
	"testing"

	rwp "github.com/SKAARHOJ/rawpanel-lib/ibeam_rawpanel"
)

func scrollText(mode rwp.HWCText_TextScrollM_ScrollModeE, speed, dwell uint32) *rwp.HWCText {
	return &rwp.HWCText{
		Title:     "A very long title that cannot fit on a small display",
		Textline1: "Short",
		TextScroll: &rwp.HWCText_TextScrollM{
			ScrollMode:      mode,
			ScrollTitle:     true,
			ScrollTextline1: true,
			ScrollSpeed:     speed,
			PauseDwell:      dwell,
		},
	}
}

func TestTextScrollerOffAndClear(t *testing.T) {
	s := NewTextScroller()
	txt := &rwp.HWCText{Title: "A very long title that cannot fit on a small display"}
	if s.Set(1, txt) {
		t.Fatal("text without TextScroll must not scroll")
	}
	if got := s.Apply(1, txt, 64, 32, 0, 0); got != txt {
		t.Fatal("Apply without state must return the input unchanged")
	}
	if !s.Set(1, scrollText(rwp.HWCText_TextScrollM_SCROLL_LOOP, 3, 1)) || !s.Active(1) {
		t.Fatal("expected scroll to be active")
	}
	s.Clear(1)
	if s.Active(1) {
		t.Fatal("Clear did not remove state")
	}
}

func TestTextScrollerLoop(t *testing.T) {
	s := NewTextScroller()
	txt := scrollText(rwp.HWCText_TextScrollM_SCROLL_LOOP, 3, 1) // fast: 1 tick/char, short dwell: 8 ticks
	s.Set(1, txt)

	// Nothing moves before the first render has measured the text
	if ids := s.Tick(); len(ids) != 0 {
		t.Fatalf("tick before measure changed %v", ids)
	}
	if got := s.Apply(1, txt, 64, 32, 0, 0); got.Title != txt.Title {
		t.Fatal("offset 0 must render the full title")
	}
	ss := s.states[1]
	if ss.maxOffset <= 0 || ss.line1Max != 0 {
		t.Fatalf("unexpected measure: max=%d line1=%d", ss.maxOffset, ss.line1Max)
	}

	// Initial dwell (8 ticks, counted from the first render), then one char per tick
	for i := 0; i < 8; i++ {
		if ids := s.Tick(); len(ids) != 0 {
			t.Fatalf("moved during initial dwell at tick %d", i)
		}
	}
	if ids := s.Tick(); len(ids) != 1 || ids[0] != 1 {
		t.Fatalf("expected first advance, got %v", ids)
	}
	got := s.Apply(1, txt, 64, 32, 0, 0)
	if got.Title != txt.Title[1:] || got.Textline1 != "Short" {
		t.Fatalf("unexpected trim: %q / %q", got.Title, got.Textline1)
	}
	if txt.Title[0] != 'A' {
		t.Fatal("input was modified")
	}

	// Run to the end, then it must wrap back to 0
	for ss.charOffset < ss.maxOffset {
		s.Tick()
	}
	for i := 0; i < 8; i++ {
		s.Tick() // end dwell
	}
	s.Tick()
	if ss.charOffset != 0 {
		t.Fatalf("loop did not wrap, offset %d", ss.charOffset)
	}
}

func TestTextScrollerBounceAndResend(t *testing.T) {
	s := NewTextScroller()
	txt := scrollText(rwp.HWCText_TextScrollM_SCROLL_BOUNCE, 3, 1)
	s.Set(1, txt)
	s.Apply(1, txt, 64, 32, 0, 0)
	ss := s.states[1]
	for i := 0; i < 8+ss.maxOffset; i++ {
		s.Tick()
	}
	if ss.charOffset != ss.maxOffset || ss.direction != -1 {
		t.Fatalf("expected to reach end and turn: offset %d dir %d", ss.charOffset, ss.direction)
	}

	// Re-sending identical text keeps the animation
	s.Set(1, scrollText(rwp.HWCText_TextScrollM_SCROLL_BOUNCE, 3, 1))
	if s.states[1] != ss {
		t.Fatal("identical text restarted the animation")
	}
	// Different text restarts
	changed := scrollText(rwp.HWCText_TextScrollM_SCROLL_BOUNCE, 3, 1)
	changed.Title += "!"
	s.Set(1, changed)
	if s.states[1] == ss || s.states[1].charOffset != 0 {
		t.Fatal("new text did not restart the animation")
	}
}

func TestTextScrollerAdaptiveTrim(t *testing.T) {
	ss := &textScrollState{adaptive: true, maxOffset: 20, charOffset: 10, direction: 1}
	if got := ss.trim(10); got != 5 {
		t.Fatalf("adaptive half way: got %d want 5", got)
	}
	ss = &textScrollState{maxOffset: 20, charOffset: 15, direction: 1}
	if got := ss.trim(10); got != 10 {
		t.Fatalf("forward cap: got %d want 10", got)
	}
	ss.direction = -1
	if got := ss.trim(10); got != 5 {
		t.Fatalf("backward: got %d want 5", got)
	}
}
