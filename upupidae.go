package upupidae

import (
	"context"
	"fmt"
	"io"
	"maps"
	"slices"

	"github.com/alecthomas/participle/v2"

	"github.com/thinkofher/upupidae/internal/parser"
)

// Varer reads variables from variable storage.
type Varer interface {
	// Var returns variable value as string for given string key.
	Var(ctx context.Context, key string) (value string)
}

// VarerFunc implements [Varer] interface as function.
type VarerFunc func(ctx context.Context, key string) (value string)

// Var returns variable value as string for given string key.
func (f VarerFunc) Var(ctx context.Context, key string) (value string) {
	return f(ctx, key)
}

// Namer creates CSS selectors for queries.
type Namer interface {
	// Name creates CSS selector name for given class.
	Name(ctx context.Context, class string) (name string)
}

// NamerFunc implements [Namer] interface as function.
type NamerFunc func(ctx context.Context, class string) (value string)

// Name creates CSS selector name for given class.
func (f NamerFunc) Name(ctx context.Context, class string) (name string) {
	return f(ctx, class)
}

// KV is a single CSS key/value entry inside as CSS query.
type KV struct {
	Key, Value string
}

// Entry is a single CSS document entry. It can be eiether a CSS [Query] or a
// line (like @import ...;).
type Entry struct {
	Query *Query
	Line  string
}

func (e *Entry) render(w io.Writer) error {
	switch {
	case e.Query != nil:
		return e.Query.render(w, 0)
	case e.Line != "":
		fmt.Fprintf(w, "%s;", e.Line)
	}

	return nil
}

// Query is a complete CSS query with a selector as Name.
type Query struct {
	// Name is a CSS selector's name.
	Name string

	// Nested queries.
	Nested []Query

	// KVs are key/value pairs associated with a given selector.
	KVs []KV
}

func (q *Query) render(w io.Writer, shift int) error {
	p := cssPreffix(shift)

	fmt.Fprintf(w, "%s%s {\n", p, q.Name)

	for _, kv := range q.KVs {
		fmt.Fprintf(w, "%s  %s: %s;\n", p, kv.Key, kv.Value)
	}

	for _, e := range q.Nested {
		e.render(w, shift+1)
	}

	fmt.Fprintf(w, "%s}\n", p)

	return nil
}

// Doc is entire CSS documented composed of CSS entries.
type Doc struct {
	Entries []Entry
}

func (doc *Doc) render(w io.Writer) error {
	for i, e := range doc.Entries {
		e.render(w)

		// separate entries with new line (except for the last entry)
		if i != len(doc.Entries)-1 {
			w.Write([]byte("\n"))
		}
	}

	return nil
}

// Reader returns rendered CSS document as a [io.ReadCloser].
func (doc *Doc) Reader() io.ReadCloser {
	pr, pw := io.Pipe()

	go func() {
		err := doc.render(pw)
		pw.CloseWithError(err)
	}()

	return pr
}

// OnFinishedDocWrapWithLayer wrapes existing doc with layer with given name.
// You should use this before you add additional entries to the doc.
func OnFinishedDocWrapWithLayer(name string) func(props []string, d *Doc) *Doc {
	return func(props []string, d *Doc) *Doc {
		lines, queries := entriesToLinesAndQueries(d.Entries)

		q := &Query{
			Name:   fmt.Sprintf("@layer %s", name),
			Nested: queries,
		}

		es := []Entry{}
		for _, l := range lines {
			es = append(es, Entry{Line: l})
		}

		es = append(es, Entry{Query: q})
		d.Entries = es

		return d
	}
}

// OnFinishedDocAddReset adds default CSS reset at the beginning of CSS
// document.
func OnFinishedDocAddReset(_ []string, d *Doc) *Doc {
	e := append([]Entry{}, twCSSReset...)
	d.Entries = append(e, d.Entries...)
	return d
}

