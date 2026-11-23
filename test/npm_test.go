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
// own package, with the arguments it was given, and ends as the program
// ends; without that package it says how to install it. Run on Node and on
// Bun, whichever is there.
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
			for name, text := range files {
				path := filepath.Join(root, filepath.FromSlash(name))
				if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(path, []byte(text), 0o644); err != nil {
					t.Fatal(err)
				}
			}
			run := exec.Command(runner, filepath.Join(root, "node_modules", "shugo", "bin", "shugo.js"), "build", "-check")
			out, _ := run.CombinedOutput()
			if run.ProcessState.ExitCode() != 1 || !strings.Contains(string(out), "the package @shugo/"+system+", the shugo command for this system, is not installed") {
				t.Fatalf("without the program of the system: exit %d, %s", run.ProcessState.ExitCode(), out)
			}
			program := filepath.Join(root, "node_modules", "@shugo", system, "bin", "shugo")
			if err := os.MkdirAll(filepath.Dir(program), 0o755); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(program, []byte("#!/bin/sh\necho \"shugo $*\"\nexit 3\n"), 0o755); err != nil {
				t.Fatal(err)
			}
			run = exec.Command(runner, filepath.Join(root, "node_modules", "shugo", "bin", "shugo.js"), "build", "-check")
			out, _ = run.CombinedOutput()
			if run.ProcessState.ExitCode() != 3 || strings.TrimSpace(string(out)) != "shugo build -check" {
				t.Fatalf("the program of the system: exit %d, %q", run.ProcessState.ExitCode(), out)
			}
		})
	}
}
