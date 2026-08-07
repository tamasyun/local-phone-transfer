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

	// Keep repository development builds usable. Public release packages do not
	// contain a .git directory and must be started through the installed launcher.
	exe, err := os.Executable()
	if err == nil {
		dir := filepath.Dir(exe)
		if _, err := os.Stat(filepath.Join(dir, ".git")); err == nil {
			return
		}
	}

	fatalDialog("Offline File Transfer", "This is an internal application file.\n\nRun Setup.cmd first, then start the application from the desktop shortcut.")
	os.Exit(1)
}
