package upupidae

import (
	"slices"
	"strings"
)

// optCombine merges cssDoc entries whose outermost nodes are the same at-rule.
//
// Run it after optUnnest. Generation emits one entry per Tailwind class; combine
// folds entries that start with an identical @media, @supports, @container, or
// other @ rule into a single tree by splicing their nested children together
// and recursing.
//
// A node is treated as an at-rule wrapper when its name begins with "@", it has
// no declarations, and it has nested children. Selector rules (.class with kvs),
// utilities that embed their own at-rules (such as container), and line entries
// are not merged with each other; line entries are copied through unchanged.
//
// Sibling order is stable: non-wrapper nodes keep their original order, then
// merged at-rule groups appear in the order each wrapper name was first seen.
//
// For example, two entries both rooted at @media (hover: hover) collapse into
// one block whose children are the union of their nested rules. Entries rooted
// at different wrappers, or with no wrapper at all, remain separate.
//
// optCombine does not modify d. It returns nil if d is nil.
func optCombine(d *Doc) *Doc {
	if d == nil {
		return nil
	}

	out := &Doc{}
	queries := make([]*Query, 0, len(d.Entries))

	for _, e := range d.Entries {
		if e.Line != "" {
			out.Entries = append(out.Entries, e)
			continue
		}
		if e.Query != nil {
			queries = append(queries, e.Query)
		}
	}

	for _, q := range optCombineQueries(queries) {
		out.Entries = append(out.Entries, Entry{Query: q})
	}

	return out
}

func optCombineQueries(queries []*Query) []*Query {
	if len(queries) == 0 {
		return nil
	}

	combined := make([]*Query, len(queries))
	for i, q := range queries {
		combined[i] = optCombineQuery(q)
	}

	order := make([]string, 0, len(combined))
	byName := make(map[string]*Query, len(combined))

	for _, q := range combined {
		if q == nil {
			continue
		}

		existing, ok := byName[q.Name]
		if !ok {
			merged := *q
			merged.KVs = append([]KV(nil), q.KVs...)
			merged.Nested = append([]Query(nil), q.Nested...)

			byName[q.Name] = &merged
			order = append(order, q.Name)
			continue
		}

		// Merge KVs by key. Later values replace earlier values.
		existing.KVs = optMergeKVs(existing.KVs, q.KVs)

		// Combine nested selectors.
		existing.Nested = append(existing.Nested, q.Nested...)
	}

	out := make([]*Query, 0, len(order))

	for _, name := range order {
		q := byName[name]

		if len(q.Nested) > 0 {
			nested := make([]*Query, len(q.Nested))
			for i := range q.Nested {
				nested[i] = &q.Nested[i]
			}

			q.Nested = optCombineQueriesToSlice(nested)
		}

		out = append(out, q)
	}

	return out
}

func optMergeKVs(a, b []KV) []KV {
	m := make(map[string]KV, len(a)+len(b))

	for _, kv := range a {
		m[kv.Key] = kv
	}

	for _, kv := range b {
		// Later values overwrite earlier values.
		m[kv.Key] = kv
	}

	out := make([]KV, 0, len(m))
	for _, kv := range m {
		out = append(out, kv)
	}

	slices.SortFunc(out, func(a, b KV) int {
		return strings.Compare(a.Key, b.Key)
	})

	return out
}

func optCombineQueriesToSlice(queries []*Query) []Query {
	combined := optCombineQueries(queries)
	out := make([]Query, len(combined))
	for i, q := range combined {
		out[i] = *q
	}
	return out
}

func optCombineQuery(q *Query) *Query {
	if q == nil {
		return nil
	}

	out := *q
	if len(q.Nested) > 0 {
		nested := make([]*Query, len(q.Nested))
		for i := range q.Nested {
			nested[i] = &q.Nested[i]
		}
		out.Nested = optCombineQueriesToSlice(nested)
	}

	return &out
}

func optIsAtWrapper(q *Query) bool {
	return strings.HasPrefix(q.Name, "@") && len(q.KVs) == 0 && len(q.Nested) > 0
}

