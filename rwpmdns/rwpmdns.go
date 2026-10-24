// Package rwpmdns holds the mDNS naming rules for SKAARHOJ devices and Raw Panel
// announcers, for both sides:
//
// Announcers (software panels such as xpanels) register _skaarhoj-rwp._tcp with
// a flat hostname "xp-<serial>" (SoftwareHostname) and serial/model in TXT
// records (RWPText). The instance name stays "MODEL [SERIAL]". LinkIO hardware
// uses "sk-<serial>".
//
// Consumers must not parse serial or model from the hostname any more. Resolve
// reads them from TXT first, then the instance name, then the old
// "SERIAL.MODEL.skaarhoj.local" hostname that older firmware still uses.
//
// The package has no dependencies, so it can be used with any mDNS library.
package rwpmdns

import (
	"regexp"
	"strings"
)

// MaxLabel is the longest DNS label allowed in a hostname.
const MaxLabel = 63

// Slug lowercases s, turns every character outside [a-z0-9] into "-",
// collapses repeated "-" and trims "-" at both ends.
func Slug(s string) string {
	var b strings.Builder
	dash := false
	for _, r := range strings.ToLower(s) {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
			dash = false
		} else if !dash {
			b.WriteByte('-')
			dash = true
		}
	}
	return strings.Trim(b.String(), "-")
}

// SoftwareHostname returns "xp-<slug(serial)>[-<slug(suffix)>...]", at most
// 63 characters, without ".local" (mDNS libraries add the domain).
//
// Use a suffix when the serial alone can collide on one machine, e.g. the
// model for dummy panels or "proxy" for a proxy that forwards another panel's
// serial. An empty serial becomes "unknown".
func SoftwareHostname(serial string, suffix ...string) string {
	parts := []string{Slug(serial)}
	if parts[0] == "" {
		parts[0] = "unknown"
	}
	for _, s := range suffix {
		if s = Slug(s); s != "" {
			parts = append(parts, s)
		}
	}
	h := "xp-" + strings.Join(parts, "-")
	if len(h) > MaxLabel {
		h = strings.TrimRight(h[:MaxLabel], "-")
	}
	return h
}

// RWPText returns the TXT records for a _skaarhoj-rwp._tcp announcement.
//
// serial is the bare serial (not slugged). name is the product name,
// deviceName the user-facing name. restricted adds "restricted=1", meaning the
// panel only accepts some clients (an IP allow-list is set). Empty values are
// left out, and each entry is cut to the 255-byte DNS limit.
func RWPText(serial, model, name, deviceName, protocol string, restricted bool) []string {
	var txt []string
	add := func(key, value string) {
		if value == "" {
			return
		}
		entry := key + "=" + value
		if len(entry) > 255 {
			entry = entry[:255]
		}
		txt = append(txt, entry)
	}
	add("protocol", protocol)
	add("devicename", deviceName)
	add("name", name)
	add("model", model)
	add("serial", serial)
	if restricted {
		add("restricted", "1")
	}
	return txt
}

// Identity is a device's serial and model. Either can be empty when nothing
// provides it.
type Identity struct {
	Serial string
	Model  string
}

// Complete tells if both serial and model are known.
func (id Identity) Complete() bool { return id.Serial != "" && id.Model != "" }

// fill sets empty fields from serial and model.
func (id *Identity) fill(serial, model string) {
	if id.Serial == "" {
		id.Serial = serial
	}
	if id.Model == "" {
		id.Model = model
	}
}

// ParseTXT turns TXT strings ("key=value") into a map. The first value of a
// key wins; entries without "=" are ignored.
func ParseTXT(txt []string) map[string]string {
	m := make(map[string]string, len(txt))
	for _, t := range txt {
		if k, v, ok := strings.Cut(t, "="); ok {
			if _, seen := m[k]; !seen {
				m[k] = v
			}
		}
	}
	return m
}

var (
	instanceRe    = regexp.MustCompile(`^(.+?)\s*\[([^\]]+)\]$`)
	newHostnameRe = regexp.MustCompile(`^(?:sk|xp)-([a-z0-9-]+)\.local\.?$`)
)

// ParseInstance splits an instance name of the form "MODEL [SERIAL]". DNS-SD
// escapes such as "SK_X\ [123]" (grandcat/zeroconf) are removed first.
func ParseInstance(instance string) (model, serial string, ok bool) {
	m := instanceRe.FindStringSubmatch(strings.TrimSpace(Unescape(instance)))
	if m == nil {
		return "", "", false
	}
	return strings.TrimSpace(m[1]), strings.TrimSpace(m[2]), true
}

// ParseLegacyHostname reads serial and model from the old hostname format
// "SERIAL.MODEL.skaarhoj.local[.]". Other hostnames give ok == false.
func ParseLegacyHostname(host string) (serial, model string, ok bool) {
	h := strings.TrimSuffix(host, ".")
	if !strings.HasSuffix(strings.ToLower(h), ".skaarhoj.local") {
		return "", "", false
	}
	parts := strings.Split(h, ".")
	if len(parts) < 4 {
		return "", "", false
	}
	return parts[0], parts[1], true
}

// SerialFromHostname reads the serial from a new-format hostname
// ("sk-442852.local", "xp-abc.local"). Use it for display only: software
// serials are slugged, so the result may differ from the real serial and must
// never be a key for stored data. It has no model.
func SerialFromHostname(host string) (serial string, ok bool) {
	m := newHostnameRe.FindStringSubmatch(strings.ToLower(host))
	if m == nil {
		return "", false
	}
	return m[1], true
}

// Resolve works out serial and model from one service answer: TXT serial= and
// model=, then the instance name "MODEL [SERIAL]", then the old-format
// hostname. It never reads the new-format hostname (see SerialFromHostname).
// txt may be nil.
//
// When a device has several services, call Resolve on the _skaarhoj._tcp
// answer first and fill gaps from the others (see Identity.Merge).
func Resolve(txt map[string]string, instance, host string) Identity {
	var id Identity
	id.fill(txt["serial"], txt["model"])
	if !id.Complete() {
		if model, serial, ok := ParseInstance(instance); ok {
			id.fill(serial, model)
		}
	}
	if !id.Complete() {
		if serial, model, ok := ParseLegacyHostname(host); ok {
			id.fill(serial, model)
		}
	}
	return id
}

// Merge fills empty fields of id from other and returns the result.
func (id Identity) Merge(other Identity) Identity {
	id.fill(other.Serial, other.Model)
	return id
}

// Unescape removes DNS-SD backslash escapes ("SK_X\ [123]" -> "SK_X [123]").
func Unescape(s string) string {
	if !strings.Contains(s, `\`) {
		return s
	}
	var b strings.Builder
	escaped := false
	for _, r := range s {
		if !escaped && r == '\\' {
			escaped = true
			continue
		}
		escaped = false
		b.WriteRune(r)
	}
	return b.String()
}
