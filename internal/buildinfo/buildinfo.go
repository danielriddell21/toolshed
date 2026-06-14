// Package buildinfo exposes the build version and a shared --version handler
// for toolshed's command binaries.
package buildinfo

import (
	"fmt"
	"os"
	"path/filepath"
)

// Version is the build version, set at release time via
// -ldflags "-X github.com/danielriddell21/toolshed/internal/buildinfo.Version=...".
// It defaults to "dev" for local builds.
var Version = "dev"

// HandleVersionFlag prints "<command> <version>" and exits 0 when the first CLI
// argument is a version flag. The command name is taken from the invoked binary
// (so it stays correct even when installed under a different name). Call it at
// the very top of main, before any TUI starts.
func HandleVersionFlag() {
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "--version", "-v", "version":
			fmt.Println(filepath.Base(os.Args[0]), Version)
			os.Exit(0)
		}
	}
}
