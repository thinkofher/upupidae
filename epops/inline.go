package epops

import (
	"context"

	"github.com/thinkofher/upupidae"
)

// InlineViewer inlines CSS document into the HTMl document directly.
type InlineViewer[T any] interface {
	// Style creates a <style> element of type T.
	Style(ctx context.Context, doc *upupidae.Doc) T

	Attributer[T]
	NodeCompositor[T]
}

// InlineGen generates CSS documents dynamically by inlining them in the HTML
// DOM tree.
type InlineGen[T any] struct {
	genFactory func(ctx context.Context, id string) *upupidae.Gen
	viewer     InlineViewer[T]
}

// NewInlineGen is the only way to initialize [InlineGen].
func NewInlineGen[T any](
	gf func(ctx context.Context, id string) *upupidae.Gen,
	v InlineViewer[T],
) *InlineGen[T] {
	return &InlineGen[T]{
		genFactory: gf,
		viewer:     v,
	}
}

// Compositor returns current compositor for the Node type you are using.
func (g *InlineGen[T]) Compositor() NodeCompositor[T] {
	return g.viewer
}

// Nodes return styles and attribute nodes that you inject into your HTML
// document.
func (g *InlineGen[T]) Nodes(ctx context.Context, classes ...string) (styles T, attr T) {
	hs := hashClasses(classes)

	s := g.viewer.Empty()
	a := g.viewer.Attr(ctx, hs)

	if !acquireClass(ctx, hs) {
		dg := g.genFactory(ctx, hs)
		d := dg.Doc(ctx, classes...)
		s = g.viewer.Style(ctx, d)
	}

	return s, a
}
