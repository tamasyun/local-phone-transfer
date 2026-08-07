//go:build windows

package main

import (
	"os"
	"path/filepath"
	"strings"
)

func init() {
	if strings.TrimSpace(os.Getenv("LOCAL_PHONE_TRANSFER_LAUNCHED")) == "1" {
		return
	}

	// `go test` executes a temporary *.test.exe outside the repository tree.
	// It must not be treated as a user directly launching the production EXE.
	if exe, err := os.Executable(); err == nil {
		if strings.HasSuffix(strings.ToLower(filepath.Base(exe)), ".test.exe") {
			return
		}

		// Keep repository development builds usable. Public release packages do
		// not contain a .git directory and must be started through the launcher.
		dir := filepath.Dir(exe)
		if _, err := os.Stat(filepath.Join(dir, ".git")); err == nil {
			return
		}
	}

	fatalDialog("Offline File Transfer", "This is an internal application file.\n\nStart Offline File Transfer with Start.cmd in the extracted folder.")
	os.Exit(1)
}
