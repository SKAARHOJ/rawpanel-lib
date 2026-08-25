package rawpanellib

import (
	"fmt"
	"testing"

	rwp "github.com/SKAARHOJ/rawpanel-lib/ibeam_rawpanel"
	log "github.com/s00500/env_logger"
	"google.golang.org/protobuf/proto"
)

// A maximal TouchUI config exercising all widget types, both layout modes and options:
func maximalTouchUIConfig() *rwp.TouchUIConfig {
	return &rwp.TouchUIConfig{
		Title:      "Test Config",
		ActivePage: 1,
		Pages: []*rwp.TouchUIPage{
			{
				Id:       1,
				Title:    "Main",
				GridRows: 2,
				GridCols: 3,
				Widgets: []*rwp.TouchUIWidget{
					{
						HWCID: 201,
						Type:  rwp.TouchUIWidget_BUTTON,
						Label: "Cut",
						Row:   1,
						Col:   1,
					},
					{
						HWCID: 202,
						Type:  rwp.TouchUIWidget_TOGGLE,
						Label: "On Air",
						Row:   1,
						Col:   2,
						Options: &rwp.TouchUIWidgetOptions{
							Momentary: true,
						},
					},
					{
						HWCID:   203,
						Type:    rwp.TouchUIWidget_SLIDER,
						Label:   "Level",
						Row:     1,
						Col:     3,
						RowSpan: 2,
						Options: &rwp.TouchUIWidgetOptions{
							Min:      0,
							Max:      1000,
							Step:     10,
							Vertical: true,
						},
					},
					{
						HWCID:     204,
						Type:      rwp.TouchUIWidget_KNOB,
						Label:     "Gain",
						Row:       2,
						Col:       1,
						EventMask: TouchUIEventPulsed | TouchUIEventBinary,
					},
					{
						HWCID: 205,
						Type:  rwp.TouchUIWidget_METER,
						Label: "VU",
						Row:   2,
						Col:   2,
					},
				},
			},
			{
				Id:    2,
				Title: "Aux",
				Widgets: []*rwp.TouchUIWidget{
					{
						HWCID: 206,
						Type:  rwp.TouchUIWidget_LABEL,
						Label: "Status",
						X:     0,
						Y:     0,
						W:     320,
						H:     60,
					},
					{
						HWCID: 207,
						Type:  rwp.TouchUIWidget_IMAGE,
						X:     0,
						Y:     60,
						W:     320,
						H:     180,
						Options: &rwp.TouchUIWidgetOptions{
							NoTapEvents: true,
							Color: &rwp.Color{
								ColorIndex: &rwp.ColorIndex{
									Index: rwp.ColorIndex_RED,
								},
							},
						},
					},
					{
						HWCID: 208,
						Type:  rwp.TouchUIWidget_VIDEO,
						Label: "Cam 1",
						X:     320,
						Y:     0,
						W:     640,
						H:     240,
						Options: &rwp.TouchUIWidgetOptions{
							Source:       "http://127.0.0.1:8090/stream.mjpg",
							HiddenPolicy: rwp.TouchUIWidgetOptions_PAUSE_DECODE,
							SourceWidth:  1280,
							SourceHeight: 720,
						},
					},
				},
			},
		},
	}
}

func TestTouchUIInbound(t *testing.T) {
	configJSON := touchUIStringFromConfig(maximalTouchUIConfig())

	var tests = []struct {
		give, want []string
	}{
		{
			[]string{"ClearTouchUI"},
			[]string{"ClearTouchUI"},
		},
		{
			[]string{"TouchUICapabilities?"},
			[]string{"TouchUICapabilities?"},
		},
		{
			[]string{"TouchUIConfig?"},
			[]string{"TouchUIConfig?"},
		},
		{
			[]string{"SetTouchUI=" + configJSON},
			[]string{"SetTouchUI=" + configJSON},
		},
	}

	for i, tt := range tests {
		testname := fmt.Sprintf("TestTouchUIInbound%d", i)
		t.Run(testname, func(t *testing.T) {
			protobufObj := RawPanelASCIIstringsToInboundMessages(tt.give)
			roundtrip := InboundMessagesToRawPanelASCIIstrings(protobufObj)

			if len(roundtrip) != len(tt.want) {
				log.Println(log.Indent(protobufObj))
				t.Errorf("Round trip %v didn't match wanted %v", roundtrip, tt.want)
			} else {
				for i := range roundtrip {
					if roundtrip[i] != tt.want[i] {
						log.Println(log.Indent(protobufObj))
						t.Errorf("Round trip %v didn't match wanted %v", roundtrip, tt.want)
						continue
					}
				}
			}
		})
	}
}

