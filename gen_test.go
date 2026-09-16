package upupidae

import (
	"context"
	"fmt"
	"slices"
	"testing"

	"github.com/thinkofher/upupidae/internal/parser"
)

// Expected composite CSS values for tests (mirrors gen composite builders).
const (
	testGradientStops          = "var(--tw-gradient-via-stops, var(--tw-gradient-position), var(--tw-gradient-from) var(--tw-gradient-from-position), var(--tw-gradient-to) var(--tw-gradient-to-position))"
	testGradientViaStops       = "var(--tw-gradient-position), var(--tw-gradient-from) var(--tw-gradient-from-position), var(--tw-gradient-via) var(--tw-gradient-via-position), var(--tw-gradient-to) var(--tw-gradient-to-position)"
	testBorderSpacingValue     = "var(--tw-border-spacing-x) var(--tw-border-spacing-y)"
	testBoxShadowValue         = "var(--tw-inset-shadow), var(--tw-inset-ring-shadow), var(--tw-ring-offset-shadow), var(--tw-ring-shadow), var(--tw-shadow)"
	testCssFilterValue         = "var(--tw-blur, ) var(--tw-brightness, ) var(--tw-contrast, ) var(--tw-grayscale, ) var(--tw-hue-rotate, ) var(--tw-invert, ) var(--tw-saturate, ) var(--tw-sepia, ) var(--tw-drop-shadow, )"
	testCssBackdropFilterValue = "var(--tw-backdrop-blur, ) var(--tw-backdrop-brightness, ) var(--tw-backdrop-contrast, ) var(--tw-backdrop-grayscale, ) var(--tw-backdrop-hue-rotate, ) var(--tw-backdrop-invert, ) var(--tw-backdrop-opacity, ) var(--tw-backdrop-saturate, ) var(--tw-backdrop-sepia, )"
	testTranslateValue         = "var(--tw-translate-x) var(--tw-translate-y)"
	testTranslate3dValue       = "var(--tw-translate-x) var(--tw-translate-y) var(--tw-translate-z)"
	testScaleValue             = "var(--tw-scale-x) var(--tw-scale-y)"
	testScale3dValue           = "var(--tw-scale-x) var(--tw-scale-y) var(--tw-scale-z)"
	testTransformValue         = "var(--tw-rotate-x, ) var(--tw-rotate-y, ) var(--tw-rotate-z, ) var(--tw-skew-x, ) var(--tw-skew-y, )"
	testTouchActionValue       = "var(--tw-pan-x, ) var(--tw-pan-y, ) var(--tw-pinch-zoom, )"
	testScrollbarColorValue    = "var(--tw-scrollbar-thumb) var(--tw-scrollbar-track)"
)

func testRingShadowValue(width string) string {
	return fmt.Sprintf(
		"var(--tw-ring-inset,) 0 0 0 calc(%s + var(--tw-ring-offset-width)) var(--tw-ring-color, currentcolor)",
		width,
	)
}

func testInsetRingShadowValue(width string) string {
	return fmt.Sprintf("inset 0 0 0 %s var(--tw-inset-ring-color, currentcolor)", width)
}

type testVarer struct{}

func (tv *testVarer) Var(_ context.Context, key string) string {
	return fmt.Sprintf("var(%s)", key)
}

func testCssQueriesEqual(a, b Query) bool {
	if a.Name != b.Name {
		return false
	}

	if !slices.Equal(a.KVs, b.KVs) {
		return false
	}

	if len(a.Nested) != len(b.Nested) {
		return false
	}

	if len(a.Nested) == 0 {
		return true
	}

	for i := range a.Nested {
		if !testCssQueriesEqual(a.Nested[i], b.Nested[i]) {
			return false
		}
	}

	return true
}

func testUtilityGeneratedEqual(a, b *utilityGenerated) bool {
	if a == nil && b == nil {
		return true
	}

	if (a == nil && b != nil) || (a != nil && b == nil) {
		return false
	}

	return testCssQueriesEqual(
		Query{KVs: a.kvs, Nested: a.others},
		Query{KVs: b.kvs, Nested: b.others},
	)
}

