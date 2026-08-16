package upupidae

import (
	"fmt"
	"regexp"
	"strings"
)

var cssVarUseRe = regexp.MustCompile(`var\((--[^,)]+)`)

func cssVarUses(value string) []string {
	seen := map[string]struct{}{}
	var keys []string
	for _, m := range cssVarUseRe.FindAllStringSubmatch(value, -1) {
		if _, ok := seen[m[1]]; !ok {
			seen[m[1]] = struct{}{}
			keys = append(keys, m[1])
		}
	}
	return keys
}

func cssEscape(value string) string {
	if value == "" {
		return ""
	}

	runes := []rune(value)
	if len(runes) == 1 && runes[0] == '-' {
		return `\` + value
	}

	var b strings.Builder
	first := runes[0]

	for i, r := range runes {
		switch {
		case r == 0:
			b.WriteRune('\uFFFD')
		case (r >= 1 && r <= 31) || r == 127 ||
			(i == 0 && r >= '0' && r <= '9') ||
			(i == 1 && r >= '0' && r <= '9' && first == '-'):
			fmt.Fprintf(&b, "\\%x ", r)
		case r >= 128 || r == '-' || r == '_' ||
			(r >= '0' && r <= '9') ||
			(r >= 'A' && r <= 'Z') ||
			(r >= 'a' && r <= 'z'):
			b.WriteRune(r)
		default:
			b.WriteRune('\\')
			b.WriteRune(r)
		}
	}

	return b.String()
}

func cssSelectorClass(class string) string {
	return "." + cssEscape(class)
}

func newCssKV(k, v string) KV {
	return KV{Key: k, Value: v}
}

func newCssKVf(k, format string, a ...any) KV {
	return KV{Key: k, Value: fmt.Sprintf(format, a...)}
}

func newSimpleCssKV(k, v string) []KV {
	return []KV{newCssKV(k, v)}
}

func newSimpleCssKVf(k, format string, a ...any) []KV {
	return []KV{newCssKVf(k, format, a...)}
}

func cssPreffix(shift int) (preffix string) {
	for range shift {
		preffix = preffix + "  "
	}

	return
}

var (
	cssLengthRe        = regexp.MustCompile(`^-?(?:\d+(\.\d+)?|\.\d+)(px|rem|em|%|vw|vh|vmin|vmax|dvw|dvh|lvw|lvh|svw|svh|ch|ex|cm|mm|in|pt|pc)$`)
	cssHexColorRe      = regexp.MustCompile(`^#([0-9a-fA-F]{3,4}|[0-9a-fA-F]{6}|[0-9a-fA-F]{8})$`)
	cssFunctionColorRe = regexp.MustCompile(`^(rgb|rgba|hsl|hsla|hwb|lab|lch|oklab|oklch|color)\(.*\)$`)
)

func cssIsLength(v string) bool {
	v = strings.TrimSpace(v)

	// CSS variables/functions can resolve to lengths
	if strings.HasPrefix(v, "calc(") ||
		strings.HasPrefix(v, "min(") ||
		strings.HasPrefix(v, "max(") ||
		strings.HasPrefix(v, "clamp(") {
		return true
	}

	// 0 is a valid CSS length without a unit
	if v == "0" {
		return true
	}

	return cssLengthRe.MatchString(v)
}

func cssIsColor(v string) bool {
	v = strings.TrimSpace(strings.ToLower(v))

	if v == "" {
		return false
	}

	// css custom properties/functions
	if strings.HasPrefix(v, "var(") {
		return true
	}

	// css custom properties/functions
	if strings.HasPrefix(v, "light-dark(") {
		return true
	}

	// hex colors
	if cssHexColorRe.MatchString(v) {
		return true
	}

	// functional colors
	if cssFunctionColorRe.MatchString(v) {
		return true
	}

	// css named colors
	_, ok := cssColors[v]

	return ok
}

var cssColors = map[string]struct{}{
	"aliceblue":            {},
	"antiquewhite":         {},
	"aqua":                 {},
	"aquamarine":           {},
	"azure":                {},
	"beige":                {},
	"bisque":               {},
	"black":                {},
	"blanchedalmond":       {},
	"blue":                 {},
	"blueviolet":           {},
	"brown":                {},
	"burlywood":            {},
	"cadetblue":            {},
	"chartreuse":           {},
	"chocolate":            {},
	"coral":                {},
	"cornflowerblue":       {},
	"cornsilk":             {},
	"crimson":              {},
	"cyan":                 {},
	"darkblue":             {},
	"darkcyan":             {},
	"darkgoldenrod":        {},
	"darkgray":             {},
	"darkgreen":            {},
	"darkgrey":             {},
	"darkkhaki":            {},
	"darkmagenta":          {},
	"darkolivegreen":       {},
	"darkorange":           {},
	"darkorchid":           {},
	"darkred":              {},
	"darksalmon":           {},
	"darkseagreen":         {},
	"darkslateblue":        {},
	"darkslategray":        {},
	"darkslategrey":        {},
	"darkturquoise":        {},
	"darkviolet":           {},
	"deeppink":             {},
	"deepskyblue":          {},
	"dimgray":              {},
	"dimgrey":              {},
	"dodgerblue":           {},
	"firebrick":            {},
	"floralwhite":          {},
	"forestgreen":          {},
	"fuchsia":              {},
	"gainsboro":            {},
	"ghostwhite":           {},
	"gold":                 {},
	"goldenrod":            {},
	"gray":                 {},
	"green":                {},
	"greenyellow":          {},
	"grey":                 {},
	"honeydew":             {},
	"hotpink":              {},
	"indianred":            {},
	"indigo":               {},
	"ivory":                {},
	"khaki":                {},
	"lavender":             {},
	"lavenderblush":        {},
	"lawngreen":            {},
	"lemonchiffon":         {},
	"lightblue":            {},
	"lightcoral":           {},
	"lightcyan":            {},
	"lightgoldenrodyellow": {},
	"lightgray":            {},
	"lightgreen":           {},
	"lightgrey":            {},
	"lightpink":            {},
	"lightsalmon":          {},
	"lightseagreen":        {},
	"lightskyblue":         {},
	"lightslategray":       {},
	"lightslategrey":       {},
	"lightsteelblue":       {},
	"lightyellow":          {},
	"lime":                 {},
	"limegreen":            {},
	"linen":                {},
	"magenta":              {},
	"maroon":               {},
	"mediumaquamarine":     {},
	"mediumblue":           {},
	"mediumorchid":         {},
	"mediumpurple":         {},
	"mediumseagreen":       {},
	"mediumslateblue":      {},
	"mediumspringgreen":    {},
	"mediumturquoise":      {},
	"mediumvioletred":      {},
	"midnightblue":         {},
	"mintcream":            {},
	"mistyrose":            {},
	"moccasin":             {},
	"navajowhite":          {},
	"navy":                 {},
	"oldlace":              {},
	"olive":                {},
	"olivedrab":            {},
	"orange":               {},
	"orangered":            {},
	"orchid":               {},
	"palegoldenrod":        {},
	"palegreen":            {},
	"paleturquoise":        {},
	"palevioletred":        {},
	"papayawhip":           {},
	"peachpuff":            {},
	"peru":                 {},
	"pink":                 {},
	"plum":                 {},
	"powderblue":           {},
	"purple":               {},
	"rebeccapurple":        {},
	"red":                  {},
	"rosybrown":            {},
	"royalblue":            {},
	"saddlebrown":          {},
	"salmon":               {},
	"sandybrown":           {},
	"seagreen":             {},
	"seashell":             {},
	"sienna":               {},
	"silver":               {},
	"skyblue":              {},
	"slateblue":            {},
	"slategray":            {},
	"slategrey":            {},
	"snow":                 {},
	"springgreen":          {},
	"steelblue":            {},
	"tan":                  {},
	"teal":                 {},
	"thistle":              {},
	"tomato":               {},
	"turquoise":            {},
	"violet":               {},
	"wheat":                {},
	"white":                {},
	"whitesmoke":           {},
	"yellow":               {},
	"yellowgreen":          {},

	// CSS special values
	"transparent":  {},
	"currentcolor": {},
}
