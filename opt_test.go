package upupidae

import (
	"bytes"
	"strings"
	"testing"
)

func TestOptUnnest(t *testing.T) {
	class := `.supports-[display\:grid]\:@md\:hover\:group-[\:nth-of-type(3)_&_&_&]\:block`

	input := &Query{
		Name: class,
		Nested: []Query{{
			Name: `@supports (display:grid)`,
			Nested: []Query{{
				Name: `@container (width >= 28rem)`,
				Nested: []Query{{
					Name: `&:hover`,
					Nested: []Query{{
						Name: `@media (hover: hover)`,
						Nested: []Query{{
							Name: `&:is(:nth-of-type(3) :where(.group) :where(.group) :where(.group) *)`,
							KVs:  newSimpleCssKV("display", "block"),
						}},
					}},
				}},
			}},
		}},
	}

	want := strings.TrimSpace(`
@supports (display:grid) {
  @container (width >= 28rem) {
    @media (hover: hover) {
      .supports-[display\:grid]\:@md\:hover\:group-[\:nth-of-type(3)_&_&_&]\:block:hover:is(:nth-of-type(3) :where(.group) :where(.group) :where(.group) *) {
        display: block;
      }
    }
  }
}
`)

	got := renderQuery(t, optUnnest(input))
	if got != want {
		t.Fatalf("got:\n%s\n\nwant:\n%s", got, want)
	}
}

func TestOptUnnestSimpleUtility(t *testing.T) {
	input := &Query{
		Name: `.flex`,
		KVs:  newSimpleCssKV("display", "flex"),
	}

	want := strings.TrimSpace(`
.flex {
  display: flex;
}
`)

	got := renderQuery(t, optUnnest(input))
	if got != want {
		t.Fatalf("got:\n%s\n\nwant:\n%s", got, want)
	}
}

func TestOptUnnestKeepsUtilityNestedAtRules(t *testing.T) {
	input := &Query{
		Name: `.container`,
		KVs:  newSimpleCssKV("width", "100%"),
		Nested: []Query{{
			Name: `@media (width >= 40rem)`,
			KVs:  newSimpleCssKV("max-width", "40rem"),
		}},
	}

	want := strings.TrimSpace(`
.container {
  width: 100%;
  @media (width >= 40rem) {
    max-width: 40rem;
  }
}
`)

	got := renderQuery(t, optUnnest(input))
	if got != want {
		t.Fatalf("got:\n%s\n\nwant:\n%s", got, want)
	}
}

func TestOptUnnestInVariant(t *testing.T) {
	input := &Query{
		Name: `.in-focus\:opacity-100`,
		Nested: []Query{{
			Name: `:where(:focus) &`,
			KVs:  newSimpleCssKV("opacity", "100%"),
		}},
	}

	want := strings.TrimSpace(`
:where(:focus) .in-focus\:opacity-100 {
  opacity: 100%;
}
`)

	got := renderQuery(t, optUnnest(input))
	if got != want {
		t.Fatalf("got:\n%s\n\nwant:\n%s", got, want)
	}
}

func TestOptUnnestInVariantWithHover(t *testing.T) {
	input := &Query{
		Name: `.in-hover\:opacity-100`,
		Nested: []Query{{
			Name: `:where(:hover) &`,
			Nested: []Query{{
				Name: `@media (hover: hover)`,
				KVs:  newSimpleCssKV("opacity", "100%"),
			}},
		}},
	}

	want := strings.TrimSpace(`
@media (hover: hover) {
  :where(:hover) .in-hover\:opacity-100 {
    opacity: 100%;
  }
}
`)

	got := renderQuery(t, optUnnest(input))
	if got != want {
		t.Fatalf("got:\n%s\n\nwant:\n%s", got, want)
	}
}

func TestOptUnnestChildSelector(t *testing.T) {
	input := &Query{
		Name: `.\*:flex`,
		Nested: []Query{{
			Name: `:is(& > *)`,
			KVs:  newSimpleCssKV("display", "flex"),
		}},
	}

	want := strings.TrimSpace(`
:is(.\*:flex > *) {
  display: flex;
}
`)

	got := renderQuery(t, optUnnest(input))
	if got != want {
		t.Fatalf("got:\n%s\n\nwant:\n%s", got, want)
	}
}

