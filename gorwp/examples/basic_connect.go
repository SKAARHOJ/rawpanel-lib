package main

import (
	"context"
	"fmt"

	gorwp "github.com/SKAARHOJ/rawpanel-lib/gorwp"
	log "github.com/s00500/env_logger"
)

func main() {

	ipAndPort := "192.168.11.8:9923"
	ipAndPort = "10.0.10.100:9923"

	// Connecting to the SKAARHOJ Raw Panel device:
	log.Printf("Trying to connect to panel on %s...\n", ipAndPort)
	ctx, cancel := context.WithCancel(context.Background())
	rp, err := gorwp.Connect(ipAndPort, ctx, cancel)

	if !log.Should(err) {
		log.Printf("Success - Connected to %s (%s, S/N %s)\n", rp.State.GetName(), rp.State.GetModel(), rp.State.GetSerial())

		// Make a simple binding on hardware component 1 (assuming hardware component 1 is a binary trigger such as a button)
		rp.BindBinary(1, func(hwc uint32, trigger gorwp.BinaryStatus, edge gorwp.BinaryEdge) {
			fmt.Println("Binary event: ", hwc, trigger, edge)
		})

		// Make a simple binding on hardware component 2 (assuming hardware component 1 is a binary trigger such as a button)
		rp.BindBinary(2, func(hwc uint32, trigger gorwp.BinaryStatus, edge gorwp.BinaryEdge) {
			fmt.Println("Binary event: ", hwc, trigger, edge)
		})

		select {
		case <-ctx.Done():
			log.Println("Panel disconnected")
			return
		}
	}
}
