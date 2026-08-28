package calibration

import (
	"encoding/json"
	"strings"
	"testing"
)

// What a BPI panel with both an analog component and an encoder returns for
// "CalibrationProfile?". The encoder entry carries EncoderDriverKey/TicksPerPulse, the
// analog one the XC8-style fields.
const sampleProfile = `[{"HWCid":49,"AnalogKey":"av1","Config":{"Type":"Absolute","Comment":"Fader","CenterPoint":2000,"Deadzone":110,"End":3790,"Start":210,"Tolerance":25}},{"HWCid":12,"EncoderDriverKey":"enc3","Config":{"Type":"Relative","Comment":"Encoder 3","Hysteresis":30,"KineticTimeout":250,"TicksPerPulse":4}}]`

// A profile read from a panel and written straight back must not lose any field. This is
// the whole reason the type lives here rather than being re-declared per program: an
// unmodelled field is dropped silently, and it has shipped that way before.
func TestRoundTripPreservesEveryField(t *testing.T) {
	p, err := Parse(sampleProfile)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}

	out, err := p.JSON()
	if err != nil {
		t.Fatalf("JSON: %v", err)
	}

	for _, want := range []string{
		`"AnalogKey":"av1"`, `"EncoderDriverKey":"enc3"`, `"TicksPerPulse":4`,
		`"Hysteresis":30`, `"KineticTimeout":250`, `"CenterPoint":2000`,
		`"Deadzone":110`, `"End":3790`, `"Start":210`, `"Tolerance":25`,
	} {
		if !strings.Contains(out, want) {
			t.Errorf("round-trip dropped %s\ngot: %s", want, out)
		}
	}
}

// Round-tripping must be byte-identical, not merely semantically equal: the field order
// here is the panel's own, and callers verify a save by comparing what they sent against
// what the panel reports back.
func TestRoundTripIsByteIdentical(t *testing.T) {
	p, err := Parse(sampleProfile)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}

	out, err := p.JSON()
	if err != nil {
		t.Fatalf("JSON: %v", err)
	}

	if out != sampleProfile {
		t.Errorf("round-trip changed the profile:\n want: %s\n  got: %s", sampleProfile, out)
	}
}

// nil-ness is load-bearing: it is how a panel says "this parameter does not apply here",
// and how editors decide which controls to render. An absent field must not come back as
// a zero, and must stay absent on the way out.
func TestAbsentFieldsStayAbsent(t *testing.T) {
	p, err := Parse(`[{"HWCid":7,"Config":{"Type":"Absolute"}}]`)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if len(p) != 1 {
		t.Fatalf("got %d entries, want 1", len(p))
	}
	if c := p[0].Config; c.CenterPoint != nil || c.Deadzone != nil || c.End != nil ||
		c.Start != nil || c.Tolerance != nil || c.Hysteresis != nil ||
		c.KineticTimeout != nil || c.TicksPerPulse != nil {
		t.Errorf("absent parameters decoded as non-nil: %+v", c)
	}

	out, err := p.JSON()
	if err != nil {
		t.Fatalf("JSON: %v", err)
	}
	if want := `[{"HWCid":7,"Config":{"Type":"Absolute"}}]`; out != want {
		t.Errorf("absent parameters reappeared:\n want: %s\n  got: %s", want, out)
	}
}

// A panel with nothing stored reports an empty value. That is not a parse failure.
func TestParseEmptyIsNotAnError(t *testing.T) {
	p, err := Parse("")
	if err != nil {
		t.Errorf("Parse(\"\"): %v", err)
	}
	if p != nil {
		t.Errorf("Parse(\"\") = %v, want nil", p)
	}
}

func TestParseRejectsMalformedJSON(t *testing.T) {
	if _, err := Parse(`[{"HWCid":`); err == nil {
		t.Error("Parse accepted truncated JSON")
	}
}

// The profile is a bare array on the wire, with no enclosing object.
func TestProfileEncodesAsBareArray(t *testing.T) {
	out, err := Profile{{HWCid: 1}}.JSON()
	if err != nil {
		t.Fatalf("JSON: %v", err)
	}
	var v any
	if err := json.Unmarshal([]byte(out), &v); err != nil {
		t.Fatalf("re-decode: %v", err)
	}
	if _, ok := v.([]any); !ok {
		t.Errorf("encoded as %T, want a JSON array: %s", v, out)
	}
}