func TestTouchUIInboundFromBinary(t *testing.T) {
	config := maximalTouchUIConfig()

	var tests = []struct {
		give []*rwp.InboundMessage
		want []string
	}{
		{
			[]*rwp.InboundMessage{
				{
					Command: &rwp.Command{
						SetTouchUI: config,
					},
				},
			},
			[]string{"SetTouchUI=" + touchUIStringFromConfig(config)},
		},
		{
			[]*rwp.InboundMessage{
				{
					Command: &rwp.Command{
						ClearTouchUI:            true,
						SendTouchUICapabilities: true,
						SendTouchUIConfig:       true,
					},
				},
			},
			[]string{"ClearTouchUI", "TouchUICapabilities?", "TouchUIConfig?"},
		},
	}

	for i, tt := range tests {
		testname := fmt.Sprintf("TestTouchUIInboundFromBinary%d", i)
		t.Run(testname, func(t *testing.T) {
			ASCIIstrings := InboundMessagesToRawPanelASCIIstrings(tt.give)

			if len(ASCIIstrings) != len(tt.want) {
				log.Println(log.Indent(tt.give))
				t.Errorf("Round trip %v didn't match wanted %v", ASCIIstrings, tt.want)
			} else {
				for i := range ASCIIstrings {
					if ASCIIstrings[i] != tt.want[i] {
						log.Println(log.Indent(tt.give))
						t.Errorf("Round trip %v didn't match wanted %v", ASCIIstrings, tt.want)
						continue
					}
				}
			}
		})
	}
}

func TestTouchUIOutbound(t *testing.T) {
	capabilities := &rwp.TouchUICapabilities{
		ScreenWidth:  320,
		ScreenHeight: 240,
		MaxPages:     4,
		MaxWidgets:   32,
		GridRows:     3,
		GridCols:     4,
		WidgetTypes: []*rwp.TouchUIWidgetTypeCap{
			{
				Type:      rwp.TouchUIWidget_BUTTON,
				EventMask: TouchUIEventBinary,
				StateMask: TouchUIStateMode | TouchUIStateColor | TouchUIStateText | TouchUIStateGfx,
			},
			{
				Type:      rwp.TouchUIWidget_SLIDER,
				EventMask: TouchUIEventAbsolute,
				StateMask: TouchUIStateText | TouchUIStateExtended,
			},
		},
	}
	capabilitiesJSON := touchUIStringFromCapabilities(capabilities)
	configJSON := touchUIStringFromConfig(maximalTouchUIConfig())

	var tests = []struct {
		give, want []string
	}{
		{
			[]string{"_touchUICapabilities=" + capabilitiesJSON},
			[]string{"_touchUICapabilities=" + capabilitiesJSON},
		},
		{
			[]string{"_touchUIConfig=" + configJSON},
			[]string{"_touchUIConfig=" + configJSON},
		},
		{
			[]string{"_touchUIActivePage=3"},
			[]string{"_touchUIActivePage=3"},
		},
		{
			[]string{"_support=ASCII,Binary,TouchUI"},
			[]string{"_support=ASCII,Binary,TouchUI"},
		},

		// Availability map values, including the bit-31 offscreen flag (0x80000000|201 = 2147483849):
		{
			[]string{"map=5:0"},
			[]string{"map=5:0"},
		},
		{
			[]string{"map=5:1"},
			[]string{"map=5:1"},
		},
		{
			[]string{"map=201:201"},
			[]string{"map=201:201"},
		},
		{
			[]string{"map=201:2147483849"},
			[]string{"map=201:2147483849"},
		},

		// Text events (editable LABEL commits). The value is the rest of the line,
		// so nothing needs escaping — that is the whole point of the encoding.
		{
			[]string{"HWC#100=Text:hello world"},
			[]string{"HWC#100=Text:hello world"},
		},
		{
			// Empty is legal: the user cleared the field. Must not be dropped.
			[]string{"HWC#100=Text:"},
			[]string{"HWC#100=Text:"},
		},
		{
			// The delimiters that break every other line format survive verbatim.
			[]string{"HWC#100=Text:a,b:c=d"},
			[]string{"HWC#100=Text:a,b:c=d"},
		},
		{
			[]string{"HWC#100=Text:192.168.1.10"},
			[]string{"HWC#100=Text:192.168.1.10"},
		},
		{
			// Leading/trailing spaces belong to the value (a password may end in one).
			[]string{"HWC#100=Text:  padded  "},
			[]string{"HWC#100=Text:  padded  "},
		},

		// Regression guards: adding Text: must not have widened the numeric forms.
		{
			[]string{"HWC#100=Abs:57"}, // scalar stays scalar, not a 1-element vector
			[]string{"HWC#100=Abs:57"},
		},
		{
			[]string{"HWC#100=Abs:250,750"}, // XYPAD position
			[]string{"HWC#100=Abs:250,750"},
		},
		{
			[]string{"HWC#100=Speed:-3,4"}, // XYPAD relative movement
			[]string{"HWC#100=Speed:-3,4"},
		},
	}

	for i, tt := range tests {
		testname := fmt.Sprintf("TestTouchUIOutbound%d", i)
		t.Run(testname, func(t *testing.T) {
			protobufObj := RawPanelASCIIstringsToOutboundMessages(tt.give)
			roundtrip := OutboundMessagesToRawPanelASCIIstrings(protobufObj)

			if len(roundtrip) != len(tt.want) {
				log.Println(log.Indent(protobufObj))
				t.Errorf("Round trip %v didn't match wanted %v", roundtrip, tt.want)
			} else {
				for i := range roundtrip {
					if roundtrip[i] != tt.want[i] {
						log.Println(log.Indent(protobufObj))
						t.Errorf("Round trip %v didn't match wanted %v", roundtrip, tt.want)
						continue
					}
				}
			}
		})
	}
}

