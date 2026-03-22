package middleware

import (
	"io"
	"net/http"
	"strings"
	"sync"

	"github.com/klauspost/compress/flate"
	"github.com/klauspost/compress/gzip"
	"github.com/klauspost/compress/zstd"

	"github.com/pocketbase/pocketbase/core"
)

// compressible content types worth compressing (prefix match).
var compressibleTypes = []string{
	"text/",
	"application/json",
	"application/xml",
	"application/rss+xml",
	"application/atom+xml",
	"application/activity+json",
	"application/jrd+json",
	"application/javascript",
	"image/svg+xml",
	"text/calendar",
}

var gzipWriterPool = sync.Pool{
	New: func() any {
		w, _ := gzip.NewWriterLevel(io.Discard, gzip.DefaultCompression)
		return w
	},
}

var flateWriterPool = sync.Pool{
	New: func() any {
		w, _ := flate.NewWriter(io.Discard, flate.DefaultCompression)
		return w
	},
}

var zstdWriterPool = sync.Pool{
	New: func() any {
		w, _ := zstd.NewWriter(nil, zstd.WithEncoderLevel(zstd.SpeedDefault))
		return w
	},
}

// Compress is a PocketBase middleware that compresses responses using
// the best encoding the client supports (zstd > gzip > deflate).
func Compress(e *core.RequestEvent) error {
	enc := negotiateEncoding(e.Request.Header.Get("Accept-Encoding"))
	if enc == "" {
		return e.Next()
	}

	rw := e.Response
	cw := &compressWriter{
		ResponseWriter: rw,
		encoding:       enc,
	}
	e.Response = cw
	defer cw.Close()

	return e.Next()
}

// negotiateEncoding picks the best supported encoding from Accept-Encoding.
// Preference order: zstd > gzip > deflate.
func negotiateEncoding(accept string) string {
	if accept == "" {
		return ""
	}
	if strings.Contains(accept, "zstd") {
		return "zstd"
	}
	if strings.Contains(accept, "gzip") {
		return "gzip"
	}
	if strings.Contains(accept, "deflate") {
		return "deflate"
	}
	return ""
}

type compressWriter struct {
	http.ResponseWriter
	encoding    string
	writer      io.WriteCloser
	wroteHeader bool
}

func (cw *compressWriter) Write(b []byte) (int, error) {
	if !cw.wroteHeader {
		cw.WriteHeader(http.StatusOK)
	}
	if cw.writer != nil {
		return cw.writer.Write(b)
	}
	return cw.ResponseWriter.Write(b)
}

func (cw *compressWriter) WriteHeader(code int) {
	if cw.wroteHeader {
		return
	}
	cw.wroteHeader = true

	h := cw.ResponseWriter.Header()

	// Don't compress if already encoded
	if h.Get("Content-Encoding") != "" {
		cw.ResponseWriter.WriteHeader(code)
		return
	}

	// Only compress compressible content types
	ct := h.Get("Content-Type")
	if !isCompressible(ct) {
		cw.ResponseWriter.WriteHeader(code)
		return
	}

	// Set up compression
	h.Set("Content-Encoding", cw.encoding)
	h.Add("Vary", "Accept-Encoding")
	h.Del("Content-Length") // length changes with compression

	switch cw.encoding {
	case "gzip":
		w := gzipWriterPool.Get().(*gzip.Writer)
		w.Reset(cw.ResponseWriter)
		cw.writer = w
	case "deflate":
		w := flateWriterPool.Get().(*flate.Writer)
		w.Reset(cw.ResponseWriter)
		cw.writer = w
	case "zstd":
		w := zstdWriterPool.Get().(*zstd.Encoder)
		w.Reset(cw.ResponseWriter)
		cw.writer = w
	}

	cw.ResponseWriter.WriteHeader(code)
}

func (cw *compressWriter) Close() error {
	if cw.writer == nil {
		return nil
	}
	err := cw.writer.Close()

	switch cw.encoding {
	case "gzip":
		gzipWriterPool.Put(cw.writer)
	case "deflate":
		flateWriterPool.Put(cw.writer)
	case "zstd":
		zstdWriterPool.Put(cw.writer)
	}

	return err
}

func isCompressible(contentType string) bool {
	ct := strings.ToLower(contentType)
	for _, prefix := range compressibleTypes {
		if strings.HasPrefix(ct, prefix) {
			return true
		}
	}
	return false
}