func TestOptUnnestInFocusHover(t *testing.T) {
	input := &Query{
		Name: `.in-focus\:hover\:opacity-100`,
		Nested: []Query{{
			Name: `:where(:focus) &`,
			Nested: []Query{{
				Name: `&:hover`,
				Nested: []Query{{
					Name: `@media (hover: hover)`,
					KVs:  newSimpleCssKV("opacity", "100%"),
				}},
			}},
		}},
	}

	want := strings.TrimSpace(`
@media (hover: hover) {
  :where(:focus) .in-focus\:hover\:opacity-100:hover {
    opacity: 100%;
  }
}
`)

	got := renderQuery(t, optUnnest(input))
	if got != want {
		t.Fatalf("got:\n%s\n\nwant:\n%s", got, want)
	}
}

func TestOptUnnestInVariantStack(t *testing.T) {
	class := `.in-even\:in-first\:hover\:has-only\:in-focus\:opacity-100`
	input := &Query{
		Name: class,
		Nested: []Query{{
			Name: `:where(:nth-child(even)) &`,
			Nested: []Query{{
				Name: `:where(:first-child) &`,
				Nested: []Query{{
					Name: `&:hover`,
					Nested: []Query{{
						Name: `@media (hover: hover)`,
						Nested: []Query{{
							Name: `&:has(:only-child)`,
							Nested: []Query{{
								Name: `:where(:focus) &`,
								KVs:  newSimpleCssKV("opacity", "100%"),
							}},
						}},
					}},
				}},
			}},
		}},
	}

	want := strings.TrimSpace(`
@media (hover: hover) {
  :where(:focus) :is(:where(:first-child) :is(:where(:nth-child(even)) .in-even\:in-first\:hover\:has-only\:in-focus\:opacity-100):hover:has(:only-child)) {
    opacity: 100%;
  }
}
`)

	got := renderQuery(t, optUnnest(input))
	if got != want {
		t.Fatalf("got:\n%s\n\nwant:\n%s", got, want)
	}
}

func TestOptUnnestInVariantStackSelection(t *testing.T) {
	class := `.in-even\:in-first\:hover\:selection\:has-only\:in-focus\:opacity-100`
	input := &Query{
		Name: class,
		Nested: []Query{{
			Name: `:where(:nth-child(even)) &`,
			Nested: []Query{{
				Name: `:where(:first-child) &`,
				Nested: []Query{{
					Name: `&:hover`,
					Nested: []Query{{
						Name: `@media (hover: hover)`,
						Nested: []Query{{
							Name: `& *::selection, &::selection`,
							Nested: []Query{{
								Name: `&:has(:only-child)`,
								Nested: []Query{{
									Name: `:where(:focus) &`,
									KVs:  newSimpleCssKV("opacity", "100%"),
								}},
							}},
						}},
					}},
				}},
			}},
		}},
	}

	want := strings.TrimSpace(`
@media (hover: hover) {
  :where(:focus) :is(:where(:first-child) :is(:where(:nth-child(even)) .in-even\:in-first\:hover\:selection\:has-only\:in-focus\:opacity-100):hover ::selection:has(:only-child)) {
    opacity: 100%;
  }
  :where(:focus) :is(:where(:first-child) :is(:where(:nth-child(even)) .in-even\:in-first\:hover\:selection\:has-only\:in-focus\:opacity-100):hover::selection:has(:only-child)) {
    opacity: 100%;
  }
}
`)

	got := renderQuery(t, optUnnest(input))
	if got != want {
		t.Fatalf("got:\n%s\n\nwant:\n%s", got, want)
	}
}

