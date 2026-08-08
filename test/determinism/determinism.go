// relais, a realtime SFU backend for community self-hosters.
// Copyright (C) 2026 Nils Lehnen
//
// Licensed under the GNU Affero General Public License, version 3. See LICENSE
// for the full terms, including the warranty and liability disclaimer.

// Package determinism holds the three properties issue #26 fixes about this
// repository's suite, in the form a machine can refuse a violation of.
//
// Those properties are that no production path reads the machine's clock, that
// no unit test opens a network socket, and that no test waits on real time. All
// three are true of the tree today and none of them was held by anything: the
// analyser set in .golangci.yml judges none of these calls, so each property was
// a fact about the current files rather than a rule, and an empty grep result is
// the state of a tree on the day it ran. This package is what turns the three
// into refusals.
//
// It is written now, while the tree still satisfies all three, on purpose. A
// check added after the first violation is a check written around that
// violation, and these are the properties that erode one convenient test at a
// time rather than in a single visible step.
//
// The rules are data rather than code, so the table can be read as prose beside
// the property it holds, and so a fixture can be put through exactly the
// decision the tree is put through. Nothing here imports anything else in this
// repository: the tree is read as files rather than as packages, so a rule still
// has an answer on a tree that does not compile.
package determinism

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
)

// A Use is one mention of an identifier belonging to an imported package:
// time.Sleep, net.Listen, the type time.Time. Every rule below is a statement
// about a use.
//
// It is a use rather than a call because the distinction does not survive
// reading a file as text. A program can take time.Sleep as a value and call it
// under another name three files away, and a check that only looked at call
// expressions would report the tree as clean.
type Use struct {
	// File is the repository-relative path of the file the use is in, with
	// forward slashes on every operating system.
	File string
	// Test reports whether that file is a Go test file. It is the subject
	// half of two of the three rules, so it is carried rather than
	// recomputed at each rule.
	Test bool
	// Package is the import path the identifier came from, as written in the
	// import declaration: "time", "net", "net/http". It is the path and not
	// the local name, so an alias hides nothing and a package whose name
	// merely begins with a watched one is a different package.
	Package string
	// Name is the identifier selected from that package: "Sleep", "Listen",
	// "Time". A dot import has no such identifier and is recorded as
	// [DotImport].
	Name string
	// Line is where the use is, so a failure sends a reader to it rather
	// than to the file it is somewhere inside.
	Line int
}

// DotImport is the Name recorded for a dot import, which brings a package's
// identifiers into the file's own scope and leaves no selector for any other
// rule to read. It is a name no Go identifier can take, so it cannot collide
// with a real one.
const DotImport = "."

// A Rule refuses one thing. Reason is the sentence the rule is stated in,
// written out in full, so a failing test reads as the argument rather than as an
// identifier a reader has to go and look up.
type Rule struct {
	ID     string
	Reason string
	// Refuses reports whether this use breaks the rule.
	Refuses func(u Use) bool
}

// systemClock names the calls in the time package that read the machine's clock.
// A value built from parts is not among them: time.Date and time.Unix construct
// a moment from what the caller already knew, which is what a test does when it
// fixes one.
var systemClock = map[string]bool{
	"Now":   true,
	"Since": true,
	"Until": true,
}

// realWait names the calls in the time package that wait on the machine's clock,
// whether or not the word sleep appears. time.Sleep is the one the issue names;
// the others are the same wait spelled differently, and a check that only knew
// the first would be evaded by a select on time.After the first time somebody
// wanted a timeout in a test.
//
// time.NewTimer and time.NewTicker are here for the same reason and not because
// constructing one waits: what they hand back fires off real time, so a test
// holding one has a result that depends on how loaded the machine is.
var realWait = map[string]bool{
	"Sleep":     true,
	"After":     true,
	"Tick":      true,
	"NewTimer":  true,
	"NewTicker": true,
	"AfterFunc": true,
}

