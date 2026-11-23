package test

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// The command of the package shugo runs the program of the system from its
// own package, with the arguments it was given and with where Svelte and
// svelte-check are, and ends as the program ends; without a package it
// needs it says how to install it. Run on Node and on Bun, whichever is
// there.
func TestPackageCommand(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("the program of the system is a shell script here")
	}
	// The system as Node names it: process.platform and process.arch.
	system := strings.NewReplacer("windows", "win32", "amd64", "x64").Replace(runtime.GOOS + "-" + runtime.GOARCH)
	command, err := os.ReadFile(filepath.Join("..", "npm", "shugo", "bin", "shugo.js"))
	if err != nil {
		t.Fatal(err)
	}
	// The command finds the packages itself, as in a project; the tests of
	// the repository name them for the program.
	var alone []string
	for _, variable := range os.Environ() {
		if !strings.HasPrefix(variable, "SHUGO_SVELTE") {
			alone = append(alone, variable)
		}
	}
	for _, runner := range []string{"node", "bun"} {
		if _, err := exec.LookPath(runner); err != nil {
			continue
		}
		t.Run(runner, func(t *testing.T) {
			root := t.TempDir()
			files := map[string]string{
				"node_modules/shugo/package.json": `{"name": "shugo", "type": "module"}`,
				"node_modules/shugo/bin/shugo.js": string(command),
			}
			write := func(files map[string]string) {
				for name, text := range files {
					path := filepath.Join(root, filepath.FromSlash(name))
					if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
						t.Fatal(err)
					}
					if err := os.WriteFile(path, []byte(text), 0o755); err != nil {
						t.Fatal(err)
					}
				}
			}
			write(files)
			run := exec.Command(runner, filepath.Join(root, "node_modules", "shugo", "bin", "shugo.js"), "build", "-check")
			run.Env = alone
			out, _ := run.CombinedOutput()
			if run.ProcessState.ExitCode() != 1 || !strings.Contains(string(out), "the package @shugo/"+system+", the shugo command for this system, is not installed") {
				t.Fatalf("without the program of the system: exit %d, %s", run.ProcessState.ExitCode(), out)
			}
			write(map[string]string{"node_modules/@shugo/" + system + "/bin/shugo": "#!/bin/sh\necho \"shugo $* $SHUGO_SVELTE\"\nexit 3\n"})
			run = exec.Command(runner, filepath.Join(root, "node_modules", "shugo", "bin", "shugo.js"), "build", "-check")
			run.Env = alone
			out, _ = run.CombinedOutput()
			if run.ProcessState.ExitCode() != 1 || !strings.Contains(string(out), "a dependency of the package shugo is not installed") {
				t.Fatalf("without Svelte: exit %d, %s", run.ProcessState.ExitCode(), out)
			}
			write(map[string]string{
				"node_modules/svelte/package.json":           `{"name": "svelte", "version": "5.59.1"}`,
				"node_modules/svelte-check/package.json":     `{"name": "svelte-check", "bin": "./bin/svelte-check"}`,
				"node_modules/svelte-check/bin/svelte-check": "",
			})
			real, err := filepath.EvalSymlinks(root)
			if err != nil {
				t.Fatal(err)
			}
			run = exec.Command(runner, filepath.Join(root, "node_modules", "shugo", "bin", "shugo.js"), "build", "-check")
			run.Env = alone
			out, _ = run.CombinedOutput()
			if want := "shugo build -check " + filepath.Join(real, "node_modules", "svelte"); run.ProcessState.ExitCode() != 3 || strings.TrimSpace(string(out)) != want {
				t.Fatalf("the program of the system: exit %d, %q, want %q", run.ProcessState.ExitCode(), out, want)
			}
		})
	}
}
