package touchui

import rwp "github.com/SKAARHOJ/rawpanel-lib/ibeam_rawpanel"

// HWCIDBase is the first HWC id auto-assigned to a widget. Widget ids share the panel's HWc
// id space (folded widgets become topology HWCs), so the base sits above the physical HWc
// id range panels use. Layout editors leave a widget's HWC id at 0 to mean "assign on push"
// (a 0 id is reserved and would fail Validate); Reactor fills them in before pushing.
const HWCIDBase = 100

// AssignHWCIDs fills in an HWC id for every widget and every VIDEO marker left at 0, so the
// config passes Validate and folds into the topology without colliding. Existing non-zero ids
// are preserved; new ids start at HWCIDBase and skip any already in use. Returns true if any id
// was assigned.
//
// The claimed set covers compressor members and markers as well as widgets, even though members
// are never auto-assigned. They are addressable HWCs in the same id space, so leaving them out
// of the set would let a widget be handed an id a member already owns — which Validate then
// rejects, turning a hand-authored compressor into a config that will not push.
func AssignHWCIDs(cfg *rwp.TouchUIConfig) bool {
	used := map[uint32]bool{}
	for _, page := range cfg.GetPages() {
		for _, widget := range page.GetWidgets() {
			if id := widget.GetHWCID(); id != 0 {
				used[id] = true
			}
			for _, p := range widget.GetOptions().GetParams() {
				if id := p.GetHWCID(); id != 0 {
					used[id] = true
				}
			}
			for _, m := range widget.GetOptions().GetMarkers() {
				if id := m.GetHWCID(); id != 0 {
					used[id] = true
				}
			}
		}
	}

	next := uint32(HWCIDBase)
	assigned := false
	claim := func() uint32 {
		for used[next] {
			next++
		}
		id := next
		used[id] = true
		next++
		assigned = true
		return id
	}

	for _, page := range cfg.GetPages() {
		for _, widget := range page.GetWidgets() {
			if widget.GetHWCID() == 0 {
				widget.HWCID = claim()
			}
			for _, m := range widget.GetOptions().GetMarkers() {
				if m.GetHWCID() == 0 {
					m.HWCID = claim()
				}
			}
		}
	}
	return assigned
}
