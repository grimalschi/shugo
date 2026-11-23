package compile

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"

	"github.com/Masterminds/semver/v3"
	"github.com/evanw/esbuild/pkg/api"

	"github.com/grimalschi/shugo/internal/project"
	"github.com/grimalschi/shugo/internal/svelte"
)

// The packages of npm that the code of the site imports are installed by
// the package manager of the project into node_modules and bundled with
// that code: Svelte in them is the one the package shugo brings, as in the
// code of the site, and every file of theirs that goes into the bundle is a source
// of the build.

// bareImport finds an import of a package: a name, not a path, and not a
// module of the runtime such as node:fs.
const bareImport = `^(@[^/:]+/)?[^./@:][^:]*$`

// checkPackages has the bundler tell of the imports of packages that the
// build cannot take. pages tells that the bundle computes pages, which
// cannot show the components of packages yet.
func (b *builder) checkPackages(build api.PluginBuild, pages bool) {
	build.OnResolve(api.OnResolveOptions{Filter: bareImport, Namespace: "file"}, func(args api.OnResolveArgs) (api.OnResolveResult, error) {
		return api.OnResolveResult{}, b.checkPackage(build, args, pages)
	})
}

// checkPackage tells of an import of a package that is not installed, of
// one that gives components of Svelte to pages, and of one that needs
// another version of Svelte than Shugo is made for. nil lets the bundler
// resolve the import as it does.
func (b *builder) checkPackage(build api.PluginBuild, args api.OnResolveArgs, pages bool) error {
	if _, again := args.PluginData.(svelte.Resolving); again {
		return nil
	}
	name := packageName(args.Path)
	found := build.Resolve(args.Path, api.ResolveOptions{
		Importer: args.Importer, ResolveDir: args.ResolveDir, Kind: args.Kind,
		Namespace: args.Namespace, PluginData: svelte.Resolving{},
	})
	if len(found.Errors) > 0 {
		folder := svelte.Installed(args.ResolveDir, name)
		if folder == "" {
			return fmt.Errorf("%s is not installed: %s", name, b.install(name))
		}
		// The bundler tells what it does not find in the package, unless
		// the build could not take the package anyway.
		return b.packages.of(folder).problem(name, pages, false)
	}
	if found.External || found.Namespace != "file" {
		return nil // a module of the runtime, such as fs
	}
	_, folder := project.PackageOf(found.Path)
	if folder == "" {
		return nil // a package linked from outside node_modules
	}
	return b.packages.of(folder).problem(name, pages, strings.HasSuffix(found.Path, project.TemplateExtension))
}

// packageName returns the package an import names: "date-fns" of
// "date-fns/format", "@scope/ui" of "@scope/ui/button".
func packageName(specifier string) string {
	parts := strings.SplitN(specifier, "/", 3)
	if strings.HasPrefix(parts[0], "@") && len(parts) > 1 {
		return parts[0] + "/" + parts[1]
	}
	return parts[0]
}

// install tells how the package manager of the project adds a package: the
// one whose lock file the project has, or else the one that comes with the
// runtime that computes the pages.
func (b *builder) install(name string) string {
	for _, manager := range []struct{ lock, add string }{
		{"bun.lock", "bun add"},
		{"bun.lockb", "bun add"},
		{"package-lock.json", "npm install"},
		{"pnpm-lock.yaml", "pnpm add"},
		{"yarn.lock", "yarn add"},
	} {
		if _, err := os.Stat(b.project.Path(manager.lock)); err == nil {
			return manager.add + " " + name
		}
	}
	if filepath.Base(b.project.Runtime()) == "node" {
		return "npm install " + name
	}
	return "bun add " + name
}

// npmPackages is what the build has read of the packages the code imports,
// by their folders. The bundler resolves imports from many goroutines.
type npmPackages struct {
	mu    sync.Mutex
	known map[string]*npmPackage
}

// npmPackage is what the build needs to know of a package of npm, from its
// package.json.
type npmPackage struct {
	components bool   // it gives components of Svelte
	svelte     string // the versions of Svelte it needs, when Shugo's is not among them
}

func (p *npmPackages) of(folder string) *npmPackage {
	p.mu.Lock()
	defer p.mu.Unlock()
	if pkg, read := p.known[folder]; read {
		return pkg
	}
	if p.known == nil {
		p.known = map[string]*npmPackage{}
	}
	pkg := readPackage(folder)
	p.known[folder] = pkg
	return pkg
}

// problem tells why the build cannot take a package, or nil. pages tells
// that the package is imported by the code that computes pages, and
// component that the import names a component of Svelte.
func (pkg *npmPackage) problem(name string, pages, component bool) error {
	if pages && (pkg.components || component) {
		return fmt.Errorf("%s gives components of Svelte, and Shugo does not compile the components of packages yet: copy the component into src/components/ and import it from there", name)
	}
	if pkg.svelte != "" {
		return fmt.Errorf("%s needs Svelte %s, and Shugo is made for Svelte %s: take a version of %s made for Svelte %s", name, pkg.svelte, svelte.Version, name, strings.SplitN(svelte.Version, ".", 2)[0])
	}
	return nil
}

// readPackage reads what the build needs to know of a package. A package
// gives components of Svelte when its package.json names them, by the field
// "svelte" or the condition "svelte" of "exports", as packages of components
// tell SvelteKit. The versions of Svelte it needs are those it names among
// its peer dependencies or its dependencies; a range that is not one of npm,
// such as workspace:*, says nothing.
func readPackage(folder string) *npmPackage {
	pkg := &npmPackage{}
	data, err := os.ReadFile(filepath.Join(folder, "package.json"))
	if err != nil {
		return pkg
	}
	var manifest struct {
		Svelte           string            `json:"svelte"`
		Exports          json.RawMessage   `json:"exports"`
		Dependencies     map[string]string `json:"dependencies"`
		PeerDependencies map[string]string `json:"peerDependencies"`
	}
	if json.Unmarshal(data, &manifest) != nil {
		return pkg
	}
	pkg.components = manifest.Svelte != "" || bytes.Contains(manifest.Exports, []byte(`"svelte"`))
	ours := semver.MustParse(svelte.Version)
	for _, needs := range []string{manifest.PeerDependencies["svelte"], manifest.Dependencies["svelte"]} {
		if versions, err := semver.NewConstraint(needs); err == nil && !versions.Check(ours) {
			pkg.svelte = needs
		}
	}
	return pkg
}

// noLoader finds the message of the bundler about a file that is not code:
// what a package brings that cannot go into the bundle, such as native
// code, x.node, or WebAssembly.
var noLoader = regexp.MustCompile(`^No loader is configured for "[^"]+" files: (\S+)`)

// notCode explains a file of a package that the bundler cannot take in.
// Nothing when the message is of another kind.
func notCode(text string) string {
	m := noLoader.FindStringSubmatch(text)
	if m == nil {
		return ""
	}
	name, _ := project.PackageOf(m[1])
	if name == "" {
		return ""
	}
	return fmt.Sprintf("%s: the package %s brings a file that is not JavaScript, and the packages of the site are bundled with its code, where such a file cannot go: take a package written in JavaScript alone", text, name)
}
