package parser_test

import (
	"testing"

	"github.com/alecthomas/participle/v2"

	"github.com/thinkofher/upupidae/internal/parser"
)

var p = parser.Must()

func parse(t *testing.T, input string) *parser.ClassList {
	t.Helper()
	cl, err := p.ParseString("", input)
	if err != nil {
		t.Fatal(err)
	}
	return cl
}

func TestRoundTrip(t *testing.T) {
	cases := []string{
		// simple utilities
		"flex",
		"p-4",
		"bg-red-500",
		"text-2xl",
		"p-0.5",

		// variants
		"hover:bg-red-500",
		"sm:hover:bg-red-500",
		"dark:md:hover:bg-gradient-to-r",
		"2xl:text-lg",

		// arbitrary values
		"w-[100px]",
		"bg-[#ff0000]",
		"text-[14px]",
		"min-w-[100px]",

		// arbitrary properties
		"[display:flex]",

		// arbitrary variants
		"[&:nth-child(3)]:underline",

		// important / negative
		"!text-red-500",
		"-mt-4",
		"!-mt-4",

		// opacity / modifier
		"bg-red-500/50",
		"bg-red-500/0.5",
		"bg-red-500/[.5]",

		// group / peer variants
		"group-hover:text-white",
		"group/sidebar:text-white",
		"peer-[.is-dirty]:peer-required:block",
		"data-[size=large]:p-8",
		"aria-[sort=ascending]:bg-[url('img.png')]/50",

		// child selector variants
		"*:flex",
		"**:grid",
		"hover:*:underline",

		// container queries
		"@md:flex",
		"@max-lg:hidden",
		"@min-xl:block",
		"@[700px]:flex",
		"@lg/sidebar:flex",
		"@container",
		"@container/sidebar",
		"@container-normal",

		// multiple classes
		"flex items-center p-4",

		// complex
		"sm:hover:!-bg-red-500/50",
	}

	for _, input := range cases {
		t.Run(input, func(t *testing.T) {
			cl := parse(t, input)
			if got := cl.String(); got != input {
				t.Errorf("round trip:\n  got:  %q\n  want: %q", got, input)
			}
		})
	}
}

func TestEmpty(t *testing.T) {
	cl := parse(t, "")
	if len(cl.Classes) != 0 {
		t.Errorf("expected 0 classes, got %d", len(cl.Classes))
	}
}

func TestSimpleUtility(t *testing.T) {
	cl := parse(t, "flex")
	requireClasses(t, cl, 1)
	c := cl.Classes[0]
	u := c.Utility()
	if len(c.Variants()) != 0 {
		t.Errorf("expected 0 variants, got %d", len(c.Variants()))
	}
	if u.Important {
		t.Error("expected Important=false")
	}
	if u.Negative {
		t.Error("expected Negative=false")
	}
	if u.Name == nil || u.Name.Head() != "flex" {
		t.Errorf("Head: got %q, want %q", u.Name.Head(), "flex")
	}
}

func TestIntegerPart(t *testing.T) {
	cl := parse(t, "bg-red-500")
	u := cl.Classes[0].Utility()
	if u.Name == nil {
		t.Fatal("expected non-nil Name")
	}
	if len(u.Name.Parts()) != 2 {
		t.Fatalf("expected 2 parts, got %d", len(u.Name.Parts()))
	}
	red := u.Name.Parts()[0]
	if red.Ident != "red" {
		t.Errorf("part 0: expected Ident %q, got %q", "red", red.Ident)
	}
	num := u.Name.Parts()[1]
	if num.Int == nil || *num.Int != 500 {
		t.Errorf("part 1: expected Int 500, got %v", num)
	}
}

func TestFloatPart(t *testing.T) {
	cl := parse(t, "p-0.5")
	u := cl.Classes[0].Utility()
	if u.Name == nil || len(u.Name.Parts()) != 1 {
		t.Fatal("expected 1 part")
	}
	part := u.Name.Parts()[0]
	if part.Float == nil || *part.Float != 0.5 {
		t.Errorf("expected Float 0.5, got %v", part)
	}
}

func TestAlphaNumPart(t *testing.T) {
	cl := parse(t, "text-2xl")
	u := cl.Classes[0].Utility()
	if u.Name == nil || len(u.Name.Parts()) != 1 {
		t.Fatal("expected 1 part")
	}
	part := u.Name.Parts()[0]
	if part.AlphaNum != "2xl" {
		t.Errorf("expected AlphaNum %q, got %q", "2xl", part.AlphaNum)
	}
}

