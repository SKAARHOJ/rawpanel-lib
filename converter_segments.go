package rawpanellib

import (
	"strconv"
	"strings"

	su "github.com/SKAARHOJ/ibeam-lib-utils"
	rwp "github.com/SKAARHOJ/rawpanel-lib/ibeam_rawpanel"
)

// ASCII encodings shared by HWC#, HWCc# and HWCs#.

// modeToASCII encodes an HWCMode as the HWC# value: state in bits 0-2, output in bit 5 and
// the blink pattern in bits 8-11.
func modeToASCII(mode *rwp.HWCMode) uint32 {
	return uint32(mode.State&0x7) | uint32((mode.BlinkPattern&0xF)<<8) | uint32(su.Qint(mode.Output, 0b100000, 0))
}

func modeFromASCII(value int) *rwp.HWCMode {
	return &rwp.HWCMode{
		State:        rwp.HWCMode_StateE(value & 0xF),
		Output:       (value & 0x20) == 0x20,
		BlinkPattern: uint32((value >> 8) & 0xF),
	}
}

// colorToASCII encodes an HWCColor as the HWCc# value: bit 7 set, then either bit 6 and 2 bits
// per RGB channel, or a color index in bits 0-4. False when the color carries neither.
func colorToASCII(color *rwp.HWCColor) (int, bool) {
	if color.ColorRGB != nil {
		return 0b11000000 |
			((su.MapAndConstrainValue(int(color.ColorRGB.Red), 0, 0xFF, 0, 0x3) & 0x3) << 4) |
			((su.MapAndConstrainValue(int(color.ColorRGB.Green), 0, 0xFF, 0, 0x3) & 0x3) << 2) |
			((su.MapAndConstrainValue(int(color.ColorRGB.Blue), 0, 0xFF, 0, 0x3) & 0x3) << 0), true
	}
	if color.ColorIndex != nil {
		return 0b10000000 | int(color.ColorIndex.Index&0x1F), true
	}
	return 0, false
}

func colorFromASCII(value int) *rwp.HWCColor {
	if value&0b1000000 > 0 {
		return &rwp.HWCColor{
			ColorRGB: &rwp.ColorRGB{
				Red:   uint32(su.MapAndConstrainValue((value>>4)&0x3, 0, 0x3, 0, 0xFF)),
				Green: uint32(su.MapAndConstrainValue((value>>2)&0x3, 0, 0x3, 0, 0xFF)),
				Blue:  uint32(su.MapAndConstrainValue((value>>0)&0x3, 0, 0x3, 0, 0xFF)),
			},
		}
	}
	return &rwp.HWCColor{
		ColorIndex: &rwp.ColorIndex{
			Index: rwp.ColorIndex_Colors(value & 0x1F),
		},
	}
}

// segmentsToASCII encodes HWCSegments as the HWCs# value: one "mode,color" per LED, separated
// by "|", with mode and color encoded as in HWC# and HWCc#. The color is left out when a
// segment has none, and no segments give an empty value, which clears them on the panel.
//
//	HWCs#30=4,204|4,204|772,240|5,255|5,255|5,255
func segmentsToASCII(segments *rwp.HWCSegments) string {
	parts := make([]string, len(segments.Segments))
	for i, segment := range segments.Segments {
		mode := uint32(0)
		if segment.HWCMode != nil {
			mode = modeToASCII(segment.HWCMode)
		}
		parts[i] = strconv.Itoa(int(mode))
		if segment.HWCColor != nil {
			if color, ok := colorToASCII(segment.HWCColor); ok {
				parts[i] += "," + strconv.Itoa(color)
			}
		}
	}
	return strings.Join(parts, "|")
}

func segmentsFromASCII(value string) *rwp.HWCSegments {
	segments := &rwp.HWCSegments{}
	if strings.TrimSpace(value) == "" {
		return segments
	}
	for _, part := range strings.Split(value, "|") {
		fields := strings.Split(part, ",")
		mode, _ := strconv.Atoi(strings.TrimSpace(fields[0]))
		segment := &rwp.HWCSegments_Segment{HWCMode: modeFromASCII(mode)}
		if len(fields) > 1 && strings.TrimSpace(fields[1]) != "" {
			color, _ := strconv.Atoi(strings.TrimSpace(fields[1]))
			segment.HWCColor = colorFromASCII(color)
		}
		segments.Segments = append(segments.Segments, segment)
	}
	return segments
}
