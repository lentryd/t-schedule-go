// Package colorize maps a Google Calendar / CSS color string to the closest
// Google Calendar colorId, mirroring src/utils/colorize.ts (which relies on
// chroma-js). Named colors, "darken-N"/"brighten-N" modifiers and the CIE94
// distance formula are reimplemented by hand to match chroma-js output
// closely enough for this purpose.
package colorize

import (
	"math"
	"strconv"
	"strings"
)

// eventColor pairs a reference hex color with a Google Calendar colorId.
type eventColor struct {
	hex     string
	colorID int
}

var eventColors = []eventColor{
	{"#959dd5", 1},
	{"#6ac594", 2},
	{"#a559b8", 3},
	{"#ed978e", 4},
	{"#fbcb62", 5},
	{"#fa7b50", 6},
	{"#66aee9", 7},
	{"#7e7e7e", 8},
	{"#6e71c2", 9},
	{"#509967", 10},
	{"#e3573a", 11},
}

// defaultHex is returned by stringToHex() for unrecognized names, mirroring
// the '#669933' fallback in the original stringToHex().
const defaultHex = "#669933"

// CIE94 coefficients, matching the constants in colorize.ts (kL/kC/kH/k1/k2).
const (
	kL = 1.0
	kC = 1.0
	kH = 1.0
	k1 = 0.045
	k2 = 0.015
)

type lch struct {
	l, c, h float64
}

// NearestColorID returns the Google Calendar colorId whose reference hex is
// closest (by the same CIE94-derived distance as the Node bot) to the given
// color. Mirrors nearestColor() in colorize.ts.
func NearestColorID(color string) int {
	if color == "" {
		return 2
	}

	hex := color
	if !strings.HasPrefix(hex, "#") {
		hex = stringToHex(color)
	}

	source := hexToLCH(hex)

	minDiff := math.Inf(1)
	closest := 0

	for _, ec := range eventColors {
		target := hexToLCH(ec.hex)

		diff := cie94(source, target)
		// NaN comparisons are always false, exactly like JS: a candidate
		// that produces NaN (e.g. an ill-defined hue delta) is simply never
		// selected, so this must NOT be clamped/guarded away.
		if diff < minDiff {
			minDiff = diff
			closest = ec.colorID
		}
	}

	return closest
}

// cie94 mirrors the (non-standard, but original) formula in colorize.ts's
// cie94(). Do not "fix" the deltaH sqrt-of-negative case: chroma-js/JS
// leaves it as NaN and NaN comparisons short-circuit to false, which is
// itself part of the original's behavior.
func cie94(c1, c2 lch) float64 {
	deltaL := c1.l - c2.l
	deltaC := math.Sqrt(c1.c*c1.c+c1.h*c1.h) - math.Sqrt(c2.c*c2.c+c2.h*c2.h)
	deltaH := math.Sqrt((c1.c-c2.c)*(c1.c-c2.c) + (c1.h-c2.h)*(c1.h-c2.h) - deltaC*deltaC)

	return math.Sqrt(
		(deltaL/(kL*k1))*(deltaL/(kL*k1)) +
			(deltaC/(kC*k2))*(deltaC/(kC*k2)) +
			(deltaH/(kH*k2))*(deltaH/(kH*k2)),
	)
}

// hexToLCH converts a "#rrggbb" color to CIE LCH (D65). Returns the zero LCH
// for malformed input (matching chroma's tendency to coerce garbage to
// black rather than erroring).
func hexToLCH(hex string) lch {
	l, a, b, ok := hexToLab(hex)
	if !ok {
		return lch{}
	}

	c := math.Sqrt(a*a + b*b)
	h := math.Atan2(b, a)

	return lch{l: l, c: c, h: h}
}

// hexToLab converts "#rrggbb" to CIE Lab (D65).
func hexToLab(hex string) (l, a, b float64, ok bool) {
	hex = strings.TrimPrefix(hex, "#")
	if len(hex) != 6 {
		return 0, 0, 0, false
	}

	rv, err1 := strconv.ParseUint(hex[0:2], 16, 8)
	gv, err2 := strconv.ParseUint(hex[2:4], 16, 8)
	bv, err3 := strconv.ParseUint(hex[4:6], 16, 8)
	if err1 != nil || err2 != nil || err3 != nil {
		return 0, 0, 0, false
	}

	r, g, bl := srgbToLinear(float64(rv)/255), srgbToLinear(float64(gv)/255), srgbToLinear(float64(bv)/255)

	// sRGB -> XYZ (D65)
	x := r*0.4124564 + g*0.3575761 + bl*0.1804375
	y := r*0.2126729 + g*0.7151522 + bl*0.0721750
	z := r*0.0193339 + g*0.1191920 + bl*0.9503041

	fx, fy, fz := labF(x/xn), labF(y/yn), labF(z/zn)

	l = 116*fy - 16
	a = 500 * (fx - fy)
	b = 200 * (fy - fz)

	return l, a, b, true
}

