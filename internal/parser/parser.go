package parser

import (
	"strconv"
	"strings"

	"github.com/alecthomas/participle/v2"
	"github.com/alecthomas/participle/v2/lexer"
)

var tailwindLexer = lexer.MustSimple([]lexer.SimpleRule{
	{"Whitespace", `\s+`},
	{"Custom", `\((?:[^\[\]]|\[[^\[\]]*\])*\)`},
	{"Arbitrary", `\[(?:[^\[\]]|\[[^\[\]]*\])*\]`},
	{"Percentage", `\d+(?:\.\d+)?%`},
	{"AlphaNum", `\d+[a-zA-Z_]\w*`},
	{"Float", `\d+\.\d+`},
	{"Int", `\d+`},
	{"Ident", `[a-zA-Z_]\w*`},
	{"AtVariant", `@(?:max-|min-)?(?:\[[^\]]*\]|\d+[a-zA-Z_][\w-]*|[a-zA-Z_][\w-]*)`},
	{"StarStar", `\*\*`},
	{"Star", `\*`},
	{"Colon", `:`},
	{"Slash", `/`},
	{"Bang", `!`},
	{"Dash", `-`},
})

// ClassList is a whitespace-separated list of Tailwind CSS classes.
type ClassList struct {
	Classes []*Class `@@*`
}

func (cl *ClassList) String() string {
	parts := make([]string, len(cl.Classes))
	for i, c := range cl.Classes {
		parts[i] = c.String()
	}
	return strings.Join(parts, " ")
}

// Class is a single Tailwind CSS class parsed as colon-separated segments.
// All segments except the last are variants; the last segment is the utility.
type Class struct {
	Segments []*Segment `@@ ( ":" @@ )*`
}

func (c *Class) String() string {
	parts := make([]string, len(c.Segments))
	for i, s := range c.Segments {
		parts[i] = s.String()
	}
	return strings.Join(parts, ":")
}

// Variants returns all segments except the last (the variant prefixes).
func (c *Class) Variants() []*Segment {
	if len(c.Segments) <= 1 {
		return nil
	}
	return c.Segments[:len(c.Segments)-1]
}

// Utility returns the last segment (the utility class itself).
func (c *Class) Utility() *Segment {
	return c.Segments[len(c.Segments)-1]
}

// Segment is one colon-separated part of a Tailwind class. It can represent
// a variant (e.g. "hover", "sm", "group/sidebar", "[&:nth-child(3)]") or
// a utility (e.g. "bg-red-500", "w-[100px]", "[display:flex]").
//
// When used as a variant, the Modifier field acts as a group label
// (e.g. group/sidebar → Modifier.Ident = "sidebar").
// When used as a utility, the Modifier field acts as an opacity modifier
// (e.g. bg-red-500/50 → Modifier.Int = 50).
type Segment struct {
	Important bool        `@"!"?`
	Negative  bool        `@"-"?`
	Arbitrary string      `( @Arbitrary`
	Name      *Name       `| @@ )`
	Modifier  *SlashValue `( "/" @@ )?`
}

func (s *Segment) String() string {
	var b strings.Builder
	if s.Important {
		b.WriteByte('!')
	}
	if s.Negative {
		b.WriteByte('-')
	}
	if s.Arbitrary != "" {
		b.WriteString(s.Arbitrary)
	} else if s.Name != nil {
		b.WriteString(s.Name.String())
	}
	if s.Modifier != nil {
		b.WriteByte('/')
		b.WriteString(s.Modifier.String())
	}
	return b.String()
}

// FullName returns the segment's name including any arbitrary suffix,
// but excluding Important, Negative, and Modifier.
func (s *Segment) FullName() string {
	if s.Arbitrary != "" {
		return s.Arbitrary
	}
	if s.Name != nil {
		return s.Name.String()
	}
	return ""
}

// Name is the hyphen-separated name of a utility or variant, such as
// "bg-red-500", "min-w-[100px]", "group-hover", or "@md".
type Name struct {
	At      *AtName      `@@`
	Regular *RegularName `| @@`
}

// AtName is a container-query variant or @container utility (@md, @max-lg, @container, …).
type AtName struct {
	Raw string `@AtVariant`
}

func atNameParts(raw string) []*NamePart {
	body := strings.TrimPrefix(raw, "@")
	if body == "" {
		return nil
	}

	if strings.HasPrefix(body, "[") {
		return []*NamePart{{Arbitrary: body}}
	}

	segments := strings.Split(body, "-")
	parts := make([]*NamePart, len(segments))
	for i, segment := range segments {
		if segment == "" {
			continue
		}
		if segment[0] >= '0' && segment[0] <= '9' {
			parts[i] = &NamePart{AlphaNum: segment}
		} else {
			parts[i] = &NamePart{Ident: segment}
		}
	}
	return parts
}

func formatAtName(raw string) string {
	parts := atNameParts(raw)
	if len(parts) == 0 {
		return raw
	}

	var b strings.Builder
	b.WriteString("@")
	b.WriteString(parts[0].String())
	for _, p := range parts[1:] {
		b.WriteByte('-')
		b.WriteString(p.String())
	}
	return b.String()
}

// RegularName is a standard hyphen-separated utility or variant name.
type RegularName struct {
	Head  string      `@(Ident | AlphaNum | Float | Int | StarStar | Star)`
	Parts []*NamePart `( "-" @@ )*`
}

