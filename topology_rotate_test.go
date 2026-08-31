package rawpanellib

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/subchen/go-xmldom"

	"github.com/SKAARHOJ/rawpanel-lib/topology"
)

// The real CLIX/base fixtures from ibeam-hardware-definitions. The panel is a 280x1424
// portrait screen worn as a 1424x280 bar, so this is what it reports at its normal mounted
// orientation and what every expectation below is measured against.
const clixTopologyJSON = `{
	"HWc": [{"id": 1, "txt": "Screen", "type": 500, "x": 1101, "y": 222}],
	"title": "Clix",
	"typeIndex": {
		"500": {"desc": "Touch screen", "w": 1709, "h": 336, "subidx": -1,
			"disp": {"w": 1424, "h": 280, "subidx": -1, "type": "touch"}}
	}
}`

const clixTopologySVG = `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 2200 445" width="100%" id="ctrlimg">
  <defs>
    <linearGradient id="RackProMiniFrontBlueRaised" x1="0%" y1="0%" x2="0%" y2="100%">
      <stop offset="0%" style="stop-color:rgb(99, 171, 235);stop-opacity:1" />
    </linearGradient>
  </defs>
  <style>text {font-family:Sans,Arial;}</style>
  <rect x="0" y="0" width="2200" height="445" rx="20" ry="20" style="fill:url(#RackProMiniFrontBlueRaised);" />
  <rect width="1709" height="336" y="54" x="246" style="fill:rgb(22,22,22);" />
</svg>`

func clixTopology(t *testing.T) *topology.Topology {
	t.Helper()
	parsed := &topology.Topology{}
	if err := json.Unmarshal([]byte(clixTopologyJSON), parsed); err != nil {
		t.Fatalf("parsing the Clix fixture: %v", err)
	}
	return parsed
}

// A quarter turn transposes the canvas, so the screen has to land centered in a 445x2200 one
// and its dimensions — chassis footprint and framebuffer alike — have to trade places.
func TestRotateTopologyClixQuarterTurn(t *testing.T) {
	rotated := RotateTopology(clixTopology(t), 90, 2200, 445)

	screen := rotated.HWc[0]
	if screen.X != 223 || screen.Y != 1101 {
		t.Errorf("screen center = %d,%d, want 223,1101", screen.X, screen.Y)
	}

	def := rotated.TypeIndex[500]
	if def.W != 336 || def.H != 1709 {
		t.Errorf("chassis footprint = %dx%d, want 336x1709", def.W, def.H)
	}
	if def.Disp.W != 280 || def.Disp.H != 1424 {
		t.Errorf("framebuffer = %dx%d, want 280x1424", def.Disp.W, def.Disp.H)
	}
	if def.Disp.Type != "touch" {
		t.Errorf("display type = %q, want touch", def.Disp.Type)
	}
}

// The source must not be touched: the core rotates its base topology on every fold, so a
// mutating rotation would compound turn after turn.
func TestRotateTopologyDoesNotMutateSource(t *testing.T) {
	base := clixTopology(t)
	RotateTopology(base, 90, 2200, 445)

	if base.HWc[0].X != 1101 || base.HWc[0].Y != 222 {
		t.Errorf("source center moved to %d,%d", base.HWc[0].X, base.HWc[0].Y)
	}
	if def := base.TypeIndex[500]; def.W != 1709 || def.Disp.W != 1424 {
		t.Errorf("source dimensions changed: %dx%d, disp %dx%d", def.W, def.H, def.Disp.W, def.Disp.H)
	}
}

// Four quarter turns are the identity. This is the cheapest check that rotatePoint's canvas
// transposition and the dimension swaps agree with each other.
func TestRotateTopologyFourQuarterTurnsRoundTrip(t *testing.T) {
	current := clixTopology(t)
	width, height := 2200, 445
	for turn := 0; turn < 4; turn++ {
		current = RotateTopology(current, 90, width, height)
		width, height = height, width
	}

	original := clixTopology(t)
	if current.HWc[0] != original.HWc[0] {
		t.Errorf("after four turns HWc = %+v, want %+v", current.HWc[0], original.HWc[0])
	}
	got, want := current.TypeIndex[500], original.TypeIndex[500]
	if got.W != want.W || got.H != want.H || got.Disp.W != want.Disp.W || got.Disp.H != want.Disp.H {
		t.Errorf("after four turns %dx%d disp %dx%d, want %dx%d disp %dx%d",
			got.W, got.H, got.Disp.W, got.Disp.H, want.W, want.H, want.Disp.W, want.Disp.H)
	}
}