// networkPackages names the packages a socket is opened through, mapped to the
// identifiers in each that open one. Everything else in those packages is
// permitted, because parsing an address, joining a host and a port, or naming a
// status code reaches no network.
//
// net/http/httptest is here rather than trusted for being a testing package: its
// servers bind a real port on the loopback interface, which is a socket on the
// machine the suite is running on and is exactly what makes a suite depend on
// that machine.
var networkPackages = map[string]map[string]bool{
	"net": {
		"Dial": true, "DialTimeout": true, "DialIP": true, "DialTCP": true,
		"DialUDP": true, "DialUnix": true,
		"Listen": true, "ListenIP": true, "ListenTCP": true, "ListenUDP": true,
		"ListenUnix": true, "ListenUnixgram": true, "ListenPacket": true,
		"ListenMulticastUDP": true,
		"LookupHost":         true, "LookupIP": true, "LookupAddr": true,
		"LookupCNAME": true, "LookupPort": true, "LookupTXT": true,
	},
	"net/http": {
		"Get": true, "Post": true, "PostForm": true, "Head": true,
		"ListenAndServe": true, "ListenAndServeTLS": true, "Serve": true,
	},
	"net/http/httptest": {
		"NewServer": true, "NewTLSServer": true, "NewUnstartedServer": true,
	},
}

// watchedPackages is every package the rules above read. A dot import of one of
// them is refused by the last rule, and this is the set that rule is written
// against, so a package added to the tables above is covered by it without a
// second edit.
func watchedPackages() map[string]bool {
	watched := map[string]bool{"time": true}
	for path := range networkPackages {
		watched[path] = true
	}
	return watched
}

// Rules is the whole table. The first three are the three done-when conditions
// of issue #26 in order; the fourth exists because the first three read
// selectors and one import form leaves none to read.
var Rules = []Rule{
	{
		ID: "production-reads-no-clock",
		Reason: "A production path does not read the machine's clock. Time arrives as a parameter, the way `Clock` reaches the minter and the verifier in internal/orchestration/credential, " +
			"because an expiry compared against a clock nobody injected cannot be driven across its own boundary by a test, and a boundary nothing has driven across is a boundary nobody has seen work. " +
			"Waiting counts as reading: a path that sleeps, ticks or times out on real time makes the same test probable rather than exact. " +
			"The day a production path genuinely has to reach the machine's clock, the one file that adapts it is named in this table, in one commit a reader can see.",
		Refuses: func(u Use) bool {
			return !u.Test && u.Package == "time" && (systemClock[u.Name] || realWait[u.Name])
		},
	},
	{
		ID: "unit-test-opens-no-socket",
		Reason: "A unit test opens no network socket. A suite that binds a port or resolves a name is a suite whose result depends on the machine it ran on, on what else was listening, and on whether anything answered, " +
			"which is the whole of what a unit result is supposed to be free of. " +
			"Every test in this tree is a unit test today. The media integration harness in issue #40 is where that stops being true, and it is where this rule acquires the one directory it does not judge.",
		Refuses: func(u Use) bool {
			return u.Test && networkPackages[u.Package][u.Name]
		},
	},
	{
		ID: "test-does-not-wait-on-real-time",
		Reason: "A test does not wait on real time. A sleep in a test is either a hidden race or wasted seconds and is usually both, and the interesting case in a project full of timeouts, retransmissions and expiry is the boundary, which real waiting cannot reach exactly. " +
			"time.After, time.Tick, time.NewTimer, time.NewTicker and time.AfterFunc are refused beside time.Sleep because they are the same wait under another name. " +
			"A test that has to show something does not happen changes this table with its argument, rather than reaching for the spelling this rule has not heard of.",
		Refuses: func(u Use) bool {
			return u.Test && u.Package == "time" && realWait[u.Name]
		},
	},
	{
		ID: "a-watched-package-is-not-dot-imported",
		Reason: "None of the packages the rules above read is dot-imported. A dot import puts Sleep, Listen and Now into the file's own scope with no package name in front of them, " +
			"so every rule above goes on passing while the thing it refuses is written in plain sight. It is refused in production and in tests alike, because the evasion does not care which file it is in.",
		Refuses: func(u Use) bool {
			return u.Name == DotImport && watchedPackages()[u.Package]
		},
	},
}