// labToHex is the inverse of hexToLab, used to apply darken()/brighten().
func labToHex(l, a, b float64) string {
	fy := (l + 16) / 116
	fx := fy + a/500
	fz := fy - b/200

	x := xn * labFInv(fx)
	y := yn * labFInv(fy)
	z := zn * labFInv(fz)

	// XYZ -> linear sRGB (D65)
	r := 3.2404542*x - 1.5371385*y - 0.4985314*z
	g := -0.9692660*x + 1.8760108*y + 0.0415560*z
	bl := 0.0556434*x - 0.2040259*y + 1.0572252*z

	return "#" + hexByte(linearToSRGB(r)) + hexByte(linearToSRGB(g)) + hexByte(linearToSRGB(bl))
}

// D65 white point.
const (
	xn = 0.95047
	yn = 1.00000
	zn = 1.08883
)

func srgbToLinear(v float64) float64 {
	if v <= 0.04045 {
		return v / 12.92
	}
	return math.Pow((v+0.055)/1.055, 2.4)
}

func linearToSRGB(v float64) float64 {
	v = math.Max(0, math.Min(1, v))
	if v <= 0.0031308 {
		return v * 12.92
	}
	return 1.055*math.Pow(v, 1/2.4) - 0.055
}

func labF(t float64) float64 {
	const delta = 6.0 / 29.0
	if t > delta*delta*delta {
		return math.Cbrt(t)
	}
	return t/(3*delta*delta) + 4.0/29.0
}

func labFInv(t float64) float64 {
	const delta = 6.0 / 29.0
	if t > delta {
		return t * t * t
	}
	return 3 * delta * delta * (t - 4.0/29.0)
}

func hexByte(v float64) string {
	n := int(math.Round(v * 255))
	if n < 0 {
		n = 0
	}
	if n > 255 {
		n = 255
	}
	s := strconv.FormatInt(int64(n), 16)
	if len(s) == 1 {
		s = "0" + s
	}
	return s
}

// labKn is chroma-js's Kn constant used by darken()/brighten(): each unit of
// "amount" shifts Lab lightness by this many points.
const labKn = 18.0

// stringToHex resolves a CSS/space-separated color string ("blue",
// "blue darken-2", "blue brighten-1") to a hex color, mirroring
// stringToHex() in colorize.ts. Unrecognized names fall back to
// defaultHex, exactly like the original (it does NOT return early from
// nearestColor - the fallback color still goes through the full distance
// computation).
func stringToHex(str string) string {
	fields := strings.Fields(str)

	colorName := ""
	lightness := ""
	if len(fields) > 0 {
		colorName = fields[0]
	}
	if len(fields) > 1 {
		lightness = fields[1]
	}

	baseHex, ok := namedColors[strings.ToLower(colorName)]
	if !ok {
		return defaultHex
	}

	switch {
	case strings.HasPrefix(lightness, "darken-"):
		if amount, err := strconv.Atoi(strings.TrimPrefix(lightness, "darken-")); err == nil {
			return shiftLightness(baseHex, -float64(amount)*labKn)
		}
	case strings.HasPrefix(lightness, "brighten-"):
		if amount, err := strconv.Atoi(strings.TrimPrefix(lightness, "brighten-")); err == nil {
			return shiftLightness(baseHex, float64(amount)*labKn)
		}
	}

	return baseHex
}

func shiftLightness(hex string, deltaL float64) string {
	l, a, b, ok := hexToLab(hex)
	if !ok {
		return hex
	}
	return labToHex(l+deltaL, a, b)
}

