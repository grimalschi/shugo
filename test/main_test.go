package test

import (
	"os"
	"path/filepath"
	"testing"
)

// TestMain runs the tests with the packages svelte and svelte-check of the
// repository, which bun install puts into its node_modules: the projects of the tests lie in
// folders of their own, around which Shugo would not find it.
func TestMain(m *testing.M) {
	if os.Getenv("SHUGO_SVELTE") == "" {
		if dir, err := filepath.Abs(filepath.Join("..", "node_modules", "svelte")); err == nil {
			os.Setenv("SHUGO_SVELTE", dir)
		}
	}
	if os.Getenv("SHUGO_SVELTE_CHECK") == "" {
		if program, err := filepath.Abs(filepath.Join("..", "node_modules", "svelte-check", "bin", "svelte-check")); err == nil {
			os.Setenv("SHUGO_SVELTE_CHECK", program)
		}
	}
	os.Exit(m.Run())
}
