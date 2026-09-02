// relais, a realtime SFU backend for community self-hosters.
// Copyright (C) 2026 Nils Lehnen
//
// Licensed under the GNU Affero General Public License, version 3. See LICENSE
// for the full terms, including the warranty and liability disclaimer.

// Command gate runs the legs a pull request is judged by whose result is the
// same on every machine, and names the ones it did not run.
//
//	go run ./gate
//
// Three legs, in order, stopping at the first failure: the build, the suite and
// the style gate. The contribution guide named one command before this existed
// and that command does not compile a test file, so a contributor who followed
// the guide exactly got a green result from a run that had never built the
// suite, and found out on the pull request page.
//
// The second half is what makes this a verb rather than three commands in a
// paragraph. A local run that says nothing about what it skipped is read as a
// green pull request, so this one ends by naming every workflow it did not run,
// derived from .github/workflows/ rather than from a list kept here. A gate
// added tomorrow appears in that section by itself.
//
// One leg cannot move and its absence is not an omission. The test gate asserts,
// before it runs anything, that the runner has no display, no audio device, is
// not uid 0 and holds no capabilities. Those are properties of a Linux runner,
// read out of /proc and the device tree. A machine somebody works at has a
// display by definition and a machine that is not Linux has no /proc, so a local
// command asserting the same thing would refuse every developer machine it ran
// on. The assertion stays where it is, because the property it protects is that
// the suite runs on a clean runner, and the runner is where that is true.
package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// A leg is one command this verb runs and the workflow it stands in for.
//
// Covers is the sentence printed beside that workflow in the accounting. It says
// which part of the job was reproduced, so the line cannot be read as the job
// having run.
type leg struct {
	Name     string
	Command  []string
	Workflow string
	Covers   string
}

// legs is the whole of what this verb runs, in the order it runs them.
//
// The order is not arbitrary. A suite cannot run against a tree that does not
// compile and a linter's report on one is noise, so the build is first and the
// run stops at the first failure rather than printing three failures with one
// cause.
var legs = []leg{
	{
		Name:     "build",
		Command:  []string{"go", "build", "./..."},
		Workflow: "build.yml",
		Covers:   "the compile, and nothing that job does around it",
	},
	{
		Name:     "suite",
		Command:  []string{"go", "test", "./..."},
		Workflow: "test.yml",
		Covers:   "the suite; the headless assertions belong to the runner, see below",
	},
	{
		Name:     "style",
		Command:  []string{"golangci-lint", "run"},
		Workflow: "lint.yml",
		Covers:   "the same command with the same .golangci.yml, at the version below",
	},
}

func main() {
	root, err := repositoryRoot()
	if err != nil {
		fmt.Fprintln(os.Stderr, "gate:", err)
		os.Exit(1)
	}

	fmt.Printf("gate: %d legs, in order, stopping at the first failure.\n\n", len(legs))

	for i, l := range legs {
		fmt.Printf("  %-6s %s\n", l.Name, strings.Join(l.Command, " "))

		cmd := exec.Command(l.Command[0], l.Command[1:]...)
		cmd.Dir = root
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr

		if err := cmd.Run(); err != nil {
			fmt.Printf("\n  %-6s FAILED: %v\n", l.Name, err)
			for _, skipped := range legs[i+1:] {
				fmt.Printf("  %-6s not run, because %s failed before it\n", skipped.Name, l.Name)
			}
			fmt.Println()
			report(root)
			os.Exit(1)
		}
		fmt.Printf("  %-6s ok\n\n", l.Name)
	}

	report(root)
}

// report prints what this run did not judge. It runs on the way out of a red run
// as well as a green one, because a failed run is exactly when somebody is most
// likely to read the legs that did pass as the whole set.
func report(root string) {
	fmt.Println("What this run did not judge, derived from .github/workflows/ rather than listed here:")
	fmt.Println()

	found, err := workflowsIn(filepath.Join(root, ".github", "workflows"))
	if err != nil {
		fmt.Println("  the workflow directory could not be read, so this accounting is missing:", err)
		return
	}

	partly, notRun := accountFor(legs, found)
	for _, line := range append(partly, notRun...) {
		fmt.Println("  " + line)
	}

	fmt.Println()
	fmt.Println("  The headless assertions in .github/workflows/test.yml read the display variables,")
	fmt.Println("  the audio device nodes and /proc of a Linux runner. Running them here would refuse")
	fmt.Println("  every machine somebody works at, so they are the runner's and this verb does not")
	fmt.Println("  imitate them.")
	fmt.Println()

	fmt.Println("Versions, read from the files the workflows read them from:")
	fmt.Println()
	fmt.Println("  " + toolchainLine(root))
	fmt.Println("  " + linterLine(root))
	fmt.Println()
	fmt.Println("The pull request page is the authority for what ran on the server and what it said.")
}

// toolchainLine states the floor go.mod declares beside the toolchain that is
// actually here, rather than asserting the two agree.
func toolchainLine(root string) string {
	source, err := os.ReadFile(filepath.Join(root, "go.mod"))
	if err != nil {
		return fmt.Sprintf("go.mod could not be read, so the toolchain floor is unknown: %v", err)
	}
	floor := toolchainFloor(string(source))
	if floor == "" {
		return "go.mod declares no go line, so there is no floor to compare against"
	}
	return fmt.Sprintf("go.mod declares go %s; this toolchain is %s", floor, firstLineOf("go", "version"))
}

// linterLine states the pinned version beside the installed one. A difference is
// printed rather than refused: the two runs can disagree for a rule nobody
// chose, and saying so is what a reader needs before treating a green style leg
// as a green Lint gate.
func linterLine(root string) string {
	source, err := os.ReadFile(filepath.Join(root, ".github", "workflows", "lint.yml"))
	if err != nil {
		return fmt.Sprintf("lint.yml could not be read, so the pinned linter version is unknown: %v", err)
	}
	pinned := pinnedLinter(string(source))
	if pinned == "" {
		return "lint.yml pins no linter version, so there is nothing to compare against"
	}

	here := firstLineOf("golangci-lint", "version")
	line := fmt.Sprintf(".github/workflows/lint.yml pins golangci-lint %s; here: %s", pinned, here)
	if !strings.Contains(here, strings.TrimPrefix(pinned, "v")) {
		line += "\n  Those differ, so a green style leg here is not by itself a green Lint gate there."
	}
	return line
}

// firstLineOf runs a version command and returns one line of whatever it said,
// including its failure. A missing tool is a fact about this machine that the
// accounting states rather than hides behind an empty string.
func firstLineOf(name string, args ...string) string {
	out, err := exec.Command(name, args...).CombinedOutput()
	if err != nil && len(out) == 0 {
		return fmt.Sprintf("not readable here (%v)", err)
	}
	return strings.TrimSpace(strings.SplitN(strings.TrimSpace(string(out)), "\n", 2)[0])
}

// repositoryRoot walks up from the working directory to the module root, so the
// verb judges the tree it is in rather than whichever directory it was called
// from.
func repositoryRoot() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("no go.mod above the working directory, so there is no tree to judge")
		}
		dir = parent
	}
}
