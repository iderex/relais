// relais, a realtime SFU backend for community self-hosters.
// Copyright (C) 2026 Nils Lehnen
//
// Licensed under the GNU Affero General Public License, version 3. See LICENSE
// for the full terms, including the warranty and liability disclaimer.

// Package fuzz holds the fuzz targets over the surfaces that take bytes from
// strangers, and the derivation that says which surfaces those are.
//
// Issue #91 asks for a target per external parser and for the list of parsers to
// come from the code rather than from somebody's memory. A hand-written list is
// correct on the day it is written and silently wrong afterwards, and the parser
// nobody added is exactly the one nobody was thinking about. So the list is
// derived here, on every run, from what the tree imports.
//
// WHAT MARKS A PARSER. A package that reads bytes it did not write reaches for a
// decoder to do it, and the decoders are in a small, stable part of the standard
// library. Decoders below is that vocabulary. A package under internal/ that
// imports one of them is a package this repository decodes external input in, and
// it owes a fuzz target. The derivation is over imports rather than over call
// sites because an import is what a file cannot decode without.
//
// THE BOUND ON THAT, said here rather than left to be discovered by somebody
// reading a green result as more than it is. The vocabulary is a floor: a parser
// written by hand out of index expressions and string slicing imports nothing and
// is invisible to this. It holds what a parser in this tree actually looks like
// today, and a shape nobody has written yet walks past it. Widening it is one
// entry in one map. What it does catch is the ordinary case, which is a package
// that grows a decoder in a change about something else.
//
// It is also coarse on purpose. A target is credited to every local package its
// file imports, so a target that imports a parsing package without exercising it
// satisfies this check. Deciding which function a fuzz target actually reaches is
// a call graph rather than a reading, and a check claiming to have made that
// decision would be claiming more than it did.
//
// IT FAILS CLOSED IN BOTH DIRECTIONS. A parsing package with no target is
// refused, which is the case this exists for. A target exercising no parsing
// package is refused too, because that is a target left behind by a decoder that
// moved, still running, still green, and aimed at nothing.
//
// This package imports the packages it fuzzes, which is the one place the
// convention that these suites import nothing else in this tree cannot hold: a
// fuzz target is bytes handed to a real function, and there is no version of that
// which reads the tree as text.
package fuzz

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

// Module is this repository's import path, so a local import can be told from a
// standard library one and turned back into a directory.
const Module = "github.com/iderex/relais"

// Under is the part of the tree searched for parsers. cmd/ is wiring and test/
// is the suites, and neither is where bytes from a stranger are decoded.
const Under = "internal"

// Directory is where the targets live, relative to the module root.
const Directory = "test/fuzz"

// Decoders is the vocabulary. Each entry is a standard library package whose
// presence in a file means that file turns bytes somebody else chose into a
// value this program then acts on, and the sentence beside it is what a failure
// prints so the reader is not sent to look the entry up.
//
// The encoding family is here in full rather than only the members the tree uses
// today, because the entry that matters is the one added the day a package starts
// decoding something new, and an entry that is already present costs nothing.
// net/url is here for the same reason: a URL arriving from a stranger is parsed
// input whatever it is called.
var Decoders = map[string]string{
	"encoding/asn1":   "decodes ASN.1, which is length-prefixed and arrives from whoever sent it",
	"encoding/base64": "decodes base64, which is where a token stops being text and starts being bytes",
	"encoding/binary": "reads fixed-width fields out of a byte slice somebody else laid out",
	"encoding/csv":    "decodes separated text, where the separators are the sender's",
	"encoding/hex":    "decodes hexadecimal, which is bytes wearing text",
	"encoding/json":   "decodes JSON, whose shape and depth are the sender's to choose",
	"encoding/pem":    "decodes PEM, which is base64 inside a framing the sender writes",
	"encoding/xml":    "decodes XML, whose nesting is the sender's to choose",
	"net/url":         "parses a URL, which arrives as text from whoever sent it",
}

// A Parser is one file under Under that imports a decoder, and the decoder it
// imported. It is per import rather than per package so a failure names the file
// somebody has to open.
type Parser struct {
	// Package is the repository-relative directory, such as
	// "internal/orchestration/credential".
	Package string
	// File is the repository-relative path, with forward slashes on every
	// operating system.
	File string
	// Decoder is the standard library import that put this file here.
	Decoder string
}

// A Target is one fuzz target in Directory and the local packages its file
// imports.
type Target struct {
	// Name is the function name, such as "FuzzACredentialFromAStranger".
	Name string
	// File is the repository-relative path of the file declaring it.
	File string
	// Exercises is every package of this repository the file imports, as
	// repository-relative directories.
	Exercises []string
}

