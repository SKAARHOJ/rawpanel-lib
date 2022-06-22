/*
   Copyright 2022 SKAARHOJ ApS

   Released under MIT License
*/

// Package gorwp provides a Go interface for talking to a
// SKAARHOJ Raw Panel control surface.
//
// The SKAARHOJ Raw Panel appears as a network device, and we need to
// talk to it via TCP.

package gorwp

import (
	"bufio"
	"context"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"io"
	"net"
	"strings"
	"time"

	"google.golang.org/protobuf/proto"

	su "github.com/SKAARHOJ/ibeam-lib-utils"
	helpers "github.com/SKAARHOJ/rawpanel-lib"
	rwp "github.com/SKAARHOJ/rawpanel-lib/ibeam_rawpanel"
	topology "github.com/SKAARHOJ/rawpanel-lib/topology"
	log "github.com/s00500/env_logger"
)

// Type RawPanel describes a SKAARHOJ Raw Panel device
type RawPanel struct {
	connection  net.Conn
	cancel      *context.CancelFunc
	binaryPanel bool

	// Message channels:
	toPanel   chan []*rwp.InboundMessage
	fromPanel chan []*rwp.OutboundMessage

	// Trigger Bindings
	binaryBindings    map[uint32]BinaryFunc
	pulsedBindings    map[uint32]PulsedFunc
	absoluteBindings  map[uint32]AbsoluteFunc
	intensityBindings map[uint32]IntensityFunc
	triggerBindings   map[uint32]TriggerFunc

	// State
	State RawPanelState
}

// Connects to a SKAARHOJ Raw Panel at a specified URL. If successful it returns a new RawPanel
func Connect(panelIPAndPort string, ctx context.Context, cancel context.CancelFunc) (*RawPanel, error) {

	c, err := net.Dial("tcp", panelIPAndPort)
	if log.Should(err) {
		return nil, err
	}

	// Set up new raw panel, handshake and initialize:
	newRawPanel := &RawPanel{
		connection: c,
		cancel:     &cancel,
		toPanel:    make(chan []*rwp.InboundMessage, 10),
		fromPanel:  make(chan []*rwp.OutboundMessage, 10),

		binaryBindings:    make(map[uint32]BinaryFunc),
		pulsedBindings:    make(map[uint32]PulsedFunc),
		absoluteBindings:  make(map[uint32]AbsoluteFunc),
		intensityBindings: make(map[uint32]IntensityFunc),
		triggerBindings:   make(map[uint32]TriggerFunc),
	}
	newRawPanel.State.hwcAvailability = make(map[uint32]uint32)

	// Start listening:
	go newRawPanel.listen(ctx)

	// Try to initialize:
	err = newRawPanel.init(ctx)
	if log.Should(err) {
		c.Close()
		return nil, err
	}

	return newRawPanel, nil
}

// Closes a raw panel connection by calling the context cancel function
func (rp *RawPanel) Close() {
	(*rp.cancel)()
}

// Asking a panel for initial information:
func (rp *RawPanel) init(ctx context.Context) error {
	rp.binaryPanel = true

	// Sending request for various standard information from panel, all things we consider mandatory for initialization:
	rp.toPanel <- []*rwp.InboundMessage{&rwp.InboundMessage{
		Command: &rwp.Command{
			SendPanelInfo:         true,
			SendPanelTopology:     true,
			ReportHWCavailability: true,
			SetHeartBeatTimer: &rwp.HeartBeatTimer{
				Value: 3000,
			},
		},
	}}

	// Check for initialization, if we get it, return in channel:
	initialized := make(chan bool, 1)
	go func() {
		for {
			if rp.IsInitialized() {
				initialized <- true
				return
			}
			time.Sleep(time.Millisecond * 10)
		}
	}()

	// Wait for either signal that init was OK - or return after two seconds where it did not happen
	select {
	case <-ctx.Done():
		return nil
	case <-initialized:
		return nil
	case <-time.After(2 * time.Second):
		return fmt.Errorf("Panel did not respond to initialization timely")
	}
}