func TestOptUnnestSelectionFirstInVariantStack(t *testing.T) {
	class := `.selection\:in-even\:in-first\:hover\:has-only\:in-focus\:opacity-100`
	input := &Query{
		Name: class,
		Nested: []Query{{
			Name: `& *::selection, &::selection`,
			Nested: []Query{{
				Name: `:where(:nth-child(even)) &`,
				Nested: []Query{{
					Name: `:where(:first-child) &`,
					Nested: []Query{{
						Name: `&:hover`,
						Nested: []Query{{
							Name: `@media (hover: hover)`,
							Nested: []Query{{
								Name: `&:has(:only-child)`,
								Nested: []Query{{
									Name: `:where(:focus) &`,
									KVs:  newSimpleCssKV("opacity", "100%"),
								}},
							}},
						}},
					}},
				}},
			}},
		}},
	}

	want := strings.TrimSpace(`
@media (hover: hover) {
  :where(:focus) :is(:where(:first-child) :is(:where(:nth-child(even)) :is(.selection\:in-even\:in-first\:hover\:has-only\:in-focus\:opacity-100 ::selection)):hover:has(:only-child)) {
    opacity: 100%;
  }
  :where(:focus) :is(:where(:first-child) :is(:where(:nth-child(even)) .selection\:in-even\:in-first\:hover\:has-only\:in-focus\:opacity-100::selection):hover:has(:only-child)) {
    opacity: 100%;
  }
}
`)

	got := renderQuery(t, optUnnest(input))
	if got != want {
		t.Fatalf("got:\n%s\n\nwant:\n%s", got, want)
	}
}

func TestOptUnnestSelectionLastInVariantStack(t *testing.T) {
	class := `.in-even\:in-first\:hover\:has-only\:in-focus\:selection\:opacity-100`
	input := &Query{
		Name: class,
		Nested: []Query{{
			Name: `:where(:nth-child(even)) &`,
			Nested: []Query{{
				Name: `:where(:first-child) &`,
				Nested: []Query{{
					Name: `&:hover`,
					Nested: []Query{{
						Name: `@media (hover: hover)`,
						Nested: []Query{{
							Name: `&:has(:only-child)`,
							Nested: []Query{{
								Name: `:where(:focus) &`,
								Nested: []Query{{
									Name: `& *::selection, &::selection`,
									KVs:  newSimpleCssKV("opacity", "100%"),
								}},
							}},
						}},
					}},
				}},
			}},
		}},
	}

	want := strings.TrimSpace(`
@media (hover: hover) {
  :where(:focus) :is(:where(:first-child) :is(:where(:nth-child(even)) .in-even\:in-first\:hover\:has-only\:in-focus\:selection\:opacity-100):hover:has(:only-child)) ::selection {
    opacity: 100%;
  }
  :where(:focus) :is(:where(:first-child) :is(:where(:nth-child(even)) .in-even\:in-first\:hover\:has-only\:in-focus\:selection\:opacity-100):hover:has(:only-child))::selection {
    opacity: 100%;
  }
}
`)

	got := renderQuery(t, optUnnest(input))
	if got != want {
		t.Fatalf("got:\n%s\n\nwant:\n%s", got, want)
	}
}

func TestOptUnnestSelection(t *testing.T) {
	input := &Query{
		Name: `.selection\:flex`,
		Nested: []Query{{
			Name: `& *::selection, &::selection`,
			KVs:  newSimpleCssKV("display", "flex"),
		}},
	}

	want := strings.TrimSpace(`
.selection\:flex ::selection {
  display: flex;
}
.selection\:flex::selection {
  display: flex;
}
`)

	got := strings.TrimSpace(renderDoc(t, &Doc{Entries: optUnnestEntries([]Entry{{Query: input}})}))
	got = strings.ReplaceAll(got, "\n\n", "\n")
	if got != want {
		t.Fatalf("got:\n%s\n\nwant:\n%s", got, want)
	}
}

