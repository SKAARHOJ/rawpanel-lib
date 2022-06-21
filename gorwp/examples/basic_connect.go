package main

import (
	"context"
	"fmt"

	gorwp "github.com/SKAARHOJ/rawpanel-lib/gorwp"
	log "github.com/s00500/env_logger"
)

func main() {

	ipAndPort := "192.168.11.8:9923"

	// Connecting to the SKAARHOJ Raw Panel device:
	log.Printf("Trying to connect to panel on %s...\n", ipAndPort)
	ctx, cancel := context.WithCancel(context.Background())
	rp, err := gorwp.Connect(ipAndPort, ctx, cancel)

	if !log.Should(err) {
		log.Printf("Success - Connected to %s (%s, S/N %s)\n", rp.State.GetName(), rp.State.GetModel(), rp.State.GetSerial())

		top := rp.State.GetTopology() // The topology is a structure that describes the panels features.

		for _, hwc := range top.GetHWCs() { // Loop over all hardware components (HWCs) on the panel
			typeDef, _ := top.GetHWCtype(hwc) // Type Definition tells us what type this HWC is

			// Register any trigger type, providing the Raw Panel event structure to work with:
			/*
				rp.BindTrigger(hwc, func(hwc uint32, event *rwp.HWCEvent) {
					fmt.Println("Any type of event: ", hwc, event)
				})
			*/

			// If HWC has a binary trigger (buttons, gpis and encoders with push), register it here:
			if typeDef.IsBinary() {
				rp.BindBinary(hwc, func(hwc uint32, trigger gorwp.BinaryStatus, edge gorwp.BinaryEdge) {
					fmt.Println("Binary event: ", hwc, trigger, edge)
				})
			}

			// If HWC is a pulsed type (encoders), provide this call back:
			if typeDef.IsPulsed() {
				rp.BindPulsed(hwc, func(hwc uint32, direction int) {
					fmt.Println("Pulsed event: ", hwc, direction)
				})
			}

			// If HWC is an absolute position type (analog faders, T-bars), provide this call back:
			if typeDef.IsAbsolute() {
				rp.BindAbsolute(hwc, func(hwc uint32, position int) {
					fmt.Println("Absolute event: ", hwc, position)
				})
			}

			// If HWC is an intensity type (joysticks, shuttle wheels with spring loaded fall back), provide this call back:
			if typeDef.IsIntensity() {
				rp.BindIntensity(hwc, func(hwc uint32, intensity int) {
					fmt.Println("Intensity event: ", hwc, intensity)
				})
			}
		}

		select {
		case <-ctx.Done():
			log.Println("Panel disconnected")
			return
		}
	}
}