func (rp *RawPanel) listen(ctx context.Context) {

	// Listening for messages to/from panel
	go func() {
		ticker := time.NewTicker(1000 * time.Millisecond)
		for {
			select {
			case <-ctx.Done():
				fmt.Println("Stops listening for toPanel messages")
				return
			case messagesToPanel := <-rp.toPanel: // Messages from us to the panel.
				if rp.binaryPanel {
					for _, msg := range messagesToPanel {
						pbdata, _ := proto.Marshal(msg) // Encode data
						log.Debugln("System -> Panel: ", pbdata)

						header := make([]byte, 4)                                  // Create a 4-bytes header
						binary.LittleEndian.PutUint32(header, uint32(len(pbdata))) // Fill it in
						pbdata = append(header, pbdata...)                         // and concatenate it with the binary message
						rp.connection.Write(pbdata)
					}
				} else {
					lines := helpers.InboundMessagesToRawPanelASCIIstrings(messagesToPanel)
					for _, line := range lines {
						log.Println(string("System -> Panel: " + strings.TrimSpace(string(line))))
						rp.connection.Write([]byte(line + "\n"))
					}
				}
			case <-ticker.C: // Sending a ping periodically to the panel to make sure TCP will close connection if it doesn't get through. Strictly, the panel should answer back with ACK, but we don't check for that (seems this is enough)
				rp.toPanel <- []*rwp.InboundMessage{&rwp.InboundMessage{
					FlowMessage: rwp.InboundMessage_PING,
				}}
			case messagesFromPanel := <-rp.fromPanel:
				rp.procesMessagesFromPanel(messagesFromPanel)
			}
		}
	}()

	// Read from panel. This will send into the rp.fromPanel channel. It returns when there is an error:
	err := rp.readFromPanel()
	log.Should(err)

	rp.connection.Close()
	(*rp.cancel)()
}

func (rp *RawPanel) readFromPanel() error {
	// Reading from panel:
	if rp.binaryPanel {
		for {
			rp.connection.SetReadDeadline(time.Time{}) // Reset deadline, waiting for header
			headerArray := make([]byte, 4)
			_, err := io.ReadFull(rp.connection, headerArray) // Read 4 header bytes
			if err != nil {
				if err == io.EOF {
					log.Errorln("Panel: " + rp.connection.RemoteAddr().String() + " disconnected")
				} else {
					log.Errorln("Binary: ", err)
				}
				return err
			} else {
				currentPayloadLength := binary.LittleEndian.Uint32(headerArray[0:4])
				if currentPayloadLength < 500000 {
					payload := make([]byte, currentPayloadLength)
					rp.connection.SetReadDeadline(time.Now().Add(2 * time.Second)) // Set a deadline that we want all data within at most 2 seconds. This helps a run-away scenario where not all data arrives or we read the wront (and too big) header
					_, err := io.ReadFull(rp.connection, payload)
					if err != nil {
						log.Errorln(err)
						break
					} else {
						outgoingMessage := &rwp.OutboundMessage{}
						proto.Unmarshal(payload, outgoingMessage)
						if outgoingMessage.FlowMessage != 2 { // ack
							rp.fromPanel <- []*rwp.OutboundMessage{outgoingMessage}
						}
					}
				} else {
					log.Println("Error: Payload", currentPayloadLength, "exceed limit")
				}
			}
		}
	} else {
		connectionReader := bufio.NewReader(rp.connection) // Define OUTSIDE the for loop
		for {
			netData, err := connectionReader.ReadString('\n')
			if err != nil {
				if err == io.EOF {
					log.Errorln("Panel: " + rp.connection.RemoteAddr().String() + " disconnected")
					time.Sleep(time.Second)
				} else {
					log.Errorln(err)
				}
				return err
			} else {
				netDataStr := strings.TrimSpace(netData)
				switch netDataStr {
				case "ack":
				default:
					rp.fromPanel <- helpers.RawPanelASCIIstringsToOutboundMessages([]string{netDataStr})
				}
			}
		}
	}

	return nil
}

func (rp *RawPanel) procesMessagesFromPanel(messagesFromPanel []*rwp.OutboundMessage) {
	rp.State.Lock()
	for _, msg := range messagesFromPanel {

		// Respond to ping:
		if msg.FlowMessage == rwp.OutboundMessage_PING {
			rp.toPanel <- []*rwp.InboundMessage{&rwp.InboundMessage{
				FlowMessage: rwp.InboundMessage_ACK,
			}}
		}

		// Store panel info:
		if msg.PanelInfo != nil {
			if msg.PanelInfo.Model != "" {
				rp.State.model = msg.PanelInfo.Model
				log.Debugln("Model:", msg.PanelInfo.Model)
			}
			if msg.PanelInfo.Serial != "" {
				rp.State.serial = msg.PanelInfo.Serial
				log.Debugln("Serial:", msg.PanelInfo.Serial)
			}
			if msg.PanelInfo.Name != "" {
				rp.State.name = msg.PanelInfo.Name
				log.Debugln("Name:", msg.PanelInfo.Name)
			}
		}

		// Panel availability:
		if msg.HWCavailability != nil {
			for k, v := range msg.HWCavailability {
				rp.State.hwcAvailability[k] = v
			}
		}

		// Topology:
		if msg.PanelTopology != nil { // Receiving topology
			if msg.PanelTopology.Json != "" {
				rp.State.topologyJSON = msg.PanelTopology.Json
				rp.State.topology = &topology.Topology{}
				err := json.Unmarshal([]byte(rp.State.topologyJSON), rp.State.topology)
				if err != nil {
					log.Errorln("Topology JSON parsing Error: ", err)
				} else {
					log.Debugln(log.Indent(rp.State.topology))
				}
			}
			if msg.PanelTopology.Svgbase != "" {
				rp.State.topologySVG = msg.PanelTopology.Svgbase
				log.Debugln("Received Topology SVG")
			}
		}

		// Events:
		if len(msg.Events) > 0 {
			for _, event := range msg.Events {
				if receiverFunc, exists := rp.triggerBindings[event.HWCID]; exists {
					receiverFunc(event.HWCID, event)
				}
				if receiverFunc, exists := rp.binaryBindings[event.HWCID]; exists && event.Binary != nil {
					receiverFunc(event.HWCID, BinaryStatus(su.Qint(event.Binary.Pressed, 1, 0)), BinaryEdge(event.Binary.Edge))
				}
				if receiverFunc, exists := rp.pulsedBindings[event.HWCID]; exists && event.Pulsed != nil {
					receiverFunc(event.HWCID, int(event.Pulsed.Value))
				}
				if receiverFunc, exists := rp.absoluteBindings[event.HWCID]; exists && event.Absolute != nil {
					receiverFunc(event.HWCID, int(event.Absolute.Value))
				}
				if receiverFunc, exists := rp.intensityBindings[event.HWCID]; exists && event.Speed != nil {
					receiverFunc(event.HWCID, int(event.Speed.Value))
				}
			}
		}
	}
	rp.State.Unlock()
}

