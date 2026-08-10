// relais, a realtime SFU backend for community self-hosters.
// Copyright (C) 2026 Nils Lehnen
//
// Licensed under the GNU Affero General Public License, version 3. See LICENSE
// for the full terms, including the warranty and liability disclaimer.

package fuzz_test

import (
	"strings"
	"testing"

	"github.com/iderex/relais/test/fuzz"
)

// TestEveryPackageThatDecodesAStrangerIsFuzzed is the check itself. Everything
// below it exists to show that this one could have failed.
func TestEveryPackageThatDecodesAStrangerIsFuzzed(t *testing.T) {
	root, err := fuzz.ModuleRoot()
	if err != nil {
		t.Fatalf("finding the module root: %v", err)
	}

	parsers, files, err := fuzz.ReadParsers(root)
	if err != nil {
		t.Fatalf("reading %s under %s: %v", fuzz.Under, root, err)
	}
	targets, err := fuzz.ReadTargets(root)
	if err != nil {
		t.Fatalf("reading the targets in %s: %v", fuzz.Directory, err)
	}

	judged, err := fuzz.Judge(parsers, targets, files)
	if err != nil {
		t.Fatalf("deriving the surfaces to fuzz: %v", err)
	}

	t.Logf("%d Go file(s) read under %s/, %d decoder import(s) in them, %d target(s) declared in %s",
		judged.Files, fuzz.Under, len(parsers), len(judged.Targets), fuzz.Directory)
	for pkg, decoders := range judged.Parsing {
		t.Logf("%s decodes external input, by %s", pkg, strings.Join(decoders, " and "))
	}
	for _, target := range judged.Targets {
		t.Logf("%s in %s exercises %s", target.Name, target.File, strings.Join(target.Exercises, ", "))
	}

	for _, r := range judged.Refusals {
		t.Errorf("%s\n\t%s: %s\n\t%s", r.RuleID, r.Subject, r.Detail, fuzz.Reason(r.RuleID))
	}
}

// TestTheDerivationFindsTheSurfaceThisTreeHas holds the derivation against the
// tree rather than against a fixture. A reading that found nothing would satisfy
// every rule above by having nothing to judge, and this is what tells the two
// apart.
func TestTheDerivationFindsTheSurfaceThisTreeHas(t *testing.T) {
	root, err := fuzz.ModuleRoot()
	if err != nil {
		t.Fatalf("finding the module root: %v", err)
	}
	parsers, files, err := fuzz.ReadParsers(root)
	if err != nil {
		t.Fatalf("reading %s under %s: %v", fuzz.Under, root, err)
	}
	if files == 0 {
		t.Fatalf("no Go file was read under %s/, so the walk found nothing to derive from", fuzz.Under)
	}

	const admission = "internal/orchestration/credential"
	for _, p := range parsers {
		if p.Package == admission {
			return
		}
	}
	t.Fatalf("the derivation did not find %s among %d parser import(s), and that package decodes a token "+
		"a stranger presented; either the walk is broken or the vocabulary lost an entry", admission, len(parsers))
}

// The fixtures below carry the shape the derivation produces rather than a shape
// that would be convenient to judge, and they are fixtures rather than the tree:
// a case judging this repository's real packages proves the state of the tree on
// the day it ran and never the rule.
const (
	parsingPackage = "internal/orchestration/credential"
	quietPackage   = "internal/mediaplane"
)

var aParser = []fuzz.Parser{{
	Package: parsingPackage,
	File:    parsingPackage + "/credential.go",
	Decoder: "encoding/base64",
}}

func aTargetExercising(packages ...string) []fuzz.Target {
	return []fuzz.Target{{
		Name:      "FuzzTheFixture",
		File:      fuzz.Directory + "/fixture_test.go",
		Exercises: packages,
	}}
}

