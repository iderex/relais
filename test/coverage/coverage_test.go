// relais, a realtime SFU backend for community self-hosters.
// Copyright (C) 2026 Nils Lehnen
//
// Licensed under the GNU Affero General Public License, version 3. See LICENSE
// for the full terms, including the warranty and liability disclaimer.

package coverage_test

import (
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/iderex/relais/test/coverage"
)

// coverageProfile is where the coverage job puts the profile the toolchain
// wrote, so this suite judges the real measurement in the job that produced it.
// It is empty everywhere else, and the test that reads it says so rather than
// passing quietly, because a gate that judged nothing and a gate that judged
// everything and was happy exit the same way.
const coverageProfile = "RELAIS_COVERAGE_PROFILE"

// TestEverySurfaceIsAtOrAboveItsBar is the check itself. Everything below it
// exists to show that this one could have failed.
func TestEverySurfaceIsAtOrAboveItsBar(t *testing.T) {
	path := os.Getenv(coverageProfile)
	if path == "" {
		t.Skipf("%s names no file, so no coverage was judged here; the coverage job sets it "+
			"to the profile the toolchain wrote and this test is the gate there", coverageProfile)
	}

	profile, err := os.ReadFile(path)
	if err != nil {
		// The working directory is in the message because a test binary
		// runs in its own package directory rather than where the job
		// that set the variable stood, so a path that was right on the
		// command line resolves somewhere else here.
		where, _ := os.Getwd()
		t.Fatalf("reading the coverage profile at %s from %s: %v", path, where, err)
	}

	judged, err := coverage.Judge(profile, coverage.Surfaces)
	if err != nil {
		t.Fatalf("judging %s: %v", path, err)
	}

	t.Logf("mode %s, %d block(s) read, %d of them outside %s and counted rather than judged",
		judged.Mode, judged.Blocks, judged.Outside, coverage.Root)
	for _, v := range judged.Judged {
		t.Logf("%s: %.4f%% covered (%d of %d statements) against a bar of %.1f%%",
			v.Package, v.Percent, v.Covered, v.Statements, v.Bar)
	}
	for _, name := range judged.Unbarred {
		t.Logf("%s carries no bar and was not judged", name)
	}

	for _, r := range judged.Refusals {
		t.Errorf("%s\n\t%s: %s\n\t%s", r.RuleID, r.Package, r.Detail, coverage.Reason(r.RuleID))
	}
}

// TestTheListItselfIsWellFormed judges the table rather than a profile. A
// duplicate entry would make which bar applies depend on map iteration, and an
// entry with no sentence is the one shape this list cannot be read as data.
func TestTheListItselfIsWellFormed(t *testing.T) {
	seen := map[string]bool{}
	for _, s := range coverage.Surfaces {
		if seen[s.Package] {
			t.Errorf("%s has two entries, so which bar applies depends on the order they are read in", s.Package)
		}
		seen[s.Package] = true

		if !strings.HasPrefix(s.Package, coverage.Root) {
			t.Errorf("%s is not under %s, and this list judges nothing outside it", s.Package, coverage.Root)
		}
		if strings.TrimSpace(s.Why) == "" {
			t.Errorf("%s carries no reason, and an entry with no reason is a number nobody can argue with", s.Package)
		}
		if s.Bar < 0 || s.Bar > 100 {
			t.Errorf("%s carries a bar of %.1f, which is not a percentage of statements", s.Package, s.Bar)
		}
	}
}

// profileOf writes a profile carrying one package's worth of blocks, so a
// fixture reads as the case it is about rather than as a wall of ranges.
//
// Each block is one statement, which makes a fixture's percentage the count of
// covered blocks over the count of blocks and keeps the arithmetic in the test
// visible to the reader.
func profileOf(pkg string, covered, uncovered int) string {
	var b strings.Builder
	b.WriteString("mode: set\n")
	line := 10
	for range covered {
		fmt.Fprintf(&b, "%s/file.go:%d.13,%d.2 1 1\n", pkg, line, line+1)
		line += 3
	}
	for range uncovered {
		fmt.Fprintf(&b, "%s/file.go:%d.13,%d.2 1 0\n", pkg, line, line+1)
		line += 3
	}
	return b.String()
}

// theCredential is the surface that carries a bar today, so the fixtures below
// are about it rather than about an invented path that no entry would match.
const theCredential = "github.com/iderex/relais/internal/orchestration/credential"

// refusalsOf runs the judge and returns the rule ids it refused under, which is
// what every case below asserts about.
func refusalsOf(t *testing.T, profile string, surfaces []coverage.Surface) []string {
	t.Helper()
	judged, err := coverage.Judge([]byte(profile), surfaces)
	if err != nil {
		t.Fatalf("judging the fixture: %v", err)
	}
	ids := make([]string, 0, len(judged.Refusals))
	for _, r := range judged.Refusals {
		ids = append(ids, r.RuleID)
	}
	return ids
}

// theBar is the list this suite judges fixtures against. It is the shape of the
// real one rather than the real one, because a row judging the tree proves the
// state of the tree on the day it ran and not the guard.
var theBar = []coverage.Surface{
	{Package: theCredential, Bar: 90, Why: "the fixture's barred surface"},
	{Package: coverage.Root + "vocabulary", Bar: 0, Why: "the fixture's unbarred surface"},
}

func TestASurfaceBelowItsBarIsRefused(t *testing.T) {
	// Eighty-nine covered statements in a hundred, against a bar of ninety.
	// One statement below, which is the near miss the bar exists for rather
	// than a fixture that could not have passed.
	ids := refusalsOf(t, profileOf(theCredential, 89, 11), theBar)
	if len(ids) != 1 || ids[0] != coverage.RuleBar {
		t.Fatalf("89 covered of 100 against a bar of 90 refused %v, and the bar is what should have refused it", ids)
	}
}