func (rp *RawPanel) IsInitialized() bool {
	rp.State.Lock()
	defer rp.State.Unlock()
	if rp.State.model != "" &&
		rp.State.name != "" &&
		rp.State.serial != "" &&
		rp.State.topologyJSON != "" &&
		rp.State.topologySVG != "" {
		return true
	}
	return false
}

// Sets the panel brightness (same for OLED and LEDs in this case)
func (rp *RawPanel) SetBrightness(brightness int) {
	rp.toPanel <- []*rwp.InboundMessage{
		&rwp.InboundMessage{
			Command: &rwp.Command{
				PanelBrightness: &rwp.Brightness{
					OLEDs: uint32(brightness),
					LEDs:  uint32(brightness),
				},
			},
		},
	}
}

// Sets the color of a specific LED.
func (rp *RawPanel) SetLEDColor(hwc uint32, c color.RGBA, intensity rwp.HWCMode_StateE) {
	r, g, b, _ := c.RGBA()
	rp.toPanel <- []*rwp.InboundMessage{
		&rwp.InboundMessage{
			States: []*rwp.HWCState{
				&rwp.HWCState{
					HWCIDs: []uint32{hwc},
					HWCMode: &rwp.HWCMode{
						State: rwp.HWCMode_StateE(intensity),
					},
					HWCColor: &rwp.HWCColor{
						ColorRGB: &rwp.ColorRGB{
							Red:   uint32(r >> 8),
							Green: uint32(g >> 8),
							Blue:  uint32(b >> 8),
						},
					},
				},
			},
		},
	}
}

// Sets the color of a specific LED by index
func (rp *RawPanel) SetLEDColorByIndex(hwc uint32, colorIndex rwp.ColorIndex_Colors, intensity rwp.HWCMode_StateE) {
	rp.toPanel <- []*rwp.InboundMessage{
		&rwp.InboundMessage{
			States: []*rwp.HWCState{
				&rwp.HWCState{
					HWCIDs: []uint32{hwc},
					HWCMode: &rwp.HWCMode{
						State: rwp.HWCMode_StateE(intensity),
					},
					HWCColor: &rwp.HWCColor{
						ColorIndex: &rwp.ColorIndex{
							Index: colorIndex,
						},
					},
				},
			},
		},
	}
}

// Sets the raw panel ASCII text of a display (text lines and header type)
func (rp *RawPanel) SetRWPText(hwc uint32, title string, text1 string, text2 string, headerBar bool) {
	txtStruct := &rwp.HWCText{
		Title:          title,
		Formatting:     7,
		Textline1:      text1,
		Textline2:      text2,
		SolidHeaderBar: headerBar,
		PairMode:       rwp.HWCText_PairModeE(su.Qint(text2 != "", 1, 0)),
	}
	rp.SetRWPTextByStruct(hwc, txtStruct)
}

// Sets the raw panel ASCII text of a display by forwarding a full text struct
func (rp *RawPanel) SetRWPTextByStruct(hwc uint32, txtStruct *rwp.HWCText) {
	rp.toPanel <- []*rwp.InboundMessage{
		&rwp.InboundMessage{
			States: []*rwp.HWCState{
				&rwp.HWCState{
					HWCIDs:  []uint32{hwc},
					HWCText: txtStruct,
				},
			},
		},
	}
}

// Function Draw draws an image onto a specific display of the
// SKAARHOJ Raw Panel.
func (rp *RawPanel) DrawImage(hwc uint32, im image.Image, xoff, yoff int) {

}
