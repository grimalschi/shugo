package test

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"sync/atomic"
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
		html = page.Body
		bytes += len(html)
	}
	b.ReportMetric(float64(bytes)/float64(b.N), "bytes/page")
}

// BenchmarkServe asks the server for product pages from as many clients as
// there are processors: as they are, and compressed by gzip. The clients
// keep their connections and do not unpack what they get.
func BenchmarkServe(b *testing.B) {
	base, size := benchCatalogue(b)
	s := openSite(b, base.root)
	s.build()
	ts := serveSite(b, s)
	transport := &http.Transport{DisableCompression: true, MaxIdleConnsPerHost: runtime.GOMAXPROCS(0)}
	b.Cleanup(transport.CloseIdleConnections)
	for _, encoding := range []string{"identity", "gzip"} {
		b.Run(encoding, func(b *testing.B) {
			var sent atomic.Int64
			b.RunParallel(func(pb *testing.PB) {
				client := &http.Client{Transport: transport}
				for i := 0; pb.Next(); i++ {
					request, _ := http.NewRequest(http.MethodGet, ts.URL+fmt.Sprintf("/products/product-%d", i%size+1), nil)
					request.Header.Set("Accept-Encoding", encoding)
					response, err := client.Do(request)
					if err != nil {
						b.Fatal(err)
					}
					n, _ := io.Copy(io.Discard, response.Body)
					response.Body.Close()
					if response.StatusCode != 200 {
						b.Fatal(response.Status)
					}
					sent.Add(n)
				}
			})
			b.ReportMetric(float64(sent.Load())/float64(b.N), "bytes/page")
		})
	}
}