// Head returns the leading identifier of the name.
func (n *Name) Head() string {
	if n.At != nil {
		return "@"
	}
	if n.Regular != nil {
		return n.Regular.Head
	}
	return ""
}

// Parts returns all hyphen-separated components after the head.
func (n *Name) Parts() []*NamePart {
	if n.At != nil {
		return atNameParts(n.At.Raw)
	}
	if n.Regular != nil {
		return n.Regular.Parts
	}
	return nil
}

// Regular constructs a non-@ Name for synthetic segments built during variant expansion.
func Regular(head string, parts []*NamePart) *Name {
	return &Name{Regular: &RegularName{Head: head, Parts: parts}}
}

func (n *Name) String() string {
	if n.At != nil {
		return formatAtName(n.At.Raw)
	}

	if n.Regular == nil {
		return ""
	}

	var b strings.Builder
	b.WriteString(n.Regular.Head)
	for _, p := range n.Regular.Parts {
		b.WriteByte('-')
		b.WriteString(p.String())
	}
	return b.String()
}

// NamePart is a single hyphen-separated component within a Name.
// Exactly one field is populated depending on the token type:
//   - Arbitrary for bracket expressions like "[100px]"
//   - Float for decimal numbers like 0.5
//   - Int for integers like 500
//   - AlphaNum for digit-leading mixed tokens like "2xl"
//   - Ident for identifiers like "red"
type NamePart struct {
	Arbitrary  string   `  @Arbitrary`
	Custom     string   `| @Custom`
	Percentage string   `| @Percentage`
	Float      *float64 `| @Float`
	Int        *int     `| @Int`
	AlphaNum   string   `| @AlphaNum`
	Ident      string   `| @Ident`
}

func (np *NamePart) String() string {
	switch {
	case np.Arbitrary != "":
		return np.Arbitrary
	case np.Custom != "":
		return np.Custom
	case np.Percentage != "":
		return np.Percentage
	case np.Float != nil:
		return strconv.FormatFloat(*np.Float, 'f', -1, 64)
	case np.Int != nil:
		return strconv.Itoa(*np.Int)
	case np.AlphaNum != "":
		return np.AlphaNum
	default:
		return np.Ident
	}
}

func typeHintCut(s, prefix, suffix string) (key, value string) {
	raw := strings.TrimSuffix(strings.TrimPrefix(s, prefix), suffix)

	key, value, found := strings.Cut(raw, ":")
	if !found {
		return "", raw
	}

	return key, value
}

// CustomParts splits a custom utility value into an optional key and value.
//
// Tailwind custom values may optionally contain a type hint:
//
//	(color:red)          -> key="color", value="red"
//	(--my-variable)      -> key="", value="--my-variable"
//
// The key is empty when no type hint is provided.
func (np *NamePart) CustomParts() (key, value string) {
	return typeHintCut(np.Custom, "(", ")")
}

// ArbitraryParts splits an arbitrary utility value into an optional key and value.
//
// Tailwind arbitrary values may optionally contain a type hint:
//
//	[color:red]          -> key="color", value="red"
//	[--my-variable]      -> key="", value="--my-variable"
//
// The key is empty when no type hint is provided.
func (np *NamePart) ArbitraryParts() (key, value string) {
	return typeHintCut(np.Arbitrary, "[", "]")
}

// SlashValue is the value after a "/" in a class. Depending on context it is
// either a variant group label or an opacity modifier.
// Exactly one field is populated depending on the token type.
type SlashValue struct {
	Arbitrary string   `  @Arbitrary`
	Custom    string   `| @Custom`
	Float     *float64 `| @Float`
	Int       *int     `| @Int`
	AlphaNum  string   `| @AlphaNum`
	Ident     string   `| @Ident`
}

// CustomParts splits a custom utility value into an optional key and value.
//
// Tailwind custom values may optionally contain a type hint:
//
//	(color:red)          -> key="color", value="red"
//	(--my-variable)      -> key="", value="--my-variable"
//
// The key is empty when no type hint is provided.
func (sv *SlashValue) CustomParts() (key, value string) {
	return typeHintCut(sv.Custom, "(", ")")
}

// ArbitraryParts splits an arbitrary utility value into an optional key and value.
//
// Tailwind arbitrary values may optionally contain a type hint:
//
//	[color:red]          -> key="color", value="red"
//	[--my-variable]      -> key="", value="--my-variable"
//
// The key is empty when no type hint is provided.
func (sv *SlashValue) ArbitraryParts() (key, value string) {
	return typeHintCut(sv.Arbitrary, "[", "]")
}

func (sv *SlashValue) String() string {
	switch {
	case sv.Arbitrary != "":
		return sv.Arbitrary
	case sv.Custom != "":
		return sv.Custom
	case sv.Float != nil:
		return strconv.FormatFloat(*sv.Float, 'f', -1, 64)
	case sv.Int != nil:
		return strconv.Itoa(*sv.Int)
	case sv.AlphaNum != "":
		return sv.AlphaNum
	default:
		return sv.Ident
	}
}

// New builds a participle parser for Tailwind CSS class syntax.
func New() (*participle.Parser[ClassList], error) {
	return participle.Build[ClassList](
		participle.Lexer(tailwindLexer),
		participle.Elide("Whitespace"),
	)
}

// Must is like New but panics on error.
func Must() *participle.Parser[ClassList] {
	p, err := New()
	if err != nil {
		panic(err)
	}
	return p
}
