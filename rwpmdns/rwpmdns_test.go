package rwpmdns

import (
	"reflect"
	"strings"
	"testing"
)

func TestSoftwareHostname(t *testing.T) {
	cases := []struct {
		serial string
		suffix []string
		want   string
	}{
		{"442852", nil, "xp-442852"},
		{"CL15K1A00123", nil, "xp-cl15k1a00123"},
		{"  A_B//c  ", nil, "xp-a-b-c"},
		{"", nil, "xp-unknown"},
		{"\x01\x02", nil, "xp-unknown"},
		{"777777", []string{"SK_RAWPANEL-DUMMY"}, "xp-777777-sk-rawpanel-dummy"},
		{"442852", []string{"proxy"}, "xp-442852-proxy"},
		{"442852", []string{"", "--"}, "xp-442852"},
		{strings.Repeat("a", 100), nil, "xp-" + strings.Repeat("a", 60)},
		{strings.Repeat("a", 59) + "-b", nil, "xp-" + strings.Repeat("a", 59)},
	}
	for _, c := range cases {
		if got := SoftwareHostname(c.serial, c.suffix...); got != c.want {
			t.Errorf("SoftwareHostname(%q, %q) = %q, want %q", c.serial, c.suffix, got, c.want)
		}
	}
}

func TestRWPText(t *testing.T) {
	got := RWPText("A1 B2", "XP_STREAMDECK-XL", "Streamdeck XL", "", "rwp2.0A/B", false)
	want := []string{"protocol=rwp2.0A/B", "name=Streamdeck XL", "model=XP_STREAMDECK-XL", "serial=A1 B2"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("RWPText = %q, want %q", got, want)
	}
	got = RWPText("442852", "SK_X", "", "", "rwp1.0A", true)
	if got[len(got)-1] != "restricted=1" {
		t.Errorf("restricted not last: %q", got)
	}
	if long := RWPText("1", "", "", strings.Repeat("x", 300), "", false); len(long[0]) != 255 {
		t.Errorf("long entry not cut to 255: %d", len(long[0]))
	}
}

func TestParseTXT(t *testing.T) {
	got := ParseTXT([]string{"serial=1", "model=SK_X", "serial=2", "flag", "empty=", "eq=a=b"})
	want := map[string]string{"serial": "1", "model": "SK_X", "empty": "", "eq": "a=b"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("ParseTXT = %v, want %v", got, want)
	}
}

func TestParseInstance(t *testing.T) {
	cases := []struct{ in, model, serial string }{
		{"SK_SMARTTALLY [442852]", "SK_SMARTTALLY", "442852"},
		{`SK_SMARTTALLY\ [442852]`, "SK_SMARTTALLY", "442852"},
		{"SKAARHOJ PiPlayer [ab12cd34]", "SKAARHOJ PiPlayer", "ab12cd34"},
		{"XP_STREAMDECK-XL[CL15]", "XP_STREAMDECK-XL", "CL15"},
	}
	for _, c := range cases {
		m, s, ok := ParseInstance(c.in)
		if !ok || m != c.model || s != c.serial {
			t.Errorf("ParseInstance(%q) = (%q, %q, %v), want (%q, %q)", c.in, m, s, ok, c.model, c.serial)
		}
	}
	for _, in := range []string{"", "Camera 4", "[123]", "SK_X [123] extra"} {
		if _, _, ok := ParseInstance(in); ok {
			t.Errorf("ParseInstance(%q) should fail", in)
		}
	}
}

func TestParseLegacyHostname(t *testing.T) {
	s, m, ok := ParseLegacyHostname("442852.SK_SMARTTALLY.skaarhoj.local.")
	if !ok || s != "442852" || m != "SK_SMARTTALLY" {
		t.Errorf("got (%q, %q, %v)", s, m, ok)
	}
	if _, m, ok := ParseLegacyHostname("ab12cd34.PiPlayer.SKAARHOJ.local"); !ok || m != "PiPlayer" {
		t.Errorf("PiPlayer hostname: got (%q, %v)", m, ok)
	}
	for _, h := range []string{"sk-442852.local.", "12345.AIRFLY.local", "skaarhoj.local", "x.skaarhoj.local", ""} {
		if _, _, ok := ParseLegacyHostname(h); ok {
			t.Errorf("ParseLegacyHostname(%q) should fail", h)
		}
	}
}

func TestSerialFromHostname(t *testing.T) {
	for host, want := range map[string]string{
		"sk-442852.local.":       "442852",
		"xp-cl15k1a00123.local":  "cl15k1a00123",
		"XP-777777-proxy.local.": "777777-proxy",
	} {
		if got, ok := SerialFromHostname(host); !ok || got != want {
			t.Errorf("SerialFromHostname(%q) = (%q, %v), want %q", host, got, ok, want)
		}
	}
	for _, h := range []string{"442852.SK_X.skaarhoj.local.", "macbook.local", "sk-.local", ""} {
		if _, ok := SerialFromHostname(h); ok {
			t.Errorf("SerialFromHostname(%q) should fail", h)
		}
	}
}

func TestResolve(t *testing.T) {
	cases := []struct {
		name           string
		txt            []string
		instance, host string
		want           Identity
	}{
		{"new firmware", []string{"serial=442852", "model=SK_SMARTTALLY"},
			`SK_SMARTTALLY\ [442852]`, "sk-442852.local.", Identity{"442852", "SK_SMARTTALLY"}},
		{"old firmware rwp", []string{"name=Smart Tally", "model=SK_SMARTTALLY"},
			"SK_SMARTTALLY [442852]", "442852.SK_SMARTTALLY.skaarhoj.local.", Identity{"442852", "SK_SMARTTALLY"}},
		{"txt wins", []string{"serial=111", "model=SK_A"},
			"SK_B [222]", "333.SK_C.skaarhoj.local.", Identity{"111", "SK_A"}},
		{"instance wins over hostname", nil,
			"SK_B [222]", "333.SK_C.skaarhoj.local.", Identity{"222", "SK_B"}},
		{"legacy hostname only", nil, "Camera 4", "442852.SK_X.skaarhoj.local", Identity{"442852", "SK_X"}},
		{"old PiPlayer", nil, "Lobby", "ab12cd34.PiPlayer.skaarhoj.local.", Identity{"ab12cd34", "PiPlayer"}},
		{"new hostname is not used", nil, "Camera 4", "sk-442852.local.", Identity{}},
		{"model is never 'local'", nil, "", "xp-abc.local.", Identity{}},
		{"empty", nil, "", "", Identity{}},
	}
	for _, c := range cases {
		if got := Resolve(ParseTXT(c.txt), c.instance, c.host); got != c.want {
			t.Errorf("%s: got %+v, want %+v", c.name, got, c.want)
		}
	}
	if got := Resolve(nil, "SK_X [1]", ""); got != (Identity{"1", "SK_X"}) {
		t.Errorf("nil txt: got %+v", got)
	}
}

func TestMerge(t *testing.T) {
	got := Identity{Serial: "1"}.Merge(Identity{Serial: "2", Model: "SK_X"})
	if got != (Identity{"1", "SK_X"}) {
		t.Errorf("Merge = %+v", got)
	}
}
