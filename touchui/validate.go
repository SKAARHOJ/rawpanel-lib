package touchui

import (
	"fmt"
	"strings"

	rwp "github.com/SKAARHOJ/rawpanel-lib/ibeam_rawpanel"
)

// Caps mirror proto/touchmanager.options — exceeding them would break the
// UI's static nanopb decode, so configs are rejected here first.
const (
	MaxPages          = 8
	MaxWidgetsPerPage = 24
	MaxLabelLen       = 47 // nanopb max_size includes the NUL
	MaxTitleLen       = 23
	MaxSourceLen      = 127
	MaxVideoWidgets   = 4 // pi DRM video planes

	MaxChoices          = 32 // ROLLER: entries in Options.Choices
	MaxChoiceLen        = 23
	MaxCompressorParams = 6 // COMPRESSOR: one per RoleE
	MaxParamLabelLen    = 15
	MaxEditLen          = 63 // LABEL: Options.EditMaxLen ceiling

	MaxMarkersPerWidget = 8  // VIDEO: Options.Markers on one widget
	MaxMarkers          = 16 // VIDEO: markers across the whole config, matching the tree's static cap
	MaxMarkerCoord      = 1000
)

// Validate checks a TouchUIConfig against the panel's static limits and its
// referential integrity. It returns the first violation found.
func Validate(cfg *rwp.TouchUIConfig) error {
	pages := cfg.GetPages()
	if len(pages) == 0 {
		return fmt.Errorf("config has no pages")
	}
	if len(pages) > MaxPages {
		return fmt.Errorf("%d pages exceeds the maximum of %d", len(pages), MaxPages)
	}

	pageIDs := map[uint32]bool{}
	hwcIDs := map[uint32]bool{}
	videoCount := 0
	markerCount := 0

	for _, page := range pages {
		if page.GetId() == 0 {
			return fmt.Errorf("page id 0 is reserved")
		}
		if pageIDs[page.GetId()] {
			return fmt.Errorf("duplicate page id %d", page.GetId())
		}
		pageIDs[page.GetId()] = true
		if len(page.GetTitle()) > MaxTitleLen {
			return fmt.Errorf("page %d title exceeds %d bytes", page.GetId(), MaxTitleLen)
		}
		if len(page.GetWidgets()) > MaxWidgetsPerPage {
			return fmt.Errorf("page %d has %d widgets, maximum is %d", page.GetId(), len(page.GetWidgets()), MaxWidgetsPerPage)
		}
		gridMode := page.GetGridRows() > 0 && page.GetGridCols() > 0

		for _, widget := range page.GetWidgets() {
			if widget.GetHWCID() == 0 {
				return fmt.Errorf("page %d: widget HWC id 0 is reserved", page.GetId())
			}
			if hwcIDs[widget.GetHWCID()] {
				return fmt.Errorf("duplicate widget HWC id %d", widget.GetHWCID())
			}
			hwcIDs[widget.GetHWCID()] = true
			if len(widget.GetLabel()) > MaxLabelLen {
				return fmt.Errorf("widget %d label exceeds %d bytes", widget.GetHWCID(), MaxLabelLen)
			}
			if gridMode {
				row, col := widget.GetRow(), widget.GetCol()
				if row < 1 || col < 1 || row > page.GetGridRows() || col > page.GetGridCols() {
					return fmt.Errorf("widget %d at %d,%d is outside the %dx%d grid",
						widget.GetHWCID(), row, col, page.GetGridRows(), page.GetGridCols())
				}
			}
			if widget.GetType() == rwp.TouchUIWidget_VIDEO {
				videoCount++
				markerCount += len(widget.GetOptions().GetMarkers())
				if len(widget.GetOptions().GetSource()) > MaxSourceLen {
					return fmt.Errorf("widget %d video source exceeds %d bytes", widget.GetHWCID(), MaxSourceLen)
				}
			}
			if err := validateWidgetOptions(widget, hwcIDs); err != nil {
				return err
			}
		}
	}

	if videoCount > MaxVideoWidgets {
		return fmt.Errorf("%d VIDEO widgets exceeds the maximum of %d", videoCount, MaxVideoWidgets)
	}
	// Config-wide, not per widget: markers travel as one flat list on the widget tree, so the
	// static cap the UI decodes into is a total.
	if markerCount > MaxMarkers {
		return fmt.Errorf("%d markers exceeds the maximum of %d for a config", markerCount, MaxMarkers)
	}
	if want := cfg.GetActivePage(); want != 0 && !pageIDs[want] {
		return fmt.Errorf("ActivePage %d is not a declared page", want)
	}
	return nil
}