func refusalsOf(t *testing.T, parsers []fuzz.Parser, targets []fuzz.Target, files int) []string {
	t.Helper()
	judged, err := fuzz.Judge(parsers, targets, files)
	if err != nil {
		t.Fatalf("judging the fixture: %v", err)
	}
	ids := make([]string, 0, len(judged.Refusals))
	for _, r := range judged.Refusals {
		ids = append(ids, r.RuleID)
	}
	return ids
}

func TestAParsingPackageWithNoTargetIsRefused(t *testing.T) {
	// The case this derivation exists for. A package grows a decoder in a
	// change about something else, nobody writes a target, and without this
	// the suite goes on reporting that the surfaces are fuzzed.
	ids := refusalsOf(t, aParser, aTargetExercising(quietPackage, "internal/forwarding"), 12)
	if len(ids) != 2 || ids[0] != fuzz.RuleCovered || ids[1] != fuzz.RuleAimed {
		t.Fatalf("a decoding package with no target refused %v, and both rules bite on this fixture: "+
			"the package is uncovered and the only target is aimed at nothing", ids)
	}
}

func TestAParsingPackageWithATargetPasses(t *testing.T) {
	ids := refusalsOf(t, aParser, aTargetExercising(parsingPackage, quietPackage), 12)
	if len(ids) != 0 {
		t.Fatalf("a decoding package exercised by a target refused %v, and that is the covered case", ids)
	}
}

func TestATargetAimedAtNothingIsRefused(t *testing.T) {
	// The other direction. The decoder moved out of the package, the target
	// stayed, and it now runs green over bytes nothing parses.
	targets := append(aTargetExercising(parsingPackage), fuzz.Target{
		Name:      "FuzzWhatMoved",
		File:      fuzz.Directory + "/moved_test.go",
		Exercises: []string{quietPackage},
	})
	ids := refusalsOf(t, aParser, targets, 12)
	if len(ids) != 1 || ids[0] != fuzz.RuleAimed {
		t.Fatalf("a target exercising only a package that decodes nothing refused %v, "+
			"and being aimed at nothing is what should have refused it", ids)
	}
}

// The shapes below are refused before any package is looked at, because a green
// verdict on each of them would mean the derivation never happened rather than
// that every surface is fuzzed.
func TestTheDerivationFailsClosed(t *testing.T) {
	cases := []struct {
		name    string
		parsers []fuzz.Parser
		targets []fuzz.Target
		files   int
		says    string
	}{
		{
			name:    "no file was read",
			parsers: aParser,
			targets: aTargetExercising(parsingPackage),
			files:   0,
			says:    "read an empty tree",
		},
		{
			name:    "no target was declared",
			parsers: aParser,
			targets: nil,
			files:   12,
			says:    "no fuzz target was found",
		},
		{
			name:    "nothing in the tree decodes anything",
			parsers: nil,
			targets: aTargetExercising(parsingPackage),
			files:   12,
			says:    "found nothing to fuzz",
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			_, err := fuzz.Judge(c.parsers, c.targets, c.files)
			if err == nil {
				t.Fatalf("this shape was judged rather than refused, and a verdict on it would say the derivation happened")
			}
			if !strings.Contains(err.Error(), c.says) {
				t.Fatalf("the refusal reads %q and a reader meeting it needs %q in it", err, c.says)
			}
		})
	}
}

// TestTheVocabularyIsWhatTheRuleSaysItIs holds the one piece of data this
// derivation rests on. An entry with no sentence beside it is an entry a failure
// cannot explain, and a name outside the standard library is a dependency this
// suite does not have.
func TestTheVocabularyIsWhatTheRuleSaysItIs(t *testing.T) {
	if len(fuzz.Decoders) == 0 {
		t.Fatalf("the vocabulary is empty, so nothing in any tree would ever be found to parse")
	}
	for path, why := range fuzz.Decoders {
		if strings.TrimSpace(why) == "" {
			t.Errorf("%s carries no sentence, and a refusal naming it would send the reader nowhere", path)
		}
		if strings.Contains(path, ".") {
			t.Errorf("%s looks like a module path rather than a standard library package, "+
				"and this derivation reads imports of the standard library", path)
		}
	}
}