func TestTouchUIOutboundFromBinary(t *testing.T) {
	capabilities := &rwp.TouchUICapabilities{
		ScreenWidth:  320,
		ScreenHeight: 240,
	}
	config := maximalTouchUIConfig()

	var tests = []struct {
		give []*rwp.OutboundMessage
		want []string
	}{
		{
			[]*rwp.OutboundMessage{
				{
					TouchUICapabilities: capabilities,
				},
			},
			[]string{"_touchUICapabilities=" + touchUIStringFromCapabilities(capabilities)},
		},
		{
			[]*rwp.OutboundMessage{
				{
					TouchUIConfig: config,
				},
			},
			[]string{"_touchUIConfig=" + touchUIStringFromConfig(config)},
		},
		{
			[]*rwp.OutboundMessage{
				{
					TouchUIActivePage: &rwp.TouchUIActivePage{PageId: 3},
				},
			},
			[]string{"_touchUIActivePage=3"},
		},
		{
			[]*rwp.OutboundMessage{
				{
					HWCavailability: map[uint32]uint32{201: HWCAvailabilityOffscreenFlag | 201},
				},
			},
			[]string{"map=201:2147483849"},
		},
	}

	for i, tt := range tests {
		testname := fmt.Sprintf("TestTouchUIOutboundFromBinary%d", i)
		t.Run(testname, func(t *testing.T) {
			ASCIIstrings := OutboundMessagesToRawPanelASCIIstrings(tt.give)

			if len(ASCIIstrings) != len(tt.want) {
				log.Println(log.Indent(tt.give))
				t.Errorf("Round trip %v didn't match wanted %v", ASCIIstrings, tt.want)
			} else {
				for i := range ASCIIstrings {
					if ASCIIstrings[i] != tt.want[i] {
						log.Println(log.Indent(tt.give))
						t.Errorf("Round trip %v didn't match wanted %v", ASCIIstrings, tt.want)
						continue
					}
				}
			}
		})
	}
}

// Full fidelity: maximal config ASCII -> proto -> ASCII -> proto, compared with proto.Equal
func TestTouchUIConfigFullFidelity(t *testing.T) {
	original := maximalTouchUIConfig()

	line := "SetTouchUI=" + touchUIStringFromConfig(original)
	messages := RawPanelASCIIstringsToInboundMessages([]string{line})
	if len(messages) != 1 || messages[0].Command == nil || messages[0].Command.SetTouchUI == nil {
		t.Fatalf("Parsing %s did not yield a SetTouchUI command", line)
	}
	if !proto.Equal(original, messages[0].Command.SetTouchUI) {
		t.Errorf("Config after ASCII parse doesn't match original.\nOriginal: %v\nParsed: %v", original, messages[0].Command.SetTouchUI)
	}

	roundtrip := InboundMessagesToRawPanelASCIIstrings(messages)
	if len(roundtrip) != 1 || roundtrip[0] != line {
		t.Errorf("ASCII round trip %v didn't match original line %s", roundtrip, line)
	}

	messages2 := RawPanelASCIIstringsToInboundMessages(roundtrip)
	if len(messages2) != 1 || !proto.Equal(original, messages2[0].Command.SetTouchUI) {
		t.Errorf("Config after second ASCII round trip doesn't match original")
	}
}

func TestHWCAvailabilityHelpers(t *testing.T) {
	var tests = []struct {
		value    uint32
		present  bool
		onscreen bool
		mappedTo uint32
	}{
		{0, false, false, 0},
		{1, true, true, 1},
		{201, true, true, 201},
		{HWCAvailabilityOffscreenFlag, true, false, 0},
		{HWCAvailabilityOffscreenFlag | 201, true, false, 201},
	}

	for _, tt := range tests {
		if HWCAvailabilityPresent(tt.value) != tt.present {
			t.Errorf("HWCAvailabilityPresent(%d) should be %v", tt.value, tt.present)
		}
		if HWCAvailabilityOnscreen(tt.value) != tt.onscreen {
			t.Errorf("HWCAvailabilityOnscreen(%d) should be %v", tt.value, tt.onscreen)
		}
		if HWCAvailabilityMappedTo(tt.value) != tt.mappedTo {
			t.Errorf("HWCAvailabilityMappedTo(%d) should be %d", tt.value, tt.mappedTo)
		}
	}
}