// OnFinishedDocAddResetWithLayer adds default CSS reset at the beginning of
// CSS document wrapped in a layer with given name.
func OnFinishedDocAddResetWithLayer(name string) func(_ []string, d *Doc) *Doc {
	return func(s []string, d *Doc) *Doc {
		_, queries := entriesToLinesAndQueries(twCSSReset)

		q := &Query{
			Name:   fmt.Sprintf("@layer %s", name),
			Nested: queries,
		}
		d.Entries = append(d.Entries, Entry{Query: q})

		return d
	}
}

// OnFinishedDocAddVars adds variables, associated with given selector, at the
// beginning of CSS document.
func OnFinishedDocAddVars(sel string, vars map[string]string) func(props []string, d *Doc) *Doc {
	return func(props []string, d *Doc) *Doc {
		q := &Query{
			Name: sel,
		}

		for _, k := range slices.Sorted(slices.Values(props)) {
			v, ok := vars[k]
			if !ok {
				continue
			}

			q.KVs = append(q.KVs, KV{Key: k, Value: v})
		}

		if len(q.KVs) == 0 {
			return d
		}

		d.Entries = append([]Entry{{Query: q}}, d.Entries...)

		return d
	}
}

// OnFinishedDocAddVars adds variables, associated with given selector, at the
// beginning of CSS document, with given layer.
func OnFinishedDocAddVarsWithLayer(
	layer, sel string, vars map[string]string,
) func(props []string, d *Doc) *Doc {
	return func(props []string, d *Doc) *Doc {
		q := Query{
			Name: sel,
		}

		for _, k := range slices.Sorted(slices.Values(props)) {
			v, ok := vars[k]
			if !ok {
				continue
			}

			q.KVs = append(q.KVs, KV{Key: k, Value: v})
		}

		if len(q.KVs) == 0 {
			return d
		}

		layered := &Query{
			Name:   fmt.Sprintf("@layer %s", layer),
			Nested: []Query{q},
		}

		d.Entries = append([]Entry{{Query: layered}}, d.Entries...)

		return d
	}
}

// OnFinishedDocAddProperties adds @property entries at the end of CSS document
// together with theirs default values.
func OnFinishedDocAddProperties(props []string, d *Doc) *Doc {
	for _, p := range props {
		v, ok := twProperties[p]
		if !ok {
			continue
		}

		d.Entries = append(d.Entries, v.entry(p))
	}

	if e, ok := layerProperties(props); ok {
		d.Entries = append(d.Entries, e)
	}

	return d
}

// OnFinishedDocAddProperties adds @property entries at the end of CSS document
// together with theirs default values.
func OnFinishedDocAddPropertiesWithLayer(layer string) func(props []string, d *Doc) *Doc {
	return func(props []string, d *Doc) *Doc {
		for _, p := range props {
			v, ok := twProperties[p]
			if !ok {
				continue
			}

			d.Entries = append(d.Entries, v.entry(p))
		}

		if e, ok := layerProperties(props); ok {
			d.Entries = append(d.Entries, Entry{
				Query: &Query{
					Name:   fmt.Sprintf("@layer %s", layer),
					KVs:    e.Query.KVs,
					Nested: e.Query.Nested,
				},
			})
		}

		return d
	}
}

// OnFinishedDocAddAnimations adds @keyframe entries to the CSS document.
func OnFinishedDocAddAnimations(anims map[string]Query) func(props []string, d *Doc) *Doc {
	return func(props []string, d *Doc) *Doc {
		for _, p := range props {
			v, ok := anims[p]
			if !ok {
				continue
			}

			d.Entries = append(d.Entries, Entry{Query: &v})
		}

		return d
	}
}

// OnFinishedDocAddDefaultAnimations adds default @keyframe entries to the CSS
// document.
func OnFinishedDocAddDefaultAnimations(props []string, d *Doc) *Doc {
	return OnFinishedDocAddAnimations(twAnimations)(props, d)
}

// OnFinishedDocComposite combines multiple options that modify finished CSS
// document.
func OnFinishedDocComposite(fs ...func([]string, *Doc) *Doc) func([]string, *Doc) *Doc {
	return func(s []string, d *Doc) *Doc {
		for _, f := range fs {
			d = f(s, d)
		}

		return d
	}
}

