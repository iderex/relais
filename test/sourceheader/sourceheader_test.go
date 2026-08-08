// relais, a realtime SFU backend for community self-hosters.
// Copyright (C) 2026 Nils Lehnen
//
// Licensed under the GNU Affero General Public License, version 3. See LICENSE
// for the full terms, including the warranty and liability disclaimer.

package sourceheader_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/iderex/relais/test/sourceheader"
)

// TestEveryGoFileCarriesTheNotice is the guard itself. It is the check the
// licence issue asks for: a file added without the notice reds here.
func TestEveryGoFileCarriesTheNotice(t *testing.T) {
	root, err := sourceheader.ModuleRoot()
	if err != nil {
		t.Fatalf("finding the module root: %v", err)
	}

	files, err := sourceheader.GoFiles(root)
	if err != nil {
		t.Fatalf("reading the tree at %s: %v", root, err)
	}

	// A pass over nothing is not a pass. A walk that had silently stopped
	// finding Go files would report a tree in which every file carries the
	// notice, in exactly the words of one that does.
	t.Logf("%d Go files under %s", len(files), root)
	const anchor = "cmd/relais/main.go"
	if !carries(files, anchor) {
		t.Fatalf("%s was not among the files read, so this run judged a tree it did not find; "+
			"if that file has legitimately gone, move this anchor to one that exists rather than deleting it", anchor)
	}

	for _, file := range files {
		source, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(file)))
		if err != nil {
			t.Errorf("reading %s: %v", file, err)
			continue
		}
		if refusal := sourceheader.Refuse(file, string(source)); refusal != "" {
			t.Error(refusal)
		}
	}
}

// TestTheNoticeSaysWhatTheLicenceAsksFor holds the notice to the three things
// the licence's own instructions ask a file to carry. It is what stops the
// constant being quietly shortened into a marker nobody can act on: a line
// saying only that a file is licensed names no licence, no holder and no place
// to read the terms.
func TestTheNoticeSaysWhatTheLicenceAsksFor(t *testing.T) {
	required := []struct {
		what string
		text string
	}{
		{"the program it belongs to", "relais"},
		{"the copyright line the licence asks every file to carry", "Copyright (C) 2026 Nils Lehnen"},
		{"the licence, by name", "GNU Affero General Public License"},
		{"the version", "version 3"},
		{"a pointer to where the full terms are", "LICENSE"},
		{"the warranty position, so a file read on its own does not read as a promise", "warranty"},
	}
	for _, r := range required {
		if !strings.Contains(sourceheader.Header, r.text) {
			t.Errorf("the notice no longer states %s: %q is not in it", r.what, r.text)
		}
	}
}

// a fixture is one file somebody could plausibly write, and what has to happen
// to it.
type fixture struct {
	why    string
	file   string
	source string
}

// notice is the header as a file would carry it, so the fixtures below are built
// from the constant rather than from a second copy of it that could drift.
var notice = sourceheader.Header + "\n"

// refused holds the three ways a file gets this wrong. They are different
// mistakes and the refusal says which one, because the repair differs.
var refused = []fixture{
	{
		why:    "a new file written the way every file in this tree was written before the convention existed",
		file:   "internal/api/room.go",
		source: "package api\n\ntype Room struct{}\n",
	},
	{
		why:    "the notice pasted under the package documentation, where the tools that collect notices and the reader opening the file both miss it",
		file:   "internal/api/event.go",
		source: "// Package api translates what arrives on the wire.\npackage api\n\n" + notice + "\ntype Event struct{}\n",
	},
	{
		why:    "the notice retyped rather than copied, which is how thirteen files end up with thirteen notices",
		file:   "internal/api/limits.go",
		source: "// Copyright 2026 the relais authors. Licensed under the GNU Affero General Public License.\n\npackage api\n",
	},
	{
		why:    "a file with no package clause at all, which is a file this check has nothing to judge and must not pass in silence",
		file:   "internal/api/fragment.go",
		source: notice + "\ntype Room struct{}\n",
	},
}

func TestTheNoticeIsRefusedWhenItIsAbsentOrMisplaced(t *testing.T) {
	for _, f := range refused {
		t.Run(f.file, func(t *testing.T) {
			refusal := sourceheader.Refuse(f.file, f.source)
			if refusal == "" {
				t.Fatalf("%s is permitted and must not be: %s", f.file, f.why)
			}
			if !strings.Contains(refusal, f.file) {
				t.Errorf("the refusal does not name the file it is about: %s", refusal)
			}
		})
	}
}

// TestEachMistakeIsToldFromTheOthers is what makes the refusals worth having
// separately. A check reporting a misplaced notice as an absent one sends
// somebody to write a second copy of a notice the file already has.
func TestEachMistakeIsToldFromTheOthers(t *testing.T) {
	absent := sourceheader.Refuse("a.go", "package api\n")
	misplaced := sourceheader.Refuse("b.go", "package api\n\n"+notice)
	altered := sourceheader.Refuse("c.go", "// Licensed under the GNU Affero General Public License v3.\n\npackage api\n")

	if !strings.Contains(misplaced, "after its package clause") {
		t.Errorf("a misplaced notice is not reported as misplaced: %s", misplaced)
	}
	if strings.Contains(absent, "after its package clause") {
		t.Errorf("an absent notice is reported as a misplaced one: %s", absent)
	}
	if !strings.Contains(altered, "close to the notice but not the notice") {
		t.Errorf("an altered notice is not reported as altered: %s", altered)
	}
}

// permitted is the near-miss set: the file one line away from a refusal, which
// has to stay legal. A check that refused these would stop the work rather than
// the mistake.
var permitted = []fixture{
	{
		why:    "the ordinary shape, notice then package documentation then the clause",
		file:   "internal/api/api.go",
		source: notice + "\n// Package api translates what arrives on the wire.\npackage api\n",
	},
	{
		why:    "a file with no package documentation is still a file with a notice",
		file:   "internal/api/bare.go",
		source: notice + "\npackage api\n",
	},
	{
		why:    "a build constraint has to sit above the package clause too, and this convention has no opinion on which of the two comes first",
		file:   "internal/api/linux.go",
		source: "//go:build linux\n\n" + notice + "\npackage api\n",
	},
	{
		why:    "the same file with the constraint under the notice, for the same reason",
		file:   "internal/api/windows.go",
		source: notice + "\n//go:build windows\n\npackage api\n",
	},
	{
		why:    "a checkout on a machine that writes carriage returns is still the same file, and refusing it would send somebody to fight their editor rather than write the notice",
		file:   "internal/api/crlf.go",
		source: strings.ReplaceAll(notice+"\npackage api\n", "\n", "\r\n"),
	},
}

func TestThePermittedFilesStayPermitted(t *testing.T) {
	for _, p := range permitted {
		t.Run(p.file, func(t *testing.T) {
			if refusal := sourceheader.Refuse(p.file, p.source); refusal != "" {
				t.Errorf("%s is refused and must not be: %s\n%s", p.file, p.why, refusal)
			}
		})
	}
}

func carries(files []string, want string) bool {
	for _, file := range files {
		if file == want {
			return true
		}
	}
	return false
}
