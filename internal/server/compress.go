package server

import (
	"bytes"
	"compress/gzip"
	"hash/crc64"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
)

// Pages and text files go out compressed to a browser that takes gzip: an
// HTML page is several times smaller so. A page carries a tag made of its
// content, and a browser that asks with the tag of the page it has gets 304
// and no body when the page is the same.

// compressFrom is the size under which an answer goes as it is: gzip would
// save little and cost a few microseconds.
const compressFrom = 1024

// Text files of the site, which compress well; pictures and fonts are
// compressed already.
var compressible = map[string]bool{
	".html": true, ".css": true, ".js": true, ".mjs": true, ".json": true, ".map": true,
	".svg": true, ".txt": true, ".xml": true, ".webmanifest": true,
}

var (
	gzipWriters = sync.Pool{New: func() any {
		w, _ := gzip.NewWriterLevel(nil, gzip.BestSpeed)
		return w
	}}
	compressed = sync.Pool{New: func() any { return new(bytes.Buffer) }}
	tagTable   = crc64.MakeTable(crc64.ECMA)
)

// acceptsGzip tells whether the browser takes answers compressed by gzip.
func acceptsGzip(r *http.Request) bool {
	for _, part := range strings.Split(r.Header.Get("Accept-Encoding"), ",") {
		coding, params, _ := strings.Cut(part, ";")
		if !strings.EqualFold(strings.TrimSpace(coding), "gzip") {
			continue
		}
		// gzip;q=0 refuses it.
		if q, found := strings.CutPrefix(strings.ReplaceAll(params, " ", ""), "q="); found {
			value, err := strconv.ParseFloat(q, 64)
			return err == nil && value > 0
		}
		return true
	}
	return false
}

// gzipped compresses body into a buffer from the pool; put it back with
// release.
func gzipped(body []byte) *bytes.Buffer {
	out := compressed.Get().(*bytes.Buffer)
	out.Reset()
	w := gzipWriters.Get().(*gzip.Writer)
	w.Reset(out)
	w.Write(body)
	w.Close()
	gzipWriters.Put(w)
	return out
}

func release(b *bytes.Buffer) {
	if b.Cap() <= 1<<20 {
		compressed.Put(b)
	}
}

// pageTag is the tag of a page: it changes when the page does. It is weak,
// W/"…", because the page goes out now compressed, now not.
func pageTag(html []byte) string {
	return `W/"` + strconv.FormatUint(crc64.Checksum(html, tagTable), 36) + `"`
}

// hasTag tells whether the If-None-Match of a request names the tag.
func hasTag(header, tag string) bool {
	if header == "" {
		return false
	}
	if strings.TrimSpace(header) == "*" {
		return true
	}
	want := strings.TrimPrefix(tag, "W/")
	for _, one := range strings.Split(header, ",") {
		if strings.TrimPrefix(strings.TrimSpace(one), "W/") == want {
			return true
		}
	}
	return false
}

// write sends a body, compressed if the browser takes it and it is worth it.
func write(w http.ResponseWriter, r *http.Request, status int, body []byte) {
	header := w.Header()
	header.Add("Vary", "Accept-Encoding")
	if len(body) >= compressFrom && acceptsGzip(r) {
		packed := gzipped(body)
		defer release(packed)
		body = packed.Bytes()
		header.Set("Content-Encoding", "gzip")
	}
	header.Set("Content-Length", strconv.Itoa(len(body)))
	w.WriteHeader(status)
	if r.Method != http.MethodHead {
		w.Write(body)
	}
}

// serveFile sends a file of a folder. A text file goes compressed to a
// browser that takes gzip; everything else, and ranges, http.ServeFile
// answers.
func serveFile(w http.ResponseWriter, r *http.Request, name string, info os.FileInfo) {
	if !compressible[strings.ToLower(filepath.Ext(name))] || info.Size() < compressFrom || info.Size() > 8<<20 ||
		!acceptsGzip(r) || r.Header.Get("Range") != "" {
		w.Header().Add("Vary", "Accept-Encoding")
		http.ServeFile(w, r, name)
		return
	}
	data, err := os.ReadFile(name)
	if err != nil {
		http.ServeFile(w, r, name)
		return
	}
	packed := gzipped(data)
	defer release(packed)
	header := w.Header()
	header.Add("Vary", "Accept-Encoding")
	header.Set("Content-Encoding", "gzip")
	http.ServeContent(w, r, name, info.ModTime(), bytes.NewReader(packed.Bytes()))
}
