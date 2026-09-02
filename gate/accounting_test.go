// relais, a realtime SFU backend for community self-hosters.
// Copyright (C) 2026 Nils Lehnen
//
// Licensed under the GNU Affero General Public License, version 3. See LICENSE
// for the full terms, including the warranty and liability disclaimer.

package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// writeWorkflows lays a workflow directory out in a temporary place, so every
// case below states its own input rather than depending on what the real
// .github/workflows/ happens to hold on the day it runs.
func writeWorkflows(t *testing.T, files map[string]string) string {
	t.Helper()

	dir := t.TempDir()
	for name, source := range files {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(source), 0o600); err != nil {
			t.Fatalf("writing %s: %v", name, err)
		}
	}
	return dir
}

// The accounting is what makes this a verb rather than three commands in a
// paragraph, and its whole value is that a gate nobody told it about still turns
// up. This is the case that says so: two of the four workflows are named by a
// leg and the other two are not, and all four appear.
func TestEveryWorkflowAppearsAsPartlyRunOrNotRun(t *testing.T) {
	dir := writeWorkflows(t, map[string]string{
		"build.yml":     "name: Build\n",
		"lint.yml":      "name: Lint\n",
		"codeql.yml":    "name: Code scanning\n",
		"scorecard.yml": "name: Scorecard supply-chain security\n",
	})

	found, err := workflowsIn(dir)
	if err != nil {
		t.Fatalf("reading the workflow directory: %v", err)
	}
	if len(found) != 4 {
		t.Fatalf("read %d workflows, want 4: %v", len(found), found)
	}

	table := []leg{
		{Name: "build", Workflow: "build.yml", Covers: "the compile"},
		{Name: "style", Workflow: "lint.yml", Covers: "the same command"},
	}
	partly, notRun := accountFor(table, found)

	if len(partly) != 2 {
		t.Errorf("%d workflows reported as partly run, want 2: %v", len(partly), partly)
	}
	if len(notRun) != 2 {
		t.Errorf("%d workflows reported as not run, want 2: %v", len(notRun), notRun)
	}

	whole := strings.Join(append(partly, notRun...), "\n")
	for _, want := range []string{"Build", "Lint", "Code scanning", "Scorecard supply-chain security"} {
		if !strings.Contains(whole, want) {
			t.Errorf("the accounting does not mention %q, so that gate is missing from it:\n%s", want, whole)
		}
	}
	if !strings.Contains(whole, "not run  Code scanning (codeql.yml)") {
		t.Errorf("a workflow no leg names is not reported as not run:\n%s", whole)
	}
}

// The near miss the shape above exists against. A leg naming a workflow file
// that is not in the directory must not take a workflow out of the accounting,
// because that is the mistake a rename makes and it fails in the direction that
// flatters the run.
func TestALegNamingAWorkflowThatIsNotThereRemovesNothing(t *testing.T) {
	dir := writeWorkflows(t, map[string]string{"test.yml": "name: Test\n"})

	found, err := workflowsIn(dir)
	if err != nil {
		t.Fatalf("reading the workflow directory: %v", err)
	}

	partly, notRun := accountFor([]leg{{Name: "suite", Workflow: "tests.yml", Covers: "the suite"}}, found)
	if len(partly) != 0 {
		t.Errorf("a leg naming a file that is not there claimed to cover something: %v", partly)
	}
	if len(notRun) != 1 || !strings.Contains(notRun[0], "Test (test.yml)") {
		t.Errorf("the workflow that is there is not reported as not run: %v", notRun)
	}
}

// A file with no top-level name is named by its filename rather than skipped. A
// silent skip is the same failure as the one above with a different cause.
func TestAWorkflowWithNoTopLevelNameIsStillAccountedFor(t *testing.T) {
	dir := writeWorkflows(t, map[string]string{"nameless.yml": "on:\n  push:\n"})

	found, err := workflowsIn(dir)
	if err != nil {
		t.Fatalf("reading the workflow directory: %v", err)
	}
	if len(found) != 1 {
		t.Fatalf("read %d workflows, want 1", len(found))
	}
	if !strings.Contains(found[0].Name, "nameless.yml") {
		t.Errorf("a workflow with no top-level name is reported as %q, which names nothing a reader can find", found[0].Name)
	}
}

// The name read is the workflow's own, not a job's. Every workflow in this tree
// carries an indented `name:` under its job as well, so a reading that took the
// first line containing the key would report a job name as the gate's name and
// a reader comparing it against the pull request page would find neither.
func TestTheNameReadIsTheWorkflowsRatherThanAJobs(t *testing.T) {
	source := "on:\n  pull_request:\n\njobs:\n  analyse:\n    name: CodeQL\n\nname: Code scanning\n"

	if got := declaredName(source, "codeql.yml"); got != "Code scanning" {
		t.Errorf("declaredName returned %q, want the top-level name %q", got, "Code scanning")
	}
}

// The toolchain floor comes out of go.mod, which is the file every workflow here
// reads it from through `go-version-file: go.mod`.
func TestTheToolchainFloorIsTheGoLineOfGoMod(t *testing.T) {
	if got := toolchainFloor("module example\n\ngo 1.24.0\n"); got != "1.24.0" {
		t.Errorf("toolchainFloor returned %q, want %q", got, "1.24.0")
	}

	// The near miss. A go.mod may carry a `toolchain` line, whose value is
	// spelled `go1.26.0` and is a different statement: the floor the module
	// requires against the toolchain a build should fetch. A reading that
	// matched the substring rather than the line would return that one and
	// the printed floor would be wrong in the direction nobody checks.
	if got := toolchainFloor("module example\n\ntoolchain go1.26.0\n\ngo 1.24.0\n"); got != "1.24.0" {
		t.Errorf("toolchainFloor returned %q with a toolchain line present, want %q", got, "1.24.0")
	}
}

// The linter version comes out of the workflow that installs it, for the same
// reason: a version pinned twice is a second source of truth, and the first
// divergence is invisible because both copies are green.
func TestThePinnedLinterVersionIsNotTheToolchainFile(t *testing.T) {
	source := strings.Join([]string{
		"      - name: Set up the toolchain",
		"        uses: actions/setup-go@abc",
		"        with:",
		"          go-version-file: go.mod",
		"          check-latest: false",
		"      - name: Run formatting, vetting and linting",
		"        uses: golangci/golangci-lint-action@def",
		"        with:",
		"          version: v2.12.2",
	}, "\n")

	if got := pinnedLinter(source); got != "v2.12.2" {
		t.Errorf("pinnedLinter returned %q, want %q", got, "v2.12.2")
	}
}

// The two readers above are proved against fixtures, which say nothing about
// whether they find anything in this repository. This one is about the tree: it
// asserts that both files are still shaped the way the readers expect, so a
// reshuffle of either turns up here rather than as an empty line in the output.
func TestBothVersionsAreReadableOutOfThisTree(t *testing.T) {
	root, err := repositoryRoot()
	if err != nil {
		t.Fatalf("finding the module root: %v", err)
	}

	gomod, err := os.ReadFile(filepath.Join(root, "go.mod"))
	if err != nil {
		t.Fatalf("reading go.mod: %v", err)
	}
	if toolchainFloor(string(gomod)) == "" {
		t.Error("go.mod carries no line this verb can read a toolchain floor from")
	}

	lint, err := os.ReadFile(filepath.Join(root, ".github", "workflows", "lint.yml"))
	if err != nil {
		t.Fatalf("reading lint.yml: %v", err)
	}
	if pinnedLinter(string(lint)) == "" {
		t.Error("lint.yml carries no line this verb can read a pinned linter version from")
	}
}
