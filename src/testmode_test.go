package main

import (
	"strings"
	"testing"
)

func TestListenAddressTestModeIsLoopbackOnly(t *testing.T) {
	got := listenAddress(8765, true)
	if got != "127.0.0.1:8765" {
		t.Fatalf("test mode must bind only to loopback, got %q", got)
	}
}

func TestListenAddressProductionUsesConfiguredPort(t *testing.T) {
	got := listenAddress(8765, false)
	if got != ":8765" {
		t.Fatalf("production listen address changed unexpectedly, got %q", got)
	}
}

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
