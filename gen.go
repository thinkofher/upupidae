package upupidae

import (
	"context"
	"errors"
	"fmt"
	"math"
	"slices"
	"strconv"
	"strings"

	"github.com/thinkofher/upupidae/internal/parser"
)

var (
	errConditionUnknown = errors.New("condition unknown")
	errUtilityNoName    = errors.New("utility needs name at the beginning")
	errUtilityUnknown   = errors.New("utility unknown")
)

type class struct {
	name       string
	conditions []string
	kvs        []KV
	queries    []Query
}

func (c *class) entry() Entry {
	q := &Query{
		KVs:    c.kvs,
		Nested: c.queries,
	}

	slices.Reverse(c.conditions)
	for _, c := range c.conditions {
		q.Name = c
		newq := &Query{}
		newq.Nested = []Query{*q}
		q = newq
	}

	q.Name = c.name

	return Entry{
		Query: q,
	}
}

type gen struct {
	namer      Namer
	varer      Varer
	registerer *registerer
}

func (g *gen) registerVar(ctx context.Context, key string) {
	if g.registerer != nil {
		g.registerer.Register(ctx, key)
	}
}

func (g *gen) cssVar(ctx context.Context, key string) string {
	key = customValue(key)
	g.registerVar(ctx, key)
	return g.varer.Var(ctx, key)
}

func (g *gen) cssRef(ctx context.Context, key string) string {
	g.registerVar(ctx, key)
	return fmt.Sprintf("var(%s)", key)
}

func (g *gen) cssVarFallback(ctx context.Context, key, fallback string) string {
	g.registerVar(ctx, key)
	if fallback == "" {
		return fmt.Sprintf("var(%s,)", key)
	}
	return fmt.Sprintf("var(%s, %s)", key, fallback)
}

func (g *gen) cssVarBlankFallback(ctx context.Context, key string) string {
	g.registerVar(ctx, key)
	return fmt.Sprintf("var(%s, )", key)
}

func (g *gen) cssVarFallbackTight(ctx context.Context, key, fallback string) string {
	g.registerVar(ctx, key)
	return fmt.Sprintf("var(%s,%s)", key, fallback)
}

// cssArbitraryValue registers custom properties referenced in user-provided CSS fragments.
func (g *gen) cssArbitraryValue(ctx context.Context, value string) string {
	for _, key := range cssVarUses(value) {
		g.registerVar(ctx, key)
	}
	return value
}

func (g *gen) cssDefine(ctx context.Context, k, v string) KV {
	g.registerVar(ctx, k)
	return KV{Key: k, Value: v}
}

func (g *gen) cssDefinef(ctx context.Context, k, format string, a ...any) KV {
	return g.cssDefine(ctx, k, fmt.Sprintf(format, a...))
}

func (g *gen) cssKV(_ context.Context, k, v string) KV {
	return KV{Key: k, Value: v}
}

func (g *gen) cssKVf(ctx context.Context, k, format string, a ...any) KV {
	return g.cssKV(ctx, k, fmt.Sprintf(format, a...))
}

func (g *gen) generate(ctx context.Context, c *parser.Class) (*class, error) {
	variants := c.Variants()

	conditions := []string{}
	for _, v := range variants {
		c, err := g.condition(ctx, v)
		if err != nil {
			return nil, fmt.Errorf("condition: %w", err)
		}

		conditions = append(conditions, c...)
	}

	u, err := g.utility(ctx, c.Utility())
	if err != nil {
		return nil, fmt.Errorf("utility: %w", err)
	}

	kvs := g.applyTwContent(ctx, u.kvs, c.Utility(), variants)

	return &class{
		name:       g.namer.Name(ctx, c.String()),
		conditions: conditions,
		kvs:        kvs,
		queries:    u.others,
	}, nil
}

const twContentVar = "--tw-content"

func (g *gen) applyTwContent(ctx context.Context, kvs []KV, utility *parser.Segment, variants []*parser.Segment) []KV {
	pseudo := hasBeforeAfterVariant(variants)

	switch {
	case isContentPropertyUtility(utility):
		if isContentNoneUtility(utility) && !pseudo {
			return append(kvs, g.cssKV(ctx, "content", "none"))
		}
		return append(kvs, g.cssKV(ctx, "content", "var("+twContentVar+")"))
	case pseudo:
		return append(kvs, g.cssKV(ctx, "content", "var("+twContentVar+")"))
	default:
		return kvs
	}
}

func decodeArbitraryVariant(raw string) string {
	var b strings.Builder
	for i := 0; i < len(raw); i++ {
		if raw[i] == '\\' && i+1 < len(raw) && raw[i+1] == '_' {
			b.WriteByte('_')
			i++
			continue
		}
		if raw[i] == '_' {
			b.WriteByte(' ')
			continue
		}
		b.WriteByte(raw[i])
	}
	return b.String()
}

func (g *gen) condition(ctx context.Context, s *parser.Segment) ([]string, error) {
	if s.Arbitrary != "" {
		sel := decodeArbitraryVariant(strings.TrimPrefix(strings.TrimSuffix(s.Arbitrary, "]"), "["))
		if sel == "" || strings.TrimSpace(sel) == "" {
			return nil, errConditionUnknown
		}

		relative := sel[0] == '>' || sel[0] == '+' || sel[0] == '~'
		if !relative && !strings.HasPrefix(sel, "@") && !strings.Contains(sel, "&") {
			sel = "&:is(" + sel + ")"
		}

		return []string{sel}, nil
	}

	if s.Name == nil {
		return nil, errConditionUnknown
	}

	head := s.Name.Head()
	parts := s.Name.Parts()

	switch head {
	case "group", "peer":
		if len(parts) == 0 {
			return nil, errConditionUnknown
		}

		var inner *parser.Segment
		switch parts[0].Ident {
		case "has":
			inner = &parser.Segment{Name: parser.Regular("has", parts[1:])}
		case "not":
			inner = &parser.Segment{Name: parser.Regular("not", parts[1:])}
		default:
			if parts[0].Ident == "" && parts[0].AlphaNum != "" {
				inner = &parser.Segment{Name: parser.Regular(parts[0].AlphaNum, parts[1:])}
			} else {
				inner = &parser.Segment{Name: parser.Regular(parts[0].Ident, parts[1:])}
			}
			if parts[0].Ident == "" && parts[0].Arbitrary != "" {
				inner = &parser.Segment{Arbitrary: parts[0].Arbitrary}
			}
		}

		groupSel := ":where(.group)"
		if s.Modifier != nil && s.Modifier.Ident != "" {
			groupSel = ":where(.group\\/" + s.Modifier.Ident + ")"
		}
		combinator := " *"
		if head == "peer" {
			groupSel = strings.Replace(groupSel, ".group", ".peer", 1)
			combinator = " ~ *"
		}

		if parts[0].Arbitrary != "" && parts[0].Ident != "has" && parts[0].Ident != "not" {
			sel := decodeArbitraryVariant(strings.TrimPrefix(strings.TrimSuffix(parts[0].Arbitrary, "]"), "["))
			if strings.Contains(sel, "&") {
				sel = strings.ReplaceAll(sel, "&", groupSel)
			} else {
				sel = groupSel + sel
			}
			if strings.Count(sel, ",") > 0 {
				sel = ":is(" + sel + ")"
			}
			return []string{"&:is(" + sel + combinator + ")"}, nil
		}

		innerConds, err := g.condition(ctx, inner)
		if err != nil {
			return nil, err
		}

		out := make([]string, 0, len(innerConds))
		for _, c := range innerConds {
			if strings.HasPrefix(c, "@") {
				out = append(out, c)
				continue
			}
			if strings.HasPrefix(c, "&") {
				sel := strings.Replace(c, "&", groupSel, 1) + combinator
				out = append(out, "&:is("+sel+")")
				continue
			}
			out = append(out, c)
		}

		return out, nil

	case "not":
		if len(parts) == 0 {
			return nil, errConditionUnknown
		}

		var inner *parser.Segment
		if parts[0].Ident == "" && parts[0].AlphaNum != "" {
			inner = &parser.Segment{Name: parser.Regular(parts[0].AlphaNum, parts[1:])}
		} else {
			inner = &parser.Segment{Name: parser.Regular(parts[0].Ident, parts[1:])}
		}
		if parts[0].Arbitrary != "" {
			inner = &parser.Segment{Arbitrary: parts[0].Arbitrary}
		}

		innerConds, err := g.condition(ctx, inner)
		if err != nil {
			return nil, err
		}

		hasAmp := false
		for _, c := range innerConds {
			if strings.HasPrefix(c, "&") {
				hasAmp = true
				break
			}
		}

		out := make([]string, 0, len(innerConds))
		for _, c := range innerConds {
			if strings.HasPrefix(c, "@") {
				if hasAmp {
					out = append(out, c)
					continue
				}
				if negated, ok := negateAtCondition(c); ok {
					out = append(out, negated)
					continue
				}
				out = append(out, c)
				continue
			}
			if strings.HasPrefix(c, "&") {
				out = append(out, "&:not("+strings.TrimPrefix(c, "&")+")")
				continue
			}
			out = append(out, c)
		}

		return out, nil

	case "has":
		if len(parts) == 0 {
			return nil, errConditionUnknown
		}

		if parts[0].Arbitrary != "" {
			sel := strings.TrimPrefix(strings.TrimSuffix(parts[0].Arbitrary, "]"), "[")
			return []string{"&:has(" + sel + ")"}, nil
		}

		inner := &parser.Segment{Name: parser.Regular(parts[0].Ident, parts[1:])}
		innerConds, err := g.condition(ctx, inner)
		if err != nil {
			return nil, err
		}

		out := make([]string, 0, len(innerConds))
		for _, c := range innerConds {
			if strings.HasPrefix(c, "@") {
				out = append(out, c)
				continue
			}
			if strings.HasPrefix(c, "&") {
				out = append(out, "&:has("+strings.TrimPrefix(c, "&")+")")
				continue
			}
			out = append(out, c)
		}

		return out, nil

	case "in":
		if len(parts) == 1 && parts[0].Ident == "range" {
			return []string{"&:in-range"}, nil
		}

		if len(parts) == 0 {
			return nil, errConditionUnknown
		}

		if parts[0].Arbitrary != "" {
			sel := strings.TrimPrefix(strings.TrimSuffix(parts[0].Arbitrary, "]"), "[")
			if !strings.HasPrefix(sel, ":") && !strings.HasPrefix(sel, ".") && !strings.HasPrefix(sel, "#") {
				sel = ":is(" + sel + ")"
			}
			return []string{":where(" + sel + ") &"}, nil
		}

		if len(parts) == 2 && parts[0].Ident == "data" && parts[1].Ident != "" {
			return []string{":where([data-" + parts[1].Ident + "]) &"}, nil
		}

		var inner *parser.Segment
		if parts[0].Ident == "" && parts[0].AlphaNum != "" {
			inner = &parser.Segment{Name: parser.Regular(parts[0].AlphaNum, parts[1:])}
		} else {
			inner = &parser.Segment{Name: parser.Regular(parts[0].Ident, parts[1:])}
		}

		innerConds, err := g.condition(ctx, inner)
		if err != nil {
			return nil, err
		}

		out := make([]string, 0, len(innerConds))
		for _, c := range innerConds {
			if strings.HasPrefix(c, "@") {
				out = append(out, c)
				continue
			}
			if strings.HasPrefix(c, "&") {
				out = append(out, ":where("+strings.TrimPrefix(c, "&")+") &")
				continue
			}
			out = append(out, c)
		}

		return out, nil

	case "aria":
		if len(parts) != 1 {
			return nil, errConditionUnknown
		}

		if parts[0].Arbitrary != "" {
			sel := strings.TrimPrefix(strings.TrimSuffix(parts[0].Arbitrary, "]"), "[")
			return []string{"&[aria-" + sel + "]"}, nil
		}

		if parts[0].Ident != "" {
			return []string{fmt.Sprintf(`&[aria-%s="true"]`, parts[0].Ident)}, nil
		}

		return nil, errConditionUnknown

	case "data":
		if len(parts) != 1 || parts[0].Arbitrary == "" {
			return nil, errConditionUnknown
		}

		sel := strings.TrimPrefix(strings.TrimSuffix(parts[0].Arbitrary, "]"), "[")
		return []string{"&[data-" + sel + "]"}, nil

	case "nth":
		if len(parts) == 1 && parts[0].Int != nil {
			return []string{fmt.Sprintf("&:nth-child(%d)", *parts[0].Int)}, nil
		}

		if len(parts) == 2 && parts[0].Ident == "last" && parts[1].Int != nil {
			return []string{fmt.Sprintf("&:nth-last-child(%d)", *parts[1].Int)}, nil
		}

		if len(parts) == 3 && parts[0].Ident == "of" && parts[1].Ident == "type" && parts[2].Int != nil {
			return []string{fmt.Sprintf("&:nth-of-type(%d)", *parts[2].Int)}, nil
		}

		if len(parts) == 4 && parts[0].Ident == "last" && parts[1].Ident == "of" && parts[2].Ident == "type" && parts[3].Int != nil {
			return []string{fmt.Sprintf("&:nth-last-of-type(%d)", *parts[3].Int)}, nil
		}

		return nil, errConditionUnknown

	case "supports":
		if len(parts) != 1 || parts[0].Arbitrary == "" {
			return nil, errConditionUnknown
		}

		value := strings.TrimPrefix(strings.TrimSuffix(parts[0].Arbitrary, "]"), "[")
		if !strings.Contains(value, ":") {
			value = value + ": " + g.cssRef(ctx, "--tw")
		}
		if !strings.HasPrefix(value, "(") {
			value = "(" + value + ")"
		}

		return []string{"@supports " + value}, nil

	case "max":
		if len(parts) != 1 {
			return nil, errConditionUnknown
		}

		if parts[0].Arbitrary != "" {
			width := strings.TrimPrefix(strings.TrimSuffix(parts[0].Arbitrary, "]"), "[")
			return []string{fmt.Sprintf("@media (width < %s)", width)}, nil
		}

		switch parts[0].Ident {
		case "sm":
			return []string{"@media (width < 40rem)"}, nil
		case "md":
			return []string{"@media (width < 48rem)"}, nil
		case "lg":
			return []string{"@media (width < 64rem)"}, nil
		case "xl":
			return []string{"@media (width < 80rem)"}, nil
		}

		if parts[0].AlphaNum == "2xl" {
			return []string{"@media (width < 96rem)"}, nil
		}

		return nil, errConditionUnknown

	case "min":
		if len(parts) != 1 {
			return nil, errConditionUnknown
		}

		if parts[0].Arbitrary != "" {
			width := strings.TrimPrefix(strings.TrimSuffix(parts[0].Arbitrary, "]"), "[")
			return []string{fmt.Sprintf("@media (width >= %s)", width)}, nil
		}

		switch parts[0].Ident {
		case "sm":
			return []string{"@media (width >= 40rem)"}, nil
		case "md":
			return []string{"@media (width >= 48rem)"}, nil
		case "lg":
			return []string{"@media (width >= 64rem)"}, nil
		case "xl":
			return []string{"@media (width >= 80rem)"}, nil
		}

		if parts[0].AlphaNum == "2xl" {
			return []string{"@media (width >= 96rem)"}, nil
		}

		return nil, errConditionUnknown

	case "sm":
		return []string{"@media (width >= 40rem)"}, nil
	case "md":
		return []string{"@media (width >= 48rem)"}, nil
	case "lg":
		return []string{"@media (width >= 64rem)"}, nil
	case "xl":
		return []string{"@media (width >= 80rem)"}, nil
	case "2xl":
		return []string{"@media (width >= 96rem)"}, nil

	case "hover":
		return []string{"&:hover", "@media (hover: hover)"}, nil
	case "focus":
		if len(parts) == 0 {
			return []string{"&:focus"}, nil
		}

		switch parts[0].Ident {
		case "within", "visible":
			return []string{"&:" + head + "-" + parts[0].Ident}, nil
		}
	case "active", "visited", "target", "default", "checked", "indeterminate", "autofill", "optional", "required", "valid", "invalid", "enabled", "disabled", "empty":
		return []string{"&:" + head}, nil
	case "placeholder":
		if len(parts) == 1 && parts[0].Ident == "shown" {
			return []string{"&:placeholder-shown"}, nil
		}
		return []string{"&::placeholder"}, nil
	case "user":
		if len(parts) == 1 && (parts[0].Ident == "valid" || parts[0].Ident == "invalid") {
			return []string{"&:user-" + parts[0].Ident}, nil
		}
	case "out":
		if len(parts) == 2 && parts[0].Ident == "of" && parts[1].Ident == "range" {
			return []string{"&:out-of-range"}, nil
		}
	case "read":
		if len(parts) == 1 && parts[0].Ident == "only" {
			return []string{"&:read-only"}, nil
		}
	case "first":
		if len(parts) == 0 {
			return []string{"&:first-child"}, nil
		}
		if len(parts) == 2 && parts[0].Ident == "of" && parts[1].Ident == "type" {
			return []string{"&:first-of-type"}, nil
		}
		if len(parts) == 1 && parts[0].Ident == "letter" {
			return []string{"&::first-letter"}, nil
		}
		if len(parts) == 1 && parts[0].Ident == "line" {
			return []string{"&::first-line"}, nil
		}
	case "last":
		if len(parts) == 0 {
			return []string{"&:last-child"}, nil
		}
		if len(parts) == 2 && parts[0].Ident == "of" && parts[1].Ident == "type" {
			return []string{"&:last-of-type"}, nil
		}
	case "only":
		if len(parts) == 0 {
			return []string{"&:only-child"}, nil
		}
		if len(parts) == 2 && parts[0].Ident == "of" && parts[1].Ident == "type" {
			return []string{"&:only-of-type"}, nil
		}
	case "odd":
		return []string{"&:nth-child(odd)"}, nil
	case "even":
		return []string{"&:nth-child(even)"}, nil
	case "open":
		return []string{"&:is([open], :popover-open, :open)"}, nil
	case "inert":
		return []string{"&:is([inert], [inert] *)"}, nil
	case "*":
		return []string{":is(& > *)"}, nil
	case "**":
		return []string{":is(& *)"}, nil
	case "marker":
		return []string{"& *::marker, &::marker, & *::-webkit-details-marker, &::-webkit-details-marker"}, nil
	case "selection":
		return []string{"& *::selection, &::selection"}, nil
	case "file":
		return []string{"&::file-selector-button"}, nil
	case "backdrop":
		return []string{"&::backdrop"}, nil
	case "details":
		if len(parts) == 1 && parts[0].Ident == "content" {
			return []string{"&::details-content"}, nil
		}
	case "before":
		return []string{"&::before"}, nil
	case "after":
		return []string{"&::after"}, nil
	case "motion":
		if len(parts) == 1 {
			switch parts[0].Ident {
			case "safe":
				return []string{"@media (prefers-reduced-motion: no-preference)"}, nil
			case "reduce":
				return []string{"@media (prefers-reduced-motion: reduce)"}, nil
			}
		}
	case "contrast":
		if len(parts) == 1 {
			switch parts[0].Ident {
			case "more":
				return []string{"@media (prefers-contrast: more)"}, nil
			case "less":
				return []string{"@media (prefers-contrast: less)"}, nil
			}
		}
	case "forced":
		if len(parts) == 2 && parts[0].Ident == "color" && parts[1].Ident == "adjust" {
			return nil, errConditionUnknown
		}
		if len(parts) == 1 && parts[0].Ident == "colors" {
			return []string{"@media (forced-colors: active)"}, nil
		}
	case "inverted":
		if len(parts) == 1 && parts[0].Ident == "colors" {
			return []string{"@media (inverted-colors: inverted)"}, nil
		}
	case "pointer":
		if len(parts) == 1 {
			switch parts[0].Ident {
			case "none":
				return []string{"@media (pointer: none)"}, nil
			case "coarse":
				return []string{"@media (pointer: coarse)"}, nil
			case "fine":
				return []string{"@media (pointer: fine)"}, nil
			}
		}
	case "any":
		if len(parts) == 2 && parts[0].Ident == "pointer" {
			switch parts[1].Ident {
			case "none":
				return []string{"@media (any-pointer: none)"}, nil
			case "coarse":
				return []string{"@media (any-pointer: coarse)"}, nil
			case "fine":
				return []string{"@media (any-pointer: fine)"}, nil
			}
		}
	case "noscript":
		return []string{"@media (scripting: none)"}, nil
	case "portrait":
		return []string{"@media (orientation: portrait)"}, nil
	case "landscape":
		return []string{"@media (orientation: landscape)"}, nil
	case "dark":
		return []string{"@media (prefers-color-scheme: dark)"}, nil
	case "print":
		return []string{"@media print"}, nil
	case "@":
		if len(parts) == 0 {
			return nil, errConditionUnknown
		}

		containerName := ""
		if s.Modifier != nil && s.Modifier.Ident != "" {
			containerName = s.Modifier.Ident + " "
		}

		if parts[0].Ident == "max" {
			width := g.cssVar(ctx, fmt.Sprintf("--container-%s", partsJoin(parts[1:])))
			if width == "" {
				return nil, errConditionUnknown
			}
			return []string{fmt.Sprintf("@container %s(width < %s)", containerName, width)}, nil
		}

		queryParts := parts
		if parts[0].Ident == "min" {
			queryParts = parts[1:]
		}

		if len(queryParts) == 1 && queryParts[0].Arbitrary != "" {
			width := decodeArbitraryVariant(strings.TrimPrefix(strings.TrimSuffix(queryParts[0].Arbitrary, "]"), "["))
			return []string{fmt.Sprintf("@container %s(width >= %s)", containerName, width)}, nil
		}

		width := g.cssVar(ctx, fmt.Sprintf("--container-%s", partsJoin(queryParts)))
		if width == "" {
			return nil, errConditionUnknown
		}
		return []string{fmt.Sprintf("@container %s(width >= %s)", containerName, width)}, nil
	}

	return nil, errConditionUnknown
}

type utilityGenerated struct {
	kvs    []KV
	others []Query
}

func newUtilityGeneratedKV(k, v string) *utilityGenerated {
	return &utilityGenerated{
		kvs: newSimpleCssKV(k, v),
	}
}

func newUtilityGeneratedKVf(k, format string, a ...any) *utilityGenerated {
	return &utilityGenerated{
		kvs: newSimpleCssKVf(k, format, a...),
	}
}