// The rule ids, and the argument behind each one, printed beside a refusal so
// somebody meeting it reads what the rule is for rather than only that it fired.
const (
	// RuleCovered refuses a package that decodes external input and has no
	// fuzz target.
	RuleCovered = "a-package-that-decodes-a-stranger-has-a-fuzz-target"
	// RuleAimed refuses a fuzz target that exercises no such package.
	RuleAimed = "a-fuzz-target-is-aimed-at-something-that-parses"
)

var reasons = map[string]string{
	RuleCovered: "Every package under " + Under + "/ that imports a decoder has a fuzz target. This is the whole of what issue #91 asks for and the reason the list is derived rather than written: " +
		"a parser reachable by anyone who can send a packet is the surface least exercised by ordinary use and most exercised by somebody trying, " +
		"and the one that gets missed is the one that arrived in a change about something else. The repair is a target in " + Directory + " whose file imports the package.",
	RuleAimed: "A fuzz target exercises at least one package that decodes external input. A target aimed at nothing runs, passes and reports a surface as fuzzed, " +
		"which is the failure a green result cannot be told apart from. It is what a decoder moving out of a package leaves behind, and the repair is to point the target at where the parsing went or to delete it.",
}

// Reason returns the argument behind a rule id.
func Reason(id string) string { return reasons[id] }

// A Refusal is one thing this derivation will not pass.
type Refusal struct {
	RuleID  string
	Subject string
	Detail  string
}

// A Report is what one derivation produced. The counts are part of the result
// rather than a debugging aid: a run that read an empty tree exits exactly like
// one that read the whole of it and was happy.
type Report struct {
	// Parsing is every package under Under that imports a decoder, sorted,
	// with the decoders that put it there.
	Parsing map[string][]string
	// Files is how many files under Under were read.
	Files int
	// Targets is every fuzz target found, sorted by name.
	Targets []Target
	// Refusals is everything that failed, rule by rule.
	Refusals []Refusal
}

// Judge derives the parsing packages from the parsers, matches them against the
// targets, and reports what is uncovered and what is aimed at nothing.
func Judge(parsers []Parser, targets []Target, files int) (Report, error) {
	if files == 0 {
		return Report{}, fmt.Errorf("no Go file was read under %s/, so nothing was derived; "+
			"a derivation that read an empty tree passes exactly like one that read the whole of it", Under)
	}
	if len(targets) == 0 {
		return Report{}, fmt.Errorf("no fuzz target was found under %s, and a run with no target "+
			"reports every surface as unfuzzed or as fine depending only on which way this check is read", Directory)
	}

	judged := Report{Parsing: map[string][]string{}, Files: files, Targets: targets}
	for _, p := range parsers {
		judged.Parsing[p.Package] = appendOnce(judged.Parsing[p.Package], p.Decoder)
	}
	if len(judged.Parsing) == 0 {
		return Report{}, fmt.Errorf("no package under %s/ imports any of the %d decoders this derivation knows, "+
			"so it found nothing to fuzz; on a tree that decodes nothing that is a check to change rather than one to pass quietly",
			Under, len(Decoders))
	}

	exercised := map[string]bool{}
	for _, t := range targets {
		for _, pkg := range t.Exercises {
			exercised[pkg] = true
		}
	}

	for _, pkg := range sortedKeys(judged.Parsing) {
		if exercised[pkg] {
			continue
		}
		judged.Refusals = append(judged.Refusals, Refusal{
			RuleID:  RuleCovered,
			Subject: pkg,
			Detail: fmt.Sprintf("imports %s and no fuzz target in %s exercises it",
				strings.Join(judged.Parsing[pkg], ", "), Directory),
		})
	}

	for _, t := range targets {
		aimed := false
		for _, pkg := range t.Exercises {
			if _, ok := judged.Parsing[pkg]; ok {
				aimed = true
				break
			}
		}
		if aimed {
			continue
		}
		judged.Refusals = append(judged.Refusals, Refusal{
			RuleID:  RuleAimed,
			Subject: t.Name,
			Detail: fmt.Sprintf("in %s, exercising %s, none of which decodes external input",
				t.File, describe(t.Exercises)),
		})
	}

	return judged, nil
}

