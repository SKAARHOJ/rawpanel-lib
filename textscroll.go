package rawpanellib

import (
	"sync"
	"time"

	rwp "github.com/SKAARHOJ/rawpanel-lib/ibeam_rawpanel"
	"google.golang.org/protobuf/proto"
)

// TextScrollTickInterval is how often TextScroller.Tick should be called.
// Speeds and dwell times are expressed in ticks of this interval.
const TextScrollTickInterval = 100 * time.Millisecond

// TextScroller animates HWCText.TextScroll for the displays of a panel.
// It is safe for concurrent use.
//
// Typical use in a panel implementation:
//
//   - When HWCText arrives for a display: scroller.Set(hwcID, text)
//   - When HWCGfx arrives or the display is cleared: scroller.Clear(hwcID)
//   - Every TextScrollTickInterval: for each id in scroller.Tick(), mark the display for redraw
//   - When rendering text: text = scroller.Apply(hwcID, text, w, h, shrink, border) before WriteDisplayTileNew
//
// Panels that implement this should announce RawPanelSupport.TextScroll.
type TextScroller struct {
	mu     sync.Mutex
	states map[uint32]*textScrollState
}

// textScrollState is the animation state of one display.
// All scrolled fields share charOffset so they advance together. Per-field max offsets
// let shorter fields stop trimming while longer ones continue.
type textScrollState struct {
	text *rwp.HWCText // Text the state was set up for (to detect re-sends of the same text)

	// Settings from TextScroll
	mode     rwp.HWCText_TextScrollM_ScrollModeE
	speed    uint32
	dwell    uint32
	adaptive bool

	// Measured lazily on first Apply (or when the render size changes)
	measured  bool
	dims      [4]int // w, h, shrink, border used for measuring
	titleMax  int    // Offset at which the title fits (0 = fits already or not scrolled)
	line1Max  int
	line2Max  int
	maxOffset int // max(titleMax, line1Max, line2Max)

	// Animation
	charOffset  int // Current shared character offset (0 = original position)
	direction   int // 1 = forward, -1 = backward (bounce)
	paused      int // Ticks left to dwell at an endpoint
	tickCounter int // Ticks since last advance (speed control)
}

// NewTextScroller returns an empty TextScroller.
func NewTextScroller() *TextScroller {
	return &TextScroller{states: make(map[uint32]*textScrollState)}
}

// Set registers new text for a display. If the text has no active TextScroll, any
// scroll state is removed. Re-sending identical text keeps the running animation.
// Returns true if the display has scroll enabled.
func (s *TextScroller) Set(hwcID uint32, text *rwp.HWCText) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	if text == nil || text.TextScroll == nil || text.TextScroll.ScrollMode == rwp.HWCText_TextScrollM_SCROLL_OFF ||
		!(text.TextScroll.ScrollTitle || text.TextScroll.ScrollTextline1 || text.TextScroll.ScrollTextline2) {
		delete(s.states, hwcID)
		return false
	}

	if old, ok := s.states[hwcID]; ok && proto.Equal(old.text, text) {
		return true // Same text re-sent: keep animating
	}

	ts := text.TextScroll
	s.states[hwcID] = &textScrollState{
		text:      proto.Clone(text).(*rwp.HWCText),
		mode:      ts.ScrollMode,
		speed:     ts.ScrollSpeed,
		dwell:     ts.PauseDwell,
		adaptive:  ts.AdaptiveSpeed,
		direction: 1,
		paused:    textScrollDwellTicks(ts.PauseDwell), // Dwell at the start before moving
	}
	return true
}

// Clear removes scroll state for a display (e.g. when graphics replace the text).
func (s *TextScroller) Clear(hwcID uint32) {
	s.mu.Lock()
	delete(s.states, hwcID)
	s.mu.Unlock()
}

// Reset removes all scroll state (e.g. on client disconnect).
func (s *TextScroller) Reset() {
	s.mu.Lock()
	s.states = make(map[uint32]*textScrollState)
	s.mu.Unlock()
}

// Active reports whether a display currently has scroll state.
func (s *TextScroller) Active(hwcID uint32) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	_, ok := s.states[hwcID]
	return ok
}

// Tick advances all animations by one TextScrollTickInterval and returns the IDs
// of displays whose scroll position changed and must be redrawn.
func (s *TextScroller) Tick() []uint32 {
	s.mu.Lock()
	defer s.mu.Unlock()

	var changed []uint32
	for hwcID, ss := range s.states {
		if ss.advance() {
			changed = append(changed, hwcID)
		}
	}
	return changed
}