// validateWidgetOptions checks the per-type Options of one widget. hwcIDs is the running
// set of ids claimed so far and is MUTATED here: compressor member parameters are
// addressable HWCs in their own right and share the widget id space, so they must be
// registered alongside widget ids. A member colliding with a widget (or with another
// compressor's member) would make the renderer's lookup and the core's state store
// ambiguous, routing a fader's state to the wrong thing — so it is rejected up front.
func validateWidgetOptions(widget *rwp.TouchUIWidget, hwcIDs map[uint32]bool) error {
	id := widget.GetHWCID()
	opts := widget.GetOptions()

	if opts.GetEditKind() != rwp.TouchUIWidgetOptions_NONE && widget.GetType() != rwp.TouchUIWidget_LABEL {
		return fmt.Errorf("widget %d: EditKind is only valid on a LABEL", id)
	}
	if opts.GetEditMaxLen() > MaxEditLen {
		return fmt.Errorf("widget %d: EditMaxLen %d exceeds the maximum of %d", id, opts.GetEditMaxLen(), MaxEditLen)
	}
	if len(opts.GetMarkers()) > 0 && widget.GetType() != rwp.TouchUIWidget_VIDEO {
		return fmt.Errorf("widget %d: Markers are only valid on a VIDEO widget", id)
	}

	switch widget.GetType() {
	case rwp.TouchUIWidget_ROLLER:
		choices := opts.GetChoices()
		if len(choices) == 0 {
			return fmt.Errorf("widget %d: ROLLER has no Choices", id)
		}
		if len(choices) > MaxChoices {
			return fmt.Errorf("widget %d: %d Choices exceeds the maximum of %d", id, len(choices), MaxChoices)
		}
		for i, c := range choices {
			if len(c) > MaxChoiceLen {
				return fmt.Errorf("widget %d: choice %d exceeds %d bytes", id, i, MaxChoiceLen)
			}
			// Choices reach the renderer newline-joined (the format lv_roller wants), so an
			// embedded newline would silently split one option into two and shift every
			// index after it — exactly the desync the fixed-list rule exists to prevent.
			if strings.ContainsAny(c, "\r\n") {
				return fmt.Errorf("widget %d: choice %d contains a line break", id, i)
			}
		}

	case rwp.TouchUIWidget_XYPAD:
		if opts.GetRelative() && opts.GetCenterReturn() {
			return fmt.Errorf("widget %d: CenterReturn is meaningless with Relative — a delta pad has no position to return to", id)
		}

	case rwp.TouchUIWidget_VIDEO:
		// A video region draws no cursor of its own, so there is nothing to send home; unlike
		// the XYPAD rule this holds in both modes.
		if opts.GetCenterReturn() {
			return fmt.Errorf("widget %d: CenterReturn is meaningless on a VIDEO widget — it has no cursor to return", id)
		}
		markers := opts.GetMarkers()
		if len(markers) > MaxMarkersPerWidget {
			return fmt.Errorf("widget %d: %d Markers exceeds the maximum of %d", id, len(markers), MaxMarkersPerWidget)
		}
		for _, m := range markers {
			mid := m.GetHWCID()
			if mid == 0 {
				return fmt.Errorf("widget %d: marker HWC id 0 is reserved", id)
			}
			// Same reason the compressor members register here: a marker is an addressable
			// HWC sharing the widget id space, and a collision would route one HWC's state to
			// the wrong box (or to a widget), which fails silently at render time.
			if hwcIDs[mid] {
				return fmt.Errorf("marker HWC id %d (widget %d) collides with another HWC id", mid, id)
			}
			hwcIDs[mid] = true
			if m.GetW() > MaxMarkerCoord || m.GetH() > MaxMarkerCoord {
				return fmt.Errorf("widget %d: marker %d size %dx%d is outside the 0..%d domain",
					id, mid, m.GetW(), m.GetH(), MaxMarkerCoord)
			}
		}

	case rwp.TouchUIWidget_COMPRESSOR:
		params := opts.GetParams()
		if len(params) == 0 {
			return fmt.Errorf("widget %d: COMPRESSOR has no Params", id)
		}
		if len(params) > MaxCompressorParams {
			return fmt.Errorf("widget %d: %d Params exceeds the maximum of %d", id, len(params), MaxCompressorParams)
		}
		roles := map[rwp.TouchUICompressorParam_RoleE]bool{}
		for _, p := range params {
			pid := p.GetHWCID()
			if pid == 0 {
				return fmt.Errorf("widget %d: compressor param HWC id 0 is reserved", id)
			}
			if hwcIDs[pid] {
				return fmt.Errorf("compressor param HWC id %d (widget %d) collides with another HWC id", pid, id)
			}
			hwcIDs[pid] = true
			if roles[p.GetRole()] {
				return fmt.Errorf("widget %d: duplicate compressor role %v", id, p.GetRole())
			}
			roles[p.GetRole()] = true
			if len(p.GetLabel()) > MaxParamLabelLen {
				return fmt.Errorf("widget %d: compressor param %d label exceeds %d bytes", id, pid, MaxParamLabelLen)
			}
			// 0/0 means "panel default for the role" and is resolved later; any other
			// pair must describe a real range or the 0..1000 mapping collapses.
			if (p.GetMin() != 0 || p.GetMax() != 0) && p.GetMin() >= p.GetMax() {
				return fmt.Errorf("widget %d: compressor param %d has an empty range %d..%d", id, pid, p.GetMin(), p.GetMax())
			}
		}
	}
	return nil
}