// OnFinishedDocUnnestEntries unnests entries that can be a single selector
// instead of group of nested CSS selectors.
func OnFinishedDocUnnestEntries(_ []string, d *Doc) *Doc {
	d.Entries = optUnnestEntries(d.Entries)
	return d
}

// OnFinishedDocCombine combines together key/value pairs and children queries
// grouped under the same parent query.
func OnFinishedDocCombine(_ []string, d *Doc) *Doc {
	return optCombine(d)
}

// Gen generates CSS [Doc] from utility classes.
type Gen struct {
	parser        *participle.Parser[parser.ClassList]
	gen           *gen
	onFinishedDoc func(vars []string, d *Doc) *Doc
	onParseError  func(error)
}

// NewGen is the only way to define and initialize [Gen].
func NewGen(v Varer, n Namer) *Gen {
	return &Gen{
		parser: parser.Must(),
		gen: &gen{
			namer: n,
			varer: v,
			registerer: &registerer{
				registry: map[string]struct{}{},
			},
		},
		onFinishedDoc: func(_ []string, d *Doc) *Doc { return d },
		onParseError:  func(error) {},
	}
}

// NewDefaultGen returns [Gen] with default options set.
func NewDefaultGen() *Gen {
	g := NewGen(VarerFromVars(map[string]string{}), NamerEscape())
	g.OnFinishedDoc(OnFinishedDocComposite(
		OnFinishedDocUnnestEntries,
		OnFinishedDocCombine,
		OnFinishedDocAddReset,
		OnFinishedDocAddVars(":root, :host", twVars),
		OnFinishedDocAddProperties,
		OnFinishedDocAddDefaultAnimations,
	))
	return g
}

// OnFinishedDoc executes given function after [Doc] is generated.
func (g *Gen) OnFinishedDoc(f func([]string, *Doc) *Doc) {
	g.onFinishedDoc = f
}

// OnParseError executes given function when parser failed to parse given
// utility class.
func (g *Gen) OnParseError(f func(error)) {
	g.onParseError = f
}

// Doc generates CSS [Doc] based on given set of utility classes.
func (g *Gen) Doc(ctx context.Context, classes ...string) *Doc {
	doc := &Doc{}

	for _, c := range classes {
		ast, err := g.parser.ParseString("", c)
		if err != nil {
			g.onParseError(err)
		}

		for _, c := range ast.Classes {
			cls, err := g.gen.generate(ctx, c)
			if err != nil {
				g.onParseError(err)
				continue
			}

			doc.Entries = append(doc.Entries, cls.entry())
		}
	}

	vars := []string{}
	for v := range g.gen.registerer.registry {
		vars = append(vars, v)
	}

	return g.onFinishedDoc(vars, doc)
}

// VarerFromVars returns [Varer] that lookups variables from given map.
func VarerFromVars(vars map[string]string) Varer {
	return VarerFunc(func(ctx context.Context, key string) (value string) {
		v, ok := vars[key]
		if ok {
			return v
		}

		return fmt.Sprintf("var(%s)", key)
	})
}

// VarerDefault returns [Varer] that lookups variables from default
// configuration.
func VarerDefault() Varer {
	return VarerFromVars(twVars)
}

// NamerEscape creates a safe and escaped CSS class selector for given utility class.
func NamerEscape() Namer {
	return NamerFunc(func(_ context.Context, c string) (value string) {
		return cssSelectorClass(c)
	})
}

// DefaultVars returns copy of default configuration variables.
func DefaultVars() (m map[string]string) {
	m = map[string]string{}
	maps.Copy(m, twVars)
	return m
}

// DefaultReset returns default CSS reset entries.
func DefaultReset() []Entry {
	return slices.Clone(twCSSReset)
}

func entriesToLinesAndQueries(es []Entry) (lines []string, queries []Query) {
	for _, e := range es {
		if e.Line != "" {
			lines = append(lines, e.Line)
		}

		if e.Query != nil {
			queries = append(queries, *e.Query)
		}
	}

	return
}
