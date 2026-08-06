package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNetworkURLsTestModeNeverExposeLANAddress(t *testing.T) {
	a := &App{cfg: Config{Port: 8765}, phoneToken: "token", testMode: true}
	urls := a.networkURLs()
	if len(urls) != 1 {
		t.Fatalf("expected one loopback URL, got %d", len(urls))
	}
	if urls[0].IP != "127.0.0.1" || !strings.HasPrefix(urls[0].URL, "http://127.0.0.1:8765/") {
		t.Fatalf("test mode exposed a non-loopback URL: %+v", urls[0])
	}
}

func TestProductionURLUsesOnlyTransferIP(t *testing.T) {
	a := &App{cfg: Config{Port: 8765}, phoneToken: "token", transferIP: "192.168.137.1"}
	urls := a.networkURLs()
	if len(urls) != 1 || urls[0].IP != "192.168.137.1" {
		t.Fatalf("production URL must use only transfer IP: %+v", urls)
	}
}

func TestCleanupUploadTemps(t *testing.T) {
	dir := t.TempDir()
	tmp := filepath.Join(dir, ".upload-abcd")
	keep := filepath.Join(dir, "complete.pdf")
	if err := os.WriteFile(tmp, []byte("partial"), 0600); err != nil { t.Fatal(err) }
	if err := os.WriteFile(keep, []byte("ok"), 0600); err != nil { t.Fatal(err) }
	if err := cleanupUploadTemps(dir); err != nil { t.Fatal(err) }
	if _, err := os.Stat(tmp); !os.IsNotExist(err) { t.Fatalf("temporary upload was not removed") }
	if _, err := os.Stat(keep); err != nil { t.Fatalf("completed file was removed: %v", err) }
}