// A half turn keeps the canvas shape, so only the position moves.
func TestRotateTopologyHalfTurn(t *testing.T) {
	rotated := RotateTopology(clixTopology(t), 180, 2200, 445)

	if screen := rotated.HWc[0]; screen.X != 1099 || screen.Y != 223 {
		t.Errorf("screen center = %d,%d, want 1099,223", screen.X, screen.Y)
	}
	if def := rotated.TypeIndex[500]; def.W != 1709 || def.H != 336 {
		t.Errorf("footprint = %dx%d, want it unchanged at 1709x336", def.W, def.H)
	}
}

// A zero or full turn is the identity, and anything that is not a quarter turn is refused the
// same way rather than half-applied.
func TestRotateTopologyPassesThroughNonRotations(t *testing.T) {
	base := clixTopology(t)
	for _, deg := range []int{0, 360, -360, 45, 91} {
		if got := RotateTopology(base, deg, 2200, 445); got != base {
			t.Errorf("RotateTopology(%d) returned a new topology, want the source unchanged", deg)
		}
	}
	if RotateTopology(nil, 90, 2200, 445) != nil {
		t.Error("RotateTopology(nil) should stay nil")
	}
}

// A negative or over-full angle is still a quarter turn and must behave as one.
func TestRotateTopologyNormalizesAngle(t *testing.T) {
	clockwise := RotateTopology(clixTopology(t), 270, 2200, 445)
	equivalent := RotateTopology(clixTopology(t), -90, 2200, 445)

	if clockwise.HWc[0] != equivalent.HWc[0] {
		t.Errorf("-90 gave %+v, want the same as 270: %+v", equivalent.HWc[0], clockwise.HWc[0])
	}
	if clockwise.HWc[0].X != 222 || clockwise.HWc[0].Y != 1099 {
		t.Errorf("270 gave %d,%d, want 222,1099", clockwise.HWc[0].X, clockwise.HWc[0].Y)
	}
}

// H == 0 means a circle of diameter W. Swapping would zero the width and invent a height.
func TestRotateTopologyLeavesCirclesAlone(t *testing.T) {
	base := &topology.Topology{
		TypeIndex: map[uint32]topology.TopologyHWcTypeDef{7: {Desc: "Button", W: 120}},
		HWc:       []topology.TopologyHWcomponent{{Id: 1, Type: 7, X: 500, Y: 100}},
	}

	def := RotateTopology(base, 90, 1000, 400).TypeIndex[7]
	if def.W != 120 || def.H != 0 {
		t.Errorf("circle became %dx%d, want 120x0", def.W, def.H)
	}
}

// Sub elements are drawn at an offset from the component's center. A rectangle's offset is its
// top-left corner, so it has to be recomputed once width and height trade places; a circle's is
// its center and only turns.
func TestRotateTopologyTurnsSubElements(t *testing.T) {
	base := &topology.Topology{
		TypeIndex: map[uint32]topology.TopologyHWcTypeDef{7: {W: 200, H: 100, Sub: []topology.TopologyHWcTypeDefSubEl{
			{ObjType: "r", X: -100, Y: -50, W: 200, H: 100},
			{ObjType: "c", X: 60, Y: 0, R: 10},
		}}},
		HWc: []topology.TopologyHWcomponent{{Id: 1, Type: 7, X: 500, Y: 200}},
	}

	sub := RotateTopology(base, 90, 1000, 400).TypeIndex[7].Sub

	// The rectangle filled the component and must still fill it, now 100x200.
	if sub[0].W != 100 || sub[0].H != 200 || sub[0].X != -50 || sub[0].Y != -100 {
		t.Errorf("rect = %d,%d %dx%d, want -50,-100 100x200", sub[0].X, sub[0].Y, sub[0].W, sub[0].H)
	}
	// The circle sat to the right of center; a clockwise turn puts it below, radius intact.
	if sub[1].X != 0 || sub[1].Y != 60 || sub[1].R != 10 {
		t.Errorf("circle = %d,%d r%d, want 0,60 r10", sub[1].X, sub[1].Y, sub[1].R)
	}
}