func (g *gen) utility(ctx context.Context, s *parser.Segment) (*utilityGenerated, error) {
	if s.Name == nil {
		return nil, errUtilityNoName
	}

	switch {
	case s.Name.Head() == "aspect":
		return g.utilityAspect(ctx, s.Name.Parts(), s.Modifier)
	case s.Name.Head() == "columns":
		return g.utilityColumns(ctx, s.Name.Parts())
	case s.Name.Head() == "break":
		return g.utilityBreak(s.Name.Parts())
	case s.Name.Head() == "box":
		return g.utilityBox(s.Name.Parts())
	case s.Name.Head() == "float":
		return g.utilityFloat(s.Name.Parts())
	case s.Name.Head() == "clear":
		return g.utilityClear(s.Name.Parts())
	case s.Name.Head() == "object":
		return g.utilityObjectFitPosition(ctx, s.Name.Parts())
	case s.Name.Head() == "overflow":
		return g.utilityOverflow(s.Name.Parts())
	case s.Name.Head() == "overscroll":
		return g.utilityOverscroll(s.Name.Parts())
	case s.Name.Head() == "inset":
		return g.utilityInset(ctx, s.Name.Parts(), s.Negative, s.Modifier)
	case len(s.Name.Parts()) == 1 && slices.Contains([]string{"top", "right", "left", "bottom"}, s.Name.Head()):
		return g.utilityPlacement(ctx, s.Name.Head(), s.Name.Parts()[0], s.Negative, s.Modifier)
	case s.Name.Head() == "z":
		return g.utilityZIndex(ctx, s.Name.Parts())
	case s.Name.Head() == "opacity":
		return g.utilityOpacity(ctx, s.Name.Parts(), s.Modifier)
	case s.Name.Head() == "mix":
		return g.utilityMix(s.Name.Parts(), s.Negative, s.Modifier)
	case s.Name.Head() == "basis":
		return g.utilityFlexBasis(ctx, s.Name.Parts(), s.Modifier)
	case s.Name.Head() == "grow" || s.Name.Head() == "shrink":
		return g.utilityFlexGrowShrink(ctx, s.Name.Head(), s.Name.Parts())
	case s.Name.Head() == "order" && len(s.Name.Parts()) == 1:
		return g.utilityOrder(ctx, s.Name.Parts()[0], s.Negative)
	case s.Name.Head() == "col":
		return g.utilityColRow(ctx, "grid-column", s.Name.Parts(), s.Negative)
	case s.Name.Head() == "row":
		return g.utilityColRow(ctx, "grid-row", s.Name.Parts(), s.Negative)
	case s.Name.Head() == "auto" && len(s.Name.Parts()) == 2:
		p0, p1 := s.Name.Parts()[0], s.Name.Parts()[1]
		switch p0.Ident {
		case "cols":
			return g.utilityGridAuto(ctx, "grid-auto-columns", p1)
		case "rows":
			return g.utilityGridAuto(ctx, "grid-auto-rows", p1)
		}
	case s.Name.Head() == "gap" && len(s.Name.Parts()) >= 1:
		p0 := s.Name.Parts()[0]

		switch len(s.Name.Parts()) {
		case 1:
			return g.utilityGap(ctx, "gap", p0)
		case 2:
			p1 := s.Name.Parts()[1]
			switch p0.Ident {
			case "x":
				return g.utilityGap(ctx, "column-gap", p1)
			case "y":
				return g.utilityGap(ctx, "row-gap", p1)
			}
		}
	case s.Name.Head() == "justify":
		return g.utilityJustify(s.Name.Parts())
	case s.Name.Head() == "content":
		return g.utilityContent(ctx, s.Name.Parts())
	case s.Name.Head() == "items":
		return g.utilityItems(s.Name.Parts())
	case s.Name.Head() == "self":
		return g.utilitySelf(s.Name.Parts())
	case s.Name.Head() == "place":
		return g.utilityPlace(s.Name.Parts())

	// padding and margin
	case s.Name.Head() == "p" && len(s.Name.Parts()) == 1:
		return g.utilityPadding(ctx, "padding", s.Name.Parts()[0])
	case s.Name.Head() == "px" && len(s.Name.Parts()) == 1:
		return g.utilityPadding(ctx, "padding-inline", s.Name.Parts()[0])
	case s.Name.Head() == "py" && len(s.Name.Parts()) == 1:
		return g.utilityPadding(ctx, "padding-block", s.Name.Parts()[0])
	case s.Name.Head() == "ps" && len(s.Name.Parts()) == 1:
		return g.utilityPadding(ctx, "padding-inline-start", s.Name.Parts()[0])
	case s.Name.Head() == "pe" && len(s.Name.Parts()) == 1:
		return g.utilityPadding(ctx, "padding-inline-end", s.Name.Parts()[0])
	case s.Name.Head() == "pbs" && len(s.Name.Parts()) == 1:
		return g.utilityPadding(ctx, "padding-block-start", s.Name.Parts()[0])
	case s.Name.Head() == "pbe" && len(s.Name.Parts()) == 1:
		return g.utilityPadding(ctx, "padding-block-end", s.Name.Parts()[0])
	case s.Name.Head() == "pt" && len(s.Name.Parts()) == 1:
		return g.utilityPadding(ctx, "padding-top", s.Name.Parts()[0])
	case s.Name.Head() == "pr" && len(s.Name.Parts()) == 1:
		return g.utilityPadding(ctx, "padding-right", s.Name.Parts()[0])
	case s.Name.Head() == "pb" && len(s.Name.Parts()) == 1:
		return g.utilityPadding(ctx, "padding-bottom", s.Name.Parts()[0])
	case s.Name.Head() == "pl" && len(s.Name.Parts()) == 1:
		return g.utilityPadding(ctx, "padding-left", s.Name.Parts()[0])
	case s.Name.Head() == "m" && len(s.Name.Parts()) == 1:
		return g.utilityMargin(ctx, "margin", s.Name.Parts()[0], s.Negative)
	case s.Name.Head() == "mx" && len(s.Name.Parts()) == 1:
		return g.utilityMargin(ctx, "margin-inline", s.Name.Parts()[0], s.Negative)
	case s.Name.Head() == "my" && len(s.Name.Parts()) == 1:
		return g.utilityMargin(ctx, "margin-block", s.Name.Parts()[0], s.Negative)
	case s.Name.Head() == "ms" && len(s.Name.Parts()) == 1:
		return g.utilityMargin(ctx, "margin-inline-start", s.Name.Parts()[0], s.Negative)
	case s.Name.Head() == "me" && len(s.Name.Parts()) == 1:
		return g.utilityMargin(ctx, "margin-inline-end", s.Name.Parts()[0], s.Negative)
	case s.Name.Head() == "mbs" && len(s.Name.Parts()) == 1:
		return g.utilityMargin(ctx, "margin-block-start", s.Name.Parts()[0], s.Negative)
	case s.Name.Head() == "mbe" && len(s.Name.Parts()) == 1:
		return g.utilityMargin(ctx, "margin-block-end", s.Name.Parts()[0], s.Negative)
	case s.Name.Head() == "mt" && len(s.Name.Parts()) == 1:
		return g.utilityMargin(ctx, "margin-top", s.Name.Parts()[0], s.Negative)
	case s.Name.Head() == "mr" && len(s.Name.Parts()) == 1:
		return g.utilityMargin(ctx, "margin-right", s.Name.Parts()[0], s.Negative)
	case s.Name.Head() == "mb" && len(s.Name.Parts()) == 1:
		return g.utilityMargin(ctx, "margin-bottom", s.Name.Parts()[0], s.Negative)
	case s.Name.Head() == "ml" && len(s.Name.Parts()) == 1:
		return g.utilityMargin(ctx, "margin-left", s.Name.Parts()[0], s.Negative)
	case s.Name.Head() == "space":
		return g.utilitySpace(ctx, s.Name.Parts(), s.Negative)

	case s.Name.Head() == "container" && len(s.Name.Parts()) == 0:
		return &utilityGenerated{
			kvs: newSimpleCssKV("width", "100%"),
			others: []Query{
				{Name: "@media (width >= 40rem)", KVs: newSimpleCssKV("max-width", "40rem")},
				{Name: "@media (width >= 48rem)", KVs: newSimpleCssKV("max-width", "48rem")},
				{Name: "@media (width >= 64rem)", KVs: newSimpleCssKV("max-width", "64rem")},
				{Name: "@media (width >= 80rem)", KVs: newSimpleCssKV("max-width", "80rem")},
				{Name: "@media (width >= 96rem)", KVs: newSimpleCssKV("max-width", "96rem")},
			},
		}, nil
	case s.Name.Head() == "@" && len(s.Name.Parts()) >= 1 && s.Name.Parts()[0].Ident == "container":
		return g.utilityAtContainer(ctx, s.Name.Parts()[1:], s.Modifier)

	// sizing

	case s.Name.Head() == "w" && len(s.Name.Parts()) == 1:
		return g.utilitySizing(ctx, "width", s.Name.Parts()[0], s.Modifier, isSimpleSize, simpleSize)
	case s.Name.Head() == "h" && len(s.Name.Parts()) == 1:
		return g.utilitySizing(ctx, "height", s.Name.Parts()[0], s.Modifier, isSimpleSizeHeight, simpleSizeHeight)
	case s.Name.Head() == "size" && len(s.Name.Parts()) == 1:
		return g.utilitySize(ctx, s.Name.Parts()[0], s.Modifier)
	case s.Name.Head() == "min" && len(s.Name.Parts()) == 2:
		return g.utilityMin(ctx, s.Name.Parts(), s.Modifier)
	case s.Name.Head() == "max" && len(s.Name.Parts()) == 2:
		return g.utilityMax(ctx, s.Name.Parts(), s.Modifier)

	// isolation entries

	case s.Name.Head() == "isolate" && len(s.Name.Parts()) == 0:
		return newUtilityGeneratedKV("isolation", "isolate"), nil
	case s.Name.Head() == "isolation" && len(s.Name.Parts()) == 1 && s.Name.Parts()[0].Ident == "auto":
		return newUtilityGeneratedKV("isolation", "auto"), nil

	// display entries

	case s.Name.Head() == "inline":
		return g.utilityInline(ctx, s.Name.Parts(), s.Modifier)
	case s.Name.Head() == "block":
		return g.utilityBlock(ctx, s.Name.Parts(), s.Modifier)
	case s.Name.Head() == "flow" && len(s.Name.Parts()) == 1 && s.Name.Parts()[0].Ident == "root":
		return newUtilityGeneratedKV("display", "flow-root"), nil
	case s.Name.Head() == "flex":
		return g.utilityFlex(ctx, s.Name.Parts(), s.Modifier)
	case s.Name.Head() == "grid":
		return g.utilityGrid(ctx, s.Name.Parts())
	case s.Name.Head() == "hidden":
		return newUtilityGeneratedKV("display", "none"), nil
	case s.Name.Head() == "contents":
		return newUtilityGeneratedKV("display", "contents"), nil
	case s.Name.Head() == "sr" && len(s.Name.Parts()) == 1 && s.Name.Parts()[0].Ident == "only":
		return &utilityGenerated{kvs: []KV{
			{Key: "position", Value: "absolute"},
			{Key: "width", Value: "1px"},
			{Key: "height", Value: "1px"},
			{Key: "padding", Value: "0"},
			{Key: "margin", Value: "-1px"},
			{Key: "overflow", Value: "hidden"},
			{Key: "clip-path", Value: "inset(50%)"},
			{Key: "white-space", Value: "nowrap"},
			{Key: "border-width", Value: "0"},
		}}, nil
	case s.Name.Head() == "not" && simpleMatch(s.Name.Parts(), "sr", "only"):
		return &utilityGenerated{kvs: []KV{
			{Key: "position", Value: "static"},
			{Key: "width", Value: "auto"},
			{Key: "height", Value: "auto"},
			{Key: "padding", Value: "0"},
			{Key: "margin", Value: "0"},
			{Key: "overflow", Value: "visible"},
			{Key: "clip-path", Value: "none"},
			{Key: "white-space", Value: "normal"},
		}}, nil
	case s.Name.Head() == "table":
		return g.utilityTable(ctx, s.Name.Parts())
	case s.Name.Head() == "caption":
		return g.utilityCaption(s.Name.Parts())

	// typography

	case s.Name.Head() == "font":
		return g.utilityFont(ctx, s.Name.Parts())
	case s.Name.Head() == "text":
		return g.utilityText(ctx, s.Name.Parts(), s.Modifier)
	case s.Name.Head() == "antialiased" && len(s.Name.Parts()) == 0:
		return &utilityGenerated{
			kvs: []KV{
				g.cssKV(ctx, "-webkit-font-smoothing", "antialiased"),
				g.cssKV(ctx, "-moz-osx-font-smoothing", "grayscale"),
			},
		}, nil
	case s.Name.Head() == "subpixel" && simpleMatch(s.Name.Parts(), "antialiased"):
		return &utilityGenerated{
			kvs: []KV{
				g.cssKV(ctx, "-webkit-font-smoothing", "auto"),
				g.cssKV(ctx, "-moz-osx-font-smoothing", "auto"),
			},
		}, nil
	case s.Name.Head() == "italic" && len(s.Name.Parts()) == 0:
		return newUtilityGeneratedKV("font-style", "italic"), nil
	case s.Name.Head() == "not" && simpleMatch(s.Name.Parts(), "italic"):
		return newUtilityGeneratedKV("font-style", "normal"), nil

	// font-variant-numeric

	case s.Name.Head() == "normal" && simpleMatch(s.Name.Parts(), "nums"):
		return newUtilityGeneratedKV("font-variant-numeric", "normal"), nil
	case s.Name.Head() == "ordinal" && len(s.Name.Parts()) == 0:
		return newUtilityGeneratedKV("font-variant-numeric", "ordinal"), nil
	case s.Name.Head() == "slashed" && simpleMatch(s.Name.Parts(), "zero"):
		return newUtilityGeneratedKV("font-variant-numeric", "slashed-zero"), nil
	case s.Name.Head() == "lining" && simpleMatch(s.Name.Parts(), "nums"):
		return newUtilityGeneratedKV("font-variant-numeric", "lining-nums"), nil
	case s.Name.Head() == "oldstyle" && simpleMatch(s.Name.Parts(), "nums"):
		return newUtilityGeneratedKV("font-variant-numeric", "oldstyle-nums"), nil
	case s.Name.Head() == "proportional" && simpleMatch(s.Name.Parts(), "nums"):
		return newUtilityGeneratedKV("font-variant-numeric", "proportional-nums"), nil
	case s.Name.Head() == "tabular" && simpleMatch(s.Name.Parts(), "nums"):
		return newUtilityGeneratedKV("font-variant-numeric", "tabular-nums"), nil
	case s.Name.Head() == "diagonal" && simpleMatch(s.Name.Parts(), "fractions"):
		return newUtilityGeneratedKV("font-variant-numeric", "diagonal-fractions"), nil
	case s.Name.Head() == "stacked" && simpleMatch(s.Name.Parts(), "fractions"):
		return newUtilityGeneratedKV("font-variant-numeric", "stacked-fractions"), nil

	case s.Name.Head() == "tracking" && len(s.Name.Parts()) == 1:
		p := s.Name.Parts()[0]
		return g.utilityTracking(ctx, p)
	case s.Name.Head() == "line":
		return g.utilityLine(ctx, s.Name.Parts())
	case s.Name.Head() == "leading" && len(s.Name.Parts()) == 1:
		p := s.Name.Parts()[0]
		return g.utilityLeading(ctx, p)
	case s.Name.Head() == "list":
		return g.utilityList(ctx, s.Name.Parts())
	case s.Name.Head() == "tab" && len(s.Name.Parts()) == 1:
		return g.utilityTab(ctx, s.Name.Parts()[0])
	case s.Name.Head() == "align" && len(s.Name.Parts()) >= 1:
		return g.utilityAlign(ctx, s.Name.Parts())
	case s.Name.Head() == "whitespace" && len(s.Name.Parts()) >= 1:
		return g.utilityWhitespace(s.Name.Parts())

	case s.Name.Head() == "wrap":
		return g.utilityWrap(s.Name.Parts())
	case s.Name.Head() == "hyphens":
		return g.utilityHyphens(s.Name.Parts())

	case s.Name.Head() == "underline":
		return g.utilityUnderline(ctx, s.Name.Parts(), s.Negative)
	case s.Name.Head() == "overline" && len(s.Name.Parts()) == 0:
		return newUtilityGeneratedKV("text-decoration-line", "overline"), nil
	case s.Name.Head() == "no" && len(s.Name.Parts()) == 1 && s.Name.Parts()[0].Ident == "underline":
		return newUtilityGeneratedKV("text-decoration-line", "none"), nil
	case s.Name.Head() == "decoration":
		return g.utilityDecoration(ctx, s.Name.Parts(), s.Modifier)

	// text transform

	case len(s.Name.Parts()) == 0 && slices.Contains([]string{
		"uppercase", "lowercase", "capitalize",
	}, s.Name.Head()):
		return newUtilityGeneratedKV("text-transform", s.Name.Head()), nil
	case s.Name.Head() == "normal" && simpleMatch(s.Name.Parts(), "case"):
		return newUtilityGeneratedKV("text-transform", "none"), nil

	// text overflow

	case s.Name.Head() == "truncate" && len(s.Name.Parts()) == 0:
		return &utilityGenerated{
			kvs: []KV{
				g.cssKV(ctx, "overflow", "hidden"),
				g.cssKV(ctx, "text-overflow", "ellipsis"),
				g.cssKV(ctx, "white-space", "nowrap"),
			},
		}, nil

	case s.Name.Head() == "indent" && len(s.Name.Parts()) == 1:
		return g.utilityIndent(ctx, s.Name.Parts()[0], s.Negative)

	// position entries

	case len(s.Name.Parts()) == 0 && s.Name.Head() == "static",
		len(s.Name.Parts()) == 0 && s.Name.Head() == "fixed",
		len(s.Name.Parts()) == 0 && s.Name.Head() == "absolute",
		len(s.Name.Parts()) == 0 && s.Name.Head() == "relative",
		len(s.Name.Parts()) == 0 && s.Name.Head() == "sticky":
		return newUtilityGeneratedKV("position", s.Name.Head()), nil

		// visibility entries
	case len(s.Name.Parts()) == 0 && s.Name.Head() == "visible",
		len(s.Name.Parts()) == 0 && s.Name.Head() == "collapse":
		return newUtilityGeneratedKV("visibility", s.Name.Head()), nil
	case len(s.Name.Parts()) == 0 && s.Name.Head() == "invisible":
		return newUtilityGeneratedKV("visibility", "hidden"), nil

	case s.Name.Head() == "bg":
		return g.utilityBackground(ctx, s.Name.Parts(), s.Negative, s.Modifier)
	case s.Name.Head() == "mask":
		return g.utilityMask(ctx, s.Name.Parts(), s.Negative, s.Modifier)
	case s.Name.Head() == "from":
		return g.utilityFrom(ctx, s.Name.Parts(), s.Modifier)
	case s.Name.Head() == "via":
		return g.utilityVia(ctx, s.Name.Parts(), s.Modifier)
	case s.Name.Head() == "to":
		return g.utilityTo(ctx, s.Name.Parts(), s.Modifier)

	case s.Name.Head() == "rounded":
		return g.utilityRounded(ctx, s.Name.Parts())
	case s.Name.Head() == "border":
		return g.utilityBorder(ctx, s.Name.Parts(), s.Modifier)
	case s.Name.Head() == "divide":
		return g.utilityDivide(ctx, s.Name.Parts(), s.Modifier)
	case s.Name.Head() == "outline":
		return g.utilityOutline(ctx, s.Name.Parts(), s.Negative, s.Modifier)
	case s.Name.Head() == "shadow":
		return g.utilityShadow(ctx, s.Name.Parts(), s.Modifier)
	case s.Name.Head() == "ring":
		return g.utilityRing(ctx, s.Name.Parts(), s.Modifier)
	case s.Name.Head() == "filter":
		return g.utilityFilter(ctx, s.Name.Parts(), s.Modifier)
	case s.Name.Head() == "blur":
		return g.utilityBlur(ctx, s.Name.Parts(), s.Modifier)
	case s.Name.Head() == "brightness":
		return g.utilityBrightness(ctx, s.Name.Parts(), s.Modifier)
	case s.Name.Head() == "contrast":
		return g.utilityContrast(ctx, s.Name.Parts(), s.Modifier)
	case s.Name.Head() == "grayscale":
		return g.utilityGrayscale(ctx, s.Name.Parts(), s.Modifier)
	case s.Name.Head() == "invert":
		return g.utilityInvert(ctx, s.Name.Parts(), s.Modifier)
	case s.Name.Head() == "saturate":
		return g.utilitySaturate(ctx, s.Name.Parts(), s.Modifier)
	case s.Name.Head() == "sepia":
		return g.utilitySepia(ctx, s.Name.Parts(), s.Modifier)
	case s.Name.Head() == "hue":
		return g.utilityHueRotate(ctx, s.Name.Parts(), s.Negative, s.Modifier)
	case s.Name.Head() == "drop":
		return g.utilityDropShadow(ctx, s.Name.Parts(), s.Modifier)
	case s.Name.Head() == "backdrop":
		return g.utilityBackdrop(ctx, s.Name.Parts(), s.Negative, s.Modifier)
	case s.Name.Head() == "transition":
		return g.utilityTransition(ctx, s.Name.Parts(), s.Modifier)
	case s.Name.Head() == "delay":
		return g.utilityTransitionDelay(ctx, s.Name.Parts(), s.Modifier)
	case s.Name.Head() == "duration":
		return g.utilityTransitionDuration(ctx, s.Name.Parts(), s.Modifier)
	case s.Name.Head() == "ease":
		return g.utilityTransitionTimingFunction(ctx, s.Name.Parts(), s.Modifier)
	case s.Name.Head() == "animate":
		return g.utilityAnimate(ctx, s.Name.Parts(), s.Modifier)
	case s.Name.Head() == "translate":
		return g.utilityTranslate(ctx, s.Name.Parts(), s.Negative, s.Modifier)
	case s.Name.Head() == "scale":
		return g.utilityScale(ctx, s.Name.Parts(), s.Negative, s.Modifier)
	case s.Name.Head() == "rotate":
		return g.utilityRotate(ctx, s.Name.Parts(), s.Negative, s.Modifier)
	case s.Name.Head() == "skew":
		return g.utilitySkew(ctx, s.Name.Parts(), s.Negative, s.Modifier)
	case s.Name.Head() == "transform":
		return g.utilityTransform(ctx, s.Name.Parts(), s.Modifier)
	case s.Name.Head() == "origin":
		return g.utilityOrigin(ctx, s.Name.Parts(), s.Modifier)
	case s.Name.Head() == "perspective":
		return g.utilityPerspective(ctx, s.Name.Parts(), s.Modifier)
	case s.Name.Head() == "backface":
		return g.utilityBackface(s.Name.Parts(), s.Modifier)
	case s.Name.Head() == "zoom":
		return g.utilityZoom(ctx, s.Name.Parts(), s.Modifier)
	case s.Name.Head() == "pointer":
		return g.utilityPointerEvents(s.Name.Parts(), s.Modifier)
	case s.Name.Head() == "cursor":
		return g.utilityCursor(ctx, s.Name.Parts(), s.Modifier)
	case s.Name.Head() == "touch":
		return g.utilityTouchAction(ctx, s.Name.Parts(), s.Modifier)
	case s.Name.Head() == "select":
		return g.utilityUserSelect(ctx, s.Name.Parts(), s.Modifier)
	case s.Name.Head() == "resize":
		return g.utilityResize(s.Name.Parts(), s.Modifier)
	case s.Name.Head() == "snap":
		return g.utilitySnap(ctx, s.Name.Parts(), s.Modifier)
	case s.Name.Head() == "scroll":
		return g.utilityScroll(ctx, s.Name.Parts(), s.Negative, s.Modifier)
	case s.Name.Head() == "scrollbar":
		return g.utilityScrollbar(ctx, s.Name.Parts(), s.Modifier)
	case s.Name.Head() == "appearance":
		return g.utilityAppearance(s.Name.Parts(), s.Modifier)
	case s.Name.Head() == "scheme":
		return g.utilityColorScheme(s.Name.Parts(), s.Modifier)
	case s.Name.Head() == "accent":
		return g.utilityAccentColor(ctx, s.Name.Parts(), s.Modifier)
	case s.Name.Head() == "caret":
		return g.utilityCaretColor(ctx, s.Name.Parts(), s.Modifier)
	case s.Name.Head() == "field":
		return g.utilityFieldSizing(s.Name.Parts(), s.Modifier)
	case s.Name.Head() == "will":
		return g.utilityWillChange(ctx, s.Name.Parts(), s.Modifier)
	case s.Name.Head() == "fill":
		return g.utilityFill(ctx, s.Name.Parts(), s.Modifier)
	case s.Name.Head() == "stroke":
		return g.utilityStroke(ctx, s.Name.Parts(), s.Modifier)
	case s.Name.Head() == "forced":
		return g.utilityForcedColorAdjust(s.Name.Parts(), s.Modifier)
	}

	return nil, errUtilityUnknown
}

func (g *gen) utilityAspect(ctx context.Context, parts []*parser.NamePart, s *parser.SlashValue) (*utilityGenerated, error) {
	if len(parts) != 1 {
		return nil, fmt.Errorf("only one part is allowed for aspect")
	}

	p := parts[0]
	k := "aspect-ratio"

	switch {
	case p.Ident == "square":
		return newUtilityGeneratedKV(k, "1 / 1"), nil
	case p.Ident == "video":
		return newUtilityGeneratedKV(k, g.cssVar(ctx, "--aspect-video")), nil
	case p.Ident == "auto":
		return newUtilityGeneratedKV(k, "auto"), nil
	case p.Int != nil && s != nil && s.Int != nil:
		return newUtilityGeneratedKV(k, fmt.Sprintf("%d / %d", *p.Int, *s.Int)), nil
	case p.Custom != "":
		return g.cssCustom(ctx, k, p), nil
	case p.Arbitrary != "":
		return g.cssArbitrary(ctx, k, p), nil
	}

	return nil, errUtilityUnknown
}

func (g *gen) utilityColumns(ctx context.Context, parts []*parser.NamePart) (*utilityGenerated, error) {
	if len(parts) != 1 {
		return nil, fmt.Errorf("only one part is allowed for columns")
	}

	p := parts[0]
	k := "columns"

	switch {
	case p.Ident == "auto":
		return newUtilityGeneratedKV(k, "auto"), nil
	case isContainerSize(p.AlphaNum) || isContainerSize(p.Ident):
		return newUtilityGeneratedKV(k, g.cssVar(ctx, fmt.Sprintf("--container-%s", p.String()))), nil
	case p.Int != nil:
		return newUtilityGeneratedKV(k, p.String()), nil
	case p.Custom != "":
		return g.cssCustom(ctx, k, p), nil
	case p.Arbitrary != "":
		return g.cssArbitrary(ctx, k, p), nil
	}

	return nil, errUtilityUnknown
}

func (g *gen) utilityBreak(parts []*parser.NamePart) (*utilityGenerated, error) {
	l := len(parts)
	if l < 1 || l > 3 {
		return nil, fmt.Errorf("there should be one, two or three parts for break utility")
	}

	switch {
	case simpleMatch(parts, "normal"):
		return newUtilityGeneratedKV("word-break", "normal"), nil
	case simpleMatch(parts, "all"):
		return newUtilityGeneratedKV("word-break", "break-all"), nil
	case simpleMatch(parts, "keep"):
		return newUtilityGeneratedKV("word-break", "keep-all"), nil
	case simpleMatch(parts, "after", "auto"):
	case simpleMatch(parts, "after", "avoid"):
	case simpleMatch(parts, "after", "all"):
	case simpleMatch(parts, "after", "avoid", "page"):
	case simpleMatch(parts, "after", "page"):
	case simpleMatch(parts, "after", "left"):
	case simpleMatch(parts, "after", "right"):
	case simpleMatch(parts, "after", "column"):
	case simpleMatch(parts, "before", "auto"):
	case simpleMatch(parts, "before", "avoid"):
	case simpleMatch(parts, "before", "all"):
	case simpleMatch(parts, "before", "avoid", "page"):
	case simpleMatch(parts, "before", "page"):
	case simpleMatch(parts, "before", "left"):
	case simpleMatch(parts, "before", "right"):
	case simpleMatch(parts, "before", "column"):
	case simpleMatch(parts, "inside", "auto"):
	case simpleMatch(parts, "inside", "avoid"):
	case simpleMatch(parts, "inside", "avoid", "page"):
	case simpleMatch(parts, "inside", "avoid", "column"):
	default:
		return nil, errUtilityUnknown
	}

	k := "break-" + parts[0].Ident
	v := partsJoin(parts[1:])

	return newUtilityGeneratedKV(k, v), nil
}

func (g *gen) utilityBox(parts []*parser.NamePart) (*utilityGenerated, error) {
	switch len(parts) {
	case 1, 2:
	default:
		return nil, fmt.Errorf("there should be one or two parts for box-* utility")
	}

	switch {
	case simpleMatch(parts, "border"):
		return newUtilityGeneratedKV("box-sizing", "border-box"), nil
	case simpleMatch(parts, "content"):
		return newUtilityGeneratedKV("box-sizing", "content-box"), nil
	case simpleMatch(parts, "decoration", "clone"):
		return newUtilityGeneratedKV("box-decoration-break", "clone"), nil
	case simpleMatch(parts, "decoration", "slice"):
		return newUtilityGeneratedKV("box-decoration-break", "slice"), nil
	}

	return nil, errUtilityUnknown
}

func (g *gen) utilityFloat(parts []*parser.NamePart) (*utilityGenerated, error) {
	if len(parts) != 1 {
		return nil, fmt.Errorf("there should be only one part for float-* utility")
	}

	switch {
	case slices.Contains([]string{"right", "left", "none"}, parts[0].Ident):
		return newUtilityGeneratedKV("float", parts[0].Ident), nil
	case slices.Contains([]string{"start", "end"}, parts[0].Ident):
		return newUtilityGeneratedKV("float", fmt.Sprintf("inline-%s", parts[0].Ident)), nil
	}

	return nil, errUtilityUnknown
}

func (g *gen) utilityClear(parts []*parser.NamePart) (*utilityGenerated, error) {
	if len(parts) != 1 {
		return nil, fmt.Errorf("there should be only one part for clear-* utility")
	}

	switch {
	case slices.Contains([]string{"right", "left", "none", "both"}, parts[0].Ident):
		return newUtilityGeneratedKV("clear", parts[0].Ident), nil
	case slices.Contains([]string{"start", "end"}, parts[0].Ident):
		return newUtilityGeneratedKV("clear", fmt.Sprintf("inline-%s", parts[0].Ident)), nil
	}

	return nil, errUtilityUnknown
}

func (g *gen) utilityObjectFitPosition(ctx context.Context, parts []*parser.NamePart) (*utilityGenerated, error) {
	switch len(parts) {
	case 1, 2:
	default:
		return nil, fmt.Errorf("there should be only one or two parts for object-* utility")
	}

	switch {

	// object-fit

	case slices.Contains([]string{"contain", "none", "fill", "cover"}, parts[0].Ident):
		return newUtilityGeneratedKV("object-fit", parts[0].Ident), nil
	case simpleMatch(parts, "scale", "down"):
		return newUtilityGeneratedKV("object-fit", "scale-down"), nil

	// object-position

	case simpleMatch(parts, "top", "right"):
		return newUtilityGeneratedKV("object-position", "top right"), nil
	case simpleMatch(parts, "top", "left"):
		return newUtilityGeneratedKV("object-position", "top left"), nil
	case simpleMatch(parts, "bottom", "right"):
		return newUtilityGeneratedKV("object-position", "bottom right"), nil
	case simpleMatch(parts, "bottom", "left"):
		return newUtilityGeneratedKV("object-position", "bottom left"), nil
	case slices.Contains([]string{"top", "bottom", "left", "right", "center"}, parts[0].Ident):
		return newUtilityGeneratedKV("object-position", parts[0].Ident), nil

	case len(parts) == 1 && parts[0].Custom != "":
		return g.cssCustom(ctx, "object-position", parts[0]), nil
	case len(parts) == 1 && parts[0].Arbitrary != "":
		return g.cssArbitrary(ctx, "object-position", parts[0]), nil

	}

	return nil, errUtilityUnknown
}

func (g *gen) utilityOverflow(parts []*parser.NamePart) (*utilityGenerated, error) {
	switch len(parts) {
	case 1, 2:
	default:
		return nil, fmt.Errorf("there should be only one or two parts for overflow-* utility")
	}

	overflowTerms := []string{"auto", "hidden", "clip", "visible", "scroll"}

	switch {
	case len(parts) == 1 &&
		slices.Contains(overflowTerms, parts[0].Ident):
		return newUtilityGeneratedKV("overflow", parts[0].Ident), nil

	case len(parts) == 2 &&
		slices.Contains([]string{"x", "y"}, parts[0].Ident) &&
		slices.Contains(overflowTerms, parts[1].Ident):
		return newUtilityGeneratedKV(fmt.Sprintf("overflow-%s", parts[0].Ident), parts[1].Ident), nil
	}

	return nil, errUtilityUnknown
}

func (g *gen) utilityOverscroll(parts []*parser.NamePart) (*utilityGenerated, error) {
	switch len(parts) {
	case 1, 2:
	default:
		return nil, fmt.Errorf("there should be only one or two parts for overscroll-* utility")
	}

	overscrollTerms := []string{"auto", "contain", "none"}

	switch {
	case len(parts) == 1 &&
		slices.Contains(overscrollTerms, parts[0].Ident):
		return newUtilityGeneratedKV("overscroll-behavior", parts[0].Ident), nil

	case len(parts) == 2 &&
		slices.Contains([]string{"x", "y"}, parts[0].Ident) &&
		slices.Contains(overscrollTerms, parts[1].Ident):
		return newUtilityGeneratedKV(fmt.Sprintf("overscroll-behavior-%s", parts[0].Ident), parts[1].Ident), nil
	}

	return nil, errUtilityUnknown
}

func (g *gen) utilityPlacement(
	ctx context.Context, k string, p *parser.NamePart, negative bool, mod *parser.SlashValue,
) (*utilityGenerated, error) {
	n := ""
	if negative {
		n = "-"
	}

	switch {
	case p.Int != nil && mod != nil && mod.Int != nil:
		v := fmt.Sprintf("%scalc(%s/%s * 100%%)", n, p.String(), mod.String())
		return newUtilityGeneratedKV(k, v), nil
	case p.Int != nil, p.Float != nil:
		return g.cssCalcSpacing(ctx, k, p, negative), nil
	case p.Ident == "px":
		v := fmt.Sprintf("%s1px", n)
		return newUtilityGeneratedKV(k, v), nil
	case p.Ident == "full":
		v := fmt.Sprintf("%s100%%", n)
		return newUtilityGeneratedKV(k, v), nil
	case p.Ident == "auto":
		return newUtilityGeneratedKV(k, "auto"), nil
	case p.Custom != "":
		return g.cssCustom(ctx, k, p), nil
	case p.Arbitrary != "":
		return g.cssArbitrary(ctx, k, p), nil
	}

	return nil, errUtilityUnknown
}

func (g *gen) utilityInset(
	ctx context.Context, parts []*parser.NamePart, negative bool, mod *parser.SlashValue,
) (*utilityGenerated, error) {
	if len(parts) >= 1 {
		switch parts[0].Ident {
		case "shadow":
			return g.utilityInsetShadow(ctx, parts[1:], mod)
		case "ring":
			return g.utilityInsetRing(ctx, parts[1:], mod)
		}
	}

	switch len(parts) {
	case 1:
		return g.utilityPlacement(ctx, "inset", parts[0], negative, mod)

	case 2:
		f := func(k string) (*utilityGenerated, error) {
			return g.utilityPlacement(ctx, k, parts[1], negative, mod)
		}

		switch parts[0].Ident {
		case "x":
			return f("inset-inline")
		case "y":
			return f("inset-block")
		case "s":
			return f("inset-inline-start")
		case "e":
			return f("inset-inline-end")
		case "bs":
			return f("inset-block-start")
		case "be":
			return f("inset-block-end")
		}

	default:
		return nil, fmt.Errorf("there should be only one or two parts for inset-* utility")
	}

	return nil, errUtilityUnknown
}

func (g *gen) utilityInline(
	ctx context.Context, parts []*parser.NamePart, m *parser.SlashValue,
) (*utilityGenerated, error) {
	kv := func(v string) *utilityGenerated { return newUtilityGeneratedKV("display", v) }

	switch {
	case len(parts) == 0:
		return kv("inline"), nil
	case len(parts) == 1:
		p := parts[0]
		switch {
		case slices.Contains([]string{"block", "flex", "grid", "table"}, p.Ident):
			return kv(fmt.Sprintf("inline-%s", parts[0].Ident)), nil
		default:
			return g.utilitySizing(ctx, "inline-size", p, m, isSimpleSize, simpleSize)
		}
	}

	return nil, errUtilityUnknown
}

func (g *gen) utilityBlock(
	ctx context.Context, parts []*parser.NamePart, m *parser.SlashValue,
) (*utilityGenerated, error) {
	switch {
	case len(parts) == 0:
		return newUtilityGeneratedKV("display", "block"), nil
	case len(parts) == 1:
		k := "block-size"
		p := parts[0]
		switch {
		case p.Int != nil:
			if m != nil && m.Int != nil {
				return newUtilityGeneratedKV(k, fmt.Sprintf("calc(%s/%s * 100%%)", p.String(), m.String())), nil
			}

			return g.cssCalcSpacing(ctx, k, p, false), nil
		case p.Float != nil:
			return g.cssCalcSpacing(ctx, k, p, false), nil
		case isSimpleSizeHeight(p.Ident):
			return newUtilityGeneratedKV(k, simpleSizeHeight(p.Ident)), nil
		case p.Custom != "":
			return g.cssCustom(ctx, k, p), nil
		case p.Arbitrary != "":
			return g.cssArbitrary(ctx, k, p), nil
		}
	}

	return nil, errUtilityUnknown
}

func (g *gen) utilityZIndex(ctx context.Context, parts []*parser.NamePart) (*utilityGenerated, error) {
	if len(parts) != 1 {
		return nil, fmt.Errorf("there should be only one part for z-* utility")
	}

	k := "z-index"
	p := parts[0]

	switch {
	case p.Ident == "auto", p.Int != nil:
		return newUtilityGeneratedKV(k, p.String()), nil
	case p.Custom != "":
		return g.cssCustom(ctx, k, p), nil
	case p.Arbitrary != "":
		return g.cssArbitrary(ctx, k, p), nil
	}

	return nil, errUtilityUnknown
}

func isValidOpacityValue(s string) bool {
	num, err := strconv.ParseFloat(s, 64)
	if err != nil || num < 0 {
		return false
	}

	remainder := math.Mod(num, 0.25)
	return remainder < 1e-9 && strconv.FormatFloat(num, 'f', -1, 64) == s
}

func opacityPercent(p *parser.NamePart) (string, bool) {
	switch {
	case p.Int != nil, p.Float != nil:
		s := p.String()
		if !isValidOpacityValue(s) {
			return "", false
		}

		return s + "%", true
	default:
		return "", false
	}
}

func (g *gen) utilityOpacity(
	ctx context.Context, parts []*parser.NamePart, mod *parser.SlashValue,
) (*utilityGenerated, error) {
	if len(parts) != 1 {
		return nil, fmt.Errorf("there should be only one part for opacity-* utility")
	}

	if mod != nil {
		return nil, fmt.Errorf("opacity-* does not support modifiers")
	}

	k := "opacity"
	p := parts[0]

	switch {
	case p.Int != nil, p.Float != nil:
		if v, ok := opacityPercent(p); ok {
			return newUtilityGeneratedKV(k, v), nil
		}
	case p.Custom != "":
		return g.cssCustom(ctx, k, p), nil
	case p.Arbitrary != "":
		return g.cssArbitrary(ctx, k, p), nil
	}

	return nil, errUtilityUnknown
}

func (g *gen) utilityMix(
	parts []*parser.NamePart, negative bool, mod *parser.SlashValue,
) (*utilityGenerated, error) {
	if negative || mod != nil {
		return nil, errUtilityUnknown
	}

	switch {
	case simpleMatch(parts, "blend", "normal"):
		return newUtilityGeneratedKV("mix-blend-mode", "normal"), nil
	case simpleMatch(parts, "blend", "multiply"):
		return newUtilityGeneratedKV("mix-blend-mode", "multiply"), nil
	case simpleMatch(parts, "blend", "screen"):
		return newUtilityGeneratedKV("mix-blend-mode", "screen"), nil
	case simpleMatch(parts, "blend", "overlay"):
		return newUtilityGeneratedKV("mix-blend-mode", "overlay"), nil
	case simpleMatch(parts, "blend", "darken"):
		return newUtilityGeneratedKV("mix-blend-mode", "darken"), nil
	case simpleMatch(parts, "blend", "lighten"):
		return newUtilityGeneratedKV("mix-blend-mode", "lighten"), nil
	case simpleMatch(parts, "blend", "color", "dodge"):
		return newUtilityGeneratedKV("mix-blend-mode", "color-dodge"), nil
	case simpleMatch(parts, "blend", "color", "burn"):
		return newUtilityGeneratedKV("mix-blend-mode", "color-burn"), nil
	case simpleMatch(parts, "blend", "hard", "light"):
		return newUtilityGeneratedKV("mix-blend-mode", "hard-light"), nil
	case simpleMatch(parts, "blend", "soft", "light"):
		return newUtilityGeneratedKV("mix-blend-mode", "soft-light"), nil
	case simpleMatch(parts, "blend", "difference"):
		return newUtilityGeneratedKV("mix-blend-mode", "difference"), nil
	case simpleMatch(parts, "blend", "exclusion"):
		return newUtilityGeneratedKV("mix-blend-mode", "exclusion"), nil
	case simpleMatch(parts, "blend", "hue"):
		return newUtilityGeneratedKV("mix-blend-mode", "hue"), nil
	case simpleMatch(parts, "blend", "saturation"):
		return newUtilityGeneratedKV("mix-blend-mode", "saturation"), nil
	case simpleMatch(parts, "blend", "color"):
		return newUtilityGeneratedKV("mix-blend-mode", "color"), nil
	case simpleMatch(parts, "blend", "luminosity"):
		return newUtilityGeneratedKV("mix-blend-mode", "luminosity"), nil
	case simpleMatch(parts, "blend", "plus", "darker"):
		return newUtilityGeneratedKV("mix-blend-mode", "plus-darker"), nil
	case simpleMatch(parts, "blend", "plus", "lighter"):
		return newUtilityGeneratedKV("mix-blend-mode", "plus-lighter"), nil
	}

	return nil, errUtilityUnknown
}

func (g *gen) newMaskUtility(ctx context.Context, prop, value string) *utilityGenerated {
	return &utilityGenerated{
		kvs: []KV{
			g.cssKV(ctx, "-webkit-"+prop, value),
			g.cssKV(ctx, prop, value),
		},
	}
}

func maskInferArbitraryProperty(k, v string) string {
	v = arbitraryValue(v)

	switch k {
	case "position":
		return "mask-position"
	case "size", "length":
		return "mask-size"
	case "image", "url":
		return "mask-image"
	}

	switch v {
	case "cover", "contain":
		return "mask-size"
	}

	if strings.HasSuffix(strings.TrimSpace(v), "%") {
		return "mask-position"
	}

	if strings.HasPrefix(v, "url(") || strings.Contains(v, "gradient(") {
		return "mask-image"
	}

	if cssIsLength(v) || strings.Contains(v, " ") {
		return "mask-position"
	}

	return "mask-image"
}

func isMaskArbitraryTypeHint(k string) bool {
	switch k {
	case "position", "size", "length", "image", "url":
		return true
	default:
		return false
	}
}

func (g *gen) maskArbitraryFromPart(ctx context.Context, p *parser.NamePart) *utilityGenerated {
	k, v := p.ArbitraryParts()
	if k != "" && !isMaskArbitraryTypeHint(k) {
		v = strings.Trim(p.Arbitrary, "[]")
		k = ""
	}

	v = arbitraryValue(v)
	prop := maskInferArbitraryProperty(k, v)

	return g.newMaskUtility(ctx, prop, v)
}

