package main

import "C"

import (
	"unsafe"

	h "github.com/SKAARHOJ/rawpanel-lib"
	rwp "github.com/SKAARHOJ/rawpanel-lib/ibeam_rawpanel"
	"google.golang.org/protobuf/proto"
)

//export RawPanelASCIIstringToInboundMessage
func RawPanelASCIIstringToInboundMessage(ascii string) (unsafe.Pointer, int) {
	m := h.RawPanelASCIIstringsToInboundMessages([]string{ascii})
	if len(m) != 1 {
		return nil, 0
	}
	b, err := proto.Marshal(m[0])
	if err != nil {
		return nil, 0
	}
	return C.CBytes(b), len(b)
}

//export OutboundMessageToRawPanelASCIIstring
func OutboundMessageToRawPanelASCIIstring(bytes []byte) *C.char {
	msg := &rwp.OutboundMessage{}
	if proto.Unmarshal(bytes, msg) != nil {
		return C.CString("")
	}
	strs := h.OutboundMessagesToRawPanelASCIIstrings([]*rwp.OutboundMessage{msg})
	if len(strs) != 1 {
		return C.CString("")
	}
	return C.CString(strs[0])
}

func main() {
}