func TestAlphaNumVariant(t *testing.T) {
	cl := parse(t, "2xl:text-lg")
	c := cl.Classes[0]
	variants := c.Variants()
	if len(variants) != 1 {
		t.Fatalf("expected 1 variant, got %d", len(variants))
	}
	if variants[0].Name == nil || variants[0].Name.Head() != "2xl" {
		t.Errorf("variant head: got %q, want %q", variants[0].Name.Head(), "2xl")
	}
}

func TestVariants(t *testing.T) {
	cl := parse(t, "sm:hover:bg-red-500")
	c := cl.Classes[0]
	variants := c.Variants()
	if len(variants) != 2 {
		t.Fatalf("expected 2 variants, got %d", len(variants))
	}
	if variants[0].FullName() != "sm" {
		t.Errorf("variant 0: got %q, want %q", variants[0].FullName(), "sm")
	}
	if variants[1].FullName() != "hover" {
		t.Errorf("variant 1: got %q, want %q", variants[1].FullName(), "hover")
	}
	if c.Utility().FullName() != "bg-red-500" {
		t.Errorf("utility: got %q, want %q", c.Utility().FullName(), "bg-red-500")
	}
}

func TestArbitraryValue(t *testing.T) {
	cl := parse(t, "w-[100px]")
	u := cl.Classes[0].Utility()
	if u.Name == nil {
		t.Fatal("expected non-nil Name")
	}
	if u.Name.Head() != "w" {
		t.Errorf("Head: got %q, want %q", u.Name.Head(), "w")
	}
	if len(u.Name.Parts()) != 1 || u.Name.Parts()[0].Arbitrary != "[100px]" {
		t.Errorf("expected 1 arbitrary part [100px], got %v", u.Name.Parts())
	}
}

func TestArbitraryProperty(t *testing.T) {
	cl := parse(t, "[display:flex]")
	u := cl.Classes[0].Utility()
	if u.Arbitrary != "[display:flex]" {
		t.Errorf("Arbitrary: got %q, want %q", u.Arbitrary, "[display:flex]")
	}
}

func TestArbitraryVariant(t *testing.T) {
	cl := parse(t, "[&:nth-child(3)]:underline")
	c := cl.Classes[0]
	variants := c.Variants()
	if len(variants) != 1 {
		t.Fatalf("expected 1 variant, got %d", len(variants))
	}
	if variants[0].Arbitrary != "[&:nth-child(3)]" {
		t.Errorf("variant: got %q, want %q", variants[0].Arbitrary, "[&:nth-child(3)]")
	}
	if c.Utility().FullName() != "underline" {
		t.Errorf("utility: got %q, want %q", c.Utility().FullName(), "underline")
	}
}

func TestImportant(t *testing.T) {
	cl := parse(t, "!text-red-500")
	u := cl.Classes[0].Utility()
	if !u.Important {
		t.Error("expected Important=true")
	}
	if u.FullName() != "text-red-500" {
		t.Errorf("FullName: got %q, want %q", u.FullName(), "text-red-500")
	}
}

func TestNegative(t *testing.T) {
	cl := parse(t, "-mt-4")
	u := cl.Classes[0].Utility()
	if !u.Negative {
		t.Error("expected Negative=true")
	}
	if u.FullName() != "mt-4" {
		t.Errorf("FullName: got %q, want %q", u.FullName(), "mt-4")
	}
	if u.Name.Parts()[0].Int == nil || *u.Name.Parts()[0].Int != 4 {
		t.Errorf("expected Int 4, got %v", u.Name.Parts()[0])
	}
}

func TestImportantNegative(t *testing.T) {
	cl := parse(t, "!-mt-4")
	u := cl.Classes[0].Utility()
	if !u.Important {
		t.Error("expected Important=true")
	}
	if !u.Negative {
		t.Error("expected Negative=true")
	}
	if u.FullName() != "mt-4" {
		t.Errorf("FullName: got %q, want %q", u.FullName(), "mt-4")
	}
}

func TestModifierInt(t *testing.T) {
	cl := parse(t, "bg-red-500/50")
	u := cl.Classes[0].Utility()
	if u.FullName() != "bg-red-500" {
		t.Errorf("FullName: got %q, want %q", u.FullName(), "bg-red-500")
	}
	if u.Modifier == nil {
		t.Fatal("expected non-nil Modifier")
	}
	if u.Modifier.Int == nil || *u.Modifier.Int != 50 {
		t.Errorf("expected Modifier.Int=50, got %v", u.Modifier)
	}
}

func TestModifierFloat(t *testing.T) {
	cl := parse(t, "bg-red-500/0.5")
	u := cl.Classes[0].Utility()
	if u.Modifier == nil {
		t.Fatal("expected non-nil Modifier")
	}
	if u.Modifier.Float == nil || *u.Modifier.Float != 0.5 {
		t.Errorf("expected Modifier.Float=0.5, got %v", u.Modifier)
	}
}

