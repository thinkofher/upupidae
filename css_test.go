package upupidae

import (
	"context"
	"slices"
	"testing"

	"github.com/thinkofher/upupidae/internal/parser"
)

func TestCssEscape(t *testing.T) {
	tests := []struct {
		value string
		want  string
	}{
		{"flex", "flex"},
		{"hover:flex", "hover\\:flex"},
		{"@md:flex", "\\@md\\:flex"},
		{"@container/sidebar:flex", "\\@container\\/sidebar\\:flex"},
		{"supports-[display:grid]:@md:hover:flex", "supports-\\[display\\:grid\\]\\:\\@md\\:hover\\:flex"},
		{"w-[calc(100%-1rem)]", "w-\\[calc\\(100\\%-1rem\\)\\]"},
		{"bg-[#fff]", "bg-\\[\\#fff\\]"},
		{"text-[length:var(--foo)]", "text-\\[length\\:var\\(--foo\\)\\]"},
		{"group-[.foo]:flex", "group-\\[\\.foo\\]\\:flex"},
		{"not-[.foo]:flex", "not-\\[\\.foo\\]\\:flex"},
		{"in-[.group]:flex", "in-\\[\\.group\\]\\:flex"},
		{"top-1/2", "top-1\\/2"},
		{"bg-red-500/50", "bg-red-500\\/50"},
		{"[&_p]:flex", "\\[\\&_p\\]\\:flex"},
		{"aria-[expanded=true]:flex", "aria-\\[expanded\\=true\\]\\:flex"},
		{"has-[.child]:flex", "has-\\[\\.child\\]\\:flex"},
		{".foo", "\\.foo"},
		{"foo.bar", "foo\\.bar"},
		{"foo(bar)", "foo\\(bar\\)"},
		{"foo+bar", "foo\\+bar"},
		{"sm:!flex", "sm\\:\\!flex"},
		{`before:content-[""]`, `before\:content-\[\"\"\]`},
		{"underline-offset-2", "underline-offset-2"},
		{"-top-4", "-top-4"},
		{"2xl:flex", `\32 xl\:flex`},
	}

	for _, tt := range tests {
		t.Run(tt.value, func(t *testing.T) {
			if got := cssEscape(tt.value); got != tt.want {
				t.Fatalf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestCssSelectorClass(t *testing.T) {
	if got := cssSelectorClass("hover:flex"); got != ".hover\\:flex" {
		t.Fatalf("got %q, want %q", got, ".hover\\:flex")
	}
}

func TestGenerateBeforeContent(t *testing.T) {
	g := &gen{
		namer: NamerFunc(func(_ context.Context, c string) string { return cssSelectorClass(c) }),
		varer: &testVarer{},
		registerer: &registerer{
			registry: map[string]struct{}{},
		},
	}
	p := parser.Must()

	cases := []struct {
		class string
		want  []KV
	}{
		{
			class: "before:flex",
			want: []KV{
				newCssKV("display", "flex"),
				newCssKV("content", "var(--tw-content)"),
			},
		},
		{
			class: "before:content-['hello']",
			want: []KV{
				newCssKV("--tw-content", "'hello'"),
				newCssKV("content", "var(--tw-content)"),
			},
		},
		{
			class: "content-none",
			want: []KV{
				newCssKV("--tw-content", "none"),
				newCssKV("content", "none"),
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.class, func(t *testing.T) {
			ast, err := p.ParseString("", tc.class)
			if err != nil {
				t.Fatalf("parse: %v", err)
			}

			cls, err := g.generate(t.Context(), ast.Classes[0])
			if err != nil {
				t.Fatalf("generate: %v", err)
			}

			if !slices.Equal(cls.kvs, tc.want) {
				t.Fatalf("got %v, want %v", cls.kvs, tc.want)
			}
		})
	}
}