func TestOptUnnestHoverSelectionInEmpty(t *testing.T) {
	class := `.hover\:selection\:in-empty\:bg-fuchsia-300`
	input := &Query{
		Name: class,
		Nested: []Query{{
			Name: `&:hover`,
			Nested: []Query{{
				Name: `@media (hover: hover)`,
				Nested: []Query{{
					Name: `& *::selection, &::selection`,
					Nested: []Query{{
						Name: `:where(:empty) &`,
						KVs:  newSimpleCssKV("background-color", "var(--color-fuchsia-300)"),
					}},
				}},
			}},
		}},
	}

	want := strings.TrimSpace(`
@media (hover: hover) {
  :where(:empty) :is(.hover\:selection\:in-empty\:bg-fuchsia-300:hover ::selection) {
    background-color: var(--color-fuchsia-300);
  }
  :where(:empty) .hover\:selection\:in-empty\:bg-fuchsia-300:hover::selection {
    background-color: var(--color-fuchsia-300);
  }
}
`)

	got := renderQuery(t, optUnnest(input))
	if got != want {
		t.Fatalf("got:\n%s\n\nwant:\n%s", got, want)
	}
}

func TestOptUnnestNotSupports(t *testing.T) {
	input := &Query{
		Name: `.not-supports-[display\:grid]\:flex`,
		Nested: []Query{{
			Name: `@supports not (display:grid)`,
			KVs:  newSimpleCssKV("display", "flex"),
		}},
	}

	want := strings.TrimSpace(`
@supports not (display:grid) {
  .not-supports-[display\:grid]\:flex {
    display: flex;
  }
}
`)

	got := renderQuery(t, optUnnest(input))
	if got != want {
		t.Fatalf("got:\n%s\n\nwant:\n%s", got, want)
	}
}

func TestOptCombine(t *testing.T) {
	doc := &Doc{Entries: []Entry{
		{Query: &Query{
			Name: `@media (hover: hover)`,
			Nested: []Query{{
				Name: `.hover\:not-empty\:first\:bg-indigo-600:hover:not(:empty):first-child`,
				KVs:  newSimpleCssKV("background-color", "var(--color-indigo-600)"),
			}},
		}},
		{Query: &Query{
			Name: `@media (hover: hover)`,
			Nested: []Query{{
				Name: `@media (width >= 48rem)`,
				Nested: []Query{{
					Name: `.hover\:md\:not-empty\:bg-indigo-600:hover:not(:empty)`,
					KVs:  newSimpleCssKV("background-color", "var(--color-indigo-600)"),
				}},
			}},
		}},
	}}

	want := strings.TrimSpace(`
@media (hover: hover) {
  .hover\:not-empty\:first\:bg-indigo-600:hover:not(:empty):first-child {
    background-color: var(--color-indigo-600);
  }
  @media (width >= 48rem) {
    .hover\:md\:not-empty\:bg-indigo-600:hover:not(:empty) {
      background-color: var(--color-indigo-600);
    }
  }
}
`)

	got := renderDoc(t, optCombine(doc))
	if got != want {
		t.Fatalf("got:\n%s\n\nwant:\n%s", got, want)
	}
}

func TestOptCombineKeepsDistinctTopLevelRules(t *testing.T) {
	doc := &Doc{Entries: []Entry{
		{Query: &Query{
			Name: `.flex`,
			KVs:  newSimpleCssKV("display", "flex"),
		}},
		{Query: &Query{
			Name: `@media (prefers-color-scheme: dark)`,
			Nested: []Query{{
				Name: `.dark\:flex`,
				KVs:  newSimpleCssKV("display", "flex"),
			}},
		}},
	}}

	got := renderDoc(t, optCombine(doc))
	if strings.Count(got, "@media (prefers-color-scheme: dark)") != 1 {
		t.Fatalf("expected one dark media block, got:\n%s", got)
	}
	if !strings.Contains(got, ".flex {") {
		t.Fatalf("expected bare .flex rule, got:\n%s", got)
	}
}

func renderDoc(t *testing.T, doc *Doc) string {
	t.Helper()

	var buf bytes.Buffer
	if err := doc.render(&buf); err != nil {
		t.Fatal(err)
	}

	return strings.TrimSpace(buf.String())
}

func renderQuery(t *testing.T, q *Query) string {
	t.Helper()

	var buf bytes.Buffer
	if err := q.render(&buf, 0); err != nil {
		t.Fatal(err)
	}

	return strings.TrimSpace(buf.String())
}
