package cli

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/grimalschi/shugo/internal/svelte"
)

// checkTypes checks the types of the project, and the props of its
// components: svelte-check does it, with the TypeScript of the package
// shugo, by tsconfig.json of the project. What it says goes out as it is.
func (c *command) checkTypes() error {
	program := os.Getenv("SHUGO_SVELTE_CHECK")
	if program == "" {
		if dir := svelte.Installed(c.project.Root, "svelte-check"); dir != "" {
			program = filepath.Join(dir, "bin", "svelte-check")
		}
	}
	if program == "" {
		return errors.New("shugo check runs svelte-check, which comes with the package shugo, and it is not installed: bun add shugo")
	}
	if _, err := os.Stat(c.project.Path("tsconfig.json")); err != nil {
		return errors.New(`shugo check reads the settings of TypeScript from tsconfig.json of the project, and there is none: write { "extends": "shugo/tsconfig.json" } into it`)
	}
	run := exec.Command(c.project.Runtime(), program, "--tsconfig", "./tsconfig.json", "--output", "human")
	run.Dir = c.project.Root
	run.Stdout, run.Stderr = c.stdout, c.stderr
	err := run.Run()
	var exit *exec.ExitError
	if errors.As(err, &exit) {
		return errors.New("svelte-check found mistakes in the project")
	}
	return err
}