func (g *gen) utilityMask(
	ctx context.Context, parts []*parser.NamePart, negative bool, mod *parser.SlashValue,
) (*utilityGenerated, error) {
	if negative || mod != nil {
		return nil, errUtilityUnknown
	}

	l := len(parts)

	switch {
	case simpleMatch(parts, "type", "alpha"):
		return newUtilityGeneratedKV("mask-type", "alpha"), nil
	case simpleMatch(parts, "type", "luminance"):
		return newUtilityGeneratedKV("mask-type", "luminance"), nil

	case simpleMatch(parts, "clip", "border"):
		return g.newMaskUtility(ctx, "mask-clip", "border-box"), nil
	case simpleMatch(parts, "clip", "padding"):
		return g.newMaskUtility(ctx, "mask-clip", "padding-box"), nil
	case simpleMatch(parts, "clip", "content"):
		return g.newMaskUtility(ctx, "mask-clip", "content-box"), nil
	case simpleMatch(parts, "clip", "fill"):
		return g.newMaskUtility(ctx, "mask-clip", "fill-box"), nil
	case simpleMatch(parts, "clip", "stroke"):
		return g.newMaskUtility(ctx, "mask-clip", "stroke-box"), nil
	case simpleMatch(parts, "clip", "view"):
		return g.newMaskUtility(ctx, "mask-clip", "view-box"), nil
	case simpleMatch(parts, "no", "clip"):
		return g.newMaskUtility(ctx, "mask-clip", "no-clip"), nil

	case simpleMatch(parts, "origin", "border"):
		return g.newMaskUtility(ctx, "mask-origin", "border-box"), nil
	case simpleMatch(parts, "origin", "padding"):
		return g.newMaskUtility(ctx, "mask-origin", "padding-box"), nil
	case simpleMatch(parts, "origin", "content"):
		return g.newMaskUtility(ctx, "mask-origin", "content-box"), nil
	case simpleMatch(parts, "origin", "fill"):
		return g.newMaskUtility(ctx, "mask-origin", "fill-box"), nil
	case simpleMatch(parts, "origin", "stroke"):
		return g.newMaskUtility(ctx, "mask-origin", "stroke-box"), nil
	case simpleMatch(parts, "origin", "view"):
		return g.newMaskUtility(ctx, "mask-origin", "view-box"), nil

	case l == 2 && parts[0].Ident == "position" && parts[1].Custom != "":
		_, v := parts[1].CustomParts()
		return g.newMaskUtility(ctx, "mask-position", g.cssVar(ctx, v)), nil
	case l == 2 && parts[0].Ident == "position" && parts[1].Arbitrary != "":
		_, v := parts[1].ArbitraryParts()
		return g.newMaskUtility(ctx, "mask-position", arbitraryValue(v)), nil

	case l == 2 && parts[0].Ident == "size" && parts[1].Custom != "":
		_, v := parts[1].CustomParts()
		return g.newMaskUtility(ctx, "mask-size", g.cssVar(ctx, v)), nil
	case l == 2 && parts[0].Ident == "size" && parts[1].Arbitrary != "":
		_, v := parts[1].ArbitraryParts()
		return g.newMaskUtility(ctx, "mask-size", arbitraryValue(v)), nil

	case simpleMatch(parts, "no", "repeat"):
		return g.newMaskUtility(ctx, "mask-repeat", "no-repeat"), nil
	case simpleMatch(parts, "repeat", "x"):
		return g.newMaskUtility(ctx, "mask-repeat", "repeat-x"), nil
	case simpleMatch(parts, "repeat", "y"):
		return g.newMaskUtility(ctx, "mask-repeat", "repeat-y"), nil
	case simpleMatch(parts, "repeat", "round"):
		return g.newMaskUtility(ctx, "mask-repeat", "round"), nil
	case simpleMatch(parts, "repeat", "space"):
		return g.newMaskUtility(ctx, "mask-repeat", "space"), nil
	case simpleMatch(parts, "repeat"):
		return g.newMaskUtility(ctx, "mask-repeat", "repeat"), nil

	case simpleMatch(parts, "top", "left"):
		return g.newMaskUtility(ctx, "mask-position", "0 0"), nil
	case simpleMatch(parts, "top", "right"):
		return g.newMaskUtility(ctx, "mask-position", "100% 0"), nil
	case simpleMatch(parts, "bottom", "left"):
		return g.newMaskUtility(ctx, "mask-position", "0 100%"), nil
	case simpleMatch(parts, "bottom", "right"):
		return g.newMaskUtility(ctx, "mask-position", "100% 100%"), nil
	case simpleMatch(parts, "top"):
		return g.newMaskUtility(ctx, "mask-position", "top"), nil
	case simpleMatch(parts, "bottom"):
		return g.newMaskUtility(ctx, "mask-position", "bottom"), nil
	case simpleMatch(parts, "left"):
		return g.newMaskUtility(ctx, "mask-position", "0"), nil
	case simpleMatch(parts, "right"):
		return g.newMaskUtility(ctx, "mask-position", "100%"), nil
	case simpleMatch(parts, "center"):
		return g.newMaskUtility(ctx, "mask-position", "center"), nil

	case simpleMatch(parts, "add"):
		return &utilityGenerated{
			kvs: []KV{
				g.cssKV(ctx, "-webkit-mask-composite", "source-over"),
				g.cssKV(ctx, "mask-composite", "add"),
			},
		}, nil
	case simpleMatch(parts, "subtract"):
		return &utilityGenerated{
			kvs: []KV{
				g.cssKV(ctx, "-webkit-mask-composite", "source-out"),
				g.cssKV(ctx, "mask-composite", "subtract"),
			},
		}, nil
	case simpleMatch(parts, "intersect"):
		return &utilityGenerated{
			kvs: []KV{
				g.cssKV(ctx, "-webkit-mask-composite", "source-in"),
				g.cssKV(ctx, "mask-composite", "intersect"),
			},
		}, nil
	case simpleMatch(parts, "exclude"):
		return &utilityGenerated{
			kvs: []KV{
				g.cssKV(ctx, "-webkit-mask-composite", "xor"),
				g.cssKV(ctx, "mask-composite", "exclude"),
			},
		}, nil

	case simpleMatch(parts, "alpha"):
		return &utilityGenerated{
			kvs: []KV{
				g.cssKV(ctx, "-webkit-mask-source-type", "alpha"),
				g.cssKV(ctx, "mask-mode", "alpha"),
			},
		}, nil
	case simpleMatch(parts, "luminance"):
		return &utilityGenerated{
			kvs: []KV{
				g.cssKV(ctx, "-webkit-mask-source-type", "luminance"),
				g.cssKV(ctx, "mask-mode", "luminance"),
			},
		}, nil
	case simpleMatch(parts, "match"):
		return &utilityGenerated{
			kvs: []KV{
				g.cssKV(ctx, "-webkit-mask-source-type", "auto"),
				g.cssKV(ctx, "mask-mode", "match-source"),
			},
		}, nil

	case simpleMatch(parts, "auto"):
		return g.newMaskUtility(ctx, "mask-size", "auto"), nil
	case simpleMatch(parts, "cover"):
		return g.newMaskUtility(ctx, "mask-size", "cover"), nil
	case simpleMatch(parts, "contain"):
		return g.newMaskUtility(ctx, "mask-size", "contain"), nil

	case simpleMatch(parts, "none"):
		return g.newMaskUtility(ctx, "mask-image", "none"), nil

	case l == 1 && parts[0].Arbitrary != "":
		return g.maskArbitraryFromPart(ctx, parts[0]), nil

	case l == 1 && parts[0].Custom != "":
		_, v := parts[0].CustomParts()
		return g.newMaskUtility(ctx, "mask-image", g.cssVar(ctx, v)), nil
	}

	return nil, errUtilityUnknown
}

func (g *gen) utilityFlex(
	ctx context.Context, parts []*parser.NamePart, mod *parser.SlashValue,
) (*utilityGenerated, error) {
	switch {
	case len(parts) == 0:
		return newUtilityGeneratedKV("display", "flex"), nil

	case len(parts) == 1:
		p := parts[0]
		peq := func(v string) bool {
			return p.Ident == v
		}

		switch {
		case p.Int != nil:
			if mod != nil && mod.Int != nil {
				return newUtilityGeneratedKV("flex", fmt.Sprintf("calc(%s/%s * 100%%)", p.String(), mod.String())), nil
			}

			return newUtilityGeneratedKV("flex", p.String()), nil
		case peq("row"):
			return newUtilityGeneratedKV("flex-direction", "row"), nil
		case peq("col"):
			return newUtilityGeneratedKV("flex-direction", "column"), nil
		case peq("nowrap"):
			return newUtilityGeneratedKV("flex-wrap", "nowrap"), nil
		case peq("wrap"):
			return newUtilityGeneratedKV("flex-wrap", "wrap"), nil
		case peq("auto"):
			return newUtilityGeneratedKV("flex", "auto"), nil
		case peq("initial"):
			return newUtilityGeneratedKV("flex", "0 auto"), nil
		case peq("none"):
			return newUtilityGeneratedKV("flex", "none"), nil
		case p.Custom != "":
			return g.cssCustom(ctx, "flex", p), nil
		case p.Arbitrary != "":
			return g.cssArbitrary(ctx, "flex", p), nil
		}

	case len(parts) == 2:
		p1, p2 := parts[0], parts[1]
		p1p2eq := func(v1, v2 string) bool {
			return p1.Ident == v1 && p2.Ident == v2
		}

		switch {
		case p1p2eq("row", "reverse"):
			return newUtilityGeneratedKV("flex-direction", "row-reverse"), nil
		case p1p2eq("col", "reverse"):
			return newUtilityGeneratedKV("flex-direction", "column-reverse"), nil
		case p1p2eq("wrap", "reverse"):
			return newUtilityGeneratedKV("flex-wrap", "wrap-reverse"), nil
		}
	}

	return nil, errUtilityUnknown
}

func (g *gen) utilityFlexGrowShrink(ctx context.Context, v string, parts []*parser.NamePart) (*utilityGenerated, error) {
	k := "flex-" + v

	switch {
	case len(parts) == 0:
		return newUtilityGeneratedKV(k, "1"), nil
	case len(parts) == 1:
		p := parts[0]

		switch {
		case p.Int != nil:
			return newUtilityGeneratedKV(k, p.String()), nil
		case p.Custom != "":
			return g.cssCustom(ctx, k, p), nil
		case p.Arbitrary != "":
			return g.cssArbitrary(ctx, k, p), nil
		}
	default:
		return nil, fmt.Errorf("there should be one or two parts for %s utility", k)
	}

	return nil, errUtilityUnknown
}

func (g *gen) utilityColRow(
	ctx context.Context, k string, parts []*parser.NamePart, n bool,
) (*utilityGenerated, error) {
	switch len(parts) {
	case 1:
		p := parts[0]
		switch {
		case p.Ident == "auto":
			return newUtilityGeneratedKV(k, "auto"), nil
		case p.Int != nil:
			if n {
				return newUtilityGeneratedKVf(k, "calc(%s * -1)", p.String()), nil
			}

			return newUtilityGeneratedKV(k, p.String()), nil
		case p.Custom != "":
			return g.cssCustom(ctx, k, p), nil
		case p.Arbitrary != "":
			return g.cssArbitrary(ctx, k, p), nil
		}

	case 2:
		p1, p2 := parts[0], parts[1]

		switch p1.Ident {
		case "span":
			return g.utilityColRowSpan(ctx, k, p2)
		case "start", "end":
			return g.utilityColRowStartEnd(ctx, k+"-"+p1.Ident, p2, n)
		}

	default:
		return nil, fmt.Errorf("there should be one or two parts for %s utility", k)
	}

	return nil, errUtilityUnknown
}

func (g *gen) utilityColRowSpan(
	ctx context.Context, k string, p *parser.NamePart,
) (*utilityGenerated, error) {
	switch {
	case p.Int != nil:
		s := p.String()
		return newUtilityGeneratedKVf(k, "span %s / span %s", s, s), nil
	case p.Ident == "full":
		return newUtilityGeneratedKV(k, "1 / -1"), nil
	case p.Custom != "":
		_, v := p.CustomParts()
		vv := g.cssVar(ctx, v)
		return newUtilityGeneratedKVf(k, "span %s / span %s", vv, vv), nil
	case p.Arbitrary != "":
		_, v := p.ArbitraryParts()
		vv := arbitraryValue(v)
		return newUtilityGeneratedKVf(k, "span %s / span %s", vv, vv), nil
	}

	return nil, errUtilityUnknown
}

func (g *gen) utilityColRowStartEnd(
	ctx context.Context, k string, p *parser.NamePart, n bool,
) (*utilityGenerated, error) {
	switch {
	case p.Int != nil:
		if n {
			return newUtilityGeneratedKVf(k, "calc(%s * -1)", p.String()), nil
		}

		return newUtilityGeneratedKV(k, p.String()), nil
	case p.Ident == "auto":
		return newUtilityGeneratedKV(k, "auto"), nil
	case p.Custom != "":
		return g.cssCustom(ctx, k, p), nil
	case p.Arbitrary != "":
		return g.cssArbitrary(ctx, k, p), nil
	}

	return nil, errUtilityUnknown
}

func (g *gen) utilityOrder(ctx context.Context, p *parser.NamePart, n bool) (*utilityGenerated, error) {
	k := "order"

	switch {
	case p.Ident == "first":
		return newUtilityGeneratedKV(k, "-9999"), nil
	case p.Ident == "last":
		return newUtilityGeneratedKV(k, "9999"), nil
	case p.Int != nil:
		if n {
			return newUtilityGeneratedKVf(k, "calc(%s * -1)", p.String()), nil
		}

		return newUtilityGeneratedKV(k, p.String()), nil
	case p.Custom != "":
		return g.cssCustom(ctx, k, p), nil
	case p.Arbitrary != "":
		return g.cssArbitrary(ctx, k, p), nil
	}

	return nil, errUtilityUnknown
}

func (g *gen) utilityFlexBasis(ctx context.Context, parts []*parser.NamePart, mod *parser.SlashValue) (*utilityGenerated, error) {
	if len(parts) != 1 {
		return nil, fmt.Errorf("there should be only one part for flex basis-* utility")
	}

	k := "flex-basis"
	p := parts[0]

	switch {
	case p.Int != nil:
		if mod != nil && mod.Int != nil {
			return newUtilityGeneratedKV(k, fmt.Sprintf("calc(%s / %s * 100%%)", p.String(), mod.String())), nil
		}

		return g.cssCalcSpacing(ctx, k, p, false), nil
	case p.Float != nil:
		return g.cssCalcSpacing(ctx, k, p, false), nil
	case p.Ident == "auto":
		return newUtilityGeneratedKV(k, p.Ident), nil
	case p.Ident == "full":
		return newUtilityGeneratedKV(k, "100%"), nil
	case isContainerSize(p.AlphaNum) || isContainerSize(p.Ident):
		return newUtilityGeneratedKV(k, g.cssVar(ctx, fmt.Sprintf("--container-%s", p.String()))), nil
	case p.Custom != "":
		return g.cssCustom(ctx, k, p), nil
	case p.Arbitrary != "":
		return g.cssArbitrary(ctx, k, p), nil
	}

	return nil, errUtilityUnknown
}

func (g *gen) utilityGrid(ctx context.Context, parts []*parser.NamePart) (*utilityGenerated, error) {
	switch {
	case len(parts) == 0:
		return newUtilityGeneratedKV("display", "grid"), nil

	case len(parts) > 1 && parts[0].Ident == "flow":
		switch {
		case simpleMatch(parts, "flow", "row"):
			return newUtilityGeneratedKV("grid-auto-flow", "row"), nil
		case simpleMatch(parts, "flow", "col"):
			return newUtilityGeneratedKV("grid-auto-flow", "column"), nil
		case simpleMatch(parts, "flow", "dense"):
			return newUtilityGeneratedKV("grid-auto-flow", "dense"), nil
		case simpleMatch(parts, "flow", "row", "dense"):
			return newUtilityGeneratedKV("grid-auto-flow", "row dense"), nil
		case simpleMatch(parts, "flow", "col", "dense"):
			return newUtilityGeneratedKV("grid-auto-flow", "column dense"), nil
		}

	case len(parts) == 2:
		p1, p2 := parts[0], parts[1]

		switch p1.Ident {
		case "cols":
			return g.utilityGridTemplate(ctx, "grid-template-columns", p2)
		case "rows":
			return g.utilityGridTemplate(ctx, "grid-template-rows", p2)
		}
	}

	return nil, errUtilityUnknown
}

func (g *gen) utilityGridTemplate(ctx context.Context, k string, p *parser.NamePart) (*utilityGenerated, error) {
	switch {
	case p.Int != nil:
		return newUtilityGeneratedKVf(k, "repeat(%s, minmax(0, 1fr))", p.String()), nil
	case p.Ident == "none" || p.Ident == "subgrid":
		return newUtilityGeneratedKV(k, p.Ident), nil
	case p.Custom != "":
		return g.cssCustom(ctx, k, p), nil
	case p.Arbitrary != "":
		return g.cssArbitrary(ctx, k, p), nil
	}

	return nil, errUtilityUnknown
}

func (g *gen) utilityGridAuto(ctx context.Context, k string, p *parser.NamePart) (*utilityGenerated, error) {
	switch {
	case p.Ident == "auto":
		return newUtilityGeneratedKV(k, "auto"), nil
	case p.Ident == "min":
		return newUtilityGeneratedKV(k, "min-content"), nil
	case p.Ident == "max":
		return newUtilityGeneratedKV(k, "max-content"), nil
	case p.Ident == "fr":
		return newUtilityGeneratedKV(k, "minmax(0, 1fr)"), nil
	case p.Int != nil:
		return g.cssCalcSpacing(ctx, k, p, false), nil
	case p.Custom != "":
		return g.cssCustom(ctx, k, p), nil
	case p.Arbitrary != "":
		return g.cssArbitrary(ctx, k, p), nil
	}

	return nil, errUtilityUnknown
}

func (g *gen) utilityGap(ctx context.Context, k string, p *parser.NamePart) (*utilityGenerated, error) {
	switch {
	case p.Int != nil:
		return g.cssCalcSpacing(ctx, k, p, false), nil
	case p.Float != nil:
		return g.cssCalcSpacing(ctx, k, p, false), nil
	case p.Custom != "":
		return g.cssCustom(ctx, k, p), nil
	case p.Arbitrary != "":
		return g.cssArbitrary(ctx, k, p), nil
	}
	return nil, errUtilityUnknown
}

func (g *gen) utilityJustify(parts []*parser.NamePart) (*utilityGenerated, error) {
	if len(parts) > 3 {
		return nil, fmt.Errorf("it is impossible to have more than 3 parts for justify-* utility")
	}

	k := "justify-content"
	switch {
	case simpleMatch(parts, "start"):
		return newUtilityGeneratedKV(k, "flex-start"), nil
	case simpleMatch(parts, "start", "safe"):
		return newUtilityGeneratedKV(k, "safe flex-start"), nil
	case simpleMatch(parts, "end"):
		return newUtilityGeneratedKV(k, "flex-end"), nil
	case simpleMatch(parts, "end", "safe"):
		return newUtilityGeneratedKV(k, "safe flex-end"), nil
	case simpleMatch(parts, "center"):
		return newUtilityGeneratedKV(k, "center"), nil
	case simpleMatch(parts, "center", "safe"):
		return newUtilityGeneratedKV(k, "safe center"), nil
	case simpleMatch(parts, "between"):
		return newUtilityGeneratedKV(k, "space-between"), nil
	case simpleMatch(parts, "around"):
		return newUtilityGeneratedKV(k, "space-around"), nil
	case simpleMatch(parts, "evenly"):
		return newUtilityGeneratedKV(k, "space-evenly"), nil
	case simpleMatch(parts, "stretch"):
		return newUtilityGeneratedKV(k, "stretch"), nil
	case simpleMatch(parts, "baseline"):
		return newUtilityGeneratedKV(k, "baseline"), nil
	case simpleMatch(parts, "normal"):
		return newUtilityGeneratedKV(k, "normal"), nil

	case len(parts) >= 2 && parts[0].Ident == "items":
		k = "justify-items"
		switch {
		case simpleMatch(parts, "items", "start"):
			return newUtilityGeneratedKV(k, "start"), nil
		case simpleMatch(parts, "items", "end"):
			return newUtilityGeneratedKV(k, "end"), nil
		case simpleMatch(parts, "items", "end", "safe"):
			return newUtilityGeneratedKV(k, "safe end"), nil
		case simpleMatch(parts, "items", "center"):
			return newUtilityGeneratedKV(k, "center"), nil
		case simpleMatch(parts, "items", "center", "safe"):
			return newUtilityGeneratedKV(k, "safe center"), nil
		case simpleMatch(parts, "items", "stretch"):
			return newUtilityGeneratedKV(k, "stretch"), nil
		case simpleMatch(parts, "items", "normal"):
			return newUtilityGeneratedKV(k, "normal"), nil
		}

	case len(parts) >= 2 && parts[0].Ident == "self":
		k = "justify-self"
		switch {
		case simpleMatch(parts, "self", "auto"):
			return newUtilityGeneratedKV(k, "auto"), nil
		case simpleMatch(parts, "self", "start"):
			return newUtilityGeneratedKV(k, "start"), nil
		case simpleMatch(parts, "self", "end"):
			return newUtilityGeneratedKV(k, "end"), nil
		case simpleMatch(parts, "self", "end", "safe"):
			return newUtilityGeneratedKV(k, "safe end"), nil
		case simpleMatch(parts, "self", "center"):
			return newUtilityGeneratedKV(k, "center"), nil
		case simpleMatch(parts, "self", "center", "safe"):
			return newUtilityGeneratedKV(k, "safe center"), nil
		case simpleMatch(parts, "self", "stretch"):
			return newUtilityGeneratedKV(k, "stretch"), nil
		}

	}

	return nil, errUtilityUnknown
}

func (g *gen) utilityTable(_ context.Context, parts []*parser.NamePart) (*utilityGenerated, error) {
	switch {
	case len(parts) == 0:
		return newUtilityGeneratedKV("display", "table"), nil
	case simpleMatch(parts, "auto"):
		return newUtilityGeneratedKV("table-layout", "auto"), nil
	case simpleMatch(parts, "fixed"):
		return newUtilityGeneratedKV("table-layout", "fixed"), nil
	case simpleMatch(parts, "caption"):
		return newUtilityGeneratedKV("display", "table-caption"), nil
	case simpleMatch(parts, "cell"):
		return newUtilityGeneratedKV("display", "table-cell"), nil
	case simpleMatch(parts, "column"):
		return newUtilityGeneratedKV("display", "table-column"), nil
	case simpleMatch(parts, "column", "group"):
		return newUtilityGeneratedKV("display", "table-column-group"), nil
	case simpleMatch(parts, "footer", "group"):
		return newUtilityGeneratedKV("display", "table-footer-group"), nil
	case simpleMatch(parts, "header", "group"):
		return newUtilityGeneratedKV("display", "table-header-group"), nil
	case simpleMatch(parts, "row"):
		return newUtilityGeneratedKV("display", "table-row"), nil
	case simpleMatch(parts, "row", "group"):
		return newUtilityGeneratedKV("display", "table-row-group"), nil
	}

	return nil, errUtilityUnknown
}

func (g *gen) utilityCaption(parts []*parser.NamePart) (*utilityGenerated, error) {
	switch {
	case simpleMatch(parts, "top"):
		return newUtilityGeneratedKV("caption-side", "top"), nil
	case simpleMatch(parts, "bottom"):
		return newUtilityGeneratedKV("caption-side", "bottom"), nil
	}

	return nil, errUtilityUnknown
}

func (g *gen) utilityContent(ctx context.Context, parts []*parser.NamePart) (*utilityGenerated, error) {
	k := "align-content"

	switch {
	case simpleMatch(parts, "normal"):
		return newUtilityGeneratedKV(k, "normal"), nil
	case simpleMatch(parts, "start"):
		return newUtilityGeneratedKV(k, "flex-start"), nil
	case simpleMatch(parts, "end"):
		return newUtilityGeneratedKV(k, "flex-end"), nil
	case simpleMatch(parts, "center"):
		return newUtilityGeneratedKV(k, "center"), nil
	case simpleMatch(parts, "between"):
		return newUtilityGeneratedKV(k, "space-between"), nil
	case simpleMatch(parts, "around"):
		return newUtilityGeneratedKV(k, "space-around"), nil
	case simpleMatch(parts, "evenly"):
		return newUtilityGeneratedKV(k, "space-evenly"), nil
	case simpleMatch(parts, "baseline"):
		return newUtilityGeneratedKV(k, "baseline"), nil
	case simpleMatch(parts, "stretch"):
		return newUtilityGeneratedKV(k, "stretch"), nil
	case simpleMatch(parts, "none"):
		return &utilityGenerated{kvs: []KV{g.cssDefine(ctx, twContentVar, "none")}}, nil
	case len(parts) == 1 && parts[0].Custom != "":
		p := parts[0]
		return &utilityGenerated{kvs: []KV{g.cssCustomDefine(ctx, twContentVar, p)}}, nil
	case len(parts) == 1 && parts[0].Arbitrary != "":
		p := parts[0]
		return &utilityGenerated{kvs: []KV{g.cssArbitraryDefine(ctx, twContentVar, p)}}, nil
	}

	return nil, errUtilityUnknown
}

func (g *gen) utilityItems(parts []*parser.NamePart) (*utilityGenerated, error) {
	if len(parts) > 2 {
		return nil, fmt.Errorf("it is impossible to have more than 2 parts for items-* utility")
	}

	k := "align-items"

	switch {
	case simpleMatch(parts, "stretch"):
		return newUtilityGeneratedKV(k, "stretch"), nil
	case simpleMatch(parts, "start"):
		return newUtilityGeneratedKV(k, "flex-start"), nil
	case simpleMatch(parts, "end"):
		return newUtilityGeneratedKV(k, "flex-end"), nil
	case simpleMatch(parts, "end", "safe"):
		return newUtilityGeneratedKV(k, "safe flex-end"), nil
	case simpleMatch(parts, "center"):
		return newUtilityGeneratedKV(k, "center"), nil
	case simpleMatch(parts, "center", "safe"):
		return newUtilityGeneratedKV(k, "safe center"), nil
	case simpleMatch(parts, "baseline"):
		return newUtilityGeneratedKV(k, "baseline"), nil
	case simpleMatch(parts, "baseline", "last"):
		return newUtilityGeneratedKV(k, "last baseline"), nil
	}

	return nil, errUtilityUnknown
}

func (g *gen) utilitySelf(parts []*parser.NamePart) (*utilityGenerated, error) {
	if len(parts) > 2 {
		return nil, fmt.Errorf("it is impossible to have more than 2 parts for self-* utility")
	}

	k := "align-self"

	switch {
	case simpleMatch(parts, "auto"):
		return newUtilityGeneratedKV(k, "auto"), nil
	case simpleMatch(parts, "start"):
		return newUtilityGeneratedKV(k, "flex-start"), nil
	case simpleMatch(parts, "end"):
		return newUtilityGeneratedKV(k, "flex-end"), nil
	case simpleMatch(parts, "end", "safe"):
		return newUtilityGeneratedKV(k, "safe flex-end"), nil
	case simpleMatch(parts, "center"):
		return newUtilityGeneratedKV(k, "center"), nil
	case simpleMatch(parts, "center", "safe"):
		return newUtilityGeneratedKV(k, "safe center"), nil
	case simpleMatch(parts, "stretch"):
		return newUtilityGeneratedKV(k, "stretch"), nil
	case simpleMatch(parts, "baseline"):
		return newUtilityGeneratedKV(k, "baseline"), nil
	case simpleMatch(parts, "baseline", "last"):
		return newUtilityGeneratedKV(k, "last baseline"), nil
	}

	return nil, errUtilityUnknown
}

func (g *gen) utilityPlace(parts []*parser.NamePart) (*utilityGenerated, error) {
	if len(parts) > 3 {
		return nil, fmt.Errorf("it is impossible to have more than 3 parts for place-* utility")
	}

	k := "place-content"

	switch {
	case simpleMatch(parts, "content", "center"):
		return newUtilityGeneratedKV(k, "center"), nil
	case simpleMatch(parts, "content", "center", "safe"):
		return newUtilityGeneratedKV(k, "safe center"), nil
	case simpleMatch(parts, "content", "start"):
		return newUtilityGeneratedKV(k, "start"), nil
	case simpleMatch(parts, "content", "end"):
		return newUtilityGeneratedKV(k, "end"), nil
	case simpleMatch(parts, "content", "end", "safe"):
		return newUtilityGeneratedKV(k, "safe end"), nil
	case simpleMatch(parts, "content", "between"):
		return newUtilityGeneratedKV(k, "space-between"), nil
	case simpleMatch(parts, "content", "around"):
		return newUtilityGeneratedKV(k, "space-around"), nil
	case simpleMatch(parts, "content", "evenly"):
		return newUtilityGeneratedKV(k, "space-evenly"), nil
	case simpleMatch(parts, "content", "baseline"):
		return newUtilityGeneratedKV(k, "baseline"), nil
	case simpleMatch(parts, "content", "stretch"):
		return newUtilityGeneratedKV(k, "stretch"), nil

	case len(parts) >= 2 && parts[0].Ident == "items":
		k = "place-items"

		switch {
		case simpleMatch(parts, "items", "start"):
			return newUtilityGeneratedKV(k, "start"), nil
		case simpleMatch(parts, "items", "end"):
			return newUtilityGeneratedKV(k, "end"), nil
		case simpleMatch(parts, "items", "end", "safe"):
			return newUtilityGeneratedKV(k, "safe end"), nil
		case simpleMatch(parts, "items", "center"):
			return newUtilityGeneratedKV(k, "center"), nil
		case simpleMatch(parts, "items", "center", "safe"):
			return newUtilityGeneratedKV(k, "safe center"), nil
		case simpleMatch(parts, "items", "baseline"):
			return newUtilityGeneratedKV(k, "baseline"), nil
		case simpleMatch(parts, "items", "stretch"):
			return newUtilityGeneratedKV(k, "stretch"), nil
		}

	case len(parts) >= 2 && parts[0].Ident == "self":
		k = "place-self"

		switch {
		case simpleMatch(parts, "self", "auto"):
			return newUtilityGeneratedKV(k, "auto"), nil
		case simpleMatch(parts, "self", "start"):
			return newUtilityGeneratedKV(k, "start"), nil
		case simpleMatch(parts, "self", "end"):
			return newUtilityGeneratedKV(k, "end"), nil
		case simpleMatch(parts, "self", "end", "safe"):
			return newUtilityGeneratedKV(k, "safe end"), nil
		case simpleMatch(parts, "self", "center"):
			return newUtilityGeneratedKV(k, "center"), nil
		case simpleMatch(parts, "self", "center", "safe"):
			return newUtilityGeneratedKV(k, "safe center"), nil
		case simpleMatch(parts, "self", "stretch"):
			return newUtilityGeneratedKV(k, "stretch"), nil
		}
	}

	return nil, errUtilityUnknown
}

func (g *gen) utilityPadding(ctx context.Context, k string, p *parser.NamePart) (*utilityGenerated, error) {
	switch {
	case p.Ident == "px":
		return newUtilityGeneratedKV(k, "1px"), nil
	case p.Int != nil || p.Float != nil:
		return g.cssCalcSpacing(ctx, k, p, false), nil
	case p.Custom != "":
		return g.cssCustom(ctx, k, p), nil
	case p.Arbitrary != "":
		return g.cssArbitrary(ctx, k, p), nil
	}

	return nil, errUtilityUnknown
}

func (g *gen) utilityMargin(ctx context.Context, k string, p *parser.NamePart, negative bool) (*utilityGenerated, error) {
	switch {
	case p.Ident == "auto":
		return newUtilityGeneratedKV(k, "auto"), nil
	case p.Ident == "px" && negative:
		return newUtilityGeneratedKV(k, "-1px"), nil
	case p.Ident == "px" && !negative:
		return newUtilityGeneratedKV(k, "1px"), nil
	case p.Int != nil || p.Float != nil:
		return g.cssCalcSpacing(ctx, k, p, negative), nil
	case p.Custom != "":
		return g.cssCustom(ctx, k, p), nil
	case p.Arbitrary != "":
		return g.cssArbitrary(ctx, k, p), nil
	}

	return nil, errUtilityUnknown
}

func (g *gen) utilitySpace(
	ctx context.Context, parts []*parser.NamePart, negative bool,
) (*utilityGenerated, error) {
	if len(parts) != 2 {
		return nil, fmt.Errorf("you are supposed to have two parts for space-* utilities")
	}

	n := ""
	if negative {
		n = "-"
	}

	header := "& > :not(:last-child)"
	spaceXReverse := "--tw-space-x-reverse"
	spacing := g.cssVar(ctx, "--spacing")
	spaceYReverse := "--tw-space-y-reverse"
	p1, p2 := parts[0], parts[1]

	switch p1.Ident {
	case "x":
		switch {
		case p2.Int != nil || p2.Float != nil:
			return &utilityGenerated{
				others: []Query{
					{
						Name: header,
						KVs: []KV{
							g.cssDefine(ctx, spaceXReverse, "0"),
							g.cssKVf(ctx,
								"margin-inline-start", "calc(calc(%s * %s%s) * %s)",
								spacing, n, p2.String(),
								g.cssRef(ctx, spaceXReverse),
							),
							g.cssKVf(ctx,
								"margin-inline-end", "calc(calc(%s * %s%s) * calc(1 - %s))",
								spacing, n, p2.String(),
								g.cssRef(ctx, spaceXReverse),
							),
						},
					},
				},
			}, nil
		case p2.Ident == "px":
			return &utilityGenerated{
				others: []Query{
					{
						Name: header,
						KVs: []KV{
							g.cssDefine(ctx, spaceXReverse, "0"),
							g.cssKVf(ctx,
								"margin-inline-start", "calc(%s1px * %s)",
								n, g.cssRef(ctx, spaceXReverse),
							),
							g.cssKVf(ctx,
								"margin-inline-end", "calc(%s1px * calc(1 - %s))",
								n, g.cssRef(ctx, spaceXReverse),
							),
						},
					},
				},
			}, nil
		case p2.Custom != "" || p2.Arbitrary != "":
			var v string
			switch {
			case p2.Custom != "":
				_, vv := p2.CustomParts()
				v = g.cssVar(ctx, vv)
			case p2.Arbitrary != "":
				_, vv := p2.ArbitraryParts()
				v = arbitraryValue(vv)
			}

			return &utilityGenerated{
				others: []Query{
					{
						Name: header,
						KVs: []KV{
							g.cssDefine(ctx, spaceXReverse, "0"),
							g.cssKVf(ctx,
								"margin-inline-start", "calc(%s * %s)",
								v, g.cssRef(ctx, spaceXReverse),
							),
							g.cssKVf(ctx,
								"margin-inline-end", "calc(%s * calc(1 - %s))",
								v, g.cssRef(ctx, spaceXReverse),
							),
						},
					},
				},
			}, nil
		case p2.Ident == "reverse":
			return &utilityGenerated{
				others: []Query{{Name: header, KVs: []KV{g.cssDefine(ctx, spaceXReverse, "1")}}},
			}, nil
		}

	case "y":
		switch {
		case p2.Int != nil || p2.Float != nil:
			return &utilityGenerated{
				others: []Query{
					{
						Name: header,
						KVs: []KV{
							g.cssDefine(ctx, spaceYReverse, "0"),
							g.cssKVf(ctx,
								"margin-block-start", "calc(calc(%s * %s%s) * %s)",
								spacing, n, p2.String(),
								g.cssRef(ctx, spaceYReverse),
							),
							g.cssKVf(ctx,
								"margin-block-end", "calc(calc(%s * %s%s) * calc(1 - %s))",
								spacing, n, p2.String(),
								g.cssRef(ctx, spaceYReverse),
							),
						},
					},
				},
			}, nil
		case p2.Ident == "px":
			return &utilityGenerated{
				others: []Query{
					{
						Name: header,
						KVs: []KV{
							g.cssDefine(ctx, spaceYReverse, "0"),
							g.cssKVf(ctx,
								"margin-block-start", "calc(%s1px * %s)",
								n, g.cssRef(ctx, spaceYReverse),
							),
							g.cssKVf(ctx,
								"margin-block-end", "calc(%s1px * calc(1 - %s))",
								n, g.cssRef(ctx, spaceYReverse),
							),
						},
					},
				},
			}, nil
		case p2.Custom != "" || p2.Arbitrary != "":
			var v string
			switch {
			case p2.Custom != "":
				_, vv := p2.CustomParts()
				v = g.cssVar(ctx, vv)
			case p2.Arbitrary != "":
				_, vv := p2.ArbitraryParts()
				v = arbitraryValue(vv)
			}

			return &utilityGenerated{
				others: []Query{
					{
						Name: header,
						KVs: []KV{
							g.cssDefine(ctx, spaceYReverse, "0"),
							g.cssKVf(ctx,
								"margin-block-start", "calc(%s * %s)",
								v, g.cssRef(ctx, spaceYReverse),
							),
							g.cssKVf(ctx,
								"margin-block-end", "calc(%s * calc(1 - %s))",
								v, g.cssRef(ctx, spaceYReverse),
							),
						},
					},
				},
			}, nil
		case p2.Ident == "reverse":
			return &utilityGenerated{
				others: []Query{{Name: header, KVs: []KV{g.cssDefine(ctx, spaceYReverse, "1")}}},
			}, nil
		}
	}

	return nil, errUtilityUnknown
}