type optSelLayer struct {
	kind  string // "prefix", "suffix", "embed", "multi"
	value string
}

// optUnnest rewrites a variant cssQuery tree so at-rules wrap the final selector.
//
// Generation nests variants inside the class selector: the escaped utility name
// is the outermost node, @ rules and & selectors sit in between, and declarations
// live at the leaf. optUnnest peels that chain and rebuilds it with @ rules on
// the outside and a single compound selector on the inside.
//
// While peeling a single-child wrapper chain, nodes are classified by name:
//   - "." — the utility class selector; at most one per chain
//   - "@" — an at-rule kept as a wrapper in the output
//   - "&…" — a variant suffix appended to the class selector after removing "&"
//   - "… &" — a variant prefix (in-*) prepended before the class selector
//   - other names containing "&" — embedded selectors (:is(& > *)) with "&" replaced by the class
//
// Peeling stops at a node that carries declarations, has multiple nested
// children, or is not a variant wrapper. Plain utilities and utilities with
// embedded at-rules (such as container) are only recursed into, not re-rooted.
//
// Selector flattening follows Tailwind's compound in-* stacking: consecutive
// in-* prefixes nest with :is(), suffixes append inside the :is group, and an
// in-* prefix after suffixes wraps with :where(X) :is(…).
//
// optUnnest does not modify q. It returns nil if q is nil.
func optUnnest(q *Query) *Query {
	if q == nil {
		return nil
	}

	classSel, layers, atRules, kvs, nested := optPeelQuery(q)
	if classSel != "" {
		selectors := optBuildSelectors(classSel, layers)
		rules := make([]Query, len(selectors))
		for i, sel := range selectors {
			rules[i] = Query{
				Name:   sel,
				KVs:    kvs,
				Nested: optUnnestAll(nested),
			}
		}

		var result *Query
		switch {
		case len(rules) == 1:
			result = &rules[0]
		case len(atRules) > 0:
			result = &Query{Name: atRules[len(atRules)-1], Nested: rules}
			for i := len(atRules) - 2; i >= 0; i-- {
				result = &Query{Name: atRules[i], Nested: []Query{*result}}
			}
			return result
		default:
			result = &Query{Nested: rules}
			return result
		}

		for i := len(atRules) - 1; i >= 0; i-- {
			result = &Query{
				Name:   atRules[i],
				Nested: []Query{*result},
			}
		}

		return result
	}

	out := *q
	out.Nested = optUnnestAll(q.Nested)
	return &out
}

func optUnnestAll(qs []Query) []Query {
	if len(qs) == 0 {
		return nil
	}

	out := make([]Query, len(qs))
	for i := range qs {
		if unnested := optUnnest(&qs[i]); unnested != nil {
			out[i] = *unnested
		}
	}
	return out
}

func optBuildSelectors(classSel string, layers []optSelLayer) []string {
	for _, layer := range layers {
		if layer.kind == "embed" {
			return []string{strings.ReplaceAll(layer.value, "&", classSel)}
		}
	}

	prefixesBefore, suffixesBefore, multi, suffixesAfter, prefixesAfter, multiOuter, multiLast := optPartitionLayers(layers)

	if multi == "" {
		suffixes := append(suffixesBefore, suffixesAfter...)
		return []string{optBuildFinalSelector(classSel, prefixesBefore, suffixes, prefixesAfter, "", false, false)}
	}

	parts := strings.Split(multi, ",")
	selectors := make([]string, 0, len(parts))
	for _, part := range parts {
		selectors = append(selectors, optBuildFinalSelector(
			classSel,
			prefixesBefore,
			suffixesBefore,
			prefixesAfter,
			strings.TrimSpace(part),
			multiOuter,
			multiLast,
			suffixesAfter...,
		))
	}

	return selectors
}

