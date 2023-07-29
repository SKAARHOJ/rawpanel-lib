package main

import "C"

import (
	"encoding/json"
	"strings"
	"unsafe"

	h "github.com/SKAARHOJ/rawpanel-lib"
	rwp "github.com/SKAARHOJ/rawpanel-lib/ibeam_rawpanel"
	rawpanelproc "github.com/SKAARHOJ/rawpanel-processors"
	log "github.com/s00500/env_logger"
	"google.golang.org/protobuf/proto"
)

// The convertor needs to store its state to handle graphics, which is split into multiple messages.
// Unfortunately, global variables doesn't seem to work for this. Global strings objects get damaged
// somehow somewhere in between function calls (garbage collector?). For now, the state gets marshalled
// and passed to the C-side.
//
//export RawPanelASCIIstringToInboundMessage
func RawPanelASCIIstringToInboundMessage(ascii string, state []byte) (unsafe.Pointer, int, unsafe.Pointer, int) {
	var reader h.ASCIIreader
	if state != nil {
		if json.Unmarshal(state, &reader) != nil {
			log.Errorf("Couldn't unmarshal ASCII-bin convertor state")
		}
	}
	m := reader.Parse(ascii)
	updatedState, _ := json.Marshal(reader)
	if m == nil || len(m) != 1 {
		return nil, 0, C.CBytes(updatedState), len(updatedState)
	}
	b, err := proto.Marshal(m[0])
	if err != nil {
		return nil, 0, C.CBytes(updatedState), len(updatedState)
	}
	return C.CBytes(b), len(b), C.CBytes(updatedState), len(updatedState)
}

//export OutboundMessageToRawPanelASCIIstring
func OutboundMessageToRawPanelASCIIstring(bytes []byte) *C.char {
	msg := &rwp.OutboundMessage{}
	if proto.Unmarshal(bytes, msg) != nil {
		return C.CString("")
	}
	strs := h.OutboundMessagesToRawPanelASCIIstrings([]*rwp.OutboundMessage{msg})
	if len(strs) < 1 {
		return C.CString("")
	}
	return C.CString(strings.Join(strs, "\n"))
}

//export InboundStateProcessor
func InboundStateProcessor(bytes []byte) (unsafe.Pointer, int) {
	msg := &rwp.HWCState{}
	if proto.Unmarshal(bytes, msg) != nil {
		return nil, 0
	}
	rawpanelproc.StateProcessor(msg)

	jsonmsg, _ := json.Marshal(msg)
	return C.CBytes(jsonmsg), len(jsonmsg)
}

func main() {
}
