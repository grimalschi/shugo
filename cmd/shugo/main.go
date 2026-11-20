// Command shugo builds and shows static sites.
package main

import (
	"os"

	"github.com/grimalschi/shugo/internal/cli"
)

func main() {
	os.Exit(cli.Main(os.Args[1:], os.Stdout, os.Stderr))
}