func (g *gen) utilitySizing(
	ctx context.Context, k string, p *parser.NamePart, m *parser.SlashValue,
	verify func(string) bool, key func(string) string,
) (*utilityGenerated, error) {
	switch {
	case p.Int != nil:
		if m != nil && m.Int != nil {
			return newUtilityGeneratedKV(k, fmt.Sprintf("calc(%s/%s * 100%%)", p.String(), m.String())), nil
		}

		return g.cssCalcSpacing(ctx, k, p, false), nil
	case p.Float != nil:
		return g.cssCalcSpacing(ctx, k, p, false), nil
	case isContainerSizePart(p):
		return g.cssCustomContainerSize(ctx, k, p.String()), nil
	case verify(p.Ident):
		return newUtilityGeneratedKV(k, key(p.Ident)), nil
	case p.Custom != "":
		return g.cssCustom(ctx, k, p), nil
	case p.Arbitrary != "":
		return g.cssArbitrary(ctx, k, p), nil
	}

	return nil, errUtilityUnknown
}

func (g *gen) utilityMin(
	ctx context.Context, parts []*parser.NamePart, m *parser.SlashValue,
) (*utilityGenerated, error) {
	p1, p2 := parts[0], parts[1]

	switch p1.Ident {
	case "w":
		return g.utilityMinMaxWidth(ctx, "min-width", p2, m, false)
	case "h":
		return g.utilityMinMaxHeight(ctx, "min-height", p2, m, false)
	case "inline":
		return g.utilityMinMaxWidth(ctx, "min-inline-size", p2, m, false)
	case "block":
		return g.utilityMinMaxHeight(ctx, "min-block-size", p2, m, false)
	}

	return nil, errUtilityUnknown
}

func (g *gen) utilityMax(
	ctx context.Context, parts []*parser.NamePart, m *parser.SlashValue,
) (*utilityGenerated, error) {
	p1, p2 := parts[0], parts[1]

	switch p1.Ident {
	case "w":
		return g.utilityMinMaxWidth(ctx, "max-width", p2, m, true)
	case "h":
		return g.utilityMinMaxHeight(ctx, "max-height", p2, m, true)
	case "inline":
		return g.utilityMinMaxWidth(ctx, "max-inline-size", p2, m, true)
	case "block":
		return g.utilityMinMaxHeight(ctx, "max-block-size", p2, m, true)
	}

	return nil, errUtilityUnknown
}

func (g *gen) utilityMinMaxWidth(
	ctx context.Context, k string, p *parser.NamePart, m *parser.SlashValue, none bool,
) (*utilityGenerated, error) {
	switch {
	case none && p.Ident == "none":
		return newUtilityGeneratedKV(k, "none"), nil
	case p.Int != nil:
		if m != nil && m.Int != nil {
			return newUtilityGeneratedKV(k, fmt.Sprintf("calc(%s/%s * 100%%)", p.String(), m.String())), nil
		}

		return g.cssCalcSpacing(ctx, k, p, false), nil
	case p.Float != nil:
		return g.cssCalcSpacing(ctx, k, p, false), nil
	case isContainerSizePart(p):
		return g.cssCustomContainerSize(ctx, k, p.String()), nil
	case isSimpleSize(p.Ident):
		return newUtilityGeneratedKV(k, simpleSize(p.Ident)), nil
	case p.Custom != "":
		return g.cssCustom(ctx, k, p), nil
	case p.Arbitrary != "":
		return g.cssArbitrary(ctx, k, p), nil
	}

	return nil, errUtilityUnknown
}

func (g *gen) utilityMinMaxHeight(
	ctx context.Context, k string, p *parser.NamePart, m *parser.SlashValue, none bool,
) (*utilityGenerated, error) {
	switch {
	case none && p.Ident == "none":
		return newUtilityGeneratedKV(k, "none"), nil
	case p.Int != nil:
		if m != nil && m.Int != nil {
			return newUtilityGeneratedKV(k, fmt.Sprintf("calc(%s/%s * 100%%)", p.String(), m.String())), nil
		}

		return g.cssCalcSpacing(ctx, k, p, false), nil
	case p.Float != nil:
		return g.cssCalcSpacing(ctx, k, p, false), nil
	case isSimpleSizeHeight(p.Ident):
		return newUtilityGeneratedKV(k, simpleSizeHeight(p.Ident)), nil
	case p.Custom != "":
		return g.cssCustom(ctx, k, p), nil
	case p.Arbitrary != "":
		return g.cssArbitrary(ctx, k, p), nil
	}

	return nil, errUtilityUnknown
}

func (g *gen) utilitySize(
	ctx context.Context, p *parser.NamePart, m *parser.SlashValue,
) (*utilityGenerated, error) {
	w, h := "width", "height"

	switch {
	case p.Int != nil:
		if m != nil && m.Int != nil {
			return &utilityGenerated{
				kvs: []KV{
					g.cssKV(ctx, w, fmt.Sprintf("calc(%s/%s * 100%%)", p.String(), m.String())),
					g.cssKV(ctx, h, fmt.Sprintf("calc(%s/%s * 100%%)", p.String(), m.String())),
				},
			}, nil
		}

		return &utilityGenerated{
			kvs: []KV{
				g.cssKVf(ctx, w, "calc(%s * %s)", g.cssVar(ctx, "--spacing"), p.String()),
				g.cssKVf(ctx, h, "calc(%s * %s)", g.cssVar(ctx, "--spacing"), p.String()),
			},
		}, nil
	case p.Float != nil:
		return &utilityGenerated{
			kvs: []KV{
				g.cssKVf(ctx, w, "calc(%s * %s)", g.cssVar(ctx, "--spacing"), p.String()),
				g.cssKVf(ctx, h, "calc(%s * %s)", g.cssVar(ctx, "--spacing"), p.String()),
			},
		}, nil
	case isContainerSizePart(p):
		return &utilityGenerated{
			kvs: []KV{
				g.cssKV(ctx, w, g.cssVar(ctx, fmt.Sprintf("--container-%s", p.String()))),
				g.cssKV(ctx, h, g.cssVar(ctx, fmt.Sprintf("--container-%s", p.String()))),
			},
		}, nil
	case isSimpleSize(p.Ident):
		return &utilityGenerated{
			kvs: []KV{
				g.cssKV(ctx, w, simpleSize(p.Ident)),
				g.cssKV(ctx, h, simpleSize(p.Ident)),
			},
		}, nil
	case p.Custom != "":
		_, v := p.CustomParts()
		return &utilityGenerated{
			kvs: []KV{
				g.cssKV(ctx, w, g.cssVar(ctx, v)),
				g.cssKV(ctx, h, g.cssVar(ctx, v)),
			},
		}, nil
	case p.Arbitrary != "":
		_, v := p.ArbitraryParts()
		return &utilityGenerated{
			kvs: []KV{
				g.cssKV(ctx, w, g.cssArbitraryValue(ctx, arbitraryValue(v))),
				g.cssKV(ctx, h, g.cssArbitraryValue(ctx, arbitraryValue(v))),
			},
		}, nil
	}

	return nil, errUtilityUnknown
}

func (g *gen) utilityFont(ctx context.Context, parts []*parser.NamePart) (*utilityGenerated, error) {
	switch {
	case len(parts) == 1:
		p := parts[0]
		switch {
		case slices.Contains([]string{"sans", "serif", "mono"}, p.Ident):
			v := fmt.Sprintf("--font-%s", p.Ident)
			return newUtilityGeneratedKV("font-family", g.cssVar(ctx, v)), nil
		case slices.Contains([]string{
			"thin", "extralight", "light", "normal", "medium", "semibold", "bold", "extrabold", "black",
		}, p.Ident):
			return newUtilityGeneratedKV("font-weight", g.cssVar(ctx, "--font-weight-"+p.Ident)), nil
		case p.Custom != "":
			k, _ := p.CustomParts()
			switch k {
			case "family-name":
				return g.cssCustom(ctx, "font-family", p), nil
			default:
				return g.cssCustom(ctx, "font-weight", p), nil
			}
		case p.Arbitrary != "":
			k, v := p.ArbitraryParts()
			switch {
			case k == "family-name":
				return g.cssArbitrary(ctx, "font-family", p), nil
			case k == "length" || k == "size":
				return g.cssArbitrary(ctx, "font-weight", p), nil
			case cssIsLength(v):
				return g.cssArbitrary(ctx, "font-weight", p), nil
			default:
				n, ok := parseLeadingInt(v)
				if ok {
					return newUtilityGeneratedKV("font-weight", fmt.Sprintf("%d", n)), nil
				} else {
					return g.cssArbitrary(ctx, "font-family", p), nil
				}
			}
		}
	case len(parts) >= 2:
		p1, p2 := parts[0], parts[1]
		switch p1.Ident {
		case "stretch":
			switch {
			case simpleMatch(parts, "stretch", "ultra", "condensed"):
				fallthrough
			case simpleMatch(parts, "stretch", "extra", "condensed"):
				fallthrough
			case simpleMatch(parts, "stretch", "condensed"):
				fallthrough
			case simpleMatch(parts, "stretch", "semi", "condensed"):
				fallthrough
			case simpleMatch(parts, "stretch", "normal"):
				fallthrough
			case simpleMatch(parts, "stretch", "semi", "expanded"):
				fallthrough
			case simpleMatch(parts, "stretch", "expanded"):
				fallthrough
			case simpleMatch(parts, "stretch", "extra", "expanded"):
				fallthrough
			case simpleMatch(parts, "stretch", "ultra", "expanded"):
				s := []string{}
				for _, p := range parts[1:] {
					s = append(s, p.String())
				}

				return newUtilityGeneratedKV("font-stretch", strings.Join(s, "-")), nil
			case p2.Percentage != "":
				n, _ := parseLeadingInt(p2.Percentage)
				if n < 50 || n > 200 {
					return nil, errUtilityUnknown
				}

				return newUtilityGeneratedKV("font-stretch", p2.Percentage), nil
			case p2.Custom != "":
				return g.cssCustom(ctx, "font-stretch", p2), nil
			case p2.Arbitrary != "":
				return g.cssArbitrary(ctx, "font-stretch", p2), nil
			}
		case "features":
			switch {
			case p2.Custom != "":
				return g.cssCustom(ctx, "font-feature-settings", p2), nil
			case p2.Arbitrary != "":
				return g.cssArbitrary(ctx, "font-feature-settings", p2), nil
			}
		}
	}

	return nil, errUtilityUnknown
}

func (g *gen) utilityTracking(ctx context.Context, p *parser.NamePart) (*utilityGenerated, error) {
	k := "letter-spacing"

	switch {
	case slices.Contains([]string{
		"tighter", "tight", "normal", "wide", "wider", "widest",
	}, p.Ident):
		return newUtilityGeneratedKV(
			k, g.cssVar(ctx, fmt.Sprintf("--tracking-%s", p.Ident)),
		), nil

	case p.Custom != "":
		return g.cssCustom(ctx, k, p), nil
	case p.Arbitrary != "":
		return g.cssArbitrary(ctx, k, p), nil
	}

	return nil, errUtilityUnknown
}

func (g *gen) utilityLine(ctx context.Context, parts []*parser.NamePart) (*utilityGenerated, error) {
	switch {
	case len(parts) == 1 && parts[0].Ident == "through":
		return newUtilityGeneratedKV("text-decoration-line", "line-through"), nil
	case len(parts) == 2 && parts[0].Ident == "clamp":
		p := parts[1]
		switch {
		case p.Ident == "none":
			return &utilityGenerated{
				kvs: []KV{
					g.cssKV(ctx, "overflow", "visible"),
					g.cssKV(ctx, "display", "block"),
					g.cssKV(ctx, "-webkit-box-orient", "horizontal"),
					g.cssKV(ctx, "-webkit-line-clamp", "unset"),
				},
			}, nil
		case p.Int != nil:
			return &utilityGenerated{
				kvs: []KV{
					g.cssKV(ctx, "overflow", "hidden"),
					g.cssKV(ctx, "display", "-webkit-box"),
					g.cssKV(ctx, "-webkit-box-orient", "vertical"),
					g.cssKV(ctx, "-webkit-line-clamp", p.String()),
				},
			}, nil
		case p.Custom != "":
			_, v := p.CustomParts()

			return &utilityGenerated{
				kvs: []KV{
					g.cssKV(ctx, "overflow", "hidden"),
					g.cssKV(ctx, "display", "-webkit-box"),
					g.cssKV(ctx, "-webkit-box-orient", "vertical"),
					g.cssKV(ctx, "-webkit-line-clamp", g.cssVar(ctx, v)),
				},
			}, nil
		case p.Arbitrary != "":
			_, v := p.ArbitraryParts()

			return &utilityGenerated{
				kvs: []KV{
					g.cssKV(ctx, "overflow", "hidden"),
					g.cssKV(ctx, "display", "-webkit-box"),
					g.cssKV(ctx, "-webkit-box-orient", "vertical"),
					g.cssKV(ctx, "-webkit-line-clamp", g.cssArbitraryValue(ctx, arbitraryValue(v))),
				},
			}, nil
		}
	}

	return nil, errUtilityUnknown
}

func (g *gen) utilityLeading(ctx context.Context, p *parser.NamePart) (*utilityGenerated, error) {
	k := "line-height"

	switch {
	case p.Ident == "none":
		v := "1"
		return &utilityGenerated{
			kvs: []KV{g.cssDefine(ctx, "--tw-leading", v), g.cssKV(ctx, k, v)},
		}, nil
	case p.Int != nil:
		kv := g.cssCalcSpacingKV(ctx, k, p, false)
		return &utilityGenerated{
			kvs: []KV{g.cssDefine(ctx, "--tw-leading", kv.Value), kv},
		}, nil
	case p.Custom != "":
		kv := g.cssCustomKV(ctx, k, p)
		return &utilityGenerated{
			kvs: []KV{g.cssDefine(ctx, "--tw-leading", kv.Value), kv},
		}, nil
	case p.Arbitrary != "":
		kv := g.cssArbitraryKV(ctx, k, p)
		return &utilityGenerated{
			kvs: []KV{g.cssDefine(ctx, "--tw-leading", kv.Value), kv},
		}, nil
	}

	return nil, errUtilityUnknown
}

func (g *gen) utilityList(ctx context.Context, parts []*parser.NamePart) (*utilityGenerated, error) {
	switch {
	case len(parts) == 1:
		p := parts[0]
		switch {
		case p.Ident == "item":
			return newUtilityGeneratedKV("display", "list-item"), nil
		case p.Ident == "inside" || p.Ident == "outside":
			return newUtilityGeneratedKV("list-style-position", p.Ident), nil
		case p.Ident == "disc" || p.Ident == "decimal" || p.Ident == "none":
			return newUtilityGeneratedKV("list-style-type", p.Ident), nil
		case p.Custom != "":
			return g.cssCustom(ctx, "list-style-type", p), nil
		case p.Arbitrary != "":
			return g.cssArbitrary(ctx, "list-style-type", p), nil
		}
	case len(parts) == 2 && parts[0].Ident == "image":
		p := parts[1]
		k := "list-style-image"
		switch {
		case p.Ident == "none":
			return newUtilityGeneratedKV(k, "none"), nil
		case p.Custom != "":
			return g.cssCustom(ctx, k, p), nil
		case p.Arbitrary != "":
			return g.cssArbitrary(ctx, k, p), nil
		}
	}

	return nil, errUtilityUnknown
}

func (g *gen) utilityText(
	ctx context.Context, parts []*parser.NamePart, mod *parser.SlashValue,
) (*utilityGenerated, error) {
	first := func() *parser.NamePart {
		return parts[0]
	}
	onlyOne := len(parts) == 1

	if len(parts) >= 1 && parts[0].Ident == "shadow" {
		sp := parts[1:]
		const (
			alphaVar = "--tw-text-shadow-alpha"
			colorVar = "--tw-text-shadow-color"
			prop     = "text-shadow"
		)

		switch len(sp) {
		case 1:
			p := sp[0]

			switch {
			case p.Ident == "none":
				if mod != nil {
					return nil, errUtilityUnknown
				}
				return newUtilityGeneratedKV(prop, "none"), nil

			case p.Ident == "inherit":
				if mod != nil {
					return nil, errUtilityUnknown
				}
				return newUtilityGeneratedKV(colorVar, "inherit"), nil

			case p.Custom != "":
				cuk, cv := p.CustomParts()
				switch cuk {
				case "color":
					return &utilityGenerated{kvs: []KV{g.cssDefine(ctx, colorVar, g.shadowColorMix(ctx, g.cssVar(ctx, cv), alphaVar))}}, nil
				default:
					kvs := []KV{g.cssCustomKV(ctx, prop, p)}
					if mod != nil {
						alpha, err := shadowModifierAlpha(mod)
						if err != nil {
							return nil, err
						}
						kvs = append([]KV{g.cssDefine(ctx, alphaVar, alpha)}, kvs...)
					}
					return &utilityGenerated{kvs: kvs}, nil
				}

			case p.Arbitrary != "":
				ak, av := p.ArbitraryParts()
				switch {
				case ak == "color":
					color := arbitraryValue(av)
					if mod != nil && mod.Int != nil {
						color = fmt.Sprintf("color-mix(in oklab, %s %s%%, transparent)", color, mod.String())
					}
					return &utilityGenerated{kvs: []KV{g.cssDefine(ctx, colorVar, g.shadowColorMix(ctx, color, alphaVar))}}, nil
				case cssIsColor(av):
					return g.cssShadowColor(ctx, colorVar, alphaVar, sp, mod)
				default:
					kvs := []KV{g.cssArbitraryKV(ctx, prop, p)}
					if mod != nil {
						alpha, err := shadowModifierAlpha(mod)
						if err != nil {
							return nil, err
						}
						kvs = append([]KV{g.cssDefine(ctx, alphaVar, alpha)}, kvs...)
					}
					return &utilityGenerated{kvs: kvs}, nil
				}

			default:
				if theme, ok := themeTextShadows[p.String()]; ok {
					alpha, err := shadowModifierAlpha(mod)
					if err != nil {
						return nil, err
					}

					kvs := []KV{g.cssKV(ctx, prop, g.buildThemedShadowValue(ctx, theme, colorVar, alpha))}
					if mod != nil {
						kvs = append([]KV{g.cssDefine(ctx, alphaVar, alpha)}, kvs...)
					}

					return &utilityGenerated{kvs: kvs}, nil
				}
			}
		}

		return g.cssShadowColor(ctx, colorVar, alphaVar, sp, mod)
	}

	switch {
	case onlyOne && slices.Contains([]string{
		"left", "center", "right", "justify", "start", "end",
	}, first().Ident):
		return newUtilityGeneratedKV("text-align", first().Ident), nil
	case onlyOne && slices.Contains([]string{
		"ellipsis", "clip",
	}, first().Ident):
		return newUtilityGeneratedKV("text-overflow", first().Ident), nil
	case onlyOne && slices.Contains([]string{
		"wrap", "nowrap", "balance", "pretty",
	}, first().Ident):
		return newUtilityGeneratedKV("text-wrap", first().Ident), nil
	case onlyOne && isFontSize(first().String()):
		p := first()
		fontSize := g.cssKV(ctx, "font-size", g.cssVar(ctx, fmt.Sprintf("--text-%s", first().String())))

		var lineHeight KV
		switch {
		case mod != nil && mod.Int != nil:
			lineHeight = g.cssCalcSpacingKV(ctx, "line-height", mod, false)
		case mod != nil && mod.Custom != "":
			_, v := mod.CustomParts()
			lineHeight = g.cssKV(ctx, "line-height", g.cssVar(ctx, v))
		case mod != nil && mod.Arbitrary != "":
			_, v := mod.ArbitraryParts()
			lineHeight = g.cssKV(ctx, "line-height", g.cssArbitraryValue(ctx, arbitraryValue(v)))
		default:
			fallback := g.cssVar(ctx, fmt.Sprintf("--text-%s--line-height", p.String()))
			lineHeight = g.cssKV(ctx, "line-height", g.cssVarFallback(ctx, "--tw-leading", fallback))
		}

		return &utilityGenerated{kvs: []KV{fontSize, lineHeight}}, nil
	case len(parts) == 1 && first().Custom != "":
		p := first()
		k, _ := p.CustomParts()
		switch k {
		case "length":
			return g.cssCustom(ctx, "font-size", p), nil
		default:
			return g.cssCustom(ctx, "color", p), nil
		}
	case len(parts) == 1 && first().Arbitrary != "":
		p := first()
		k, v := p.ArbitraryParts()
		switch {
		case k == "length":
			return g.cssArbitrary(ctx, "font-size", p), nil
		case k == "color":
			return g.cssArbitrary(ctx, "color", p), nil
		case cssIsLength(v):
			return g.cssArbitrary(ctx, "font-size", p), nil
		case cssIsColor(v):
			return g.cssArbitrary(ctx, "color", p), nil
		default:
			return g.cssArbitrary(ctx, "color", p), nil
		}
	default:
		return g.cssColor(ctx, "color", parts, mod)
	}
}

func (g *gen) utilityDecoration(ctx context.Context, parts []*parser.NamePart, mod *parser.SlashValue) (*utilityGenerated, error) {
	first := func() *parser.NamePart { return parts[0] }

	switch {
	case len(parts) == 1 && slices.Contains([]string{
		"solid", "double", "dotted", "dashed", "wavy",
	}, first().Ident):
		return newUtilityGeneratedKV("text-decoration-style", first().Ident), nil
	case len(parts) == 1 && first().Int != nil:
		p := first()
		return newUtilityGeneratedKVf("text-decoration-thickness", "%spx", p.String()), nil
	case len(parts) == 1 && first().Ident == "auto":
		return newUtilityGeneratedKV("text-decoration-thickness", "auto"), nil
	case simpleMatch(parts, "from", "font"):
		return newUtilityGeneratedKV("text-decoration-thickness", "from-font"), nil
	case len(parts) == 1 && first().Custom != "":
		p := first()
		k, _ := p.CustomParts()
		switch k {
		case "length":
			return g.cssCustom(ctx, "text-decoration-thickness", p), nil
		default:
			return g.cssCustom(ctx, "text-decoration-color", p), nil
		}
	case len(parts) == 1 && first().Arbitrary != "":
		p := first()
		k, v := p.ArbitraryParts()
		switch {
		case k == "length":
			return g.cssArbitrary(ctx, "text-decoration-thickness", p), nil
		case k == "color":
			return g.cssArbitrary(ctx, "text-decoration-color", p), nil
		case cssIsLength(v):
			return g.cssArbitrary(ctx, "text-decoration-thickness", p), nil
		case cssIsColor(v):
			return g.cssArbitrary(ctx, "text-decoration-color", p), nil
		default:
			return g.cssArbitrary(ctx, "text-decoration-color", p), nil
		}
	default:
		return g.cssColor(ctx, "text-decoration-color", parts, mod)
	}
}

func (g *gen) utilityUnderline(
	ctx context.Context, parts []*parser.NamePart, negative bool,
) (*utilityGenerated, error) {
	switch {
	case len(parts) == 0:
		return newUtilityGeneratedKV("text-decoration-line", "underline"), nil
	case len(parts) == 2 && parts[0].Ident == "offset":
		p := parts[1]
		switch {
		case p.Ident == "auto":
			return newUtilityGeneratedKV("text-underline-offset", "auto"), nil
		case p.Int != nil && !negative:
			return newUtilityGeneratedKVf("text-underline-offset", "%spx", p.String()), nil
		case p.Int != nil && negative:
			return newUtilityGeneratedKVf("text-underline-offset", "calc(%spx * -1)", p.String()), nil
		case p.Custom != "":
			return g.cssCustom(ctx, "text-underline-offset", p), nil
		case p.Arbitrary != "":
			return g.cssArbitrary(ctx, "text-underline-offset", p), nil
		}
	}

	return nil, errUtilityUnknown
}

func (g *gen) utilityIndent(
	ctx context.Context, p *parser.NamePart, negative bool,
) (*utilityGenerated, error) {
	k := "text-indent"

	switch {
	case p.Int != nil || p.Float != nil:
		return g.cssCalcSpacing(ctx, k, p, negative), nil
	case p.Ident == "px" && !negative:
		return newUtilityGeneratedKV(k, "1px"), nil
	case p.Ident == "px" && negative:
		return newUtilityGeneratedKV(k, "-1px"), nil
	case p.Custom != "":
		return g.cssCustom(ctx, k, p), nil
	case p.Arbitrary != "":
		return g.cssArbitrary(ctx, k, p), nil
	}

	return nil, errUtilityUnknown
}

func (g *gen) utilityTab(ctx context.Context, p *parser.NamePart) (*utilityGenerated, error) {
	k := "tab-size"

	switch {
	case p.Int != nil:
		return newUtilityGeneratedKV(k, p.String()), nil
	case p.Custom != "":
		return g.cssCustom(ctx, k, p), nil
	case p.Arbitrary != "":
		return g.cssArbitrary(ctx, k, p), nil
	}

	return nil, errUtilityUnknown
}

func (g *gen) utilityAlign(ctx context.Context, parts []*parser.NamePart) (*utilityGenerated, error) {
	k := "vertical-align"
	p := parts[0]

	switch {
	case simpleMatch(parts, "baseline"):
		return newUtilityGeneratedKV(k, "baseline"), nil
	case simpleMatch(parts, "top"):
		return newUtilityGeneratedKV(k, "top"), nil
	case simpleMatch(parts, "middle"):
		return newUtilityGeneratedKV(k, "middle"), nil
	case simpleMatch(parts, "bottom"):
		return newUtilityGeneratedKV(k, "bottom"), nil
	case simpleMatch(parts, "text", "top"):
		return newUtilityGeneratedKV(k, "text-top"), nil
	case simpleMatch(parts, "text", "bottom"):
		return newUtilityGeneratedKV(k, "text-bottom"), nil
	case simpleMatch(parts, "sub"):
		return newUtilityGeneratedKV(k, "sub"), nil
	case simpleMatch(parts, "super"):
		return newUtilityGeneratedKV(k, "super"), nil
	case len(parts) == 1 && p.Custom != "":
		return g.cssCustom(ctx, k, p), nil
	case len(parts) == 1 && p.Arbitrary != "":
		return g.cssArbitrary(ctx, k, p), nil
	}

	return nil, errUtilityUnknown
}

func (g *gen) utilityWhitespace(parts []*parser.NamePart) (*utilityGenerated, error) {
	k := "white-space"

	switch {
	case simpleMatch(parts, "normal"):
		return newUtilityGeneratedKV(k, "normal"), nil
	case simpleMatch(parts, "nowrap"):
		return newUtilityGeneratedKV(k, "nowrap"), nil
	case simpleMatch(parts, "pre"):
		return newUtilityGeneratedKV(k, "pre"), nil
	case simpleMatch(parts, "pre", "line"):
		return newUtilityGeneratedKV(k, "pre-line"), nil
	case simpleMatch(parts, "pre", "wrap"):
		return newUtilityGeneratedKV(k, "pre-wrap"), nil
	case simpleMatch(parts, "break", "spaces"):
		return newUtilityGeneratedKV(k, "break-spaces"), nil
	}

	return nil, errUtilityUnknown
}

func (g *gen) utilityWrap(parts []*parser.NamePart) (*utilityGenerated, error) {
	k := "overflow-wrap"

	switch {
	case simpleMatch(parts, "break", "word"):
		return newUtilityGeneratedKV(k, "break-word"), nil
	case simpleMatch(parts, "anywhere"):
		return newUtilityGeneratedKV(k, "anywhere"), nil
	case simpleMatch(parts, "normal"):
		return newUtilityGeneratedKV(k, "normal"), nil
	}

	return nil, errUtilityUnknown
}

func (g *gen) utilityHyphens(parts []*parser.NamePart) (*utilityGenerated, error) {
	k := "hyphens"

	switch {
	case simpleMatch(parts, "none"):
		return newUtilityGeneratedKV(k, "none"), nil
	case simpleMatch(parts, "manual"):
		return newUtilityGeneratedKV(k, "manual"), nil
	case simpleMatch(parts, "auto"):
		return newUtilityGeneratedKV(k, "auto"), nil
	}

	return nil, errUtilityUnknown
}

func (g *gen) utilityBackground(
	ctx context.Context, parts []*parser.NamePart, negative bool, mod *parser.SlashValue,
) (*utilityGenerated, error) {
	l := len(parts)
	bgColor := "background-color"

	switch {
	case simpleMatch(parts, "none"):
		return newUtilityGeneratedKV("background-image", "none"), nil

	case simpleMatch(parts, "top", "left"):
		return newUtilityGeneratedKV("background-position", "top left"), nil
	case simpleMatch(parts, "top"):
		return newUtilityGeneratedKV("background-position", "top"), nil
	case simpleMatch(parts, "top", "right"):
		return newUtilityGeneratedKV("background-position", "top right"), nil
	case simpleMatch(parts, "left"):
		return newUtilityGeneratedKV("background-position", "left"), nil
	case simpleMatch(parts, "center"):
		return newUtilityGeneratedKV("background-position", "center"), nil
	case simpleMatch(parts, "right"):
		return newUtilityGeneratedKV("background-position", "right"), nil
	case simpleMatch(parts, "bottom", "left"):
		return newUtilityGeneratedKV("background-position", "bottom left"), nil
	case simpleMatch(parts, "bottom"):
		return newUtilityGeneratedKV("background-position", "bottom"), nil
	case simpleMatch(parts, "bottom", "right"):
		return newUtilityGeneratedKV("background-position", "bottom right"), nil

	case l == 2 && parts[0].Ident == "position" && parts[1].Custom != "":
		return g.cssCustom(ctx, "background-position", parts[1]), nil
	case l == 2 && parts[0].Ident == "position" && parts[1].Arbitrary != "":
		return g.cssArbitrary(ctx, "background-position", parts[1]), nil

	case simpleMatch(parts, "repeat"):
		return newUtilityGeneratedKV("background-repeat", "repeat"), nil
	case simpleMatch(parts, "repeat", "x"):
		return newUtilityGeneratedKV("background-repeat", "repeat-x"), nil
	case simpleMatch(parts, "repeat", "y"):
		return newUtilityGeneratedKV("background-repeat", "repeat-y"), nil
	case simpleMatch(parts, "repeat", "space"):
		return newUtilityGeneratedKV("background-repeat", "space"), nil
	case simpleMatch(parts, "repeat", "round"):
		return newUtilityGeneratedKV("background-repeat", "round"), nil
	case simpleMatch(parts, "no", "repeat"):
		return newUtilityGeneratedKV("background-repeat", "no-repeat"), nil

	case l == 1 && slices.Contains([]string{"fixed", "local", "scroll"}, parts[0].Ident):
		return newUtilityGeneratedKV("background-attachment", parts[0].Ident), nil

	case l == 1 && slices.Contains([]string{"auto", "cover", "contain"}, parts[0].Ident):
		return newUtilityGeneratedKV("background-size", parts[0].Ident), nil
	case l == 2 && parts[0].Ident == "size" && parts[1].Custom != "":
		return g.cssCustom(ctx, "background-size", parts[1]), nil
	case l == 2 && parts[0].Ident == "size" && parts[1].Arbitrary != "":
		return g.cssArbitrary(ctx, "background-size", parts[1]), nil

	case l == 2 && parts[0].Ident == "clip":
		p := parts[1]
		bgClip := "background-clip"

		switch {
		case slices.Contains([]string{"border", "padding", "content"}, p.Ident):
			return newUtilityGeneratedKVf(bgClip, "%s-box", p.Ident), nil
		case p.Ident == "text":
			return newUtilityGeneratedKV(bgClip, p.Ident), nil
		}

	case l == 2 && parts[0].Ident == "origin" &&
		slices.Contains([]string{"border", "padding", "content"}, parts[1].Ident):
		return newUtilityGeneratedKVf("background-origin", "%s-box", parts[1].Ident), nil

	case !negative && mod == nil && simpleMatch(parts, "blend", "normal"):
		return newUtilityGeneratedKV("background-blend-mode", "normal"), nil
	case !negative && mod == nil && simpleMatch(parts, "blend", "multiply"):
		return newUtilityGeneratedKV("background-blend-mode", "multiply"), nil
	case !negative && mod == nil && simpleMatch(parts, "blend", "screen"):
		return newUtilityGeneratedKV("background-blend-mode", "screen"), nil
	case !negative && mod == nil && simpleMatch(parts, "blend", "overlay"):
		return newUtilityGeneratedKV("background-blend-mode", "overlay"), nil
	case !negative && mod == nil && simpleMatch(parts, "blend", "darken"):
		return newUtilityGeneratedKV("background-blend-mode", "darken"), nil
	case !negative && mod == nil && simpleMatch(parts, "blend", "lighten"):
		return newUtilityGeneratedKV("background-blend-mode", "lighten"), nil
	case !negative && mod == nil && simpleMatch(parts, "blend", "color", "dodge"):
		return newUtilityGeneratedKV("background-blend-mode", "color-dodge"), nil
	case !negative && mod == nil && simpleMatch(parts, "blend", "color", "burn"):
		return newUtilityGeneratedKV("background-blend-mode", "color-burn"), nil
	case !negative && mod == nil && simpleMatch(parts, "blend", "hard", "light"):
		return newUtilityGeneratedKV("background-blend-mode", "hard-light"), nil
	case !negative && mod == nil && simpleMatch(parts, "blend", "soft", "light"):
		return newUtilityGeneratedKV("background-blend-mode", "soft-light"), nil
	case !negative && mod == nil && simpleMatch(parts, "blend", "difference"):
		return newUtilityGeneratedKV("background-blend-mode", "difference"), nil
	case !negative && mod == nil && simpleMatch(parts, "blend", "exclusion"):
		return newUtilityGeneratedKV("background-blend-mode", "exclusion"), nil
	case !negative && mod == nil && simpleMatch(parts, "blend", "hue"):
		return newUtilityGeneratedKV("background-blend-mode", "hue"), nil
	case !negative && mod == nil && simpleMatch(parts, "blend", "saturation"):
		return newUtilityGeneratedKV("background-blend-mode", "saturation"), nil
	case !negative && mod == nil && simpleMatch(parts, "blend", "color"):
		return newUtilityGeneratedKV("background-blend-mode", "color"), nil
	case !negative && mod == nil && simpleMatch(parts, "blend", "luminosity"):
		return newUtilityGeneratedKV("background-blend-mode", "luminosity"), nil

	case l >= 2 && parts[0].Ident == "linear":
		switch {
		case simpleMatch(parts, "linear", "to", "t"):
			return g.gradientLinear(ctx, "to top"), nil
		case simpleMatch(parts, "linear", "to", "tr"):
			return g.gradientLinear(ctx, "to top right"), nil
		case simpleMatch(parts, "linear", "to", "r"):
			return g.gradientLinear(ctx, "to right"), nil
		case simpleMatch(parts, "linear", "to", "br"):
			return g.gradientLinear(ctx, "to bottom right"), nil
		case simpleMatch(parts, "linear", "to", "b"):
			return g.gradientLinear(ctx, "to bottom"), nil
		case simpleMatch(parts, "linear", "to", "bl"):
			return g.gradientLinear(ctx, "to bottom left"), nil
		case simpleMatch(parts, "linear", "to", "l"):
			return g.gradientLinear(ctx, "to left"), nil
		case simpleMatch(parts, "linear", "to", "tl"):
			return g.gradientLinear(ctx, "to top left"), nil
		case l == 2 && parts[1].Int != nil:
			angle := fmt.Sprintf("%ddeg", *parts[1].Int)
			if negative {
				angle = fmt.Sprintf("calc(%s * -1)", angle)
			}

			return g.gradientLinear(ctx, angle), nil
		case l == 2 && parts[1].Custom != "":
			kv := g.cssCustomKV(ctx, "", parts[1])
			return g.gradientLinearFallback(ctx, kv.Value), nil
		case l == 2 && parts[1].Arbitrary != "":
			kv := g.cssArbitraryKV(ctx, "", parts[1])
			return g.gradientLinearFallback(ctx, kv.Value), nil
		}

	case simpleMatch(parts, "radial"):
		return &utilityGenerated{
			kvs: []KV{
				g.cssDefine(ctx, "--tw-gradient-position", "in oklab"),
				g.cssKV(ctx, "background-image", g.radialGradientImage(ctx)),
			},
		}, nil

	case l == 2 && parts[0].Ident == "radial":
		switch {
		case parts[1].Custom != "":
			kv := g.cssCustomKV(ctx, "", parts[1])
			return g.gradientRadialFallback(ctx, kv.Value), nil
		case parts[1].Arbitrary != "":
			kv := g.cssArbitraryKV(ctx, "", parts[1])
			return g.gradientRadialFallback(ctx, kv.Value), nil
		}

	case l == 2 && parts[0].Ident == "conic":
		switch {
		case parts[1].Int != nil:
			angle := fmt.Sprintf("from %ddeg in oklab", *parts[1].Int)
			if negative {
				angle = fmt.Sprintf("from calc(%ddeg * -1) in oklab", *parts[1].Int)
			}

			return g.gradientConic(ctx, angle), nil
		case parts[1].Custom != "":
			kv := g.cssCustomKV(ctx, "", parts[1])
			return g.gradientConicFallback(ctx, kv.Value), nil
		case parts[1].Arbitrary != "":
			kv := g.cssArbitraryKV(ctx, "", parts[1])
			return g.gradientConicFallback(ctx, kv.Value), nil
		}

	case l == 1 && parts[0].Custom != "":
		k, _ := parts[0].CustomParts()
		if k == "image" {
			return g.cssCustom(ctx, "background-image", parts[0]), nil
		}

		return g.cssCustom(ctx, bgColor, parts[0]), nil

	case l == 1 && parts[0].Arbitrary != "":
		k, v := parts[0].ArbitraryParts()
		switch {
		case k == "color" || cssIsColor(v):
			kv := g.cssArbitraryKV(ctx, bgColor, parts[0])

			if mod != nil && mod.Int != nil {
				return newUtilityGeneratedKVf(
					kv.Key, "color-mix(in oklab, %s %s%%, transparent)", kv.Value, mod.String(),
				), nil
			}

			return g.cssArbitrary(ctx, bgColor, parts[0]), nil
		}

		return g.cssArbitrary(ctx, "background-image", parts[0]), nil

	default:
		return g.cssColor(ctx, bgColor, parts, mod)
	}

	return nil, errUtilityUnknown
}

