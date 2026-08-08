// relais, a realtime SFU backend for community self-hosters.
// Copyright (C) 2026 Nils Lehnen
//
// Licensed under the GNU Affero General Public License, version 3. See LICENSE
// for the full terms, including the warranty and liability disclaimer.

// Package sourceheader holds the notice every Go file in this tree carries, and
// the reading of the tree that refuses a file without it.
//
// The licence this project is under says where the notice belongs, in the
// section of LICENSE headed "How to Apply These Terms to Your New Programs": it
// is safest to attach the notices to the start of each source file, and each
// file should have at least the copyright line and a pointer to where the full
// notice is found. That is the form below. It states the program, who holds the
// copyright and which licence applies, and it sends a reader to the file with
// the terms in it rather than repeating them thirteen times.
//
// A file is the unit that travels. A repository carries LICENSE at its root and
// a file copied out of it carries nothing, which is the case the notice exists
// for and the case a reader of one file is in.
//
// The check is here rather than in a workflow because the tree is what decides
// it and the suite is what already reads the tree. Nothing here imports anything
// else in this repository, and the files are read as text rather than parsed, so
// a file that does not compile is still judged.
package sourceheader

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Header is the notice, and it is the whole of the convention. One constant
// rather than a pattern, because a pattern admits thirteen slightly different
// notices and the point of a convention is that a reader who has read one has
// read them all.
//
// The year is the project's rather than the file's. A file added later carries
// this same block, which is what makes the convention checkable by comparison
// instead of by judgement; when the year changes it changes here and in every
// file, in one commit a reader can see.
//
// What it deliberately does not say is whether a later version of this licence
// may be used at the recipient's option. The licence chosen for this repository
// is recorded as version 3, in README.md and in the answer on issue #1, and
// neither states a position on later versions. A notice inventing one would be
// making that choice in a place nobody would look for it.
const Header = `// relais, a realtime SFU backend for community self-hosters.
// Copyright (C) 2026 Nils Lehnen
//
// Licensed under the GNU Affero General Public License, version 3. See LICENSE
// for the full terms, including the warranty and liability disclaimer.`

// licenceMention is what an altered notice still has in it. It is used only to
// tell "somebody edited the notice" from "there is no notice", because those are
// different mistakes with different repairs and a check that reported both as
// absence would send half its readers to the wrong one.
const licenceMention = "GNU Affero General Public License"

// Refuse reports why this file breaks the convention, or an empty string if it
// does not.
//
// The notice has to sit before the package clause. Anywhere else it is a comment
// about whatever it happens to precede: a reader opening the file sees the
// package documentation first and the tools that collect notices read the
// leading comments, so a correct notice in the wrong place is a file with no
// notice and a paragraph that looks like one.
//
// Above the package clause is deliberately looser than at the first byte. A file
// with a build constraint has to carry it in that region too, and a rule
// demanding the notice at line one would decide the order of two things this
// convention has no opinion about.
func Refuse(path, source string) string {
	source = strings.ReplaceAll(source, "\r\n", "\n")

	preamble, ok := beforePackageClause(source)
	if !ok {
		return fmt.Sprintf("%s has no package clause, so there is nothing for a notice to be in front of", path)
	}

	if strings.Contains(preamble, Header) {
		return ""
	}
	if strings.Contains(source, Header) {
		return fmt.Sprintf("%s carries the notice after its package clause, where it is a comment about the code beneath it "+
			"rather than the file's own notice; move it above the package clause", path)
	}
	if strings.Contains(source, licenceMention) {
		return fmt.Sprintf("%s carries something close to the notice but not the notice; it is one block, copied rather than "+
			"rewritten, so that a reader who has read one file has read them all:\n%s", path, Header)
	}
	return fmt.Sprintf("%s carries no licence notice. Every Go file in this tree begins with:\n%s", path, Header)
}

// beforePackageClause returns everything above the file's package clause. The
// second result is false when there is none.
//
// The clause is found by scanning lines rather than by parsing, so a file that
// does not compile is still judged, and a line inside a comment that merely
// begins with the word is not mistaken for it: a comment line starts with a
// slash and this test is anchored at the start of the line.
func beforePackageClause(source string) (string, bool) {
	lines := strings.Split(source, "\n")
	for i, line := range lines {
		if strings.HasPrefix(line, "package ") {
			return strings.Join(lines[:i], "\n"), true
		}
	}
	return "", false
}

// GoFiles returns every Go file under root, as repository-relative paths with
// forward slashes on every operating system.
func GoFiles(root string) ([]string, error) {
	var found []string

	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			if skippedDir(entry.Name()) && path != root {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(entry.Name(), ".go") {
			return nil
		}
		relative, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		found = append(found, filepath.ToSlash(relative))
		return nil
	})
	if err != nil {
		return nil, err
	}
	return found, nil
}

// skippedDir names the directories the Go toolchain does not build from, plus
// this repository's own metadata.
func skippedDir(name string) bool {
	return name == ".git" || name == ".github" || name == "testdata" || name == "vendor"
}

// ModuleRoot walks up from the working directory to the directory holding
// go.mod, so the tree being judged is found rather than assumed from a relative
// path that stops being true when this package moves.
func ModuleRoot() (string, error) {
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
			return "", fmt.Errorf("no go.mod above the working directory")
		}
		dir = parent
	}
}