// Refusals returns the rules that refuse this use, in table order. An empty
// result is a permitted use.
func Refusals(u Use) []Rule {
	var refused []Rule
	for _, r := range Rules {
		if r.Refuses(u) {
			refused = append(refused, r)
		}
	}
	return refused
}

// UsesInFile parses one file's source and returns every use of an identifier
// belonging to a package it imports.
//
// Local names are resolved to import paths rather than assumed, so an alias
// carries no weight: `import clock "time"` produces uses whose Package is
// "time". The reverse also holds, which is the case a name-based check gets
// wrong: `import time "golang.org/x/time/rate"` produces uses whose Package is
// that module path and which no rule here judges.
//
// What it does not see is a local identifier shadowing an import, so a variable
// called time with a field called Now would be reported as the standard library.
// That is a false positive rather than a hole, it is refused loudly rather than
// silently, and the repair is to rename the variable.
func UsesInFile(path, source string) ([]Use, error) {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, path, source, parser.SkipObjectResolution)
	if err != nil {
		return nil, fmt.Errorf("parsing %s: %w", path, err)
	}

	test := strings.HasSuffix(path, "_test.go")

	// localName maps the name an import is reachable by in this file to the
	// path it came from. A blank import is left out: it binds no name, so
	// nothing in the file can select from it.
	localName := map[string]string{}
	var uses []Use
	for _, spec := range file.Imports {
		importPath := strings.Trim(spec.Path.Value, `"`)
		switch {
		case spec.Name == nil:
			localName[lastSegment(importPath)] = importPath
		case spec.Name.Name == "_":
		case spec.Name.Name == ".":
			uses = append(uses, Use{
				File:    path,
				Test:    test,
				Package: importPath,
				Name:    DotImport,
				Line:    fset.Position(spec.Pos()).Line,
			})
		default:
			localName[spec.Name.Name] = importPath
		}
	}

	ast.Inspect(file, func(node ast.Node) bool {
		selector, ok := node.(*ast.SelectorExpr)
		if !ok {
			return true
		}
		ident, ok := selector.X.(*ast.Ident)
		if !ok {
			return true
		}
		importPath, ok := localName[ident.Name]
		if !ok {
			return true
		}
		uses = append(uses, Use{
			File:    path,
			Test:    test,
			Package: importPath,
			Name:    selector.Sel.Name,
			Line:    fset.Position(selector.Sel.Pos()).Line,
		})
		return true
	})

	return uses, nil
}

// ReadUses walks the tree at root and returns every use in every Go file in it,
// test files included, since two of the three rules are about test files and a
// reader that skipped them would judge nothing.
func ReadUses(root string) ([]Use, error) {
	var found []Use

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
		relative = filepath.ToSlash(relative)

		source, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		uses, err := UsesInFile(relative, string(source))
		if err != nil {
			return err
		}
		found = append(found, uses...)
		return nil
	})
	if err != nil {
		return nil, err
	}
	return found, nil
}

// lastSegment is the package name an import is reachable by when it declares no
// alias. It is the last path segment, which is right for every import in this
// tree and is what the language specification says a compiler may not rely on:
// a package may declare a name other than its directory. Such an import is
// written with an explicit alias in this tree, and the near-miss set says so.
func lastSegment(importPath string) string {
	if cut := strings.LastIndex(importPath, "/"); cut >= 0 {
		return importPath[cut+1:]
	}
	return importPath
}

// skippedDir names the directories the Go toolchain does not build from, plus
// this repository's own metadata. Reading them would report uses no rule is
// about.
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
