package rawpanellib

import (
	"reflect"
	"testing"

	rwp "github.com/SKAARHOJ/rawpanel-lib/ibeam_rawpanel"
)

// An SVG element spanning several lines must keep its content when joined to one ASCII line.
func TestTopologySvgMultiLine(t *testing.T) {
	svg := "<svg>\n  <path d=\"M0 0\n  L10 10\"/>\n  <rect x=\"1\"\n    y=\"2\"/>\n</svg>"
	got := OutboundMessagesToRawPanelASCIIstrings([]*rwp.OutboundMessage{{PanelTopology: &rwp.PanelTopology{Svgbase: svg}}})
	want := []string{`_panelTopology_svgbase=<svg><path d="M0 0 L10 10"/><rect x="1" y="2"/></svg>`}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %q, want %q", got, want)
	}
}

// Only the topology parts present are written: an empty line would read back as a blank SVG.
func TestTopologyPartsOnlyWhenPresent(t *testing.T) {
	cases := []struct {
		topo *rwp.PanelTopology
		want []string
	}{
		{&rwp.PanelTopology{Json: "{}"}, []string{"_panelTopology_HWC={}"}},
		{&rwp.PanelTopology{Svgbase: "<svg/>"}, []string{"_panelTopology_svgbase=<svg/>"}},
		{&rwp.PanelTopology{Svgbase: "<svg/>", Json: "{}"}, []string{"_panelTopology_svgbase=<svg/>", "_panelTopology_HWC={}"}},
	}
	for _, c := range cases {
		got := OutboundMessagesToRawPanelASCIIstrings([]*rwp.OutboundMessage{{PanelTopology: c.topo}})
		if !reflect.DeepEqual(got, c.want) {
			t.Errorf("%v: got %q, want %q", c.topo, got, c.want)
		}
		back := RawPanelASCIIstringsToOutboundMessages(got)
		var svg, json string
		for _, m := range back {
			if m.PanelTopology != nil {
				svg += m.PanelTopology.Svgbase
				json += m.PanelTopology.Json
			}
		}
		if svg != c.topo.Svgbase || json != c.topo.Json {
			t.Errorf("%v: read back svg=%q json=%q", c.topo, svg, json)
		}
	}
}
