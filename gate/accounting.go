// relais, a realtime SFU backend for community self-hosters.
// Copyright (C) 2026 Nils Lehnen
//
// Licensed under the GNU Affero General Public License, version 3. See LICENSE
// for the full terms, including the warranty and liability disclaimer.

package main

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// A workflow is one file under .github/workflows/ and the name it declares. The
// name is what a check run is called on the pull request page, so it is the
// word a reader will be looking for when they compare this output against what
// the server said.
type workflow struct {
	File string
	Name string
}

// workflowsIn reads every workflow file in a directory. The accounting is
// derived from the directory rather than from a list in this program, which is
// the whole reason it is worth writing: a workflow added tomorrow appears in the
// not-run section on the next run, with nobody having to remember this file.
//
// A file with no top-level name is reported rather than skipped. A silent skip
// would take a gate out of the accounting, which is the failure this section
// exists to prevent, and it would do it in the direction that flatters the run.
func workflowsIn(dir string) ([]workflow, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("reading %s: %w", dir, err)
	}

	var found []workflow
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if !strings.HasSuffix(name, ".yml") && !strings.HasSuffix(name, ".yaml") {
			continue
		}
		source, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil {
			return nil, fmt.Errorf("reading %s: %w", name, err)
		}
		found = append(found, workflow{File: name, Name: declaredName(string(source), name)})
	}

	sort.Slice(found, func(i, j int) bool { return found[i].File < found[j].File })
	return found, nil
}

// declaredName is the workflow's own top-level name. It is the first line
// beginning at column zero with `name:`, because a job's name is indented under
// its job and this reading must not pick one of those up.
func declaredName(source, file string) string {
	for _, line := range strings.Split(strings.ReplaceAll(source, "\r\n", "\n"), "\n") {
		if !strings.HasPrefix(line, "name:") {
			continue
		}
		if value := strings.TrimSpace(strings.TrimPrefix(line, "name:")); value != "" {
			return strings.Trim(value, `"'`)
		}
	}
	return file + " (declares no top-level name)"
}

// accountFor sorts every workflow into the two things this run can honestly say
// about it: reproduced in part by one of the legs above, or not run here at all.
//
// Partly is the only honest word for the three it touches. This verb compiles
// what the build gate compiles, runs the suite the test gate runs and runs the
// linter the style gate runs, and each of those jobs does more than that around
// it. A run claiming to be those gates would be claiming the half it skipped.
func accountFor(legs []leg, found []workflow) (partly, notRun []string) {
	covered := map[string]string{}
	for _, l := range legs {
		covered[l.Workflow] = l.Covers
	}

	for _, w := range found {
		if reason, ok := covered[w.File]; ok {
			partly = append(partly, fmt.Sprintf("partly   %s (%s) - %s", w.Name, w.File, reason))
			continue
		}
		notRun = append(notRun, fmt.Sprintf("not run  %s (%s)", w.Name, w.File))
	}
	return partly, notRun
}

// toolchainFloor is the `go` line of go.mod, which is where every workflow in
// this tree gets its toolchain from through `go-version-file: go.mod`. Reading
// it here rather than naming a version keeps that one source of truth: a verb
// pinning its own would be a second one, and the first divergence between them
// would be invisible because both would be green.
func toolchainFloor(gomod string) string {
	for _, line := range strings.Split(strings.ReplaceAll(gomod, "\r\n", "\n"), "\n") {
		if strings.HasPrefix(line, "go ") {
			return strings.TrimSpace(strings.TrimPrefix(line, "go "))
		}
	}
	return ""
}

// pinnedLinter is the version the style gate installs, read out of the workflow
// that installs it. The same argument as above and the same failure if it were
// copied: the action pins the linter so that a rule nobody chose cannot turn an
// unrelated pull request red, and a local run at a different version is not the
// run the server will make.
//
// The value is matched on the `v` rather than on the key alone, because the same
// workflow carries `go-version-file` a few lines above and a reading that took
// the first key containing "version" would return a filename.
func pinnedLinter(workflow string) string {
	for _, line := range strings.Split(strings.ReplaceAll(workflow, "\r\n", "\n"), "\n") {
		value := strings.TrimSpace(line)
		if !strings.HasPrefix(value, "version:") {
			continue
		}
		value = strings.TrimSpace(strings.TrimPrefix(value, "version:"))
		if strings.HasPrefix(value, "v") {
			return value
		}
	}
	return ""
}
