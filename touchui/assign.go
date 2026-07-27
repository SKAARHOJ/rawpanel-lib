package touchui

import rwp "github.com/SKAARHOJ/rawpanel-lib/ibeam_rawpanel"

// HWCIDBase is the first HWC id auto-assigned to a widget. Widget ids share the panel's HWc
// id space (folded widgets become topology HWCs), so the base sits above the physical HWc
// id range panels use. Layout editors leave a widget's HWC id at 0 to mean "assign on push"
// (a 0 id is reserved and would fail Validate); Reactor fills them in before pushing.
const HWCIDBase = 100

// AssignHWCIDs fills in an HWC id for every widget left at 0, so the config passes Validate
// and folds into the topology without colliding. Existing non-zero ids are preserved; new
// ids start at HWCIDBase and skip any already in use. Returns true if any id was assigned.
func AssignHWCIDs(cfg *rwp.TouchUIConfig) bool {
	used := map[uint32]bool{}
	for _, page := range cfg.GetPages() {
		for _, widget := range page.GetWidgets() {
			if id := widget.GetHWCID(); id != 0 {
				used[id] = true
			}
		}
	}

	next := uint32(HWCIDBase)
	assigned := false
	for _, page := range cfg.GetPages() {
		for _, widget := range page.GetWidgets() {
			if widget.GetHWCID() != 0 {
				continue
			}
			for used[next] {
				next++
			}
			widget.HWCID = next
			used[next] = true
			next++
			assigned = true
		}
	}
	return assigned
}