func (g *gen) utilityFrom(
	ctx context.Context, parts []*parser.NamePart, mod *parser.SlashValue,
) (*utilityGenerated, error) {
	k := "--tw-gradient-from"

	switch {
	case len(parts) == 1 && parts[0].Percentage != "":
		return newUtilityGeneratedKV("--tw-gradient-from-position", parts[0].Percentage), nil

	case len(parts) == 1 && parts[0].Custom != "":
		return &utilityGenerated{
			kvs: []KV{
				g.cssCustomDefine(ctx, k, parts[0]),
				g.cssDefine(ctx, "--tw-gradient-stops", g.gradientStopsValue(ctx)),
			},
		}, nil

	case len(parts) == 1 && parts[0].Arbitrary != "":
		return &utilityGenerated{
			kvs: []KV{
				g.cssArbitraryDefine(ctx, k, parts[0]),
				g.cssDefine(ctx, "--tw-gradient-stops", g.gradientStopsValue(ctx)),
			},
		}, nil

	default:
		c, err := g.cssColorDefine(ctx, k, parts, mod)
		if err != nil {
			return nil, errUtilityUnknown
		}

		return &utilityGenerated{
			kvs: []KV{
				c,
				g.cssDefine(ctx, "--tw-gradient-stops", g.gradientStopsValue(ctx)),
			},
		}, nil
	}
}

func (g *gen) utilityVia(
	ctx context.Context, parts []*parser.NamePart, mod *parser.SlashValue,
) (*utilityGenerated, error) {
	k := "--tw-gradient-via"

	switch {
	case len(parts) == 1 && parts[0].Percentage != "":
		return newUtilityGeneratedKV("--tw-gradient-via-position", parts[0].Percentage), nil

	case len(parts) == 1 && parts[0].Custom != "":
		return &utilityGenerated{
			kvs: []KV{
				g.cssCustomDefine(ctx, k, parts[0]),
				g.cssDefine(ctx, "--tw-gradient-via-stops", g.gradientViaStopsValue(ctx)),
				g.cssDefine(ctx, "--tw-gradient-stops", g.cssRef(ctx, "--tw-gradient-via-stops")),
			},
		}, nil

	case len(parts) == 1 && parts[0].Arbitrary != "":
		return &utilityGenerated{
			kvs: []KV{
				g.cssArbitraryDefine(ctx, k, parts[0]),
				g.cssDefine(ctx, "--tw-gradient-via-stops", g.gradientViaStopsValue(ctx)),
				g.cssDefine(ctx, "--tw-gradient-stops", g.cssRef(ctx, "--tw-gradient-via-stops")),
			},
		}, nil

	default:
		c, err := g.cssColorDefine(ctx, k, parts, mod)
		if err != nil {
			return nil, errUtilityUnknown
		}

		return &utilityGenerated{
			kvs: []KV{
				c,
				g.cssDefine(ctx, "--tw-gradient-via-stops", g.gradientViaStopsValue(ctx)),
				g.cssDefine(ctx, "--tw-gradient-stops", g.cssRef(ctx, "--tw-gradient-via-stops")),
			},
		}, nil
	}
}

func (g *gen) utilityTo(
	ctx context.Context, parts []*parser.NamePart, mod *parser.SlashValue,
) (*utilityGenerated, error) {
	k := "--tw-gradient-to"

	switch {
	case len(parts) == 1 && parts[0].Percentage != "":
		return newUtilityGeneratedKV("--tw-gradient-to-position", parts[0].Percentage), nil

	case len(parts) == 1 && parts[0].Custom != "":
		return &utilityGenerated{
			kvs: []KV{
				g.cssCustomDefine(ctx, k, parts[0]),
				g.cssDefine(ctx, "--tw-gradient-stops", g.gradientStopsValue(ctx)),
			},
		}, nil

	case len(parts) == 1 && parts[0].Arbitrary != "":
		return &utilityGenerated{
			kvs: []KV{
				g.cssArbitraryDefine(ctx, k, parts[0]),
				g.cssDefine(ctx, "--tw-gradient-stops", g.gradientStopsValue(ctx)),
			},
		}, nil

	default:
		c, err := g.cssColorDefine(ctx, k, parts, mod)
		if err != nil {
			return nil, errUtilityUnknown
		}

		return &utilityGenerated{
			kvs: []KV{
				c,
				g.cssDefine(ctx, "--tw-gradient-stops", g.gradientStopsValue(ctx)),
			},
		}, nil
	}
}

func (g *gen) utilityBorderRadius(
	ctx context.Context, p *parser.NamePart, keys ...string,
) (*utilityGenerated, error) {
	kvs := []KV{}

	switch {
	case p.Ident == "none":
		for _, k := range keys {
			kvs = append(kvs, KV{k, "0"})
		}

	case p.Ident == "full":
		for _, k := range keys {
			kvs = append(kvs, KV{k, "calc(infinity * 1px)"})
		}

	case isRadiusSizePart(p):
		for _, k := range keys {
			kvs = append(kvs, KV{k, g.cssVar(ctx, fmt.Sprintf("--radius-%s", p.String()))})
		}

	case p.Custom != "":
		for _, k := range keys {
			kvs = append(kvs, g.cssCustomKV(ctx, k, p))
		}

	case p.Arbitrary != "":
		for _, k := range keys {
			kvs = append(kvs, g.cssArbitraryKV(ctx, k, p))
		}

	default:
		return nil, errUtilityUnknown
	}

	return &utilityGenerated{kvs: kvs}, nil
}

func (g *gen) utilityRounded(ctx context.Context, parts []*parser.NamePart) (*utilityGenerated, error) {
	switch {
	case len(parts) == 1:
		return g.utilityBorderRadius(ctx, parts[0], "border-radius")

	case len(parts) == 2:
		p0, p1 := parts[0], parts[1]

		switch p0.Ident {
		case "s":
			return g.utilityBorderRadius(ctx, p1, "border-start-start-radius", "border-end-start-radius")
		case "e":
			return g.utilityBorderRadius(ctx, p1, "border-start-end-radius", "border-end-end-radius")
		case "t":
			return g.utilityBorderRadius(ctx, p1, "border-top-left-radius", "border-top-right-radius")
		case "r":
			return g.utilityBorderRadius(ctx, p1, "border-top-right-radius", "border-bottom-right-radius")
		case "b":
			return g.utilityBorderRadius(ctx, p1, "border-bottom-right-radius", "border-bottom-left-radius")
		case "l":
			return g.utilityBorderRadius(ctx, p1, "border-top-left-radius", "border-bottom-left-radius")
		case "ss":
			return g.utilityBorderRadius(ctx, p1, "border-start-start-radius")
		case "se":
			return g.utilityBorderRadius(ctx, p1, "border-start-end-radius")
		case "ee":
			return g.utilityBorderRadius(ctx, p1, "border-end-end-radius")
		case "es":
			return g.utilityBorderRadius(ctx, p1, "border-end-start-radius")
		case "tl":
			return g.utilityBorderRadius(ctx, p1, "border-top-left-radius")
		case "tr":
			return g.utilityBorderRadius(ctx, p1, "border-top-right-radius")
		case "br":
			return g.utilityBorderRadius(ctx, p1, "border-bottom-right-radius")
		case "bl":
			return g.utilityBorderRadius(ctx, p1, "border-bottom-left-radius")
		}
	}

	return nil, errUtilityUnknown
}

func (g *gen) utilityBorder(
	ctx context.Context, parts []*parser.NamePart, mod *parser.SlashValue,
) (*utilityGenerated, error) {
	sub := func(k string) bool { return len(parts) >= 1 && parts[0].Ident == k }
	style := func(k string) bool { return len(parts) == 1 && parts[0].Ident == k }

	switch {
	case style("solid") || style("dashed") || style("dotted") ||
		style("double") || style("hidden") || style("none"):
		v := parts[0].String()
		return &utilityGenerated{
			kvs: []KV{
				g.cssDefine(ctx, "--tw-border-style", v),
				g.cssKV(ctx, "border-style", v),
			},
		}, nil
	case len(parts) == 1 && parts[0].Ident == "collapse":
		if mod != nil {
			return nil, errUtilityUnknown
		}
		return newUtilityGeneratedKV("border-collapse", "collapse"), nil
	case len(parts) == 1 && parts[0].Ident == "separate":
		if mod != nil {
			return nil, errUtilityUnknown
		}
		return newUtilityGeneratedKV("border-collapse", "separate"), nil
	case len(parts) >= 2 && parts[0].Ident == "spacing":
		return g.utilityBorderSpacing(ctx, parts[1:], mod)
	case sub("x"):
		return g.utilityBorderAny(ctx, "border-inline", parts[1:], mod)
	case sub("y"):
		return g.utilityBorderAny(ctx, "border-block", parts[1:], mod)
	case sub("s"):
		return g.utilityBorderAny(ctx, "border-inline-start", parts[1:], mod)
	case sub("e"):
		return g.utilityBorderAny(ctx, "border-inline-end", parts[1:], mod)
	case sub("bs"):
		return g.utilityBorderAny(ctx, "border-block-start", parts[1:], mod)
	case sub("be"):
		return g.utilityBorderAny(ctx, "border-block-end", parts[1:], mod)
	case sub("t"):
		return g.utilityBorderAny(ctx, "border-top", parts[1:], mod)
	case sub("r"):
		return g.utilityBorderAny(ctx, "border-right", parts[1:], mod)
	case sub("b"):
		return g.utilityBorderAny(ctx, "border-bottom", parts[1:], mod)
	case sub("l"):
		return g.utilityBorderAny(ctx, "border-left", parts[1:], mod)
	default:
		return g.utilityBorderAny(ctx, "border", parts, mod)
	}
}

func (g *gen) borderSpacingValue(ctx context.Context, p *parser.NamePart) (string, error) {
	switch {
	case p.Int != nil || p.Float != nil:
		return fmt.Sprintf("calc(%s * %s)", g.cssVar(ctx, "--spacing"), p.String()), nil
	case p.Custom != "":
		_, v := p.CustomParts()
		return g.cssVar(ctx, v), nil
	case p.Arbitrary != "":
		_, v := p.ArbitraryParts()
		return arbitraryValue(v), nil
	default:
		return "", errUtilityUnknown
	}
}

func (g *gen) utilityBorderSpacing(
	ctx context.Context, parts []*parser.NamePart, mod *parser.SlashValue,
) (*utilityGenerated, error) {
	if mod != nil {
		return nil, errUtilityUnknown
	}

	switch len(parts) {
	case 1:
		v, err := g.borderSpacingValue(ctx, parts[0])
		if err != nil {
			return nil, err
		}

		return &utilityGenerated{
			kvs: []KV{
				g.cssDefine(ctx, "--tw-border-spacing-x", v),
				g.cssDefine(ctx, "--tw-border-spacing-y", v),
				g.cssKV(ctx, "border-spacing", g.borderSpacingComposite(ctx)),
			},
		}, nil

	case 2:
		v, err := g.borderSpacingValue(ctx, parts[1])
		if err != nil {
			return nil, err
		}

		switch parts[0].Ident {
		case "x":
			return &utilityGenerated{
				kvs: []KV{
					g.cssDefine(ctx, "--tw-border-spacing-x", v),
					g.cssKV(ctx, "border-spacing", g.borderSpacingComposite(ctx)),
				},
			}, nil
		case "y":
			return &utilityGenerated{
				kvs: []KV{
					g.cssDefine(ctx, "--tw-border-spacing-y", v),
					g.cssKV(ctx, "border-spacing", g.borderSpacingComposite(ctx)),
				},
			}, nil
		}
	}

	return nil, errUtilityUnknown
}

func (g *gen) utilityBorderAny(
	ctx context.Context, k string, parts []*parser.NamePart, mod *parser.SlashValue,
) (*utilityGenerated, error) {
	wk, ck, sk := fmt.Sprintf("%s-width", k), fmt.Sprintf("%s-color", k), fmt.Sprintf("%s-style", k)

	switch {
	case len(parts) == 0:
		return &utilityGenerated{
			kvs: []KV{
				g.cssKV(ctx, sk, g.cssVar(ctx, "--tw-border-style")),
				g.cssKV(ctx, wk, "1px"),
			},
		}, nil

	case len(parts) == 1 && parts[0].Int != nil:
		return &utilityGenerated{
			kvs: []KV{
				g.cssKV(ctx, sk, g.cssVar(ctx, "--tw-border-style")),
				g.cssKV(ctx, wk, parts[0].String()+"px"),
			},
		}, nil

	case len(parts) == 1 && parts[0].Custom != "":
		p := parts[0]
		cuk, _ := p.CustomParts()
		switch cuk {
		case "length":
			return &utilityGenerated{
				kvs: []KV{
					g.cssKV(ctx, sk, g.cssVar(ctx, "--tw-border-style")),
					g.cssCustomKV(ctx, wk, p),
				},
			}, nil
		default:
			return g.cssCustom(ctx, ck, p), nil
		}

	case len(parts) == 1 && parts[0].Arbitrary != "":
		p := parts[0]
		ak, av := p.ArbitraryParts()
		switch {
		case ak == "length":
			return &utilityGenerated{
				kvs: []KV{
					g.cssKV(ctx, sk, g.cssVar(ctx, "--tw-border-style")),
					g.cssArbitraryKV(ctx, wk, p),
				},
			}, nil
		case ak == "color":
			return g.cssArbitrary(ctx, ck, p), nil
		case cssIsColor(av):
			return g.cssArbitrary(ctx, ck, p), nil
		case cssIsLength(av):
			return &utilityGenerated{
				kvs: []KV{
					g.cssKV(ctx, sk, g.cssVar(ctx, "--tw-border-style")),
					g.cssArbitraryKV(ctx, wk, p),
				},
			}, nil
		default:
			return g.cssCustom(ctx, ck, p), nil
		}

	default:
		return g.cssColor(ctx, ck, parts, mod)
	}
}

func (g *gen) utilityDivide(
	ctx context.Context, parts []*parser.NamePart, mod *parser.SlashValue,
) (*utilityGenerated, error) {
	nested := func(kvs ...KV) *utilityGenerated {
		q := "& > :not(:last-child)"
		return &utilityGenerated{
			others: []Query{
				{Name: q, KVs: kvs},
			},
		}

	}
	style := func(k string) bool { return len(parts) == 1 && parts[0].Ident == k }

	switch {
	case style("solid") || style("dashed") || style("dotted") ||
		style("double") || style("hidden") || style("none"):
		v := parts[0].String()
		return nested(
			g.cssDefine(ctx, "--tw-border-style", v),
			g.cssKV(ctx, "border-style", v),
		), nil

	case len(parts) >= 1 && parts[0].Ident == "x":
		return g.utilityDivideAny(
			ctx,
			[]string{"border-inline-style"},
			"--tw-divide-x-reverse", "border-inline-start-width", "border-inline-end-width",
			parts[1:],
		)

	case len(parts) >= 1 && parts[0].Ident == "y":
		return g.utilityDivideAny(
			ctx,
			[]string{"border-bottom-style", "border-top-style"},
			"--tw-divide-y-reverse", "border-top-width", "border-bottom-width",
			parts[1:],
		)

	case len(parts) == 1 && parts[0].Custom != "":
		return nested(g.cssCustomKV(ctx, "border-color", parts[0])), nil

	case len(parts) == 1 && parts[0].Arbitrary != "":
		return nested(g.cssArbitraryKV(ctx, "border-color", parts[0])), nil

	default:
		kv, err := g.cssColorKV(ctx, "border-color", parts, mod)
		if err != nil {
			return nil, err
		}

		return nested(kv), nil
	}
}

func (g *gen) utilityDivideAny(
	ctx context.Context, styleKeys []string, ktw, k1, k2 string, parts []*parser.NamePart,
) (*utilityGenerated, error) {
	nested := func(kvs ...KV) *utilityGenerated {
		q := "& > :not(:last-child)"
		return &utilityGenerated{
			others: []Query{
				{Name: q, KVs: kvs},
			},
		}
	}

	switch {
	case len(parts) == 0:
		kvs := []KV{g.cssDefine(ctx, ktw, "0")}
		for _, k := range styleKeys {
			kvs = append(kvs, g.cssKV(ctx, k, g.cssVar(ctx, "--tw-border-style")))
		}
		kvs = append(kvs,
			g.cssKV(ctx, k1, fmt.Sprintf("calc(1px * %s)", g.cssVar(ctx, ktw))),
			g.cssKV(ctx, k2, fmt.Sprintf("calc(1px * calc(1 - %s))", g.cssVar(ctx, ktw))),
		)

		return nested(kvs...), nil

	case len(parts) == 1 && parts[0].Ident == "reverse":
		return nested(g.cssDefine(ctx, ktw, "1")), nil

	case len(parts) == 1 && parts[0].Int != nil:
		n := parts[0].String()
		kvs := []KV{g.cssDefine(ctx, ktw, "0")}
		for _, k := range styleKeys {
			kvs = append(kvs, g.cssKV(ctx, k, g.cssVar(ctx, "--tw-border-style")))
		}
		kvs = append(kvs,
			g.cssKV(ctx, k1, fmt.Sprintf("calc(%spx * %s)", n, g.cssVar(ctx, ktw))),
			g.cssKV(ctx, k2, fmt.Sprintf("calc(%spx * calc(1 - %s))", n, g.cssVar(ctx, ktw))),
		)

		return nested(kvs...), nil

	case len(parts) == 1 && parts[0].Custom != "":
		_, cv := parts[0].CustomParts()
		n := g.cssVar(ctx, cv)

		kvs := []KV{g.cssDefine(ctx, ktw, "0")}
		for _, k := range styleKeys {
			kvs = append(kvs, g.cssKV(ctx, k, g.cssVar(ctx, "--tw-border-style")))
		}
		kvs = append(kvs,
			g.cssKV(ctx, k1, fmt.Sprintf("calc(%s * %s)", n, g.cssVar(ctx, ktw))),
			g.cssKV(ctx, k2, fmt.Sprintf("calc(%s * calc(1 - %s))", n, g.cssVar(ctx, ktw))),
		)

		return nested(kvs...), nil

	case len(parts) == 1 && parts[0].Arbitrary != "":
		_, n := parts[0].ArbitraryParts()

		kvs := []KV{g.cssDefine(ctx, ktw, "0")}
		for _, k := range styleKeys {
			kvs = append(kvs, g.cssKV(ctx, k, g.cssVar(ctx, "--tw-border-style")))
		}
		kvs = append(kvs,
			g.cssKV(ctx, k1, fmt.Sprintf("calc(%s * %s)", n, g.cssVar(ctx, ktw))),
			g.cssKV(ctx, k2, fmt.Sprintf("calc(%s * calc(1 - %s))", n, g.cssVar(ctx, ktw))),
		)

		return nested(kvs...), nil

	default:
		return nil, errUtilityUnknown
	}
}

func (g *gen) utilityOutline(
	ctx context.Context, parts []*parser.NamePart, negative bool, mod *parser.SlashValue,
) (*utilityGenerated, error) {
	var (
		osk = "outline-style"
		owk = "outline-width"
		ock = "outline-color"
	)

	switch {
	case len(parts) == 0:
		return &utilityGenerated{
			kvs: []KV{
				g.cssKV(ctx, osk, g.cssVar(ctx, "--tw-outline-style")),
				g.cssKV(ctx, owk, "1px"),
			},
		}, nil

	case len(parts) == 1 && parts[0].Int != nil:
		return &utilityGenerated{
			kvs: []KV{
				g.cssKV(ctx, osk, g.cssVar(ctx, "--tw-outline-style")),
				g.cssKVf(ctx, owk, "%spx", parts[0].String()),
			},
		}, nil

	case len(parts) == 1 && slices.Contains(
		[]string{"dashed", "dotted", "double", "solid", "none"}, parts[0].Ident,
	):
		p0 := parts[0]

		return &utilityGenerated{
			kvs: []KV{
				g.cssDefine(ctx, "--tw-outline-style", p0.Ident),
				g.cssKV(ctx, "outline-style", p0.Ident),
			},
		}, nil

	case len(parts) == 1 && parts[0].Ident == "hidden":
		return &utilityGenerated{
			kvs: []KV{
				g.cssDefine(ctx, "--tw-outline-style", "none"),
				g.cssKV(ctx, "outline-style", "none"),
			},
			others: []Query{
				{
					Name: "@media (forced-colors: active)",
					KVs: []KV{
						g.cssKV(ctx, "outline", "2px solid transparent"),
						g.cssKV(ctx, "outline-offset", "2px"),
					},
				},
			},
		}, nil

	case len(parts) == 1 && parts[0].Custom != "":
		p := parts[0]
		cuk, _ := p.CustomParts()
		switch cuk {
		case "length":
			return &utilityGenerated{
				kvs: []KV{
					g.cssKV(ctx, osk, g.cssVar(ctx, "--tw-outline-style")),
					g.cssCustomKV(ctx, owk, p),
				},
			}, nil
		default:
			return g.cssCustom(ctx, ock, p), nil
		}

	case len(parts) == 1 && parts[0].Arbitrary != "":
		p := parts[0]
		ak, av := p.ArbitraryParts()
		switch {
		case ak == "length":
			return &utilityGenerated{
				kvs: []KV{
					g.cssKV(ctx, osk, g.cssVar(ctx, "--tw-outline-style")),
					g.cssArbitraryKV(ctx, owk, p),
				},
			}, nil
		case ak == "color":
			return g.cssArbitrary(ctx, ock, p), nil
		case cssIsColor(av):
			return g.cssArbitrary(ctx, ock, p), nil
		case cssIsLength(av):
			return &utilityGenerated{
				kvs: []KV{
					g.cssKV(ctx, osk, g.cssVar(ctx, "--tw-outline-style")),
					g.cssArbitraryKV(ctx, owk, p),
				},
			}, nil
		default:
			return g.cssCustom(ctx, ock, p), nil
		}

	case len(parts) == 2 && parts[0].Ident == "offset":
		k := "outline-offset"
		p := parts[1]

		switch {
		case p.Int != nil:
			n := p.String()

			if negative {
				return newUtilityGeneratedKVf(k, "calc(%spx * -1)", n), nil
			}

			return newUtilityGeneratedKVf(k, "%spx", n), nil
		case p.Custom != "":
			return g.cssCustom(ctx, k, p), nil
		case p.Arbitrary != "":
			return g.cssArbitrary(ctx, k, p), nil
		}

	default:
		return g.cssColor(ctx, ock, parts, mod)
	}

	return nil, errUtilityUnknown
}

const (
	genNullShadow               = "0 0 #0000"
	transitionDefaultProperty   = "color, background-color, border-color, outline-color, text-decoration-color, fill, stroke, --tw-gradient-from, --tw-gradient-via, --tw-gradient-to, opacity, box-shadow, transform, translate, scale, rotate, filter, -webkit-backdrop-filter, backdrop-filter, display, content-visibility, overlay, pointer-events"
	transitionColorsProperty    = "color, background-color, border-color, outline-color, text-decoration-color, fill, stroke, --tw-gradient-from, --tw-gradient-via, --tw-gradient-to"
	transitionTransformProperty = "transform, translate, scale, rotate"
)

func (g *gen) joinCssRefs(ctx context.Context, sep string, keys ...string) string {
	parts := make([]string, len(keys))
	for i, k := range keys {
		parts[i] = g.cssRef(ctx, k)
	}
	return strings.Join(parts, sep)
}

func (g *gen) boxShadowComposite(ctx context.Context) string {
	return g.joinCssRefs(ctx, ", ",
		"--tw-inset-shadow", "--tw-inset-ring-shadow", "--tw-ring-offset-shadow", "--tw-ring-shadow", "--tw-shadow",
	)
}

func (g *gen) filterComposite(ctx context.Context) string {
	return strings.Join([]string{
		g.cssVarBlankFallback(ctx, "--tw-blur"),
		g.cssVarBlankFallback(ctx, "--tw-brightness"),
		g.cssVarBlankFallback(ctx, "--tw-contrast"),
		g.cssVarBlankFallback(ctx, "--tw-grayscale"),
		g.cssVarBlankFallback(ctx, "--tw-hue-rotate"),
		g.cssVarBlankFallback(ctx, "--tw-invert"),
		g.cssVarBlankFallback(ctx, "--tw-saturate"),
		g.cssVarBlankFallback(ctx, "--tw-sepia"),
		g.cssVarBlankFallback(ctx, "--tw-drop-shadow"),
	}, " ")
}

func (g *gen) backdropFilterComposite(ctx context.Context) string {
	return strings.Join([]string{
		g.cssVarBlankFallback(ctx, "--tw-backdrop-blur"),
		g.cssVarBlankFallback(ctx, "--tw-backdrop-brightness"),
		g.cssVarBlankFallback(ctx, "--tw-backdrop-contrast"),
		g.cssVarBlankFallback(ctx, "--tw-backdrop-grayscale"),
		g.cssVarBlankFallback(ctx, "--tw-backdrop-hue-rotate"),
		g.cssVarBlankFallback(ctx, "--tw-backdrop-invert"),
		g.cssVarBlankFallback(ctx, "--tw-backdrop-opacity"),
		g.cssVarBlankFallback(ctx, "--tw-backdrop-saturate"),
		g.cssVarBlankFallback(ctx, "--tw-backdrop-sepia"),
	}, " ")
}

func (g *gen) borderSpacingComposite(ctx context.Context) string {
	return g.joinCssRefs(ctx, " ", "--tw-border-spacing-x", "--tw-border-spacing-y")
}

func (g *gen) transitionDefaultEase(ctx context.Context) string {
	return g.cssVarFallback(ctx, "--tw-ease", "ease")
}

func (g *gen) transitionDefaultDuration(ctx context.Context) string {
	return g.cssVarFallback(ctx, "--tw-duration", "0s")
}

func (g *gen) transformComposite(ctx context.Context) string {
	return strings.Join([]string{
		g.cssVarBlankFallback(ctx, "--tw-rotate-x"),
		g.cssVarBlankFallback(ctx, "--tw-rotate-y"),
		g.cssVarBlankFallback(ctx, "--tw-rotate-z"),
		g.cssVarBlankFallback(ctx, "--tw-skew-x"),
		g.cssVarBlankFallback(ctx, "--tw-skew-y"),
	}, " ")
}

func (g *gen) translateComposite(ctx context.Context) string {
	return g.joinCssRefs(ctx, " ", "--tw-translate-x", "--tw-translate-y")
}

func (g *gen) translate3dComposite(ctx context.Context) string {
	return g.joinCssRefs(ctx, " ", "--tw-translate-x", "--tw-translate-y", "--tw-translate-z")
}

func (g *gen) scaleComposite(ctx context.Context) string {
	return g.joinCssRefs(ctx, " ", "--tw-scale-x", "--tw-scale-y")
}

func (g *gen) scale3dComposite(ctx context.Context) string {
	return g.joinCssRefs(ctx, " ", "--tw-scale-x", "--tw-scale-y", "--tw-scale-z")
}

func (g *gen) touchActionComposite(ctx context.Context) string {
	return strings.Join([]string{
		g.cssVarBlankFallback(ctx, "--tw-pan-x"),
		g.cssVarBlankFallback(ctx, "--tw-pan-y"),
		g.cssVarBlankFallback(ctx, "--tw-pinch-zoom"),
	}, " ")
}

func (g *gen) scrollbarColorComposite(ctx context.Context) string {
	return g.joinCssRefs(ctx, " ", "--tw-scrollbar-thumb", "--tw-scrollbar-track")
}

func (g *gen) gradientStopsValue(ctx context.Context) string {
	fallback := strings.Join([]string{
		g.cssRef(ctx, "--tw-gradient-position"),
		g.cssRef(ctx, "--tw-gradient-from") + " " + g.cssRef(ctx, "--tw-gradient-from-position"),
		g.cssRef(ctx, "--tw-gradient-to") + " " + g.cssRef(ctx, "--tw-gradient-to-position"),
	}, ", ")
	return g.cssVarFallback(ctx, "--tw-gradient-via-stops", fallback)
}

func (g *gen) gradientViaStopsValue(ctx context.Context) string {
	return strings.Join([]string{
		g.cssRef(ctx, "--tw-gradient-position"),
		g.cssRef(ctx, "--tw-gradient-from") + " " + g.cssRef(ctx, "--tw-gradient-from-position"),
		g.cssRef(ctx, "--tw-gradient-via") + " " + g.cssRef(ctx, "--tw-gradient-via-position"),
		g.cssRef(ctx, "--tw-gradient-to") + " " + g.cssRef(ctx, "--tw-gradient-to-position"),
	}, ", ")
}

func (g *gen) linearGradientImage(ctx context.Context) string {
	return fmt.Sprintf("linear-gradient(%s)", g.cssRef(ctx, "--tw-gradient-stops"))
}

func (g *gen) radialGradientImage(ctx context.Context) string {
	return fmt.Sprintf("radial-gradient(%s)", g.cssRef(ctx, "--tw-gradient-stops"))
}

func (g *gen) conicGradientImage(ctx context.Context) string {
	return fmt.Sprintf("conic-gradient(%s)", g.cssRef(ctx, "--tw-gradient-stops"))
}

func (g *gen) linearGradientImageFallback(ctx context.Context, fallback string) string {
	return fmt.Sprintf("linear-gradient(%s)", g.cssVarFallbackTight(ctx, "--tw-gradient-stops", fallback))
}

func (g *gen) radialGradientImageFallback(ctx context.Context, fallback string) string {
	return fmt.Sprintf("radial-gradient(%s)", g.cssVarFallbackTight(ctx, "--tw-gradient-stops", fallback))
}

func (g *gen) conicGradientImageFallback(ctx context.Context, fallback string) string {
	return fmt.Sprintf("conic-gradient(%s)", g.cssVarFallbackTight(ctx, "--tw-gradient-stops", fallback))
}

func (g *gen) dropShadowSizeRef(ctx context.Context) string {
	return g.cssRef(ctx, "--tw-drop-shadow-size")
}

var transformOriginStatic = map[string]string{
	"center":       "center",
	"top":          "top",
	"top-right":    "100% 0",
	"right":        "100%",
	"bottom-right": "100% 100%",
	"bottom":       "bottom",
	"bottom-left":  "0 100%",
	"left":         "0",
	"top-left":     "0 0",
}

