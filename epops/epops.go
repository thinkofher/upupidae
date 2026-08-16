package epops

import (
	"context"
	_ "embed"
	"hash/fnv"
	"net/http"
	"slices"
)

var (
	// UpupidaeStyleJS is a JavaScript program that defines <upupidae-style>
	// custom element that lets you inject CSS stylesheet into <head> element of
	// HTML document.
	//go:embed epops.js
	UpupidaeStyleJS string

	// UpupidaeStyleMinJS is a minified version of [UpupidaeStyleJS].
	//go:embed epops.min.js
	UpupidaeStyleMinJS string
)

// Attributer creates the attribute, HTML element.
type Attributer[T any] interface {
	// Attr creates an HTML attribute that marks the element as target of CSS
	// queries from the doc.
	Attr(ctx context.Context, hash string) T
}

// NodeCompositor defines set of operations on elements. This way you can use
// upupidae/epops with whatever template library you want.
type NodeCompositor[T any] interface {
	// Group groups together many HTML nodes, both attribtues and children
	// elements.
	Group([]T) T

	// Empty return empty node.
	Empty() T

	// IsEmpty returns true if given T node is empty.
	IsEmpty(T) bool
}

// LimitStyleNodes limit rendering of <style> and other related stylesheet
// definition elements to the very minimum per context within a single HTTP
// request.
func LimitStyleNodes(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := context.WithValue(r.Context(), acquireClassMapKey, map[string]struct{}{})
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

const acquireClassMapKey = "epops_____acquire_class_map"

func acquireClass(ctx context.Context, c string) (alreadyAquired bool) {
	m, ok := ctx.Value(acquireClassMapKey).(map[string]struct{})
	if !ok || m == nil {
		return false
	}

	_, ok = m[c]
	if !ok {
		m[c] = struct{}{}
		return false
	}

	return true
}

func hashClasses(classes []string) string {
	h := fnv.New64a()

	for _, class := range slices.Sorted(slices.Values(classes)) {
		h.Write([]byte(class))
		h.Write([]byte{0}) // separator
	}

	return base62(h.Sum64())
}

const base62Alphabet = "0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"

func base62(n uint64) string {
	if n == 0 {
		return "0"
	}

	var buf [11]byte
	i := len(buf)

	for n > 0 {
		i--
		buf[i] = base62Alphabet[n%62]
		n /= 62
	}

	return string(buf[i:])
}

// Noder generates style nodes and supports composition operations on those
// nodes.
type Noder[T any] interface {
	// Compositor returns [NodeCompositor] for given node type T.
	Compositor() NodeCompositor[T]

	// Nodes returns style and attribute node for given set of utility classes.
	Nodes(ctx context.Context, classes ...string) (styles T, attr T)
}

// NodesBuilder creates HTML element nodes for both Container and Void elements.
type NodesBuilder[T any] struct {
	noder Noder[T]
}

// NewNodesBuilder is the only way to initialize [NodesBuilder].
func NewNodesBuilder[T any](n Noder[T]) *NodesBuilder[T] {
	return &NodesBuilder[T]{noder: n}
}

// Container returns nodes grouped together that you can inject into container
// HTML element like <body> or <main>.
func (n *NodesBuilder[T]) Container(ctx context.Context, classes ...string) T {
	s, a := n.noder.Nodes(ctx, classes...)

	c := n.noder.Compositor()
	if c.IsEmpty(s) {
		return a
	}

	return c.Group([]T{s, a})
}

// Void returns VoidBuilder for styling void element like <input>. This way you
// can have list of classes first and then define your element in the node
// tree.
func (n *NodesBuilder[T]) Void(ctx context.Context, classes ...string) VoidBuilder[T] {
	s, a := n.noder.Nodes(ctx, classes...)
	c := n.noder.Compositor()

	if c.IsEmpty(s) {
		return VoidBuilder[T](func(nf func(attr T) T) T {
			return nf(a)
		})
	}

	return VoidBuilder[T](func(nf func(attr T) T) T {
		return c.Group([]T{s, nf(a)})
	})
}

// VoidBuilder lets you style void elements like <input>.
type VoidBuilder[T any] func(nf func(attr T) T) T

// El accepts void factory function that accepts attribute and returns void
// element with that attributed injected as child.
func (vb VoidBuilder[T]) El(nf func(attr T) T) T {
	return vb(nf)
}