func optPartitionLayers(layers []optSelLayer) (
	prefixesBefore []optSelLayer,
	suffixesBefore []optSelLayer,
	multi string,
	suffixesAfter []optSelLayer,
	prefixesAfter []optSelLayer,
	multiOuter bool,
	multiLast bool,
) {
	if len(layers) > 0 && layers[0].kind == "multi" {
		prefixesBefore, suffixesBefore, multi, suffixesAfter, prefixesAfter, multiOuter = optPartitionLayersMultiFirst(layers)
		return prefixesBefore, suffixesBefore, multi, suffixesAfter, prefixesAfter, multiOuter, false
	}

	seenVariant := false
	afterMulti := false

	for _, layer := range layers {
		switch layer.kind {
		case "prefix":
			if seenVariant {
				prefixesAfter = append(prefixesAfter, layer)
			} else {
				prefixesBefore = append(prefixesBefore, layer)
			}
		case "suffix":
			seenVariant = true
			if afterMulti {
				suffixesAfter = append(suffixesAfter, layer)
			} else {
				suffixesBefore = append(suffixesBefore, layer)
			}
		case "multi":
			seenVariant = true
			multi = layer.value
			afterMulti = true
		}
	}

	multiLast = multi != "" && layers[len(layers)-1].kind == "multi"

	return prefixesBefore, suffixesBefore, multi, suffixesAfter, prefixesAfter, false, multiLast
}

func optPartitionLayersMultiFirst(layers []optSelLayer) (
	prefixesBefore []optSelLayer,
	suffixesBefore []optSelLayer,
	multi string,
	suffixesAfter []optSelLayer,
	prefixesAfter []optSelLayer,
	multiOuter bool,
) {
	multi = layers[0].value
	i := 1

	for i < len(layers) && layers[i].kind == "prefix" {
		prefixesBefore = append(prefixesBefore, layers[i])
		i++
	}

	for i < len(layers) && layers[i].kind == "suffix" {
		suffixesBefore = append(suffixesBefore, layers[i])
		i++
	}

	for i < len(layers) && layers[i].kind == "prefix" {
		prefixesAfter = append(prefixesAfter, layers[i])
		i++
	}

	return prefixesBefore, suffixesBefore, multi, suffixesAfter, prefixesAfter, true
}

func optBuildPrefixStack(prefixes []optSelLayer, classSel string) string {
	sel := classSel
	for i, layer := range prefixes {
		if i == 0 {
			sel = layer.value + " " + sel
			continue
		}
		sel = layer.value + " :is(" + sel + ")"
	}
	return sel
}

func optJoinSuffixes(suffixes []optSelLayer) string {
	var b strings.Builder
	for _, layer := range suffixes {
		b.WriteString(layer.value)
	}
	return b.String()
}

func optAppendSuffixes(sel string, suffixes []optSelLayer) string {
	if len(suffixes) == 0 {
		return sel
	}
	return sel + optJoinSuffixes(suffixes)
}

func optBuildFinalSelector(
	classSel string,
	prefixesBefore []optSelLayer,
	suffixesBefore []optSelLayer,
	prefixesAfter []optSelLayer,
	partTemplate string,
	multiOuter bool,
	multiLast bool,
	suffixesAfter ...optSelLayer,
) string {
	if partTemplate != "" && multiOuter && optHasInnerVariants(prefixesBefore, suffixesBefore, prefixesAfter, suffixesAfter) {
		allSuffixes := append(suffixesBefore, suffixesAfter...)
		if strings.Contains(partTemplate, " *::") {
			replaced := strings.ReplaceAll(partTemplate, "&", classSel)
			replaced = strings.ReplaceAll(replaced, " *::", " ::")
			stack := optBuildPrefixStack(prefixesBefore, ":is("+replaced+")")
			stack = optAppendSuffixes(stack, allSuffixes)
			return optApplyTrailingPrefixes(prefixesAfter, stack)
		}

		pseudo := strings.TrimPrefix(partTemplate, "&")
		stack := optBuildPrefixStack(prefixesBefore, classSel+pseudo)
		stack = optAppendSuffixes(stack, allSuffixes)
		return optApplyTrailingPrefixes(prefixesAfter, stack)
	}

	if partTemplate != "" && multiLast {
		allSuffixes := append(suffixesBefore, suffixesAfter...)
		stack := optBuildPrefixStack(prefixesBefore, classSel)
		stack = optAppendSuffixes(stack, allSuffixes)
		stack = optApplyTrailingPrefixes(prefixesAfter, stack)
		return stack + optMultiPartSuffix(partTemplate)
	}

	stack := optBuildPrefixStack(prefixesBefore, classSel)
	stack = optAppendSuffixes(stack, suffixesBefore)

	sel := stack
	if partTemplate != "" {
		sel = strings.ReplaceAll(partTemplate, "&", stack)
		sel = strings.ReplaceAll(sel, " *::", " ::")
		sel += optJoinSuffixes(suffixesAfter)
	} else {
		stack = optAppendSuffixes(stack, suffixesAfter)
		sel = stack
	}

	return optApplyTrailingPrefixes(prefixesAfter, sel)
}