// namedColors is the standard CSS Color Module Level 4 extended keyword
// list, matching what chroma.valid()/chroma(name) accepts in the original.
var namedColors = map[string]string{
	"aliceblue": "#f0f8ff", "antiquewhite": "#faebd7", "aqua": "#00ffff",
	"aquamarine": "#7fffd4", "azure": "#f0ffff", "beige": "#f5f5dc",
	"bisque": "#ffe4c4", "black": "#000000", "blanchedalmond": "#ffebcd",
	"blue": "#0000ff", "blueviolet": "#8a2be2", "brown": "#a52a2a",
	"burlywood": "#deb887", "cadetblue": "#5f9ea0", "chartreuse": "#7fff00",
	"chocolate": "#d2691e", "coral": "#ff7f50", "cornflowerblue": "#6495ed",
	"cornsilk": "#fff8dc", "crimson": "#dc143c", "cyan": "#00ffff",
	"darkblue": "#00008b", "darkcyan": "#008b8b", "darkgoldenrod": "#b8860b",
	"darkgray": "#a9a9a9", "darkgreen": "#006400", "darkgrey": "#a9a9a9",
	"darkkhaki": "#bdb76b", "darkmagenta": "#8b008b", "darkolivegreen": "#556b2f",
	"darkorange": "#ff8c00", "darkorchid": "#9932cc", "darkred": "#8b0000",
	"darksalmon": "#e9967a", "darkseagreen": "#8fbc8f", "darkslateblue": "#483d8b",
	"darkslategray": "#2f4f4f", "darkslategrey": "#2f4f4f", "darkturquoise": "#00ced1",
	"darkviolet": "#9400d3", "deeppink": "#ff1493", "deepskyblue": "#00bfff",
	"dimgray": "#696969", "dimgrey": "#696969", "dodgerblue": "#1e90ff",
	"firebrick": "#b22222", "floralwhite": "#fffaf0", "forestgreen": "#228b22",
	"fuchsia": "#ff00ff", "gainsboro": "#dcdcdc", "ghostwhite": "#f8f8ff",
	"gold": "#ffd700", "goldenrod": "#daa520", "gray": "#808080",
	"green": "#008000", "greenyellow": "#adff2f", "grey": "#808080",
	"honeydew": "#f0fff0", "hotpink": "#ff69b4", "indianred": "#cd5c5c",
	"indigo": "#4b0082", "ivory": "#fffff0", "khaki": "#f0e68c",
	"lavender": "#e6e6fa", "lavenderblush": "#fff0f5", "lawngreen": "#7cfc00",
	"lemonchiffon": "#fffacd", "lightblue": "#add8e6", "lightcoral": "#f08080",
	"lightcyan": "#e0ffff", "lightgoldenrodyellow": "#fafad2", "lightgray": "#d3d3d3",
	"lightgreen": "#90ee90", "lightgrey": "#d3d3d3", "lightpink": "#ffb6c1",
	"lightsalmon": "#ffa07a", "lightseagreen": "#20b2aa", "lightskyblue": "#87cefa",
	"lightslategray": "#778899", "lightslategrey": "#778899", "lightsteelblue": "#b0c4de",
	"lightyellow": "#ffffe0", "lime": "#00ff00", "limegreen": "#32cd32",
	"linen": "#faf0e6", "magenta": "#ff00ff", "maroon": "#800000",
	"mediumaquamarine": "#66cdaa", "mediumblue": "#0000cd", "mediumorchid": "#ba55d3",
	"mediumpurple": "#9370db", "mediumseagreen": "#3cb371", "mediumslateblue": "#7b68ee",
	"mediumspringgreen": "#00fa9a", "mediumturquoise": "#48d1cc", "mediumvioletred": "#c71585",
	"midnightblue": "#191970", "mintcream": "#f5fffa", "mistyrose": "#ffe4e1",
	"moccasin": "#ffe4b5", "navajowhite": "#ffdead", "navy": "#000080",
	"oldlace": "#fdf5e6", "olive": "#808000", "olivedrab": "#6b8e23",
	"orange": "#ffa500", "orangered": "#ff4500", "orchid": "#da70d6",
	"palegoldenrod": "#eee8aa", "palegreen": "#98fb98", "paleturquoise": "#afeeee",
	"palevioletred": "#db7093", "papayawhip": "#ffefd5", "peachpuff": "#ffdab9",
	"peru": "#cd853f", "pink": "#ffc0cb", "plum": "#dda0dd",
	"powderblue": "#b0e0e6", "purple": "#800080", "rebeccapurple": "#663399",
	"red": "#ff0000", "rosybrown": "#bc8f8f", "royalblue": "#4169e1",
	"saddlebrown": "#8b4513", "salmon": "#fa8072", "sandybrown": "#f4a460",
	"seagreen": "#2e8b57", "seashell": "#fff5ee", "sienna": "#a0522d",
	"silver": "#c0c0c0", "skyblue": "#87ceeb", "slateblue": "#6a5acd",
	"slategray": "#708090", "slategrey": "#708090", "snow": "#fffafa",
	"springgreen": "#00ff7f", "steelblue": "#4682b4", "tan": "#d2b48c",
	"teal": "#008080", "thistle": "#d8bfd8", "tomato": "#ff6347",
	"turquoise": "#40e0d0", "violet": "#ee82ee", "wheat": "#f5deb3",
	"white": "#ffffff", "whitesmoke": "#f5f5f5", "yellow": "#ffff00",
	"yellowgreen": "#9acd32",
}
