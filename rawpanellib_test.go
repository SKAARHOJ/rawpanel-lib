package rawpanellib

import (
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