func TestGen(t *testing.T) {
	g := &gen{
		varer: &testVarer{},
		registerer: &registerer{
			registry: map[string]struct{}{},
		},
	}
	p := parser.Must()

	t.Run("utilities", func(t *testing.T) {
		scenario := func(class string, want *utilityGenerated) (string, func(*testing.T)) {
			return class, func(t *testing.T) {
				ast, err := p.ParseString("", class)
				if err != nil {
					t.Error("parse ast")
				}

				got, err := g.utility(t.Context(), ast.Classes[0].Utility())
				if err != nil {
					t.Errorf("utility error: %s", err)
				}

				if !testUtilityGeneratedEqual(got, want) {
					t.Errorf("got (%v) != want (%v)", got, want)
				}
			}
		}

		t.Run(scenario("aspect-square", newUtilityGeneratedKV("aspect-ratio", "1 / 1")))
		t.Run(scenario("aspect-video", newUtilityGeneratedKV("aspect-ratio", "var(--aspect-video)")))
		t.Run(scenario("aspect-auto", newUtilityGeneratedKV("aspect-ratio", "auto")))
		t.Run(scenario("aspect-1/4", newUtilityGeneratedKV("aspect-ratio", "1 / 4")))
		t.Run(scenario("aspect-(--ratio)", newUtilityGeneratedKV("aspect-ratio", "var(--ratio)")))
		t.Run(scenario("aspect-[3/4]", newUtilityGeneratedKV("aspect-ratio", "3/4")))

		t.Run(scenario("columns-3xs", newUtilityGeneratedKV("columns", "var(--container-3xs)")))
		t.Run(scenario("columns-xl", newUtilityGeneratedKV("columns", "var(--container-xl)")))
		t.Run(scenario("columns-auto", newUtilityGeneratedKV("columns", "auto")))
		t.Run(scenario("columns-5", newUtilityGeneratedKV("columns", "5")))
		t.Run(scenario("columns-(--num)", newUtilityGeneratedKV("columns", "var(--num)")))
		t.Run(scenario("columns-[10]", newUtilityGeneratedKV("columns", "10")))

		t.Run(scenario("break-after-auto", newUtilityGeneratedKV("break-after", "auto")))
		t.Run(scenario("break-before-page", newUtilityGeneratedKV("break-before", "page")))
		t.Run(scenario("break-inside-avoid-column", newUtilityGeneratedKV("break-inside", "avoid-column")))

		t.Run(scenario("box-decoration-clone", newUtilityGeneratedKV("box-decoration-break", "clone")))
		t.Run(scenario("box-border", newUtilityGeneratedKV("box-sizing", "border-box")))

		t.Run(scenario("block", newUtilityGeneratedKV("display", "block")))
		t.Run(scenario("inline-grid", newUtilityGeneratedKV("display", "inline-grid")))
		t.Run(scenario("inline-flex", newUtilityGeneratedKV("display", "inline-flex")))
		t.Run(scenario("flow-root", newUtilityGeneratedKV("display", "flow-root")))
		t.Run(scenario("flex", newUtilityGeneratedKV("display", "flex")))
		t.Run(scenario("grid", newUtilityGeneratedKV("display", "grid")))
		t.Run(scenario("list-item", newUtilityGeneratedKV("display", "list-item")))
		t.Run(scenario("sr-only", &utilityGenerated{kvs: []KV{
			{Key: "position", Value: "absolute"},
			{Key: "width", Value: "1px"},
			{Key: "height", Value: "1px"},
			{Key: "padding", Value: "0"},
			{Key: "margin", Value: "-1px"},
			{Key: "overflow", Value: "hidden"},
			{Key: "clip-path", Value: "inset(50%)"},
			{Key: "white-space", Value: "nowrap"},
			{Key: "border-width", Value: "0"},
		}}))
		t.Run(scenario("not-sr-only", &utilityGenerated{kvs: []KV{
			{Key: "position", Value: "static"},
			{Key: "width", Value: "auto"},
			{Key: "height", Value: "auto"},
			{Key: "padding", Value: "0"},
			{Key: "margin", Value: "0"},
			{Key: "overflow", Value: "visible"},
			{Key: "clip-path", Value: "none"},
			{Key: "white-space", Value: "normal"},
		}}))
		t.Run(scenario("table-row", newUtilityGeneratedKV("display", "table-row")))
		t.Run(scenario("table-footer-group", newUtilityGeneratedKV("display", "table-footer-group")))
		t.Run(scenario("table-auto", newUtilityGeneratedKV("table-layout", "auto")))
		t.Run(scenario("table-fixed", newUtilityGeneratedKV("table-layout", "fixed")))
		t.Run(scenario("caption-top", newUtilityGeneratedKV("caption-side", "top")))
		t.Run(scenario("caption-bottom", newUtilityGeneratedKV("caption-side", "bottom")))

		t.Run(scenario("float-left", newUtilityGeneratedKV("float", "left")))
		t.Run(scenario("float-right", newUtilityGeneratedKV("float", "right")))
		t.Run(scenario("float-none", newUtilityGeneratedKV("float", "none")))
		t.Run(scenario("float-start", newUtilityGeneratedKV("float", "inline-start")))
		t.Run(scenario("float-end", newUtilityGeneratedKV("float", "inline-end")))

		t.Run(scenario("clear-left", newUtilityGeneratedKV("clear", "left")))
		t.Run(scenario("clear-right", newUtilityGeneratedKV("clear", "right")))
		t.Run(scenario("clear-none", newUtilityGeneratedKV("clear", "none")))
		t.Run(scenario("clear-both", newUtilityGeneratedKV("clear", "both")))
		t.Run(scenario("clear-start", newUtilityGeneratedKV("clear", "inline-start")))
		t.Run(scenario("clear-end", newUtilityGeneratedKV("clear", "inline-end")))

		t.Run(scenario("isolate", newUtilityGeneratedKV("isolation", "isolate")))
		t.Run(scenario("isolation-auto", newUtilityGeneratedKV("isolation", "auto")))

		t.Run(scenario("object-contain", newUtilityGeneratedKV("object-fit", "contain")))
		t.Run(scenario("object-none", newUtilityGeneratedKV("object-fit", "none")))
		t.Run(scenario("object-fill", newUtilityGeneratedKV("object-fit", "fill")))
		t.Run(scenario("object-cover", newUtilityGeneratedKV("object-fit", "cover")))
		t.Run(scenario("object-scale-down", newUtilityGeneratedKV("object-fit", "scale-down")))

		t.Run(scenario("object-top", newUtilityGeneratedKV("object-position", "top")))
		t.Run(scenario("object-top-left", newUtilityGeneratedKV("object-position", "top left")))
		t.Run(scenario("object-top-right", newUtilityGeneratedKV("object-position", "top right")))
		t.Run(scenario("object-bottom", newUtilityGeneratedKV("object-position", "bottom")))
		t.Run(scenario("object-bottom-left", newUtilityGeneratedKV("object-position", "bottom left")))
		t.Run(scenario("object-bottom-right", newUtilityGeneratedKV("object-position", "bottom right")))
		t.Run(scenario("object-left", newUtilityGeneratedKV("object-position", "left")))
		t.Run(scenario("object-right", newUtilityGeneratedKV("object-position", "right")))
		t.Run(scenario("object-center", newUtilityGeneratedKV("object-position", "center")))
		t.Run(scenario("object-(--some-var)", newUtilityGeneratedKV("object-position", "var(--some-var)")))
		t.Run(scenario("object-[test]", newUtilityGeneratedKV("object-position", "test")))
		t.Run(scenario("object-[25%_75%]", newUtilityGeneratedKV("object-position", "25% 75%")))

		t.Run(scenario("overflow-auto", newUtilityGeneratedKV("overflow", "auto")))
		t.Run(scenario("overflow-hidden", newUtilityGeneratedKV("overflow", "hidden")))
		t.Run(scenario("overflow-clip", newUtilityGeneratedKV("overflow", "clip")))
		t.Run(scenario("overflow-visible", newUtilityGeneratedKV("overflow", "visible")))
		t.Run(scenario("overflow-scroll", newUtilityGeneratedKV("overflow", "scroll")))
		t.Run(scenario("overflow-x-auto", newUtilityGeneratedKV("overflow-x", "auto")))
		t.Run(scenario("overflow-x-hidden", newUtilityGeneratedKV("overflow-x", "hidden")))
		t.Run(scenario("overflow-x-clip", newUtilityGeneratedKV("overflow-x", "clip")))
		t.Run(scenario("overflow-x-visible", newUtilityGeneratedKV("overflow-x", "visible")))
		t.Run(scenario("overflow-x-scroll", newUtilityGeneratedKV("overflow-x", "scroll")))
		t.Run(scenario("overflow-y-auto", newUtilityGeneratedKV("overflow-y", "auto")))
		t.Run(scenario("overflow-y-hidden", newUtilityGeneratedKV("overflow-y", "hidden")))
		t.Run(scenario("overflow-y-clip", newUtilityGeneratedKV("overflow-y", "clip")))
		t.Run(scenario("overflow-y-visible", newUtilityGeneratedKV("overflow-y", "visible")))
		t.Run(scenario("overflow-y-scroll", newUtilityGeneratedKV("overflow-y", "scroll")))

		t.Run(scenario("overscroll-auto", newUtilityGeneratedKV("overscroll-behavior", "auto")))
		t.Run(scenario("overscroll-contain", newUtilityGeneratedKV("overscroll-behavior", "contain")))
		t.Run(scenario("overscroll-none", newUtilityGeneratedKV("overscroll-behavior", "none")))
		t.Run(scenario("overscroll-x-auto", newUtilityGeneratedKV("overscroll-behavior-x", "auto")))
		t.Run(scenario("overscroll-x-contain", newUtilityGeneratedKV("overscroll-behavior-x", "contain")))
		t.Run(scenario("overscroll-x-none", newUtilityGeneratedKV("overscroll-behavior-x", "none")))
		t.Run(scenario("overscroll-y-auto", newUtilityGeneratedKV("overscroll-behavior-y", "auto")))
		t.Run(scenario("overscroll-y-contain", newUtilityGeneratedKV("overscroll-behavior-y", "contain")))
		t.Run(scenario("overscroll-y-none", newUtilityGeneratedKV("overscroll-behavior-y", "none")))

		t.Run(scenario("static", newUtilityGeneratedKV("position", "static")))
		t.Run(scenario("fixed", newUtilityGeneratedKV("position", "fixed")))
		t.Run(scenario("absolute", newUtilityGeneratedKV("position", "absolute")))
		t.Run(scenario("relative", newUtilityGeneratedKV("position", "relative")))
		t.Run(scenario("sticky", newUtilityGeneratedKV("position", "sticky")))

		t.Run(scenario("inset-5", newUtilityGeneratedKV("inset", "calc(var(--spacing) * 5)")))
		t.Run(scenario("inset-3.5", newUtilityGeneratedKV("inset", "calc(var(--spacing) * 3.5)")))
		t.Run(scenario("inset-3.2324", newUtilityGeneratedKV("inset", "calc(var(--spacing) * 3.2324)")))
		t.Run(scenario("-inset-5", newUtilityGeneratedKV("inset", "calc(var(--spacing) * -5)")))
		t.Run(scenario("-inset-px", newUtilityGeneratedKV("inset", "-1px")))
		t.Run(scenario("inset-px", newUtilityGeneratedKV("inset", "1px")))
		t.Run(scenario("inset-full", newUtilityGeneratedKV("inset", "100%")))
		t.Run(scenario("-inset-full", newUtilityGeneratedKV("inset", "-100%")))
		t.Run(scenario("inset-auto", newUtilityGeneratedKV("inset", "auto")))
		t.Run(scenario("inset-(--my-position)", newUtilityGeneratedKV("inset", "var(--my-position)")))
		t.Run(scenario("inset-[3px]", newUtilityGeneratedKV("inset", "3px")))
		t.Run(scenario("inset-1/2", newUtilityGeneratedKV("inset", "calc(1/2 * 100%)")))
		t.Run(scenario("-inset-1/2", newUtilityGeneratedKV("inset", "-calc(1/2 * 100%)")))

		t.Run(scenario("inset-x-5", newUtilityGeneratedKV("inset-inline", "calc(var(--spacing) * 5)")))
		t.Run(scenario("inset-x-3.5", newUtilityGeneratedKV("inset-inline", "calc(var(--spacing) * 3.5)")))
		t.Run(scenario("inset-x-3.2324", newUtilityGeneratedKV("inset-inline", "calc(var(--spacing) * 3.2324)")))
		t.Run(scenario("-inset-x-5", newUtilityGeneratedKV("inset-inline", "calc(var(--spacing) * -5)")))
		t.Run(scenario("-inset-x-px", newUtilityGeneratedKV("inset-inline", "-1px")))
		t.Run(scenario("inset-x-px", newUtilityGeneratedKV("inset-inline", "1px")))
		t.Run(scenario("inset-x-full", newUtilityGeneratedKV("inset-inline", "100%")))
		t.Run(scenario("-inset-x-full", newUtilityGeneratedKV("inset-inline", "-100%")))
		t.Run(scenario("inset-x-auto", newUtilityGeneratedKV("inset-inline", "auto")))
		t.Run(scenario("inset-x-(--my-position)", newUtilityGeneratedKV("inset-inline", "var(--my-position)")))
		t.Run(scenario("inset-x-[3px]", newUtilityGeneratedKV("inset-inline", "3px")))
		t.Run(scenario("inset-x-1/2", newUtilityGeneratedKV("inset-inline", "calc(1/2 * 100%)")))

		t.Run(scenario("inset-y-5", newUtilityGeneratedKV("inset-block", "calc(var(--spacing) * 5)")))
		t.Run(scenario("inset-y-3.5", newUtilityGeneratedKV("inset-block", "calc(var(--spacing) * 3.5)")))
		t.Run(scenario("inset-y-3.2324", newUtilityGeneratedKV("inset-block", "calc(var(--spacing) * 3.2324)")))
		t.Run(scenario("-inset-y-5", newUtilityGeneratedKV("inset-block", "calc(var(--spacing) * -5)")))
		t.Run(scenario("-inset-y-px", newUtilityGeneratedKV("inset-block", "-1px")))
		t.Run(scenario("inset-y-px", newUtilityGeneratedKV("inset-block", "1px")))
		t.Run(scenario("inset-y-full", newUtilityGeneratedKV("inset-block", "100%")))
		t.Run(scenario("-inset-y-full", newUtilityGeneratedKV("inset-block", "-100%")))
		t.Run(scenario("inset-y-auto", newUtilityGeneratedKV("inset-block", "auto")))
		t.Run(scenario("inset-y-(--my-position)", newUtilityGeneratedKV("inset-block", "var(--my-position)")))
		t.Run(scenario("inset-y-[3px]", newUtilityGeneratedKV("inset-block", "3px")))

		t.Run(scenario("inset-s-5", newUtilityGeneratedKV("inset-inline-start", "calc(var(--spacing) * 5)")))
		t.Run(scenario("inset-s-3.5", newUtilityGeneratedKV("inset-inline-start", "calc(var(--spacing) * 3.5)")))
		t.Run(scenario("inset-s-3.2324", newUtilityGeneratedKV("inset-inline-start", "calc(var(--spacing) * 3.2324)")))
		t.Run(scenario("-inset-s-5", newUtilityGeneratedKV("inset-inline-start", "calc(var(--spacing) * -5)")))
		t.Run(scenario("-inset-s-px", newUtilityGeneratedKV("inset-inline-start", "-1px")))
		t.Run(scenario("inset-s-px", newUtilityGeneratedKV("inset-inline-start", "1px")))
		t.Run(scenario("inset-s-full", newUtilityGeneratedKV("inset-inline-start", "100%")))
		t.Run(scenario("-inset-s-full", newUtilityGeneratedKV("inset-inline-start", "-100%")))
		t.Run(scenario("inset-s-auto", newUtilityGeneratedKV("inset-inline-start", "auto")))
		t.Run(scenario("inset-s-(--my-position)", newUtilityGeneratedKV("inset-inline-start", "var(--my-position)")))
		t.Run(scenario("inset-s-[3px]", newUtilityGeneratedKV("inset-inline-start", "3px")))

		t.Run(scenario("inset-e-5", newUtilityGeneratedKV("inset-inline-end", "calc(var(--spacing) * 5)")))
		t.Run(scenario("inset-e-3.5", newUtilityGeneratedKV("inset-inline-end", "calc(var(--spacing) * 3.5)")))
		t.Run(scenario("inset-e-3.2324", newUtilityGeneratedKV("inset-inline-end", "calc(var(--spacing) * 3.2324)")))
		t.Run(scenario("-inset-e-5", newUtilityGeneratedKV("inset-inline-end", "calc(var(--spacing) * -5)")))
		t.Run(scenario("-inset-e-px", newUtilityGeneratedKV("inset-inline-end", "-1px")))
		t.Run(scenario("inset-e-px", newUtilityGeneratedKV("inset-inline-end", "1px")))
		t.Run(scenario("inset-e-full", newUtilityGeneratedKV("inset-inline-end", "100%")))
		t.Run(scenario("-inset-e-full", newUtilityGeneratedKV("inset-inline-end", "-100%")))
		t.Run(scenario("inset-e-auto", newUtilityGeneratedKV("inset-inline-end", "auto")))
		t.Run(scenario("inset-e-(--my-position)", newUtilityGeneratedKV("inset-inline-end", "var(--my-position)")))
		t.Run(scenario("inset-e-[3px]", newUtilityGeneratedKV("inset-inline-end", "3px")))

		t.Run(scenario("inset-bs-5", newUtilityGeneratedKV("inset-block-start", "calc(var(--spacing) * 5)")))
		t.Run(scenario("inset-bs-3.5", newUtilityGeneratedKV("inset-block-start", "calc(var(--spacing) * 3.5)")))
		t.Run(scenario("inset-bs-3.2324", newUtilityGeneratedKV("inset-block-start", "calc(var(--spacing) * 3.2324)")))
		t.Run(scenario("-inset-bs-5", newUtilityGeneratedKV("inset-block-start", "calc(var(--spacing) * -5)")))
		t.Run(scenario("-inset-bs-px", newUtilityGeneratedKV("inset-block-start", "-1px")))
		t.Run(scenario("inset-bs-px", newUtilityGeneratedKV("inset-block-start", "1px")))
		t.Run(scenario("inset-bs-full", newUtilityGeneratedKV("inset-block-start", "100%")))
		t.Run(scenario("-inset-bs-full", newUtilityGeneratedKV("inset-block-start", "-100%")))
		t.Run(scenario("inset-bs-auto", newUtilityGeneratedKV("inset-block-start", "auto")))
		t.Run(scenario("inset-bs-(--my-position)", newUtilityGeneratedKV("inset-block-start", "var(--my-position)")))
		t.Run(scenario("inset-bs-[3px]", newUtilityGeneratedKV("inset-block-start", "3px")))

		t.Run(scenario("inset-be-5", newUtilityGeneratedKV("inset-block-end", "calc(var(--spacing) * 5)")))
		t.Run(scenario("inset-be-3.5", newUtilityGeneratedKV("inset-block-end", "calc(var(--spacing) * 3.5)")))
		t.Run(scenario("inset-be-3.2324", newUtilityGeneratedKV("inset-block-end", "calc(var(--spacing) * 3.2324)")))
		t.Run(scenario("-inset-be-5", newUtilityGeneratedKV("inset-block-end", "calc(var(--spacing) * -5)")))
		t.Run(scenario("-inset-be-px", newUtilityGeneratedKV("inset-block-end", "-1px")))
		t.Run(scenario("inset-be-px", newUtilityGeneratedKV("inset-block-end", "1px")))
		t.Run(scenario("inset-be-full", newUtilityGeneratedKV("inset-block-end", "100%")))
		t.Run(scenario("-inset-be-full", newUtilityGeneratedKV("inset-block-end", "-100%")))
		t.Run(scenario("inset-be-auto", newUtilityGeneratedKV("inset-block-end", "auto")))
		t.Run(scenario("inset-be-(--my-position)", newUtilityGeneratedKV("inset-block-end", "var(--my-position)")))
		t.Run(scenario("inset-be-[3px]", newUtilityGeneratedKV("inset-block-end", "3px")))

		t.Run(scenario("top-5", newUtilityGeneratedKV("top", "calc(var(--spacing) * 5)")))
		t.Run(scenario("top-3.5", newUtilityGeneratedKV("top", "calc(var(--spacing) * 3.5)")))
		t.Run(scenario("top-3.2324", newUtilityGeneratedKV("top", "calc(var(--spacing) * 3.2324)")))
		t.Run(scenario("-top-5", newUtilityGeneratedKV("top", "calc(var(--spacing) * -5)")))
		t.Run(scenario("-top-px", newUtilityGeneratedKV("top", "-1px")))
		t.Run(scenario("top-px", newUtilityGeneratedKV("top", "1px")))
		t.Run(scenario("top-full", newUtilityGeneratedKV("top", "100%")))
		t.Run(scenario("-top-full", newUtilityGeneratedKV("top", "-100%")))
		t.Run(scenario("top-auto", newUtilityGeneratedKV("top", "auto")))
		t.Run(scenario("top-(--my-position)", newUtilityGeneratedKV("top", "var(--my-position)")))
		t.Run(scenario("top-[3px]", newUtilityGeneratedKV("top", "3px")))
		t.Run(scenario("top-1/2", newUtilityGeneratedKV("top", "calc(1/2 * 100%)")))
		t.Run(scenario("-top-1/2", newUtilityGeneratedKV("top", "-calc(1/2 * 100%)")))

		t.Run(scenario("bottom-5", newUtilityGeneratedKV("bottom", "calc(var(--spacing) * 5)")))
		t.Run(scenario("bottom-3.5", newUtilityGeneratedKV("bottom", "calc(var(--spacing) * 3.5)")))
		t.Run(scenario("bottom-3.2324", newUtilityGeneratedKV("bottom", "calc(var(--spacing) * 3.2324)")))
		t.Run(scenario("-bottom-5", newUtilityGeneratedKV("bottom", "calc(var(--spacing) * -5)")))
		t.Run(scenario("-bottom-px", newUtilityGeneratedKV("bottom", "-1px")))
		t.Run(scenario("bottom-px", newUtilityGeneratedKV("bottom", "1px")))
		t.Run(scenario("bottom-full", newUtilityGeneratedKV("bottom", "100%")))
		t.Run(scenario("-bottom-full", newUtilityGeneratedKV("bottom", "-100%")))
		t.Run(scenario("bottom-auto", newUtilityGeneratedKV("bottom", "auto")))
		t.Run(scenario("bottom-(--my-position)", newUtilityGeneratedKV("bottom", "var(--my-position)")))
		t.Run(scenario("bottom-[3px]", newUtilityGeneratedKV("bottom", "3px")))

		t.Run(scenario("left-5", newUtilityGeneratedKV("left", "calc(var(--spacing) * 5)")))
		t.Run(scenario("left-3.5", newUtilityGeneratedKV("left", "calc(var(--spacing) * 3.5)")))
		t.Run(scenario("left-3.2324", newUtilityGeneratedKV("left", "calc(var(--spacing) * 3.2324)")))
		t.Run(scenario("-left-5", newUtilityGeneratedKV("left", "calc(var(--spacing) * -5)")))
		t.Run(scenario("-left-px", newUtilityGeneratedKV("left", "-1px")))
		t.Run(scenario("left-px", newUtilityGeneratedKV("left", "1px")))
		t.Run(scenario("left-full", newUtilityGeneratedKV("left", "100%")))
		t.Run(scenario("-left-full", newUtilityGeneratedKV("left", "-100%")))
		t.Run(scenario("left-auto", newUtilityGeneratedKV("left", "auto")))
		t.Run(scenario("left-(--my-position)", newUtilityGeneratedKV("left", "var(--my-position)")))
		t.Run(scenario("left-[3px]", newUtilityGeneratedKV("left", "3px")))

		t.Run(scenario("right-5", newUtilityGeneratedKV("right", "calc(var(--spacing) * 5)")))
		t.Run(scenario("right-3.5", newUtilityGeneratedKV("right", "calc(var(--spacing) * 3.5)")))
		t.Run(scenario("right-3.2324", newUtilityGeneratedKV("right", "calc(var(--spacing) * 3.2324)")))
		t.Run(scenario("-right-5", newUtilityGeneratedKV("right", "calc(var(--spacing) * -5)")))
		t.Run(scenario("-right-px", newUtilityGeneratedKV("right", "-1px")))
		t.Run(scenario("right-px", newUtilityGeneratedKV("right", "1px")))
		t.Run(scenario("right-full", newUtilityGeneratedKV("right", "100%")))
		t.Run(scenario("-right-full", newUtilityGeneratedKV("right", "-100%")))
		t.Run(scenario("right-auto", newUtilityGeneratedKV("right", "auto")))
		t.Run(scenario("right-(--my-position)", newUtilityGeneratedKV("right", "var(--my-position)")))
		t.Run(scenario("right-[3px]", newUtilityGeneratedKV("right", "3px")))

		t.Run(scenario("visible", newUtilityGeneratedKV("visibility", "visible")))
		t.Run(scenario("invisible", newUtilityGeneratedKV("visibility", "hidden")))
		t.Run(scenario("collapse", newUtilityGeneratedKV("visibility", "collapse")))

		t.Run(scenario("z-auto", newUtilityGeneratedKV("z-index", "auto")))
		t.Run(scenario("z-10", newUtilityGeneratedKV("z-index", "10")))
		t.Run(scenario("z-(--test)", newUtilityGeneratedKV("z-index", "var(--test)")))
		t.Run(scenario("z-[5]", newUtilityGeneratedKV("z-index", "5")))

		t.Run(scenario("opacity-0", newUtilityGeneratedKV("opacity", "0%")))
		t.Run(scenario("opacity-50", newUtilityGeneratedKV("opacity", "50%")))
		t.Run(scenario("opacity-100", newUtilityGeneratedKV("opacity", "100%")))
		t.Run(scenario("opacity-(--my-opacity)", newUtilityGeneratedKV("opacity", "var(--my-opacity)")))
		t.Run(scenario("opacity-[.67]", newUtilityGeneratedKV("opacity", ".67")))

		t.Run(scenario("mix-blend-normal", newUtilityGeneratedKV("mix-blend-mode", "normal")))
		t.Run(scenario("mix-blend-multiply", newUtilityGeneratedKV("mix-blend-mode", "multiply")))
		t.Run(scenario("mix-blend-screen", newUtilityGeneratedKV("mix-blend-mode", "screen")))
		t.Run(scenario("mix-blend-overlay", newUtilityGeneratedKV("mix-blend-mode", "overlay")))
		t.Run(scenario("mix-blend-darken", newUtilityGeneratedKV("mix-blend-mode", "darken")))
		t.Run(scenario("mix-blend-lighten", newUtilityGeneratedKV("mix-blend-mode", "lighten")))
		t.Run(scenario("mix-blend-color-dodge", newUtilityGeneratedKV("mix-blend-mode", "color-dodge")))
		t.Run(scenario("mix-blend-color-burn", newUtilityGeneratedKV("mix-blend-mode", "color-burn")))
		t.Run(scenario("mix-blend-hard-light", newUtilityGeneratedKV("mix-blend-mode", "hard-light")))
		t.Run(scenario("mix-blend-soft-light", newUtilityGeneratedKV("mix-blend-mode", "soft-light")))
		t.Run(scenario("mix-blend-difference", newUtilityGeneratedKV("mix-blend-mode", "difference")))
		t.Run(scenario("mix-blend-exclusion", newUtilityGeneratedKV("mix-blend-mode", "exclusion")))
		t.Run(scenario("mix-blend-hue", newUtilityGeneratedKV("mix-blend-mode", "hue")))
		t.Run(scenario("mix-blend-saturation", newUtilityGeneratedKV("mix-blend-mode", "saturation")))
		t.Run(scenario("mix-blend-color", newUtilityGeneratedKV("mix-blend-mode", "color")))
		t.Run(scenario("mix-blend-luminosity", newUtilityGeneratedKV("mix-blend-mode", "luminosity")))
		t.Run(scenario("mix-blend-plus-darker", newUtilityGeneratedKV("mix-blend-mode", "plus-darker")))
		t.Run(scenario("mix-blend-plus-lighter", newUtilityGeneratedKV("mix-blend-mode", "plus-lighter")))

		t.Run(scenario("basis-64", newUtilityGeneratedKV("flex-basis", "calc(var(--spacing) * 64)")))
		t.Run(scenario("basis-5/12", newUtilityGeneratedKV("flex-basis", "calc(5 / 12 * 100%)")))
		t.Run(scenario("basis-5/12", newUtilityGeneratedKV("flex-basis", "calc(5 / 12 * 100%)")))
		t.Run(scenario("basis-3xl", newUtilityGeneratedKV("flex-basis", "var(--container-3xl)")))
		t.Run(scenario("basis-(--test)", newUtilityGeneratedKV("flex-basis", "var(--test)")))
		t.Run(scenario("basis-[30vw]", newUtilityGeneratedKV("flex-basis", "30vw")))

		t.Run(scenario("flex-row", newUtilityGeneratedKV("flex-direction", "row")))
		t.Run(scenario("flex-col", newUtilityGeneratedKV("flex-direction", "column")))
		t.Run(scenario("flex-row-reverse", newUtilityGeneratedKV("flex-direction", "row-reverse")))
		t.Run(scenario("flex-col-reverse", newUtilityGeneratedKV("flex-direction", "column-reverse")))
		t.Run(scenario("flex-wrap", newUtilityGeneratedKV("flex-wrap", "wrap")))
		t.Run(scenario("flex-nowrap", newUtilityGeneratedKV("flex-wrap", "nowrap")))
		t.Run(scenario("flex-wrap-reverse", newUtilityGeneratedKV("flex-wrap", "wrap-reverse")))
		t.Run(scenario("flex-4", newUtilityGeneratedKV("flex", "4")))
		t.Run(scenario("flex-5/6", newUtilityGeneratedKV("flex", "calc(5/6 * 100%)")))
		t.Run(scenario("flex-auto", newUtilityGeneratedKV("flex", "auto")))
		t.Run(scenario("flex-initial", newUtilityGeneratedKV("flex", "0 auto")))
		t.Run(scenario("flex-none", newUtilityGeneratedKV("flex", "none")))
		t.Run(scenario("flex-(--my-flex)", newUtilityGeneratedKV("flex", "var(--my-flex)")))
		t.Run(scenario("flex-[3_1_auto]", newUtilityGeneratedKV("flex", "3 1 auto")))

		t.Run(scenario("grow", newUtilityGeneratedKV("flex-grow", "1")))
		t.Run(scenario("grow-3", newUtilityGeneratedKV("flex-grow", "3")))
		t.Run(scenario("grow-(--my-grow)", newUtilityGeneratedKV("flex-grow", "var(--my-grow)")))
		t.Run(scenario("grow-[25vw]", newUtilityGeneratedKV("flex-grow", "25vw")))

		t.Run(scenario("shrink", newUtilityGeneratedKV("flex-shrink", "1")))
		t.Run(scenario("shrink-3", newUtilityGeneratedKV("flex-shrink", "3")))
		t.Run(scenario("shrink-(--my-shrink)", newUtilityGeneratedKV("flex-shrink", "var(--my-shrink)")))
		t.Run(scenario("shrink-[25vw]", newUtilityGeneratedKV("flex-shrink", "25vw")))

		t.Run(scenario("order-4", newUtilityGeneratedKV("order", "4")))
		t.Run(scenario("-order-4", newUtilityGeneratedKV("order", "calc(4 * -1)")))
		t.Run(scenario("order-first", newUtilityGeneratedKV("order", "-9999")))
		t.Run(scenario("order-last", newUtilityGeneratedKV("order", "9999")))
		t.Run(scenario("order-[min(var(--total-items),10)]", newUtilityGeneratedKV("order", "min(var(--total-items), 10)")))

		t.Run(scenario("grid-rows-3", newUtilityGeneratedKV("grid-template-rows", "repeat(3, minmax(0, 1fr))")))
		t.Run(scenario("grid-rows-none", newUtilityGeneratedKV("grid-template-rows", "none")))
		t.Run(scenario("grid-rows-subgrid", newUtilityGeneratedKV("grid-template-rows", "subgrid")))
		t.Run(scenario("grid-rows-[200px_minmax(900px,1fr)_100px]", newUtilityGeneratedKV("grid-template-rows", "200px minmax(900px, 1fr) 100px")))
		t.Run(scenario("grid-rows-(--my-grid-rows)", newUtilityGeneratedKV("grid-template-rows", "var(--my-grid-rows)")))

		t.Run(scenario("grid-cols-3", newUtilityGeneratedKV("grid-template-columns", "repeat(3, minmax(0, 1fr))")))
		t.Run(scenario("grid-cols-none", newUtilityGeneratedKV("grid-template-columns", "none")))
		t.Run(scenario("grid-cols-subgrid", newUtilityGeneratedKV("grid-template-columns", "subgrid")))
		t.Run(scenario("grid-cols-[200px_minmax(900px,1fr)_100px]", newUtilityGeneratedKV("grid-template-columns", "200px minmax(900px, 1fr) 100px")))
		t.Run(scenario("grid-cols-(--my-grid-cols)", newUtilityGeneratedKV("grid-template-columns", "var(--my-grid-cols)")))

		t.Run(scenario("col-span-5", newUtilityGeneratedKV("grid-column", "span 5 / span 5")))
		t.Run(scenario("col-span-full", newUtilityGeneratedKV("grid-column", "1 / -1")))
		t.Run(scenario("col-span-(--my-test)", newUtilityGeneratedKV("grid-column", "span var(--my-test) / span var(--my-test)")))
		t.Run(scenario("col-span-[223]", newUtilityGeneratedKV("grid-column", "span 223 / span 223")))
		t.Run(scenario("col-start-5", newUtilityGeneratedKV("grid-column-start", "5")))
		t.Run(scenario("-col-start-6", newUtilityGeneratedKV("grid-column-start", "calc(6 * -1)")))
		t.Run(scenario("col-start-auto", newUtilityGeneratedKV("grid-column-start", "auto")))
		t.Run(scenario("col-start-(--my-test)", newUtilityGeneratedKV("grid-column-start", "var(--my-test)")))
		t.Run(scenario("col-start-[223]", newUtilityGeneratedKV("grid-column-start", "223")))
		t.Run(scenario("col-end-5", newUtilityGeneratedKV("grid-column-end", "5")))
		t.Run(scenario("-col-end-6", newUtilityGeneratedKV("grid-column-end", "calc(6 * -1)")))
		t.Run(scenario("col-end-auto", newUtilityGeneratedKV("grid-column-end", "auto")))
		t.Run(scenario("col-end-(--my-test)", newUtilityGeneratedKV("grid-column-end", "var(--my-test)")))
		t.Run(scenario("col-end-[223]", newUtilityGeneratedKV("grid-column-end", "223")))
		t.Run(scenario("col-auto", newUtilityGeneratedKV("grid-column", "auto")))
		t.Run(scenario("col-5", newUtilityGeneratedKV("grid-column", "5")))
		t.Run(scenario("-col-6", newUtilityGeneratedKV("grid-column", "calc(6 * -1)")))
		t.Run(scenario("col-(--my-test)", newUtilityGeneratedKV("grid-column", "var(--my-test)")))
		t.Run(scenario("col-[16_/_span_16]", newUtilityGeneratedKV("grid-column", "16 / span 16")))

		t.Run(scenario("row-span-5", newUtilityGeneratedKV("grid-row", "span 5 / span 5")))
		t.Run(scenario("row-span-full", newUtilityGeneratedKV("grid-row", "1 / -1")))
		t.Run(scenario("row-span-(--my-test)", newUtilityGeneratedKV("grid-row", "span var(--my-test) / span var(--my-test)")))
		t.Run(scenario("row-span-[223]", newUtilityGeneratedKV("grid-row", "span 223 / span 223")))
		t.Run(scenario("row-start-5", newUtilityGeneratedKV("grid-row-start", "5")))
		t.Run(scenario("-row-start-6", newUtilityGeneratedKV("grid-row-start", "calc(6 * -1)")))
		t.Run(scenario("row-start-auto", newUtilityGeneratedKV("grid-row-start", "auto")))
		t.Run(scenario("row-start-(--my-test)", newUtilityGeneratedKV("grid-row-start", "var(--my-test)")))
		t.Run(scenario("row-start-[223]", newUtilityGeneratedKV("grid-row-start", "223")))
		t.Run(scenario("row-end-5", newUtilityGeneratedKV("grid-row-end", "5")))
		t.Run(scenario("-row-end-6", newUtilityGeneratedKV("grid-row-end", "calc(6 * -1)")))
		t.Run(scenario("row-end-auto", newUtilityGeneratedKV("grid-row-end", "auto")))
		t.Run(scenario("row-end-(--my-test)", newUtilityGeneratedKV("grid-row-end", "var(--my-test)")))
		t.Run(scenario("row-end-[223]", newUtilityGeneratedKV("grid-row-end", "223")))
		t.Run(scenario("row-auto", newUtilityGeneratedKV("grid-row", "auto")))
		t.Run(scenario("row-5", newUtilityGeneratedKV("grid-row", "5")))
		t.Run(scenario("-row-6", newUtilityGeneratedKV("grid-row", "calc(6 * -1)")))
		t.Run(scenario("row-(--my-test)", newUtilityGeneratedKV("grid-row", "var(--my-test)")))
		t.Run(scenario("row-[16_/_span_16]", newUtilityGeneratedKV("grid-row", "16 / span 16")))

		t.Run(scenario("grid-flow-row", newUtilityGeneratedKV("grid-auto-flow", "row")))
		t.Run(scenario("grid-flow-col", newUtilityGeneratedKV("grid-auto-flow", "column")))
		t.Run(scenario("grid-flow-dense", newUtilityGeneratedKV("grid-auto-flow", "dense")))
		t.Run(scenario("grid-flow-row-dense", newUtilityGeneratedKV("grid-auto-flow", "row dense")))
		t.Run(scenario("grid-flow-col-dense", newUtilityGeneratedKV("grid-auto-flow", "column dense")))

		t.Run(scenario("auto-cols-auto", newUtilityGeneratedKV("grid-auto-columns", "auto")))
		t.Run(scenario("auto-cols-min", newUtilityGeneratedKV("grid-auto-columns", "min-content")))
		t.Run(scenario("auto-cols-max", newUtilityGeneratedKV("grid-auto-columns", "max-content")))
		t.Run(scenario("auto-cols-fr", newUtilityGeneratedKV("grid-auto-columns", "minmax(0, 1fr)")))
		t.Run(scenario("auto-cols-5", newUtilityGeneratedKV("grid-auto-columns", "calc(var(--spacing) * 5)")))
		t.Run(scenario("auto-cols-(--my-auto-cols)", newUtilityGeneratedKV("grid-auto-columns", "var(--my-auto-cols)")))
		t.Run(scenario("auto-cols-[minmax(0,2fr)]", newUtilityGeneratedKV("grid-auto-columns", "minmax(0, 2fr)")))

		t.Run(scenario("auto-rows-auto", newUtilityGeneratedKV("grid-auto-rows", "auto")))
		t.Run(scenario("auto-rows-min", newUtilityGeneratedKV("grid-auto-rows", "min-content")))
		t.Run(scenario("auto-rows-max", newUtilityGeneratedKV("grid-auto-rows", "max-content")))
		t.Run(scenario("auto-rows-fr", newUtilityGeneratedKV("grid-auto-rows", "minmax(0, 1fr)")))
		t.Run(scenario("auto-rows-5", newUtilityGeneratedKV("grid-auto-rows", "calc(var(--spacing) * 5)")))
		t.Run(scenario("auto-rows-(--my-auto-rows)", newUtilityGeneratedKV("grid-auto-rows", "var(--my-auto-rows)")))
		t.Run(scenario("auto-rows-[minmax(0,2fr)]", newUtilityGeneratedKV("grid-auto-rows", "minmax(0, 2fr)")))

		t.Run(scenario("gap-2", newUtilityGeneratedKV("gap", "calc(var(--spacing) * 2)")))
		t.Run(scenario("gap-[10vw]", newUtilityGeneratedKV("gap", "10vw")))
		t.Run(scenario("gap-(--my-gap)", newUtilityGeneratedKV("gap", "var(--my-gap)")))
		t.Run(scenario("gap-x-2", newUtilityGeneratedKV("column-gap", "calc(var(--spacing) * 2)")))
		t.Run(scenario("gap-x-[10vw]", newUtilityGeneratedKV("column-gap", "10vw")))
		t.Run(scenario("gap-x-(--my-gap)", newUtilityGeneratedKV("column-gap", "var(--my-gap)")))
		t.Run(scenario("gap-y-2", newUtilityGeneratedKV("row-gap", "calc(var(--spacing) * 2)")))
		t.Run(scenario("gap-y-[10vw]", newUtilityGeneratedKV("row-gap", "10vw")))
		t.Run(scenario("gap-y-(--my-gap)", newUtilityGeneratedKV("row-gap", "var(--my-gap)")))

		t.Run(scenario("justify-start", newUtilityGeneratedKV("justify-content", "flex-start")))
		t.Run(scenario("justify-start-safe", newUtilityGeneratedKV("justify-content", "safe flex-start")))
		t.Run(scenario("justify-end", newUtilityGeneratedKV("justify-content", "flex-end")))
		t.Run(scenario("justify-end-safe", newUtilityGeneratedKV("justify-content", "safe flex-end")))
		t.Run(scenario("justify-center", newUtilityGeneratedKV("justify-content", "center")))
		t.Run(scenario("justify-center-safe", newUtilityGeneratedKV("justify-content", "safe center")))
		t.Run(scenario("justify-between", newUtilityGeneratedKV("justify-content", "space-between")))
		t.Run(scenario("justify-around", newUtilityGeneratedKV("justify-content", "space-around")))
		t.Run(scenario("justify-evenly", newUtilityGeneratedKV("justify-content", "space-evenly")))
		t.Run(scenario("justify-stretch", newUtilityGeneratedKV("justify-content", "stretch")))
		t.Run(scenario("justify-baseline", newUtilityGeneratedKV("justify-content", "baseline")))
		t.Run(scenario("justify-normal", newUtilityGeneratedKV("justify-content", "normal")))

		t.Run(scenario("justify-items-start", newUtilityGeneratedKV("justify-items", "start")))
		t.Run(scenario("justify-items-end", newUtilityGeneratedKV("justify-items", "end")))
		t.Run(scenario("justify-items-end-safe", newUtilityGeneratedKV("justify-items", "safe end")))
		t.Run(scenario("justify-items-center", newUtilityGeneratedKV("justify-items", "center")))
		t.Run(scenario("justify-items-center-safe", newUtilityGeneratedKV("justify-items", "safe center")))
		t.Run(scenario("justify-items-stretch", newUtilityGeneratedKV("justify-items", "stretch")))
		t.Run(scenario("justify-items-normal", newUtilityGeneratedKV("justify-items", "normal")))

		t.Run(scenario("justify-self-auto", newUtilityGeneratedKV("justify-self", "auto")))
		t.Run(scenario("justify-self-start", newUtilityGeneratedKV("justify-self", "start")))
		t.Run(scenario("justify-self-center", newUtilityGeneratedKV("justify-self", "center")))
		t.Run(scenario("justify-self-center-safe", newUtilityGeneratedKV("justify-self", "safe center")))
		t.Run(scenario("justify-self-end", newUtilityGeneratedKV("justify-self", "end")))
		t.Run(scenario("justify-self-end-safe", newUtilityGeneratedKV("justify-self", "safe end")))
		t.Run(scenario("justify-self-stretch", newUtilityGeneratedKV("justify-self", "stretch")))

		t.Run(scenario("content-normal", newUtilityGeneratedKV("align-content", "normal")))
		t.Run(scenario("content-center", newUtilityGeneratedKV("align-content", "center")))
		t.Run(scenario("content-start", newUtilityGeneratedKV("align-content", "flex-start")))
		t.Run(scenario("content-end", newUtilityGeneratedKV("align-content", "flex-end")))
		t.Run(scenario("content-between", newUtilityGeneratedKV("align-content", "space-between")))
		t.Run(scenario("content-around", newUtilityGeneratedKV("align-content", "space-around")))
		t.Run(scenario("content-evenly", newUtilityGeneratedKV("align-content", "space-evenly")))
		t.Run(scenario("content-baseline", newUtilityGeneratedKV("align-content", "baseline")))
		t.Run(scenario("content-stretch", newUtilityGeneratedKV("align-content", "stretch")))

		t.Run(scenario("items-start", newUtilityGeneratedKV("align-items", "flex-start")))
		t.Run(scenario("items-end", newUtilityGeneratedKV("align-items", "flex-end")))
		t.Run(scenario("items-end-safe", newUtilityGeneratedKV("align-items", "safe flex-end")))
		t.Run(scenario("items-center", newUtilityGeneratedKV("align-items", "center")))
		t.Run(scenario("items-center-safe", newUtilityGeneratedKV("align-items", "safe center")))
		t.Run(scenario("items-baseline", newUtilityGeneratedKV("align-items", "baseline")))
		t.Run(scenario("items-baseline-last", newUtilityGeneratedKV("align-items", "last baseline")))
		t.Run(scenario("items-stretch", newUtilityGeneratedKV("align-items", "stretch")))

		t.Run(scenario("self-auto", newUtilityGeneratedKV("align-self", "auto")))
		t.Run(scenario("self-start", newUtilityGeneratedKV("align-self", "flex-start")))
		t.Run(scenario("self-end", newUtilityGeneratedKV("align-self", "flex-end")))
		t.Run(scenario("self-end-safe", newUtilityGeneratedKV("align-self", "safe flex-end")))
		t.Run(scenario("self-center", newUtilityGeneratedKV("align-self", "center")))
		t.Run(scenario("self-center-safe", newUtilityGeneratedKV("align-self", "safe center")))
		t.Run(scenario("self-stretch", newUtilityGeneratedKV("align-self", "stretch")))
		t.Run(scenario("self-baseline", newUtilityGeneratedKV("align-self", "baseline")))
		t.Run(scenario("self-baseline-last", newUtilityGeneratedKV("align-self", "last baseline")))

		t.Run(scenario("place-content-center", newUtilityGeneratedKV("place-content", "center")))
		t.Run(scenario("place-content-center-safe", newUtilityGeneratedKV("place-content", "safe center")))
		t.Run(scenario("place-content-start", newUtilityGeneratedKV("place-content", "start")))
		t.Run(scenario("place-content-end", newUtilityGeneratedKV("place-content", "end")))
		t.Run(scenario("place-content-end-safe", newUtilityGeneratedKV("place-content", "safe end")))
		t.Run(scenario("place-content-between", newUtilityGeneratedKV("place-content", "space-between")))
		t.Run(scenario("place-content-around", newUtilityGeneratedKV("place-content", "space-around")))
		t.Run(scenario("place-content-evenly", newUtilityGeneratedKV("place-content", "space-evenly")))
		t.Run(scenario("place-content-baseline", newUtilityGeneratedKV("place-content", "baseline")))
		t.Run(scenario("place-content-stretch", newUtilityGeneratedKV("place-content", "stretch")))

		t.Run(scenario("place-items-start", newUtilityGeneratedKV("place-items", "start")))
		t.Run(scenario("place-items-end", newUtilityGeneratedKV("place-items", "end")))
		t.Run(scenario("place-items-end-safe", newUtilityGeneratedKV("place-items", "safe end")))
		t.Run(scenario("place-items-center", newUtilityGeneratedKV("place-items", "center")))
		t.Run(scenario("place-items-center-safe", newUtilityGeneratedKV("place-items", "safe center")))
		t.Run(scenario("place-items-baseline", newUtilityGeneratedKV("place-items", "baseline")))
		t.Run(scenario("place-items-stretch", newUtilityGeneratedKV("place-items", "stretch")))

		t.Run(scenario("place-self-auto", newUtilityGeneratedKV("place-self", "auto")))
		t.Run(scenario("place-self-start", newUtilityGeneratedKV("place-self", "start")))
		t.Run(scenario("place-self-end", newUtilityGeneratedKV("place-self", "end")))
		t.Run(scenario("place-self-end-safe", newUtilityGeneratedKV("place-self", "safe end")))
		t.Run(scenario("place-self-center", newUtilityGeneratedKV("place-self", "center")))
		t.Run(scenario("place-self-center-safe", newUtilityGeneratedKV("place-self", "safe center")))
		t.Run(scenario("place-self-stretch", newUtilityGeneratedKV("place-self", "stretch")))

		t.Run(scenario("p-8", newUtilityGeneratedKV("padding", "calc(var(--spacing) * 8)")))
		t.Run(scenario("p-px", newUtilityGeneratedKV("padding", "1px")))
		t.Run(scenario("p-(--my-custom-padding)", newUtilityGeneratedKV("padding", "var(--my-custom-padding)")))
		t.Run(scenario("p-[5px]", newUtilityGeneratedKV("padding", "5px")))

		t.Run(scenario("px-8", newUtilityGeneratedKV("padding-inline", "calc(var(--spacing) * 8)")))
		t.Run(scenario("px-px", newUtilityGeneratedKV("padding-inline", "1px")))
		t.Run(scenario("px-(--my-custom-padding)", newUtilityGeneratedKV("padding-inline", "var(--my-custom-padding)")))
		t.Run(scenario("px-[5px]", newUtilityGeneratedKV("padding-inline", "5px")))

		t.Run(scenario("py-8", newUtilityGeneratedKV("padding-block", "calc(var(--spacing) * 8)")))
		t.Run(scenario("py-px", newUtilityGeneratedKV("padding-block", "1px")))
		t.Run(scenario("py-(--my-custom-padding)", newUtilityGeneratedKV("padding-block", "var(--my-custom-padding)")))
		t.Run(scenario("py-[5px]", newUtilityGeneratedKV("padding-block", "5px")))

		t.Run(scenario("pt-8", newUtilityGeneratedKV("padding-top", "calc(var(--spacing) * 8)")))
		t.Run(scenario("pt-px", newUtilityGeneratedKV("padding-top", "1px")))
		t.Run(scenario("pt-(--my-custom-padding)", newUtilityGeneratedKV("padding-top", "var(--my-custom-padding)")))
		t.Run(scenario("pt-[5px]", newUtilityGeneratedKV("padding-top", "5px")))

		t.Run(scenario("pr-8", newUtilityGeneratedKV("padding-right", "calc(var(--spacing) * 8)")))
		t.Run(scenario("pr-px", newUtilityGeneratedKV("padding-right", "1px")))
		t.Run(scenario("pr-(--my-custom-padding)", newUtilityGeneratedKV("padding-right", "var(--my-custom-padding)")))
		t.Run(scenario("pr-[5px]", newUtilityGeneratedKV("padding-right", "5px")))

		t.Run(scenario("pb-8", newUtilityGeneratedKV("padding-bottom", "calc(var(--spacing) * 8)")))
		t.Run(scenario("pb-px", newUtilityGeneratedKV("padding-bottom", "1px")))
		t.Run(scenario("pb-(--my-custom-padding)", newUtilityGeneratedKV("padding-bottom", "var(--my-custom-padding)")))
		t.Run(scenario("pb-[5px]", newUtilityGeneratedKV("padding-bottom", "5px")))

		t.Run(scenario("pl-8", newUtilityGeneratedKV("padding-left", "calc(var(--spacing) * 8)")))
		t.Run(scenario("pl-px", newUtilityGeneratedKV("padding-left", "1px")))
		t.Run(scenario("pl-(--my-custom-padding)", newUtilityGeneratedKV("padding-left", "var(--my-custom-padding)")))
		t.Run(scenario("pl-[5px]", newUtilityGeneratedKV("padding-left", "5px")))

		t.Run(scenario("ps-8", newUtilityGeneratedKV("padding-inline-start", "calc(var(--spacing) * 8)")))
		t.Run(scenario("ps-px", newUtilityGeneratedKV("padding-inline-start", "1px")))
		t.Run(scenario("ps-(--my-custom-padding)", newUtilityGeneratedKV("padding-inline-start", "var(--my-custom-padding)")))
		t.Run(scenario("ps-[5px]", newUtilityGeneratedKV("padding-inline-start", "5px")))

		t.Run(scenario("pe-8", newUtilityGeneratedKV("padding-inline-end", "calc(var(--spacing) * 8)")))
		t.Run(scenario("pe-px", newUtilityGeneratedKV("padding-inline-end", "1px")))
		t.Run(scenario("pe-(--my-custom-padding)", newUtilityGeneratedKV("padding-inline-end", "var(--my-custom-padding)")))
		t.Run(scenario("pe-[5px]", newUtilityGeneratedKV("padding-inline-end", "5px")))

		t.Run(scenario("pbs-8", newUtilityGeneratedKV("padding-block-start", "calc(var(--spacing) * 8)")))
		t.Run(scenario("pbs-px", newUtilityGeneratedKV("padding-block-start", "1px")))
		t.Run(scenario("pbs-(--my-custom-padding)", newUtilityGeneratedKV("padding-block-start", "var(--my-custom-padding)")))
		t.Run(scenario("pbs-[5px]", newUtilityGeneratedKV("padding-block-start", "5px")))

		t.Run(scenario("pbe-8", newUtilityGeneratedKV("padding-block-end", "calc(var(--spacing) * 8)")))
		t.Run(scenario("pbe-px", newUtilityGeneratedKV("padding-block-end", "1px")))
		t.Run(scenario("pbe-(--my-custom-padding)", newUtilityGeneratedKV("padding-block-end", "var(--my-custom-padding)")))
		t.Run(scenario("pbe-[5px]", newUtilityGeneratedKV("padding-block-end", "5px")))

		t.Run(scenario("m-auto", newUtilityGeneratedKV("margin", "auto")))
		t.Run(scenario("m-3", newUtilityGeneratedKV("margin", "calc(var(--spacing) * 3)")))
		t.Run(scenario("-m-3", newUtilityGeneratedKV("margin", "calc(var(--spacing) * -3)")))
		t.Run(scenario("m-px", newUtilityGeneratedKV("margin", "1px")))
		t.Run(scenario("-m-px", newUtilityGeneratedKV("margin", "-1px")))
		t.Run(scenario("m-(--my-custom-margin)", newUtilityGeneratedKV("margin", "var(--my-custom-margin)")))
		t.Run(scenario("m-[5px]", newUtilityGeneratedKV("margin", "5px")))

		t.Run(scenario("my-auto", newUtilityGeneratedKV("margin-block", "auto")))
		t.Run(scenario("my-3", newUtilityGeneratedKV("margin-block", "calc(var(--spacing) * 3)")))
		t.Run(scenario("-my-3", newUtilityGeneratedKV("margin-block", "calc(var(--spacing) * -3)")))
		t.Run(scenario("my-px", newUtilityGeneratedKV("margin-block", "1px")))
		t.Run(scenario("-my-px", newUtilityGeneratedKV("margin-block", "-1px")))
		t.Run(scenario("my-(--my-custom-margin)", newUtilityGeneratedKV("margin-block", "var(--my-custom-margin)")))
		t.Run(scenario("my-[5px]", newUtilityGeneratedKV("margin-block", "5px")))

		t.Run(scenario("ms-auto", newUtilityGeneratedKV("margin-inline-start", "auto")))
		t.Run(scenario("ms-3", newUtilityGeneratedKV("margin-inline-start", "calc(var(--spacing) * 3)")))
		t.Run(scenario("-ms-3", newUtilityGeneratedKV("margin-inline-start", "calc(var(--spacing) * -3)")))
		t.Run(scenario("ms-px", newUtilityGeneratedKV("margin-inline-start", "1px")))
		t.Run(scenario("-ms-px", newUtilityGeneratedKV("margin-inline-start", "-1px")))
		t.Run(scenario("ms-(--my-custom-margin)", newUtilityGeneratedKV("margin-inline-start", "var(--my-custom-margin)")))
		t.Run(scenario("ms-[5px]", newUtilityGeneratedKV("margin-inline-start", "5px")))

		t.Run(scenario("me-auto", newUtilityGeneratedKV("margin-inline-end", "auto")))
		t.Run(scenario("me-3", newUtilityGeneratedKV("margin-inline-end", "calc(var(--spacing) * 3)")))
		t.Run(scenario("-me-3", newUtilityGeneratedKV("margin-inline-end", "calc(var(--spacing) * -3)")))
		t.Run(scenario("me-px", newUtilityGeneratedKV("margin-inline-end", "1px")))
		t.Run(scenario("-me-px", newUtilityGeneratedKV("margin-inline-end", "-1px")))
		t.Run(scenario("me-(--my-custom-margin)", newUtilityGeneratedKV("margin-inline-end", "var(--my-custom-margin)")))
		t.Run(scenario("me-[5px]", newUtilityGeneratedKV("margin-inline-end", "5px")))

		t.Run(scenario("mbs-auto", newUtilityGeneratedKV("margin-block-start", "auto")))
		t.Run(scenario("mbs-3", newUtilityGeneratedKV("margin-block-start", "calc(var(--spacing) * 3)")))
		t.Run(scenario("-mbs-3", newUtilityGeneratedKV("margin-block-start", "calc(var(--spacing) * -3)")))
		t.Run(scenario("mbs-px", newUtilityGeneratedKV("margin-block-start", "1px")))
		t.Run(scenario("-mbs-px", newUtilityGeneratedKV("margin-block-start", "-1px")))
		t.Run(scenario("mbs-(--my-custom-margin)", newUtilityGeneratedKV("margin-block-start", "var(--my-custom-margin)")))
		t.Run(scenario("mbs-[5px]", newUtilityGeneratedKV("margin-block-start", "5px")))

		t.Run(scenario("mbe-auto", newUtilityGeneratedKV("margin-block-end", "auto")))
		t.Run(scenario("mbe-3", newUtilityGeneratedKV("margin-block-end", "calc(var(--spacing) * 3)")))
		t.Run(scenario("-mbe-3", newUtilityGeneratedKV("margin-block-end", "calc(var(--spacing) * -3)")))
		t.Run(scenario("mbe-px", newUtilityGeneratedKV("margin-block-end", "1px")))
		t.Run(scenario("-mbe-px", newUtilityGeneratedKV("margin-block-end", "-1px")))
		t.Run(scenario("mbe-(--my-custom-margin)", newUtilityGeneratedKV("margin-block-end", "var(--my-custom-margin)")))
		t.Run(scenario("mbe-[5px]", newUtilityGeneratedKV("margin-block-end", "5px")))

		t.Run(scenario("mt-auto", newUtilityGeneratedKV("margin-top", "auto")))
		t.Run(scenario("mt-3", newUtilityGeneratedKV("margin-top", "calc(var(--spacing) * 3)")))
		t.Run(scenario("-mt-3", newUtilityGeneratedKV("margin-top", "calc(var(--spacing) * -3)")))
		t.Run(scenario("mt-px", newUtilityGeneratedKV("margin-top", "1px")))
		t.Run(scenario("-mt-px", newUtilityGeneratedKV("margin-top", "-1px")))
		t.Run(scenario("mt-(--my-custom-margin)", newUtilityGeneratedKV("margin-top", "var(--my-custom-margin)")))
		t.Run(scenario("mt-[5px]", newUtilityGeneratedKV("margin-top", "5px")))

		t.Run(scenario("mr-auto", newUtilityGeneratedKV("margin-right", "auto")))
		t.Run(scenario("mr-3", newUtilityGeneratedKV("margin-right", "calc(var(--spacing) * 3)")))
		t.Run(scenario("-mr-3", newUtilityGeneratedKV("margin-right", "calc(var(--spacing) * -3)")))
		t.Run(scenario("mr-px", newUtilityGeneratedKV("margin-right", "1px")))
		t.Run(scenario("-mr-px", newUtilityGeneratedKV("margin-right", "-1px")))
		t.Run(scenario("mr-(--my-custom-margin)", newUtilityGeneratedKV("margin-right", "var(--my-custom-margin)")))
		t.Run(scenario("mr-[5px]", newUtilityGeneratedKV("margin-right", "5px")))

		t.Run(scenario("mb-auto", newUtilityGeneratedKV("margin-bottom", "auto")))
		t.Run(scenario("mb-3", newUtilityGeneratedKV("margin-bottom", "calc(var(--spacing) * 3)")))
		t.Run(scenario("-mb-3", newUtilityGeneratedKV("margin-bottom", "calc(var(--spacing) * -3)")))
		t.Run(scenario("mb-px", newUtilityGeneratedKV("margin-bottom", "1px")))
		t.Run(scenario("-mb-px", newUtilityGeneratedKV("margin-bottom", "-1px")))
		t.Run(scenario("mb-(--my-custom-margin)", newUtilityGeneratedKV("margin-bottom", "var(--my-custom-margin)")))
		t.Run(scenario("mb-[5px]", newUtilityGeneratedKV("margin-bottom", "5px")))

		t.Run(scenario("ml-auto", newUtilityGeneratedKV("margin-left", "auto")))
		t.Run(scenario("ml-3", newUtilityGeneratedKV("margin-left", "calc(var(--spacing) * 3)")))
		t.Run(scenario("-ml-3", newUtilityGeneratedKV("margin-left", "calc(var(--spacing) * -3)")))
		t.Run(scenario("ml-px", newUtilityGeneratedKV("margin-left", "1px")))
		t.Run(scenario("-ml-px", newUtilityGeneratedKV("margin-left", "-1px")))
		t.Run(scenario("ml-(--my-custom-margin)", newUtilityGeneratedKV("margin-left", "var(--my-custom-margin)")))
		t.Run(scenario("ml-[5px]", newUtilityGeneratedKV("margin-left", "5px")))

		t.Run(scenario("space-x-5", &utilityGenerated{
			others: []Query{{
				Name: "& > :not(:last-child)",
				KVs: []KV{
					newCssKV("--tw-space-x-reverse", "0"),
					newCssKV("margin-inline-start", "calc(calc(var(--spacing) * 5) * var(--tw-space-x-reverse))"),
					newCssKV("margin-inline-end", "calc(calc(var(--spacing) * 5) * calc(1 - var(--tw-space-x-reverse)))"),
				},
			}},
		}))
		t.Run(scenario("-space-x-5", &utilityGenerated{
			others: []Query{{
				Name: "& > :not(:last-child)",
				KVs: []KV{
					newCssKV("--tw-space-x-reverse", "0"),
					newCssKV("margin-inline-start", "calc(calc(var(--spacing) * -5) * var(--tw-space-x-reverse))"),
					newCssKV("margin-inline-end", "calc(calc(var(--spacing) * -5) * calc(1 - var(--tw-space-x-reverse)))"),
				},
			}},
		}))

		t.Run(scenario("space-x-px", &utilityGenerated{
			others: []Query{{
				Name: "& > :not(:last-child)",
				KVs: []KV{
					newCssKV("--tw-space-x-reverse", "0"),
					newCssKV("margin-inline-start", "calc(1px * var(--tw-space-x-reverse))"),
					newCssKV("margin-inline-end", "calc(1px * calc(1 - var(--tw-space-x-reverse)))"),
				},
			}},
		}))

		t.Run(scenario("-space-x-px", &utilityGenerated{
			others: []Query{{
				Name: "& > :not(:last-child)",
				KVs: []KV{
					newCssKV("--tw-space-x-reverse", "0"),
					newCssKV("margin-inline-start", "calc(-1px * var(--tw-space-x-reverse))"),
					newCssKV("margin-inline-end", "calc(-1px * calc(1 - var(--tw-space-x-reverse)))"),
				},
			}},
		}))

		t.Run(scenario("space-x-(--my-space)", &utilityGenerated{
			others: []Query{{
				Name: "& > :not(:last-child)",
				KVs: []KV{
					newCssKV("--tw-space-x-reverse", "0"),
					newCssKV("margin-inline-start", "calc(var(--my-space) * var(--tw-space-x-reverse))"),
					newCssKV("margin-inline-end", "calc(var(--my-space) * calc(1 - var(--tw-space-x-reverse)))"),
				},
			}},
		}))

		t.Run(scenario("space-x-[5px]", &utilityGenerated{
			others: []Query{{
				Name: "& > :not(:last-child)",
				KVs: []KV{
					newCssKV("--tw-space-x-reverse", "0"),
					newCssKV("margin-inline-start", "calc(5px * var(--tw-space-x-reverse))"),
					newCssKV("margin-inline-end", "calc(5px * calc(1 - var(--tw-space-x-reverse)))"),
				},
			}},
		}))

		t.Run(scenario("space-y-5", &utilityGenerated{
			others: []Query{{
				Name: "& > :not(:last-child)",
				KVs: []KV{
					newCssKV("--tw-space-y-reverse", "0"),
					newCssKV("margin-block-start", "calc(calc(var(--spacing) * 5) * var(--tw-space-y-reverse))"),
					newCssKV("margin-block-end", "calc(calc(var(--spacing) * 5) * calc(1 - var(--tw-space-y-reverse)))"),
				},
			}},
		}))

		t.Run(scenario("-space-y-5", &utilityGenerated{
			others: []Query{{
				Name: "& > :not(:last-child)",
				KVs: []KV{
					newCssKV("--tw-space-y-reverse", "0"),
					newCssKV("margin-block-start", "calc(calc(var(--spacing) * -5) * var(--tw-space-y-reverse))"),
					newCssKV("margin-block-end", "calc(calc(var(--spacing) * -5) * calc(1 - var(--tw-space-y-reverse)))"),
				},
			}},
		}))

		t.Run(scenario("space-y-px", &utilityGenerated{
			others: []Query{{
				Name: "& > :not(:last-child)",
				KVs: []KV{
					newCssKV("--tw-space-y-reverse", "0"),
					newCssKV("margin-block-start", "calc(1px * var(--tw-space-y-reverse))"),
					newCssKV("margin-block-end", "calc(1px * calc(1 - var(--tw-space-y-reverse)))"),
				},
			}},
		}))

		t.Run(scenario("-space-y-px", &utilityGenerated{
			others: []Query{{
				Name: "& > :not(:last-child)",
				KVs: []KV{
					newCssKV("--tw-space-y-reverse", "0"),
					newCssKV("margin-block-start", "calc(-1px * var(--tw-space-y-reverse))"),
					newCssKV("margin-block-end", "calc(-1px * calc(1 - var(--tw-space-y-reverse)))"),
				},
			}},
		}))

		t.Run(scenario("space-y-(--my-space)", &utilityGenerated{
			others: []Query{{
				Name: "& > :not(:last-child)",
				KVs: []KV{
					newCssKV("--tw-space-y-reverse", "0"),
					newCssKV("margin-block-start", "calc(var(--my-space) * var(--tw-space-y-reverse))"),
					newCssKV("margin-block-end", "calc(var(--my-space) * calc(1 - var(--tw-space-y-reverse)))"),
				},
			}},
		}))

		t.Run(scenario("space-y-[5px]", &utilityGenerated{
			others: []Query{{
				Name: "& > :not(:last-child)",
				KVs: []KV{
					newCssKV("--tw-space-y-reverse", "0"),
					newCssKV("margin-block-start", "calc(5px * var(--tw-space-y-reverse))"),
					newCssKV("margin-block-end", "calc(5px * calc(1 - var(--tw-space-y-reverse)))"),
				},
			}},
		}))
		t.Run(scenario("space-x-reverse", &utilityGenerated{
			others: []Query{{
				Name: "& > :not(:last-child)",
				KVs: []KV{
					newCssKV("--tw-space-x-reverse", "1"),
				},
			}},
		}))

		t.Run(scenario("space-y-reverse", &utilityGenerated{
			others: []Query{{
				Name: "& > :not(:last-child)",
				KVs: []KV{
					newCssKV("--tw-space-y-reverse", "1"),
				},
			}},
		}))

		t.Run(scenario("w-auto", newUtilityGeneratedKV("width", "auto")))
		t.Run(scenario("w-px", newUtilityGeneratedKV("width", "1px")))
		t.Run(scenario("w-2", newUtilityGeneratedKV("width", "calc(var(--spacing) * 2)")))
		t.Run(scenario("w-1/2", newUtilityGeneratedKV("width", "calc(1/2 * 100%)")))
		t.Run(scenario("w-3/5", newUtilityGeneratedKV("width", "calc(3/5 * 100%)")))
		t.Run(scenario("w-full", newUtilityGeneratedKV("width", "100%")))
		t.Run(scenario("w-screen", newUtilityGeneratedKV("width", "100vw")))
		t.Run(scenario("w-svw", newUtilityGeneratedKV("width", "100svw")))
		t.Run(scenario("w-lvw", newUtilityGeneratedKV("width", "100lvw")))
		t.Run(scenario("w-dvw", newUtilityGeneratedKV("width", "100dvw")))
		t.Run(scenario("w-min", newUtilityGeneratedKV("width", "min-content")))
		t.Run(scenario("w-max", newUtilityGeneratedKV("width", "max-content")))
		t.Run(scenario("w-fit", newUtilityGeneratedKV("width", "fit-content")))
		t.Run(scenario("w-(--my-width)", newUtilityGeneratedKV("width", "var(--my-width)")))
		t.Run(scenario("w-[5px]", newUtilityGeneratedKV("width", "5px")))
		t.Run(scenario("w-3xs", newUtilityGeneratedKV("width", "var(--container-3xs)")))
		t.Run(scenario("w-2xs", newUtilityGeneratedKV("width", "var(--container-2xs)")))
		t.Run(scenario("w-xs", newUtilityGeneratedKV("width", "var(--container-xs)")))
		t.Run(scenario("w-sm", newUtilityGeneratedKV("width", "var(--container-sm)")))
		t.Run(scenario("w-md", newUtilityGeneratedKV("width", "var(--container-md)")))
		t.Run(scenario("w-lg", newUtilityGeneratedKV("width", "var(--container-lg)")))
		t.Run(scenario("w-xl", newUtilityGeneratedKV("width", "var(--container-xl)")))
		t.Run(scenario("w-2xl", newUtilityGeneratedKV("width", "var(--container-2xl)")))
		t.Run(scenario("w-3xl", newUtilityGeneratedKV("width", "var(--container-3xl)")))
		t.Run(scenario("w-4xl", newUtilityGeneratedKV("width", "var(--container-4xl)")))
		t.Run(scenario("w-5xl", newUtilityGeneratedKV("width", "var(--container-5xl)")))
		t.Run(scenario("w-6xl", newUtilityGeneratedKV("width", "var(--container-6xl)")))
		t.Run(scenario("w-7xl", newUtilityGeneratedKV("width", "var(--container-7xl)")))

		t.Run(scenario("size-auto", &utilityGenerated{
			kvs: []KV{
				newCssKV("width", "auto"),
				newCssKV("height", "auto"),
			},
		}))
		t.Run(scenario("size-px", &utilityGenerated{
			kvs: []KV{
				newCssKV("width", "1px"),
				newCssKV("height", "1px"),
			},
		}))
		t.Run(scenario("size-2", &utilityGenerated{
			kvs: []KV{
				newCssKV("width", "calc(var(--spacing) * 2)"),
				newCssKV("height", "calc(var(--spacing) * 2)"),
			},
		}))
		t.Run(scenario("size-1/2", &utilityGenerated{
			kvs: []KV{
				newCssKV("width", "calc(1/2 * 100%)"),
				newCssKV("height", "calc(1/2 * 100%)"),
			},
		}))
		t.Run(scenario("size-3/5", &utilityGenerated{
			kvs: []KV{
				newCssKV("width", "calc(3/5 * 100%)"),
				newCssKV("height", "calc(3/5 * 100%)"),
			},
		}))
		t.Run(scenario("size-full", &utilityGenerated{
			kvs: []KV{
				newCssKV("width", "100%"),
				newCssKV("height", "100%"),
			},
		}))
		t.Run(scenario("size-svw", &utilityGenerated{
			kvs: []KV{
				newCssKV("width", "100svw"),
				newCssKV("height", "100svw"),
			},
		}))
		t.Run(scenario("size-lvw", &utilityGenerated{
			kvs: []KV{
				newCssKV("width", "100lvw"),
				newCssKV("height", "100lvw"),
			},
		}))
		t.Run(scenario("size-dvw", &utilityGenerated{
			kvs: []KV{
				newCssKV("width", "100dvw"),
				newCssKV("height", "100dvw"),
			},
		}))
		t.Run(scenario("size-min", &utilityGenerated{
			kvs: []KV{
				newCssKV("width", "min-content"),
				newCssKV("height", "min-content"),
			},
		}))
		t.Run(scenario("size-max", &utilityGenerated{
			kvs: []KV{
				newCssKV("width", "max-content"),
				newCssKV("height", "max-content"),
			},
		}))
		t.Run(scenario("size-fit", &utilityGenerated{
			kvs: []KV{
				newCssKV("width", "fit-content"),
				newCssKV("height", "fit-content"),
			},
		}))
		t.Run(scenario("size-(--my-size)", &utilityGenerated{
			kvs: []KV{
				newCssKV("width", "var(--my-size)"),
				newCssKV("height", "var(--my-size)"),
			},
		}))
		t.Run(scenario("size-[5px]", &utilityGenerated{
			kvs: []KV{
				newCssKV("width", "5px"),
				newCssKV("height", "5px"),
			},
		}))

		t.Run(scenario("h-2", newUtilityGeneratedKV("height", "calc(var(--spacing) * 2)")))
		t.Run(scenario("h-1/2", newUtilityGeneratedKV("height", "calc(1/2 * 100%)")))
		t.Run(scenario("h-auto", newUtilityGeneratedKV("height", "auto")))
		t.Run(scenario("h-px", newUtilityGeneratedKV("height", "1px")))
		t.Run(scenario("h-full", newUtilityGeneratedKV("height", "100%")))
		t.Run(scenario("h-screen", newUtilityGeneratedKV("height", "100vh")))
		t.Run(scenario("h-dvh", newUtilityGeneratedKV("height", "100dvh")))
		t.Run(scenario("h-dvw", newUtilityGeneratedKV("height", "100dvw")))
		t.Run(scenario("h-lvh", newUtilityGeneratedKV("height", "100lvh")))
		t.Run(scenario("h-lvw", newUtilityGeneratedKV("height", "100lvw")))
		t.Run(scenario("h-svh", newUtilityGeneratedKV("height", "100svh")))
		t.Run(scenario("h-svw", newUtilityGeneratedKV("height", "100svw")))
		t.Run(scenario("h-min", newUtilityGeneratedKV("height", "min-content")))
		t.Run(scenario("h-max", newUtilityGeneratedKV("height", "max-content")))
		t.Run(scenario("h-fit", newUtilityGeneratedKV("height", "fit-content")))
		t.Run(scenario("h-lh", newUtilityGeneratedKV("height", "1lh")))
		t.Run(scenario("h-(--my-height)", newUtilityGeneratedKV("height", "var(--my-height)")))
		t.Run(scenario("h-[5px]", newUtilityGeneratedKV("height", "5px")))

		t.Run(scenario("min-w-0", newUtilityGeneratedKV("min-width", "calc(var(--spacing) * 0)")))
		t.Run(scenario("min-w-1/2", newUtilityGeneratedKV("min-width", "calc(1/2 * 100%)")))
		t.Run(scenario("min-w-3xs", newUtilityGeneratedKV("min-width", "var(--container-3xs)")))
		t.Run(scenario("min-w-2xs", newUtilityGeneratedKV("min-width", "var(--container-2xs)")))
		t.Run(scenario("min-w-xs", newUtilityGeneratedKV("min-width", "var(--container-xs)")))
		t.Run(scenario("min-w-sm", newUtilityGeneratedKV("min-width", "var(--container-sm)")))
		t.Run(scenario("min-w-md", newUtilityGeneratedKV("min-width", "var(--container-md)")))
		t.Run(scenario("min-w-lg", newUtilityGeneratedKV("min-width", "var(--container-lg)")))
		t.Run(scenario("min-w-xl", newUtilityGeneratedKV("min-width", "var(--container-xl)")))
		t.Run(scenario("min-w-2xl", newUtilityGeneratedKV("min-width", "var(--container-2xl)")))
		t.Run(scenario("min-w-3xl", newUtilityGeneratedKV("min-width", "var(--container-3xl)")))
		t.Run(scenario("min-w-4xl", newUtilityGeneratedKV("min-width", "var(--container-4xl)")))
		t.Run(scenario("min-w-5xl", newUtilityGeneratedKV("min-width", "var(--container-5xl)")))
		t.Run(scenario("min-w-6xl", newUtilityGeneratedKV("min-width", "var(--container-6xl)")))
		t.Run(scenario("min-w-7xl", newUtilityGeneratedKV("min-width", "var(--container-7xl)")))
		t.Run(scenario("min-w-auto", newUtilityGeneratedKV("min-width", "auto")))
		t.Run(scenario("min-w-px", newUtilityGeneratedKV("min-width", "1px")))
		t.Run(scenario("min-w-full", newUtilityGeneratedKV("min-width", "100%")))
		t.Run(scenario("min-w-screen", newUtilityGeneratedKV("min-width", "100vw")))
		t.Run(scenario("min-w-dvw", newUtilityGeneratedKV("min-width", "100dvw")))
		t.Run(scenario("min-w-dvh", newUtilityGeneratedKV("min-width", "100dvh")))
		t.Run(scenario("min-w-lvw", newUtilityGeneratedKV("min-width", "100lvw")))
		t.Run(scenario("min-w-lvh", newUtilityGeneratedKV("min-width", "100lvh")))
		t.Run(scenario("min-w-svw", newUtilityGeneratedKV("min-width", "100svw")))
		t.Run(scenario("min-w-svh", newUtilityGeneratedKV("min-width", "100svh")))
		t.Run(scenario("min-w-min", newUtilityGeneratedKV("min-width", "min-content")))
		t.Run(scenario("min-w-max", newUtilityGeneratedKV("min-width", "max-content")))
		t.Run(scenario("min-w-fit", newUtilityGeneratedKV("min-width", "fit-content")))
		t.Run(scenario("min-w-(--my-min-width)", newUtilityGeneratedKV("min-width", "var(--my-min-width)")))
		t.Run(scenario("min-w-[5px]", newUtilityGeneratedKV("min-width", "5px")))

		t.Run(scenario("max-w-2", newUtilityGeneratedKV("max-width", "calc(var(--spacing) * 2)")))
		t.Run(scenario("max-w-1/2", newUtilityGeneratedKV("max-width", "calc(1/2 * 100%)")))
		t.Run(scenario("max-w-3xs", newUtilityGeneratedKV("max-width", "var(--container-3xs)")))
		t.Run(scenario("max-w-2xs", newUtilityGeneratedKV("max-width", "var(--container-2xs)")))
		t.Run(scenario("max-w-xs", newUtilityGeneratedKV("max-width", "var(--container-xs)")))
		t.Run(scenario("max-w-sm", newUtilityGeneratedKV("max-width", "var(--container-sm)")))
		t.Run(scenario("max-w-md", newUtilityGeneratedKV("max-width", "var(--container-md)")))
		t.Run(scenario("max-w-lg", newUtilityGeneratedKV("max-width", "var(--container-lg)")))
		t.Run(scenario("max-w-xl", newUtilityGeneratedKV("max-width", "var(--container-xl)")))
		t.Run(scenario("max-w-2xl", newUtilityGeneratedKV("max-width", "var(--container-2xl)")))
		t.Run(scenario("max-w-3xl", newUtilityGeneratedKV("max-width", "var(--container-3xl)")))
		t.Run(scenario("max-w-4xl", newUtilityGeneratedKV("max-width", "var(--container-4xl)")))
		t.Run(scenario("max-w-5xl", newUtilityGeneratedKV("max-width", "var(--container-5xl)")))
		t.Run(scenario("max-w-6xl", newUtilityGeneratedKV("max-width", "var(--container-6xl)")))
		t.Run(scenario("max-w-7xl", newUtilityGeneratedKV("max-width", "var(--container-7xl)")))
		t.Run(scenario("max-w-none", newUtilityGeneratedKV("max-width", "none")))
		t.Run(scenario("max-w-px", newUtilityGeneratedKV("max-width", "1px")))
		t.Run(scenario("max-w-full", newUtilityGeneratedKV("max-width", "100%")))
		t.Run(scenario("max-w-dvw", newUtilityGeneratedKV("max-width", "100dvw")))
		t.Run(scenario("max-w-dvh", newUtilityGeneratedKV("max-width", "100dvh")))
		t.Run(scenario("max-w-lvw", newUtilityGeneratedKV("max-width", "100lvw")))
		t.Run(scenario("max-w-lvh", newUtilityGeneratedKV("max-width", "100lvh")))
		t.Run(scenario("max-w-svw", newUtilityGeneratedKV("max-width", "100svw")))
		t.Run(scenario("max-w-svh", newUtilityGeneratedKV("max-width", "100svh")))
		t.Run(scenario("max-w-screen", newUtilityGeneratedKV("max-width", "100vw")))
		t.Run(scenario("max-w-min", newUtilityGeneratedKV("max-width", "min-content")))
		t.Run(scenario("max-w-max", newUtilityGeneratedKV("max-width", "max-content")))
		t.Run(scenario("max-w-fit", newUtilityGeneratedKV("max-width", "fit-content")))
		t.Run(scenario("max-w-(--my-max-width)", newUtilityGeneratedKV("max-width", "var(--my-max-width)")))
		t.Run(scenario("max-w-[5px]", newUtilityGeneratedKV("max-width", "5px")))

		t.Run(scenario("container", &utilityGenerated{
			kvs: []KV{
				newCssKV("width", "100%"),
			},
			others: []Query{
				{
					Name: "@media (width >= 40rem)",
					KVs: []KV{
						newCssKV("max-width", "40rem"),
					},
				},
				{
					Name: "@media (width >= 48rem)",
					KVs: []KV{
						newCssKV("max-width", "48rem"),
					},
				},
				{
					Name: "@media (width >= 64rem)",
					KVs: []KV{
						newCssKV("max-width", "64rem"),
					},
				},
				{
					Name: "@media (width >= 80rem)",
					KVs: []KV{
						newCssKV("max-width", "80rem"),
					},
				},
				{
					Name: "@media (width >= 96rem)",
					KVs: []KV{
						newCssKV("max-width", "96rem"),
					},
				},
			},
		}))

		t.Run(scenario("@container", newUtilityGeneratedKV("container-type", "inline-size")))
		t.Run(scenario("@container-normal", newUtilityGeneratedKV("container-type", "normal")))
		t.Run(scenario("@container-size", newUtilityGeneratedKV("container-type", "size")))
		t.Run(scenario("@container/sidebar", &utilityGenerated{
			kvs: []KV{
				newCssKV("container-type", "inline-size"),
				newCssKV("container-name", "sidebar"),
			},
		}))
		t.Run(scenario("@container-normal/sidebar", &utilityGenerated{
			kvs: []KV{
				newCssKV("container-type", "normal"),
				newCssKV("container-name", "sidebar"),
			},
		}))

		t.Run(scenario("min-h-2", newUtilityGeneratedKV("min-height", "calc(var(--spacing) * 2)")))
		t.Run(scenario("min-h-1/2", newUtilityGeneratedKV("min-height", "calc(1/2 * 100%)")))
		t.Run(scenario("min-h-px", newUtilityGeneratedKV("min-height", "1px")))
		t.Run(scenario("min-h-full", newUtilityGeneratedKV("min-height", "100%")))
		t.Run(scenario("min-h-screen", newUtilityGeneratedKV("min-height", "100vh")))
		t.Run(scenario("min-h-dvh", newUtilityGeneratedKV("min-height", "100dvh")))
		t.Run(scenario("min-h-dvw", newUtilityGeneratedKV("min-height", "100dvw")))
		t.Run(scenario("min-h-lvh", newUtilityGeneratedKV("min-height", "100lvh")))
		t.Run(scenario("min-h-lvw", newUtilityGeneratedKV("min-height", "100lvw")))
		t.Run(scenario("min-h-svw", newUtilityGeneratedKV("min-height", "100svw")))
		t.Run(scenario("min-h-svh", newUtilityGeneratedKV("min-height", "100svh")))
		t.Run(scenario("min-h-auto", newUtilityGeneratedKV("min-height", "auto")))
		t.Run(scenario("min-h-min", newUtilityGeneratedKV("min-height", "min-content")))
		t.Run(scenario("min-h-max", newUtilityGeneratedKV("min-height", "max-content")))
		t.Run(scenario("min-h-fit", newUtilityGeneratedKV("min-height", "fit-content")))
		t.Run(scenario("min-h-lh", newUtilityGeneratedKV("min-height", "1lh")))
		t.Run(scenario("min-h-(--my-min-height)", newUtilityGeneratedKV("min-height", "var(--my-min-height)")))
		t.Run(scenario("min-h-[5px]", newUtilityGeneratedKV("min-height", "5px")))

		t.Run(scenario("max-h-2", newUtilityGeneratedKV("max-height", "calc(var(--spacing) * 2)")))
		t.Run(scenario("max-h-1/2", newUtilityGeneratedKV("max-height", "calc(1/2 * 100%)")))
		t.Run(scenario("max-h-none", newUtilityGeneratedKV("max-height", "none")))
		t.Run(scenario("max-h-px", newUtilityGeneratedKV("max-height", "1px")))
		t.Run(scenario("max-h-full", newUtilityGeneratedKV("max-height", "100%")))
		t.Run(scenario("max-h-screen", newUtilityGeneratedKV("max-height", "100vh")))
		t.Run(scenario("max-h-dvh", newUtilityGeneratedKV("max-height", "100dvh")))
		t.Run(scenario("max-h-dvw", newUtilityGeneratedKV("max-height", "100dvw")))
		t.Run(scenario("max-h-lvh", newUtilityGeneratedKV("max-height", "100lvh")))
		t.Run(scenario("max-h-lvw", newUtilityGeneratedKV("max-height", "100lvw")))
		t.Run(scenario("max-h-svh", newUtilityGeneratedKV("max-height", "100svh")))
		t.Run(scenario("max-h-svw", newUtilityGeneratedKV("max-height", "100svw")))
		t.Run(scenario("max-h-min", newUtilityGeneratedKV("max-height", "min-content")))
		t.Run(scenario("max-h-max", newUtilityGeneratedKV("max-height", "max-content")))
		t.Run(scenario("max-h-fit", newUtilityGeneratedKV("max-height", "fit-content")))
		t.Run(scenario("max-h-lh", newUtilityGeneratedKV("max-height", "1lh")))
		t.Run(scenario("max-h-(--my-max-height)", newUtilityGeneratedKV("max-height", "var(--my-max-height)")))
		t.Run(scenario("max-h-[5px]", newUtilityGeneratedKV("max-height", "5px")))

		t.Run(scenario("inline-2", newUtilityGeneratedKV("inline-size", "calc(var(--spacing) * 2)")))
		t.Run(scenario("inline-1/2", newUtilityGeneratedKV("inline-size", "calc(1/2 * 100%)")))
		t.Run(scenario("inline-3xs", newUtilityGeneratedKV("inline-size", "var(--container-3xs)")))
		t.Run(scenario("inline-2xs", newUtilityGeneratedKV("inline-size", "var(--container-2xs)")))
		t.Run(scenario("inline-xs", newUtilityGeneratedKV("inline-size", "var(--container-xs)")))
		t.Run(scenario("inline-sm", newUtilityGeneratedKV("inline-size", "var(--container-sm)")))
		t.Run(scenario("inline-md", newUtilityGeneratedKV("inline-size", "var(--container-md)")))
		t.Run(scenario("inline-lg", newUtilityGeneratedKV("inline-size", "var(--container-lg)")))
		t.Run(scenario("inline-xl", newUtilityGeneratedKV("inline-size", "var(--container-xl)")))
		t.Run(scenario("inline-2xl", newUtilityGeneratedKV("inline-size", "var(--container-2xl)")))
		t.Run(scenario("inline-3xl", newUtilityGeneratedKV("inline-size", "var(--container-3xl)")))
		t.Run(scenario("inline-4xl", newUtilityGeneratedKV("inline-size", "var(--container-4xl)")))
		t.Run(scenario("inline-5xl", newUtilityGeneratedKV("inline-size", "var(--container-5xl)")))
		t.Run(scenario("inline-6xl", newUtilityGeneratedKV("inline-size", "var(--container-6xl)")))
		t.Run(scenario("inline-7xl", newUtilityGeneratedKV("inline-size", "var(--container-7xl)")))
		t.Run(scenario("inline-auto", newUtilityGeneratedKV("inline-size", "auto")))
		t.Run(scenario("inline-px", newUtilityGeneratedKV("inline-size", "1px")))
		t.Run(scenario("inline-full", newUtilityGeneratedKV("inline-size", "100%")))
		t.Run(scenario("inline-screen", newUtilityGeneratedKV("inline-size", "100vw")))
		t.Run(scenario("inline-dvw", newUtilityGeneratedKV("inline-size", "100dvw")))
		t.Run(scenario("inline-dvh", newUtilityGeneratedKV("inline-size", "100dvh")))
		t.Run(scenario("inline-lvw", newUtilityGeneratedKV("inline-size", "100lvw")))
		t.Run(scenario("inline-lvh", newUtilityGeneratedKV("inline-size", "100lvh")))
		t.Run(scenario("inline-svw", newUtilityGeneratedKV("inline-size", "100svw")))
		t.Run(scenario("inline-svh", newUtilityGeneratedKV("inline-size", "100svh")))
		t.Run(scenario("inline-min", newUtilityGeneratedKV("inline-size", "min-content")))
		t.Run(scenario("inline-max", newUtilityGeneratedKV("inline-size", "max-content")))
		t.Run(scenario("inline-fit", newUtilityGeneratedKV("inline-size", "fit-content")))
		t.Run(scenario("inline-(--my-inline-size)", newUtilityGeneratedKV("inline-size", "var(--my-inline-size)")))
		t.Run(scenario("inline-[5px]", newUtilityGeneratedKV("inline-size", "5px")))

		t.Run(scenario("block-2", newUtilityGeneratedKV("block-size", "calc(var(--spacing) * 2)")))
		t.Run(scenario("block-1/2", newUtilityGeneratedKV("block-size", "calc(1/2 * 100%)")))
		t.Run(scenario("block-auto", newUtilityGeneratedKV("block-size", "auto")))
		t.Run(scenario("block-px", newUtilityGeneratedKV("block-size", "1px")))
		t.Run(scenario("block-full", newUtilityGeneratedKV("block-size", "100%")))
		t.Run(scenario("block-screen", newUtilityGeneratedKV("block-size", "100vh")))
		t.Run(scenario("block-dvh", newUtilityGeneratedKV("block-size", "100dvh")))
		t.Run(scenario("block-dvw", newUtilityGeneratedKV("block-size", "100dvw")))
		t.Run(scenario("block-lvh", newUtilityGeneratedKV("block-size", "100lvh")))
		t.Run(scenario("block-lvw", newUtilityGeneratedKV("block-size", "100lvw")))
		t.Run(scenario("block-svh", newUtilityGeneratedKV("block-size", "100svh")))
		t.Run(scenario("block-svw", newUtilityGeneratedKV("block-size", "100svw")))
		t.Run(scenario("block-min", newUtilityGeneratedKV("block-size", "min-content")))
		t.Run(scenario("block-max", newUtilityGeneratedKV("block-size", "max-content")))
		t.Run(scenario("block-fit", newUtilityGeneratedKV("block-size", "fit-content")))
		t.Run(scenario("block-lh", newUtilityGeneratedKV("block-size", "1lh")))
		t.Run(scenario("block-(--my-block-size)", newUtilityGeneratedKV("block-size", "var(--my-block-size)")))
		t.Run(scenario("block-[5px]", newUtilityGeneratedKV("block-size", "5px")))

		t.Run(scenario("min-inline-2", newUtilityGeneratedKV("min-inline-size", "calc(var(--spacing) * 2)")))
		t.Run(scenario("min-inline-1/2", newUtilityGeneratedKV("min-inline-size", "calc(1/2 * 100%)")))
		t.Run(scenario("min-inline-3xs", newUtilityGeneratedKV("min-inline-size", "var(--container-3xs)")))
		t.Run(scenario("min-inline-2xs", newUtilityGeneratedKV("min-inline-size", "var(--container-2xs)")))
		t.Run(scenario("min-inline-xs", newUtilityGeneratedKV("min-inline-size", "var(--container-xs)")))
		t.Run(scenario("min-inline-sm", newUtilityGeneratedKV("min-inline-size", "var(--container-sm)")))
		t.Run(scenario("min-inline-md", newUtilityGeneratedKV("min-inline-size", "var(--container-md)")))
		t.Run(scenario("min-inline-lg", newUtilityGeneratedKV("min-inline-size", "var(--container-lg)")))
		t.Run(scenario("min-inline-xl", newUtilityGeneratedKV("min-inline-size", "var(--container-xl)")))
		t.Run(scenario("min-inline-2xl", newUtilityGeneratedKV("min-inline-size", "var(--container-2xl)")))
		t.Run(scenario("min-inline-3xl", newUtilityGeneratedKV("min-inline-size", "var(--container-3xl)")))
		t.Run(scenario("min-inline-4xl", newUtilityGeneratedKV("min-inline-size", "var(--container-4xl)")))
		t.Run(scenario("min-inline-5xl", newUtilityGeneratedKV("min-inline-size", "var(--container-5xl)")))
		t.Run(scenario("min-inline-6xl", newUtilityGeneratedKV("min-inline-size", "var(--container-6xl)")))
		t.Run(scenario("min-inline-7xl", newUtilityGeneratedKV("min-inline-size", "var(--container-7xl)")))
		t.Run(scenario("min-inline-auto", newUtilityGeneratedKV("min-inline-size", "auto")))
		t.Run(scenario("min-inline-px", newUtilityGeneratedKV("min-inline-size", "1px")))
		t.Run(scenario("min-inline-full", newUtilityGeneratedKV("min-inline-size", "100%")))
		t.Run(scenario("min-inline-screen", newUtilityGeneratedKV("min-inline-size", "100vw")))
		t.Run(scenario("min-inline-dvw", newUtilityGeneratedKV("min-inline-size", "100dvw")))
		t.Run(scenario("min-inline-dvh", newUtilityGeneratedKV("min-inline-size", "100dvh")))
		t.Run(scenario("min-inline-lvw", newUtilityGeneratedKV("min-inline-size", "100lvw")))
		t.Run(scenario("min-inline-lvh", newUtilityGeneratedKV("min-inline-size", "100lvh")))
		t.Run(scenario("min-inline-svw", newUtilityGeneratedKV("min-inline-size", "100svw")))
		t.Run(scenario("min-inline-svh", newUtilityGeneratedKV("min-inline-size", "100svh")))
		t.Run(scenario("min-inline-min", newUtilityGeneratedKV("min-inline-size", "min-content")))
		t.Run(scenario("min-inline-max", newUtilityGeneratedKV("min-inline-size", "max-content")))
		t.Run(scenario("min-inline-fit", newUtilityGeneratedKV("min-inline-size", "fit-content")))
		t.Run(scenario("min-inline-(--my-min-inline-size)", newUtilityGeneratedKV("min-inline-size", "var(--my-min-inline-size)")))
		t.Run(scenario("min-inline-[5px]", newUtilityGeneratedKV("min-inline-size", "5px")))

		t.Run(scenario("max-inline-2", newUtilityGeneratedKV("max-inline-size", "calc(var(--spacing) * 2)")))
		t.Run(scenario("max-inline-1/2", newUtilityGeneratedKV("max-inline-size", "calc(1/2 * 100%)")))
		t.Run(scenario("max-inline-3xs", newUtilityGeneratedKV("max-inline-size", "var(--container-3xs)")))
		t.Run(scenario("max-inline-2xs", newUtilityGeneratedKV("max-inline-size", "var(--container-2xs)")))
		t.Run(scenario("max-inline-xs", newUtilityGeneratedKV("max-inline-size", "var(--container-xs)")))
		t.Run(scenario("max-inline-sm", newUtilityGeneratedKV("max-inline-size", "var(--container-sm)")))
		t.Run(scenario("max-inline-md", newUtilityGeneratedKV("max-inline-size", "var(--container-md)")))
		t.Run(scenario("max-inline-lg", newUtilityGeneratedKV("max-inline-size", "var(--container-lg)")))
		t.Run(scenario("max-inline-xl", newUtilityGeneratedKV("max-inline-size", "var(--container-xl)")))
		t.Run(scenario("max-inline-2xl", newUtilityGeneratedKV("max-inline-size", "var(--container-2xl)")))
		t.Run(scenario("max-inline-3xl", newUtilityGeneratedKV("max-inline-size", "var(--container-3xl)")))
		t.Run(scenario("max-inline-4xl", newUtilityGeneratedKV("max-inline-size", "var(--container-4xl)")))
		t.Run(scenario("max-inline-5xl", newUtilityGeneratedKV("max-inline-size", "var(--container-5xl)")))
		t.Run(scenario("max-inline-6xl", newUtilityGeneratedKV("max-inline-size", "var(--container-6xl)")))
		t.Run(scenario("max-inline-7xl", newUtilityGeneratedKV("max-inline-size", "var(--container-7xl)")))
		t.Run(scenario("max-inline-none", newUtilityGeneratedKV("max-inline-size", "none")))
		t.Run(scenario("max-inline-px", newUtilityGeneratedKV("max-inline-size", "1px")))
		t.Run(scenario("max-inline-full", newUtilityGeneratedKV("max-inline-size", "100%")))
		t.Run(scenario("max-inline-dvw", newUtilityGeneratedKV("max-inline-size", "100dvw")))
		t.Run(scenario("max-inline-dvh", newUtilityGeneratedKV("max-inline-size", "100dvh")))
		t.Run(scenario("max-inline-lvw", newUtilityGeneratedKV("max-inline-size", "100lvw")))
		t.Run(scenario("max-inline-lvh", newUtilityGeneratedKV("max-inline-size", "100lvh")))
		t.Run(scenario("max-inline-svw", newUtilityGeneratedKV("max-inline-size", "100svw")))
		t.Run(scenario("max-inline-svh", newUtilityGeneratedKV("max-inline-size", "100svh")))
		t.Run(scenario("max-inline-screen", newUtilityGeneratedKV("max-inline-size", "100vw")))
		t.Run(scenario("max-inline-min", newUtilityGeneratedKV("max-inline-size", "min-content")))
		t.Run(scenario("max-inline-max", newUtilityGeneratedKV("max-inline-size", "max-content")))
		t.Run(scenario("max-inline-fit", newUtilityGeneratedKV("max-inline-size", "fit-content")))
		t.Run(scenario("max-inline-(--my-max-inline-size)", newUtilityGeneratedKV("max-inline-size", "var(--my-max-inline-size)")))
		t.Run(scenario("max-inline-[5px]", newUtilityGeneratedKV("max-inline-size", "5px")))

		t.Run(scenario("min-block-2", newUtilityGeneratedKV("min-block-size", "calc(var(--spacing) * 2)")))
		t.Run(scenario("min-block-1/2", newUtilityGeneratedKV("min-block-size", "calc(1/2 * 100%)")))
		t.Run(scenario("min-block-px", newUtilityGeneratedKV("min-block-size", "1px")))
		t.Run(scenario("min-block-full", newUtilityGeneratedKV("min-block-size", "100%")))
		t.Run(scenario("min-block-screen", newUtilityGeneratedKV("min-block-size", "100vh")))
		t.Run(scenario("min-block-dvh", newUtilityGeneratedKV("min-block-size", "100dvh")))
		t.Run(scenario("min-block-dvw", newUtilityGeneratedKV("min-block-size", "100dvw")))
		t.Run(scenario("min-block-lvh", newUtilityGeneratedKV("min-block-size", "100lvh")))
		t.Run(scenario("min-block-lvw", newUtilityGeneratedKV("min-block-size", "100lvw")))
		t.Run(scenario("min-block-svw", newUtilityGeneratedKV("min-block-size", "100svw")))
		t.Run(scenario("min-block-svh", newUtilityGeneratedKV("min-block-size", "100svh")))
		t.Run(scenario("min-block-auto", newUtilityGeneratedKV("min-block-size", "auto")))
		t.Run(scenario("min-block-min", newUtilityGeneratedKV("min-block-size", "min-content")))
		t.Run(scenario("min-block-max", newUtilityGeneratedKV("min-block-size", "max-content")))
		t.Run(scenario("min-block-fit", newUtilityGeneratedKV("min-block-size", "fit-content")))
		t.Run(scenario("min-block-lh", newUtilityGeneratedKV("min-block-size", "1lh")))
		t.Run(scenario("min-block-(--my-min-block-size)", newUtilityGeneratedKV("min-block-size", "var(--my-min-block-size)")))
		t.Run(scenario("min-block-[5px]", newUtilityGeneratedKV("min-block-size", "5px")))

		t.Run(scenario("max-block-2", newUtilityGeneratedKV("max-block-size", "calc(var(--spacing) * 2)")))
		t.Run(scenario("max-block-1/2", newUtilityGeneratedKV("max-block-size", "calc(1/2 * 100%)")))
		t.Run(scenario("max-block-none", newUtilityGeneratedKV("max-block-size", "none")))
		t.Run(scenario("max-block-px", newUtilityGeneratedKV("max-block-size", "1px")))
		t.Run(scenario("max-block-full", newUtilityGeneratedKV("max-block-size", "100%")))
		t.Run(scenario("max-block-screen", newUtilityGeneratedKV("max-block-size", "100vh")))
		t.Run(scenario("max-block-dvh", newUtilityGeneratedKV("max-block-size", "100dvh")))
		t.Run(scenario("max-block-dvw", newUtilityGeneratedKV("max-block-size", "100dvw")))
		t.Run(scenario("max-block-lvh", newUtilityGeneratedKV("max-block-size", "100lvh")))
		t.Run(scenario("max-block-lvw", newUtilityGeneratedKV("max-block-size", "100lvw")))
		t.Run(scenario("max-block-svh", newUtilityGeneratedKV("max-block-size", "100svh")))
		t.Run(scenario("max-block-svw", newUtilityGeneratedKV("max-block-size", "100svw")))
		t.Run(scenario("max-block-min", newUtilityGeneratedKV("max-block-size", "min-content")))
		t.Run(scenario("max-block-max", newUtilityGeneratedKV("max-block-size", "max-content")))
		t.Run(scenario("max-block-fit", newUtilityGeneratedKV("max-block-size", "fit-content")))
		t.Run(scenario("max-block-lh", newUtilityGeneratedKV("max-block-size", "1lh")))
		t.Run(scenario("max-block-(--my-max-block-size)", newUtilityGeneratedKV("max-block-size", "var(--my-max-block-size)")))
		t.Run(scenario("max-block-[5px]", newUtilityGeneratedKV("max-block-size", "5px")))

		t.Run(scenario("font-sans", newUtilityGeneratedKV("font-family", "var(--font-sans)")))
		t.Run(scenario("font-serif", newUtilityGeneratedKV("font-family", "var(--font-serif)")))
		t.Run(scenario("font-mono", newUtilityGeneratedKV("font-family", "var(--font-mono)")))
		t.Run(scenario("font-(family-name:--my-font)", newUtilityGeneratedKV("font-family", "var(--my-font)")))
		t.Run(scenario("font-[Arial]", newUtilityGeneratedKV("font-family", "Arial")))

		t.Run(scenario("font-(--my-weight)", newUtilityGeneratedKV("font-weight", "var(--my-weight)")))

		t.Run(scenario("text-xs", &utilityGenerated{
			kvs: []KV{
				newCssKV("font-size", "var(--text-xs)"),
				newCssKV("line-height", "var(--tw-leading, var(--text-xs--line-height))"),
			},
		}))
		t.Run(scenario("text-sm", &utilityGenerated{
			kvs: []KV{
				newCssKV("font-size", "var(--text-sm)"),
				newCssKV("line-height", "var(--tw-leading, var(--text-sm--line-height))"),
			},
		}))
		t.Run(scenario("text-base", &utilityGenerated{
			kvs: []KV{
				newCssKV("font-size", "var(--text-base)"),
				newCssKV("line-height", "var(--tw-leading, var(--text-base--line-height))"),
			},
		}))
		t.Run(scenario("text-lg", &utilityGenerated{
			kvs: []KV{
				newCssKV("font-size", "var(--text-lg)"),
				newCssKV("line-height", "var(--tw-leading, var(--text-lg--line-height))"),
			},
		}))
		t.Run(scenario("text-xl", &utilityGenerated{
			kvs: []KV{
				newCssKV("font-size", "var(--text-xl)"),
				newCssKV("line-height", "var(--tw-leading, var(--text-xl--line-height))"),
			},
		}))
		t.Run(scenario("text-2xl", &utilityGenerated{
			kvs: []KV{
				newCssKV("font-size", "var(--text-2xl)"),
				newCssKV("line-height", "var(--tw-leading, var(--text-2xl--line-height))"),
			},
		}))
		t.Run(scenario("text-3xl", &utilityGenerated{
			kvs: []KV{
				newCssKV("font-size", "var(--text-3xl)"),
				newCssKV("line-height", "var(--tw-leading, var(--text-3xl--line-height))"),
			},
		}))
		t.Run(scenario("text-4xl", &utilityGenerated{
			kvs: []KV{
				newCssKV("font-size", "var(--text-4xl)"),
				newCssKV("line-height", "var(--tw-leading, var(--text-4xl--line-height))"),
			},
		}))
		t.Run(scenario("text-5xl", &utilityGenerated{
			kvs: []KV{
				newCssKV("font-size", "var(--text-5xl)"),
				newCssKV("line-height", "var(--tw-leading, var(--text-5xl--line-height))"),
			},
		}))
		t.Run(scenario("text-6xl", &utilityGenerated{
			kvs: []KV{
				newCssKV("font-size", "var(--text-6xl)"),
				newCssKV("line-height", "var(--tw-leading, var(--text-6xl--line-height))"),
			},
		}))
		t.Run(scenario("text-7xl", &utilityGenerated{
			kvs: []KV{
				newCssKV("font-size", "var(--text-7xl)"),
				newCssKV("line-height", "var(--tw-leading, var(--text-7xl--line-height))"),
			},
		}))
		t.Run(scenario("text-8xl", &utilityGenerated{
			kvs: []KV{
				newCssKV("font-size", "var(--text-8xl)"),
				newCssKV("line-height", "var(--tw-leading, var(--text-8xl--line-height))"),
			},
		}))
		t.Run(scenario("text-9xl", &utilityGenerated{
			kvs: []KV{
				newCssKV("font-size", "var(--text-9xl)"),
				newCssKV("line-height", "var(--tw-leading, var(--text-9xl--line-height))"),
			},
		}))
		t.Run(scenario("text-(length:--my-text-size)", newUtilityGeneratedKV("font-size", "var(--my-text-size)")))
		t.Run(scenario("text-[32px]", newUtilityGeneratedKV("font-size", "32px")))

		t.Run(scenario("antialiased", &utilityGenerated{
			kvs: []KV{
				newCssKV("-webkit-font-smoothing", "antialiased"),
				newCssKV("-moz-osx-font-smoothing", "grayscale"),
			},
		}))

		t.Run(scenario("subpixel-antialiased", &utilityGenerated{
			kvs: []KV{
				newCssKV("-webkit-font-smoothing", "auto"),
				newCssKV("-moz-osx-font-smoothing", "auto"),
			},
		}))

		t.Run(scenario("italic", newUtilityGeneratedKV("font-style", "italic")))
		t.Run(scenario("not-italic", newUtilityGeneratedKV("font-style", "normal")))

		t.Run(scenario("font-thin", newUtilityGeneratedKV("font-weight", "var(--font-weight-thin)")))
		t.Run(scenario("font-extralight", newUtilityGeneratedKV("font-weight", "var(--font-weight-extralight)")))
		t.Run(scenario("font-light", newUtilityGeneratedKV("font-weight", "var(--font-weight-light)")))
		t.Run(scenario("font-normal", newUtilityGeneratedKV("font-weight", "var(--font-weight-normal)")))
		t.Run(scenario("font-medium", newUtilityGeneratedKV("font-weight", "var(--font-weight-medium)")))
		t.Run(scenario("font-semibold", newUtilityGeneratedKV("font-weight", "var(--font-weight-semibold)")))
		t.Run(scenario("font-bold", newUtilityGeneratedKV("font-weight", "var(--font-weight-bold)")))
		t.Run(scenario("font-extrabold", newUtilityGeneratedKV("font-weight", "var(--font-weight-extrabold)")))
		t.Run(scenario("font-black", newUtilityGeneratedKV("font-weight", "var(--font-weight-black)")))
		t.Run(scenario("font-(--my-font-weight)", newUtilityGeneratedKV("font-weight", "var(--my-font-weight)")))
		t.Run(scenario("font-[350]", newUtilityGeneratedKV("font-weight", "350")))

		t.Run(scenario("font-stretch-ultra-condensed", newUtilityGeneratedKV("font-stretch", "ultra-condensed")))
		t.Run(scenario("font-stretch-extra-condensed", newUtilityGeneratedKV("font-stretch", "extra-condensed")))
		t.Run(scenario("font-stretch-condensed", newUtilityGeneratedKV("font-stretch", "condensed")))
		t.Run(scenario("font-stretch-semi-condensed", newUtilityGeneratedKV("font-stretch", "semi-condensed")))
		t.Run(scenario("font-stretch-normal", newUtilityGeneratedKV("font-stretch", "normal")))
		t.Run(scenario("font-stretch-semi-expanded", newUtilityGeneratedKV("font-stretch", "semi-expanded")))
		t.Run(scenario("font-stretch-expanded", newUtilityGeneratedKV("font-stretch", "expanded")))
		t.Run(scenario("font-stretch-extra-expanded", newUtilityGeneratedKV("font-stretch", "extra-expanded")))
		t.Run(scenario("font-stretch-ultra-expanded", newUtilityGeneratedKV("font-stretch", "ultra-expanded")))
		t.Run(scenario("font-stretch-(--my-font-stretch)", newUtilityGeneratedKV("font-stretch", "var(--my-font-stretch)")))
		t.Run(scenario("font-stretch-[custom]", newUtilityGeneratedKV("font-stretch", "custom")))
		t.Run(scenario("font-stretch-50%", newUtilityGeneratedKV("font-stretch", "50%")))
		t.Run(scenario("font-stretch-200%", newUtilityGeneratedKV("font-stretch", "200%")))

		t.Run(scenario("normal-nums", newUtilityGeneratedKV("font-variant-numeric", "normal")))
		t.Run(scenario("ordinal", newUtilityGeneratedKV("font-variant-numeric", "ordinal")))
		t.Run(scenario("slashed-zero", newUtilityGeneratedKV("font-variant-numeric", "slashed-zero")))
		t.Run(scenario("lining-nums", newUtilityGeneratedKV("font-variant-numeric", "lining-nums")))
		t.Run(scenario("oldstyle-nums", newUtilityGeneratedKV("font-variant-numeric", "oldstyle-nums")))
		t.Run(scenario("proportional-nums", newUtilityGeneratedKV("font-variant-numeric", "proportional-nums")))
		t.Run(scenario("tabular-nums", newUtilityGeneratedKV("font-variant-numeric", "tabular-nums")))
		t.Run(scenario("diagonal-fractions", newUtilityGeneratedKV("font-variant-numeric", "diagonal-fractions")))
		t.Run(scenario("stacked-fractions", newUtilityGeneratedKV("font-variant-numeric", "stacked-fractions")))

		t.Run(scenario("font-features-['liga']", newUtilityGeneratedKV("font-feature-settings", "'liga'")))
		t.Run(scenario("font-features-['smcp','onum']", newUtilityGeneratedKV("font-feature-settings", "'smcp', 'onum'")))
		t.Run(scenario("font-features-(--my-font-features)", newUtilityGeneratedKV("font-feature-settings", "var(--my-font-features)")))

		t.Run(scenario("tracking-tighter", newUtilityGeneratedKV("letter-spacing", "var(--tracking-tighter)")))
		t.Run(scenario("tracking-tight", newUtilityGeneratedKV("letter-spacing", "var(--tracking-tight)")))
		t.Run(scenario("tracking-normal", newUtilityGeneratedKV("letter-spacing", "var(--tracking-normal)")))
		t.Run(scenario("tracking-wide", newUtilityGeneratedKV("letter-spacing", "var(--tracking-wide)")))
		t.Run(scenario("tracking-wider", newUtilityGeneratedKV("letter-spacing", "var(--tracking-wider)")))
		t.Run(scenario("tracking-widest", newUtilityGeneratedKV("letter-spacing", "var(--tracking-widest)")))
		t.Run(scenario("tracking-(--my-tracking)", newUtilityGeneratedKV("letter-spacing", "var(--my-tracking)")))
		t.Run(scenario("tracking-[0.2em]", newUtilityGeneratedKV("letter-spacing", "0.2em")))

		t.Run(scenario("line-clamp-3", &utilityGenerated{
			kvs: []KV{
				newCssKV("overflow", "hidden"),
				newCssKV("display", "-webkit-box"),
				newCssKV("-webkit-box-orient", "vertical"),
				newCssKV("-webkit-line-clamp", "3"),
			},
		}))

		t.Run(scenario("line-clamp-none", &utilityGenerated{
			kvs: []KV{
				newCssKV("overflow", "visible"),
				newCssKV("display", "block"),
				newCssKV("-webkit-box-orient", "horizontal"),
				newCssKV("-webkit-line-clamp", "unset"),
			},
		}))

		t.Run(scenario("line-clamp-(--my-line-clamp)", &utilityGenerated{
			kvs: []KV{
				newCssKV("overflow", "hidden"),
				newCssKV("display", "-webkit-box"),
				newCssKV("-webkit-box-orient", "vertical"),
				newCssKV("-webkit-line-clamp", "var(--my-line-clamp)"),
			},
		}))

		t.Run(scenario("line-clamp-[7]", &utilityGenerated{
			kvs: []KV{
				newCssKV("overflow", "hidden"),
				newCssKV("display", "-webkit-box"),
				newCssKV("-webkit-box-orient", "vertical"),
				newCssKV("-webkit-line-clamp", "7"),
			},
		}))

		t.Run(scenario("text-sm/6", &utilityGenerated{
			kvs: []KV{
				newCssKV("font-size", "var(--text-sm)"),
				newCssKV("line-height", "calc(var(--spacing) * 6)"),
			},
		}))

		t.Run(scenario("text-sm/(--my-line-height)", &utilityGenerated{
			kvs: []KV{
				newCssKV("font-size", "var(--text-sm)"),
				newCssKV("line-height", "var(--my-line-height)"),
			},
		}))

		t.Run(scenario("text-sm/[32px]", &utilityGenerated{
			kvs: []KV{
				newCssKV("font-size", "var(--text-sm)"),
				newCssKV("line-height", "32px"),
			},
		}))

		t.Run(scenario("leading-none", &utilityGenerated{
			kvs: []KV{
				newCssKV("--tw-leading", "1"),
				newCssKV("line-height", "1"),
			},
		}))
		t.Run(scenario("leading-6", &utilityGenerated{
			kvs: []KV{
				newCssKV("--tw-leading", "calc(var(--spacing) * 6)"),
				newCssKV("line-height", "calc(var(--spacing) * 6)"),
			},
		}))
		t.Run(scenario("leading-(--my-line-height)", &utilityGenerated{
			kvs: []KV{
				newCssKV("--tw-leading", "var(--my-line-height)"),
				newCssKV("line-height", "var(--my-line-height)"),
			},
		}))
		t.Run(scenario("leading-[32px]", &utilityGenerated{
			kvs: []KV{
				newCssKV("--tw-leading", "32px"),
				newCssKV("line-height", "32px"),
			},
		}))

		t.Run(scenario("list-image-none", newUtilityGeneratedKV("list-style-image", "none")))
		t.Run(scenario("list-image-(--my-list-image)", newUtilityGeneratedKV("list-style-image", "var(--my-list-image)")))
		t.Run(scenario("list-image-[url('/bullet.svg')]", newUtilityGeneratedKV("list-style-image", "url('/bullet.svg')")))
		t.Run(scenario("list-image-[linear-gradient(red,blue)]", newUtilityGeneratedKV("list-style-image", "linear-gradient(red, blue)")))
		t.Run(scenario("list-image-[\"✓\"]", newUtilityGeneratedKV("list-style-image", "\"✓\"")))
		t.Run(scenario("list-image-[var(--marker-image)]", newUtilityGeneratedKV("list-style-image", "var(--marker-image)")))
		t.Run(scenario("list-inside", newUtilityGeneratedKV("list-style-position", "inside")))
		t.Run(scenario("list-outside", newUtilityGeneratedKV("list-style-position", "outside")))
		t.Run(scenario("list-disc", newUtilityGeneratedKV("list-style-type", "disc")))
		t.Run(scenario("list-decimal", newUtilityGeneratedKV("list-style-type", "decimal")))
		t.Run(scenario("list-none", newUtilityGeneratedKV("list-style-type", "none")))
		t.Run(scenario("list-(--my-list-style)", newUtilityGeneratedKV("list-style-type", "var(--my-list-style)")))
		t.Run(scenario("list-[square]", newUtilityGeneratedKV("list-style-type", "square")))

		t.Run(scenario("text-left", newUtilityGeneratedKV("text-align", "left")))
		t.Run(scenario("text-center", newUtilityGeneratedKV("text-align", "center")))
		t.Run(scenario("text-right", newUtilityGeneratedKV("text-align", "right")))
		t.Run(scenario("text-justify", newUtilityGeneratedKV("text-align", "justify")))
		t.Run(scenario("text-start", newUtilityGeneratedKV("text-align", "start")))
		t.Run(scenario("text-end", newUtilityGeneratedKV("text-align", "end")))

		t.Run(scenario("text-red-500", newUtilityGeneratedKV("color", "var(--color-red-500)")))
		t.Run(scenario("text-current", newUtilityGeneratedKV("color", "currentcolor")))
		t.Run(scenario("text-inherit", newUtilityGeneratedKV("color", "inherit")))
		t.Run(scenario("text-transparent", newUtilityGeneratedKV("color", "transparent")))
		t.Run(scenario("text-blue-500", newUtilityGeneratedKV("color", "var(--color-blue-500)")))
		t.Run(scenario("text-black", newUtilityGeneratedKV("color", "var(--color-black)")))
		t.Run(scenario("text-white", newUtilityGeneratedKV("color", "var(--color-white)")))
		t.Run(scenario("text-(--my-text-color)", newUtilityGeneratedKV("color", "var(--my-text-color)")))
		t.Run(scenario("text-[#123456]", newUtilityGeneratedKV("color", "#123456")))
		t.Run(scenario("text-[rgb(255,0,0)]", newUtilityGeneratedKV("color", "rgb(255, 0, 0)")))

		t.Run(scenario("text-shadow-sm",
			newUtilityGeneratedKV("text-shadow",
				"0px 1px 0px var(--tw-text-shadow-color, rgb(0 0 0 / 0.075)), 0px 1px 1px var(--tw-text-shadow-color, rgb(0 0 0 / 0.075)), 0px 2px 2px var(--tw-text-shadow-color, rgb(0 0 0 / 0.075))"),
		))
		t.Run(scenario("text-shadow-md/50",
			&utilityGenerated{
				kvs: []KV{
					newCssKV("--tw-text-shadow-alpha", "50%"),
					newCssKV("text-shadow", "0px 1px 1px var(--tw-text-shadow-color, oklab(from rgb(0 0 0 / 0.1) l a b / 50%)), 0px 1px 2px var(--tw-text-shadow-color, oklab(from rgb(0 0 0 / 0.1) l a b / 50%)), 0px 2px 4px var(--tw-text-shadow-color, oklab(from rgb(0 0 0 / 0.1) l a b / 50%))"),
				},
			},
		))
		t.Run(scenario("text-shadow-none",
			newUtilityGeneratedKV("text-shadow", "none"),
		))
		t.Run(scenario("text-shadow-red-500",
			newUtilityGeneratedKV(
				"--tw-text-shadow-color",
				"color-mix(in oklab, var(--color-red-500) var(--tw-text-shadow-alpha), transparent)",
			),
		))
		t.Run(scenario("text-shadow-red-500/50",
			newUtilityGeneratedKV(
				"--tw-text-shadow-color",
				"color-mix(in oklab, color-mix(in oklab, var(--color-red-500) 50%, transparent) var(--tw-text-shadow-alpha), transparent)",
			),
		))
		t.Run(scenario("text-shadow-inherit",
			newUtilityGeneratedKV("--tw-text-shadow-color", "inherit"),
		))
		t.Run(scenario("text-shadow-transparent",
			newUtilityGeneratedKV("--tw-text-shadow-color", "transparent"),
		))
		t.Run(scenario("text-shadow-current",
			newUtilityGeneratedKV("--tw-text-shadow-color", "currentcolor"),
		))
		t.Run(scenario("text-shadow-(color:--my-ts)",
			newUtilityGeneratedKV(
				"--tw-text-shadow-color",
				"color-mix(in oklab, var(--my-ts) var(--tw-text-shadow-alpha), transparent)",
			),
		))
		t.Run(scenario("text-shadow-(--my-ts)",
			newUtilityGeneratedKV("text-shadow", "var(--my-ts)"),
		))
		t.Run(scenario("text-shadow-[1px_1px_red]",
			newUtilityGeneratedKV("text-shadow", "1px 1px red"),
		))
		t.Run(scenario("text-shadow-[color:#ff0000]/50",
			newUtilityGeneratedKV(
				"--tw-text-shadow-color",
				"color-mix(in oklab, color-mix(in oklab, #ff0000 50%, transparent) var(--tw-text-shadow-alpha), transparent)",
			),
		))

		t.Run(scenario("underline", newUtilityGeneratedKV("text-decoration-line", "underline")))
		t.Run(scenario("overline", newUtilityGeneratedKV("text-decoration-line", "overline")))
		t.Run(scenario("line-through", newUtilityGeneratedKV("text-decoration-line", "line-through")))
		t.Run(scenario("no-underline", newUtilityGeneratedKV("text-decoration-line", "none")))

		t.Run(scenario("decoration-solid", newUtilityGeneratedKV("text-decoration-style", "solid")))
		t.Run(scenario("decoration-double", newUtilityGeneratedKV("text-decoration-style", "double")))
		t.Run(scenario("decoration-dotted", newUtilityGeneratedKV("text-decoration-style", "dotted")))
		t.Run(scenario("decoration-dashed", newUtilityGeneratedKV("text-decoration-style", "dashed")))
		t.Run(scenario("decoration-wavy", newUtilityGeneratedKV("text-decoration-style", "wavy")))

		t.Run(scenario("decoration-auto", newUtilityGeneratedKV("text-decoration-thickness", "auto")))
		t.Run(scenario("decoration-from-font", newUtilityGeneratedKV("text-decoration-thickness", "from-font")))
		t.Run(scenario("decoration-0", newUtilityGeneratedKV("text-decoration-thickness", "0px")))
		t.Run(scenario("decoration-1", newUtilityGeneratedKV("text-decoration-thickness", "1px")))
		t.Run(scenario("decoration-2", newUtilityGeneratedKV("text-decoration-thickness", "2px")))
		t.Run(scenario("decoration-4", newUtilityGeneratedKV("text-decoration-thickness", "4px")))
		t.Run(scenario("decoration-8", newUtilityGeneratedKV("text-decoration-thickness", "8px")))

		t.Run(scenario("decoration-red-500", newUtilityGeneratedKV("text-decoration-color", "var(--color-red-500)")))
		t.Run(scenario("decoration-blue-500", newUtilityGeneratedKV("text-decoration-color", "var(--color-blue-500)")))
		t.Run(scenario("decoration-black", newUtilityGeneratedKV("text-decoration-color", "var(--color-black)")))
		t.Run(scenario("decoration-white", newUtilityGeneratedKV("text-decoration-color", "var(--color-white)")))

		t.Run(scenario("decoration-(--my-decoration-color)", newUtilityGeneratedKV("text-decoration-color", "var(--my-decoration-color)")))
		t.Run(scenario("decoration-(length:--my-decoration-sth)", newUtilityGeneratedKV("text-decoration-thickness", "var(--my-decoration-sth)")))
		t.Run(scenario("decoration-[#ff0000]", newUtilityGeneratedKV("text-decoration-color", "#ff0000")))
		t.Run(scenario("decoration-[3px]", newUtilityGeneratedKV("text-decoration-thickness", "3px")))
		t.Run(scenario("decoration-[color:red]", newUtilityGeneratedKV("text-decoration-color", "red")))

		t.Run(scenario("underline-offset-0", newUtilityGeneratedKV("text-underline-offset", "0px")))
		t.Run(scenario("underline-offset-2", newUtilityGeneratedKV("text-underline-offset", "2px")))
		t.Run(scenario("-underline-offset-2", newUtilityGeneratedKV("text-underline-offset", "calc(2px * -1)")))
		t.Run(scenario("underline-offset-auto", newUtilityGeneratedKV("text-underline-offset", "auto")))
		t.Run(scenario("underline-offset-(--my-underline-offset)", newUtilityGeneratedKV("text-underline-offset", "var(--my-underline-offset)")))
		t.Run(scenario("underline-offset-[3px]", newUtilityGeneratedKV("text-underline-offset", "3px")))

		t.Run(scenario("uppercase", newUtilityGeneratedKV("text-transform", "uppercase")))
		t.Run(scenario("lowercase", newUtilityGeneratedKV("text-transform", "lowercase")))
		t.Run(scenario("capitalize", newUtilityGeneratedKV("text-transform", "capitalize")))
		t.Run(scenario("normal-case", newUtilityGeneratedKV("text-transform", "none")))

		t.Run(scenario("truncate", &utilityGenerated{
			kvs: []KV{
				newCssKV("overflow", "hidden"),
				newCssKV("text-overflow", "ellipsis"),
				newCssKV("white-space", "nowrap"),
			},
		}))
		t.Run(scenario("text-ellipsis", newUtilityGeneratedKV("text-overflow", "ellipsis")))
		t.Run(scenario("text-clip", newUtilityGeneratedKV("text-overflow", "clip")))

		t.Run(scenario("text-wrap", newUtilityGeneratedKV("text-wrap", "wrap")))
		t.Run(scenario("text-nowrap", newUtilityGeneratedKV("text-wrap", "nowrap")))
		t.Run(scenario("text-balance", newUtilityGeneratedKV("text-wrap", "balance")))
		t.Run(scenario("text-pretty", newUtilityGeneratedKV("text-wrap", "pretty")))

		t.Run(scenario("indent-4", newUtilityGeneratedKV("text-indent", "calc(var(--spacing) * 4)")))
		t.Run(scenario("-indent-4", newUtilityGeneratedKV("text-indent", "calc(var(--spacing) * -4)")))
		t.Run(scenario("indent-px", newUtilityGeneratedKV("text-indent", "1px")))
		t.Run(scenario("-indent-px", newUtilityGeneratedKV("text-indent", "-1px")))
		t.Run(scenario("indent-(--my-indent)", newUtilityGeneratedKV("text-indent", "var(--my-indent)")))
		t.Run(scenario("indent-[2em]", newUtilityGeneratedKV("text-indent", "2em")))

		t.Run(scenario("tab-4", newUtilityGeneratedKV("tab-size", "4")))
		t.Run(scenario("tab-(--my-tab-size)", newUtilityGeneratedKV("tab-size", "var(--my-tab-size)")))
		t.Run(scenario("tab-[8px]", newUtilityGeneratedKV("tab-size", "8px")))

		t.Run(scenario("align-baseline", newUtilityGeneratedKV("vertical-align", "baseline")))
		t.Run(scenario("align-top", newUtilityGeneratedKV("vertical-align", "top")))
		t.Run(scenario("align-middle", newUtilityGeneratedKV("vertical-align", "middle")))
		t.Run(scenario("align-bottom", newUtilityGeneratedKV("vertical-align", "bottom")))
		t.Run(scenario("align-text-top", newUtilityGeneratedKV("vertical-align", "text-top")))
		t.Run(scenario("align-text-bottom", newUtilityGeneratedKV("vertical-align", "text-bottom")))
		t.Run(scenario("align-sub", newUtilityGeneratedKV("vertical-align", "sub")))
		t.Run(scenario("align-super", newUtilityGeneratedKV("vertical-align", "super")))
		t.Run(scenario("align-(--my-align)", newUtilityGeneratedKV("vertical-align", "var(--my-align)")))
		t.Run(scenario("align-[10px]", newUtilityGeneratedKV("vertical-align", "10px")))

		t.Run(scenario("whitespace-normal", newUtilityGeneratedKV("white-space", "normal")))
		t.Run(scenario("whitespace-nowrap", newUtilityGeneratedKV("white-space", "nowrap")))
		t.Run(scenario("whitespace-pre", newUtilityGeneratedKV("white-space", "pre")))
		t.Run(scenario("whitespace-pre-line", newUtilityGeneratedKV("white-space", "pre-line")))
		t.Run(scenario("whitespace-pre-wrap", newUtilityGeneratedKV("white-space", "pre-wrap")))
		t.Run(scenario("whitespace-break-spaces", newUtilityGeneratedKV("white-space", "break-spaces")))

		t.Run(scenario("break-normal", newUtilityGeneratedKV("word-break", "normal")))
		t.Run(scenario("break-all", newUtilityGeneratedKV("word-break", "break-all")))
		t.Run(scenario("break-keep", newUtilityGeneratedKV("word-break", "keep-all")))

		t.Run(scenario("wrap-break-word", newUtilityGeneratedKV("overflow-wrap", "break-word")))
		t.Run(scenario("wrap-anywhere", newUtilityGeneratedKV("overflow-wrap", "anywhere")))
		t.Run(scenario("wrap-normal", newUtilityGeneratedKV("overflow-wrap", "normal")))

		t.Run(scenario("hyphens-none", newUtilityGeneratedKV("hyphens", "none")))
		t.Run(scenario("hyphens-manual", newUtilityGeneratedKV("hyphens", "manual")))
		t.Run(scenario("hyphens-auto", newUtilityGeneratedKV("hyphens", "auto")))

		t.Run(scenario("content-none", &utilityGenerated{
			kvs: []KV{newCssKV("--tw-content", "none")},
		}))
		t.Run(scenario("content-(--my-content)", &utilityGenerated{
			kvs: []KV{newCssKV("--tw-content", "var(--my-content)")},
		}))
		t.Run(scenario("content-['Hello']", &utilityGenerated{
			kvs: []KV{newCssKV("--tw-content", "'Hello'")},
		}))

		t.Run(scenario("bg-fixed", newUtilityGeneratedKV("background-attachment", "fixed")))
		t.Run(scenario("bg-local", newUtilityGeneratedKV("background-attachment", "local")))
		t.Run(scenario("bg-scroll", newUtilityGeneratedKV("background-attachment", "scroll")))

		t.Run(scenario("bg-clip-border", newUtilityGeneratedKV("background-clip", "border-box")))
		t.Run(scenario("bg-clip-padding", newUtilityGeneratedKV("background-clip", "padding-box")))
		t.Run(scenario("bg-clip-content", newUtilityGeneratedKV("background-clip", "content-box")))
		t.Run(scenario("bg-clip-text", newUtilityGeneratedKV("background-clip", "text")))

		t.Run(scenario("bg-red-100", newUtilityGeneratedKV("background-color", "var(--color-red-100)")))
		t.Run(scenario("bg-white", newUtilityGeneratedKV("background-color", "var(--color-white)")))
		t.Run(scenario("bg-black", newUtilityGeneratedKV("background-color", "var(--color-black)")))
		t.Run(scenario("bg-inherit", newUtilityGeneratedKV("background-color", "inherit")))
		t.Run(scenario("bg-transparent", newUtilityGeneratedKV("background-color", "transparent")))
		t.Run(scenario("bg-current", newUtilityGeneratedKV("background-color", "currentcolor")))
		t.Run(scenario("bg-(--some-var)", newUtilityGeneratedKV("background-color", "var(--some-var)")))
		t.Run(scenario("bg-[light-dark(var(--color-white),var(--color-gray-950))]", newUtilityGeneratedKV("background-color", "light-dark(var(--color-white), var(--color-gray-950))")))

		t.Run(scenario("bg-none",
			newUtilityGeneratedKV("background-image", "none"),
		))
		t.Run(scenario("bg-[url('/image.png')]",
			newUtilityGeneratedKV("background-image", "url('/image.png')"),
		))
		t.Run(scenario("bg-(image:--my-image)",
			newUtilityGeneratedKV("background-image", "var(--my-image)"),
		))
		t.Run(scenario("bg-linear-to-t",
			&utilityGenerated{
				kvs: []KV{
					newCssKV("--tw-gradient-position", "to top"),
					newCssKV("background-image", "linear-gradient(var(--tw-gradient-stops))"),
				},
				others: []Query{
					{
						Name: "@supports (background-image: linear-gradient(in lab, red, red))",
						KVs: []KV{
							newCssKV("--tw-gradient-position", "to top in oklab"),
						},
					},
				},
			},
		))
		t.Run(scenario("bg-linear-to-tr",
			&utilityGenerated{
				kvs: []KV{
					newCssKV("--tw-gradient-position", "to top right"),
					newCssKV("background-image", "linear-gradient(var(--tw-gradient-stops))"),
				},
				others: []Query{
					{
						Name: "@supports (background-image: linear-gradient(in lab, red, red))",
						KVs: []KV{
							newCssKV("--tw-gradient-position", "to top right in oklab"),
						},
					},
				},
			},
		))
		t.Run(scenario("bg-linear-to-r",
			&utilityGenerated{
				kvs: []KV{
					newCssKV("--tw-gradient-position", "to right"),
					newCssKV("background-image", "linear-gradient(var(--tw-gradient-stops))"),
				},
				others: []Query{
					{
						Name: "@supports (background-image: linear-gradient(in lab, red, red))",
						KVs: []KV{
							newCssKV("--tw-gradient-position", "to right in oklab"),
						},
					},
				},
			},
		))
		t.Run(scenario("bg-linear-to-br",
			&utilityGenerated{
				kvs: []KV{
					newCssKV("--tw-gradient-position", "to bottom right"),
					newCssKV("background-image", "linear-gradient(var(--tw-gradient-stops))"),
				},
				others: []Query{
					{
						Name: "@supports (background-image: linear-gradient(in lab, red, red))",
						KVs: []KV{
							newCssKV("--tw-gradient-position", "to bottom right in oklab"),
						},
					},
				},
			},
		))
		t.Run(scenario("bg-linear-to-b",
			&utilityGenerated{
				kvs: []KV{
					newCssKV("--tw-gradient-position", "to bottom"),
					newCssKV("background-image", "linear-gradient(var(--tw-gradient-stops))"),
				},
				others: []Query{
					{
						Name: "@supports (background-image: linear-gradient(in lab, red, red))",
						KVs: []KV{
							newCssKV("--tw-gradient-position", "to bottom in oklab"),
						},
					},
				},
			},
		))
		t.Run(scenario("bg-linear-to-bl",
			&utilityGenerated{
				kvs: []KV{
					newCssKV("--tw-gradient-position", "to bottom left"),
					newCssKV("background-image", "linear-gradient(var(--tw-gradient-stops))"),
				},
				others: []Query{
					{
						Name: "@supports (background-image: linear-gradient(in lab, red, red))",
						KVs: []KV{
							newCssKV("--tw-gradient-position", "to bottom left in oklab"),
						},
					},
				},
			},
		))
		t.Run(scenario("bg-linear-to-l",
			&utilityGenerated{
				kvs: []KV{
					newCssKV("--tw-gradient-position", "to left"),
					newCssKV("background-image", "linear-gradient(var(--tw-gradient-stops))"),
				},
				others: []Query{
					{
						Name: "@supports (background-image: linear-gradient(in lab, red, red))",
						KVs: []KV{
							newCssKV("--tw-gradient-position", "to left in oklab"),
						},
					},
				},
			},
		))
		t.Run(scenario("bg-linear-to-tl",
			&utilityGenerated{
				kvs: []KV{
					newCssKV("--tw-gradient-position", "to top left"),
					newCssKV("background-image", "linear-gradient(var(--tw-gradient-stops))"),
				},
				others: []Query{
					{
						Name: "@supports (background-image: linear-gradient(in lab, red, red))",
						KVs: []KV{
							newCssKV("--tw-gradient-position", "to top left in oklab"),
						},
					},
				},
			},
		))
		t.Run(scenario("bg-linear-45",
			&utilityGenerated{
				kvs: []KV{
					newCssKV("--tw-gradient-position", "45deg"),
					newCssKV("background-image", "linear-gradient(var(--tw-gradient-stops))"),
				},
				others: []Query{
					{
						Name: "@supports (background-image: linear-gradient(in lab, red, red))",
						KVs: []KV{
							newCssKV("--tw-gradient-position", "45deg in oklab"),
						},
					},
				},
			},
		))
		t.Run(scenario("-bg-linear-45",
			&utilityGenerated{
				kvs: []KV{
					newCssKV("--tw-gradient-position", "calc(45deg * -1)"),
					newCssKV("background-image", "linear-gradient(var(--tw-gradient-stops))"),
				},
				others: []Query{
					{
						Name: "@supports (background-image: linear-gradient(in lab, red, red))",
						KVs: []KV{
							newCssKV("--tw-gradient-position", "calc(45deg * -1) in oklab"),
						},
					},
				},
			},
		))
		t.Run(scenario("bg-linear-(--my-linear)",
			&utilityGenerated{
				kvs: []KV{
					newCssKV("--tw-gradient-position", "var(--my-linear)"),
					newCssKV("background-image", "linear-gradient(var(--tw-gradient-stops,var(--my-linear)))"),
				},
			},
		))
		t.Run(scenario("bg-linear-[25deg]",
			&utilityGenerated{
				kvs: []KV{
					newCssKV("--tw-gradient-position", "25deg"),
					newCssKV("background-image", "linear-gradient(var(--tw-gradient-stops,25deg))"),
				},
			},
		))
		t.Run(scenario("bg-radial",
			&utilityGenerated{
				kvs: []KV{
					newCssKV("--tw-gradient-position", "in oklab"),
					newCssKV("background-image", "radial-gradient(var(--tw-gradient-stops))"),
				},
			},
		))
		t.Run(scenario("bg-radial-(--my-radial)",
			&utilityGenerated{
				kvs: []KV{
					newCssKV("--tw-gradient-position", "var(--my-radial)"),
					newCssKV("background-image", "radial-gradient(var(--tw-gradient-stops,var(--my-radial)))"),
				},
			},
		))
		t.Run(scenario("bg-radial-[circle_at_center]",
			&utilityGenerated{
				kvs: []KV{
					newCssKV("--tw-gradient-position", "circle at center"),
					newCssKV("background-image", "radial-gradient(var(--tw-gradient-stops,circle at center))"),
				},
			},
		))
		t.Run(scenario("bg-conic-45",
			&utilityGenerated{
				kvs: []KV{
					newCssKV("--tw-gradient-position", "from 45deg in oklab"),
					newCssKV("background-image", "conic-gradient(var(--tw-gradient-stops))"),
				},
			},
		))
		t.Run(scenario("-bg-conic-45",
			&utilityGenerated{
				kvs: []KV{
					newCssKV("--tw-gradient-position", "from calc(45deg * -1) in oklab"),
					newCssKV("background-image", "conic-gradient(var(--tw-gradient-stops))"),
				},
			},
		))
		t.Run(scenario("bg-conic-(--my-conic)",
			&utilityGenerated{
				kvs: []KV{
					newCssKV("--tw-gradient-position", "var(--my-conic)"),
					newCssKV("background-image", "conic-gradient(var(--tw-gradient-stops,var(--my-conic)))"),
				},
			},
		))
		t.Run(scenario("bg-conic-[from_45deg]",
			&utilityGenerated{
				kvs: []KV{
					newCssKV("--tw-gradient-position", "from 45deg"),
					newCssKV("background-image", "conic-gradient(var(--tw-gradient-stops,from 45deg))"),
				},
			},
		))

		t.Run(scenario("from-red-500",
			&utilityGenerated{
				kvs: []KV{
					newCssKV("--tw-gradient-from", "var(--color-red-500)"),
					newCssKV("--tw-gradient-stops", testGradientStops),
				},
			},
		))
		t.Run(scenario("from-50%",
			newUtilityGeneratedKV("--tw-gradient-from-position", "50%"),
		))
		t.Run(scenario("from-(--my-from)",
			&utilityGenerated{
				kvs: []KV{
					newCssKV("--tw-gradient-from", "var(--my-from)"),
					newCssKV("--tw-gradient-stops", testGradientStops),
				},
			},
		))
		t.Run(scenario("from-[#ff0000]",
			&utilityGenerated{
				kvs: []KV{
					newCssKV("--tw-gradient-from", "#ff0000"),
					newCssKV("--tw-gradient-stops", testGradientStops),
				},
			},
		))

		t.Run(scenario("via-blue-500",
			&utilityGenerated{
				kvs: []KV{
					newCssKV("--tw-gradient-via", "var(--color-blue-500)"),
					newCssKV("--tw-gradient-via-stops", testGradientViaStops),
					newCssKV("--tw-gradient-stops", "var(--tw-gradient-via-stops)"),
				},
			},
		))
		t.Run(scenario("via-25%",
			newUtilityGeneratedKV("--tw-gradient-via-position", "25%"),
		))
		t.Run(scenario("via-(--my-via)",
			&utilityGenerated{
				kvs: []KV{
					newCssKV("--tw-gradient-via", "var(--my-via)"),
					newCssKV("--tw-gradient-via-stops", testGradientViaStops),
					newCssKV("--tw-gradient-stops", "var(--tw-gradient-via-stops)"),
				},
			},
		))
		t.Run(scenario("via-[#0000ff]",
			&utilityGenerated{
				kvs: []KV{
					newCssKV("--tw-gradient-via", "#0000ff"),
					newCssKV("--tw-gradient-via-stops", testGradientViaStops),
					newCssKV("--tw-gradient-stops", "var(--tw-gradient-via-stops)"),
				},
			},
		))

		t.Run(scenario("to-green-500",
			&utilityGenerated{
				kvs: []KV{
					newCssKV("--tw-gradient-to", "var(--color-green-500)"),
					newCssKV("--tw-gradient-stops", testGradientStops),
				},
			},
		))
		t.Run(scenario("to-75%",
			newUtilityGeneratedKV("--tw-gradient-to-position", "75%"),
		))
		t.Run(scenario("to-(--my-to)",
			&utilityGenerated{
				kvs: []KV{
					newCssKV("--tw-gradient-to", "var(--my-to)"),
					newCssKV("--tw-gradient-stops", testGradientStops),
				},
			},
		))
		t.Run(scenario("to-[#00ff00]",
			&utilityGenerated{
				kvs: []KV{
					newCssKV("--tw-gradient-to", "#00ff00"),
					newCssKV("--tw-gradient-stops", testGradientStops),
				},
			},
		))

		t.Run(scenario("bg-origin-border", newUtilityGeneratedKV("background-origin", "border-box")))
		t.Run(scenario("bg-origin-padding", newUtilityGeneratedKV("background-origin", "padding-box")))
		t.Run(scenario("bg-origin-content", newUtilityGeneratedKV("background-origin", "content-box")))

		t.Run(scenario("bg-blend-normal", newUtilityGeneratedKV("background-blend-mode", "normal")))
		t.Run(scenario("bg-blend-multiply", newUtilityGeneratedKV("background-blend-mode", "multiply")))
		t.Run(scenario("bg-blend-screen", newUtilityGeneratedKV("background-blend-mode", "screen")))
		t.Run(scenario("bg-blend-overlay", newUtilityGeneratedKV("background-blend-mode", "overlay")))
		t.Run(scenario("bg-blend-darken", newUtilityGeneratedKV("background-blend-mode", "darken")))
		t.Run(scenario("bg-blend-lighten", newUtilityGeneratedKV("background-blend-mode", "lighten")))
		t.Run(scenario("bg-blend-color-dodge", newUtilityGeneratedKV("background-blend-mode", "color-dodge")))
		t.Run(scenario("bg-blend-color-burn", newUtilityGeneratedKV("background-blend-mode", "color-burn")))
		t.Run(scenario("bg-blend-hard-light", newUtilityGeneratedKV("background-blend-mode", "hard-light")))
		t.Run(scenario("bg-blend-soft-light", newUtilityGeneratedKV("background-blend-mode", "soft-light")))
		t.Run(scenario("bg-blend-difference", newUtilityGeneratedKV("background-blend-mode", "difference")))
		t.Run(scenario("bg-blend-exclusion", newUtilityGeneratedKV("background-blend-mode", "exclusion")))
		t.Run(scenario("bg-blend-hue", newUtilityGeneratedKV("background-blend-mode", "hue")))
		t.Run(scenario("bg-blend-saturation", newUtilityGeneratedKV("background-blend-mode", "saturation")))
		t.Run(scenario("bg-blend-color", newUtilityGeneratedKV("background-blend-mode", "color")))
		t.Run(scenario("bg-blend-luminosity", newUtilityGeneratedKV("background-blend-mode", "luminosity")))

		t.Run(scenario("mask-none", &utilityGenerated{
			kvs: []KV{
				newCssKV("-webkit-mask-image", "none"),
				newCssKV("mask-image", "none"),
			},
		}))
		t.Run(scenario("mask-[linear-gradient(#ffff,#0000)]", &utilityGenerated{
			kvs: []KV{
				newCssKV("-webkit-mask-image", "linear-gradient(#ffff, #0000)"),
				newCssKV("mask-image", "linear-gradient(#ffff, #0000)"),
			},
		}))
		t.Run(scenario("mask-[url(http://example.com)]", &utilityGenerated{
			kvs: []KV{
				newCssKV("-webkit-mask-image", "url(http://example.com)"),
				newCssKV("mask-image", "url(http://example.com)"),
			},
		}))
		t.Run(scenario("mask-[var(--some-var)]", &utilityGenerated{
			kvs: []KV{
				newCssKV("-webkit-mask-image", "var(--some-var)"),
				newCssKV("mask-image", "var(--some-var)"),
			},
		}))
		t.Run(scenario("mask-[image:var(--some-var)]", &utilityGenerated{
			kvs: []KV{
				newCssKV("-webkit-mask-image", "var(--some-var)"),
				newCssKV("mask-image", "var(--some-var)"),
			},
		}))
		t.Run(scenario("mask-[url:var(--some-var)]", &utilityGenerated{
			kvs: []KV{
				newCssKV("-webkit-mask-image", "var(--some-var)"),
				newCssKV("mask-image", "var(--some-var)"),
			},
		}))

		t.Run(scenario("mask-add", &utilityGenerated{
			kvs: []KV{
				newCssKV("-webkit-mask-composite", "source-over"),
				newCssKV("mask-composite", "add"),
			},
		}))
		t.Run(scenario("mask-subtract", &utilityGenerated{
			kvs: []KV{
				newCssKV("-webkit-mask-composite", "source-out"),
				newCssKV("mask-composite", "subtract"),
			},
		}))
		t.Run(scenario("mask-intersect", &utilityGenerated{
			kvs: []KV{
				newCssKV("-webkit-mask-composite", "source-in"),
				newCssKV("mask-composite", "intersect"),
			},
		}))
		t.Run(scenario("mask-exclude", &utilityGenerated{
			kvs: []KV{
				newCssKV("-webkit-mask-composite", "xor"),
				newCssKV("mask-composite", "exclude"),
			},
		}))

		t.Run(scenario("mask-alpha", &utilityGenerated{
			kvs: []KV{
				newCssKV("-webkit-mask-source-type", "alpha"),
				newCssKV("mask-mode", "alpha"),
			},
		}))
		t.Run(scenario("mask-luminance", &utilityGenerated{
			kvs: []KV{
				newCssKV("-webkit-mask-source-type", "luminance"),
				newCssKV("mask-mode", "luminance"),
			},
		}))
		t.Run(scenario("mask-match", &utilityGenerated{
			kvs: []KV{
				newCssKV("-webkit-mask-source-type", "auto"),
				newCssKV("mask-mode", "match-source"),
			},
		}))

		t.Run(scenario("mask-type-alpha", newUtilityGeneratedKV("mask-type", "alpha")))
		t.Run(scenario("mask-type-luminance", newUtilityGeneratedKV("mask-type", "luminance")))

		t.Run(scenario("mask-auto", &utilityGenerated{
			kvs: []KV{
				newCssKV("-webkit-mask-size", "auto"),
				newCssKV("mask-size", "auto"),
			},
		}))
		t.Run(scenario("mask-cover", &utilityGenerated{
			kvs: []KV{
				newCssKV("-webkit-mask-size", "cover"),
				newCssKV("mask-size", "cover"),
			},
		}))
		t.Run(scenario("mask-contain", &utilityGenerated{
			kvs: []KV{
				newCssKV("-webkit-mask-size", "contain"),
				newCssKV("mask-size", "contain"),
			},
		}))
		t.Run(scenario("mask-[cover]", &utilityGenerated{
			kvs: []KV{
				newCssKV("-webkit-mask-size", "cover"),
				newCssKV("mask-size", "cover"),
			},
		}))
		t.Run(scenario("mask-[contain]", &utilityGenerated{
			kvs: []KV{
				newCssKV("-webkit-mask-size", "contain"),
				newCssKV("mask-size", "contain"),
			},
		}))
		t.Run(scenario("mask-[size:120px_120px]", &utilityGenerated{
			kvs: []KV{
				newCssKV("-webkit-mask-size", "120px 120px"),
				newCssKV("mask-size", "120px 120px"),
			},
		}))
		t.Run(scenario("mask-size-[120px]", &utilityGenerated{
			kvs: []KV{
				newCssKV("-webkit-mask-size", "120px"),
				newCssKV("mask-size", "120px"),
			},
		}))
		t.Run(scenario("mask-size-[120px_120px]", &utilityGenerated{
			kvs: []KV{
				newCssKV("-webkit-mask-size", "120px 120px"),
				newCssKV("mask-size", "120px 120px"),
			},
		}))
		t.Run(scenario("mask-size-[var(--some-var)]", &utilityGenerated{
			kvs: []KV{
				newCssKV("-webkit-mask-size", "var(--some-var)"),
				newCssKV("mask-size", "var(--some-var)"),
			},
		}))

		t.Run(scenario("mask-center", &utilityGenerated{
			kvs: []KV{
				newCssKV("-webkit-mask-position", "center"),
				newCssKV("mask-position", "center"),
			},
		}))
		t.Run(scenario("mask-top", &utilityGenerated{
			kvs: []KV{
				newCssKV("-webkit-mask-position", "top"),
				newCssKV("mask-position", "top"),
			},
		}))
		t.Run(scenario("mask-top-right", &utilityGenerated{
			kvs: []KV{
				newCssKV("-webkit-mask-position", "100% 0"),
				newCssKV("mask-position", "100% 0"),
			},
		}))
		t.Run(scenario("mask-top-left", &utilityGenerated{
			kvs: []KV{
				newCssKV("-webkit-mask-position", "0 0"),
				newCssKV("mask-position", "0 0"),
			},
		}))
		t.Run(scenario("mask-bottom", &utilityGenerated{
			kvs: []KV{
				newCssKV("-webkit-mask-position", "bottom"),
				newCssKV("mask-position", "bottom"),
			},
		}))
		t.Run(scenario("mask-bottom-right", &utilityGenerated{
			kvs: []KV{
				newCssKV("-webkit-mask-position", "100% 100%"),
				newCssKV("mask-position", "100% 100%"),
			},
		}))
		t.Run(scenario("mask-bottom-left", &utilityGenerated{
			kvs: []KV{
				newCssKV("-webkit-mask-position", "0 100%"),
				newCssKV("mask-position", "0 100%"),
			},
		}))
		t.Run(scenario("mask-left", &utilityGenerated{
			kvs: []KV{
				newCssKV("-webkit-mask-position", "0"),
				newCssKV("mask-position", "0"),
			},
		}))
		t.Run(scenario("mask-right", &utilityGenerated{
			kvs: []KV{
				newCssKV("-webkit-mask-position", "100%"),
				newCssKV("mask-position", "100%"),
			},
		}))
		t.Run(scenario("mask-[50%]", &utilityGenerated{
			kvs: []KV{
				newCssKV("-webkit-mask-position", "50%"),
				newCssKV("mask-position", "50%"),
			},
		}))
		t.Run(scenario("mask-[120px]", &utilityGenerated{
			kvs: []KV{
				newCssKV("-webkit-mask-position", "120px"),
				newCssKV("mask-position", "120px"),
			},
		}))
		t.Run(scenario("mask-[120px_120px]", &utilityGenerated{
			kvs: []KV{
				newCssKV("-webkit-mask-position", "120px 120px"),
				newCssKV("mask-position", "120px 120px"),
			},
		}))
		t.Run(scenario("mask-[position:120px_120px]", &utilityGenerated{
			kvs: []KV{
				newCssKV("-webkit-mask-position", "120px 120px"),
				newCssKV("mask-position", "120px 120px"),
			},
		}))
		t.Run(scenario("mask-position-[120px]", &utilityGenerated{
			kvs: []KV{
				newCssKV("-webkit-mask-position", "120px"),
				newCssKV("mask-position", "120px"),
			},
		}))
		t.Run(scenario("mask-position-[120px_120px]", &utilityGenerated{
			kvs: []KV{
				newCssKV("-webkit-mask-position", "120px 120px"),
				newCssKV("mask-position", "120px 120px"),
			},
		}))
		t.Run(scenario("mask-position-[var(--some-var)]", &utilityGenerated{
			kvs: []KV{
				newCssKV("-webkit-mask-position", "var(--some-var)"),
				newCssKV("mask-position", "var(--some-var)"),
			},
		}))

		t.Run(scenario("mask-repeat", &utilityGenerated{
			kvs: []KV{
				newCssKV("-webkit-mask-repeat", "repeat"),
				newCssKV("mask-repeat", "repeat"),
			},
		}))
		t.Run(scenario("mask-no-repeat", &utilityGenerated{
			kvs: []KV{
				newCssKV("-webkit-mask-repeat", "no-repeat"),
				newCssKV("mask-repeat", "no-repeat"),
			},
		}))
		t.Run(scenario("mask-repeat-x", &utilityGenerated{
			kvs: []KV{
				newCssKV("-webkit-mask-repeat", "repeat-x"),
				newCssKV("mask-repeat", "repeat-x"),
			},
		}))
		t.Run(scenario("mask-repeat-y", &utilityGenerated{
			kvs: []KV{
				newCssKV("-webkit-mask-repeat", "repeat-y"),
				newCssKV("mask-repeat", "repeat-y"),
			},
		}))
		t.Run(scenario("mask-repeat-round", &utilityGenerated{
			kvs: []KV{
				newCssKV("-webkit-mask-repeat", "round"),
				newCssKV("mask-repeat", "round"),
			},
		}))
		t.Run(scenario("mask-repeat-space", &utilityGenerated{
			kvs: []KV{
				newCssKV("-webkit-mask-repeat", "space"),
				newCssKV("mask-repeat", "space"),
			},
		}))

		t.Run(scenario("mask-clip-border", &utilityGenerated{
			kvs: []KV{
				newCssKV("-webkit-mask-clip", "border-box"),
				newCssKV("mask-clip", "border-box"),
			},
		}))
		t.Run(scenario("mask-clip-padding", &utilityGenerated{
			kvs: []KV{
				newCssKV("-webkit-mask-clip", "padding-box"),
				newCssKV("mask-clip", "padding-box"),
			},
		}))
		t.Run(scenario("mask-clip-content", &utilityGenerated{
			kvs: []KV{
				newCssKV("-webkit-mask-clip", "content-box"),
				newCssKV("mask-clip", "content-box"),
			},
		}))
		t.Run(scenario("mask-clip-fill", &utilityGenerated{
			kvs: []KV{
				newCssKV("-webkit-mask-clip", "fill-box"),
				newCssKV("mask-clip", "fill-box"),
			},
		}))
		t.Run(scenario("mask-clip-stroke", &utilityGenerated{
			kvs: []KV{
				newCssKV("-webkit-mask-clip", "stroke-box"),
				newCssKV("mask-clip", "stroke-box"),
			},
		}))
		t.Run(scenario("mask-clip-view", &utilityGenerated{
			kvs: []KV{
				newCssKV("-webkit-mask-clip", "view-box"),
				newCssKV("mask-clip", "view-box"),
			},
		}))
		t.Run(scenario("mask-no-clip", &utilityGenerated{
			kvs: []KV{
				newCssKV("-webkit-mask-clip", "no-clip"),
				newCssKV("mask-clip", "no-clip"),
			},
		}))

		t.Run(scenario("mask-origin-border", &utilityGenerated{
			kvs: []KV{
				newCssKV("-webkit-mask-origin", "border-box"),
				newCssKV("mask-origin", "border-box"),
			},
		}))
		t.Run(scenario("mask-origin-padding", &utilityGenerated{
			kvs: []KV{
				newCssKV("-webkit-mask-origin", "padding-box"),
				newCssKV("mask-origin", "padding-box"),
			},
		}))
		t.Run(scenario("mask-origin-content", &utilityGenerated{
			kvs: []KV{
				newCssKV("-webkit-mask-origin", "content-box"),
				newCssKV("mask-origin", "content-box"),
			},
		}))
		t.Run(scenario("mask-origin-fill", &utilityGenerated{
			kvs: []KV{
				newCssKV("-webkit-mask-origin", "fill-box"),
				newCssKV("mask-origin", "fill-box"),
			},
		}))
		t.Run(scenario("mask-origin-stroke", &utilityGenerated{
			kvs: []KV{
				newCssKV("-webkit-mask-origin", "stroke-box"),
				newCssKV("mask-origin", "stroke-box"),
			},
		}))
		t.Run(scenario("mask-origin-view", &utilityGenerated{
			kvs: []KV{
				newCssKV("-webkit-mask-origin", "view-box"),
				newCssKV("mask-origin", "view-box"),
			},
		}))

		t.Run(scenario("bg-top-left", newUtilityGeneratedKV("background-position", "top left")))
		t.Run(scenario("bg-top", newUtilityGeneratedKV("background-position", "top")))
		t.Run(scenario("bg-top-right", newUtilityGeneratedKV("background-position", "top right")))
		t.Run(scenario("bg-left", newUtilityGeneratedKV("background-position", "left")))
		t.Run(scenario("bg-center", newUtilityGeneratedKV("background-position", "center")))
		t.Run(scenario("bg-right", newUtilityGeneratedKV("background-position", "right")))
		t.Run(scenario("bg-bottom-left", newUtilityGeneratedKV("background-position", "bottom left")))
		t.Run(scenario("bg-bottom", newUtilityGeneratedKV("background-position", "bottom")))
		t.Run(scenario("bg-bottom-right", newUtilityGeneratedKV("background-position", "bottom right")))
		t.Run(scenario("bg-position-(--my-position)", newUtilityGeneratedKV("background-position", "var(--my-position)")))
		t.Run(scenario("bg-position-[25%_50%]", newUtilityGeneratedKV("background-position", "25% 50%")))

		t.Run(scenario("bg-repeat", newUtilityGeneratedKV("background-repeat", "repeat")))
		t.Run(scenario("bg-repeat-x", newUtilityGeneratedKV("background-repeat", "repeat-x")))
		t.Run(scenario("bg-repeat-y", newUtilityGeneratedKV("background-repeat", "repeat-y")))
		t.Run(scenario("bg-repeat-space", newUtilityGeneratedKV("background-repeat", "space")))
		t.Run(scenario("bg-repeat-round", newUtilityGeneratedKV("background-repeat", "round")))
		t.Run(scenario("bg-no-repeat", newUtilityGeneratedKV("background-repeat", "no-repeat")))

		t.Run(scenario("bg-auto", newUtilityGeneratedKV("background-size", "auto")))
		t.Run(scenario("bg-cover", newUtilityGeneratedKV("background-size", "cover")))
		t.Run(scenario("bg-contain", newUtilityGeneratedKV("background-size", "contain")))
		t.Run(scenario("bg-size-(--my-size)", newUtilityGeneratedKV("background-size", "var(--my-size)")))
		t.Run(scenario("bg-size-[100px_50px]", newUtilityGeneratedKV("background-size", "100px 50px")))

		t.Run(scenario("rounded-xs", newUtilityGeneratedKV("border-radius", "var(--radius-xs)")))
		t.Run(scenario("rounded-sm", newUtilityGeneratedKV("border-radius", "var(--radius-sm)")))
		t.Run(scenario("rounded-md", newUtilityGeneratedKV("border-radius", "var(--radius-md)")))
		t.Run(scenario("rounded-lg", newUtilityGeneratedKV("border-radius", "var(--radius-lg)")))
		t.Run(scenario("rounded-xl", newUtilityGeneratedKV("border-radius", "var(--radius-xl)")))
		t.Run(scenario("rounded-2xl", newUtilityGeneratedKV("border-radius", "var(--radius-2xl)")))
		t.Run(scenario("rounded-3xl", newUtilityGeneratedKV("border-radius", "var(--radius-3xl)")))
		t.Run(scenario("rounded-4xl", newUtilityGeneratedKV("border-radius", "var(--radius-4xl)")))
		t.Run(scenario("rounded-none", newUtilityGeneratedKV("border-radius", "0")))
		t.Run(scenario("rounded-full", newUtilityGeneratedKV("border-radius", "calc(infinity * 1px)")))
		t.Run(scenario("rounded-(--my-radius)", newUtilityGeneratedKV("border-radius", "var(--my-radius)")))
		t.Run(scenario("rounded-[12px]", newUtilityGeneratedKV("border-radius", "12px")))

		roundedVariants := []struct {
			name string
			kvs  func(value string) []KV
		}{
			{
				"",
				func(value string) []KV {
					return []KV{
						newCssKV("border-radius", value),
					}
				},
			},
			{
				"t",
				func(value string) []KV {
					return []KV{
						newCssKV("border-top-left-radius", value),
						newCssKV("border-top-right-radius", value),
					}
				},
			},
			{
				"r",
				func(value string) []KV {
					return []KV{
						newCssKV("border-top-right-radius", value),
						newCssKV("border-bottom-right-radius", value),
					}
				},
			},
			{
				"b",
				func(value string) []KV {
					return []KV{
						newCssKV("border-bottom-right-radius", value),
						newCssKV("border-bottom-left-radius", value),
					}
				},
			},
			{
				"l",
				func(value string) []KV {
					return []KV{
						newCssKV("border-top-left-radius", value),
						newCssKV("border-bottom-left-radius", value),
					}
				},
			},
			{
				"s",
				func(value string) []KV {
					return []KV{
						newCssKV("border-start-start-radius", value),
						newCssKV("border-end-start-radius", value),
					}
				},
			},
			{
				"e",
				func(value string) []KV {
					return []KV{
						newCssKV("border-start-end-radius", value),
						newCssKV("border-end-end-radius", value),
					}
				},
			},
			{
				"tl",
				func(value string) []KV {
					return []KV{
						newCssKV("border-top-left-radius", value),
					}
				},
			},
			{
				"tr",
				func(value string) []KV {
					return []KV{
						newCssKV("border-top-right-radius", value),
					}
				},
			},
			{
				"br",
				func(value string) []KV {
					return []KV{
						newCssKV("border-bottom-right-radius", value),
					}
				},
			},
			{
				"bl",
				func(value string) []KV {
					return []KV{
						newCssKV("border-bottom-left-radius", value),
					}
				},
			},
			{
				"ss",
				func(value string) []KV {
					return []KV{
						newCssKV("border-start-start-radius", value),
					}
				},
			},
			{
				"se",
				func(value string) []KV {
					return []KV{
						newCssKV("border-start-end-radius", value),
					}
				},
			},
			{
				"es",
				func(value string) []KV {
					return []KV{
						newCssKV("border-end-start-radius", value),
					}
				},
			},
			{
				"ee",
				func(value string) []KV {
					return []KV{
						newCssKV("border-end-end-radius", value),
					}
				},
			},
		}

		radii := []struct {
			name  string
			value string
		}{
			{"xs", "var(--radius-xs)"},
			{"sm", "var(--radius-sm)"},
			{"md", "var(--radius-md)"},
			{"lg", "var(--radius-lg)"},
			{"xl", "var(--radius-xl)"},
			{"2xl", "var(--radius-2xl)"},
			{"3xl", "var(--radius-3xl)"},
			{"4xl", "var(--radius-4xl)"},
			{"none", "0"},
			{"full", "calc(infinity * 1px)"},
			{"(--my-radius)", "var(--my-radius)"},
			{"[12px]", "12px"},
		}

		for _, radius := range radii {
			for _, variant := range roundedVariants {
				name := "rounded-"
				if variant.name != "" {
					name += variant.name + "-"
				}
				name += radius.name

				kvs := variant.kvs(radius.value)
				t.Run(scenario(name, &utilityGenerated{kvs: kvs}))
			}
		}

		t.Run(scenario("border",
			&utilityGenerated{
				kvs: []KV{
					newCssKV("border-style", "var(--tw-border-style)"),
					newCssKV("border-width", "1px"),
				},
			},
		))
		t.Run(scenario("border-4",
			&utilityGenerated{
				kvs: []KV{
					newCssKV("border-style", "var(--tw-border-style)"),
					newCssKV("border-width", "4px"),
				},
			},
		))
		t.Run(scenario("border-(length:--my-width)",
			&utilityGenerated{
				kvs: []KV{
					newCssKV("border-style", "var(--tw-border-style)"),
					newCssKV("border-width", "var(--my-width)"),
				},
			},
		))
		t.Run(scenario("border-[3px]",
			&utilityGenerated{
				kvs: []KV{
					newCssKV("border-style", "var(--tw-border-style)"),
					newCssKV("border-width", "3px"),
				},
			},
		))
		t.Run(scenario("border-[rgb(255_0_0)]",
			newUtilityGeneratedKV("border-color", "rgb(255 0 0)"),
		))
		t.Run(scenario("border-[length:rgb(255_0_0)]",
			&utilityGenerated{
				kvs: []KV{
					newCssKV("border-style", "var(--tw-border-style)"),
					newCssKV("border-width", "rgb(255 0 0)"),
				},
			},
		))

		t.Run(scenario("border-x",
			&utilityGenerated{
				kvs: []KV{
					newCssKV("border-inline-style", "var(--tw-border-style)"),
					newCssKV("border-inline-width", "1px"),
				},
			},
		))
		t.Run(scenario("border-x-4",
			&utilityGenerated{
				kvs: []KV{
					newCssKV("border-inline-style", "var(--tw-border-style)"),
					newCssKV("border-inline-width", "4px"),
				},
			},
		))
		t.Run(scenario("border-x-(length:--my-width)",
			&utilityGenerated{
				kvs: []KV{
					newCssKV("border-inline-style", "var(--tw-border-style)"),
					newCssKV("border-inline-width", "var(--my-width)"),
				},
			},
		))
		t.Run(scenario("border-x-[3px]",
			&utilityGenerated{
				kvs: []KV{
					newCssKV("border-inline-style", "var(--tw-border-style)"),
					newCssKV("border-inline-width", "3px"),
				},
			},
		))

		t.Run(scenario("border-y",
			&utilityGenerated{
				kvs: []KV{
					newCssKV("border-block-style", "var(--tw-border-style)"),
					newCssKV("border-block-width", "1px"),
				},
			},
		))
		t.Run(scenario("border-y-4",
			&utilityGenerated{
				kvs: []KV{
					newCssKV("border-block-style", "var(--tw-border-style)"),
					newCssKV("border-block-width", "4px"),
				},
			},
		))
		t.Run(scenario("border-y-(length:--my-width)",
			&utilityGenerated{
				kvs: []KV{
					newCssKV("border-block-style", "var(--tw-border-style)"),
					newCssKV("border-block-width", "var(--my-width)"),
				},
			},
		))
		t.Run(scenario("border-y-[3px]",
			&utilityGenerated{
				kvs: []KV{
					newCssKV("border-block-style", "var(--tw-border-style)"),
					newCssKV("border-block-width", "3px"),
				},
			},
		))

		t.Run(scenario("border-s",
			&utilityGenerated{
				kvs: []KV{
					newCssKV("border-inline-start-style", "var(--tw-border-style)"),
					newCssKV("border-inline-start-width", "1px"),
				},
			},
		))
		t.Run(scenario("border-s-4",
			&utilityGenerated{
				kvs: []KV{
					newCssKV("border-inline-start-style", "var(--tw-border-style)"),
					newCssKV("border-inline-start-width", "4px"),
				},
			},
		))
		t.Run(scenario("border-s-(length:--my-width)",
			&utilityGenerated{
				kvs: []KV{
					newCssKV("border-inline-start-style", "var(--tw-border-style)"),
					newCssKV("border-inline-start-width", "var(--my-width)"),
				},
			},
		))
		t.Run(scenario("border-s-[3px]",
			&utilityGenerated{
				kvs: []KV{
					newCssKV("border-inline-start-style", "var(--tw-border-style)"),
					newCssKV("border-inline-start-width", "3px"),
				},
			},
		))

		t.Run(scenario("border-e",
			&utilityGenerated{
				kvs: []KV{
					newCssKV("border-inline-end-style", "var(--tw-border-style)"),
					newCssKV("border-inline-end-width", "1px"),
				},
			},
		))
		t.Run(scenario("border-e-4",
			&utilityGenerated{
				kvs: []KV{
					newCssKV("border-inline-end-style", "var(--tw-border-style)"),
					newCssKV("border-inline-end-width", "4px"),
				},
			},
		))
		t.Run(scenario("border-e-(length:--my-width)",
			&utilityGenerated{
				kvs: []KV{
					newCssKV("border-inline-end-style", "var(--tw-border-style)"),
					newCssKV("border-inline-end-width", "var(--my-width)"),
				},
			},
		))
		t.Run(scenario("border-e-[3px]",
			&utilityGenerated{
				kvs: []KV{
					newCssKV("border-inline-end-style", "var(--tw-border-style)"),
					newCssKV("border-inline-end-width", "3px"),
				},
			},
		))

		t.Run(scenario("border-bs",
			&utilityGenerated{
				kvs: []KV{
					newCssKV("border-block-start-style", "var(--tw-border-style)"),
					newCssKV("border-block-start-width", "1px"),
				},
			},
		))
		t.Run(scenario("border-bs-4",
			&utilityGenerated{
				kvs: []KV{
					newCssKV("border-block-start-style", "var(--tw-border-style)"),
					newCssKV("border-block-start-width", "4px"),
				},
			},
		))
		t.Run(scenario("border-bs-(length:--my-width)",
			&utilityGenerated{
				kvs: []KV{
					newCssKV("border-block-start-style", "var(--tw-border-style)"),
					newCssKV("border-block-start-width", "var(--my-width)"),
				},
			},
		))
		t.Run(scenario("border-bs-[3px]",
			&utilityGenerated{
				kvs: []KV{
					newCssKV("border-block-start-style", "var(--tw-border-style)"),
					newCssKV("border-block-start-width", "3px"),
				},
			},
		))

		t.Run(scenario("border-be",
			&utilityGenerated{
				kvs: []KV{
					newCssKV("border-block-end-style", "var(--tw-border-style)"),
					newCssKV("border-block-end-width", "1px"),
				},
			},
		))
		t.Run(scenario("border-be-4",
			&utilityGenerated{
				kvs: []KV{
					newCssKV("border-block-end-style", "var(--tw-border-style)"),
					newCssKV("border-block-end-width", "4px"),
				},
			},
		))
		t.Run(scenario("border-be-(length:--my-width)",
			&utilityGenerated{
				kvs: []KV{
					newCssKV("border-block-end-style", "var(--tw-border-style)"),
					newCssKV("border-block-end-width", "var(--my-width)"),
				},
			},
		))
		t.Run(scenario("border-be-[3px]",
			&utilityGenerated{
				kvs: []KV{
					newCssKV("border-block-end-style", "var(--tw-border-style)"),
					newCssKV("border-block-end-width", "3px"),
				},
			},
		))

		t.Run(scenario("border-solid",
			&utilityGenerated{
				kvs: []KV{
					newCssKV("--tw-border-style", "solid"),
					newCssKV("border-style", "solid"),
				},
			},
		))
		t.Run(scenario("border-dashed",
			&utilityGenerated{
				kvs: []KV{
					newCssKV("--tw-border-style", "dashed"),
					newCssKV("border-style", "dashed"),
				},
			},
		))
		t.Run(scenario("border-dotted",
			&utilityGenerated{
				kvs: []KV{
					newCssKV("--tw-border-style", "dotted"),
					newCssKV("border-style", "dotted"),
				},
			},
		))
		t.Run(scenario("border-double",
			&utilityGenerated{
				kvs: []KV{
					newCssKV("--tw-border-style", "double"),
					newCssKV("border-style", "double"),
				},
			},
		))
		t.Run(scenario("border-hidden",
			&utilityGenerated{
				kvs: []KV{
					newCssKV("--tw-border-style", "hidden"),
					newCssKV("border-style", "hidden"),
				},
			},
		))
		t.Run(scenario("border-none",
			&utilityGenerated{
				kvs: []KV{
					newCssKV("--tw-border-style", "none"),
					newCssKV("border-style", "none"),
				},
			},
		))

		t.Run(scenario("border-collapse", newUtilityGeneratedKV("border-collapse", "collapse")))
		t.Run(scenario("border-separate", newUtilityGeneratedKV("border-collapse", "separate")))
		t.Run(scenario("border-spacing-1", &utilityGenerated{
			kvs: []KV{
				newCssKV("--tw-border-spacing-x", "calc(var(--spacing) * 1)"),
				newCssKV("--tw-border-spacing-y", "calc(var(--spacing) * 1)"),
				newCssKV("border-spacing", testBorderSpacingValue),
			},
		}))
		t.Run(scenario("border-spacing-[123px]", &utilityGenerated{
			kvs: []KV{
				newCssKV("--tw-border-spacing-x", "123px"),
				newCssKV("--tw-border-spacing-y", "123px"),
				newCssKV("border-spacing", testBorderSpacingValue),
			},
		}))
		t.Run(scenario("border-spacing-(--my-spacing)", &utilityGenerated{
			kvs: []KV{
				newCssKV("--tw-border-spacing-x", "var(--my-spacing)"),
				newCssKV("--tw-border-spacing-y", "var(--my-spacing)"),
				newCssKV("border-spacing", testBorderSpacingValue),
			},
		}))
		t.Run(scenario("border-spacing-x-1", &utilityGenerated{
			kvs: []KV{
				newCssKV("--tw-border-spacing-x", "calc(var(--spacing) * 1)"),
				newCssKV("border-spacing", testBorderSpacingValue),
			},
		}))
		t.Run(scenario("border-spacing-x-[123px]", &utilityGenerated{
			kvs: []KV{
				newCssKV("--tw-border-spacing-x", "123px"),
				newCssKV("border-spacing", testBorderSpacingValue),
			},
		}))
		t.Run(scenario("border-spacing-y-1", &utilityGenerated{
			kvs: []KV{
				newCssKV("--tw-border-spacing-y", "calc(var(--spacing) * 1)"),
				newCssKV("border-spacing", testBorderSpacingValue),
			},
		}))
		t.Run(scenario("border-spacing-y-[123px]", &utilityGenerated{
			kvs: []KV{
				newCssKV("--tw-border-spacing-y", "123px"),
				newCssKV("border-spacing", testBorderSpacingValue),
			},
		}))

		t.Run(scenario("divide-x",
			&utilityGenerated{
				others: []Query{
					{
						Name: "& > :not(:last-child)",
						KVs: []KV{
							newCssKV("--tw-divide-x-reverse", "0"),
							newCssKV("border-inline-style", "var(--tw-border-style)"),
							newCssKV("border-inline-start-width", "calc(1px * var(--tw-divide-x-reverse))"),
							newCssKV("border-inline-end-width", "calc(1px * calc(1 - var(--tw-divide-x-reverse)))"),
						},
					},
				},
			},
		))
		t.Run(scenario("divide-x-4",
			&utilityGenerated{
				others: []Query{
					{
						Name: "& > :not(:last-child)",
						KVs: []KV{
							newCssKV("--tw-divide-x-reverse", "0"),
							newCssKV("border-inline-style", "var(--tw-border-style)"),
							newCssKV("border-inline-start-width", "calc(4px * var(--tw-divide-x-reverse))"),
							newCssKV("border-inline-end-width", "calc(4px * calc(1 - var(--tw-divide-x-reverse)))"),
						},
					},
				},
			},
		))
		t.Run(scenario("divide-x-(length:--my-width)",
			&utilityGenerated{
				others: []Query{
					{
						Name: "& > :not(:last-child)",
						KVs: []KV{
							newCssKV("--tw-divide-x-reverse", "0"),
							newCssKV("border-inline-style", "var(--tw-border-style)"),
							newCssKV("border-inline-start-width", "calc(var(--my-width) * var(--tw-divide-x-reverse))"),
							newCssKV("border-inline-end-width", "calc(var(--my-width) * calc(1 - var(--tw-divide-x-reverse)))"),
						},
					},
				},
			},
		))
		t.Run(scenario("divide-x-[3px]",
			&utilityGenerated{
				others: []Query{
					{
						Name: "& > :not(:last-child)",
						KVs: []KV{
							newCssKV("--tw-divide-x-reverse", "0"),
							newCssKV("border-inline-style", "var(--tw-border-style)"),
							newCssKV("border-inline-start-width", "calc(3px * var(--tw-divide-x-reverse))"),
							newCssKV("border-inline-end-width", "calc(3px * calc(1 - var(--tw-divide-x-reverse)))"),
						},
					},
				},
			},
		))

		t.Run(scenario("divide-y",
			&utilityGenerated{
				others: []Query{
					{
						Name: "& > :not(:last-child)",
						KVs: []KV{
							newCssKV("--tw-divide-y-reverse", "0"),
							newCssKV("border-bottom-style", "var(--tw-border-style)"),
							newCssKV("border-top-style", "var(--tw-border-style)"),
							newCssKV("border-top-width", "calc(1px * var(--tw-divide-y-reverse))"),
							newCssKV("border-bottom-width", "calc(1px * calc(1 - var(--tw-divide-y-reverse)))"),
						},
					},
				},
			},
		))
		t.Run(scenario("divide-y-4",
			&utilityGenerated{
				others: []Query{
					{
						Name: "& > :not(:last-child)",
						KVs: []KV{
							newCssKV("--tw-divide-y-reverse", "0"),
							newCssKV("border-bottom-style", "var(--tw-border-style)"),
							newCssKV("border-top-style", "var(--tw-border-style)"),
							newCssKV("border-top-width", "calc(4px * var(--tw-divide-y-reverse))"),
							newCssKV("border-bottom-width", "calc(4px * calc(1 - var(--tw-divide-y-reverse)))"),
						},
					},
				},
			},
		))
		t.Run(scenario("divide-y-(length:--my-width)",
			&utilityGenerated{
				others: []Query{
					{
						Name: "& > :not(:last-child)",
						KVs: []KV{
							newCssKV("--tw-divide-y-reverse", "0"),
							newCssKV("border-bottom-style", "var(--tw-border-style)"),
							newCssKV("border-top-style", "var(--tw-border-style)"),
							newCssKV("border-top-width", "calc(var(--my-width) * var(--tw-divide-y-reverse))"),
							newCssKV("border-bottom-width", "calc(var(--my-width) * calc(1 - var(--tw-divide-y-reverse)))"),
						},
					},
				},
			},
		))
		t.Run(scenario("divide-y-[3px]",
			&utilityGenerated{
				others: []Query{
					{
						Name: "& > :not(:last-child)",
						KVs: []KV{
							newCssKV("--tw-divide-y-reverse", "0"),
							newCssKV("border-bottom-style", "var(--tw-border-style)"),
							newCssKV("border-top-style", "var(--tw-border-style)"),
							newCssKV("border-top-width", "calc(3px * var(--tw-divide-y-reverse))"),
							newCssKV("border-bottom-width", "calc(3px * calc(1 - var(--tw-divide-y-reverse)))"),
						},
					},
				},
			},
		))

		t.Run(scenario("divide-x-reverse",
			&utilityGenerated{
				others: []Query{
					{
						Name: "& > :not(:last-child)",
						KVs: []KV{
							newCssKV("--tw-divide-x-reverse", "1"),
						},
					},
				},
			},
		))
		t.Run(scenario("divide-y-reverse",
			&utilityGenerated{
				others: []Query{
					{
						Name: "& > :not(:last-child)",
						KVs: []KV{
							newCssKV("--tw-divide-y-reverse", "1"),
						},
					},
				},
			},
		))

		t.Run(scenario("divide-solid",
			&utilityGenerated{
				others: []Query{
					{
						Name: "& > :not(:last-child)",
						KVs: []KV{
							newCssKV("--tw-border-style", "solid"),
							newCssKV("border-style", "solid"),
						},
					},
				},
			},
		))
		t.Run(scenario("divide-dashed",
			&utilityGenerated{
				others: []Query{
					{
						Name: "& > :not(:last-child)",
						KVs: []KV{
							newCssKV("--tw-border-style", "dashed"),
							newCssKV("border-style", "dashed"),
						},
					},
				},
			},
		))
		t.Run(scenario("divide-dotted",
			&utilityGenerated{
				others: []Query{
					{
						Name: "& > :not(:last-child)",
						KVs: []KV{
							newCssKV("--tw-border-style", "dotted"),
							newCssKV("border-style", "dotted"),
						},
					},
				},
			},
		))
		t.Run(scenario("divide-double",
			&utilityGenerated{
				others: []Query{
					{
						Name: "& > :not(:last-child)",
						KVs: []KV{
							newCssKV("--tw-border-style", "double"),
							newCssKV("border-style", "double"),
						},
					},
				},
			},
		))
		t.Run(scenario("divide-hidden",
			&utilityGenerated{
				others: []Query{
					{
						Name: "& > :not(:last-child)",
						KVs: []KV{
							newCssKV("--tw-border-style", "hidden"),
							newCssKV("border-style", "hidden"),
						},
					},
				},
			},
		))
		t.Run(scenario("divide-none",
			&utilityGenerated{
				others: []Query{
					{
						Name: "& > :not(:last-child)",
						KVs: []KV{
							newCssKV("--tw-border-style", "none"),
							newCssKV("border-style", "none"),
						},
					},
				},
			},
		))

		t.Run(scenario("divide-amber-500",
			&utilityGenerated{
				others: []Query{
					{
						Name: "& > :not(:last-child)",
						KVs: []KV{
							newCssKV("border-color", "var(--color-amber-500)"),
						},
					},
				},
			},
		))
		t.Run(scenario("divide-red-500",
			&utilityGenerated{
				others: []Query{
					{
						Name: "& > :not(:last-child)",
						KVs: []KV{
							newCssKV("border-color", "var(--color-red-500)"),
						},
					},
				},
			},
		))
		t.Run(scenario("divide-(--my-color)",
			&utilityGenerated{
				others: []Query{
					{
						Name: "& > :not(:last-child)",
						KVs: []KV{
							newCssKV("border-color", "var(--my-color)"),
						},
					},
				},
			},
		))
		t.Run(scenario("divide-[#123456]",
			&utilityGenerated{
				others: []Query{
					{
						Name: "& > :not(:last-child)",
						KVs: []KV{
							newCssKV("border-color", "#123456"),
						},
					},
				},
			},
		))

		t.Run(scenario("outline",
			&utilityGenerated{
				kvs: []KV{
					newCssKV("outline-style", "var(--tw-outline-style)"),
					newCssKV("outline-width", "1px"),
				},
			},
		))
		t.Run(scenario("outline-4",
			&utilityGenerated{
				kvs: []KV{
					newCssKV("outline-style", "var(--tw-outline-style)"),
					newCssKV("outline-width", "4px"),
				},
			},
		))
		t.Run(scenario("outline-(length:--my-width)",
			&utilityGenerated{
				kvs: []KV{
					newCssKV("outline-style", "var(--tw-outline-style)"),
					newCssKV("outline-width", "var(--my-width)"),
				},
			},
		))
		t.Run(scenario("outline-[3px]",
			&utilityGenerated{
				kvs: []KV{
					newCssKV("outline-style", "var(--tw-outline-style)"),
					newCssKV("outline-width", "3px"),
				},
			},
		))

		t.Run(scenario("outline-(--my-outline-color)",
			newUtilityGeneratedKV("outline-color", "var(--my-outline-color)"),
		))
		t.Run(scenario("outline-[rgb(255_0_0)]",
			newUtilityGeneratedKV("outline-color", "rgb(255 0 0)"),
		))
		t.Run(scenario("outline-black",
			newUtilityGeneratedKV("outline-color", "var(--color-black)"),
		))
		t.Run(scenario("outline-current",
			newUtilityGeneratedKV("outline-color", "currentcolor"),
		))
		t.Run(scenario("outline-red-500",
			newUtilityGeneratedKV("outline-color", "var(--color-red-500)"),
		))
		t.Run(scenario("outline-transparent",
			newUtilityGeneratedKV("outline-color", "transparent"),
		))
		t.Run(scenario("outline-white",
			newUtilityGeneratedKV("outline-color", "var(--color-white)"),
		))

		t.Run(scenario("outline-solid",
			&utilityGenerated{
				kvs: []KV{
					newCssKV("--tw-outline-style", "solid"),
					newCssKV("outline-style", "solid"),
				},
			},
		))
		t.Run(scenario("outline-dashed",
			&utilityGenerated{
				kvs: []KV{
					newCssKV("--tw-outline-style", "dashed"),
					newCssKV("outline-style", "dashed"),
				},
			},
		))
		t.Run(scenario("outline-dotted",
			&utilityGenerated{
				kvs: []KV{
					newCssKV("--tw-outline-style", "dotted"),
					newCssKV("outline-style", "dotted"),
				},
			},
		))
		t.Run(scenario("outline-double",
			&utilityGenerated{
				kvs: []KV{
					newCssKV("--tw-outline-style", "double"),
					newCssKV("outline-style", "double"),
				},
			},
		))
		t.Run(scenario("outline-none",
			&utilityGenerated{
				kvs: []KV{
					newCssKV("--tw-outline-style", "none"),
					newCssKV("outline-style", "none"),
				},
			},
		))
		t.Run(scenario("outline-hidden",
			&utilityGenerated{
				kvs: []KV{
					newCssKV("--tw-outline-style", "none"),
					newCssKV("outline-style", "none"),
				},
				others: []Query{
					{
						Name: "@media (forced-colors: active)",
						KVs: []KV{
							newCssKV("outline", "2px solid transparent"),
							newCssKV("outline-offset", "2px"),
						},
					},
				},
			},
		))

		t.Run(scenario("outline-offset-4",
			newUtilityGeneratedKV("outline-offset", "4px"),
		))
		t.Run(scenario("-outline-offset-4",
			newUtilityGeneratedKV("outline-offset", "calc(4px * -1)"),
		))
		t.Run(scenario("outline-offset-(--my-offset)",
			newUtilityGeneratedKV("outline-offset", "var(--my-offset)"),
		))
		t.Run(scenario("outline-offset-[5px]",
			newUtilityGeneratedKV("outline-offset", "5px"),
		))

		t.Run(scenario("bg-red-500/50",
			newUtilityGeneratedKV("background-color", "color-mix(in oklab, var(--color-red-500) 50%, transparent)"),
		))
		t.Run(scenario("border-red-500/50",
			newUtilityGeneratedKV("border-color", "color-mix(in oklab, var(--color-red-500) 50%, transparent)"),
		))
		t.Run(scenario("outline-red-500/50",
			newUtilityGeneratedKV("outline-color", "color-mix(in oklab, var(--color-red-500) 50%, transparent)"),
		))
		t.Run(scenario("divide-red-500/50",
			&utilityGenerated{
				others: []Query{
					{
						Name: "& > :not(:last-child)",
						KVs: []KV{
							newCssKV("border-color", "color-mix(in oklab, var(--color-red-500) 50%, transparent)"),
						},
					},
				},
			},
		))
		t.Run(scenario("text-red-500/50",
			newUtilityGeneratedKV("color", "color-mix(in oklab, var(--color-red-500) 50%, transparent)"),
		))
		t.Run(scenario("decoration-red-500/50",
			newUtilityGeneratedKV("text-decoration-color", "color-mix(in oklab, var(--color-red-500) 50%, transparent)"),
		))

		t.Run(scenario("shadow-sm",
			&utilityGenerated{
				kvs: []KV{
					newCssKV("--tw-shadow", "0 1px 3px 0 var(--tw-shadow-color, rgb(0 0 0 / 0.1)), 0 1px 2px -1px var(--tw-shadow-color, rgb(0 0 0 / 0.1))"),
					newCssKV("box-shadow", testBoxShadowValue),
				},
			},
		))
		t.Run(scenario("shadow-md/50",
			&utilityGenerated{
				kvs: []KV{
					newCssKV("--tw-shadow-alpha", "50%"),
					newCssKV("--tw-shadow", "0 4px 6px -1px var(--tw-shadow-color, oklab(from rgb(0 0 0 / 0.1) l a b / 50%)), 0 2px 4px -2px var(--tw-shadow-color, oklab(from rgb(0 0 0 / 0.1) l a b / 50%))"),
					newCssKV("box-shadow", testBoxShadowValue),
				},
			},
		))
		t.Run(scenario("shadow-none",
			&utilityGenerated{
				kvs: []KV{
					newCssKV("--tw-shadow", genNullShadow),
					newCssKV("box-shadow", testBoxShadowValue),
				},
			},
		))
		t.Run(scenario("shadow-red-500",
			newUtilityGeneratedKV(
				"--tw-shadow-color",
				"color-mix(in oklab, var(--color-red-500) var(--tw-shadow-alpha), transparent)",
			),
		))
		t.Run(scenario("shadow-red-500/50",
			newUtilityGeneratedKV(
				"--tw-shadow-color",
				"color-mix(in oklab, color-mix(in oklab, var(--color-red-500) 50%, transparent) var(--tw-shadow-alpha), transparent)",
			),
		))
		t.Run(scenario("shadow-[0_4px_6px_rgba(0,0,0,0.1)]",
			&utilityGenerated{
				kvs: []KV{
					newCssKV("--tw-shadow", "0 4px 6px rgba(0, 0, 0, 0.1)"),
					newCssKV("box-shadow", testBoxShadowValue),
				},
			},
		))

		t.Run(scenario("inset-shadow-sm",
			&utilityGenerated{
				kvs: []KV{
					newCssKV("--tw-inset-shadow", "inset 0 2px 4px var(--tw-inset-shadow-color, rgb(0 0 0 / 0.05))"),
					newCssKV("box-shadow", testBoxShadowValue),
				},
			},
		))
		t.Run(scenario("inset-shadow-none",
			&utilityGenerated{
				kvs: []KV{
					newCssKV("--tw-inset-shadow", "inset 0 0 #0000"),
					newCssKV("box-shadow", testBoxShadowValue),
				},
			},
		))
		t.Run(scenario("inset-shadow-blue-500",
			newUtilityGeneratedKV(
				"--tw-inset-shadow-color",
				"color-mix(in oklab, var(--color-blue-500) var(--tw-inset-shadow-alpha), transparent)",
			),
		))

		t.Run(scenario("ring",
			&utilityGenerated{
				kvs: []KV{
					newCssKV("--tw-ring-shadow", testRingShadowValue("1px")),
					newCssKV("box-shadow", testBoxShadowValue),
				},
			},
		))
		t.Run(scenario("ring-2",
			&utilityGenerated{
				kvs: []KV{
					newCssKV("--tw-ring-shadow", testRingShadowValue("2px")),
					newCssKV("box-shadow", testBoxShadowValue),
				},
			},
		))
		t.Run(scenario("ring-red-500",
			newUtilityGeneratedKV("--tw-ring-color", "var(--color-red-500)"),
		))
		t.Run(scenario("ring-inset",
			newUtilityGeneratedKV("--tw-ring-inset", "inset"),
		))

		t.Run(scenario("inset-ring",
			&utilityGenerated{
				kvs: []KV{
					newCssKV("--tw-inset-ring-shadow", testInsetRingShadowValue("1px")),
					newCssKV("box-shadow", testBoxShadowValue),
				},
			},
		))
		t.Run(scenario("inset-ring-4",
			&utilityGenerated{
				kvs: []KV{
					newCssKV("--tw-inset-ring-shadow", testInsetRingShadowValue("4px")),
					newCssKV("box-shadow", testBoxShadowValue),
				},
			},
		))
		t.Run(scenario("inset-ring-green-500",
			newUtilityGeneratedKV("--tw-inset-ring-color", "var(--color-green-500)"),
		))

		filterWant := func(cssVar, value string) *utilityGenerated {
			return &utilityGenerated{
				kvs: []KV{
					newCssKV(cssVar, value),
					newCssKV("filter", testCssFilterValue),
				},
			}
		}

		t.Run(scenario("filter", &utilityGenerated{kvs: []KV{newCssKV("filter", testCssFilterValue)}}))
		t.Run(scenario("filter-none", newUtilityGeneratedKV("filter", "none")))
		t.Run(scenario("filter-(--test)", newUtilityGeneratedKV("filter", "var(--test)")))
		t.Run(scenario("filter-[var(--value)]", newUtilityGeneratedKV("filter", "var(--value)")))

		t.Run(scenario("blur-xl", filterWant("--tw-blur", "blur(var(--blur-xl))")))
		t.Run(scenario("blur-none", filterWant("--tw-blur", " ")))
		t.Run(scenario("blur-[4px]", filterWant("--tw-blur", "blur(4px)")))
		t.Run(scenario("blur-(--test)", filterWant("--tw-blur", "blur(var(--test))")))

		t.Run(scenario("brightness-50", filterWant("--tw-brightness", "brightness(50%)")))
		t.Run(scenario("brightness-[1.23]", filterWant("--tw-brightness", "brightness(1.23)")))
		t.Run(scenario("brightness-(--test)", filterWant("--tw-brightness", "brightness(var(--test))")))

		t.Run(scenario("contrast-50", filterWant("--tw-contrast", "contrast(50%)")))
		t.Run(scenario("contrast-[1.23]", filterWant("--tw-contrast", "contrast(1.23)")))
		t.Run(scenario("contrast-(--test)", filterWant("--tw-contrast", "contrast(var(--test))")))

		t.Run(scenario("grayscale", filterWant("--tw-grayscale", "grayscale(100%)")))
		t.Run(scenario("grayscale-0", filterWant("--tw-grayscale", "grayscale(0%)")))
		t.Run(scenario("grayscale-[var(--value)]", filterWant("--tw-grayscale", "grayscale(var(--value))")))
		t.Run(scenario("grayscale-(--test)", filterWant("--tw-grayscale", "grayscale(var(--test))")))

		t.Run(scenario("hue-rotate-15", filterWant("--tw-hue-rotate", "hue-rotate(15deg)")))
		t.Run(scenario("hue-rotate-[45deg]", filterWant("--tw-hue-rotate", "hue-rotate(45deg)")))
		t.Run(scenario("hue-rotate-(--test)", filterWant("--tw-hue-rotate", "hue-rotate(var(--test))")))
		t.Run(scenario("-hue-rotate-15", filterWant("--tw-hue-rotate", "hue-rotate(calc(15deg * -1))")))
		t.Run(scenario("-hue-rotate-[45deg]", filterWant("--tw-hue-rotate", "hue-rotate(calc(45deg * -1))")))

		t.Run(scenario("invert", filterWant("--tw-invert", "invert(100%)")))
		t.Run(scenario("invert-0", filterWant("--tw-invert", "invert(0%)")))
		t.Run(scenario("invert-[var(--value)]", filterWant("--tw-invert", "invert(var(--value))")))
		t.Run(scenario("invert-(--test)", filterWant("--tw-invert", "invert(var(--test))")))

		t.Run(scenario("saturate-0", filterWant("--tw-saturate", "saturate(0%)")))
		t.Run(scenario("saturate-[1.75]", filterWant("--tw-saturate", "saturate(1.75)")))
		t.Run(scenario("saturate-[var(--value)]", filterWant("--tw-saturate", "saturate(var(--value))")))
		t.Run(scenario("saturate-(--test)", filterWant("--tw-saturate", "saturate(var(--test))")))

		t.Run(scenario("sepia", filterWant("--tw-sepia", "sepia(100%)")))
		t.Run(scenario("sepia-0", filterWant("--tw-sepia", "sepia(0%)")))
		t.Run(scenario("sepia-[50%]", filterWant("--tw-sepia", "sepia(50%)")))
		t.Run(scenario("sepia-[var(--value)]", filterWant("--tw-sepia", "sepia(var(--value))")))
		t.Run(scenario("sepia-(--test)", filterWant("--tw-sepia", "sepia(var(--test))")))

		t.Run(scenario("drop-shadow", &utilityGenerated{
			kvs: []KV{
				newCssKV("--tw-drop-shadow-size", "drop-shadow(0 1px 1px var(--tw-drop-shadow-color, rgb(0 0 0 / 0.05)))"),
				newCssKV("--tw-drop-shadow", "drop-shadow(var(--drop-shadow))"),
				newCssKV("filter", testCssFilterValue),
			},
		}))
		t.Run(scenario("drop-shadow/25", &utilityGenerated{
			kvs: []KV{
				newCssKV("--tw-drop-shadow-alpha", "25%"),
				newCssKV("--tw-drop-shadow-size", "drop-shadow(0 1px 1px var(--tw-drop-shadow-color, oklab(from rgb(0 0 0 / 0.05) l a b / 25%)))"),
				newCssKV("--tw-drop-shadow", "drop-shadow(var(--drop-shadow))"),
				newCssKV("filter", testCssFilterValue),
			},
		}))
		t.Run(scenario("drop-shadow-xl", &utilityGenerated{
			kvs: []KV{
				newCssKV("--tw-drop-shadow-size", "drop-shadow(0 9px 7px var(--tw-drop-shadow-color, rgb(0 0 0 / 0.1)))"),
				newCssKV("--tw-drop-shadow", "drop-shadow(var(--drop-shadow-xl))"),
				newCssKV("filter", testCssFilterValue),
			},
		}))
		t.Run(scenario("drop-shadow-[0_0_red]", &utilityGenerated{
			kvs: []KV{
				newCssKV("--tw-drop-shadow-size", "drop-shadow(0 0 var(--tw-drop-shadow-color, red))"),
				newCssKV("--tw-drop-shadow", "var(--tw-drop-shadow-size)"),
				newCssKV("filter", testCssFilterValue),
			},
		}))
		t.Run(scenario("drop-shadow-none", filterWant("--tw-drop-shadow", " ")))
		t.Run(scenario("drop-shadow-inherit", &utilityGenerated{
			kvs: []KV{
				newCssKV("--tw-drop-shadow-color", "inherit"),
				newCssKV("--tw-drop-shadow", "var(--tw-drop-shadow-size)"),
			},
		}))
		t.Run(scenario("drop-shadow-red-500", &utilityGenerated{
			kvs: []KV{
				newCssKV("--tw-drop-shadow-color", "color-mix(in oklab, var(--color-red-500) var(--tw-drop-shadow-alpha), transparent)"),
				newCssKV("--tw-drop-shadow", "var(--tw-drop-shadow-size)"),
			},
		}))
		t.Run(scenario("drop-shadow-red-500/50", &utilityGenerated{
			kvs: []KV{
				newCssKV("--tw-drop-shadow-color", "color-mix(in oklab, color-mix(in oklab, var(--color-red-500) 50%, transparent) var(--tw-drop-shadow-alpha), transparent)"),
				newCssKV("--tw-drop-shadow", "var(--tw-drop-shadow-size)"),
			},
		}))
		t.Run(scenario("drop-shadow-(color:--my-color)", &utilityGenerated{
			kvs: []KV{
				newCssKV("--tw-drop-shadow-color", "color-mix(in oklab, var(--my-color) var(--tw-drop-shadow-alpha), transparent)"),
				newCssKV("--tw-drop-shadow", "var(--tw-drop-shadow-size)"),
			},
		}))
		t.Run(scenario("drop-shadow-calc", &utilityGenerated{
			kvs: []KV{
				newCssKV("--tw-drop-shadow-size", "drop-shadow(0 0 calc(1 * var(--spacing)) var(--tw-drop-shadow-color, black))"),
				newCssKV("--tw-drop-shadow", "drop-shadow(var(--drop-shadow-calc))"),
				newCssKV("filter", testCssFilterValue),
			},
		}))
		t.Run(scenario("drop-shadow-(--my-shadow)", &utilityGenerated{
			kvs: []KV{
				newCssKV("--tw-drop-shadow-size", "drop-shadow(var(--my-shadow))"),
				newCssKV("--tw-drop-shadow", "var(--tw-drop-shadow-size)"),
				newCssKV("filter", testCssFilterValue),
			},
		}))
		t.Run(scenario("drop-shadow-(--my-shadow)/25", &utilityGenerated{
			kvs: []KV{
				newCssKV("--tw-drop-shadow-alpha", "25%"),
				newCssKV("--tw-drop-shadow-size", "drop-shadow(var(--my-shadow))"),
				newCssKV("--tw-drop-shadow", "var(--tw-drop-shadow-size)"),
				newCssKV("filter", testCssFilterValue),
			},
		}))

		backdropWant := func(cssVar, value string) *utilityGenerated {
			return &utilityGenerated{
				kvs: []KV{
					newCssKV(cssVar, value),
					newCssKV("-webkit-backdrop-filter", testCssBackdropFilterValue),
					newCssKV("backdrop-filter", testCssBackdropFilterValue),
				},
			}
		}

		t.Run(scenario("backdrop-filter", &utilityGenerated{
			kvs: []KV{
				newCssKV("-webkit-backdrop-filter", testCssBackdropFilterValue),
				newCssKV("backdrop-filter", testCssBackdropFilterValue),
			},
		}))
		t.Run(scenario("backdrop-filter-none", &utilityGenerated{
			kvs: []KV{
				newCssKV("-webkit-backdrop-filter", "none"),
				newCssKV("backdrop-filter", "none"),
			},
		}))
		t.Run(scenario("backdrop-filter-[var(--value)]", &utilityGenerated{
			kvs: []KV{
				newCssKV("-webkit-backdrop-filter", "var(--value)"),
				newCssKV("backdrop-filter", "var(--value)"),
			},
		}))
		t.Run(scenario("backdrop-filter-(--test)", &utilityGenerated{
			kvs: []KV{
				newCssKV("-webkit-backdrop-filter", "var(--test)"),
				newCssKV("backdrop-filter", "var(--test)"),
			},
		}))

		t.Run(scenario("backdrop-blur-xl", backdropWant("--tw-backdrop-blur", "blur(var(--blur-xl))")))
		t.Run(scenario("backdrop-blur-none", backdropWant("--tw-backdrop-blur", " ")))
		t.Run(scenario("backdrop-blur-[4px]", backdropWant("--tw-backdrop-blur", "blur(4px)")))
		t.Run(scenario("backdrop-blur-(--test)", backdropWant("--tw-backdrop-blur", "blur(var(--test))")))

		t.Run(scenario("backdrop-brightness-50", backdropWant("--tw-backdrop-brightness", "brightness(50%)")))
		t.Run(scenario("backdrop-brightness-[1.23]", backdropWant("--tw-backdrop-brightness", "brightness(1.23)")))
		t.Run(scenario("backdrop-brightness-(--test)", backdropWant("--tw-backdrop-brightness", "brightness(var(--test))")))

		t.Run(scenario("backdrop-contrast-50", backdropWant("--tw-backdrop-contrast", "contrast(50%)")))
		t.Run(scenario("backdrop-contrast-[1.23]", backdropWant("--tw-backdrop-contrast", "contrast(1.23)")))
		t.Run(scenario("backdrop-contrast-(--test)", backdropWant("--tw-backdrop-contrast", "contrast(var(--test))")))

		t.Run(scenario("backdrop-grayscale", backdropWant("--tw-backdrop-grayscale", "grayscale(100%)")))
		t.Run(scenario("backdrop-grayscale-0", backdropWant("--tw-backdrop-grayscale", "grayscale(0%)")))
		t.Run(scenario("backdrop-grayscale-[var(--value)]", backdropWant("--tw-backdrop-grayscale", "grayscale(var(--value))")))
		t.Run(scenario("backdrop-grayscale-(--test)", backdropWant("--tw-backdrop-grayscale", "grayscale(var(--test))")))

		t.Run(scenario("backdrop-hue-rotate-15", backdropWant("--tw-backdrop-hue-rotate", "hue-rotate(15deg)")))
		t.Run(scenario("backdrop-hue-rotate-[45deg]", backdropWant("--tw-backdrop-hue-rotate", "hue-rotate(45deg)")))
		t.Run(scenario("backdrop-hue-rotate-(--test)", backdropWant("--tw-backdrop-hue-rotate", "hue-rotate(var(--test))")))
		t.Run(scenario("-backdrop-hue-rotate-15", backdropWant("--tw-backdrop-hue-rotate", "hue-rotate(calc(15deg * -1))")))
		t.Run(scenario("-backdrop-hue-rotate-[45deg]", backdropWant("--tw-backdrop-hue-rotate", "hue-rotate(calc(45deg * -1))")))

		t.Run(scenario("backdrop-invert", backdropWant("--tw-backdrop-invert", "invert(100%)")))
		t.Run(scenario("backdrop-invert-0", backdropWant("--tw-backdrop-invert", "invert(0%)")))
		t.Run(scenario("backdrop-invert-[var(--value)]", backdropWant("--tw-backdrop-invert", "invert(var(--value))")))
		t.Run(scenario("backdrop-invert-(--test)", backdropWant("--tw-backdrop-invert", "invert(var(--test))")))

		t.Run(scenario("backdrop-opacity-50", backdropWant("--tw-backdrop-opacity", "opacity(50%)")))
		t.Run(scenario("backdrop-opacity-71", backdropWant("--tw-backdrop-opacity", "opacity(71%)")))
		t.Run(scenario("backdrop-opacity-1.25", backdropWant("--tw-backdrop-opacity", "opacity(1.25%)")))
		t.Run(scenario("backdrop-opacity-[0.5]", backdropWant("--tw-backdrop-opacity", "opacity(0.5)")))
		t.Run(scenario("backdrop-opacity-(--test)", backdropWant("--tw-backdrop-opacity", "opacity(var(--test))")))

		t.Run(scenario("backdrop-saturate-0", backdropWant("--tw-backdrop-saturate", "saturate(0%)")))
		t.Run(scenario("backdrop-saturate-[1.75]", backdropWant("--tw-backdrop-saturate", "saturate(1.75)")))
		t.Run(scenario("backdrop-saturate-[var(--value)]", backdropWant("--tw-backdrop-saturate", "saturate(var(--value))")))
		t.Run(scenario("backdrop-saturate-(--test)", backdropWant("--tw-backdrop-saturate", "saturate(var(--test))")))

		t.Run(scenario("backdrop-sepia", backdropWant("--tw-backdrop-sepia", "sepia(100%)")))
		t.Run(scenario("backdrop-sepia-0", backdropWant("--tw-backdrop-sepia", "sepia(0%)")))
		t.Run(scenario("backdrop-sepia-[50%]", backdropWant("--tw-backdrop-sepia", "sepia(50%)")))
		t.Run(scenario("backdrop-sepia-[var(--value)]", backdropWant("--tw-backdrop-sepia", "sepia(var(--value))")))
		t.Run(scenario("backdrop-sepia-(--test)", backdropWant("--tw-backdrop-sepia", "sepia(var(--test))")))

		transitionWant := func(property string) *utilityGenerated {
			return g.transitionWithDefaults(context.Background(), property)
		}

		t.Run(scenario("transition", transitionWant(transitionDefaultProperty)))
		t.Run(scenario("transition-none", newUtilityGeneratedKV("transition-property", "none")))
		t.Run(scenario("transition-all", transitionWant("all")))
		t.Run(scenario("transition-colors", transitionWant(transitionColorsProperty)))
		t.Run(scenario("transition-opacity", transitionWant("opacity")))
		t.Run(scenario("transition-shadow", transitionWant("box-shadow")))
		t.Run(scenario("transition-transform", transitionWant(transitionTransformProperty)))
		t.Run(scenario("transition-[var(--value)]", transitionWant("var(--value)")))
		t.Run(scenario("transition-discrete", newUtilityGeneratedKV("transition-behavior", "allow-discrete")))
		t.Run(scenario("transition-normal", newUtilityGeneratedKV("transition-behavior", "normal")))

		t.Run(scenario("delay-123", newUtilityGeneratedKV("transition-delay", ".123s")))
		t.Run(scenario("delay-200", newUtilityGeneratedKV("transition-delay", ".2s")))
		t.Run(scenario("delay-[300ms]", newUtilityGeneratedKV("transition-delay", ".3s")))
		t.Run(scenario("delay-(--test)", newUtilityGeneratedKV("transition-delay", "--test")))

		t.Run(scenario("duration-123", &utilityGenerated{
			kvs: []KV{
				newCssKV("--tw-duration", ".123s"),
				newCssKV("transition-duration", ".123s"),
			},
		}))
		t.Run(scenario("duration-200", &utilityGenerated{
			kvs: []KV{
				newCssKV("--tw-duration", ".2s"),
				newCssKV("transition-duration", ".2s"),
			},
		}))
		t.Run(scenario("duration-[300ms]", &utilityGenerated{
			kvs: []KV{
				newCssKV("--tw-duration", ".3s"),
				newCssKV("transition-duration", ".3s"),
			},
		}))
		t.Run(scenario("duration-initial", newUtilityGeneratedKV("--tw-duration", "initial")))

		t.Run(scenario("ease-in", &utilityGenerated{
			kvs: []KV{
				newCssKV("--tw-ease", "var(--ease-in)"),
				newCssKV("transition-timing-function", "var(--ease-in)"),
			},
		}))
		t.Run(scenario("ease-out", &utilityGenerated{
			kvs: []KV{
				newCssKV("--tw-ease", "var(--ease-out)"),
				newCssKV("transition-timing-function", "var(--ease-out)"),
			},
		}))
		t.Run(scenario("ease-in-out", &utilityGenerated{
			kvs: []KV{
				newCssKV("--tw-ease", "var(--ease-in-out)"),
				newCssKV("transition-timing-function", "var(--ease-in-out)"),
			},
		}))
		t.Run(scenario("ease-linear", &utilityGenerated{
			kvs: []KV{
				newCssKV("--tw-ease", "var(--ease-linear)"),
				newCssKV("transition-timing-function", "var(--ease-linear)"),
			},
		}))
		t.Run(scenario("ease-initial", newUtilityGeneratedKV("--tw-ease", "initial")))
		t.Run(scenario("ease-[var(--value)]", &utilityGenerated{
			kvs: []KV{
				newCssKV("--tw-ease", "var(--value)"),
				newCssKV("transition-timing-function", "var(--value)"),
			},
		}))

		t.Run(scenario("animate-pulse-ring", newUtilityGeneratedKV("animation", "var(--animate-pulse-ring)")))
		t.Run(scenario("animate-spin", newUtilityGeneratedKV("animation", "var(--animate-spin)")))
		t.Run(scenario("animate-none", newUtilityGeneratedKV("animation", "none")))
		t.Run(scenario("animate-[bounce_1s_infinite]", newUtilityGeneratedKV("animation", "bounce 1s infinite")))
		t.Run(scenario("animate-(--test)", newUtilityGeneratedKV("animation", "var(--test)")))

		t.Run(scenario("translate-none", newUtilityGeneratedKV("translate", "none")))
		t.Run(scenario("translate-full", &utilityGenerated{
			kvs: []KV{
				newCssKV("--tw-translate-x", "100%"),
				newCssKV("--tw-translate-y", "100%"),
				newCssKV("translate", testTranslateValue),
			},
		}))
		t.Run(scenario("-translate-full", &utilityGenerated{
			kvs: []KV{
				newCssKV("--tw-translate-x", "-100%"),
				newCssKV("--tw-translate-y", "-100%"),
				newCssKV("translate", testTranslateValue),
			},
		}))
		t.Run(scenario("translate-1/2", &utilityGenerated{
			kvs: []KV{
				newCssKV("--tw-translate-x", "calc(1 / 2 * 100%)"),
				newCssKV("--tw-translate-y", "calc(1 / 2 * 100%)"),
				newCssKV("translate", testTranslateValue),
			},
		}))
		t.Run(scenario("-translate-1/2", &utilityGenerated{
			kvs: []KV{
				newCssKV("--tw-translate-x", "calc(calc(1 / 2 * 100%) * -1)"),
				newCssKV("--tw-translate-y", "calc(calc(1 / 2 * 100%) * -1)"),
				newCssKV("translate", testTranslateValue),
			},
		}))
		t.Run(scenario("translate-x-px", &utilityGenerated{
			kvs: []KV{
				newCssKV("--tw-translate-x", "1px"),
				newCssKV("translate", testTranslateValue),
			},
		}))
		t.Run(scenario("-translate-x-1/2", &utilityGenerated{
			kvs: []KV{
				newCssKV("--tw-translate-x", "calc(calc(1 / 2 * 100%) * -1)"),
				newCssKV("translate", testTranslateValue),
			},
		}))
		t.Run(scenario("translate-y-full", &utilityGenerated{
			kvs: []KV{
				newCssKV("--tw-translate-y", "100%"),
				newCssKV("translate", testTranslateValue),
			},
		}))
		t.Run(scenario("translate-z-px", &utilityGenerated{
			kvs: []KV{
				newCssKV("--tw-translate-z", "1px"),
				newCssKV("translate", testTranslate3dValue),
			},
		}))
		t.Run(scenario("translate-3d", &utilityGenerated{
			kvs: []KV{newCssKV("translate", testTranslate3dValue)},
		}))
		t.Run(scenario("translate-[123px]", &utilityGenerated{
			kvs: []KV{
				newCssKV("--tw-translate-x", "123px"),
				newCssKV("--tw-translate-y", "123px"),
				newCssKV("translate", testTranslateValue),
			},
		}))
		t.Run(scenario("translate-(--test)", &utilityGenerated{
			kvs: []KV{
				newCssKV("--tw-translate-x", "var(--test)"),
				newCssKV("--tw-translate-y", "var(--test)"),
				newCssKV("translate", testTranslateValue),
			},
		}))
		t.Run(scenario("-translate-(--test)", &utilityGenerated{
			kvs: []KV{
				newCssKV("--tw-translate-x", "calc(var(--test) * -1)"),
				newCssKV("--tw-translate-y", "calc(var(--test) * -1)"),
				newCssKV("translate", testTranslateValue),
			},
		}))
		t.Run(scenario("translate-x-(--test)", &utilityGenerated{
			kvs: []KV{
				newCssKV("--tw-translate-x", "var(--test)"),
				newCssKV("translate", testTranslateValue),
			},
		}))
		t.Run(scenario("translate-y-(--test)", &utilityGenerated{
			kvs: []KV{
				newCssKV("--tw-translate-y", "var(--test)"),
				newCssKV("translate", testTranslateValue),
			},
		}))
		t.Run(scenario("translate-z-(--test)", &utilityGenerated{
			kvs: []KV{
				newCssKV("--tw-translate-z", "var(--test)"),
				newCssKV("translate", testTranslate3dValue),
			},
		}))

		t.Run(scenario("scale-none", newUtilityGeneratedKV("scale", "none")))
		t.Run(scenario("scale-50", &utilityGenerated{
			kvs: []KV{
				newCssKV("--tw-scale-x", "var(--scale-50)"),
				newCssKV("--tw-scale-y", "var(--scale-50)"),
				newCssKV("--tw-scale-z", "var(--scale-50)"),
				newCssKV("scale", testScaleValue),
			},
		}))
		t.Run(scenario("-scale-50", &utilityGenerated{
			kvs: []KV{
				newCssKV("--tw-scale-x", "calc(var(--scale-50) * -1)"),
				newCssKV("--tw-scale-y", "calc(var(--scale-50) * -1)"),
				newCssKV("--tw-scale-z", "calc(var(--scale-50) * -1)"),
				newCssKV("scale", testScaleValue),
			},
		}))
		t.Run(scenario("scale-[2]", newUtilityGeneratedKV("scale", "2")))
		t.Run(scenario("scale-x-50", &utilityGenerated{
			kvs: []KV{
				newCssKV("--tw-scale-x", "var(--scale-50)"),
				newCssKV("scale", testScaleValue),
			},
		}))
		t.Run(scenario("scale-z-50", &utilityGenerated{
			kvs: []KV{
				newCssKV("--tw-scale-z", "var(--scale-50)"),
				newCssKV("scale", testScale3dValue),
			},
		}))
		t.Run(scenario("scale-3d", &utilityGenerated{
			kvs: []KV{newCssKV("scale", testScale3dValue)},
		}))
		t.Run(scenario("scale-(--test)", &utilityGenerated{
			kvs: []KV{
				newCssKV("--tw-scale-x", "var(--test)"),
				newCssKV("--tw-scale-y", "var(--test)"),
				newCssKV("--tw-scale-z", "var(--test)"),
				newCssKV("scale", testScaleValue),
			},
		}))
		t.Run(scenario("-scale-(--test)", &utilityGenerated{
			kvs: []KV{
				newCssKV("--tw-scale-x", "calc(var(--test) * -1)"),
				newCssKV("--tw-scale-y", "calc(var(--test) * -1)"),
				newCssKV("--tw-scale-z", "calc(var(--test) * -1)"),
				newCssKV("scale", testScaleValue),
			},
		}))
		t.Run(scenario("scale-x-(--test)", &utilityGenerated{
			kvs: []KV{
				newCssKV("--tw-scale-x", "var(--test)"),
				newCssKV("scale", testScaleValue),
			},
		}))
		t.Run(scenario("scale-z-(--test)", &utilityGenerated{
			kvs: []KV{
				newCssKV("--tw-scale-z", "var(--test)"),
				newCssKV("scale", testScale3dValue),
			},
		}))

		t.Run(scenario("rotate-none", newUtilityGeneratedKV("rotate", "none")))
		t.Run(scenario("rotate-45", newUtilityGeneratedKV("rotate", "var(--rotate-45)")))
		t.Run(scenario("-rotate-45", newUtilityGeneratedKV("rotate", "calc(var(--rotate-45) * -1)")))
		t.Run(scenario("rotate-[123deg]", newUtilityGeneratedKV("rotate", "123deg")))
		t.Run(scenario("-rotate-[123deg]", newUtilityGeneratedKV("rotate", "-123deg")))
		t.Run(scenario("rotate-(--var)", newUtilityGeneratedKV("rotate", "var(--var)")))
		t.Run(scenario("-rotate-(--var)", newUtilityGeneratedKV("rotate", "calc(var(--var) * -1)")))
		t.Run(scenario("rotate-x-45", &utilityGenerated{
			kvs: []KV{
				newCssKV("--tw-rotate-x", "rotateX(var(--rotate-45))"),
				newCssKV("transform", testTransformValue),
			},
		}))
		t.Run(scenario("-rotate-x-45", &utilityGenerated{
			kvs: []KV{
				newCssKV("--tw-rotate-x", "rotateX(calc(var(--rotate-45) * -1))"),
				newCssKV("transform", testTransformValue),
			},
		}))
		t.Run(scenario("rotate-y-(--test)", &utilityGenerated{
			kvs: []KV{
				newCssKV("--tw-rotate-y", "rotateY(var(--test))"),
				newCssKV("transform", testTransformValue),
			},
		}))
		t.Run(scenario("-rotate-y-(--test)", &utilityGenerated{
			kvs: []KV{
				newCssKV("--tw-rotate-y", "rotateY(calc(var(--test) * -1))"),
				newCssKV("transform", testTransformValue),
			},
		}))
		t.Run(scenario("rotate-z-(--test)", &utilityGenerated{
			kvs: []KV{
				newCssKV("--tw-rotate-z", "rotateZ(var(--test))"),
				newCssKV("transform", testTransformValue),
			},
		}))

		t.Run(scenario("skew-6", &utilityGenerated{
			kvs: []KV{
				newCssKV("--tw-skew-x", "skewX(var(--skew-6))"),
				newCssKV("--tw-skew-y", "skewY(var(--skew-6))"),
				newCssKV("transform", testTransformValue),
			},
		}))
		t.Run(scenario("-skew-x-6", &utilityGenerated{
			kvs: []KV{
				newCssKV("--tw-skew-x", "skewX(calc(var(--skew-6) * -1))"),
				newCssKV("transform", testTransformValue),
			},
		}))
		t.Run(scenario("skew-[123deg]", &utilityGenerated{
			kvs: []KV{
				newCssKV("--tw-skew-x", "skewX(123deg)"),
				newCssKV("--tw-skew-y", "skewY(123deg)"),
				newCssKV("transform", testTransformValue),
			},
		}))
		t.Run(scenario("skew-(--test)", &utilityGenerated{
			kvs: []KV{
				newCssKV("--tw-skew-x", "skewX(var(--test))"),
				newCssKV("--tw-skew-y", "skewY(var(--test))"),
				newCssKV("transform", testTransformValue),
			},
		}))
		t.Run(scenario("-skew-(--test)", &utilityGenerated{
			kvs: []KV{
				newCssKV("--tw-skew-x", "skewX(calc(var(--test) * -1))"),
				newCssKV("--tw-skew-y", "skewY(calc(var(--test) * -1))"),
				newCssKV("transform", testTransformValue),
			},
		}))
		t.Run(scenario("skew-x-(--test)", &utilityGenerated{
			kvs: []KV{
				newCssKV("--tw-skew-x", "skewX(var(--test))"),
				newCssKV("transform", testTransformValue),
			},
		}))
		t.Run(scenario("-skew-x-(--test)", &utilityGenerated{
			kvs: []KV{
				newCssKV("--tw-skew-x", "skewX(calc(var(--test) * -1))"),
				newCssKV("transform", testTransformValue),
			},
		}))

		t.Run(scenario("transform", &utilityGenerated{
			kvs: []KV{newCssKV("transform", testTransformValue)},
		}))
		t.Run(scenario("transform-gpu", &utilityGenerated{
			kvs: []KV{newCssKV("transform", "translateZ(0) "+testTransformValue)},
		}))
		t.Run(scenario("transform-none", newUtilityGeneratedKV("transform", "none")))
		t.Run(scenario("transform-flat", newUtilityGeneratedKV("transform-style", "flat")))
		t.Run(scenario("transform-3d", newUtilityGeneratedKV("transform-style", "preserve-3d")))
		t.Run(scenario("transform-content", newUtilityGeneratedKV("transform-box", "content-box")))
		t.Run(scenario("transform-[scaleZ(2)_rotateY(45deg)]", newUtilityGeneratedKV("transform", "scaleZ(2) rotateY(45deg)")))
		t.Run(scenario("transform-(--test)", newUtilityGeneratedKV("transform", "var(--test)")))

		t.Run(scenario("origin-center", newUtilityGeneratedKV("transform-origin", "var(--transform-origin-center)")))
		t.Run(scenario("origin-top-left", newUtilityGeneratedKV("transform-origin", "var(--transform-origin-top-left)")))
		t.Run(scenario("origin-[50px_100px]", newUtilityGeneratedKV("transform-origin", "50px 100px")))
		t.Run(scenario("origin-(--test)", newUtilityGeneratedKV("transform-origin", "var(--test)")))

		t.Run(scenario("perspective-none", newUtilityGeneratedKV("perspective", "none")))
		t.Run(scenario("perspective-normal", newUtilityGeneratedKV("perspective", "var(--perspective-normal)")))
		t.Run(scenario("perspective-[456px]", newUtilityGeneratedKV("perspective", "456px")))
		t.Run(scenario("perspective-(--test)", newUtilityGeneratedKV("perspective", "var(--test)")))
		t.Run(scenario("perspective-origin-top", newUtilityGeneratedKV("perspective-origin", "var(--perspective-origin-top)")))
		t.Run(scenario("perspective-origin-(--test)", newUtilityGeneratedKV("perspective-origin", "var(--test)")))

		t.Run(scenario("backface-visible", newUtilityGeneratedKV("backface-visibility", "visible")))
		t.Run(scenario("backface-hidden", newUtilityGeneratedKV("backface-visibility", "hidden")))

		t.Run(scenario("zoom-50", newUtilityGeneratedKV("zoom", "50%")))
		t.Run(scenario("zoom-[var(--zoom)]", newUtilityGeneratedKV("zoom", "var(--zoom)")))
		t.Run(scenario("zoom-(--test)", newUtilityGeneratedKV("zoom", "var(--test)")))

		t.Run(scenario("pointer-events-none", newUtilityGeneratedKV("pointer-events", "none")))
		t.Run(scenario("pointer-events-auto", newUtilityGeneratedKV("pointer-events", "auto")))

		t.Run(scenario("cursor-pointer", newUtilityGeneratedKV("cursor", "pointer")))
		t.Run(scenario("cursor-not-allowed", newUtilityGeneratedKV("cursor", "not-allowed")))
		t.Run(scenario("cursor-col-resize", newUtilityGeneratedKV("cursor", "col-resize")))
		t.Run(scenario("cursor-[var(--value)]", newUtilityGeneratedKV("cursor", "var(--value)")))
		t.Run(scenario("cursor-(--test)", newUtilityGeneratedKV("cursor", "var(--test)")))

		t.Run(scenario("touch-auto", newUtilityGeneratedKV("touch-action", "auto")))
		t.Run(scenario("touch-none", newUtilityGeneratedKV("touch-action", "none")))
		t.Run(scenario("touch-manipulation", newUtilityGeneratedKV("touch-action", "manipulation")))
		t.Run(scenario("touch-pan-x", &utilityGenerated{
			kvs: []KV{
				newCssKV("--tw-pan-x", "pan-x"),
				newCssKV("touch-action", testTouchActionValue),
			},
		}))
		t.Run(scenario("touch-pan-left", &utilityGenerated{
			kvs: []KV{
				newCssKV("--tw-pan-x", "pan-left"),
				newCssKV("touch-action", testTouchActionValue),
			},
		}))
		t.Run(scenario("touch-pan-y", &utilityGenerated{
			kvs: []KV{
				newCssKV("--tw-pan-y", "pan-y"),
				newCssKV("touch-action", testTouchActionValue),
			},
		}))
		t.Run(scenario("touch-pinch-zoom", &utilityGenerated{
			kvs: []KV{
				newCssKV("--tw-pinch-zoom", "pinch-zoom"),
				newCssKV("touch-action", testTouchActionValue),
			},
		}))

		t.Run(scenario("select-none", &utilityGenerated{kvs: []KV{
			newCssKV("-webkit-user-select", "none"),
			newCssKV("user-select", "none"),
		}}))
		t.Run(scenario("select-text", &utilityGenerated{kvs: []KV{
			newCssKV("-webkit-user-select", "text"),
			newCssKV("user-select", "text"),
		}}))
		t.Run(scenario("select-all", &utilityGenerated{kvs: []KV{
			newCssKV("-webkit-user-select", "all"),
			newCssKV("user-select", "all"),
		}}))
		t.Run(scenario("select-auto", &utilityGenerated{kvs: []KV{
			newCssKV("-webkit-user-select", "auto"),
			newCssKV("user-select", "auto"),
		}}))

		t.Run(scenario("resize", newUtilityGeneratedKV("resize", "both")))
		t.Run(scenario("resize-none", newUtilityGeneratedKV("resize", "none")))
		t.Run(scenario("resize-x", newUtilityGeneratedKV("resize", "horizontal")))
		t.Run(scenario("resize-y", newUtilityGeneratedKV("resize", "vertical")))

		t.Run(scenario("snap-none", newUtilityGeneratedKV("scroll-snap-type", "none")))
		t.Run(scenario("snap-x", newUtilityGeneratedKV("scroll-snap-type", "x var(--tw-scroll-snap-strictness)")))
		t.Run(scenario("snap-y", newUtilityGeneratedKV("scroll-snap-type", "y var(--tw-scroll-snap-strictness)")))
		t.Run(scenario("snap-both", newUtilityGeneratedKV("scroll-snap-type", "both var(--tw-scroll-snap-strictness)")))
		t.Run(scenario("snap-mandatory", newUtilityGeneratedKV("--tw-scroll-snap-strictness", "mandatory")))
		t.Run(scenario("snap-proximity", newUtilityGeneratedKV("--tw-scroll-snap-strictness", "proximity")))
		t.Run(scenario("snap-align-none", newUtilityGeneratedKV("scroll-snap-align", "none")))
		t.Run(scenario("snap-start", newUtilityGeneratedKV("scroll-snap-align", "start")))
		t.Run(scenario("snap-end", newUtilityGeneratedKV("scroll-snap-align", "end")))
		t.Run(scenario("snap-center", newUtilityGeneratedKV("scroll-snap-align", "center")))
		t.Run(scenario("snap-normal", newUtilityGeneratedKV("scroll-snap-stop", "normal")))
		t.Run(scenario("snap-always", newUtilityGeneratedKV("scroll-snap-stop", "always")))

		t.Run(scenario("scroll-auto", newUtilityGeneratedKV("scroll-behavior", "auto")))
		t.Run(scenario("scroll-smooth", newUtilityGeneratedKV("scroll-behavior", "smooth")))
		t.Run(scenario("scroll-m-4", newUtilityGeneratedKV("scroll-margin", "calc(var(--spacing) * 4)")))
		t.Run(scenario("scroll-m-4.5", newUtilityGeneratedKV("scroll-margin", "calc(var(--spacing) * 4.5)")))
		t.Run(scenario("-scroll-m-2.5", newUtilityGeneratedKV("scroll-margin", "calc(var(--spacing) * -2.5)")))
		t.Run(scenario("-scroll-m-4", newUtilityGeneratedKV("scroll-margin", "calc(var(--spacing) * -4)")))
		t.Run(scenario("scroll-mx-4", newUtilityGeneratedKV("scroll-margin-inline", "calc(var(--spacing) * 4)")))
		t.Run(scenario("scroll-mt-4", newUtilityGeneratedKV("scroll-margin-top", "calc(var(--spacing) * 4)")))
		t.Run(scenario("scroll-ms-4", newUtilityGeneratedKV("scroll-margin-inline-start", "calc(var(--spacing) * 4)")))
		t.Run(scenario("scroll-p-2", newUtilityGeneratedKV("scroll-padding", "calc(var(--spacing) * 2)")))
		t.Run(scenario("scroll-px-2", newUtilityGeneratedKV("scroll-padding-inline", "calc(var(--spacing) * 2)")))
		t.Run(scenario("scroll-pt-2", newUtilityGeneratedKV("scroll-padding-top", "calc(var(--spacing) * 2)")))
		t.Run(scenario("scroll-pbs-2", newUtilityGeneratedKV("scroll-padding-block-start", "calc(var(--spacing) * 2)")))
		t.Run(scenario("scroll-m-[4px]", newUtilityGeneratedKV("scroll-margin", "4px")))
		t.Run(scenario("scroll-m-(--test)", newUtilityGeneratedKV("scroll-margin", "var(--test)")))
		t.Run(scenario("-scroll-m-(--test)", newUtilityGeneratedKV("scroll-margin", "var(--test)")))
		t.Run(scenario("scroll-p-(--test)", newUtilityGeneratedKV("scroll-padding", "var(--test)")))

		t.Run(scenario("scrollbar-auto", newUtilityGeneratedKV("scrollbar-width", "auto")))
		t.Run(scenario("scrollbar-thin", newUtilityGeneratedKV("scrollbar-width", "thin")))
		t.Run(scenario("scrollbar-none", newUtilityGeneratedKV("scrollbar-width", "none")))
		t.Run(scenario("scrollbar-gutter-auto", newUtilityGeneratedKV("scrollbar-gutter", "auto")))
		t.Run(scenario("scrollbar-gutter-stable", newUtilityGeneratedKV("scrollbar-gutter", "stable")))
		t.Run(scenario("scrollbar-gutter-both", newUtilityGeneratedKV("scrollbar-gutter", "stable both-edges")))
		t.Run(scenario("scrollbar-thumb-red-500", &utilityGenerated{
			kvs: []KV{
				newCssKV("--tw-scrollbar-thumb", "var(--color-red-500)"),
				newCssKV("scrollbar-color", testScrollbarColorValue),
			},
		}))
		t.Run(scenario("scrollbar-track-blue-500", &utilityGenerated{
			kvs: []KV{
				newCssKV("--tw-scrollbar-track", "var(--color-blue-500)"),
				newCssKV("scrollbar-color", testScrollbarColorValue),
			},
		}))
		t.Run(scenario("scrollbar-thumb-(--test)", &utilityGenerated{
			kvs: []KV{
				newCssKV("--tw-scrollbar-thumb", "var(--test)"),
				newCssKV("scrollbar-color", testScrollbarColorValue),
			},
		}))

		t.Run(scenario("appearance-none", newUtilityGeneratedKV("appearance", "none")))
		t.Run(scenario("appearance-auto", newUtilityGeneratedKV("appearance", "auto")))

		t.Run(scenario("scheme-normal", newUtilityGeneratedKV("color-scheme", "normal")))
		t.Run(scenario("scheme-dark", newUtilityGeneratedKV("color-scheme", "dark")))
		t.Run(scenario("scheme-light", newUtilityGeneratedKV("color-scheme", "light")))
		t.Run(scenario("scheme-light-dark", newUtilityGeneratedKV("color-scheme", "light dark")))
		t.Run(scenario("scheme-only-dark", newUtilityGeneratedKV("color-scheme", "only dark")))
		t.Run(scenario("scheme-only-light", newUtilityGeneratedKV("color-scheme", "only light")))

		t.Run(scenario("accent-auto", newUtilityGeneratedKV("accent-color", "auto")))
		t.Run(scenario("accent-red-500", newUtilityGeneratedKV("accent-color", "var(--color-red-500)")))
		t.Run(scenario("accent-(--test)", newUtilityGeneratedKV("accent-color", "var(--test)")))
		t.Run(scenario("accent-[#0088cc]", newUtilityGeneratedKV("accent-color", "#0088cc")))

		t.Run(scenario("caret-blue-500", newUtilityGeneratedKV("caret-color", "var(--color-blue-500)")))
		t.Run(scenario("caret-(--test)", newUtilityGeneratedKV("caret-color", "var(--test)")))
		t.Run(scenario("caret-[red]", newUtilityGeneratedKV("caret-color", "red")))

		t.Run(scenario("field-sizing-content", newUtilityGeneratedKV("field-sizing", "content")))
		t.Run(scenario("field-sizing-fixed", newUtilityGeneratedKV("field-sizing", "fixed")))

		t.Run(scenario("will-change-auto", newUtilityGeneratedKV("will-change", "auto")))
		t.Run(scenario("will-change-scroll", newUtilityGeneratedKV("will-change", "scroll-position")))
		t.Run(scenario("will-change-contents", newUtilityGeneratedKV("will-change", "contents")))
		t.Run(scenario("will-change-transform", newUtilityGeneratedKV("will-change", "transform")))
		t.Run(scenario("will-change-[var(--value)]", newUtilityGeneratedKV("will-change", "var(--value)")))
		t.Run(scenario("will-change-(--test)", newUtilityGeneratedKV("will-change", "var(--test)")))

		t.Run(scenario("fill-none", newUtilityGeneratedKV("fill", "none")))
		t.Run(scenario("fill-red-500", newUtilityGeneratedKV("fill", "var(--color-red-500)")))
		t.Run(scenario("fill-blue-500", newUtilityGeneratedKV("fill", "var(--color-blue-500)")))
		t.Run(scenario("fill-current", newUtilityGeneratedKV("fill", "currentcolor")))
		t.Run(scenario("fill-(--test)", newUtilityGeneratedKV("fill", "var(--test)")))
		t.Run(scenario("fill-[#0088cc]", newUtilityGeneratedKV("fill", "#0088cc")))

		t.Run(scenario("stroke-none", newUtilityGeneratedKV("stroke", "none")))
		t.Run(scenario("stroke-red-500", newUtilityGeneratedKV("stroke", "var(--color-red-500)")))
		t.Run(scenario("stroke-blue-500", newUtilityGeneratedKV("stroke", "var(--color-blue-500)")))
		t.Run(scenario("stroke-current", newUtilityGeneratedKV("stroke", "currentcolor")))
		t.Run(scenario("stroke-(--test)", newUtilityGeneratedKV("stroke", "var(--test)")))
		t.Run(scenario("stroke-[#0088cc]", newUtilityGeneratedKV("stroke", "#0088cc")))
		t.Run(scenario("stroke-0", newUtilityGeneratedKV("stroke-width", "var(--stroke-width-0)")))
		t.Run(scenario("stroke-1", newUtilityGeneratedKV("stroke-width", "var(--stroke-width-1)")))
		t.Run(scenario("stroke-2", newUtilityGeneratedKV("stroke-width", "var(--stroke-width-2)")))
		t.Run(scenario("stroke-width-2", newUtilityGeneratedKV("stroke-width", "var(--stroke-width-2)")))
		t.Run(scenario("stroke-[12px]", newUtilityGeneratedKV("stroke-width", "12px")))
		t.Run(scenario("stroke-[50%]", newUtilityGeneratedKV("stroke-width", "50%")))
		t.Run(scenario("stroke-[1.5]", newUtilityGeneratedKV("stroke-width", "1.5px")))
		t.Run(scenario("stroke-[length:var(--my-width)]", newUtilityGeneratedKV("stroke-width", "var(--my-width)")))
		t.Run(scenario("stroke-[color:red]", newUtilityGeneratedKV("stroke", "red")))

		t.Run(scenario("forced-color-adjust-none", newUtilityGeneratedKV("forced-color-adjust", "none")))
		t.Run(scenario("forced-color-adjust-auto", newUtilityGeneratedKV("forced-color-adjust", "auto")))

		t.Run(scenario("shadow",
			&utilityGenerated{
				kvs: []KV{
					newCssKV("--tw-shadow", "0 1px 3px 0 var(--tw-shadow-color, rgb(0 0 0 / 0.1)), 0 1px 2px -1px var(--tw-shadow-color, rgb(0 0 0 / 0.1))"),
					newCssKV("box-shadow", testBoxShadowValue),
				},
			},
		))
		t.Run(scenario("shadow/50",
			&utilityGenerated{
				kvs: []KV{
					newCssKV("--tw-shadow-alpha", "50%"),
					newCssKV("--tw-shadow", "0 1px 3px 0 var(--tw-shadow-color, oklab(from rgb(0 0 0 / 0.1) l a b / 50%)), 0 1px 2px -1px var(--tw-shadow-color, oklab(from rgb(0 0 0 / 0.1) l a b / 50%))"),
					newCssKV("box-shadow", testBoxShadowValue),
				},
			},
		))
		t.Run(scenario("shadow-lg/25",
			&utilityGenerated{
				kvs: []KV{
					newCssKV("--tw-shadow-alpha", "25%"),
					newCssKV("--tw-shadow", "0 10px 15px -3px var(--tw-shadow-color, oklab(from rgb(0 0 0 / 0.1) l a b / 25%)), 0 4px 6px -4px var(--tw-shadow-color, oklab(from rgb(0 0 0 / 0.1) l a b / 25%))"),
					newCssKV("box-shadow", testBoxShadowValue),
				},
			},
		))
		t.Run(scenario("shadow-inherit",
			newUtilityGeneratedKV("--tw-shadow-color", "inherit"),
		))
		t.Run(scenario("shadow-transparent",
			newUtilityGeneratedKV("--tw-shadow-color", "transparent"),
		))
		t.Run(scenario("shadow-current",
			newUtilityGeneratedKV("--tw-shadow-color", "currentcolor"),
		))
		t.Run(scenario("shadow-(color:--my-shadow-color)",
			newUtilityGeneratedKV(
				"--tw-shadow-color",
				"color-mix(in oklab, var(--my-shadow-color) var(--tw-shadow-alpha), transparent)",
			),
		))
		t.Run(scenario("shadow-(--my-shadow)",
			&utilityGenerated{
				kvs: []KV{
					newCssKV("--tw-shadow", "var(--my-shadow)"),
					newCssKV("box-shadow", testBoxShadowValue),
				},
			},
		))
		t.Run(scenario("shadow-(--my-shadow)/50",
			&utilityGenerated{
				kvs: []KV{
					newCssKV("--tw-shadow-alpha", "50%"),
					newCssKV("--tw-shadow", "var(--my-shadow)"),
					newCssKV("box-shadow", testBoxShadowValue),
				},
			},
		))
		t.Run(scenario("shadow-[color:#ff0000]",
			newUtilityGeneratedKV(
				"--tw-shadow-color",
				"color-mix(in oklab, #ff0000 var(--tw-shadow-alpha), transparent)",
			),
		))
		t.Run(scenario("shadow-[color:#ff0000]/50",
			newUtilityGeneratedKV(
				"--tw-shadow-color",
				"color-mix(in oklab, color-mix(in oklab, #ff0000 50%, transparent) var(--tw-shadow-alpha), transparent)",
			),
		))

		t.Run(scenario("inset-shadow",
			&utilityGenerated{
				kvs: []KV{
					newCssKV("--tw-inset-shadow", "inset 0 2px 4px var(--tw-inset-shadow-color, rgb(0 0 0 / 0.05))"),
					newCssKV("box-shadow", testBoxShadowValue),
				},
			},
		))
		t.Run(scenario("inset-shadow/40",
			&utilityGenerated{
				kvs: []KV{
					newCssKV("--tw-inset-shadow-alpha", "40%"),
					newCssKV("--tw-inset-shadow", "inset 0 2px 4px var(--tw-inset-shadow-color, oklab(from rgb(0 0 0 / 0.05) l a b / 40%))"),
					newCssKV("box-shadow", testBoxShadowValue),
				},
			},
		))
		t.Run(scenario("inset-shadow-xs",
			&utilityGenerated{
				kvs: []KV{
					newCssKV("--tw-inset-shadow", "inset 0 1px 1px var(--tw-inset-shadow-color, rgb(0 0 0 / 0.05))"),
					newCssKV("box-shadow", testBoxShadowValue),
				},
			},
		))
		t.Run(scenario("inset-shadow-xs/30",
			&utilityGenerated{
				kvs: []KV{
					newCssKV("--tw-inset-shadow-alpha", "30%"),
					newCssKV("--tw-inset-shadow", "inset 0 1px 1px var(--tw-inset-shadow-color, oklab(from rgb(0 0 0 / 0.05) l a b / 30%))"),
					newCssKV("box-shadow", testBoxShadowValue),
				},
			},
		))
		t.Run(scenario("inset-shadow-inherit",
			newUtilityGeneratedKV("--tw-inset-shadow-color", "inherit"),
		))
		t.Run(scenario("inset-shadow-current",
			newUtilityGeneratedKV("--tw-inset-shadow-color", "currentcolor"),
		))
		t.Run(scenario("inset-shadow-blue-500/50",
			newUtilityGeneratedKV(
				"--tw-inset-shadow-color",
				"color-mix(in oklab, color-mix(in oklab, var(--color-blue-500) 50%, transparent) var(--tw-inset-shadow-alpha), transparent)",
			),
		))
		t.Run(scenario("inset-shadow-(color:--my-inset-color)",
			newUtilityGeneratedKV(
				"--tw-inset-shadow-color",
				"color-mix(in oklab, var(--my-inset-color) var(--tw-inset-shadow-alpha), transparent)",
			),
		))
		t.Run(scenario("inset-shadow-(--my-inset-shadow)",
			&utilityGenerated{
				kvs: []KV{
					newCssKV("--tw-inset-shadow", "var(--my-inset-shadow)"),
					newCssKV("box-shadow", testBoxShadowValue),
				},
			},
		))

		t.Run(scenario("ring-4",
			&utilityGenerated{
				kvs: []KV{
					newCssKV("--tw-ring-shadow", testRingShadowValue("4px")),
					newCssKV("box-shadow", testBoxShadowValue),
				},
			},
		))
		t.Run(scenario("ring-8",
			&utilityGenerated{
				kvs: []KV{
					newCssKV("--tw-ring-shadow", testRingShadowValue("8px")),
					newCssKV("box-shadow", testBoxShadowValue),
				},
			},
		))
		t.Run(scenario("ring-current",
			newUtilityGeneratedKV("--tw-ring-color", "currentcolor"),
		))
		t.Run(scenario("ring-red-500/50",
			newUtilityGeneratedKV("--tw-ring-color", "color-mix(in oklab, var(--color-red-500) 50%, transparent)"),
		))
		t.Run(scenario("ring-(length:--my-ring-width)",
			&utilityGenerated{
				kvs: []KV{
					newCssKV("--tw-ring-shadow", testRingShadowValue("var(--my-ring-width)")),
					newCssKV("box-shadow", testBoxShadowValue),
				},
			},
		))
		t.Run(scenario("ring-[3px]",
			&utilityGenerated{
				kvs: []KV{
					newCssKV("--tw-ring-shadow", testRingShadowValue("3px")),
					newCssKV("box-shadow", testBoxShadowValue),
				},
			},
		))

		t.Run(scenario("inset-ring-(length:--my-inset-ring)",
			&utilityGenerated{
				kvs: []KV{
					newCssKV("--tw-inset-ring-shadow", testInsetRingShadowValue("var(--my-inset-ring)")),
					newCssKV("box-shadow", testBoxShadowValue),
				},
			},
		))
		t.Run(scenario("inset-ring-[2px]",
			&utilityGenerated{
				kvs: []KV{
					newCssKV("--tw-inset-ring-shadow", testInsetRingShadowValue("2px")),
					newCssKV("box-shadow", testBoxShadowValue),
				},
			},
		))
		t.Run(scenario("inset-ring-blue-500/50",
			newUtilityGeneratedKV("--tw-inset-ring-color", "color-mix(in oklab, var(--color-blue-500) 50%, transparent)"),
		))

		t.Run(scenario("accent-red-500/50",
			newUtilityGeneratedKV("accent-color", "color-mix(in oklab, var(--color-red-500) 50%, transparent)"),
		))
		t.Run(scenario("caret-red-500/50",
			newUtilityGeneratedKV("caret-color", "color-mix(in oklab, var(--color-red-500) 50%, transparent)"),
		))
		t.Run(scenario("fill-red-500/50",
			newUtilityGeneratedKV("fill", "color-mix(in oklab, var(--color-red-500) 50%, transparent)"),
		))
		t.Run(scenario("stroke-red-500/50",
			newUtilityGeneratedKV("stroke", "color-mix(in oklab, var(--color-red-500) 50%, transparent)"),
		))
	})
}