func optMultiPartSuffix(partTemplate string) string {
	if strings.Contains(partTemplate, " *::") {
		return strings.ReplaceAll(strings.ReplaceAll(partTemplate, "&", ""), " *::", " ::")
	}

	return strings.TrimPrefix(partTemplate, "&")
}

func optHasInnerVariants(
	prefixesBefore []optSelLayer,
	suffixesBefore []optSelLayer,
	prefixesAfter []optSelLayer,
	suffixesAfter []optSelLayer,
) bool {
	return len(prefixesBefore) > 0 ||
		len(suffixesBefore) > 0 ||
		len(prefixesAfter) > 0 ||
		len(suffixesAfter) > 0
}

func optApplyTrailingPrefixes(prefixes []optSelLayer, sel string) string {
	for _, layer := range prefixes {
		sel = optApplyInPrefix(layer.value, sel)
	}
	return sel
}

func optApplyInPrefix(prefix, sel string) string {
	if strings.Contains(sel, " ::") || strings.Contains(sel, ":is(") {
		return prefix + " :is(" + sel + ")"
	}

	return prefix + " " + sel
}

func optPeelQuery(q *Query) (
	classSel string,
	layers []optSelLayer,
	atRules []string,
	kvs []KV,
	nested []Query,
) {
	switch {
	case strings.HasPrefix(q.Name, "."):
		classSel = q.Name
	case strings.HasPrefix(q.Name, "@"):
		atRules = append(atRules, q.Name)
	case strings.HasPrefix(q.Name, "&"):
		if strings.Contains(q.Name, ",") {
			layers = append(layers, optSelLayer{kind: "multi", value: q.Name})
		} else {
			layers = append(layers, optSelLayer{kind: "suffix", value: strings.TrimPrefix(q.Name, "&")})
		}
	case strings.HasSuffix(q.Name, " &"):
		layers = append(layers, optSelLayer{kind: "prefix", value: strings.TrimSuffix(q.Name, " &")})
	case strings.Contains(q.Name, "&"):
		layers = append(layers, optSelLayer{kind: "embed", value: q.Name})
	default:
		classSel = q.Name
	}

	if len(q.Nested) == 1 && len(q.KVs) == 0 {
		c, ls, at, kv, n := optPeelQuery(&q.Nested[0])
		if classSel == "" {
			classSel = c
		}
		layers = append(layers, ls...)
		atRules = append(atRules, at...)
		return classSel, layers, atRules, kv, n
	}

	return classSel, layers, atRules, q.KVs, q.Nested
}

func optUnnestExpand(q *Query) []*Query {
	q = optUnnest(q)
	if q == nil {
		return nil
	}
	if q.Name == "" && len(q.KVs) == 0 && len(q.Nested) > 0 {
		out := make([]*Query, len(q.Nested))
		for i := range q.Nested {
			out[i] = &q.Nested[i]
		}
		return out
	}

	return []*Query{q}
}

func optUnnestEntries(entries []Entry) []Entry {
	out := make([]Entry, 0, len(entries))
	for _, e := range entries {
		if e.Line != "" || e.Query == nil {
			out = append(out, e)
			continue
		}
		for _, q := range optUnnestExpand(e.Query) {
			out = append(out, Entry{Query: q})
		}
	}
	return out
}