// ReadParsers walks the tree under root and returns every file under Under that
// imports a decoder, with how many Go files were read.
//
// Test files are skipped. A test importing a decoder is a test building an input,
// which is the opposite of the thing being looked for.
func ReadParsers(root string) ([]Parser, int, error) {
	var found []Parser
	files := 0

	err := filepath.WalkDir(filepath.Join(root, Under), func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			if skippedDir(entry.Name()) {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(entry.Name(), ".go") || strings.HasSuffix(entry.Name(), "_test.go") {
			return nil
		}
		files++

		relative, err := relativeTo(root, path)
		if err != nil {
			return err
		}
		imports, err := importsOf(path, parser.ImportsOnly)
		if err != nil {
			return err
		}
		for _, imported := range imports {
			if _, ok := Decoders[imported]; !ok {
				continue
			}
			found = append(found, Parser{
				Package: pathDir(relative),
				File:    relative,
				Decoder: imported,
			})
		}
		return nil
	})
	if err != nil {
		return nil, 0, err
	}
	return found, files, nil
}

// ReadTargets walks Directory and returns every fuzz target declared there, with
// the local packages each one's file imports.
//
// A fuzz target is a function whose name begins with Fuzz and which takes one
// parameter, which is what the toolchain itself looks for. Reading it from the
// source rather than from a list keeps a target that was renamed from being
// counted under the name it used to have.
func ReadTargets(root string) ([]Target, error) {
	var found []Target

	dir := filepath.Join(root, filepath.FromSlash(Directory))
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), "_test.go") {
			continue
		}
		path := filepath.Join(dir, entry.Name())
		relative, err := relativeTo(root, path)
		if err != nil {
			return nil, err
		}

		set := token.NewFileSet()
		file, err := parser.ParseFile(set, path, nil, parser.SkipObjectResolution)
		if err != nil {
			return nil, fmt.Errorf("reading %s: %w", relative, err)
		}

		var exercises []string
		for _, imported := range importsIn(file) {
			local, ok := strings.CutPrefix(imported, Module+"/")
			if !ok {
				continue
			}
			exercises = appendOnce(exercises, local)
		}

		for _, decl := range file.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok || fn.Recv != nil {
				continue
			}
			if !strings.HasPrefix(fn.Name.Name, "Fuzz") || fn.Name.Name == "Fuzz" {
				continue
			}
			if fn.Type.Params == nil || len(fn.Type.Params.List) != 1 {
				continue
			}
			found = append(found, Target{Name: fn.Name.Name, File: relative, Exercises: exercises})
		}
	}

	sort.Slice(found, func(i, j int) bool { return found[i].Name < found[j].Name })
	return found, nil
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

// importsOf parses one file at the given depth and returns its import paths.
func importsOf(path string, mode parser.Mode) ([]string, error) {
	set := token.NewFileSet()
	file, err := parser.ParseFile(set, path, nil, mode|parser.SkipObjectResolution)
	if err != nil {
		return nil, fmt.Errorf("reading %s: %w", filepath.ToSlash(path), err)
	}
	return importsIn(file), nil
}

// importsIn returns the import paths of a parsed file, unquoted. An import whose
// path will not unquote is skipped rather than guessed at; the compiler is what
// refuses that file.
func importsIn(file *ast.File) []string {
	var paths []string
	for _, spec := range file.Imports {
		path, err := strconv.Unquote(spec.Path.Value)
		if err != nil {
			continue
		}
		paths = append(paths, path)
	}
	return paths
}

// relativeTo returns a repository-relative path with forward slashes on every
// operating system, so a failure message reads the same everywhere.
func relativeTo(root, path string) (string, error) {
	relative, err := filepath.Rel(root, path)
	if err != nil {
		return "", err
	}
	return filepath.ToSlash(relative), nil
}

// pathDir is filepath.Dir for a path that is already using forward slashes.
func pathDir(path string) string {
	at := strings.LastIndex(path, "/")
	if at < 0 {
		return "."
	}
	return path[:at]
}

// skippedDir names the directories the Go toolchain does not build from.
func skippedDir(name string) bool {
	return name == ".git" || name == "testdata" || name == "vendor"
}

// describe renders a package list for a failure message, saying "nothing" rather
// than printing an empty bracket.
func describe(packages []string) string {
	if len(packages) == 0 {
		return "no package of this repository"
	}
	return strings.Join(packages, ", ")
}

func appendOnce(list []string, value string) []string {
	for _, held := range list {
		if held == value {
			return list
		}
	}
	list = append(list, value)
	sort.Strings(list)
	return list
}

func sortedKeys(m map[string][]string) []string {
	keys := make([]string, 0, len(m))
	for key := range m {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}
