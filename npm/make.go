//go:build ignore

// make writes the packages of npm that a project installs Shugo with, as
// esbuild ships its own: shugo, with the command, the types of the module
// shugo and the settings of TypeScript; and @shugo/<system>-<cpu>, the
// program of one system, which the package manager installs only where it
// fits. From the root of the repository:
//
//	go run npm/make.go -version 0.1.0
//	go run npm/make.go -version 0.1.0 -only linux-x64
//
// Each package is a folder of npm/dist/, ready for npm publish.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
)

// systems are the systems Shugo is made for: their names in npm, as
// process.platform and process.arch say them, and in Go.
var systems = []struct{ npm, goos, goarch string }{
	{"darwin-arm64", "darwin", "arm64"},
	{"darwin-x64", "darwin", "amd64"},
	{"linux-arm64", "linux", "arm64"},
	{"linux-x64", "linux", "amd64"},
	{"win32-arm64", "windows", "arm64"},
	{"win32-x64", "windows", "amd64"},
}

// The notices of what the program carries inside: Svelte and the code it is
// made of.
var notices = []string{"internal/svelte/vendor/SVELTE-LICENSE.md", "internal/svelte/vendor/THIRD-PARTY.md"}

func main() {
	version := flag.String("version", "", "the version of the packages, such as 0.1.0")
	only := flag.String("only", "", "the systems to make, such as linux-x64,darwin-arm64; all if empty")
	flag.Parse()
	if !regexp.MustCompile(`^\d+\.\d+\.\d+(-[0-9A-Za-z.-]+)?$`).MatchString(*version) {
		fail(fmt.Errorf("-version takes a version of npm, such as 0.1.0, got %q", *version))
	}
	dist := filepath.Join("npm", "dist")
	if err := os.RemoveAll(dist); err != nil {
		fail(err)
	}
	if err := mainPackage(filepath.Join(dist, "shugo"), *version); err != nil {
		fail(err)
	}
	for _, system := range systems {
		if *only != "" && !contains(strings.Split(*only, ","), system.npm) {
			continue
		}
		if err := systemPackage(filepath.Join(dist, "@shugo", system.npm), *version, system.npm, system.goos, system.goarch); err != nil {
			fail(err)
		}
	}
}

// mainPackage writes the package shugo: npm/shugo, its version set. The
// version stands as 0.0.0 there, for the package and for the programs of
// the systems it depends on.
func mainPackage(out, version string) error {
	return filepath.WalkDir(filepath.Join("npm", "shugo"), func(path string, entry os.DirEntry, err error) error {
		if err != nil || entry.IsDir() {
			return err
		}
		rel, err := filepath.Rel(filepath.Join("npm", "shugo"), path)
		if err != nil {
			return err
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		if rel == "package.json" {
			data = []byte(strings.ReplaceAll(string(data), `"0.0.0"`, fmt.Sprintf("%q", version)))
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		return write(filepath.Join(out, rel), data, info.Mode().Perm())
	})
}

// systemPackage writes the package of the program of one system.
func systemPackage(out, version, system, goos, goarch string) error {
	program := filepath.Join(out, "bin", "shugo")
	if goos == "windows" {
		program += ".exe"
	}
	build := exec.Command("go", "build", "-trimpath", "-ldflags=-s -w", "-o", program, "./cmd/shugo")
	build.Env = append(os.Environ(), "GOOS="+goos, "GOARCH="+goarch, "CGO_ENABLED=0")
	build.Stdout, build.Stderr = os.Stdout, os.Stderr
	if err := build.Run(); err != nil {
		return fmt.Errorf("building shugo for %s: %w", system, err)
	}
	cpu := strings.SplitN(system, "-", 2)
	manifest, err := json.MarshalIndent(struct {
		Name            string   `json:"name"`
		Version         string   `json:"version"`
		Description     string   `json:"description"`
		License         string   `json:"license"`
		OS              []string `json:"os"`
		CPU             []string `json:"cpu"`
		Files           []string `json:"files"`
		PreferUnplugged bool     `json:"preferUnplugged"`
	}{
		Name: "@shugo/" + system, Version: version,
		Description: "The shugo command for " + system + ": the package shugo runs it",
		License:     "UNLICENSED", OS: cpu[:1], CPU: cpu[1:],
		Files: []string{"bin", "SVELTE-LICENSE.md", "THIRD-PARTY.md"}, PreferUnplugged: true,
	}, "", "  ")
	if err != nil {
		return err
	}
	if err := write(filepath.Join(out, "package.json"), append(manifest, '\n'), 0o644); err != nil {
		return err
	}
	for _, notice := range notices {
		data, err := os.ReadFile(filepath.FromSlash(notice))
		if err != nil {
			return err
		}
		if err := write(filepath.Join(out, filepath.Base(notice)), data, 0o644); err != nil {
			return err
		}
	}
	info, err := os.Stat(program)
	if err != nil {
		return err
	}
	fmt.Printf("%9d %s\n", info.Size(), program)
	return nil
}

func write(path string, data []byte, mode os.FileMode) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, data, mode)
}

func contains(list []string, item string) bool {
	for _, one := range list {
		if strings.TrimSpace(one) == item {
			return true
		}
	}
	return false
}

func fail(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}
