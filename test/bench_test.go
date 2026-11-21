package test

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"testing"
)

// benchCatalogue builds the shop of examples/catalog with the given number of
// products. The number comes from SHUGO_BENCH_SIZE, 1000 if it is not set.
func benchCatalogue(b *testing.B) (*site, int) {
	b.Helper()
	size := 1000
	if text := os.Getenv("SHUGO_BENCH_SIZE"); text != "" {
		size, _ = strconv.Atoi(text)
	}
	s := &site{root: b.TempDir(), runtime: os.Getenv("SHUGO_RUNTIME")}
	if err := os.CopyFS(s.root, os.DirFS("../examples/catalog")); err != nil {
		b.Fatal(err)
	}
	os.RemoveAll(filepath.Join(s.root, "_build"))
	generate := exec.Command("node", "scripts/generate.mjs", strconv.Itoa(size))
	generate.Dir = s.root
	if out, err := generate.CombinedOutput(); err != nil {
		b.Fatalf("%v: %s", err, out)
	}
	return s, size
}

// BenchmarkRender measures building one page of the shop from the store.
func BenchmarkRender(b *testing.B) {
	base, size := benchCatalogue(b)
	s := openSite(b, base.root)
	s.build()
	var html []byte
	bytes := 0
	b.ResetTimer()
	for i := range b.N {
		page, err := s.pages.Render(html[:0], fmt.Sprintf("/products/product-%d", i%size+1))
		if err != nil || !page.Found {
			b.Fatal(err, page.Found)
		}
		html = page.HTML
		bytes += len(html)
	}
	b.ReportMetric(float64(bytes)/float64(b.N), "bytes/page")
}

// BenchmarkServe measures whole requests to a running server, from as many
// clients as there are processors.
func BenchmarkServe(b *testing.B) {
	base, size := benchCatalogue(b)
	s := openSite(b, base.root)
	s.build()
	ts := serveSite(b, s)
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		client := &http.Client{}
		for i := 0; pb.Next(); i++ {
			response, err := client.Get(ts.URL + fmt.Sprintf("/products/product-%d", i%size+1))
			if err != nil {
				b.Fatal(err)
			}
			io.Copy(io.Discard, response.Body)
			response.Body.Close()
			if response.StatusCode != 200 {
				b.Fatal(response.Status)
			}
		}
	})
}
