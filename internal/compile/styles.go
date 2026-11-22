package compile

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/evanw/esbuild/pkg/api"

	"github.com/grimalschi/shugo/internal/project"
)

// style is one style sheet of the project: a .css file that some module
// imports, or the content of a <style> tag of a template.
type style struct {
	path string // of a file, or
	text string // the content of a tag
	from string // the template the tag is in
}

// What only Tailwind understands. A project that writes any of it has
// Tailwind build its style sheet.
var tailwind = regexp.MustCompile(`@import\s+["']tailwindcss|@tailwind\b|@apply\b|@theme\b|@utility\b|@plugin\b`)

type stylesheet struct {
	sheet          []byte
	assets, inputs []string
}

// styles builds the sheet and records its public assets and source inputs.
func (b *builder) styles(sheets []style) (stylesheet, error) {
	var out stylesheet
	if len(sheets) == 0 {
		return out, nil
	}
	// One sheet that imports all the others in their order. The content of
	// <style> tags is imported by number.
	var entry strings.Builder
	needsTailwind := false
	for i, sheet := range sheets {
		text := sheet.text
		if sheet.path != "" {
			data, err := os.ReadFile(sheet.path)
			if err != nil {
				return out, err
			}
			text = string(data)
			fmt.Fprintf(&entry, "@import %s;\n", strconv.Quote(filepath.ToSlash(sheet.path)))
		} else {
			fmt.Fprintf(&entry, "@import \"shugo-style:%d\";\n", i)
		}
		needsTailwind = needsTailwind || tailwind.MatchString(text)
	}
	if needsTailwind {
		var err error
		out.sheet, err = b.tailwind(sheets)
		return out, err
	}

	plugin := api.Plugin{Name: "shugo-styles", Setup: func(build api.PluginBuild) {
		build.OnResolve(api.OnResolveOptions{Filter: `^shugo-style:`}, func(args api.OnResolveArgs) (api.OnResolveResult, error) {
			return api.OnResolveResult{Path: args.Path, Namespace: "shugo-style"}, nil
		})
		build.OnLoad(api.OnLoadOptions{Filter: `.*`, Namespace: "shugo-style"}, func(args api.OnLoadArgs) (api.OnLoadResult, error) {
			i, _ := strconv.Atoi(strings.TrimPrefix(args.Path, "shugo-style:"))
			sheet := sheets[i]
			return api.OnLoadResult{Contents: &sheet.text, Loader: api.LoaderCSS, ResolveDir: filepath.Dir(b.project.Path(filepath.FromSlash(sheet.from)))}, nil
		})
	}}
	const entryFile = "shugo:styles.css"
	result := api.Build(api.BuildOptions{
		Stdin:            &api.StdinOptions{Contents: entry.String(), Loader: api.LoaderCSS, ResolveDir: b.project.Root, Sourcefile: entryFile},
		Bundle:           true,
		MinifyWhitespace: true,
		MinifySyntax:     true,
		Plugins:          []api.Plugin{plugin},
		AbsWorkingDir:    b.project.Root,
		Outfile:          "styles.css",
		Metafile:         true,
		Write:            false,
		LogLevel:         api.LogLevelSilent,
		LegalComments:    api.LegalCommentsNone,
		Loader:           assetLoaders,
		AssetNames:       "[name].[hash]",
		PublicPath:       strings.TrimSuffix(AssetsURL, "/"),
		Charset:          api.CharsetUTF8,
	})
	if len(result.Errors) > 0 {
		return out, b.diagnoseAll(result.Errors)
	}
	var meta struct{ Inputs map[string]json.RawMessage }
	if err := json.Unmarshal([]byte(result.Metafile), &meta); err != nil {
		return out, err
	}
	for name := range meta.Inputs {
		if name != entryFile && !strings.HasPrefix(name, "shugo-style:") {
			out.inputs = append(out.inputs, name)
		}
	}
	for _, file := range result.OutputFiles {
		if strings.HasSuffix(file.Path, ".css") {
			out.sheet = file.Contents
			continue
		}
		// A font or a picture the style sheets point at.
		name := filepath.Base(file.Path)
		if err := writeHashedFile(b.project.Path(project.Output, Assets, name), file.Contents); err != nil {
			return out, err
		}
		out.assets = append(out.assets, name)
	}
	return out, nil
}

// Files a style sheet may point at with url(…): they are copied next to it.
var assetLoaders = map[string]api.Loader{
	".woff": api.LoaderFile, ".woff2": api.LoaderFile, ".ttf": api.LoaderFile, ".otf": api.LoaderFile, ".eot": api.LoaderFile,
	".png": api.LoaderFile, ".jpg": api.LoaderFile, ".jpeg": api.LoaderFile, ".gif": api.LoaderFile, ".webp": api.LoaderFile,
	".avif": api.LoaderFile, ".svg": api.LoaderFile, ".ico": api.LoaderFile,
}

// tailwind has the Tailwind of the project build the style sheets. Tailwind
// finds the classes itself by reading the files of the project as text.
func (b *builder) tailwind(sheets []style) ([]byte, error) {
	cli := b.project.Path("node_modules", ".bin", "tailwindcss")
	if _, err := os.Stat(cli); err != nil {
		return nil, errors.New("the style sheets use Tailwind, and it is not installed in the project: run `npm install tailwindcss @tailwindcss/cli` (or `bun add`)")
	}
	work := b.project.Path(project.Output, "tmp")
	if err := os.MkdirAll(work, 0o755); err != nil {
		return nil, err
	}
	var entry strings.Builder
	for i, sheet := range sheets {
		path := sheet.path
		if path == "" {
			path = filepath.Join(work, fmt.Sprintf("style-%d.css", i))
			if err := os.WriteFile(path, []byte(sheet.text), 0o644); err != nil {
				return nil, err
			}
		}
		fmt.Fprintf(&entry, "@import %s;\n", strconv.Quote(filepath.ToSlash(path)))
	}
	input, output := filepath.Join(work, "tailwind.css"), filepath.Join(work, "tailwind.out.css")
	if err := os.WriteFile(input, []byte(entry.String()), 0o644); err != nil {
		return nil, err
	}
	command := exec.Command(b.project.Runtime(), cli, "--input", input, "--output", output, "--minify")
	command.Dir = b.project.Root
	if log, err := command.CombinedOutput(); err != nil {
		return nil, fmt.Errorf("Tailwind could not build the style sheets: %w\n%s", err, log)
	}
	return os.ReadFile(output)
}
