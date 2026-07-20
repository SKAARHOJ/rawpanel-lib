package main

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"time"

	gorwp "github.com/SKAARHOJ/rawpanel-lib/gorwp"
	rwp "github.com/SKAARHOJ/rawpanel-lib/ibeam_rawpanel"
	log "github.com/s00500/env_logger"
)

// This application exemplifies event handlers in a broader sense and how you can send feedback to the panel
func main() {

	ipAndPort := "localhost:9923" // Change this to the IP and port of your SKAARHOJ panel

	// Connecting to the SKAARHOJ Raw Panel device:
	log.Printf("Trying to connect to panel on %s...\n", ipAndPort)
	ctx, cancel := context.WithCancel(context.Background())
	rp, err := gorwp.Connect(ipAndPort, ctx, cancel)

	if !log.Should(err) {
		log.Printf("Success - Connected to %s (%s, S/N %s)\n", rp.State.GetName(), rp.State.GetModel(), rp.State.GetSerial())

		// Set max brightness
		rp.SetBrightness(8)

		// The topology is a structure that describes the panels features.
		top := rp.State.GetTopology()

		// Loop over all hardware components (HWCs) on the panel
		inc := 0
		for {
			for _, hwc := range top.GetHWCs() {
				//if hwc >= 21 && hwc <= 24 {
				typeDef, _ := top.GetHWCtype(hwc) // Type Definition tells us what type this HWC is
				displayInfo := typeDef.DisplayInfo()
				if displayInfo != nil && displayInfo.W > 0 && displayInfo.H > 0 {

					state := &rwp.HWCState{
						HWCIDs: []uint32{hwc},
						Processors: &rwp.Processors{
							/*
								TextToGraphics: &rwp.ProcTextToGraphics{
									W: uint32(displayInfo.W),
									H: uint32(displayInfo.H),
								},
							*/
							AudioMeter: &rwp.ProcAudioMeter{
								MeterType: rwp.ProcAudioMeter_Fixed176x32w,
								W:         uint32(displayInfo.W),
								H:         uint32(displayInfo.H),
								Title:     "Hey!",
								Mono:      false,
								Data1:     uint32(rand.Intn(1001)),
								Peak1:     uint32(rand.Intn(1001)),
								Data2:     uint32(rand.Intn(1001)),
								Peak2:     uint32(rand.Intn(1001)),
							},
							/*Test: &rwp.ProcTest{
								W: uint32(displayInfo.W),
								H: uint32(displayInfo.H),
							},*/
							/*
								StrengthMeter: &rwp.ProcStrength{
									W:     uint32(displayInfo.W),
									H:     uint32(displayInfo.H),
									Title: "Hey!",
									//ValueString: "100.123%",
									RangeMapping: "-300,1700",
									RMYAxis:      true,
									Data1:        uint32(inc % 1000),
								},
							*/
							/*UniText: &rwp.ProcUniText{
								W:         uint32(displayInfo.W),
								H:         uint32(displayInfo.H),
								Title:     "ÆØÅ你好jÄ",
								Textline1: "ÆØÅ你好jÄ",
								//Textline2:      "ÆØÅ你好jÄ",
								SolidHeaderBar: true,
							},*/
						},
						/*
							HWCText: &rwp.HWCText{
								Title:          "Hey!",
								SolidHeaderBar: true,
								Textline1:      fmt.Sprintf("Count: %d", inc),
							},
						*/
					}

					jsonData, _ := json.Marshal(state)
					_ = jsonData
					fmt.Println(string(jsonData))

					rp.SendRawState(state)
				}
				//}
			}
			inc += 10
			time.Sleep(time.Millisecond * 200)
		}

		select {
		case <-ctx.Done():
			log.Println("Panel disconnected")
			return
		}
	}
}
