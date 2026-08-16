package upupidae

import (
	"crypto/sha256"
	"encoding/hex"
	"io"
	"net/http"
)

// DocHandler is a default [http.Handler] for CSS generated [Doc]. It support
// basic ETag HTTP specification if you set etag flag to true.
func DocHandler(doc *Doc, etag bool) http.Handler {
	var etagv string

	if etag {
		hr := doc.Reader()
		defer hr.Close()

		sh := sha256.New()
		io.Copy(sh, hr)
		etagv = hex.EncodeToString(sh.Sum(nil))
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if etag {
			if match := r.Header.Get("If-None-Match"); match == etagv {
				w.WriteHeader(http.StatusNotModified)
				return
			}

			w.Header().Set("ETag", etagv)
			w.Header().Set("Cache-Control", "public, max-age=604800")
		}

		dr := doc.Reader()
		defer dr.Close()

		w.Header().Set("Content-Type", "text/css; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		io.Copy(w, dr)
	})
}