var (
	cursorStaticValues = map[string]string{
		"auto": "auto", "default": "default", "pointer": "pointer", "wait": "wait",
		"text": "text", "move": "move", "help": "help", "not-allowed": "not-allowed",
		"none": "none", "context-menu": "context-menu", "progress": "progress",
		"cell": "cell", "crosshair": "crosshair", "vertical-text": "vertical-text",
		"alias": "alias", "copy": "copy", "no-drop": "no-drop", "grab": "grab",
		"grabbing": "grabbing", "all-scroll": "all-scroll", "col-resize": "col-resize",
		"row-resize": "row-resize", "n-resize": "n-resize", "e-resize": "e-resize",
		"s-resize": "s-resize", "w-resize": "w-resize", "ne-resize": "ne-resize",
		"nw-resize": "nw-resize", "se-resize": "se-resize", "sw-resize": "sw-resize",
		"ew-resize": "ew-resize", "ns-resize": "ns-resize", "nesw-resize": "nesw-resize",
		"nwse-resize": "nwse-resize", "zoom-in": "zoom-in", "zoom-out": "zoom-out",
	}
	scrollSpacingProperties = map[string]string{
		"m": "scroll-margin", "mx": "scroll-margin-inline", "my": "scroll-margin-block",
		"ms": "scroll-margin-inline-start", "me": "scroll-margin-inline-end",
		"mbs": "scroll-margin-block-start", "mbe": "scroll-margin-block-end",
		"mt": "scroll-margin-top", "mr": "scroll-margin-right",
		"mb": "scroll-margin-bottom", "ml": "scroll-margin-left",
		"p": "scroll-padding", "px": "scroll-padding-inline", "py": "scroll-padding-block",
		"ps": "scroll-padding-inline-start", "pe": "scroll-padding-inline-end",
		"pbs": "scroll-padding-block-start", "pbe": "scroll-padding-block-end",
		"pt": "scroll-padding-top", "pr": "scroll-padding-right",
		"pb": "scroll-padding-bottom", "pl": "scroll-padding-left",
	}
	colorSchemeStatic = map[string]string{
		"normal": "normal", "dark": "dark", "light": "light",
		"light-dark": "light dark", "only-dark": "only dark", "only-light": "only light",
	}
)

type shadowTheme struct {
	layers []string
	color  string
}

var (
	themeShadows = map[string]shadowTheme{
		"2xs": {layers: []string{"0 1px %s"}, color: "rgb(0 0 0 / 0.05)"},
		"xs":  {layers: []string{"0 1px 2px 0 %s"}, color: "rgb(0 0 0 / 0.05)"},
		"sm":  {layers: []string{"0 1px 3px 0 %s", "0 1px 2px -1px %s"}, color: "rgb(0 0 0 / 0.1)"},
		"md":  {layers: []string{"0 4px 6px -1px %s", "0 2px 4px -2px %s"}, color: "rgb(0 0 0 / 0.1)"},
		"lg":  {layers: []string{"0 10px 15px -3px %s", "0 4px 6px -4px %s"}, color: "rgb(0 0 0 / 0.1)"},
		"xl":  {layers: []string{"0 20px 25px -5px %s", "0 8px 10px -6px %s"}, color: "rgb(0 0 0 / 0.1)"},
		"2xl": {layers: []string{"0 25px 50px -12px %s"}, color: "rgb(0 0 0 / 0.25)"},
	}
	themeInsetShadows = map[string]shadowTheme{
		"2xs": {layers: []string{"inset 0 1px %s"}, color: "rgb(0 0 0 / 0.05)"},
		"xs":  {layers: []string{"inset 0 1px 1px %s"}, color: "rgb(0 0 0 / 0.05)"},
		"sm":  {layers: []string{"inset 0 2px 4px %s"}, color: "rgb(0 0 0 / 0.05)"},
	}
	themeTextShadows = map[string]shadowTheme{
		"2xs": {layers: []string{"0px 1px 0px %s"}, color: "rgb(0 0 0 / 0.15)"},
		"xs":  {layers: []string{"0px 1px 1px %s"}, color: "rgb(0 0 0 / 0.2)"},
		"sm":  {layers: []string{"0px 1px 0px %s", "0px 1px 1px %s", "0px 2px 2px %s"}, color: "rgb(0 0 0 / 0.075)"},
		"md":  {layers: []string{"0px 1px 1px %s", "0px 1px 2px %s", "0px 2px 4px %s"}, color: "rgb(0 0 0 / 0.1)"},
		"lg":  {layers: []string{"0px 1px 2px %s", "0px 3px 2px %s", "0px 4px 8px %s"}, color: "rgb(0 0 0 / 0.1)"},
	}
	themeBlurSizes = map[string]struct{}{
		"xs": {}, "sm": {}, "md": {}, "lg": {}, "xl": {}, "2xl": {}, "3xl": {},
	}
	themeDropShadows = map[string]struct {
		layers []string
		colors []string
	}{
		"xs":  {layers: []string{"0 1px 1px %s"}, colors: []string{"rgb(0 0 0 / 0.05)"}},
		"sm":  {layers: []string{"0 1px 2px %s"}, colors: []string{"rgb(0 0 0 / 0.15)"}},
		"md":  {layers: []string{"0 3px 3px %s"}, colors: []string{"rgb(0 0 0 / 0.12)"}},
		"lg":  {layers: []string{"0 4px 4px %s"}, colors: []string{"rgb(0 0 0 / 0.15)"}},
		"xl":  {layers: []string{"0 9px 7px %s"}, colors: []string{"rgb(0 0 0 / 0.1)"}},
		"2xl": {layers: []string{"0 25px 25px %s"}, colors: []string{"rgb(0 0 0 / 0.15)"}},
		"multi": {
			layers: []string{"0 1px 1px %s", "0 9px 7px %s"},
			colors: []string{"rgb(0 0 0 / 0.05)", "rgb(0 0 0 / 0.1)"},
		},
	}
)

func shadowModifierAlpha(mod *parser.SlashValue) (string, error) {
	if mod == nil {
		return "", nil
	}

	if mod.Int != nil {
		return fmt.Sprintf("%s%%", mod.String()), nil
	}

	if mod.Arbitrary != "" {
		_, v := mod.ArbitraryParts()
		return arbitraryValue(v), nil
	}

	if mod.Custom != "" {
		_, v := mod.CustomParts()
		return customValue(v), nil
	}

	return "", errUtilityUnknown
}

func shadowLayerColor(baseColor, alpha string) string {
	if alpha == "" {
		return baseColor
	}

	return fmt.Sprintf("oklab(from %s l a b / %s)", baseColor, alpha)
}

func (g *gen) buildThemedShadowValue(ctx context.Context, theme shadowTheme, colorVar, alpha string) string {
	wrapped := g.cssVarFallback(ctx, colorVar, shadowLayerColor(theme.color, alpha))
	parts := make([]string, len(theme.layers))
	for i, layer := range theme.layers {
		parts[i] = fmt.Sprintf(layer, wrapped)
	}

	return strings.Join(parts, ", ")
}

func (g *gen) themedShadowKVs(
	ctx context.Context,
	size string,
	themes map[string]shadowTheme,
	prop string,
	inset bool,
	mod *parser.SlashValue,
) ([]KV, error) {
	theme, ok := themes[size]
	if !ok {
		return nil, errUtilityUnknown
	}

	alpha, err := shadowModifierAlpha(mod)
	if err != nil {
		return nil, err
	}

	kvs := []KV{g.cssDefine(ctx, prop, g.buildThemedShadowValue(ctx, theme, prop+"-color", alpha))}
	if mod != nil {
		var kv KV
		if inset {
			kv, err = g.insetShadowAlphaKV(ctx, mod)
		} else {
			kv, err = g.shadowAlphaKV(ctx, mod)
		}
		if err != nil {
			return nil, err
		}
		kvs = append([]KV{kv}, kvs...)
	}

	return kvs, nil
}

func (g *gen) withBoxShadow(ctx context.Context, kvs ...KV) *utilityGenerated {
	kvs = append(kvs, g.cssKV(ctx, "box-shadow", g.boxShadowComposite(ctx)))
	return &utilityGenerated{kvs: kvs}
}

func (g *gen) shadowAlphaKV(ctx context.Context, mod *parser.SlashValue) (KV, error) {
	if mod == nil {
		return KV{}, errUtilityUnknown
	}

	if mod.Int != nil {
		return g.cssDefine(ctx, "--tw-shadow-alpha", fmt.Sprintf("%s%%", mod.String())), nil
	}

	if mod.Arbitrary != "" {
		_, v := mod.ArbitraryParts()
		return g.cssDefine(ctx, "--tw-shadow-alpha", g.cssArbitraryValue(ctx, arbitraryValue(v))), nil
	}

	if mod.Custom != "" {
		_, v := mod.CustomParts()
		return g.cssDefine(ctx, "--tw-shadow-alpha", customValue(v)), nil
	}

	return KV{}, errUtilityUnknown
}

func (g *gen) insetShadowAlphaKV(ctx context.Context, mod *parser.SlashValue) (KV, error) {
	if mod == nil {
		return KV{}, errUtilityUnknown
	}

	if mod.Int != nil {
		return g.cssDefine(ctx, "--tw-inset-shadow-alpha", fmt.Sprintf("%s%%", mod.String())), nil
	}

	if mod.Arbitrary != "" {
		_, v := mod.ArbitraryParts()
		return g.cssDefine(ctx, "--tw-inset-shadow-alpha", g.cssArbitraryValue(ctx, arbitraryValue(v))), nil
	}

	if mod.Custom != "" {
		_, v := mod.CustomParts()
		return g.cssDefine(ctx, "--tw-inset-shadow-alpha", customValue(v)), nil
	}

	return KV{}, errUtilityUnknown
}

func (g *gen) ringShadowValue(ctx context.Context, width string) string {
	width = g.cssArbitraryValue(ctx, width)
	return fmt.Sprintf(
		"%s 0 0 0 calc(%s + %s) %s",
		g.cssVarFallback(ctx, "--tw-ring-inset", ""),
		width,
		g.cssRef(ctx, "--tw-ring-offset-width"),
		g.cssVarFallback(ctx, "--tw-ring-color", "currentcolor"),
	)
}

func (g *gen) insetRingShadowValue(ctx context.Context, width string) string {
	width = g.cssArbitraryValue(ctx, width)
	return fmt.Sprintf("inset 0 0 0 %s %s", width, g.cssVarFallback(ctx, "--tw-inset-ring-color", "currentcolor"))
}

func (g *gen) cssShadowColorKV(
	ctx context.Context, k, alphaVar string, parts []*parser.NamePart, mod *parser.SlashValue,
) (KV, error) {
	ckey := partsJoin(parts)
	color := g.cssVar(ctx, fmt.Sprintf("--color-%s", ckey))

	switch {
	case len(parts) == 1 && slices.Contains([]string{"inherit", "transparent"}, ckey):
		return g.cssDefine(ctx, k, ckey), nil
	case len(parts) == 1 && ckey == "current":
		return g.cssDefine(ctx, k, "currentcolor"), nil
	case color != "" && mod != nil && mod.Int != nil:
		return g.cssDefinef(ctx,
			k,
			"color-mix(in oklab, color-mix(in oklab, %s %s%%, transparent) %s, transparent)",
			color,
			mod.String(),
			g.cssRef(ctx, alphaVar),
		), nil
	case color != "":
		return g.cssDefinef(ctx, k, "color-mix(in oklab, %s %s, transparent)", color, g.cssRef(ctx, alphaVar)), nil
	default:
		return KV{}, errUtilityUnknown
	}
}

func (g *gen) cssShadowColor(
	ctx context.Context, k, alphaVar string, parts []*parser.NamePart, mod *parser.SlashValue,
) (*utilityGenerated, error) {
	kv, err := g.cssShadowColorKV(ctx, k, alphaVar, parts, mod)
	if err != nil {
		return nil, err
	}

	return &utilityGenerated{kvs: []KV{kv}}, nil
}

func (g *gen) shadowColorMix(ctx context.Context, color, alphaVar string) string {
	return fmt.Sprintf("color-mix(in oklab, %s %s, transparent)", color, g.cssRef(ctx, alphaVar))
}

func (g *gen) utilityShadow(
	ctx context.Context, parts []*parser.NamePart, mod *parser.SlashValue,
) (*utilityGenerated, error) {
	const alphaVar = "--tw-shadow-alpha"

	switch len(parts) {
	case 0:
		kvs, err := g.themedShadowKVs(ctx, "sm", themeShadows, "--tw-shadow", false, mod)
		if err != nil {
			return nil, err
		}
		return g.withBoxShadow(ctx, kvs...), nil

	case 1:
		p := parts[0]

		switch {
		case p.Ident == "none":
			if mod != nil {
				return nil, errUtilityUnknown
			}
			return g.withBoxShadow(ctx, g.cssDefine(ctx, "--tw-shadow", genNullShadow)), nil

		case p.Ident == "inherit":
			if mod != nil {
				return nil, errUtilityUnknown
			}
			return newUtilityGeneratedKV("--tw-shadow-color", "inherit"), nil

		case isShadowSizePart(p):
			kvs, err := g.themedShadowKVs(ctx, p.String(), themeShadows, "--tw-shadow", false, mod)
			if err != nil {
				return nil, err
			}
			return g.withBoxShadow(ctx, kvs...), nil

		case p.Custom != "":
			cuk, cv := p.CustomParts()
			switch cuk {
			case "color":
				return &utilityGenerated{kvs: []KV{g.cssDefine(ctx, "--tw-shadow-color", g.shadowColorMix(ctx, g.cssVar(ctx, cv), alphaVar))}}, nil
			default:
				kvs := []KV{g.cssCustomDefine(ctx, "--tw-shadow", p)}
				if mod != nil {
					alpha, err := g.shadowAlphaKV(ctx, mod)
					if err != nil {
						return nil, err
					}
					kvs = append([]KV{alpha}, kvs...)
				}
				return g.withBoxShadow(ctx, kvs...), nil
			}

		case p.Arbitrary != "":
			ak, av := p.ArbitraryParts()
			switch {
			case ak == "color":
				return g.utilityShadowColorArbitrary(ctx, alphaVar, p, mod)
			case cssIsColor(av):
				return g.cssShadowColor(ctx, "--tw-shadow-color", alphaVar, parts, mod)
			default:
				kvs := []KV{g.cssArbitraryDefine(ctx, "--tw-shadow", p)}
				if mod != nil {
					alpha, err := g.shadowAlphaKV(ctx, mod)
					if err != nil {
						return nil, err
					}
					kvs = append([]KV{alpha}, kvs...)
				}
				return g.withBoxShadow(ctx, kvs...), nil
			}
		}
	}

	return g.cssShadowColor(ctx, "--tw-shadow-color", alphaVar, parts, mod)
}

func (g *gen) utilityShadowColorArbitrary(
	ctx context.Context, alphaVar string, p *parser.NamePart, mod *parser.SlashValue,
) (*utilityGenerated, error) {
	_, v := p.ArbitraryParts()
	color := arbitraryValue(v)

	switch {
	case mod != nil && mod.Int != nil:
		color = fmt.Sprintf("color-mix(in oklab, %s %s%%, transparent)", color, mod.String())
	}

	return &utilityGenerated{kvs: []KV{g.cssDefine(ctx, "--tw-shadow-color", g.shadowColorMix(ctx, color, alphaVar))}}, nil
}

func (g *gen) utilityInsetShadow(
	ctx context.Context, parts []*parser.NamePart, mod *parser.SlashValue,
) (*utilityGenerated, error) {
	const alphaVar = "--tw-inset-shadow-alpha"

	switch len(parts) {
	case 0:
		kvs, err := g.themedShadowKVs(ctx, "sm", themeInsetShadows, "--tw-inset-shadow", true, mod)
		if err != nil {
			return nil, err
		}
		return g.withBoxShadow(ctx, kvs...), nil

	case 1:
		p := parts[0]

		switch {
		case p.Ident == "none":
			if mod != nil {
				return nil, errUtilityUnknown
			}
			return g.withBoxShadow(ctx, g.cssDefine(ctx, "--tw-inset-shadow", fmt.Sprintf("inset %s", genNullShadow))), nil

		case p.Ident == "inherit":
			if mod != nil {
				return nil, errUtilityUnknown
			}
			return newUtilityGeneratedKV("--tw-inset-shadow-color", "inherit"), nil

		case isShadowSizePart(p):
			kvs, err := g.themedShadowKVs(ctx, p.String(), themeInsetShadows, "--tw-inset-shadow", true, mod)
			if err != nil {
				return nil, err
			}
			return g.withBoxShadow(ctx, kvs...), nil

		case p.Custom != "":
			cuk, cv := p.CustomParts()
			switch cuk {
			case "color":
				return &utilityGenerated{kvs: []KV{g.cssDefine(ctx, "--tw-inset-shadow-color", g.shadowColorMix(ctx, g.cssVar(ctx, cv), alphaVar))}}, nil
			default:
				kvs := []KV{g.cssCustomDefine(ctx, "--tw-inset-shadow", p)}
				if mod != nil {
					alpha, err := g.insetShadowAlphaKV(ctx, mod)
					if err != nil {
						return nil, err
					}
					kvs = append([]KV{alpha}, kvs...)
				}
				return g.withBoxShadow(ctx, kvs...), nil
			}

		case p.Arbitrary != "":
			ak, av := p.ArbitraryParts()
			switch {
			case ak == "color":
				return g.utilityInsetShadowColorArbitrary(ctx, alphaVar, p, mod)
			case cssIsColor(av):
				return g.cssShadowColor(ctx, "--tw-inset-shadow-color", alphaVar, parts, mod)
			default:
				kvs := []KV{g.cssArbitraryDefine(ctx, "--tw-inset-shadow", p)}
				if mod != nil {
					alpha, err := g.insetShadowAlphaKV(ctx, mod)
					if err != nil {
						return nil, err
					}
					kvs = append([]KV{alpha}, kvs...)
				}
				return g.withBoxShadow(ctx, kvs...), nil
			}
		}
	}

	return g.cssShadowColor(ctx, "--tw-inset-shadow-color", alphaVar, parts, mod)
}

func (g *gen) utilityInsetShadowColorArbitrary(
	ctx context.Context, alphaVar string, p *parser.NamePart, mod *parser.SlashValue,
) (*utilityGenerated, error) {
	_, v := p.ArbitraryParts()
	color := arbitraryValue(v)

	if mod != nil && mod.Int != nil {
		color = fmt.Sprintf("color-mix(in oklab, %s %s%%, transparent)", color, mod.String())
	}

	return &utilityGenerated{kvs: []KV{g.cssDefine(ctx, "--tw-inset-shadow-color", g.shadowColorMix(ctx, color, alphaVar))}}, nil
}

func (g *gen) utilityRing(
	ctx context.Context, parts []*parser.NamePart, mod *parser.SlashValue,
) (*utilityGenerated, error) {
	switch len(parts) {
	case 0:
		if mod != nil {
			return nil, errUtilityUnknown
		}
		return g.withBoxShadow(ctx, g.cssDefine(ctx, "--tw-ring-shadow", g.ringShadowValue(ctx, "1px"))), nil

	case 1:
		p := parts[0]

		switch {
		case p.Ident == "inset":
			if mod != nil {
				return nil, errUtilityUnknown
			}
			return newUtilityGeneratedKV("--tw-ring-inset", "inset"), nil

		case p.Int != nil:
			if mod != nil {
				return nil, errUtilityUnknown
			}
			return g.withBoxShadow(ctx, g.cssDefine(ctx, "--tw-ring-shadow", g.ringShadowValue(ctx, fmt.Sprintf("%spx", p.String())))), nil

		case p.Custom != "":
			cuk, _ := p.CustomParts()
			switch cuk {
			case "length":
				if mod != nil {
					return nil, errUtilityUnknown
				}
				_, v := p.CustomParts()
				return g.withBoxShadow(ctx, g.cssDefine(ctx, "--tw-ring-shadow", g.ringShadowValue(ctx, g.cssVar(ctx, v)))), nil
			default:
				return g.cssColor(ctx, "--tw-ring-color", parts, mod)
			}

		case p.Arbitrary != "":
			ak, av := p.ArbitraryParts()
			switch {
			case ak == "length":
				if mod != nil {
					return nil, errUtilityUnknown
				}
				return g.withBoxShadow(ctx, g.cssDefine(ctx, "--tw-ring-shadow", g.ringShadowValue(ctx, g.cssArbitraryValue(ctx, arbitraryValue(av))))), nil
			case ak == "color", cssIsColor(av):
				return g.cssColor(ctx, "--tw-ring-color", parts, mod)
			default:
				if mod != nil {
					return nil, errUtilityUnknown
				}
				return g.withBoxShadow(ctx, g.cssDefine(ctx, "--tw-ring-shadow", g.ringShadowValue(ctx, g.cssArbitraryValue(ctx, arbitraryValue(av))))), nil
			}
		}
	}

	return g.cssColor(ctx, "--tw-ring-color", parts, mod)
}

func (g *gen) utilityInsetRing(
	ctx context.Context, parts []*parser.NamePart, mod *parser.SlashValue,
) (*utilityGenerated, error) {
	switch len(parts) {
	case 0:
		if mod != nil {
			return nil, errUtilityUnknown
		}
		return g.withBoxShadow(ctx, g.cssDefine(ctx, "--tw-inset-ring-shadow", g.insetRingShadowValue(ctx, "1px"))), nil

	case 1:
		p := parts[0]

		switch {
		case p.Int != nil:
			if mod != nil {
				return nil, errUtilityUnknown
			}
			return g.withBoxShadow(ctx, g.cssDefine(ctx, "--tw-inset-ring-shadow", g.insetRingShadowValue(ctx, fmt.Sprintf("%spx", p.String())))), nil

		case p.Custom != "":
			cuk, _ := p.CustomParts()
			switch cuk {
			case "length":
				if mod != nil {
					return nil, errUtilityUnknown
				}
				_, v := p.CustomParts()
				return g.withBoxShadow(ctx, g.cssDefine(ctx, "--tw-inset-ring-shadow", g.insetRingShadowValue(ctx, g.cssVar(ctx, v)))), nil
			default:
				return g.cssColor(ctx, "--tw-inset-ring-color", parts, mod)
			}

		case p.Arbitrary != "":
			ak, av := p.ArbitraryParts()
			switch {
			case ak == "length":
				if mod != nil {
					return nil, errUtilityUnknown
				}
				return g.withBoxShadow(ctx, g.cssDefine(ctx, "--tw-inset-ring-shadow", g.insetRingShadowValue(ctx, g.cssArbitraryValue(ctx, arbitraryValue(av))))), nil
			case ak == "color", cssIsColor(av):
				return g.cssColor(ctx, "--tw-inset-ring-color", parts, mod)
			default:
				if mod != nil {
					return nil, errUtilityUnknown
				}
				return g.withBoxShadow(ctx, g.cssDefine(ctx, "--tw-inset-ring-shadow", g.insetRingShadowValue(ctx, g.cssArbitraryValue(ctx, arbitraryValue(av))))), nil
			}
		}
	}

	return g.cssColor(ctx, "--tw-inset-ring-color", parts, mod)
}

func (g *gen) withFilter(ctx context.Context, kvs ...KV) *utilityGenerated {
	kvs = append(kvs, g.cssKV(ctx, "filter", g.filterComposite(ctx)))
	return &utilityGenerated{kvs: kvs}
}

func (g *gen) withBackdropFilter(ctx context.Context, kvs ...KV) *utilityGenerated {
	kvs = append(kvs,
		g.cssKV(ctx, "-webkit-backdrop-filter", g.backdropFilterComposite(ctx)),
		g.cssKV(ctx, "backdrop-filter", g.backdropFilterComposite(ctx)),
	)
	return &utilityGenerated{kvs: kvs}
}

type filterCompose func(context.Context, ...KV) *utilityGenerated

func isPositiveInteger(s string) bool {
	n, err := strconv.Atoi(s)
	return err == nil && n >= 0 && strconv.Itoa(n) == s
}

func filterPercentageValue(p *parser.NamePart) (string, bool) {
	if p.Int == nil {
		return "", false
	}

	s := p.String()
	if !isPositiveInteger(s) {
		return "", false
	}

	return s + "%", true
}

func (g *gen) filterFunctionValueCompose(
	ctx context.Context, fn, cssVar string, p *parser.NamePart, mod *parser.SlashValue, compose filterCompose,
) (*utilityGenerated, error) {
	if mod != nil {
		return nil, errUtilityUnknown
	}

	switch {
	case p.Int != nil:
		if v, ok := filterPercentageValue(p); ok {
			return compose(ctx, g.cssDefine(ctx, cssVar, fmt.Sprintf("%s(%s)", fn, v))), nil
		}
	case p.Custom != "":
		_, v := p.CustomParts()
		return compose(ctx, g.cssDefine(ctx, cssVar, fmt.Sprintf("%s(%s)", fn, g.cssVar(ctx, v)))), nil
	case p.Arbitrary != "":
		_, v := p.ArbitraryParts()
		return compose(ctx, g.cssDefine(ctx, cssVar, fmt.Sprintf("%s(%s)", fn, g.cssArbitraryValue(ctx, arbitraryValue(v))))), nil
	}

	return nil, errUtilityUnknown
}

func (g *gen) filterFunctionValue(
	ctx context.Context, fn, cssVar string, p *parser.NamePart, mod *parser.SlashValue,
) (*utilityGenerated, error) {
	return g.filterFunctionValueCompose(ctx, fn, cssVar, p, mod, g.withFilter)
}

func (g *gen) filterToggleFunctionCompose(
	ctx context.Context, fn, cssVar, enabled, disabled string, parts []*parser.NamePart, mod *parser.SlashValue,
	compose filterCompose,
) (*utilityGenerated, error) {
	if mod != nil {
		return nil, errUtilityUnknown
	}

	switch len(parts) {
	case 0:
		return compose(ctx, g.cssDefine(ctx, cssVar, fmt.Sprintf("%s(%s)", fn, enabled))), nil
	case 1:
		p := parts[0]
		switch {
		case p.Int != nil && *p.Int == 0:
			return compose(ctx, g.cssDefine(ctx, cssVar, fmt.Sprintf("%s(%s)", fn, disabled))), nil
		case p.Int != nil:
			if v, ok := filterPercentageValue(p); ok {
				return compose(ctx, g.cssDefine(ctx, cssVar, fmt.Sprintf("%s(%s)", fn, v))), nil
			}
		case p.Custom != "":
			_, v := p.CustomParts()
			return compose(ctx, g.cssDefine(ctx, cssVar, fmt.Sprintf("%s(%s)", fn, g.cssVar(ctx, v)))), nil
		case p.Arbitrary != "":
			_, v := p.ArbitraryParts()
			return compose(ctx, g.cssDefine(ctx, cssVar, fmt.Sprintf("%s(%s)", fn, g.cssArbitraryValue(ctx, arbitraryValue(v))))), nil
		}
	}

	return nil, errUtilityUnknown
}

func (g *gen) filterToggleFunction(
	ctx context.Context, fn, cssVar, enabled, disabled string, parts []*parser.NamePart, mod *parser.SlashValue,
) (*utilityGenerated, error) {
	return g.filterToggleFunctionCompose(ctx, fn, cssVar, enabled, disabled, parts, mod, g.withFilter)
}

func (g *gen) utilityFilter(
	ctx context.Context, parts []*parser.NamePart, mod *parser.SlashValue,
) (*utilityGenerated, error) {
	if mod != nil {
		return nil, errUtilityUnknown
	}

	switch len(parts) {
	case 0:
		return g.withFilter(ctx), nil
	case 1:
		p := parts[0]
		switch {
		case p.Ident == "none":
			return newUtilityGeneratedKV("filter", "none"), nil
		case p.Custom != "":
			return g.cssCustom(ctx, "filter", p), nil
		case p.Arbitrary != "":
			_, v := p.ArbitraryParts()
			return newUtilityGeneratedKV("filter", arbitraryValue(v)), nil
		}
	}

	return nil, errUtilityUnknown
}

func (g *gen) utilityBlur(
	ctx context.Context, parts []*parser.NamePart, mod *parser.SlashValue,
) (*utilityGenerated, error) {
	if mod != nil {
		return nil, errUtilityUnknown
	}

	switch len(parts) {
	case 1:
		p := parts[0]
		switch {
		case p.Ident == "none":
			return g.withFilter(ctx, g.cssDefine(ctx, "--tw-blur", " ")), nil
		case isBlurSizePart(p):
			return g.withFilter(ctx, g.cssDefine(ctx,
				"--tw-blur",
				fmt.Sprintf("blur(%s)", g.cssVar(ctx, fmt.Sprintf("--blur-%s", p.String()))),
			)), nil
		case p.Custom != "":
			_, v := p.CustomParts()
			return g.withFilter(ctx, g.cssDefine(ctx, "--tw-blur", fmt.Sprintf("blur(%s)", g.cssVar(ctx, v)))), nil
		case p.Arbitrary != "":
			_, v := p.ArbitraryParts()
			return g.withFilter(ctx, g.cssDefine(ctx, "--tw-blur", fmt.Sprintf("blur(%s)", g.cssArbitraryValue(ctx, arbitraryValue(v))))), nil
		}
	}

	return nil, errUtilityUnknown
}

func isBlurSizePart(p *parser.NamePart) bool {
	_, ok := themeBlurSizes[p.String()]
	return ok
}

func (g *gen) utilityBrightness(
	ctx context.Context, parts []*parser.NamePart, mod *parser.SlashValue,
) (*utilityGenerated, error) {
	if len(parts) != 1 {
		return nil, errUtilityUnknown
	}

	return g.filterFunctionValue(ctx, "brightness", "--tw-brightness", parts[0], mod)
}

func (g *gen) utilityContrast(
	ctx context.Context, parts []*parser.NamePart, mod *parser.SlashValue,
) (*utilityGenerated, error) {
	if len(parts) != 1 {
		return nil, errUtilityUnknown
	}

	return g.filterFunctionValue(ctx, "contrast", "--tw-contrast", parts[0], mod)
}

func (g *gen) utilityGrayscale(
	ctx context.Context, parts []*parser.NamePart, mod *parser.SlashValue,
) (*utilityGenerated, error) {
	return g.filterToggleFunction(ctx, "grayscale", "--tw-grayscale", "100%", "0%", parts, mod)
}

func (g *gen) utilityInvert(
	ctx context.Context, parts []*parser.NamePart, mod *parser.SlashValue,
) (*utilityGenerated, error) {
	return g.filterToggleFunction(ctx, "invert", "--tw-invert", "100%", "0%", parts, mod)
}

func (g *gen) utilitySepia(
	ctx context.Context, parts []*parser.NamePart, mod *parser.SlashValue,
) (*utilityGenerated, error) {
	return g.filterToggleFunction(ctx, "sepia", "--tw-sepia", "100%", "0%", parts, mod)
}

func (g *gen) utilitySaturate(
	ctx context.Context, parts []*parser.NamePart, mod *parser.SlashValue,
) (*utilityGenerated, error) {
	if len(parts) != 1 {
		return nil, errUtilityUnknown
	}

	return g.filterFunctionValue(ctx, "saturate", "--tw-saturate", parts[0], mod)
}

func (g *gen) utilityHueRotate(
	ctx context.Context, parts []*parser.NamePart, negative bool, mod *parser.SlashValue,
) (*utilityGenerated, error) {
	if mod != nil || len(parts) != 2 || parts[0].Ident != "rotate" {
		return nil, errUtilityUnknown
	}

	p := parts[1]
	wrapNegative := func(v string) string {
		if negative {
			return fmt.Sprintf("calc(%s * -1)", v)
		}

		return v
	}

	switch {
	case p.Int != nil:
		s := p.String()
		if !isPositiveInteger(s) {
			return nil, errUtilityUnknown
		}

		return g.withFilter(ctx, g.cssDefine(ctx,
			"--tw-hue-rotate",
			fmt.Sprintf("hue-rotate(%s)", wrapNegative(s+"deg")),
		)), nil
	case p.Custom != "":
		_, v := p.CustomParts()
		return g.withFilter(ctx, g.cssDefine(ctx,
			"--tw-hue-rotate",
			fmt.Sprintf("hue-rotate(%s)", wrapNegative(g.cssVar(ctx, v))),
		)), nil
	case p.Arbitrary != "":
		_, v := p.ArbitraryParts()
		return g.withFilter(ctx, g.cssDefine(ctx,
			"--tw-hue-rotate",
			fmt.Sprintf("hue-rotate(%s)", wrapNegative(g.cssArbitraryValue(ctx, arbitraryValue(v)))),
		)), nil
	}

	return nil, errUtilityUnknown
}

func (g *gen) dropShadowLayer(ctx context.Context, layer, color, alpha string) string {
	wrapped := g.cssVarFallback(ctx, "--tw-drop-shadow-color", shadowLayerColor(color, alpha))
	return fmt.Sprintf("drop-shadow(%s)", fmt.Sprintf(layer, wrapped))
}

func (g *gen) dropShadowColorOnly(
	ctx context.Context, parts []*parser.NamePart, mod *parser.SlashValue,
) (*utilityGenerated, error) {
	if len(parts) == 1 && parts[0].Custom != "" {
		cuk, cv := parts[0].CustomParts()
		if cuk == "color" {
			kv := g.cssDefine(ctx,
				"--tw-drop-shadow-color",
				g.shadowColorMix(ctx, g.cssVar(ctx, cv), "--tw-drop-shadow-alpha"),
			)
			if mod != nil && mod.Int != nil {
				kv = g.cssDefinef(ctx,
					"--tw-drop-shadow-color",
					"color-mix(in oklab, color-mix(in oklab, %s %s%%, transparent) %s, transparent)",
					g.cssVar(ctx, cv),
					mod.String(),
					g.cssRef(ctx, "--tw-drop-shadow-alpha"),
				)
			}

			return &utilityGenerated{
				kvs: []KV{
					kv,
					g.cssDefine(ctx, "--tw-drop-shadow", g.dropShadowSizeRef(ctx)),
				},
			}, nil
		}
	}

	kv, err := g.cssShadowColorKV(ctx, "--tw-drop-shadow-color", "--tw-drop-shadow-alpha", parts, mod)
	if err != nil {
		return nil, errUtilityUnknown
	}

	return &utilityGenerated{
		kvs: []KV{
			kv,
			g.cssDefine(ctx, "--tw-drop-shadow", g.dropShadowSizeRef(ctx)),
		},
	}, nil
}

