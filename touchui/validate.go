package touchui

import (
	"fmt"

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
				if len(widget.GetOptions().GetSource()) > MaxSourceLen {
					return fmt.Errorf("widget %d video source exceeds %d bytes", widget.GetHWCID(), MaxSourceLen)
				}
			}
		}
	}

	if videoCount > MaxVideoWidgets {
		return fmt.Errorf("%d VIDEO widgets exceeds the maximum of %d", videoCount, MaxVideoWidgets)
	}
	if want := cfg.GetActivePage(); want != 0 && !pageIDs[want] {
		return fmt.Errorf("ActivePage %d is not a declared page", want)
	}
	return nil
}
