package rawpanellib

import (
	"strings"
	"testing"

	rwp "github.com/SKAARHOJ/rawpanel-lib/ibeam_rawpanel"
)

// Formatting 10/11 (big text) without TextStyling used to panic with a nil pointer.
func TestHWCTextBigFormattingWithoutTextStyling(t *testing.T) {
	for _, f := range []rwp.HWCText_FormattingE{10, 11} {
		msgs := []*rwp.InboundMessage{{States: []*rwp.HWCState{{HWCIDs: []uint32{1}, HWCText: &rwp.HWCText{Formatting: f, Textline1: "LIVE"}}}}}
		out := InboundMessagesToRawPanelASCIIstrings(msgs)
		if len(out) == 0 || !strings.HasPrefix(out[0], "HWCt#1=") {
			t.Errorf("formatting %d: got %q", f, out)
		}
	}
}