// Apply returns the text to render for a display at its current scroll position.
// width, height, shrink and border must be the same values passed to WriteDisplayTileNew.
// The input is never modified: if trimming is needed a copy is returned,
// otherwise text itself is returned.
func (s *TextScroller) Apply(hwcID uint32, text *rwp.HWCText, width, height, shrink, border int) *rwp.HWCText {
	if text == nil {
		return nil
	}
	s.mu.Lock()
	ss, ok := s.states[hwcID]
	if !ok {
		s.mu.Unlock()
		return text
	}
	dims := [4]int{width, height, shrink, border}
	if !ss.measured || ss.dims != dims {
		ss.measure(dims)
	}
	titleTrim := ss.trim(ss.titleMax)
	line1Trim := ss.trim(ss.line1Max)
	line2Trim := ss.trim(ss.line2Max)
	s.mu.Unlock()

	if titleTrim == 0 && line1Trim == 0 && line2Trim == 0 {
		return text
	}
	out := proto.Clone(text).(*rwp.HWCText)
	out.Title = trimRunes(out.Title, titleTrim)
	out.Textline1 = trimRunes(out.Textline1, line1Trim)
	out.Textline2 = trimRunes(out.Textline2, line2Trim)
	return out
}

// measure computes how far each scrolled field must move to show its end.
func (ss *textScrollState) measure(dims [4]int) {
	titleMax, line1Max, line2Max := ComputeScrollMaxOffset(ss.text, dims[0], dims[1], dims[2], dims[3])
	ts := ss.text.TextScroll
	if !ts.ScrollTitle {
		titleMax = 0
	}
	if !ts.ScrollTextline1 {
		line1Max = 0
	}
	if !ts.ScrollTextline2 {
		line2Max = 0
	}
	ss.titleMax, ss.line1Max, ss.line2Max = titleMax, line1Max, line2Max
	ss.maxOffset = max(titleMax, line1Max, line2Max)
	ss.dims = dims
	ss.measured = true
	if ss.charOffset > ss.maxOffset {
		ss.charOffset = ss.maxOffset
	}
}

// trim returns how many characters to cut from the front of a field with the given max offset.
//
// AdaptiveSpeed: proportional in both directions, so all fields start and finish together.
// Default forward: each field caps at its own max (shorter fields finish first and wait).
// Default backward: all fields start moving back at once at the same rate;
// shorter fields reach 0 first and wait for the longest.
func (ss *textScrollState) trim(fieldMax int) int {
	if fieldMax <= 0 || ss.charOffset <= 0 {
		return 0
	}
	if ss.adaptive && ss.maxOffset > 0 {
		return fieldMax * ss.charOffset / ss.maxOffset
	}
	if ss.direction < 0 {
		return max(fieldMax-(ss.maxOffset-ss.charOffset), 0)
	}
	return min(ss.charOffset, fieldMax)
}

// advance moves the animation one tick. Returns true if charOffset changed.
func (ss *textScrollState) advance() bool {
	if !ss.measured || ss.maxOffset <= 0 {
		return false
	}
	if ss.paused > 0 {
		ss.paused--
		return false
	}
	ss.tickCounter++
	if ss.tickCounter < textScrollTicksPerChar(ss.speed) {
		return false
	}
	ss.tickCounter = 0

	pause := textScrollDwellTicks(ss.dwell)
	switch ss.mode {
	case rwp.HWCText_TextScrollM_SCROLL_LOOP:
		ss.charOffset++
		if ss.charOffset > ss.maxOffset {
			ss.charOffset = 0 // Past the end: jump back to the start and dwell
			ss.paused = pause
		} else if ss.charOffset == ss.maxOffset {
			ss.paused = pause // Reached the end: dwell before jumping back
		}
	case rwp.HWCText_TextScrollM_SCROLL_BOUNCE:
		ss.charOffset += ss.direction
		if ss.charOffset >= ss.maxOffset {
			ss.charOffset = ss.maxOffset
			ss.direction = -1
			ss.paused = pause
		} else if ss.charOffset <= 0 {
			ss.charOffset = 0
			ss.direction = 1
			ss.paused = pause
		}
	default:
		return false
	}
	return true
}

// textScrollTicksPerChar maps ScrollSpeed to ticks per character step.
func textScrollTicksPerChar(speed uint32) int {
	switch speed {
	case 1: // slow
		return 5 // 500 ms per char
	case 3: // fast
		return 1 // 100 ms per char
	default: // 0 = default, 2 = normal
		return 3 // 300 ms per char
	}
}

// textScrollDwellTicks maps PauseDwell to ticks to wait at an endpoint.
func textScrollDwellTicks(dwell uint32) int {
	switch dwell {
	case 1: // short
		return 8 // ~0.8 s
	case 3: // long
		return 30 // ~3 s
	default: // 0 = default, 2 = medium
		return 15 // ~1.5 s
	}
}

func trimRunes(s string, n int) string {
	if n <= 0 {
		return s
	}
	r := []rune(s)
	if n >= len(r) {
		return ""
	}
	return string(r[n:])
}
