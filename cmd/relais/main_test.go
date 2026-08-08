// relais, a realtime SFU backend for community self-hosters.
// Copyright (C) 2026 Nils Lehnen
//
// Licensed under the GNU Affero General Public License, version 3. See LICENSE
// for the full terms, including the warranty and liability disclaimer.

package main

import (
	"errors"
	"runtime/debug"
	"strings"
	"testing"
)

// One of the six requirements the language was chosen against is a test story
// strong enough that behaviour runs against a fake rather than against the real
// world. These tests are the smallest honest demonstration of it on the only
// behaviour that exists: the build information is supplied by the test, the
// destination is supplied by the test, and no process is launched, no socket is
// opened and no clock is read.

func TestVersionReportsRevisionAndToolchain(t *testing.T) {
	got := version(&debug.BuildInfo{
		Main:      debug.Module{Version: "(devel)"},
		GoVersion: "go1.24.0",
		Settings: []debug.BuildSetting{
			{Key: "vcs.revision", Value: "0123456789abcdef"},
			{Key: "vcs.modified", Value: "false"},
		},
	}, true)

	want := "relais (devel), revision 0123456789abcdef, toolchain go1.24.0"
	if got != want {
		t.Errorf("version() = %q, want %q", got, want)
	}
}

func TestVersionSaysWhenTheTreeWasModified(t *testing.T) {
	got := version(&debug.BuildInfo{
		Main:      debug.Module{Version: "(devel)"},
		GoVersion: "go1.24.0",
		Settings: []debug.BuildSetting{
			{Key: "vcs.revision", Value: "0123456789abcdef"},
			{Key: "vcs.modified", Value: "true"},
		},
	}, true)

	if !strings.Contains(got, "tree modified") {
		t.Errorf("version() = %q, want it to report the modified tree", got)
	}
}

// A binary built outside a version control checkout carries no revision. It has
// to say that rather than print a revision it does not have, because a version
// string that looks complete is worse than one that admits what is missing.
func TestVersionAdmitsMissingBuildInformation(t *testing.T) {
	got := version(nil, false)

	if !strings.Contains(got, "without build information") {
		t.Errorf("version() = %q, want it to admit the absence", got)
	}
	if strings.Contains(got, "revision") {
		t.Errorf("version() = %q, want no revision claimed", got)
	}
}

func TestVersionHandlesBuildInformationWithNoRevision(t *testing.T) {
	got := version(&debug.BuildInfo{
		Main:      debug.Module{Version: "(devel)"},
		GoVersion: "go1.24.0",
	}, true)

	if !strings.Contains(got, "revision unknown") {
		t.Errorf("version() = %q, want the revision reported as unknown", got)
	}
}

// failingWriter refuses every write, which is what a closed or full destination
// looks like from here.
type failingWriter struct{}

func (failingWriter) Write([]byte) (int, error) {
	return 0, errors.New("destination refused the write")
}

func TestRunReportsAFailedWrite(t *testing.T) {
	if code := run(failingWriter{}); code == 0 {
		t.Error("run() = 0 on a destination that refused the write, want a non-zero code")
	}
}

func TestRunWritesTheVersionAndSucceeds(t *testing.T) {
	var out strings.Builder

	if code := run(&out); code != 0 {
		t.Errorf("run() = %d, want 0", code)
	}
	if !strings.HasPrefix(out.String(), "relais ") {
		t.Errorf("run() wrote %q, want it to start with the program name", out.String())
	}
	if !strings.HasSuffix(out.String(), "\n") {
		t.Errorf("run() wrote %q, want a terminating newline", out.String())
	}
}
