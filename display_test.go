package main

import (
	"os"
	"testing"
)

// This file is deliberate and temporary. It exists for one run, to show that a
// test which needs a display fails on the runner this suite is required to pass
// on. The commit after that run removes it.
//
// It is written the way such a test arrives in real life: not as an attempt to
// break the gate, but as a test that reaches for something the machine it was
// written on happened to have.
func TestNeedsADisplay(t *testing.T) {
	if os.Getenv("DISPLAY") == "" {
		t.Fatal("no display available; this test cannot run headless")
	}
}