func TestCondition(t *testing.T) {
	g := &gen{
		varer: &testVarer{},
		registerer: &registerer{
			registry: map[string]struct{}{},
		},
	}
	p := parser.Must()

	scenario := func(class string, want []string) (string, func(*testing.T)) {
		return class, func(t *testing.T) {
			t.Helper()

			ast, err := p.ParseString("", class)
			if err != nil {
				t.Fatalf("parse: %v", err)
			}

			variants := ast.Classes[0].Variants()
			if len(variants) != 1 {
				t.Fatalf("expected 1 variant, got %d", len(variants))
			}

			got, err := g.condition(t.Context(), variants[0])
			if err != nil {
				t.Fatalf("condition: %v", err)
			}

			if !slices.Equal(got, want) {
				t.Errorf("got %v != want %v", got, want)
			}
		}
	}

	stack := func(class string, want []string) (string, func(*testing.T)) {
		return class, func(t *testing.T) {
			t.Helper()

			ast, err := p.ParseString("", class)
			if err != nil {
				t.Fatalf("parse: %v", err)
			}

			var got []string
			for _, v := range ast.Classes[0].Variants() {
				conds, err := g.condition(t.Context(), v)
				if err != nil {
					t.Fatalf("condition: %v", err)
				}
				got = append(got, conds...)
			}

			if !slices.Equal(got, want) {
				t.Errorf("got %v != want %v", got, want)
			}
		}
	}

	t.Run("pseudo-classes", func(t *testing.T) {
		t.Run("interactive", func(t *testing.T) {
			t.Run(scenario("hover:flex", []string{"&:hover", "@media (hover: hover)"}))
			t.Run(scenario("focus:flex", []string{"&:focus"}))
			t.Run(scenario("focus-within:flex", []string{"&:focus-within"}))
			t.Run(scenario("focus-visible:flex", []string{"&:focus-visible"}))
			t.Run(scenario("active:flex", []string{"&:active"}))
			t.Run(scenario("visited:flex", []string{"&:visited"}))
			t.Run(scenario("target:flex", []string{"&:target"}))
		})

		t.Run("structural", func(t *testing.T) {
			t.Run(scenario("first:flex", []string{"&:first-child"}))
			t.Run(scenario("last:flex", []string{"&:last-child"}))
			t.Run(scenario("only:flex", []string{"&:only-child"}))
			t.Run(scenario("odd:flex", []string{"&:nth-child(odd)"}))
			t.Run(scenario("even:flex", []string{"&:nth-child(even)"}))
			t.Run(scenario("first-of-type:flex", []string{"&:first-of-type"}))
			t.Run(scenario("last-of-type:flex", []string{"&:last-of-type"}))
			t.Run(scenario("only-of-type:flex", []string{"&:only-of-type"}))
			t.Run(scenario("empty:flex", []string{"&:empty"}))
		})

		t.Run("nth", func(t *testing.T) {
			t.Run(scenario("nth-2:flex", []string{"&:nth-child(2)"}))
			t.Run(scenario("nth-last-2:flex", []string{"&:nth-last-child(2)"}))
			t.Run(scenario("nth-of-type-3:flex", []string{"&:nth-of-type(3)"}))
			t.Run(scenario("nth-last-of-type-2:flex", []string{"&:nth-last-of-type(2)"}))
		})

		t.Run("form", func(t *testing.T) {
			t.Run(scenario("required:flex", []string{"&:required"}))
			t.Run(scenario("optional:flex", []string{"&:optional"}))
			t.Run(scenario("valid:flex", []string{"&:valid"}))
			t.Run(scenario("invalid:flex", []string{"&:invalid"}))
			t.Run(scenario("user-valid:flex", []string{"&:user-valid"}))
			t.Run(scenario("user-invalid:flex", []string{"&:user-invalid"}))
			t.Run(scenario("in-range:flex", []string{"&:in-range"}))
			t.Run(scenario("out-of-range:flex", []string{"&:out-of-range"}))
			t.Run(scenario("read-only:flex", []string{"&:read-only"}))
			t.Run(scenario("placeholder-shown:flex", []string{"&:placeholder-shown"}))
			t.Run(scenario("autofill:flex", []string{"&:autofill"}))
			t.Run(scenario("enabled:flex", []string{"&:enabled"}))
			t.Run(scenario("disabled:flex", []string{"&:disabled"}))
			t.Run(scenario("checked:flex", []string{"&:checked"}))
			t.Run(scenario("indeterminate:flex", []string{"&:indeterminate"}))
			t.Run(scenario("default:flex", []string{"&:default"}))
		})

		t.Run("misc", func(t *testing.T) {
			t.Run(scenario("open:flex", []string{"&:is([open], :popover-open, :open)"}))
			t.Run(scenario("inert:flex", []string{"&:is([inert], [inert] *)"}))
		})
	})

	t.Run("has", func(t *testing.T) {
		t.Run(scenario("has-checked:flex", []string{"&:has(:checked)"}))
		t.Run(scenario("has-[:focus]:flex", []string{"&:has(:focus)"}))
		t.Run(scenario("has-[img]:flex", []string{"&:has(img)"}))
		t.Run(scenario("has-[a]:flex", []string{"&:has(a)"}))

		t.Run("group", func(t *testing.T) {
			t.Run(scenario("group-has-[a]:flex", []string{"&:is(:where(.group):has(a) *)"}))
			t.Run(scenario("group-has-checked:flex", []string{"&:is(:where(.group):has(:checked) *)"}))
		})

		t.Run("peer", func(t *testing.T) {
			t.Run(scenario("peer-has-checked:flex", []string{"&:is(:where(.peer):has(:checked) ~ *)"}))
		})

		t.Run("not", func(t *testing.T) {
			t.Run(scenario("not-has-checked:flex", []string{"&:not(:has(:checked))"}))
		})
	})

	t.Run("group", func(t *testing.T) {
		t.Run(scenario("group-hover:flex", []string{"&:is(:where(.group):hover *)", "@media (hover: hover)"}))
		t.Run(scenario("group-focus-visible:flex", []string{"&:is(:where(.group):focus-visible *)"}))
		t.Run(scenario("group-hover/item:flex", []string{"&:is(:where(.group\\/item):hover *)", "@media (hover: hover)"}))
		t.Run(scenario("group-[&_p]:flex", []string{"&:is(:where(.group) p *)"}))
		t.Run(scenario("group-[:nth-of-type(3)_&]:block", []string{"&:is(:nth-of-type(3) :where(.group) *)"}))
		t.Run(scenario("group-[.is-published]:flex", []string{"&:is(:where(.group).is-published *)"}))
	})

	t.Run("peer", func(t *testing.T) {
		t.Run(scenario("peer-focus:flex", []string{"&:is(:where(.peer):focus ~ *)"}))
		t.Run(scenario("peer-disabled:flex", []string{"&:is(:where(.peer):disabled ~ *)"}))
		t.Run(scenario("peer-checked/published:flex", []string{"&:is(:where(.peer\\/published):checked ~ *)"}))
		t.Run(scenario("peer-[.is-dirty]:flex", []string{"&:is(:where(.peer).is-dirty ~ *)"}))
	})

	t.Run("not", func(t *testing.T) {
		t.Run(scenario("not-hover:flex", []string{"&:not(:hover)", "@media (hover: hover)"}))
		t.Run(scenario("not-focus:flex", []string{"&:not(:focus)"}))
		t.Run(scenario("not-disabled:flex", []string{"&:not(:disabled)"}))
	})

	t.Run("in", func(t *testing.T) {
		t.Run(scenario("in-[p]:flex", []string{":where(:is(p)) &"}))
		t.Run(scenario("in-data-visible:flex", []string{":where([data-visible]) &"}))
		t.Run(scenario("in-[.group]:flex", []string{":where(.group) &"}))
		t.Run(scenario("in-focus:flex", []string{":where(:focus) &"}))
		t.Run(scenario("in-focus-visible:flex", []string{":where(:focus-visible) &"}))
		t.Run(scenario("in-hover:flex", []string{":where(:hover) &", "@media (hover: hover)"}))
		t.Run(scenario("in-checked:flex", []string{":where(:checked) &"}))
		t.Run(scenario("in-odd:flex", []string{":where(:nth-child(odd)) &"}))
	})

	t.Run("pseudo-elements", func(t *testing.T) {
		t.Run(scenario("before:flex", []string{"&::before"}))
		t.Run(scenario("after:flex", []string{"&::after"}))
		t.Run(scenario("first-letter:flex", []string{"&::first-letter"}))
		t.Run(scenario("first-line:flex", []string{"&::first-line"}))
		t.Run(scenario("marker:flex", []string{"& *::marker, &::marker, & *::-webkit-details-marker, &::-webkit-details-marker"}))
		t.Run(scenario("selection:flex", []string{"& *::selection, &::selection"}))
		t.Run(scenario("file:flex", []string{"&::file-selector-button"}))
		t.Run(scenario("backdrop:flex", []string{"&::backdrop"}))
		t.Run(scenario("placeholder:flex", []string{"&::placeholder"}))
		t.Run(scenario("details-content:flex", []string{"&::details-content"}))
	})

	t.Run("child-selectors", func(t *testing.T) {
		t.Run(scenario("*:flex", []string{":is(& > *)"}))
		t.Run(scenario("**:flex", []string{":is(& *)"}))
	})

	t.Run("media", func(t *testing.T) {
		t.Run("breakpoints", func(t *testing.T) {
			t.Run(scenario("sm:flex", []string{"@media (width >= 40rem)"}))
			t.Run(scenario("md:flex", []string{"@media (width >= 48rem)"}))
			t.Run(scenario("lg:flex", []string{"@media (width >= 64rem)"}))
			t.Run(scenario("xl:flex", []string{"@media (width >= 80rem)"}))
			t.Run(scenario("2xl:flex", []string{"@media (width >= 96rem)"}))
			t.Run(scenario("max-sm:flex", []string{"@media (width < 40rem)"}))
			t.Run(scenario("max-md:flex", []string{"@media (width < 48rem)"}))
			t.Run(scenario("max-lg:flex", []string{"@media (width < 64rem)"}))
			t.Run(scenario("max-xl:flex", []string{"@media (width < 80rem)"}))
			t.Run(scenario("max-2xl:flex", []string{"@media (width < 96rem)"}))
			t.Run(scenario("min-lg:flex", []string{"@media (width >= 64rem)"}))
			t.Run(scenario("min-[700px]:flex", []string{"@media (width >= 700px)"}))
			t.Run(scenario("max-[700px]:flex", []string{"@media (width < 700px)"}))
		})

		t.Run("prefers", func(t *testing.T) {
			t.Run(scenario("dark:flex", []string{"@media (prefers-color-scheme: dark)"}))
			t.Run(scenario("motion-safe:flex", []string{"@media (prefers-reduced-motion: no-preference)"}))
			t.Run(scenario("motion-reduce:flex", []string{"@media (prefers-reduced-motion: reduce)"}))
			t.Run(scenario("contrast-more:flex", []string{"@media (prefers-contrast: more)"}))
			t.Run(scenario("contrast-less:flex", []string{"@media (prefers-contrast: less)"}))
		})

		t.Run("other", func(t *testing.T) {
			t.Run(scenario("print:flex", []string{"@media print"}))
			t.Run(scenario("portrait:flex", []string{"@media (orientation: portrait)"}))
			t.Run(scenario("landscape:flex", []string{"@media (orientation: landscape)"}))
			t.Run(scenario("forced-colors:flex", []string{"@media (forced-colors: active)"}))
			t.Run(scenario("inverted-colors:flex", []string{"@media (inverted-colors: inverted)"}))
			t.Run(scenario("pointer-fine:flex", []string{"@media (pointer: fine)"}))
			t.Run(scenario("pointer-coarse:flex", []string{"@media (pointer: coarse)"}))
			t.Run(scenario("pointer-none:flex", []string{"@media (pointer: none)"}))
			t.Run(scenario("any-pointer-fine:flex", []string{"@media (any-pointer: fine)"}))
			t.Run(scenario("any-pointer-coarse:flex", []string{"@media (any-pointer: coarse)"}))
			t.Run(scenario("any-pointer-none:flex", []string{"@media (any-pointer: none)"}))
			t.Run(scenario("noscript:flex", []string{"@media (scripting: none)"}))
		})

		t.Run("supports", func(t *testing.T) {
			t.Run(scenario("supports-[display:flex]:flex", []string{"@supports (display:flex)"}))
			t.Run(scenario("not-supports-[display:grid]:flex", []string{"@supports not (display:grid)"}))
		})
	})

	t.Run("container-queries", func(t *testing.T) {
		t.Run(scenario("@md:flex", []string{"@container (width >= var(--container-md))"}))
		t.Run(scenario("@lg:flex", []string{"@container (width >= var(--container-lg))"}))
		t.Run(scenario("@min-lg:flex", []string{"@container (width >= var(--container-lg))"}))
		t.Run(scenario("@max-lg:flex", []string{"@container (width < var(--container-lg))"}))
		t.Run(scenario("@[700px]:flex", []string{"@container (width >= 700px)"}))
		t.Run(scenario("@md/sidebar:flex", []string{"@container sidebar (width >= var(--container-md))"}))
		t.Run(scenario("@max-xl/main:flex", []string{"@container main (width < var(--container-xl))"}))
	})

	t.Run("attributes", func(t *testing.T) {
		t.Run("aria", func(t *testing.T) {
			t.Run(scenario("aria-selected:flex", []string{`&[aria-selected="true"]`}))
			t.Run(scenario("aria-busy:flex", []string{`&[aria-busy="true"]`}))
			t.Run(scenario("aria-checked:flex", []string{`&[aria-checked="true"]`}))
			t.Run(scenario("aria-disabled:flex", []string{`&[aria-disabled="true"]`}))
			t.Run(scenario("aria-expanded:flex", []string{`&[aria-expanded="true"]`}))
			t.Run(scenario("aria-hidden:flex", []string{`&[aria-hidden="true"]`}))
			t.Run(scenario("aria-pressed:flex", []string{`&[aria-pressed="true"]`}))
			t.Run(scenario("aria-readonly:flex", []string{`&[aria-readonly="true"]`}))
			t.Run(scenario("aria-required:flex", []string{`&[aria-required="true"]`}))
		})

		t.Run("data", func(t *testing.T) {
			t.Run(scenario("data-[state=open]:flex", []string{"&[data-state=open]"}))
		})
	})

	t.Run("arbitrary", func(t *testing.T) {
		t.Run(scenario("[&:nth-child(3)]:flex", []string{"&:nth-child(3)"}))
		t.Run(scenario("[&_p]:flex", []string{"& p"}))
		t.Run(stack("[&_p]:hover:flex", []string{"& p", "&:hover", "@media (hover: hover)"}))
	})

	t.Run("stacked", func(t *testing.T) {
		t.Run(stack("dark:md:hover:flex", []string{
			"@media (prefers-color-scheme: dark)",
			"@media (width >= 48rem)",
			"&:hover",
			"@media (hover: hover)",
		}))
		t.Run(stack("hover:focus:active:flex", []string{
			"&:hover",
			"@media (hover: hover)",
			"&:focus",
			"&:active",
		}))
		t.Run(stack("dark:has-checked:flex", []string{
			"@media (prefers-color-scheme: dark)",
			"&:has(:checked)",
		}))
	})
}

func TestRegistererVars(t *testing.T) {
	p := parser.Must()
	g := &gen{
		varer: &testVarer{},
		registerer: &registerer{
			registry: map[string]struct{}{},
		},
	}

	ast, err := p.ParseString("", "shadow-sm")
	if err != nil {
		t.Fatal(err)
	}

	if _, err := g.utility(t.Context(), ast.Classes[0].Utility()); err != nil {
		t.Fatal(err)
	}

	for _, key := range []string{
		"--tw-shadow",
		"--tw-shadow-color",
		"--tw-inset-shadow",
		"--tw-inset-ring-shadow",
		"--tw-ring-offset-shadow",
		"--tw-ring-shadow",
	} {
		if _, ok := g.registerer.registry[key]; !ok {
			t.Errorf("expected %q to be registered", key)
		}
	}
}
