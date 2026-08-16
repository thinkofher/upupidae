package epops

import (
	"context"
	"crypto/rand"
	"fmt"
	"io"
	"net/http"
	"sync"

	"github.com/thinkofher/upupidae"
)

// HTTPViewer renders elements for HTTP dynamic stylesheet generation.
type HTTPViewer[T any] interface {
	// Link renders element that fetches stylesheet for given id.
	Link(ctx context.Context, id string) T

	Attributer[T]
	NodeCompositor[T]
}

// HTTPGen generates CSS documents dynamically, saves them and serve them over
// HTTP protocol.
type HTTPGen[T any] struct {
	factory func(ctx context.Context, id string) *upupidae.Gen
	execID  string
	reqID   func(*http.Request) string
	viewer  HTTPViewer[T]
	cache   *cache
}

// NewHTTPGen is the only way to initialize [HTTPGen].
func NewHTTPGen[T any](
	factory func(ctx context.Context, id string) *upupidae.Gen,
	reqID func(*http.Request) string,
	viewer HTTPViewer[T],
) *HTTPGen[T] {
	return &HTTPGen[T]{
		execID:  rand.Text(),
		factory: factory,
		reqID:   reqID,
		viewer:  viewer,
		cache: &cache{
			m:   map[string]*upupidae.Doc{},
			mtx: sync.RWMutex{},
		},
	}
}

func (g *HTTPGen[T]) docID(ctx context.Context, classes ...string) (id string) {
	if len(classes) == 0 {
		return
	}

	id = hashClasses(classes)
	if _, ok := g.cache.doc(id); ok {
		return
	}

	gg := g.factory(ctx, id)
	d := gg.Doc(ctx, classes...)

	if len(d.Entries) == 0 {
		return
	}

	g.cache.save(id, d)

	return id
}

// Nodes return styles and attribute nodes that you inject into your HTML
// document.
func (g *HTTPGen[T]) Nodes(ctx context.Context, classes ...string) (styles T, attr T) {
	id := g.docID(ctx, classes...)

	s := g.viewer.Empty()
	a := g.viewer.Attr(ctx, id)

	if !acquireClass(ctx, id) {
		s = g.viewer.Link(ctx, id)
	}

	return s, a
}

// Compositor returns current compositor for the Node type you are using.
func (g *HTTPGen[T]) Compositor() NodeCompositor[T] {
	return g.viewer
}

// ServeHTTP serves dynamically generated CSS documents over HTTP protocol.
func (g *HTTPGen[T]) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	id := g.reqID(r)

	etag := fmt.Sprintf("%s-%s", id, g.execID)
	if match := r.Header.Get("If-None-Match"); match == etag {
		w.WriteHeader(http.StatusNotModified)
		return
	}

	w.Header().Set("ETag", etag)
	w.Header().Set("Cache-Control", "public, max-age=604800")
	w.Header().Set("Content-Type", "text/css; charset=utf-8")

	d, ok := g.cache.doc(id)
	if !ok {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	dr := d.Reader()
	defer dr.Close()

	_, err := io.Copy(w, dr)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
}

type cache struct {
	m   map[string]*upupidae.Doc
	mtx sync.RWMutex
}

func (c *cache) doc(id string) (d *upupidae.Doc, ok bool) {
	c.mtx.RLock()
	defer c.mtx.RUnlock()

	d, ok = c.m[id]
	return
}

func (c *cache) save(id string, d *upupidae.Doc) {
	c.mtx.Lock()
	defer c.mtx.Unlock()
	c.m[id] = d
}