func (g *gen) dropShadowThemedSize(
	ctx context.Context, name string, mod *parser.SlashValue, alpha string,
) (*utilityGenerated, error) {
	theme, ok := themeDropShadows[name]
	if !ok {
		if mod != nil && alpha == "" {
			return nil, errUtilityUnknown
		}

		kvs := []KV{
			g.cssDefine(ctx, "--tw-drop-shadow-size", g.dropShadowLayer(ctx,
				fmt.Sprintf("0 0 calc(1 * %s) %%s", g.cssVar(ctx, "--spacing")),
				"black",
				alpha,
			)),
			g.cssDefine(ctx, "--tw-drop-shadow", fmt.Sprintf(
				"drop-shadow(%s)",
				g.cssVar(ctx, fmt.Sprintf("--drop-shadow-%s", name)),
			)),
		}
		if mod != nil {
			kvs = append([]KV{g.cssDefine(ctx, "--tw-drop-shadow-alpha", alpha)}, kvs...)
		}

		return g.withFilter(ctx, kvs...), nil
	}

	if mod != nil && alpha == "" {
		return nil, errUtilityUnknown
	}

	layers := make([]string, len(theme.layers))
	for i, layer := range theme.layers {
		color := theme.colors[0]
		if i < len(theme.colors) {
			color = theme.colors[i]
		}

		layers[i] = g.dropShadowLayer(ctx, layer, color, alpha)
	}

	kvs := []KV{g.cssDefine(ctx, "--tw-drop-shadow-size", strings.Join(layers, " "))}

	if name == "multi" {
		kvs = append(kvs, g.cssDefine(ctx,
			"--tw-drop-shadow",
			"drop-shadow(0 1px 1px rgb(0 0 0 / 0.05)) drop-shadow(0 9px 7px rgb(0 0 0 / 0.1))",
		))
	} else {
		kvs = append(kvs, g.cssDefine(ctx,
			"--tw-drop-shadow",
			fmt.Sprintf("drop-shadow(%s)", g.cssVar(ctx, fmt.Sprintf("--drop-shadow-%s", name))),
		))
	}

	if mod != nil {
		kvs = append([]KV{g.cssDefine(ctx, "--tw-drop-shadow-alpha", alpha)}, kvs...)
	}

	return g.withFilter(ctx, kvs...), nil
}

func (g *gen) utilityDropShadow(
	ctx context.Context, parts []*parser.NamePart, mod *parser.SlashValue,
) (*utilityGenerated, error) {
	if len(parts) == 0 || parts[0].Ident != "shadow" {
		return nil, errUtilityUnknown
	}

	rest := parts[1:]
	alpha, err := shadowModifierAlpha(mod)
	if err != nil {
		return nil, err
	}

	switch len(rest) {
	case 0:
		kvs := []KV{
			g.cssDefine(ctx, "--tw-drop-shadow-size", g.dropShadowLayer(ctx, "0 1px 1px %s", "rgb(0 0 0 / 0.05)", alpha)),
			g.cssDefine(ctx, "--tw-drop-shadow", fmt.Sprintf("drop-shadow(%s)", g.cssVar(ctx, "--drop-shadow"))),
		}
		if mod != nil {
			kvs = append([]KV{g.cssDefine(ctx, "--tw-drop-shadow-alpha", alpha)}, kvs...)
		}

		return g.withFilter(ctx, kvs...), nil

	case 1:
		p := rest[0]

		switch {
		case p.Ident == "none":
			if mod != nil {
				return nil, errUtilityUnknown
			}

			return g.withFilter(ctx, g.cssDefine(ctx, "--tw-drop-shadow", " ")), nil

		case p.Ident == "inherit":
			if mod != nil {
				return nil, errUtilityUnknown
			}

			return &utilityGenerated{
				kvs: []KV{
					g.cssDefine(ctx, "--tw-drop-shadow-color", "inherit"),
					g.cssDefine(ctx, "--tw-drop-shadow", g.dropShadowSizeRef(ctx)),
				},
			}, nil

		case p.Ident != "" && p.Int == nil && p.Arbitrary == "" && p.Custom == "":
			return g.dropShadowThemedSize(ctx, p.String(), mod, alpha)

		case p.Custom != "":
			cuk, cv := p.CustomParts()
			if cuk == "color" {
				return g.dropShadowColorOnly(ctx, []*parser.NamePart{{Custom: p.Custom}}, mod)
			}

			if mod != nil && alpha == "" {
				return nil, errUtilityUnknown
			}

			kvs := []KV{
				g.cssDefine(ctx,
					"--tw-drop-shadow-size",
					fmt.Sprintf("drop-shadow(%s)", g.cssVar(ctx, cv)),
				),
				g.cssDefine(ctx, "--tw-drop-shadow", g.dropShadowSizeRef(ctx)),
			}
			if mod != nil {
				kvs = append([]KV{g.cssDefine(ctx, "--tw-drop-shadow-alpha", alpha)}, kvs...)
			}

			return g.withFilter(ctx, kvs...), nil

		case p.Arbitrary != "":
			ak, av := p.ArbitraryParts()
			switch {
			case ak == "color", cssIsColor(av):
				return g.dropShadowColorOnly(ctx, []*parser.NamePart{{Arbitrary: p.Arbitrary}}, mod)
			default:
				if mod != nil && alpha == "" {
					return nil, errUtilityUnknown
				}

				v := arbitraryValue(av)
				shadowSize := v
				if strings.HasSuffix(v, " red") {
					shadowSize = strings.Replace(v, " red", " "+g.cssVarFallback(ctx, "--tw-drop-shadow-color", "red"), 1)
				}

				kvs := []KV{
					g.cssDefine(ctx, "--tw-drop-shadow-size", fmt.Sprintf("drop-shadow(%s)", shadowSize)),
					g.cssDefine(ctx, "--tw-drop-shadow", g.dropShadowSizeRef(ctx)),
				}
				if mod != nil {
					kvs = append([]KV{g.cssDefine(ctx, "--tw-drop-shadow-alpha", alpha)}, kvs...)
				}

				return g.withFilter(ctx, kvs...), nil
			}
		}
	}

	if kv, err := g.cssShadowColorKV(ctx, "--tw-drop-shadow-color", "--tw-drop-shadow-alpha", rest, mod); err == nil {
		return &utilityGenerated{
			kvs: []KV{
				kv,
				g.cssDefine(ctx, "--tw-drop-shadow", g.dropShadowSizeRef(ctx)),
			},
		}, nil
	}

	return nil, errUtilityUnknown
}

func (g *gen) utilityBackdrop(
	ctx context.Context, parts []*parser.NamePart, negative bool, mod *parser.SlashValue,
) (*utilityGenerated, error) {
	if len(parts) == 0 {
		return nil, errUtilityUnknown
	}

	switch parts[0].Ident {
	case "filter":
		return g.utilityBackdropFilter(ctx, parts[1:], mod)
	case "blur":
		return g.utilityBackdropBlur(ctx, parts[1:], mod)
	case "brightness":
		if len(parts) != 2 {
			return nil, errUtilityUnknown
		}
		return g.filterFunctionValueCompose(ctx, "brightness", "--tw-backdrop-brightness", parts[1], mod, g.withBackdropFilter)
	case "contrast":
		if len(parts) != 2 {
			return nil, errUtilityUnknown
		}
		return g.filterFunctionValueCompose(ctx, "contrast", "--tw-backdrop-contrast", parts[1], mod, g.withBackdropFilter)
	case "grayscale":
		return g.filterToggleFunctionCompose(
			ctx, "grayscale", "--tw-backdrop-grayscale", "100%", "0%", parts[1:], mod, g.withBackdropFilter,
		)
	case "invert":
		return g.filterToggleFunctionCompose(
			ctx, "invert", "--tw-backdrop-invert", "100%", "0%", parts[1:], mod, g.withBackdropFilter,
		)
	case "saturate":
		if len(parts) != 2 {
			return nil, errUtilityUnknown
		}
		return g.filterFunctionValueCompose(ctx, "saturate", "--tw-backdrop-saturate", parts[1], mod, g.withBackdropFilter)
	case "sepia":
		return g.filterToggleFunctionCompose(
			ctx, "sepia", "--tw-backdrop-sepia", "100%", "0%", parts[1:], mod, g.withBackdropFilter,
		)
	case "opacity":
		return g.utilityBackdropOpacity(ctx, parts[1:], mod)
	case "hue":
		if len(parts) >= 2 && parts[1].Ident == "rotate" {
			return g.utilityBackdropHueRotate(ctx, parts[2:], negative, mod)
		}
	}

	return nil, errUtilityUnknown
}

func (g *gen) utilityBackdropFilter(
	ctx context.Context, parts []*parser.NamePart, mod *parser.SlashValue,
) (*utilityGenerated, error) {
	if mod != nil {
		return nil, errUtilityUnknown
	}

	switch len(parts) {
	case 0:
		return g.withBackdropFilter(ctx), nil
	case 1:
		p := parts[0]
		switch {
		case p.Ident == "none":
			return &utilityGenerated{
				kvs: []KV{
					g.cssKV(ctx, "-webkit-backdrop-filter", "none"),
					g.cssKV(ctx, "backdrop-filter", "none"),
				},
			}, nil
		case p.Custom != "":
			_, v := p.CustomParts()
			val := g.cssVar(ctx, v)
			return &utilityGenerated{
				kvs: []KV{
					g.cssKV(ctx, "-webkit-backdrop-filter", val),
					g.cssKV(ctx, "backdrop-filter", val),
				},
			}, nil
		case p.Arbitrary != "":
			_, v := p.ArbitraryParts()
			val := arbitraryValue(v)
			return &utilityGenerated{
				kvs: []KV{
					g.cssKV(ctx, "-webkit-backdrop-filter", val),
					g.cssKV(ctx, "backdrop-filter", val),
				},
			}, nil
		}
	}

	return nil, errUtilityUnknown
}

func (g *gen) utilityBackdropBlur(
	ctx context.Context, parts []*parser.NamePart, mod *parser.SlashValue,
) (*utilityGenerated, error) {
	if mod != nil {
		return nil, errUtilityUnknown
	}

	switch len(parts) {
	case 1:
		p := parts[0]
		switch {
		case p.Ident == "none":
			return g.withBackdropFilter(ctx, g.cssDefine(ctx, "--tw-backdrop-blur", " ")), nil
		case isBlurSizePart(p):
			return g.withBackdropFilter(ctx, g.cssDefine(ctx,
				"--tw-backdrop-blur",
				fmt.Sprintf("blur(%s)", g.cssVar(ctx, fmt.Sprintf("--blur-%s", p.String()))),
			)), nil
		case p.Custom != "":
			_, v := p.CustomParts()
			return g.withBackdropFilter(ctx, g.cssDefine(ctx,
				"--tw-backdrop-blur",
				fmt.Sprintf("blur(%s)", g.cssVar(ctx, v)),
			)), nil
		case p.Arbitrary != "":
			_, v := p.ArbitraryParts()
			return g.withBackdropFilter(ctx, g.cssDefine(ctx,
				"--tw-backdrop-blur",
				fmt.Sprintf("blur(%s)", g.cssArbitraryValue(ctx, arbitraryValue(v))),
			)), nil
		}
	}

	return nil, errUtilityUnknown
}

func (g *gen) utilityBackdropHueRotate(
	ctx context.Context, parts []*parser.NamePart, negative bool, mod *parser.SlashValue,
) (*utilityGenerated, error) {
	if mod != nil || len(parts) != 1 {
		return nil, errUtilityUnknown
	}

	p := parts[0]
	wrapNegative := func(v string) string {
		if negative {
			return fmt.Sprintf("calc(%s * -1)", v)
		}

		return v
	}

	switch {
	case p.Int != nil:
		s := p.String()
		if !isPositiveInteger(s) {
			return nil, errUtilityUnknown
		}

		return g.withBackdropFilter(ctx, g.cssDefine(ctx,
			"--tw-backdrop-hue-rotate",
			fmt.Sprintf("hue-rotate(%s)", wrapNegative(s+"deg")),
		)), nil
	case p.Custom != "":
		_, v := p.CustomParts()
		return g.withBackdropFilter(ctx, g.cssDefine(ctx,
			"--tw-backdrop-hue-rotate",
			fmt.Sprintf("hue-rotate(%s)", wrapNegative(g.cssVar(ctx, v))),
		)), nil
	case p.Arbitrary != "":
		_, v := p.ArbitraryParts()
		return g.withBackdropFilter(ctx, g.cssDefine(ctx,
			"--tw-backdrop-hue-rotate",
			fmt.Sprintf("hue-rotate(%s)", wrapNegative(g.cssArbitraryValue(ctx, arbitraryValue(v)))),
		)), nil
	}

	return nil, errUtilityUnknown
}

func (g *gen) utilityBackdropOpacity(
	ctx context.Context, parts []*parser.NamePart, mod *parser.SlashValue,
) (*utilityGenerated, error) {
	if len(parts) != 1 || mod != nil {
		return nil, errUtilityUnknown
	}

	p := parts[0]

	switch {
	case p.Int != nil, p.Float != nil:
		if v, ok := opacityPercent(p); ok {
			return g.withBackdropFilter(ctx, g.cssDefine(ctx,
				"--tw-backdrop-opacity",
				fmt.Sprintf("opacity(%s)", v),
			)), nil
		}
	case p.Custom != "":
		_, v := p.CustomParts()
		return g.withBackdropFilter(ctx, g.cssDefine(ctx,
			"--tw-backdrop-opacity",
			fmt.Sprintf("opacity(%s)", g.cssVar(ctx, v)),
		)), nil
	case p.Arbitrary != "":
		_, v := p.ArbitraryParts()
		return g.withBackdropFilter(ctx, g.cssDefine(ctx,
			"--tw-backdrop-opacity",
			fmt.Sprintf("opacity(%s)", g.cssArbitraryValue(ctx, arbitraryValue(v))),
		)), nil
	}

	return nil, errUtilityUnknown
}

func (g *gen) transitionWithDefaults(ctx context.Context, property string) *utilityGenerated {
	return &utilityGenerated{
		kvs: []KV{
			g.cssKV(ctx, "transition-property", g.cssArbitraryValue(ctx, property)),
			g.cssKV(ctx, "transition-timing-function", g.transitionDefaultEase(ctx)),
			g.cssKV(ctx, "transition-duration", g.transitionDefaultDuration(ctx)),
		},
	}
}

func transitionMsToSeconds(ms int) string {
	if ms%1000 == 0 {
		return fmt.Sprintf("%ds", ms/1000)
	}

	s := strconv.FormatFloat(float64(ms)/1000.0, 'f', -1, 64) + "s"
	if strings.HasPrefix(s, "0.") {
		return s[1:]
	}

	return s
}

func transitionTimeValue(p *parser.NamePart) (string, bool) {
	switch {
	case p.Int != nil:
		if !isPositiveInteger(p.String()) {
			return "", false
		}

		return transitionMsToSeconds(*p.Int), true
	case p.Custom != "":
		_, v := p.CustomParts()
		return customValue(v), true
	case p.Arbitrary != "":
		_, v := p.ArbitraryParts()
		av := arbitraryValue(v)
		if strings.HasSuffix(av, "ms") {
			numStr := strings.TrimSpace(strings.TrimSuffix(av, "ms"))
			if n, err := strconv.Atoi(numStr); err == nil && isPositiveInteger(numStr) {
				return transitionMsToSeconds(n), true
			}
		}

		return av, true
	default:
		return "", false
	}
}

func (g *gen) utilityTransition(
	ctx context.Context, parts []*parser.NamePart, mod *parser.SlashValue,
) (*utilityGenerated, error) {
	if mod != nil {
		return nil, errUtilityUnknown
	}

	switch len(parts) {
	case 0:
		return g.transitionWithDefaults(ctx, transitionDefaultProperty), nil
	case 1:
		p := parts[0]

		switch {
		case p.Ident == "none":
			return newUtilityGeneratedKV("transition-property", "none"), nil
		case p.Ident == "discrete":
			return newUtilityGeneratedKV("transition-behavior", "allow-discrete"), nil
		case p.Ident == "normal":
			return newUtilityGeneratedKV("transition-behavior", "normal"), nil
		case p.Ident == "all":
			return g.transitionWithDefaults(ctx, "all"), nil
		case p.Ident == "colors":
			return g.transitionWithDefaults(ctx, transitionColorsProperty), nil
		case p.Ident == "opacity":
			return g.transitionWithDefaults(ctx, "opacity"), nil
		case p.Ident == "shadow":
			return g.transitionWithDefaults(ctx, "box-shadow"), nil
		case p.Ident == "transform":
			return g.transitionWithDefaults(ctx, transitionTransformProperty), nil
		case p.Custom != "":
			_, v := p.CustomParts()
			return g.transitionWithDefaults(ctx, g.cssVar(ctx, v)), nil
		case p.Arbitrary != "":
			_, v := p.ArbitraryParts()
			return g.transitionWithDefaults(ctx, arbitraryValue(v)), nil
		}
	}

	return nil, errUtilityUnknown
}

func (g *gen) utilityTransitionDelay(
	_ context.Context, parts []*parser.NamePart, mod *parser.SlashValue,
) (*utilityGenerated, error) {
	if mod != nil || len(parts) != 1 {
		return nil, errUtilityUnknown
	}

	v, ok := transitionTimeValue(parts[0])
	if !ok {
		return nil, errUtilityUnknown
	}

	return newUtilityGeneratedKV("transition-delay", v), nil
}

func (g *gen) utilityTransitionDuration(
	ctx context.Context, parts []*parser.NamePart, mod *parser.SlashValue,
) (*utilityGenerated, error) {
	if mod != nil || len(parts) != 1 {
		return nil, errUtilityUnknown
	}

	p := parts[0]
	if p.Ident == "initial" {
		return newUtilityGeneratedKV("--tw-duration", "initial"), nil
	}

	v, ok := transitionTimeValue(p)
	if !ok {
		return nil, errUtilityUnknown
	}

	return &utilityGenerated{
		kvs: []KV{
			g.cssDefine(ctx, "--tw-duration", v),
			g.cssKV(ctx, "transition-duration", v),
		},
	}, nil
}

func (g *gen) utilityTransitionTimingFunction(
	ctx context.Context, parts []*parser.NamePart, mod *parser.SlashValue,
) (*utilityGenerated, error) {
	if mod != nil {
		return nil, errUtilityUnknown
	}

	switch len(parts) {
	case 0:
		return nil, errUtilityUnknown
	case 1:
		p := parts[0]

		switch {
		case p.Ident == "initial":
			return newUtilityGeneratedKV("--tw-ease", "initial"), nil
		case p.Ident == "linear":
			v := g.cssVar(ctx, "--ease-linear")
			if v != "" {
				return &utilityGenerated{
					kvs: []KV{
						g.cssDefine(ctx, "--tw-ease", v),
						g.cssKV(ctx, "transition-timing-function", v),
					},
				}, nil
			}

			return &utilityGenerated{
				kvs: []KV{
					g.cssDefine(ctx, "--tw-ease", "linear"),
					g.cssKV(ctx, "transition-timing-function", "linear"),
				},
			}, nil
		case p.Ident != "":
			v := g.cssVar(ctx, fmt.Sprintf("--ease-%s", p.String()))
			return &utilityGenerated{
				kvs: []KV{
					g.cssDefine(ctx, "--tw-ease", v),
					g.cssKV(ctx, "transition-timing-function", v),
				},
			}, nil
		case p.Custom != "":
			_, v := p.CustomParts()
			val := g.cssVar(ctx, v)
			return &utilityGenerated{
				kvs: []KV{
					g.cssDefine(ctx, "--tw-ease", val),
					g.cssKV(ctx, "transition-timing-function", val),
				},
			}, nil
		case p.Arbitrary != "":
			_, av := p.ArbitraryParts()
			v := arbitraryValue(av)
			return &utilityGenerated{
				kvs: []KV{
					g.cssDefine(ctx, "--tw-ease", v),
					g.cssKV(ctx, "transition-timing-function", v),
				},
			}, nil
		}
	default:
		name := partsJoin(parts)
		v := g.cssVar(ctx, fmt.Sprintf("--ease-%s", name))
		return &utilityGenerated{
			kvs: []KV{
				g.cssDefine(ctx, "--tw-ease", v),
				g.cssKV(ctx, "transition-timing-function", v),
			},
		}, nil
	}

	return nil, errUtilityUnknown
}

func (g *gen) utilityAnimate(
	ctx context.Context, parts []*parser.NamePart, mod *parser.SlashValue,
) (*utilityGenerated, error) {
	if mod != nil {
		return nil, errUtilityUnknown
	}

	p := parts[0]

	switch {
	case p.Ident == "none":
		return newUtilityGeneratedKV("animation", "none"), nil
	case p.Ident != "":
		return newUtilityGeneratedKV(
			"animation",
			g.cssVar(ctx, fmt.Sprintf("--animate-%s", partsJoin(parts))),
		), nil
	case p.Custom != "":
		_, v := p.CustomParts()
		return newUtilityGeneratedKV("animation", g.cssVar(ctx, v)), nil
	case p.Arbitrary != "":
		_, v := p.ArbitraryParts()
		return newUtilityGeneratedKV("animation", arbitraryValue(v)), nil
	}

	return nil, errUtilityUnknown
}

func (g *gen) withTransform(ctx context.Context, kvs ...KV) *utilityGenerated {
	kvs = append(kvs, g.cssKV(ctx, "transform", g.transformComposite(ctx)))
	return &utilityGenerated{kvs: kvs}
}

func (g *gen) withTranslate2d(ctx context.Context, kvs ...KV) *utilityGenerated {
	kvs = append(kvs, g.cssKV(ctx, "translate", g.translateComposite(ctx)))
	return &utilityGenerated{kvs: kvs}
}

func (g *gen) withTranslate3d(ctx context.Context, kvs ...KV) *utilityGenerated {
	kvs = append(kvs, g.cssKV(ctx, "translate", g.translate3dComposite(ctx)))
	return &utilityGenerated{kvs: kvs}
}

func (g *gen) withScale2d(ctx context.Context, kvs ...KV) *utilityGenerated {
	kvs = append(kvs, g.cssKV(ctx, "scale", g.scaleComposite(ctx)))
	return &utilityGenerated{kvs: kvs}
}

func (g *gen) withScale3d(ctx context.Context, kvs ...KV) *utilityGenerated {
	kvs = append(kvs, g.cssKV(ctx, "scale", g.scale3dComposite(ctx)))
	return &utilityGenerated{kvs: kvs}
}

func negateTransformCalc(value string, negative bool) string {
	if !negative {
		return value
	}

	return fmt.Sprintf("calc(%s * -1)", value)
}

func (g *gen) transformPlacementValue(
	ctx context.Context, p *parser.NamePart, negative bool, mod *parser.SlashValue,
) (string, error) {
	n := ""
	if negative {
		n = "-"
	}

	switch {
	case p.Int != nil && mod != nil && mod.Int != nil:
		v := fmt.Sprintf("calc(%s / %s * 100%%)", p.String(), mod.String())
		if negative {
			v = fmt.Sprintf("calc(%s * -1)", v)
		}

		return v, nil
	case p.Int != nil:
		s := p.String()
		if !isPositiveInteger(s) {
			return "", errUtilityUnknown
		}

		if v := g.cssVar(ctx, fmt.Sprintf("--translate-%s", s)); v != "" {
			return negateTransformCalc(v, negative), nil
		}

		return fmt.Sprintf("calc(%s * %s%s)", g.cssVar(ctx, "--spacing"), n, s), nil
	case p.Ident == "px":
		return fmt.Sprintf("%s1px", n), nil
	case p.Ident == "full":
		return fmt.Sprintf("%s100%%", n), nil
	case p.Custom != "":
		_, v := p.CustomParts()
		return negateTransformCalc(g.cssVar(ctx, v), negative), nil
	case p.Arbitrary != "":
		_, v := p.ArbitraryParts()
		return negateTransformCalc(arbitraryValue(v), negative), nil
	default:
		return "", errUtilityUnknown
	}
}

func (g *gen) themeScaleValue(_ context.Context, p *parser.NamePart, negative bool) (string, bool) {
	if p.Int == nil {
		return "", false
	}

	s := p.String()
	if !isPositiveInteger(s) {
		return "", false
	}

	v := fmt.Sprintf("%s%%", s)
	return negateTransformCalc(v, negative), true
}

func (g *gen) themeSkewValue(_ context.Context, p *parser.NamePart, negative bool) (string, bool) {
	if p.Int == nil {
		return "", false
	}

	s := p.String()
	if !isPositiveInteger(s) {
		return "", false
	}

	v := fmt.Sprintf("%sdeg", s)

	return negateTransformCalc(v, negative), true
}

func (g *gen) rotateSimpleValue(ctx context.Context, p *parser.NamePart, negative bool) (string, bool) {
	switch {
	case p.Int != nil:
		s := p.String()
		if !isPositiveInteger(s) {
			return "", false
		}

		v := fmt.Sprintf("%sdeg", s)

		if negative {
			return fmt.Sprintf("calc(%s * -1)", v), true
		}

		return v, true
	case p.Ident == "none":
		return "none", true
	case p.Custom != "":
		_, v := p.CustomParts()
		vv := g.cssVar(ctx, v)
		if negative {
			return fmt.Sprintf("calc(%s * -1)", vv), true
		}

		return vv, true
	case p.Arbitrary != "":
		_, v := p.ArbitraryParts()
		vv := arbitraryValue(v)
		if negative {
			if strings.HasSuffix(vv, "deg") && !strings.Contains(vv, " ") {
				return "-" + vv, true
			}

			return fmt.Sprintf("calc(%s * -1)", vv), true
		}

		return vv, true
	default:
		return "", false
	}
}

func rotateAxisFn(axis string) string {
	switch axis {
	case "x":
		return "rotateX"
	case "y":
		return "rotateY"
	case "z":
		return "rotateZ"
	default:
		return ""
	}
}

func (g *gen) rotateAxisValue(ctx context.Context, axis string, p *parser.NamePart, negative bool) (string, bool) {
	fn := rotateAxisFn(axis)
	if fn == "" {
		return "", false
	}

	switch {
	case p.Int != nil:
		s := p.String()
		if !isPositiveInteger(s) {
			return "", false
		}

		v := fmt.Sprintf("%sdeg", s)
		v = negateTransformCalc(v, negative)
		return fmt.Sprintf("%s(%s)", fn, v), true
	case p.Custom != "":
		_, v := p.CustomParts()
		vv := g.cssVar(ctx, v)
		if negative {
			vv = fmt.Sprintf("calc(%s * -1)", vv)
		}

		return fmt.Sprintf("%s(%s)", fn, vv), true
	case p.Arbitrary != "":
		_, v := p.ArbitraryParts()
		vv := arbitraryValue(v)
		if negative {
			if strings.HasSuffix(vv, "deg") && !strings.Contains(vv, " ") {
				vv = "-" + vv
			} else {
				vv = fmt.Sprintf("calc(%s * -1)", vv)
			}
		}

		return fmt.Sprintf("%s(%s)", fn, vv), true
	default:
		return "", false
	}
}

func (g *gen) utilityTranslate(
	ctx context.Context, parts []*parser.NamePart, negative bool, mod *parser.SlashValue,
) (*utilityGenerated, error) {
	if len(parts) == 0 {
		return nil, errUtilityUnknown
	}

	switch len(parts) {
	case 1:
		p := parts[0]

		switch p.String() {
		case "none":
			return newUtilityGeneratedKV("translate", "none"), nil
		case "full":
			n := ""
			if negative {
				n = "-"
			}

			return g.withTranslate2d(ctx,
				g.cssDefine(ctx, "--tw-translate-x", fmt.Sprintf("%s100%%", n)),
				g.cssDefine(ctx, "--tw-translate-y", fmt.Sprintf("%s100%%", n)),
			), nil
		case "3d":
			return g.withTranslate3d(ctx), nil
		}

		v, err := g.transformPlacementValue(ctx, p, negative, mod)
		if err != nil {
			return nil, err
		}

		return g.withTranslate2d(ctx,
			g.cssDefine(ctx, "--tw-translate-x", v),
			g.cssDefine(ctx, "--tw-translate-y", v),
		), nil

	case 2:
		axis := parts[0].Ident
		p := parts[1]

		switch axis {
		case "x", "y":
			if p.String() == "full" {
				if mod != nil {
					return nil, errUtilityUnknown
				}

				n := ""
				if negative {
					n = "-"
				}

				return g.withTranslate2d(ctx, g.cssDefine(ctx, fmt.Sprintf("--tw-translate-%s", axis), fmt.Sprintf("%s100%%", n))), nil
			}

			v, err := g.transformPlacementValue(ctx, p, negative, mod)
			if err != nil {
				return nil, err
			}

			return g.withTranslate2d(ctx, g.cssDefine(ctx, fmt.Sprintf("--tw-translate-%s", axis), v)), nil
		case "z":
			if mod != nil {
				return nil, errUtilityUnknown
			}

			if p.String() == "full" {
				return nil, errUtilityUnknown
			}

			v, err := g.transformPlacementValue(ctx, p, negative, nil)
			if err != nil {
				return nil, err
			}

			return g.withTranslate3d(ctx, g.cssDefine(ctx, "--tw-translate-z", v)), nil
		}
	}

	return nil, errUtilityUnknown
}

func (g *gen) utilityScale(
	ctx context.Context, parts []*parser.NamePart, negative bool, mod *parser.SlashValue,
) (*utilityGenerated, error) {
	if mod != nil || len(parts) == 0 {
		return nil, errUtilityUnknown
	}

	switch len(parts) {
	case 1:
		p := parts[0]

		switch p.String() {
		case "none":
			return newUtilityGeneratedKV("scale", "none"), nil
		case "3d":
			return g.withScale3d(ctx), nil
		}

		if p.Arbitrary != "" {
			_, v := p.ArbitraryParts()
			vv := arbitraryValue(v)
			if negative {
				vv = fmt.Sprintf("calc(%s * -1)", vv)
			}

			return newUtilityGeneratedKV("scale", vv), nil
		}

		if p.Custom != "" {
			_, v := p.CustomParts()
			vv := negateTransformCalc(g.cssVar(ctx, v), negative)

			return g.withScale2d(ctx,
				g.cssDefine(ctx, "--tw-scale-x", vv),
				g.cssDefine(ctx, "--tw-scale-y", vv),
				g.cssDefine(ctx, "--tw-scale-z", vv),
			), nil
		}

		if v, ok := g.themeScaleValue(ctx, p, negative); ok {
			return g.withScale2d(ctx,
				g.cssDefine(ctx, "--tw-scale-x", v),
				g.cssDefine(ctx, "--tw-scale-y", v),
				g.cssDefine(ctx, "--tw-scale-z", v),
			), nil
		}

	case 2:
		axis := parts[0].Ident
		p := parts[1]

		if p.Arbitrary != "" {
			_, v := p.ArbitraryParts()
			vv := arbitraryValue(v)
			if negative {
				vv = fmt.Sprintf("calc(%s * -1)", vv)
			}

			switch axis {
			case "x", "y":
				return g.withScale2d(ctx, g.cssDefine(ctx, fmt.Sprintf("--tw-scale-%s", axis), vv)), nil
			case "z":
				return g.withScale3d(ctx, g.cssDefine(ctx, "--tw-scale-z", vv)), nil
			}

			return nil, errUtilityUnknown
		}

		if p.Custom != "" {
			_, v := p.CustomParts()
			vv := negateTransformCalc(g.cssVar(ctx, v), negative)

			switch axis {
			case "x", "y":
				return g.withScale2d(ctx, g.cssDefine(ctx, fmt.Sprintf("--tw-scale-%s", axis), vv)), nil
			case "z":
				return g.withScale3d(ctx, g.cssDefine(ctx, "--tw-scale-z", vv)), nil
			}

			return nil, errUtilityUnknown
		}

		if v, ok := g.themeScaleValue(ctx, p, negative); ok {
			switch axis {
			case "x", "y":
				return g.withScale2d(ctx, g.cssDefine(ctx, fmt.Sprintf("--tw-scale-%s", axis), v)), nil
			case "z":
				return g.withScale3d(ctx, g.cssDefine(ctx, "--tw-scale-z", v)), nil
			}
		}
	}

	return nil, errUtilityUnknown
}

func (g *gen) utilityRotate(
	ctx context.Context, parts []*parser.NamePart, negative bool, mod *parser.SlashValue,
) (*utilityGenerated, error) {
	if mod != nil {
		return nil, errUtilityUnknown
	}

	switch len(parts) {
	case 0:
		return nil, errUtilityUnknown
	case 1:
		if v, ok := g.rotateSimpleValue(ctx, parts[0], negative); ok {
			return newUtilityGeneratedKV("rotate", v), nil
		}
	case 2:
		axis := parts[0].Ident
		if axis != "x" && axis != "y" && axis != "z" {
			return nil, errUtilityUnknown
		}

		if v, ok := g.rotateAxisValue(ctx, axis, parts[1], negative); ok {
			return g.withTransform(ctx, g.cssDefine(ctx, fmt.Sprintf("--tw-rotate-%s", axis), v)), nil
		}
	}

	return nil, errUtilityUnknown
}

func (g *gen) utilitySkew(
	ctx context.Context, parts []*parser.NamePart, negative bool, mod *parser.SlashValue,
) (*utilityGenerated, error) {
	if mod != nil {
		return nil, errUtilityUnknown
	}

	switch len(parts) {
	case 0:
		return nil, errUtilityUnknown
	case 1:
		p := parts[0]

		if p.Arbitrary != "" {
			_, v := p.ArbitraryParts()
			vv := arbitraryValue(v)
			if negative {
				vv = fmt.Sprintf("calc(%s * -1)", vv)
			}

			return g.withTransform(ctx,
				g.cssDefine(ctx, "--tw-skew-x", fmt.Sprintf("skewX(%s)", vv)),
				g.cssDefine(ctx, "--tw-skew-y", fmt.Sprintf("skewY(%s)", vv)),
			), nil
		}

		if p.Custom != "" {
			_, v := p.CustomParts()
			vv := negateTransformCalc(g.cssVar(ctx, v), negative)

			return g.withTransform(ctx,
				g.cssDefine(ctx, "--tw-skew-x", fmt.Sprintf("skewX(%s)", vv)),
				g.cssDefine(ctx, "--tw-skew-y", fmt.Sprintf("skewY(%s)", vv)),
			), nil
		}

		if v, ok := g.themeSkewValue(ctx, p, negative); ok {
			return g.withTransform(ctx,
				g.cssDefine(ctx, "--tw-skew-x", fmt.Sprintf("skewX(%s)", v)),
				g.cssDefine(ctx, "--tw-skew-y", fmt.Sprintf("skewY(%s)", v)),
			), nil
		}
	case 2:
		axis := parts[0].Ident
		p := parts[1]

		if axis != "x" && axis != "y" {
			return nil, errUtilityUnknown
		}

		var v string
		var ok bool

		if p.Arbitrary != "" {
			_, av := p.ArbitraryParts()
			v = arbitraryValue(av)
			if negative {
				v = fmt.Sprintf("calc(%s * -1)", v)
			}

			ok = true
		} else if p.Custom != "" {
			_, cv := p.CustomParts()
			v = negateTransformCalc(g.cssVar(ctx, cv), negative)
			ok = true
		} else {
			v, ok = g.themeSkewValue(ctx, p, negative)
		}

		if !ok {
			return nil, errUtilityUnknown
		}

		fn := "skewX"
		if axis == "y" {
			fn = "skewY"
		}

		return g.withTransform(ctx, g.cssDefine(ctx, fmt.Sprintf("--tw-skew-%s", axis), fmt.Sprintf("%s(%s)", fn, v))), nil
	}

	return nil, errUtilityUnknown
}