func TestModifierArbitrary(t *testing.T) {
	cl := parse(t, "bg-red-500/[.5]")
	u := cl.Classes[0].Utility()
	if u.Modifier == nil {
		t.Fatal("expected non-nil Modifier")
	}
	if u.Modifier.Arbitrary != "[.5]" {
		t.Errorf("Modifier.Arbitrary: got %q, want %q", u.Modifier.Arbitrary, "[.5]")
	}
}

func TestGroupVariantWithLabel(t *testing.T) {
	cl := parse(t, "group/sidebar:text-white")
	c := cl.Classes[0]
	variants := c.Variants()
	if len(variants) != 1 {
		t.Fatalf("expected 1 variant, got %d", len(variants))
	}
	if variants[0].FullName() != "group" {
		t.Errorf("variant name: got %q, want %q", variants[0].FullName(), "group")
	}
	if variants[0].Modifier == nil || variants[0].Modifier.Ident != "sidebar" {
		t.Error("expected variant modifier ident 'sidebar'")
	}
}

func TestVariantWithArbitrary(t *testing.T) {
	cl := parse(t, "peer-[.is-dirty]:block")
	c := cl.Classes[0]
	variants := c.Variants()
	if len(variants) != 1 {
		t.Fatalf("expected 1 variant, got %d", len(variants))
	}
	v := variants[0]
	if v.Name == nil || v.Name.Head() != "peer" {
		t.Errorf("variant head: got %q, want %q", v.Name.Head(), "peer")
	}
	if len(v.Name.Parts()) != 1 || v.Name.Parts()[0].Arbitrary != "[.is-dirty]" {
		t.Error("expected variant name part [.is-dirty]")
	}
}

func TestStarVariants(t *testing.T) {
	cl := parse(t, "*:flex")
	requireClasses(t, cl, 1)

	v := cl.Classes[0].Variants()
	if len(v) != 1 {
		t.Fatalf("expected 1 variant, got %d", len(v))
	}
	if v[0].Name == nil || v[0].Name.Head() != "*" {
		t.Errorf("variant head: got %q, want %q", v[0].Name.Head(), "*")
	}

	cl = parse(t, "**:grid")
	v = cl.Classes[0].Variants()
	if len(v) != 1 {
		t.Fatalf("expected 1 variant, got %d", len(v))
	}
	if v[0].Name == nil || v[0].Name.Head() != "**" {
		t.Errorf("variant head: got %q, want %q", v[0].Name.Head(), "**")
	}
}

func TestMultipleClasses(t *testing.T) {
	cl := parse(t, "flex items-center justify-between p-4")
	requireClasses(t, cl, 4)
	want := []string{"flex", "items-center", "justify-between", "p-4"}
	for i, w := range want {
		if got := cl.Classes[i].Utility().FullName(); got != w {
			t.Errorf("class %d: got %q, want %q", i, got, w)
		}
	}
}

func TestInvalidInput(t *testing.T) {
	cases := []string{
		":",
		"/",
		"hover:",
	}
	for _, input := range cases {
		t.Run(input, func(t *testing.T) {
			_, err := p.ParseString("", input)
			if err == nil {
				t.Error("expected parse error")
			}
			if _, ok := err.(participle.Error); !ok {
				t.Errorf("expected participle.Error, got %T", err)
			}
		})
	}
}

func requireClasses(t *testing.T, cl *parser.ClassList, n int) {
	t.Helper()
	if len(cl.Classes) != n {
		t.Fatalf("expected %d class(es), got %d", n, len(cl.Classes))
	}
}

func TestNamePartCustomParts(t *testing.T) {
	tests := []struct {
		name      string
		custom    string
		wantKey   string
		wantValue string
	}{
		{
			name:      "custom value without key",
			custom:    "(--my-width)",
			wantKey:   "",
			wantValue: "--my-width",
		},
		{
			name:      "custom value with key",
			custom:    "(color:red)",
			wantKey:   "color",
			wantValue: "red",
		},
		{
			name:      "custom value with css function",
			custom:    "(color:var(--brand))",
			wantKey:   "color",
			wantValue: "var(--brand)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			np := &parser.NamePart{
				Custom: tt.custom,
			}

			gotKey, gotValue := np.CustomParts()

			if gotKey != tt.wantKey {
				t.Errorf("key = %q, want %q", gotKey, tt.wantKey)
			}

			if gotValue != tt.wantValue {
				t.Errorf("value = %q, want %q", gotValue, tt.wantValue)
			}
		})
	}
}
