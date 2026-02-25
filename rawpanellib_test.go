package rawpanellib

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/SKAARHOJ/rawpanel-lib/ibeam_rawpanel"
	log "github.com/s00500/env_logger"
)

func TestOutbound(t *testing.T) {
	var tests = []struct {
		give, want []string
	}{
		// Testing panel type:
		{
			[]string{"_panelType=BPI"},
			[]string{"_panelType=BPI"},
		},
		{
			[]string{"_panelType=Physical"},
			[]string{"_panelType=Physical"},
		},
		{
			[]string{"_panelType=Emulation"},
			[]string{"_panelType=Emulation"},
		},
		{
			[]string{"_panelType=Touch"},
			[]string{"_panelType=Touch"},
		},
		{
			[]string{"_panelType=Composite"},
			[]string{"_panelType=Composite"},
		},

		// Testing Support
		{
			[]string{"_support=ASCII,Binary,JSONFeedback,JSONonInbound,JSONonOutbound,System,RawADCValues,BurninProfile,EnvHealth,Registers,Calibration"},
			[]string{"_support=ASCII,Binary,JSONFeedback,JSONonInbound,JSONonOutbound,System,RawADCValues,BurninProfile,EnvHealth,Registers,Calibration"},
		},
		{
			[]string{"_support=Registers,Binary,EnvHealth,JSONonInbound,System,RawADCValues,Calibration,JSONFeedback"},
			[]string{"_support=Binary,JSONFeedback,JSONonInbound,System,RawADCValues,EnvHealth,Registers,Calibration"},
		},

		// Testing JSON strings
		{
			[]string{`_burninProfile= test1 `},
			[]string{"_burninProfile=test1"},
		},
		{
			[]string{"_calibrationProfile= test1 "},
			[]string{"_calibrationProfile=test1"},
		},
		{
			[]string{"_defaultCalibrationProfile= test1 "},
			[]string{"_defaultCalibrationProfile=test1"},
		},
		// Testing system Stats:
		{
			[]string{"SysStat=CPUUsage:4:CPUTemp:56.0:ExtTemp:-100.0:CPUVoltage:0.85:CPUFreqCurrent:-1500000:CPUFreqMin:-1400000:CPUFreqMax:-1300000:MemTotal:-1893788:MemFree:-1637268:MemAvailable:-1750128:MemBuffers:-6004:MemCached:-120080:UnderVoltageNow:1:UnderVoltage:0:FreqCapNow:0:FreqCap:1:ThrottledNow:0:Throttled:1:SoftTempLimitNow:1:SoftTempLimit:0:"},
			[]string{"SysStat=CPUUsage:4:CPUTemp:56.0:ExtTemp:-100.0:CPUVoltage:0.85:CPUFreqCurrent:-1500000:CPUFreqMin:-1400000:CPUFreqMax:-1300000:MemTotal:-1893788:MemFree:-1637268:MemAvailable:-1750128:MemBuffers:-6004:MemCached:-120080:UnderVoltageNow:1:UnderVoltage:0:FreqCapNow:0:FreqCap:1:ThrottledNow:0:Throttled:1:SoftTempLimitNow:1:SoftTempLimit:0:"},
		},

		// Environmental Health:
		{
			[]string{"EnvironmentalHealth=Normal"},
			[]string{"EnvironmentalHealth=Normal"},
		},
		{
			[]string{"EnvironmentalHealth=Safemode"},
			[]string{"EnvironmentalHealth=Safemode"},
		},
		{
			[]string{"EnvironmentalHealth=Blocked"},
			[]string{"EnvironmentalHealth=Blocked"},
		},

		// Register
		{
			[]string{"Mem=33", "MemA=12", "MemBB=355", "MemZ3Z=33"},
			[]string{"Mem=33", "MemA=12", "MemBB=355", "MemZ3Z=33"},
		},
		{
			[]string{"Flag#64=1", "Flag#=0", "Flag#64=234"},
			[]string{"Flag#64=1", "Flag#0=0", "Flag#64=1"},
		},
		{
			[]string{"Shift=5", "ShiftA=15", "ShiftA=52", "ShiftA=544", "ShiftG=21"},
			[]string{"Shift=5", "ShiftA=15", "ShiftA=52", "ShiftA=544", "ShiftG=21"},
		},
		{
			[]string{"State=5", "StateA=15", "StateA=52", "StateA=544", "StateG=21"},
			[]string{"State=5", "StateA=15", "StateA=52", "StateA=544", "StateG=21"},
		},
		// Testing Support with TextScroll
		{
			[]string{"_support=ASCII,Binary,TextScroll"},
			[]string{"_support=ASCII,Binary,TextScroll"},
		},
	}

	for i, tt := range tests {
		testname := fmt.Sprintf("TestOutbound%d", i)
		t.Run(testname, func(t *testing.T) {
			protobufObj := RawPanelASCIIstringsToOutboundMessages(tt.give)
			//log.Println(log.Indent(protobufObj))
			roundtrip := OutboundMessagesToRawPanelASCIIstrings(protobufObj)
			//log.Println(roundtrip)

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

func TestOutboundFromBinary(t *testing.T) {
	var tests = []struct {
		give []*ibeam_rawpanel.OutboundMessage
		want []string
	}{
		// Testing JSON strings
		{
			[]*ibeam_rawpanel.OutboundMessage{
				{
					BurninProfile: &ibeam_rawpanel.BurninProfile{
						Json: `
						[
							{
							 "Test": {
							  "Json": " TEST "
							 }
							}
						   ]
						   `,
					},
				},
			},
			[]string{`_burninProfile=[{"Test": {"Json": " TEST "}}]`},
		},
		{
			[]*ibeam_rawpanel.OutboundMessage{
				{
					BurninProfile:             &ibeam_rawpanel.BurninProfile{},
					CalibrationProfile:        &ibeam_rawpanel.CalibrationProfile{},
					DefaultCalibrationProfile: &ibeam_rawpanel.CalibrationProfile{},
				},
			},
			[]string{`_burninProfile=`, `_calibrationProfile=`, `_defaultCalibrationProfile=`},
		},
	}

	for i, tt := range tests {
		testname := fmt.Sprintf("TestOutboundFromBinary%d", i)
		t.Run(testname, func(t *testing.T) {
			ASCIIstrings := OutboundMessagesToRawPanelASCIIstrings(tt.give)
			//log.Println(ASCIIstrings)

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

func TestInbound(t *testing.T) {
	var tests = []struct {
		give, want []string
	}{
		// Testing panel type:
		{
			[]string{"CalibrationProfile?"},
			[]string{"CalibrationProfile?"},
		},
		{
			[]string{"SetCalibrationProfile= (JSON) "},
			[]string{"SetCalibrationProfile=(JSON)"},
		},
		{
			[]string{"SimulateEnvironmentalHealth=Normal"},
			[]string{"SimulateEnvironmentalHealth=Normal"},
		},
		{
			[]string{"SimulateEnvironmentalHealth=Safemode"},
			[]string{"SimulateEnvironmentalHealth=Safemode"},
		},
		{
			[]string{"SimulateEnvironmentalHealth=Blocked"},
			[]string{"SimulateEnvironmentalHealth=Blocked"},
		},
		{
			[]string{"JSONonOutbound=1"},
			[]string{"JSONonOutbound=1"},
		},
		{
			[]string{"JSONonOutbound=0"},
			[]string{"JSONonOutbound=0"},
		},

		// TextScroll: loop mode, scroll title + textline1 (1 | 4 | 8 = 13)
		{
			[]string{"HWCt#5=|||My Title||Long text here|||||||||13"},
			[]string{"HWCt#5=|||My Title||Long text here|||||||||13"},
		},
		// TextScroll: bounce mode, scroll textline2, speed=2, dwell=1 (2 | 16 | 64 | 128 = 210)
		{
			[]string{"HWCt#10=|||Header||Line1|Line2||1||||||210"},
			[]string{"HWCt#10=|||Header||Line1|Line2||1||||||210"},
		},
		// TextScroll: bounce mode, all fields, speed=3, dwell=3 (2 | 4 | 8 | 16 | 96 | 384 = 510)
		{
			[]string{"HWCt#1=|||Title||T1|T2||1||||||510"},
			[]string{"HWCt#1=|||Title||T1|T2||1||||||510"},
		},
		// TextScroll: loop mode, all fields, adaptive speed (1 | 4 | 8 | 16 | 512 = 541)
		{
			[]string{"HWCt#3=|||Title||T1|T2||1||||||541"},
			[]string{"HWCt#3=|||Title||T1|T2||1||||||541"},
		},

		// Register
		{
			[]string{"Registers?"},
			[]string{"Registers?"},
		},
		{
			[]string{"Mem=33", "MemA=12", "MemBB=355", "MemZ3Z=33"},
			[]string{"Mem=33", "MemA=12", "MemBB=355", "MemZ3Z=33"},
		},
		{
			[]string{"Flag#64=1", "Flag#=0", "Flag#64=234"},
			[]string{"Flag#64=1", "Flag#0=0", "Flag#64=1"},
		},
		{
			[]string{"Shift=5", "ShiftA=15", "ShiftA=52", "ShiftA=544", "ShiftG=21"},
			[]string{"Shift=5", "ShiftA=15", "ShiftA=52", "ShiftA=544", "ShiftG=21"},
		},
		{
			[]string{"State=5", "StateA=15", "StateA=52", "StateA=544", "StateG=21"},
			[]string{"State=5", "StateA=15", "StateA=52", "StateA=544", "StateG=21"},
		},
	}

	for i, tt := range tests {
		testname := fmt.Sprintf("TestInbound%d", i)
		t.Run(testname, func(t *testing.T) {
			protobufObj := RawPanelASCIIstringsToInboundMessages(tt.give)
			//log.Println(log.Indent(protobufObj))
			roundtrip := InboundMessagesToRawPanelASCIIstrings(protobufObj)
			//log.Println(roundtrip)

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

// TestInboundFromJSON tests the exact path the explorer uses:
// JSON (from browser) -> json.Unmarshal -> protobuf -> InboundMessagesToRawPanelASCIIstrings
func TestInboundFromJSON(t *testing.T) {
	var tests = []struct {
		name     string
		json     string
		wantASCII string
	}{
		{
			"TextScroll loop mode with title",
			`{"HWCIDs":[5],"HWCText":{"Formatting":7,"Title":"Hello","TextScroll":{"ScrollMode":1,"ScrollTitle":true,"ScrollTextline1":true,"ScrollTextline2":true}}}`,
			"HWCt#5=|||Hello|1||||||||||29",
		},
		{
			"TextScroll bounce, speed=2, dwell=1",
			`{"HWCIDs":[10],"HWCText":{"Formatting":7,"Title":"Header","Textline1":"Line1","Textline2":"Line2","TextScroll":{"ScrollMode":2,"ScrollTitle":true,"ScrollTextline1":true,"ScrollTextline2":true,"ScrollSpeed":2,"PauseDwell":1}}}`,
			"HWCt#10=|||Header|1|Line1|Line2||||||||222",
		},
		{
			"No TextScroll (text only)",
			`{"HWCIDs":[2],"HWCText":{"Formatting":7,"Title":"Hello","Textline1":"World"}}`,
			"HWCt#2=|||Hello|1|World",
		},
		{
			"TextScroll loop with adaptive speed",
			`{"HWCIDs":[3],"HWCText":{"Formatting":7,"Title":"Title","Textline1":"T1","Textline2":"T2","TextScroll":{"ScrollMode":1,"ScrollTitle":true,"ScrollTextline1":true,"ScrollTextline2":true,"AdaptiveSpeed":true}}}`,
			"HWCt#3=|||Title|1|T1|T2||||||||541",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			state := &ibeam_rawpanel.HWCState{}
			err := json.Unmarshal([]byte(tt.json), state)
			if err != nil {
				t.Fatalf("json.Unmarshal error: %v", err)
			}

			inboundMessages := []*ibeam_rawpanel.InboundMessage{
				{States: []*ibeam_rawpanel.HWCState{state}},
			}
			ascii := InboundMessagesToRawPanelASCIIstrings(inboundMessages)

			if len(ascii) != 1 {
				t.Fatalf("Expected 1 ASCII string, got %d: %v", len(ascii), ascii)
			}
			if ascii[0] != tt.wantASCII {
				t.Errorf("ASCII mismatch:\n  got:  %s\n  want: %s", ascii[0], tt.wantASCII)
			}
		})
	}
}

func TestInboundFromBinary(t *testing.T) {
	var tests = []struct {
		give []*ibeam_rawpanel.InboundMessage
		want []string
	}{
		// Testing JSON strings
		{
			[]*ibeam_rawpanel.InboundMessage{
				{
					Command: &ibeam_rawpanel.Command{
						SetCalibrationProfile: &ibeam_rawpanel.CalibrationProfile{
							Json: `
							[
								{
								 "Test": {
								  "Json": " TEST "
								 }
								}
							   ]
							   `,
						},
					},
				},
			},
			[]string{`SetCalibrationProfile=[{"Test": {"Json": " TEST "}}]`},
		},
	}

	for i, tt := range tests {
		testname := fmt.Sprintf("TestInboundFromBinary%d", i)
		t.Run(testname, func(t *testing.T) {
			ASCIIstrings := InboundMessagesToRawPanelASCIIstrings(tt.give)
			//log.Println(ASCIIstrings)

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
