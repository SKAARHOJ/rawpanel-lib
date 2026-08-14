package touchui

import rwp "github.com/SKAARHOJ/rawpanel-lib/ibeam_rawpanel"

// The Raw Panel side talks about orientation as an enum of quarter turns, the
// panel side as degrees. These two convert, and are the only place that mapping
// is written down.

// OrientationDegrees resolves a requested DisplayOrientation into a clockwise
// offset in degrees. forced is false for AUTO (and for any value the protocol
// grows later that this build does not know), meaning "the panel decides" — the
// caller then falls back to its own source, the accelerometer.
func OrientationDegrees(o rwp.TouchUIGlobalOptions_DisplayOrientationE) (deg int, forced bool) {
	switch o {
	case rwp.TouchUIGlobalOptions_ROT0:
		return 0, true
	case rwp.TouchUIGlobalOptions_ROT90:
		return 90, true
	case rwp.TouchUIGlobalOptions_ROT180:
		return 180, true
	case rwp.TouchUIGlobalOptions_ROT270:
		return 270, true
	default:
		return 0, false
	}
}

// OrientationOf is the inverse, for reporting what is actually on screen. It
// never returns AUTO: that is a request, not a state. Anything that is not a
// quarter turn reports ROT0 rather than inventing a value.
func OrientationOf(deg int) rwp.TouchUIGlobalOptions_DisplayOrientationE {
	switch ((deg % 360) + 360) % 360 {
	case 90:
		return rwp.TouchUIGlobalOptions_ROT90
	case 180:
		return rwp.TouchUIGlobalOptions_ROT180
	case 270:
		return rwp.TouchUIGlobalOptions_ROT270
	default:
		return rwp.TouchUIGlobalOptions_ROT0
	}
}