func (g *gen) utilityTransform(
	ctx context.Context, parts []*parser.NamePart, mod *parser.SlashValue,
) (*utilityGenerated, error) {
	if mod != nil {
		return nil, errUtilityUnknown
	}

	switch len(parts) {
	case 0:
		return g.withTransform(ctx), nil
	case 1:
		p := parts[0]

		switch p.String() {
		case "cpu":
			return g.withTransform(ctx), nil
		case "gpu":
			return &utilityGenerated{
				kvs: []KV{g.cssKV(ctx, "transform", fmt.Sprintf("translateZ(0) %s", g.transformComposite(ctx)))},
			}, nil
		case "none":
			return newUtilityGeneratedKV("transform", "none"), nil
		case "flat":
			return newUtilityGeneratedKV("transform-style", "flat"), nil
		case "3d":
			return newUtilityGeneratedKV("transform-style", "preserve-3d"), nil
		case "content":
			return newUtilityGeneratedKV("transform-box", "content-box"), nil
		case "border":
			return newUtilityGeneratedKV("transform-box", "border-box"), nil
		case "fill":
			return newUtilityGeneratedKV("transform-box", "fill-box"), nil
		case "stroke":
			return newUtilityGeneratedKV("transform-box", "stroke-box"), nil
		case "view":
			return newUtilityGeneratedKV("transform-box", "view-box"), nil
		}

		if p.Arbitrary != "" {
			return g.cssArbitrary(ctx, "transform", p), nil
		}

		if p.Custom != "" {
			return g.cssCustom(ctx, "transform", p), nil
		}
	}

	return nil, errUtilityUnknown
}

func (g *gen) utilityOriginLike(
	ctx context.Context, prop, themePrefix string, parts []*parser.NamePart, mod *parser.SlashValue,
) (*utilityGenerated, error) {
	if mod != nil || len(parts) == 0 {
		return nil, errUtilityUnknown
	}

	if len(parts) == 1 {
		p := parts[0]

		switch {
		case p.Custom != "":
			return g.cssCustom(ctx, prop, p), nil
		case p.Arbitrary != "":
			return g.cssArbitrary(ctx, prop, p), nil
		}
	}

	key := partsJoin(parts)
	if v := g.cssVar(ctx, fmt.Sprintf("%s-%s", themePrefix, key)); v != "" {
		return newUtilityGeneratedKV(prop, v), nil
	}

	if v, ok := transformOriginStatic[key]; ok {
		return newUtilityGeneratedKV(prop, v), nil
	}

	return nil, errUtilityUnknown
}

func (g *gen) utilityOrigin(
	ctx context.Context, parts []*parser.NamePart, mod *parser.SlashValue,
) (*utilityGenerated, error) {
	return g.utilityOriginLike(ctx, "transform-origin", "--transform-origin", parts, mod)
}

func (g *gen) utilityPerspective(
	ctx context.Context, parts []*parser.NamePart, mod *parser.SlashValue,
) (*utilityGenerated, error) {
	if mod != nil || len(parts) == 0 {
		return nil, errUtilityUnknown
	}

	if parts[0].Ident == "origin" {
		return g.utilityOriginLike(ctx, "perspective-origin", "--perspective-origin", parts[1:], mod)
	}

	if len(parts) == 1 && parts[0].Ident == "none" {
		return newUtilityGeneratedKV("perspective", "none"), nil
	}

	if len(parts) == 1 {
		p := parts[0]

		switch {
		case p.Custom != "":
			return g.cssCustom(ctx, "perspective", p), nil
		case p.Arbitrary != "":
			return g.cssArbitrary(ctx, "perspective", p), nil
		}
	}

	key := partsJoin(parts)
	if v := g.cssVar(ctx, fmt.Sprintf("--perspective-%s", key)); v != "" {
		return newUtilityGeneratedKV("perspective", v), nil
	}

	return nil, errUtilityUnknown
}

func (g *gen) utilityBackface(parts []*parser.NamePart, mod *parser.SlashValue) (*utilityGenerated, error) {
	if mod != nil || len(parts) != 1 {
		return nil, errUtilityUnknown
	}

	switch parts[0].Ident {
	case "visible":
		return newUtilityGeneratedKV("backface-visibility", "visible"), nil
	case "hidden":
		return newUtilityGeneratedKV("backface-visibility", "hidden"), nil
	}

	return nil, errUtilityUnknown
}

func (g *gen) utilityZoom(ctx context.Context, parts []*parser.NamePart, mod *parser.SlashValue) (*utilityGenerated, error) {
	if mod != nil || len(parts) != 1 {
		return nil, errUtilityUnknown
	}

	p := parts[0]

	switch {
	case p.Int != nil:
		if v, ok := filterPercentageValue(p); ok {
			return newUtilityGeneratedKV("zoom", v), nil
		}
	case p.Custom != "":
		return g.cssCustom(ctx, "zoom", p), nil
	case p.Arbitrary != "":
		return g.cssArbitrary(ctx, "zoom", p), nil
	}

	return nil, errUtilityUnknown
}

func (g *gen) userSelectKVs(ctx context.Context, value string) []KV {
	return []KV{
		g.cssKV(ctx, "-webkit-user-select", value),
		g.cssKV(ctx, "user-select", value),
	}
}

func (g *gen) withTouchAction(ctx context.Context, kvs ...KV) *utilityGenerated {
	kvs = append(kvs, g.cssKV(ctx, "touch-action", g.touchActionComposite(ctx)))
	return &utilityGenerated{kvs: kvs}
}

func (g *gen) withScrollbarColor(ctx context.Context, kvs ...KV) *utilityGenerated {
	kvs = append(kvs, g.cssKV(ctx, "scrollbar-color", g.scrollbarColorComposite(ctx)))
	return &utilityGenerated{kvs: kvs}
}

func (g *gen) utilityColorProperty(
	ctx context.Context, prop string, parts []*parser.NamePart, mod *parser.SlashValue, allowAuto bool,
) (*utilityGenerated, error) {
	if allowAuto && len(parts) == 1 && parts[0].Ident == "auto" {
		return newUtilityGeneratedKV(prop, "auto"), nil
	}

	if len(parts) == 1 && parts[0].Custom != "" {
		return g.cssCustom(ctx, prop, parts[0]), nil
	}

	if len(parts) == 1 && parts[0].Arbitrary != "" {
		return g.cssArbitrary(ctx, prop, parts[0]), nil
	}

	return g.cssColor(ctx, prop, parts, mod)
}

func isBareNumeric(s string) bool {
	_, err := strconv.ParseFloat(s, 64)
	return err == nil
}

func (g *gen) utilityStrokeWidth(
	ctx context.Context, parts []*parser.NamePart, mod *parser.SlashValue,
) (*utilityGenerated, error) {
	if mod != nil || len(parts) != 1 {
		return nil, errUtilityUnknown
	}

	p := parts[0]

	switch {
	case p.Int != nil:
		s := p.String()
		if v := g.cssVar(ctx, fmt.Sprintf("--stroke-width-%s", s)); v != "" {
			return newUtilityGeneratedKV("stroke-width", v), nil
		}

		if *p.Int == 0 {
			return newUtilityGeneratedKV("stroke-width", "0"), nil
		}

		if !isPositiveInteger(s) {
			return nil, errUtilityUnknown
		}

		return newUtilityGeneratedKV("stroke-width", s), nil
	case p.Custom != "":
		k, _ := p.CustomParts()
		switch k {
		case "number", "length", "percentage":
			return g.cssCustom(ctx, "stroke-width", p), nil
		default:
			return nil, errUtilityUnknown
		}
	case p.Arbitrary != "":
		return g.strokeWidthFromArbitrary(ctx, p)
	default:
		return nil, errUtilityUnknown
	}
}

func (g *gen) strokeWidthFromArbitrary(ctx context.Context, p *parser.NamePart) (*utilityGenerated, error) {
	k, v := p.ArbitraryParts()
	av := arbitraryValue(v)

	switch k {
	case "number", "length", "percentage":
		return g.cssArbitrary(ctx, "stroke-width", p), nil
	}

	if cssIsLength(av) || strings.HasSuffix(av, "%") {
		return g.cssArbitrary(ctx, "stroke-width", p), nil
	}

	if isBareNumeric(av) {
		return newUtilityGeneratedKV("stroke-width", av+"px"), nil
	}

	return nil, errUtilityUnknown
}

func (g *gen) strokeFromArbitrary(
	ctx context.Context, p *parser.NamePart, mod *parser.SlashValue,
) (*utilityGenerated, error) {
	if mod != nil {
		return nil, errUtilityUnknown
	}

	k, v := p.ArbitraryParts()
	av := arbitraryValue(v)

	switch k {
	case "number", "length", "percentage":
		return g.strokeWidthFromArbitrary(ctx, p)
	case "color":
		return g.cssArbitrary(ctx, "stroke", p), nil
	}

	if cssIsLength(av) || strings.HasSuffix(av, "%") || isBareNumeric(av) {
		return g.strokeWidthFromArbitrary(ctx, p)
	}

	return g.cssArbitrary(ctx, "stroke", p), nil
}

func (g *gen) utilityFill(
	ctx context.Context, parts []*parser.NamePart, mod *parser.SlashValue,
) (*utilityGenerated, error) {
	if len(parts) == 1 && parts[0].Ident == "none" {
		return newUtilityGeneratedKV("fill", "none"), nil
	}

	return g.utilityColorProperty(ctx, "fill", parts, mod, false)
}

func (g *gen) utilityStroke(
	ctx context.Context, parts []*parser.NamePart, mod *parser.SlashValue,
) (*utilityGenerated, error) {
	if len(parts) == 0 {
		return nil, errUtilityUnknown
	}

	if parts[0].Ident == "width" {
		return g.utilityStrokeWidth(ctx, parts[1:], mod)
	}

	if len(parts) == 1 && parts[0].Ident == "none" {
		return newUtilityGeneratedKV("stroke", "none"), nil
	}

	if len(parts) == 1 && parts[0].Int != nil {
		return g.utilityStrokeWidth(ctx, parts, mod)
	}

	if len(parts) == 1 && parts[0].Arbitrary != "" {
		return g.strokeFromArbitrary(ctx, parts[0], mod)
	}

	if len(parts) == 1 && parts[0].Custom != "" {
		k, _ := parts[0].CustomParts()
		switch k {
		case "number", "length", "percentage":
			return g.cssCustom(ctx, "stroke-width", parts[0]), nil
		default:
			return g.cssCustom(ctx, "stroke", parts[0]), nil
		}
	}

	return g.utilityColorProperty(ctx, "stroke", parts, mod, false)
}

func (g *gen) utilityPointerEvents(parts []*parser.NamePart, mod *parser.SlashValue) (*utilityGenerated, error) {
	if mod != nil || !simpleMatch(parts, "events", "none") && !simpleMatch(parts, "events", "auto") {
		return nil, errUtilityUnknown
	}

	return newUtilityGeneratedKV("pointer-events", parts[1].Ident), nil
}

func (g *gen) utilityCursor(
	ctx context.Context, parts []*parser.NamePart, mod *parser.SlashValue,
) (*utilityGenerated, error) {
	if mod != nil || len(parts) == 0 {
		return nil, errUtilityUnknown
	}

	if len(parts) == 1 {
		p := parts[0]

		switch {
		case p.Custom != "":
			return g.cssCustom(ctx, "cursor", p), nil
		case p.Arbitrary != "":
			return g.cssArbitrary(ctx, "cursor", p), nil
		}
	}

	key := partsJoin(parts)
	if v, ok := cursorStaticValues[key]; ok {
		return newUtilityGeneratedKV("cursor", v), nil
	}

	if v := g.cssVar(ctx, fmt.Sprintf("--cursor-%s", key)); v != "" {
		return newUtilityGeneratedKV("cursor", v), nil
	}

	return nil, errUtilityUnknown
}

func (g *gen) utilityTouchAction(ctx context.Context, parts []*parser.NamePart, mod *parser.SlashValue) (*utilityGenerated, error) {
	if mod != nil || len(parts) == 0 {
		return nil, errUtilityUnknown
	}

	switch {
	case len(parts) == 1 && slices.Contains([]string{"auto", "none", "manipulation"}, parts[0].Ident):
		return newUtilityGeneratedKV("touch-action", parts[0].Ident), nil
	case simpleMatch(parts, "pan", "x"):
		return g.withTouchAction(ctx, g.cssDefine(ctx, "--tw-pan-x", "pan-x")), nil
	case simpleMatch(parts, "pan", "left"):
		return g.withTouchAction(ctx, g.cssDefine(ctx, "--tw-pan-x", "pan-left")), nil
	case simpleMatch(parts, "pan", "right"):
		return g.withTouchAction(ctx, g.cssDefine(ctx, "--tw-pan-x", "pan-right")), nil
	case simpleMatch(parts, "pan", "y"):
		return g.withTouchAction(ctx, g.cssDefine(ctx, "--tw-pan-y", "pan-y")), nil
	case simpleMatch(parts, "pan", "up"):
		return g.withTouchAction(ctx, g.cssDefine(ctx, "--tw-pan-y", "pan-up")), nil
	case simpleMatch(parts, "pan", "down"):
		return g.withTouchAction(ctx, g.cssDefine(ctx, "--tw-pan-y", "pan-down")), nil
	case simpleMatch(parts, "pinch", "zoom"):
		return g.withTouchAction(ctx, g.cssDefine(ctx, "--tw-pinch-zoom", "pinch-zoom")), nil
	}

	return nil, errUtilityUnknown
}

func (g *gen) utilityUserSelect(ctx context.Context, parts []*parser.NamePart, mod *parser.SlashValue) (*utilityGenerated, error) {
	if mod != nil || len(parts) != 1 {
		return nil, errUtilityUnknown
	}

	switch parts[0].Ident {
	case "none", "text", "all", "auto":
		return &utilityGenerated{kvs: g.userSelectKVs(ctx, parts[0].Ident)}, nil
	}

	return nil, errUtilityUnknown
}

func (g *gen) utilityResize(parts []*parser.NamePart, mod *parser.SlashValue) (*utilityGenerated, error) {
	if mod != nil {
		return nil, errUtilityUnknown
	}

	switch len(parts) {
	case 0:
		return newUtilityGeneratedKV("resize", "both"), nil
	case 1:
		switch parts[0].Ident {
		case "none":
			return newUtilityGeneratedKV("resize", "none"), nil
		case "x":
			return newUtilityGeneratedKV("resize", "horizontal"), nil
		case "y":
			return newUtilityGeneratedKV("resize", "vertical"), nil
		}
	}

	return nil, errUtilityUnknown
}

func (g *gen) utilitySnap(ctx context.Context, parts []*parser.NamePart, mod *parser.SlashValue) (*utilityGenerated, error) {
	if mod != nil || len(parts) == 0 {
		return nil, errUtilityUnknown
	}

	switch {
	case len(parts) == 1 && parts[0].Ident == "none":
		return newUtilityGeneratedKV("scroll-snap-type", "none"), nil
	case len(parts) == 1 && slices.Contains([]string{"x", "y", "both"}, parts[0].Ident):
		return newUtilityGeneratedKV(
			"scroll-snap-type",
			fmt.Sprintf("%s %s", parts[0].Ident, g.cssRef(ctx, "--tw-scroll-snap-strictness")),
		), nil
	case len(parts) == 1 && parts[0].Ident == "mandatory":
		return newUtilityGeneratedKV("--tw-scroll-snap-strictness", "mandatory"), nil
	case len(parts) == 1 && parts[0].Ident == "proximity":
		return newUtilityGeneratedKV("--tw-scroll-snap-strictness", "proximity"), nil
	case simpleMatch(parts, "align", "none"):
		return newUtilityGeneratedKV("scroll-snap-align", "none"), nil
	case len(parts) == 1 && slices.Contains([]string{"start", "end", "center"}, parts[0].Ident):
		return newUtilityGeneratedKV("scroll-snap-align", parts[0].Ident), nil
	case len(parts) == 1 && parts[0].Ident == "normal":
		return newUtilityGeneratedKV("scroll-snap-stop", "normal"), nil
	case len(parts) == 1 && parts[0].Ident == "always":
		return newUtilityGeneratedKV("scroll-snap-stop", "always"), nil
	}

	return nil, errUtilityUnknown
}

func (g *gen) utilityScroll(
	ctx context.Context, parts []*parser.NamePart, negative bool, mod *parser.SlashValue,
) (*utilityGenerated, error) {
	if mod != nil || len(parts) == 0 {
		return nil, errUtilityUnknown
	}

	switch {
	case len(parts) == 1 && parts[0].Ident == "auto" && !negative:
		return newUtilityGeneratedKV("scroll-behavior", "auto"), nil
	case len(parts) == 1 && parts[0].Ident == "smooth" && !negative:
		return newUtilityGeneratedKV("scroll-behavior", "smooth"), nil
	case len(parts) == 2:
		prop, ok := scrollSpacingProperties[parts[0].String()]
		if !ok {
			return nil, errUtilityUnknown
		}

		margin := strings.HasPrefix(parts[0].String(), "m")
		if negative && !margin {
			return nil, errUtilityUnknown
		}

		p := parts[1]
		switch {
		case p.Int != nil || p.Float != nil:
			return g.cssCalcSpacing(ctx, prop, parts[1], negative), nil
		case p.Custom != "":
			return g.cssCustom(ctx, prop, p), nil
		case p.Arbitrary != "":
			return g.cssArbitrary(ctx, prop, p), nil
		}

	}

	return nil, errUtilityUnknown
}

func (g *gen) utilityScrollbar(
	ctx context.Context, parts []*parser.NamePart, mod *parser.SlashValue,
) (*utilityGenerated, error) {
	if mod != nil || len(parts) == 0 {
		return nil, errUtilityUnknown
	}

	switch {
	case len(parts) == 1 && slices.Contains([]string{"auto", "thin", "none"}, parts[0].Ident):
		return newUtilityGeneratedKV("scrollbar-width", parts[0].Ident), nil
	case simpleMatch(parts, "gutter", "auto"):
		return newUtilityGeneratedKV("scrollbar-gutter", "auto"), nil
	case simpleMatch(parts, "gutter", "stable"):
		return newUtilityGeneratedKV("scrollbar-gutter", "stable"), nil
	case simpleMatch(parts, "gutter", "both"):
		return newUtilityGeneratedKV("scrollbar-gutter", "stable both-edges"), nil
	case parts[0].Ident == "thumb" && len(parts) > 1:
		u, err := g.utilityColorProperty(ctx, "--tw-scrollbar-thumb", parts[1:], mod, false)
		if err != nil {
			return nil, err
		}

		return g.withScrollbarColor(ctx, u.kvs...), nil
	case parts[0].Ident == "track" && len(parts) > 1:
		u, err := g.utilityColorProperty(ctx, "--tw-scrollbar-track", parts[1:], mod, false)
		if err != nil {
			return nil, err
		}

		return g.withScrollbarColor(ctx, u.kvs...), nil
	}

	return nil, errUtilityUnknown
}

func (g *gen) utilityAppearance(parts []*parser.NamePart, mod *parser.SlashValue) (*utilityGenerated, error) {
	if mod != nil || len(parts) != 1 {
		return nil, errUtilityUnknown
	}

	switch parts[0].Ident {
	case "none", "auto":
		return newUtilityGeneratedKV("appearance", parts[0].Ident), nil
	}

	return nil, errUtilityUnknown
}

func (g *gen) utilityAtContainer(ctx context.Context, parts []*parser.NamePart, mod *parser.SlashValue) (*utilityGenerated, error) {
	if mod != nil && mod.Arbitrary != "" {
		return nil, errUtilityUnknown
	}

	value := "inline-size"
	switch len(parts) {
	case 0:
	case 1:
		switch {
		case parts[0].Ident == "normal":
			value = "normal"
		case parts[0].Ident == "size":
			value = "size"
		case parts[0].Arbitrary != "":
			value = decodeArbitraryVariant(strings.TrimPrefix(strings.TrimSuffix(parts[0].Arbitrary, "]"), "["))
		default:
			return nil, errUtilityUnknown
		}
	default:
		return nil, errUtilityUnknown
	}

	if mod != nil && mod.Ident != "" {
		return &utilityGenerated{
			kvs: []KV{
				g.cssKV(ctx, "container-type", value),
				g.cssKV(ctx, "container-name", mod.Ident),
			},
		}, nil
	}

	return newUtilityGeneratedKV("container-type", value), nil
}

func (g *gen) utilityColorScheme(parts []*parser.NamePart, mod *parser.SlashValue) (*utilityGenerated, error) {
	if mod != nil || len(parts) == 0 {
		return nil, errUtilityUnknown
	}

	if v, ok := colorSchemeStatic[partsJoin(parts)]; ok {
		return newUtilityGeneratedKV("color-scheme", v), nil
	}

	return nil, errUtilityUnknown
}

func (g *gen) utilityAccentColor(
	ctx context.Context, parts []*parser.NamePart, mod *parser.SlashValue,
) (*utilityGenerated, error) {
	return g.utilityColorProperty(ctx, "accent-color", parts, mod, true)
}

func (g *gen) utilityCaretColor(
	ctx context.Context, parts []*parser.NamePart, mod *parser.SlashValue,
) (*utilityGenerated, error) {
	return g.utilityColorProperty(ctx, "caret-color", parts, mod, false)
}

func (g *gen) utilityFieldSizing(parts []*parser.NamePart, mod *parser.SlashValue) (*utilityGenerated, error) {
	if mod != nil || !simpleMatch(parts, "sizing", "content") && !simpleMatch(parts, "sizing", "fixed") {
		return nil, errUtilityUnknown
	}

	return newUtilityGeneratedKV("field-sizing", parts[1].Ident), nil
}

func (g *gen) utilityForcedColorAdjust(parts []*parser.NamePart, mod *parser.SlashValue) (*utilityGenerated, error) {
	if mod != nil || !simpleMatch(parts, "color", "adjust", "none") && !simpleMatch(parts, "color", "adjust", "auto") {
		return nil, errUtilityUnknown
	}

	return newUtilityGeneratedKV("forced-color-adjust", parts[2].Ident), nil
}

func (g *gen) utilityWillChange(
	ctx context.Context, parts []*parser.NamePart, mod *parser.SlashValue,
) (*utilityGenerated, error) {
	if mod != nil || len(parts) < 2 || parts[0].Ident != "change" {
		return nil, errUtilityUnknown
	}

	p := parts[1]

	switch p.Ident {
	case "auto":
		return newUtilityGeneratedKV("will-change", "auto"), nil
	case "scroll":
		return newUtilityGeneratedKV("will-change", "scroll-position"), nil
	case "contents":
		return newUtilityGeneratedKV("will-change", "contents"), nil
	case "transform":
		return newUtilityGeneratedKV("will-change", "transform"), nil
	}

	switch {
	case p.Custom != "":
		return g.cssCustom(ctx, "will-change", p), nil
	case p.Arbitrary != "":
		return g.cssArbitrary(ctx, "will-change", p), nil
	}

	return nil, errUtilityUnknown
}

func (g *gen) cssCalcSpacingKV(ctx context.Context, k string, num fmt.Stringer, negative bool) KV {
	n := ""
	if negative {
		n = "-"
	}

	return g.cssKVf(ctx, k, "calc(%s * %s%s)", g.cssVar(ctx, "--spacing"), n, num.String())
}

func (g *gen) cssCalcSpacing(ctx context.Context, k string, num fmt.Stringer, negative bool) *utilityGenerated {
	return &utilityGenerated{kvs: []KV{g.cssCalcSpacingKV(ctx, k, num, negative)}}
}

func (g *gen) cssCustomKV(ctx context.Context, k string, p *parser.NamePart) KV {
	_, v := p.CustomParts()
	return g.cssKV(ctx, k, g.cssVar(ctx, v))
}

func (g *gen) cssCustomDefine(ctx context.Context, k string, p *parser.NamePart) KV {
	_, v := p.CustomParts()
	return g.cssDefine(ctx, k, g.cssVar(ctx, v))
}

func (g *gen) cssCustom(ctx context.Context, k string, p *parser.NamePart) *utilityGenerated {
	return &utilityGenerated{kvs: []KV{g.cssCustomKV(ctx, k, p)}}
}

func (g *gen) cssCustomContainerSize(ctx context.Context, k, size string) *utilityGenerated {
	return newUtilityGeneratedKV(k, g.cssVar(ctx, fmt.Sprintf("--container-%s", size)))
}

func (g *gen) cssColorKV(
	ctx context.Context, k string, parts []*parser.NamePart, mod *parser.SlashValue,
) (KV, error) {
	ckey := partsJoin(parts)
	v := g.cssVar(ctx, fmt.Sprintf("--color-%s", ckey))

	switch {
	case len(parts) == 1 && slices.Contains([]string{"inherit", "transparent"}, ckey):
		return g.cssKV(ctx, k, ckey), nil
	case len(parts) == 1 && ckey == "current":
		return g.cssKV(ctx, k, "currentcolor"), nil
	case v != "" && mod != nil && mod.Int != nil:
		return g.cssKVf(ctx, k, "color-mix(in oklab, %s %s%%, transparent)", v, mod.String()), nil
	case v != "":
		return g.cssKV(ctx, k, v), nil
	default:
		return KV{}, errUtilityUnknown
	}
}

func (g *gen) cssColorDefine(
	ctx context.Context, k string, parts []*parser.NamePart, mod *parser.SlashValue,
) (KV, error) {
	ckey := partsJoin(parts)
	v := g.cssVar(ctx, fmt.Sprintf("--color-%s", ckey))

	switch {
	case len(parts) == 1 && slices.Contains([]string{"inherit", "transparent"}, ckey):
		return g.cssDefine(ctx, k, ckey), nil
	case len(parts) == 1 && ckey == "current":
		return g.cssDefine(ctx, k, "currentcolor"), nil
	case v != "" && mod != nil && mod.Int != nil:
		return g.cssDefinef(ctx, k, "color-mix(in oklab, %s %s%%, transparent)", v, mod.String()), nil
	case v != "":
		return g.cssDefine(ctx, k, v), nil
	default:
		return KV{}, errUtilityUnknown
	}
}

func (g *gen) cssColor(
	ctx context.Context, k string, parts []*parser.NamePart, mod *parser.SlashValue,
) (*utilityGenerated, error) {
	kv, err := g.cssColorKV(ctx, k, parts, mod)
	if err != nil {
		return nil, err
	}

	return &utilityGenerated{kvs: []KV{kv}}, nil
}

func (g *gen) gradientLinear(ctx context.Context, position string) *utilityGenerated {
	return &utilityGenerated{
		kvs: []KV{
			g.cssDefine(ctx, "--tw-gradient-position", position),
			g.cssKV(ctx, "background-image", g.linearGradientImage(ctx)),
		},
		others: []Query{
			{
				Name: "@supports (background-image: linear-gradient(in lab, red, red))",
				KVs: []KV{
					g.cssDefine(ctx, "--tw-gradient-position", position+" in oklab"),
				},
			},
		},
	}
}

func (g *gen) gradientLinearFallback(ctx context.Context, value string) *utilityGenerated {
	return &utilityGenerated{
		kvs: []KV{
			g.cssDefine(ctx, "--tw-gradient-position", value),
			g.cssKV(ctx, "background-image", g.linearGradientImageFallback(ctx, value)),
		},
	}
}

func (g *gen) gradientRadialFallback(ctx context.Context, value string) *utilityGenerated {
	return &utilityGenerated{
		kvs: []KV{
			g.cssDefine(ctx, "--tw-gradient-position", value),
			g.cssKV(ctx, "background-image", g.radialGradientImageFallback(ctx, value)),
		},
	}
}

func (g *gen) gradientConic(ctx context.Context, value string) *utilityGenerated {
	return &utilityGenerated{
		kvs: []KV{
			g.cssDefine(ctx, "--tw-gradient-position", value),
			g.cssKV(ctx, "background-image", g.conicGradientImage(ctx)),
		},
	}
}

func (g *gen) gradientConicFallback(ctx context.Context, value string) *utilityGenerated {
	return &utilityGenerated{
		kvs: []KV{
			g.cssDefine(ctx, "--tw-gradient-position", value),
			g.cssKV(ctx, "background-image", g.conicGradientImageFallback(ctx, value)),
		},
	}
}

func (g *gen) cssArbitraryKV(ctx context.Context, k string, p *parser.NamePart) KV {
	_, v := p.ArbitraryParts()
	return g.cssKV(ctx, k, g.cssArbitraryValue(ctx, arbitraryValue(v)))
}

func (g *gen) cssArbitraryDefine(ctx context.Context, k string, p *parser.NamePart) KV {
	_, v := p.ArbitraryParts()
	return g.cssDefine(ctx, k, g.cssArbitraryValue(ctx, arbitraryValue(v)))
}

func (g *gen) cssArbitrary(ctx context.Context, k string, p *parser.NamePart) *utilityGenerated {
	return &utilityGenerated{kvs: []KV{g.cssArbitraryKV(ctx, k, p)}}
}

func simpleMatch(parts []*parser.NamePart, want ...string) bool {
	if len(parts) != len(want) {
		return false
	}

	return slices.Equal(partsStrings(parts), want)
}

func partsStrings(parts []*parser.NamePart) (s []string) {
	for _, p := range parts {
		s = append(s, p.String())
	}

	return
}

func partsJoin(parts []*parser.NamePart) string {
	return strings.Join(partsStrings(parts), "-")
}

func customValue(c string) string {
	return strings.ReplaceAll(c, "_", " ")
}

func arbitraryValue(a string) string {
	// very naive approach, probably it's going to be verified later
	a = strings.ReplaceAll(a, "_", " ")
	a = strings.ReplaceAll(a, ",", ", ")

	return strings.Trim(a, "[]")
}

func isContainerSize(size string) bool {
	switch size {
	case "3xs", "2xs", "xs", "sm", "md", "lg", "xl", "2xl", "3xl", "4xl", "5xl", "6xl", "7xl":
		return true
	default:
		return false
	}
}

func isRadiusSize(size string) bool {
	switch size {
	case "xs", "sm", "md", "lg", "xl", "2xl", "3xl", "4xl":
		return true
	default:
		return false
	}
}

func isShadowSize(size string) bool {
	switch size {
	case "2xs", "xs", "sm", "md", "lg", "xl", "2xl":
		return true
	default:
		return false
	}
}

func isFontSize(size string) bool {
	switch size {
	case "xs", "sm", "base", "lg", "xl", "2xl", "3xl", "4xl", "5xl", "6xl", "7xl", "8xl", "9xl":
		return true
	default:
		return false
	}
}

func isContainerSizePart(p *parser.NamePart) bool {
	return isContainerSize(p.AlphaNum) || isContainerSize(p.Ident)
}

func isRadiusSizePart(p *parser.NamePart) bool {
	return isRadiusSize(p.AlphaNum) || isRadiusSize(p.Ident)
}

func isShadowSizePart(p *parser.NamePart) bool {
	return isShadowSize(p.AlphaNum) || isShadowSize(p.Ident)
}

func isSimpleSize(size string) bool {
	switch size {
	case "auto", "px", "full", "screen", "dvw", "dvh", "lvw", "lvh", "svw", "svh", "min", "max", "fit":
		return true
	default:
		return false
	}
}

func isSimpleSizeHeight(size string) bool {
	return size == "lh" || isSimpleSize(size)
}

func hasBeforeAfterVariant(variants []*parser.Segment) bool {
	for _, v := range variants {
		if v.Name == nil {
			continue
		}
		switch v.Name.Head() {
		case "before", "after":
			return true
		}
	}
	return false
}

func isContentPropertyUtility(s *parser.Segment) bool {
	if s.Name == nil || s.Name.Head() != "content" {
		return false
	}

	parts := s.Name.Parts()
	if len(parts) != 1 {
		return false
	}

	switch {
	case simpleMatch(parts, "none"):
		return true
	case parts[0].Custom != "":
		return true
	case parts[0].Arbitrary != "":
		return true
	default:
		return false
	}
}

func isContentNoneUtility(s *parser.Segment) bool {
	return isContentPropertyUtility(s) && simpleMatch(s.Name.Parts(), "none")
}

func simpleSize(k string) string {
	switch k {
	case "auto":
		return k
	case "px":
		return "1px"
	case "full":
		return "100%"
	case "screen":
		return "100vw"
	case "dvw", "dvh", "lvw", "lvh", "svw", "svh":
		return "100" + k
	case "min", "max", "fit":
		return k + "-content"
	}

	return ""
}

func simpleSizeHeight(k string) string {
	switch k {
	case "lh":
		return "1lh"
	case "screen":
		return "100vh"
	}

	return simpleSize(k)
}

func parseLeadingInt(s string) (int, bool) {
	i := 0
	for i < len(s) && s[i] >= '0' && s[i] <= '9' {
		i++
	}

	if i == 0 {
		return 0, false
	}

	n, err := strconv.Atoi(s[:i])
	if err != nil {
		return 0, false
	}

	return n, true
}

func negateAtCondition(c string) (string, bool) {
	space := strings.IndexByte(c, ' ')
	if space <= 0 {
		return "", false
	}

	name := c[:space]
	params := strings.TrimSpace(c[space+1:])
	switch name {
	case "@media", "@supports", "@container":
		if strings.HasPrefix(params, "not ") {
			return name + " " + strings.TrimPrefix(params, "not "), true
		}
		return name + " not " + params, true
	default:
		return "", false
	}
}

type registerer struct {
	registry map[string]struct{}
}

func (r *registerer) Register(_ context.Context, key string) {
	r.registry[key] = struct{}{}
}
