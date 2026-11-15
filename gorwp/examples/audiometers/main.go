package main

import (
	"context"
	"flag"
	"math"
	"time"

	gorwp "github.com/SKAARHOJ/rawpanel-lib/gorwp"
	rwp "github.com/SKAARHOJ/rawpanel-lib/ibeam_rawpanel"
	log "github.com/s00500/env_logger"
)

// This application shows the AudioMeter processor: it puts an animated stereo VU meter on
// every display of the panel. The panel renders the meter itself - the client only sends
// levels (0-1000). On colour displays (like the Flywheel TFT) the bars are coloured in
// green/amber/red zones, and the meter follows the display's text scale and border.
//
//	go run . -panel 192.168.1.10:9923 -type fixed
func main() {
	ipAndPort := flag.String("panel", "localhost:9923", "IP and port of the SKAARHOJ panel")
	meterType := flag.String("type", "fixed", "Meter type: fixed, fixedw (block style) or free")
	title := flag.String("title", "PGM", "Meter title")
	hwcOnly := flag.Uint("hwc", 0, "Only draw on this HWC (0 = all displays)")
	flag.Parse()

	mt := map[string]rwp.ProcAudioMeter_MeterTypeE{
		"fixed":  rwp.ProcAudioMeter_Fixed176x32,
		"fixedw": rwp.ProcAudioMeter_Fixed176x32w,
		"free":   rwp.ProcAudioMeter_Free,
	}[*meterType]

	// Connecting to the SKAARHOJ Raw Panel device:
	log.Printf("Trying to connect to panel on %s...\n", *ipAndPort)
	ctx, cancel := context.WithCancel(context.Background())
	rp, err := gorwp.Connect(*ipAndPort, ctx, cancel)
	if log.Should(err) {
		return
	}
	log.Printf("Success - Connected to %s (%s, S/N %s)\n", rp.State.GetName(), rp.State.GetModel(), rp.State.GetSerial())
	rp.SetBrightness(8)

	// The classic meter: green up to -18dB, amber up to 0dB, red above (positions on the
	// 0-1000 scale; the Fixed176x32 ticks sit at multiples of 1000/12).
	barColors := []*rwp.ProcColorStop{
		{Pos: 0, RGB: 0x00FF00},
		{Pos: 583, RGB: 0xFFBF00},
		{Pos: 833, RGB: 0xFF0000},
	}

	top := rp.State.GetTopology()
	start := time.Now()
	for ctx.Err() == nil {
		t := time.Since(start).Seconds()
		left := level(t, 0)
		right := level(t, 1.3)

		for _, hwc := range top.GetHWCs() {
			if *hwcOnly != 0 && hwc != uint32(*hwcOnly) {
				continue
			}
			typeDef, _ := top.GetHWCtype(hwc) // Type Definition tells us what type this HWC is
			disp := typeDef.DisplayInfo()
			if disp == nil || disp.W <= 0 || disp.H <= 0 {
				continue
			}

			rp.SendRawState(&rwp.HWCState{
				HWCIDs: []uint32{hwc},
				Processors: &rwp.Processors{
					AudioMeter: &rwp.ProcAudioMeter{
						MeterType: mt,
						W:         uint32(disp.W),
						H:         uint32(disp.H),
						Title:     *title,

						// Taken from the topology, so the meter matches the display:
						ImageType:     imageType(disp.Type),
						TextScale:     float32(disp.TextScale),
						TextBorderPct: uint32(disp.TextBorderPct),
						BarColors:     barColors,

						Data1: left,
						Peak1: min(left+60, 1000),
						Data2: right,
						Peak2: min(right+60, 1000),
					},
				},
			})
		}
		time.Sleep(time.Millisecond * 50)
	}
	log.Println("Panel disconnected")
}

// imageType maps a display's topology "type" to the image type the meter is rendered in.
func imageType(dispType string) rwp.HWCGfx_ImageTypeE {
	switch dispType {
	case "color", "touch":
		return rwp.HWCGfx_RGB16bit
	case "gray":
		return rwp.HWCGfx_Gray4bit
	}
	return rwp.HWCGfx_MONO
}

// level makes up a lively audio level (0-1000) for the demo.
func level(t float64, phase float64) uint32 {
	v := 0.62 + 0.25*math.Sin(t*1.7+phase) + 0.12*math.Sin(t*7.3+phase*2)
	return uint32(math.Max(0, math.Min(1, v)) * 1000)
}
