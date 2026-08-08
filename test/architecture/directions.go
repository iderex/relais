// Package architecture holds the dependency directions from
// docs/architecture.md in the form a machine can refuse a violation of.
//
// That document states the directions and then says, in its own last section,
// that nothing enforces them and that until issue #95 lands they are a rule a
// person applies by reading a diff. This package is the half of #95 that can be
// decided from the code: which package may import which. The two rules #95 also
// names, that the forwarding core never learns who a participant is and that the
// degradation decision lives in one place, are not here, and the issue carries
// the reason.
//
// The rules are data rather than code so that the table below can be read
// against the document beside it, and so that a fixture can be put through the
// same decision the tree is put through. Nothing here imports anything else in
// this repository: the tree is read as files rather than as packages, so a
// direction that would not compile is still measurable.
package architecture

import (
	"fmt"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
)

// modulePath is the import prefix of this repository, from go.mod. An import
// carrying it is a package in this tree; anything else is external.
const modulePath = "github.com/iderex/relais"

// transportLibrary is the import prefix of the library the forwarding core is
// built on. docs/architecture.md calls it "the transport library" and does not
// name it, because naming it there would be a second place to change;
// docs/decisions/forwarding-core.md is where the choice of pion/webrtc is
// argued, and this constant is the one place this test spells it.
const transportLibrary = "github.com/pion/webrtc"

// A Rule refuses one direction. Reason is the sentence docs/architecture.md
// states it in, quoted so a failing test reads as the document rather than as an
// identifier somebody has to go and look up.
type Rule struct {
	ID     string
	Reason string
	// Refuses reports whether importer, a repository-relative package
	// directory such as "internal/api", importing imported, an import path as
	// written in the source, breaks this rule.
	Refuses func(importer, imported string) bool
}

// Rules is the whole table, in the order docs/architecture.md states them under
// "The dependency directions". Every one of them is quoted from that section.
var Rules = []Rule{
	{
		ID:     "media-plane-depends-on-neither-side",
		Reason: "`internal/mediaplane` may not import `internal/forwarding`, and may not import `internal/orchestration` or `internal/api`. It sits under both sides and depends on neither.",
		Refuses: func(importer, imported string) bool {
			pkg, local := localPackage(imported)
			return local && under(importer, "internal/mediaplane") &&
				(under(pkg, "internal/forwarding") || under(pkg, "internal/orchestration") || under(pkg, "internal/api"))
		},
	},
	{
		ID:     "transport-library-only-below-the-port",
		Reason: "`internal/mediaplane` may not import the transport library. Neither may `internal/orchestration`, `internal/api`, or `cmd/relais`.",
		Refuses: func(importer, imported string) bool {
			if !isTransportLibrary(imported) {
				return false
			}
			return under(importer, "internal/mediaplane") || under(importer, "internal/orchestration") ||
				under(importer, "internal/api") || under(importer, "cmd/relais")
		},
	},
	{
		ID:     "forwarding-core-does-not-reach-upward",
		Reason: "`internal/forwarding` may not import `internal/orchestration` or `internal/api`. The forwarding core does not reach upward.",
		Refuses: func(importer, imported string) bool {
			pkg, local := localPackage(imported)
			return local && under(importer, "internal/forwarding") &&
				(under(pkg, "internal/orchestration") || under(pkg, "internal/api"))
		},
	},
	{
		ID:     "above-the-port-reaches-the-port-and-nothing-below-it",
		Reason: "`internal/orchestration` and `internal/api` may not import `internal/forwarding`. They reach the media plane through the port and through nothing else.",
		Refuses: func(importer, imported string) bool {
			pkg, local := localPackage(imported)
			return local && under(pkg, "internal/forwarding") &&
				(under(importer, "internal/orchestration") || under(importer, "internal/api"))
		},
	},
	{
		ID:     "orchestration-does-not-know-its-wire-format",
		Reason: "`internal/api` may import `internal/orchestration`. Not the other way round: an event stream that knows its own wire format is an event stream that changes when the wire format does.",
		Refuses: func(importer, imported string) bool {
			pkg, local := localPackage(imported)
			return local && under(importer, "internal/orchestration") && under(pkg, "internal/api")
		},
	},
	{
		ID:     "bench-measures-a-target-it-is-pointed-at",
		Reason: "`bench/` may not import `cmd/` or `internal/`, and nothing under `cmd/` or `internal/` may import `bench/`. The bench measures a target it is pointed at.",
		Refuses: func(importer, imported string) bool {
			pkg, local := localPackage(imported)
			if !local {
				return false
			}
			outward := under(importer, "bench") && (under(pkg, "cmd") || under(pkg, "internal"))
			inward := (under(importer, "cmd") || under(importer, "internal")) && under(pkg, "bench")
			return outward || inward
		},
	},
	{
		ID:     "nothing-imports-the-entry-point",
		Reason: "`cmd/relais` may import anything under `internal/`. Nothing may import `cmd/relais`.",
		Refuses: func(_, imported string) bool {
			pkg, local := localPackage(imported)
			return local && under(pkg, "cmd")
		},
	},
}