func TestASurfaceExactlyAtItsBarPasses(t *testing.T) {
	// The other side of the same one-statement step. At rather than above,
	// because a bar that refused the number it names would be a different
	// bar written down wrong.
	ids := refusalsOf(t, profileOf(theCredential, 90, 10), theBar)
	if len(ids) != 0 {
		t.Fatalf("90 covered of 100 against a bar of 90 refused %v, and a surface at its bar meets it", ids)
	}
}

func TestAPackageOnNoEntryIsRefused(t *testing.T) {
	// The case this list exists to survive: work lands in a package nobody
	// classified, and the check goes red rather than reporting on the
	// packages somebody remembered.
	profile := profileOf(theCredential, 90, 10) +
		strings.TrimPrefix(profileOf(coverage.Root+"authorisation", 1, 9), "mode: set\n")
	ids := refusalsOf(t, profile, theBar)
	if len(ids) != 1 || ids[0] != coverage.RuleClassified {
		t.Fatalf("a package under %s with no entry refused %v, and being on no entry is what should have refused it",
			coverage.Root, ids)
	}
}

func TestASurfaceWithABarThatTheProfileNeverMentionsIsRefused(t *testing.T) {
	// A bar nothing was measured against reports success exactly like a bar
	// that was met. This is the profile produced over the wrong package set,
	// and the renamed package whose entry stayed behind.
	ids := refusalsOf(t, profileOf(coverage.Root+"vocabulary", 10, 0), theBar)
	if len(ids) != 1 || ids[0] != coverage.RulePresent {
		t.Fatalf("a barred surface absent from the profile refused %v, and its absence is what should have refused it", ids)
	}
}

func TestAPackageOutsideTheRootIsCountedRatherThanJudged(t *testing.T) {
	profile := profileOf(theCredential, 90, 10) +
		strings.TrimPrefix(profileOf("github.com/iderex/relais/test/doclint", 0, 4), "mode: set\n")
	judged, err := coverage.Judge([]byte(profile), theBar)
	if err != nil {
		t.Fatalf("judging the fixture: %v", err)
	}
	if len(judged.Refusals) != 0 {
		t.Fatalf("a package outside %s refused %v, and this list judges nothing outside it", coverage.Root, judged.Refusals)
	}
	if judged.Outside != 4 {
		t.Fatalf("%d block(s) reported outside %s and the fixture carries 4; a block that was neither judged nor counted "+
			"is one a reader has no way to notice", judged.Outside, coverage.Root)
	}
}

func TestAMergedProfileCountsARepeatedBlockOnce(t *testing.T) {
	// A profile merged across test binaries repeats a range once per binary
	// that instrumented it. Counting those twice inflates the percentage in
	// the direction that makes the bar easier to meet, which is the
	// direction a miscount must never go.
	one := profileOf(theCredential, 89, 11)
	merged := one + strings.TrimPrefix(one, "mode: set\n")
	ids := refusalsOf(t, merged, theBar)
	if len(ids) != 1 || ids[0] != coverage.RuleBar {
		t.Fatalf("the same 89 of 100 written twice refused %v, and folding the repeat is what keeps it the same measurement", ids)
	}
}

func TestARepeatedBlockCoveredInOnlyOneBinaryCounts(t *testing.T) {
	// The other half of folding. Two binaries instrument one range and only
	// one of them reached it, so the statement is covered. Taking the last
	// line rather than the union would report it as unreached.
	const line = theCredential + "/file.go:10.13,11.2 1 "
	profile := "mode: set\n" + line + "1\n" + line + "0\n"
	judged, err := coverage.Judge([]byte(profile), theBar)
	if err != nil {
		t.Fatalf("judging the fixture: %v", err)
	}
	if len(judged.Judged) != 1 || judged.Judged[0].Covered != 1 {
		t.Fatalf("a block reached by one binary of two was judged %+v, and it is covered", judged.Judged)
	}
}

// The shapes below are refused before any surface is looked at, because a green
// verdict on each of them would mean the reading never happened rather than that
// the tree is covered.
func TestTheReadingFailsClosed(t *testing.T) {
	cases := []struct {
		name    string
		profile string
		says    string
	}{
		{
			name:    "no mode line",
			profile: theCredential + "/file.go:10.13,11.2 1 1\n",
			says:    "opens with its mode line",
		},
		{
			name:    "a mode this reading does not know",
			profile: "mode: sampled\n" + theCredential + "/file.go:10.13,11.2 1 1\n",
			says:    "does not know",
		},
		{
			name:    "empty",
			profile: "",
			says:    "empty profile is a run that measured nothing",
		},
		{
			name:    "a mode line and nothing else",
			profile: "mode: set\n",
			says:    "instrumented nothing",
		},
		{
			name:    "a block line with a field missing",
			profile: "mode: set\n" + theCredential + "/file.go:10.13,11.2 1\n",
			says:    "a coverage block has three",
		},
		{
			name:    "a statement count that is not a number",
			profile: "mode: set\n" + theCredential + "/file.go:10.13,11.2 one 1\n",
			says:    "not a number",
		},
		{
			name:    "nothing under the root",
			profile: profileOf("github.com/iderex/relais/test/doclint", 4, 0),
			says:    "carries no coverage block under",
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			_, err := coverage.Judge([]byte(c.profile), theBar)
			if err == nil {
				t.Fatalf("this shape was judged rather than refused, and a verdict on it would say the reading happened")
			}
			if !strings.Contains(err.Error(), c.says) {
				t.Fatalf("the refusal reads %q and a reader meeting it needs %q in it", err, c.says)
			}
		})
	}
}
