package topology

import (
	"image"
	"image/color"
	"testing"
)

func TestCurveBounds(t *testing.T) {
	// Band below the center, 60..120 degrees (SVG convention: +Y down)
	c := &TopologyHWcTypeDef_DisplayCurve{InnerR: 100, OuterR: 200, StartAngle: 60, EndAngle: 120}
	b := c.Bounds(1000, 1000)
	if b.Max.Y != 1200 { // Outer radius straight down at 90 degrees
		t.Errorf("bottom = %d, want 1200", b.Max.Y)
	}
	if b.Min.Y != 1000+86 { // Inner radius at 60/120 degrees: 100*sin(60) = 86.6
		t.Errorf("top = %d, want 1086", b.Min.Y)
	}
	if b.Min.X != 900 || b.Max.X != 1100 { // Outer radius at 120/60 degrees: 200*cos(60) = 100
		t.Errorf("x = %d..%d, want 900..1100", b.Min.X, b.Max.X)
	}
}

// The bitmap's first column goes to the inner radius and its first row to the start
// angle, like the hardware bend (hw-pcb-test nv3007_bend.go). The flips reverse that.
func TestWarpToCurveOrientation(t *testing.T) {
	red := color.RGBA{255, 0, 0, 255}
	blue := color.RGBA{0, 0, 255, 255}

	// 10x40 bitmap: left half red, right half blue (W runs across the band)
	src := image.NewRGBA(image.Rect(0, 0, 10, 40))
	for y := 0; y < 40; y++ {
		for x := 0; x < 10; x++ {
			if x < 5 {
				src.SetRGBA(x, y, red)
			} else {
				src.SetRGBA(x, y, blue)
			}
		}
	}

	// Band to the right of the center (around 0 degrees)
	cx, cy := 500, 500
	at := func(c *TopologyHWcTypeDef_DisplayCurve, r int) color.RGBA {
		out := c.WarpToCurve(src, cx, cy, 1)
		b := c.Bounds(cx, cy)
		return out.RGBAAt(cx+r-b.Min.X, cy-b.Min.Y)
	}

	c := &TopologyHWcTypeDef_DisplayCurve{InnerR: 100, OuterR: 200, StartAngle: -20, EndAngle: 20}
	if got := at(c, 120); got != red {
		t.Errorf("near inner radius = %v, want red", got)
	}
	if got := at(c, 180); got != blue {
		t.Errorf("near outer radius = %v, want blue", got)
	}

	c.FlipRadial = true
	if got := at(c, 120); got != blue {
		t.Errorf("flipped, near inner radius = %v, want blue", got)
	}

	// Outside the band stays transparent
	out := c.WarpToCurve(src, cx, cy, 1)
	if got := out.RGBAAt(0, 0); got.A != 0 {
		t.Errorf("corner outside the band = %v, want transparent", got)
	}
}

func TestCurveInsetOnlyShrinksTheOutline(t *testing.T) {
	c := &TopologyHWcTypeDef_DisplayCurve{InnerR: 100, OuterR: 200, StartAngle: -20, EndAngle: 20}
	full := c.Bounds(0, 0)
	flat := c.SectorPath(0, 0)
	c.InsetR = 20
	if c.Bounds(0, 0) != full {
		t.Errorf("InsetR changed the display bounds: %v, want %v", c.Bounds(0, 0), full)
	}
	if c.SectorPath(0, 0) != annularSectorPath(0, 0, 120, 180, -20, 20) || c.SectorPath(0, 0) == flat {
		t.Errorf("outline with InsetR = %q, want the band 120..180", c.SectorPath(0, 0))
	}
}

func TestCurveInsetAngle(t *testing.T) {
	c := &TopologyHWcTypeDef_DisplayCurve{InnerR: 100, OuterR: 200, StartAngle: 55, EndAngle: 78.5, InsetR: 10, InsetAngle: 1.5}
	if got, want := c.SectorPath(0, 0), annularSectorPath(0, 0, 110, 190, 56.5, 77); got != want {
		t.Errorf("outline = %q, want %q", got, want)
	}
	full := (&TopologyHWcTypeDef_DisplayCurve{InnerR: 100, OuterR: 200, StartAngle: 55, EndAngle: 78.5}).Bounds(0, 0)
	if c.Bounds(0, 0) != full {
		t.Errorf("InsetAngle changed the display bounds")
	}
}
