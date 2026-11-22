//go:build ignore

// make writes what Shugo ships of an installed svelte package and its
// dependencies, all but the compiler:
//
//	npm install --prefix /tmp/svelte-vendor svelte@5.59.1
//	go run internal/svelte/vendor/make.go /tmp/svelte-vendor
//
// It writes runtime/, the server runtime, for pages and for the islands
// rendered on the server; runtime-browser/, the browser runtime, for islands
// in the browser; and acorn.cjs, the parser of JavaScript that Svelte uses,
// unmodified, for reading the server code of an island bound to a request.
//
// In a runtime every module a page or an island may import from Svelte is
// one file, named after it; the code they share is in chunks, so that they
// share their state too, as the modules of the package do. The code is that
// of the package as it is, bundled for production: esm-env answers DEV
// false.
package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/evanw/esbuild/pkg/api"
)

// The modules of Svelte a page may import. Each becomes a file named after
// it: svelte/internal/server is internal-server.js, as Runtime of package
// svelte expects.
var server = []string{
	"svelte",
	"svelte/internal/server",
	"svelte/server",
	"svelte/store",
	"svelte/reactivity",
	"svelte/reactivity/window",
	"svelte/easing",
	"svelte/transition",
	"svelte/animate",
	"svelte/motion",
	"svelte/events",
	"svelte/attachments",
}

// The modules of Svelte an island may import in the browser.
var browser = []string{
	"svelte",
	"svelte/internal/client",
	"svelte/internal/init-operations",
	"svelte/store",
	"svelte/reactivity",
	"svelte/reactivity/window",
	"svelte/easing",
	"svelte/transition",
	"svelte/animate",
	"svelte/motion",
	"svelte/events",
	"svelte/attachments",
}

func main() {
	if len(os.Args) != 2 {
		fmt.Fprintln(os.Stderr, "usage: go run internal/svelte/vendor/make.go <folder with node_modules/svelte>")
		os.Exit(2)
	}
	root, err := filepath.Abs(os.Args[1])
	if err != nil {
		fail(err)
	}
	// Run from the root of the repository.
	vendor, err := filepath.Abs(filepath.Join("internal", "svelte", "vendor"))
	if err != nil {
		fail(err)
	}
	bundle(root, server, filepath.Join(vendor, "runtime"), api.PlatformNode)
	bundle(root, browser, filepath.Join(vendor, "runtime-browser"), api.PlatformBrowser)
	acorn, err := os.ReadFile(filepath.Join(root, "node_modules", "acorn", "dist", "acorn.js"))
	if err != nil {
		fail(err)
	}
	if err := os.WriteFile(filepath.Join(vendor, "acorn.cjs"), acorn, 0o644); err != nil {
		fail(err)
	}
	fmt.Printf("%7d acorn.cjs\n", len(acorn))
}

// bundle writes the modules of Svelte for a platform into a folder.
func bundle(root string, modules []string, out string, platform api.Platform) {
	var points []api.EntryPoint
	for _, module := range modules {
		name := "svelte"
		if rest, sub := strings.CutPrefix(module, "svelte/"); sub {
			name = strings.ReplaceAll(rest, "/", "-")
		}
		points = append(points, api.EntryPoint{InputPath: module, OutputPath: name})
	}
	result := api.Build(api.BuildOptions{
		EntryPointsAdvanced: points,
		AbsWorkingDir:       root,
		Bundle:              true,
		Splitting:           true,
		Format:              api.FormatESModule,
		Platform:            platform,
		Target:              api.ES2022,
		Conditions:          []string{"production"},
		MinifySyntax:        true, // drops what only runs in development
		Outdir:              out,
		ChunkNames:          "chunk-[hash]",
		Charset:             api.CharsetUTF8,
		LegalComments:       api.LegalCommentsNone,
		Write:               false,
		LogLevel:            api.LogLevelError,
	})
	if len(result.Errors) > 0 {
		fail(fmt.Errorf("%v", result.Errors))
	}
	if err := os.RemoveAll(out); err != nil {
		fail(err)
	}
	if err := os.MkdirAll(out, 0o755); err != nil {
		fail(err)
	}
	for _, file := range result.OutputFiles {
		if err := os.WriteFile(file.Path, file.Contents, 0o644); err != nil {
			fail(err)
		}
		fmt.Printf("%7d %s/%s\n", len(file.Contents), filepath.Base(out), filepath.Base(file.Path))
	}
}

func fail(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}