// Refusals returns the rules that refuse this import, in table order. An empty
// result is a permitted import.
func Refusals(importer, imported string) []Rule {
	var refused []Rule
	for _, r := range Rules {
		if r.Refuses(importer, imported) {
			refused = append(refused, r)
		}
	}
	return refused
}

// localPackage turns an import path into a repository-relative package
// directory. The second result is false for anything outside this module,
// including the standard library, which no rule here judges.
func localPackage(imported string) (string, bool) {
	if imported == modulePath {
		return ".", true
	}
	if rest, ok := strings.CutPrefix(imported, modulePath+"/"); ok {
		return rest, true
	}
	return "", false
}

// under reports whether dir is prefix or sits inside it. The separator test is
// what keeps "internal/apiary" from matching "internal/api".
func under(dir, prefix string) bool {
	return dir == prefix || strings.HasPrefix(dir, prefix+"/")
}

// isTransportLibrary reports whether an import path is the transport library or
// a package inside it. The major-version suffix is part of the path, so the
// prefix test covers v4 and whatever succeeds it without naming a version.
func isTransportLibrary(imported string) bool {
	return imported == transportLibrary || strings.HasPrefix(imported, transportLibrary+"/")
}

// An Import is one import statement, located well enough for a failure message
// to send a reader straight to it.
type Import struct {
	// Package is the repository-relative directory the importing file is in.
	Package string
	// File is the repository-relative path of the importing file.
	File string
	// Path is the import path as written.
	Path string
}

// ReadImports parses every Go file under root and returns every import in them,
// test files included. An import of the forwarding core from an orchestration
// test is the failure docs/architecture.md names in its own words, so a reader
// that skipped test files would miss the case the rule exists for.
//
// The tree is read with go/parser in import-only mode rather than through the
// build system, so the answer does not depend on a package resolving, on a
// module download, or on a network being reachable.
func ReadImports(root string) ([]Import, error) {
	var found []Import
	fset := token.NewFileSet()

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

		file, err := parser.ParseFile(fset, path, nil, parser.ImportsOnly)
		if err != nil {
			return fmt.Errorf("parsing %s: %w", relative, err)
		}
		for _, spec := range file.Imports {
			found = append(found, Import{
				Package: dirOf(relative),
				File:    relative,
				Path:    strings.Trim(spec.Path.Value, `"`),
			})
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return found, nil
}

// dirOf takes the directory part of a slash-separated repository-relative path.
// filepath.Dir is not used because it returns the host separator, and every
// path a rule is written against is spelled with a forward slash whatever the
// operating system underneath is.
func dirOf(relative string) string {
	cut := strings.LastIndex(relative, "/")
	if cut < 0 {
		return "."
	}
	return relative[:cut]
}

// skippedDir names the directories the Go toolchain itself does not build from,
// plus the repository's own metadata. Reading them would report imports that no
// direction is about.
func skippedDir(name string) bool {
	return name == ".git" || name == ".github" || name == "testdata" || name == "vendor"
}

// ModuleRoot walks up from the working directory to the directory holding
// go.mod. A test finds the tree it is judging this way rather than from a
// relative path, so moving this package does not silently point it at nothing.
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
