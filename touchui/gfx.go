package touchui

import (
	"fmt"

	rwp "github.com/SKAARHOJ/rawpanel-lib/ibeam_rawpanel"

	gen "github.com/SKAARHOJ/rawpanel-lib/touchui/gen"
)

// Bounds for the RGB565 blobs sent to the UI. 320x200x2 = 128000 bytes, within
// the nanopb cap (WidgetGfx.rgb565 max_size:131072). Larger client images are
// downscaled preserving aspect.
const (
	maxGfxW = 320
	maxGfxH = 200
)

// GfxToWidgetGfx decodes an inbound HWCGfx (MONO / Gray4bit / RGB16bit wire
// formats, see ibeam_lib_monogfx for the layout reference) into a
// little-endian RGB565 WidgetGfx frame, downscaling to the transport bounds.
func GfxToWidgetGfx(hwc uint32, g *rwp.HWCGfx, epoch uint32) (*gen.WidgetGfx, error) {
	w, h := int(g.GetW()), int(g.GetH())
	if w <= 0 || h <= 0 {
		return nil, fmt.Errorf("touchui: gfx without dimensions")
	}

	pix, err := decodeGfx(g, w, h) // 0xRRGGBB per pixel, row-major
	if err != nil {
		return nil, err
	}

	outW, outH := fitWithin(w, h, maxGfxW, maxGfxH)
	out := &gen.WidgetGfx{
		Epoch: epoch,
		HwcId: hwc,
		W:     uint32(outW),
		H:     uint32(outH),
	}
	out.Rgb565 = make([]byte, outW*outH*2)
	for y := 0; y < outH; y++ {
		srcY := y * h / outH
		for x := 0; x < outW; x++ {
			srcX := x * w / outW
			rgb := pix[srcY*w+srcX]
			v := packRGB565(rgb)
			i := (y*outW + x) * 2
			out.Rgb565[i] = byte(v)        // little-endian: LVGL-native on all targets
			out.Rgb565[i+1] = byte(v >> 8)
		}
	}
	return out, nil
}

func fitWithin(w, h, maxW, maxH int) (int, int) {
	if w <= maxW && h <= maxH {
		return w, h
	}
	outW, outH := w, h
	if outW > maxW {
		outH = outH * maxW / outW
		outW = maxW
	}
	if outH > maxH {
		outW = outW * maxH / outH
		outH = maxH
	}
	if outW < 1 {
		outW = 1
	}
	if outH < 1 {
		outH = 1
	}
	return outW, outH
}

func packRGB565(rgb uint32) uint16 {
	r, g, b := (rgb>>16)&0xFF, (rgb>>8)&0xFF, rgb&0xFF
	return uint16((r>>3)<<11 | (g>>2)<<5 | b>>3)
}

// decodeGfx expands the three wire formats to 0xRRGGBB pixels.
func decodeGfx(g *rwp.HWCGfx, w, h int) ([]uint32, error) {
	data := g.GetImageData()
	pix := make([]uint32, w*h)

	switch g.GetImageType() {
	case rwp.HWCGfx_MONO:
		// 1 bit per pixel, MSB first, each row padded to whole bytes.
		stride := (w + 7) / 8
		if stride*h > len(data) {
			return nil, fmt.Errorf("touchui: MONO gfx short: %d < %d", len(data), stride*h)
		}
		for y := 0; y < h; y++ {
			for x := 0; x < w; x++ {
				if data[y*stride+x/8]&(1<<(7-x%8)) != 0 {
					pix[y*w+x] = 0xFFFFFF
				}
			}
		}

	case rwp.HWCGfx_Gray4bit:
		// 2 pixels per byte, most-significant nibble first.
		if (w*h+1)/2 > len(data) {
			return nil, fmt.Errorf("touchui: Gray4 gfx short: %d < %d", len(data), (w*h+1)/2)
		}
		for i := 0; i < w*h; i++ {
			nibble := data[i/2]
			if i%2 == 0 {
				nibble >>= 4
			}
			nibble &= 0x0F
			v := uint32(nibble) * 17 // 0..15 -> 0..255
			pix[i] = v<<16 | v<<8 | v
		}

	case rwp.HWCGfx_RGB16bit:
		// 2 bytes per pixel, big-endian, and the channels run blue-green-red from the top bit
		// down rather than red-green-blue: every writer packs (b<<11)|(g<<5)|r — see the
		// pixelColor in Reactor's DFeedback and SetOLEDPixelColor in ibeam_lib_monogfx — and
		// the reader in rawpanelhelpers.go takes them back out in that order. Reading this as
		// plain RGB565 swaps red and blue, which is what turned dark blue into dark red.
		if w*h*2 > len(data) {
			return nil, fmt.Errorf("touchui: RGB16 gfx short: %d < %d", len(data), w*h*2)
		}
		for i := 0; i < w*h; i++ {
			v := uint16(data[i*2])<<8 | uint16(data[i*2+1])
			b := uint32(v>>11) & 0x1F
			gch := uint32(v>>5) & 0x3F
			r := uint32(v) & 0x1F
			pix[i] = (r<<3|r>>2)<<16 | (gch<<2|gch>>4)<<8 | (b<<3 | b>>2)
		}

	default:
		return nil, fmt.Errorf("touchui: unknown gfx image type %v", g.GetImageType())
	}
	return pix, nil
}
