// Command relais is the backend this repository builds.
//
// It does one thing today: it reports what the build knows about itself and
// exits. That is deliberate. The language decision is worth no more than a
// binary that actually builds from a clean checkout, so this file exists to make
// the decision checkable rather than asserted.
//
// It sits at the module root on purpose, which is the absence of a layout
// decision rather than one. Where the entry point finally belongs is settled
// separately, and nothing here should be read as having settled it.
package main

import (
	"fmt"
	"io"
	"os"
	"runtime/debug"
)

// version renders what the toolchain recorded about this build.
//
// Nothing is injected at link time. The revision comes from the build
// information the toolchain embeds when it builds from a version control
// checkout, so the one documented build command produces a binary that can name
// its own source without a second flag to remember. A build that carries no such
// information says so rather than reporting a version it does not have.
func version(bi *debug.BuildInfo, ok bool) string {
	if !ok || bi == nil {
		return "relais unknown, built without build information"
	}

	revision := "unknown"
	modified := ""
	for _, setting := range bi.Settings {
		switch setting.Key {
		case "vcs.revision":
			revision = setting.Value
		case "vcs.modified":
			if setting.Value == "true" {
				modified = ", tree modified"
			}
		}
	}

	return fmt.Sprintf("relais %s, revision %s%s, toolchain %s",
		bi.Main.Version, revision, modified, bi.GoVersion)
}

// run holds the behaviour so a test can reach it without launching a process.
// The destination is a parameter for the same reason.
//
// A failed write is reported rather than ignored. This program's whole output is
// that one line, so a write that did not happen means it did nothing, and exiting
// zero would tell a caller otherwise. There is nowhere to report the failure to,
// since the place to report it is what just failed, so the exit code is the whole
// of the signal.
func run(out io.Writer) int {
	bi, ok := debug.ReadBuildInfo()
	if _, err := fmt.Fprintln(out, version(bi, ok)); err != nil {
		return 1
	}
	return 0
}

func main() {
	os.Exit(run(os.Stdout))
}