func TestRotateSVGClixQuarterTurn(t *testing.T) {
	rotated, err := RotateSVG(clixTopologySVG, 90)
	if err != nil {
		t.Fatalf("RotateSVG: %v", err)
	}

	doc, err := xmldom.ParseXML(rotated)
	if err != nil {
		t.Fatalf("the result does not parse: %v", err)
	}
	if got := doc.Root.GetAttributeValue("viewBox"); got != "0 0 445 2200" {
		t.Errorf("viewBox = %q, want \"0 0 445 2200\"", got)
	}
	if got := doc.Root.GetAttributeValue("id"); got != "ctrlimg" {
		t.Errorf("root attributes were lost: id = %q", got)
	}

	// defs and style stay direct children of <svg>; the artwork moves into one <g>.
	var childNames []string
	for _, child := range doc.Root.Children {
		childNames = append(childNames, child.Name)
	}
	if strings.Join(childNames, ",") != "defs,style,g" {
		t.Errorf("root children = %v, want defs,style,g", childNames)
	}
	group := doc.Root.Children[2]
	if got := group.GetAttributeValue("transform"); got != "translate(445,0) rotate(90)" {
		t.Errorf("transform = %q, want \"translate(445,0) rotate(90)\"", got)
	}
	if len(group.Children) != 2 {
		t.Errorf("group holds %d shapes, want the 2 rects", len(group.Children))
	}
	if doc.Root.GetChild("defs").GetChild("linearGradient").GetAttributeValue("id") != "RackProMiniFrontBlueRaised" {
		t.Error("the gradient definition did not survive")
	}
}

// The whole point of doing both together: the rotated SVG and the rotated topology have to
// agree about where the screen is. The SVG draws the screen rect at 246,54 1709x336 — center
// 1100.5,222 — and the topology puts the screen HWc at 1101,222.
func TestRotateSVGAgreesWithRotateTopology(t *testing.T) {
	for _, deg := range []int{90, 180, 270} {
		rotatedSVG, err := RotateSVG(clixTopologySVG, deg)
		if err != nil {
			t.Fatalf("RotateSVG(%d): %v", deg, err)
		}
		doc, err := xmldom.ParseXML(rotatedSVG)
		if err != nil {
			t.Fatalf("RotateSVG(%d) does not parse: %v", deg, err)
		}

		// Apply the emitted transform to the screen rect's center by hand.
		transform := doc.Root.Children[len(doc.Root.Children)-1].GetAttributeValue("transform")
		gotX, gotY := applyRotationTransform(t, transform, 246+1709/2, 54+336/2)

		screen := RotateTopology(clixTopology(t), deg, 2200, 445).HWc[0]
		if abs(gotX-screen.X) > 1 || abs(gotY-screen.Y) > 1 {
			t.Errorf("at %d degrees the SVG puts the screen at %d,%d but the topology at %d,%d",
				deg, gotX, gotY, screen.X, screen.Y)
		}
	}
}

func TestRotateSVGRejectsBadInput(t *testing.T) {
	if _, err := RotateSVG(clixTopologySVG, 45); err == nil {
		t.Error("a 45 degree rotation should be refused")
	}
	if _, err := RotateSVG(`<svg viewBox="10 0 2200 445"></svg>`, 90); err == nil {
		t.Error("a viewBox with an offset origin should be refused")
	}
	if _, err := RotateSVG(`<svg width="100%"></svg>`, 90); err == nil {
		t.Error("an SVG with no viewBox should be refused")
	}
	if got, err := RotateSVG(clixTopologySVG, 0); err != nil || got != clixTopologySVG {
		t.Error("a zero rotation should hand the SVG straight back")
	}
}

func TestSVGCanvasSize(t *testing.T) {
	width, height, err := SVGCanvasSize(clixTopologySVG)
	if err != nil {
		t.Fatalf("SVGCanvasSize: %v", err)
	}
	if width != 2200 || height != 445 {
		t.Errorf("canvas = %dx%d, want 2200x445", width, height)
	}
}

// applyRotationTransform parses the "translate(a,b) rotate(deg)" form rotationTransform emits
// and applies it, so the test checks the string the SVG renderer will actually see rather than
// trusting the code that produced it.
func applyRotationTransform(t *testing.T, transform string, x, y int) (int, int) {
	t.Helper()
	cleaned := strings.NewReplacer("translate(", "", "rotate(", "", ")", " ", ",", " ").Replace(transform)

	var translateX, translateY, deg int
	if _, err := fmt.Sscan(cleaned, &translateX, &translateY, &deg); err != nil {
		t.Fatalf("cannot read transform %q: %v", transform, err)
	}
	rotatedX, rotatedY := rotateOffset(x, y, deg)
	return rotatedX + translateX, rotatedY + translateY
}

func abs(v int) int {
	if v < 0 {
		return -v
	}
	return v
}
