// relais, a realtime SFU backend for community self-hosters.
// Copyright (C) 2026 Nils Lehnen
//
// Licensed under the GNU Affero General Public License, version 3. See LICENSE
// for the full terms, including the warranty and liability disclaimer.

// Package coverage holds the bar this project puts on the code that decides
// admission, the list of packages that bar applies to, and the reading of a Go
// coverage profile that refuses a package below it.
//
// Issue #89 asks for a high bar on the code that decides security outcomes
// rather than a mediocre one on everything. A whole-tree percentage rewards
// testing whatever is easiest: a project can raise it by covering a formatter
// while the branch that decides whether a stranger is admitted goes unreached,
// and the number goes up either way. So the subject is a list rather than a
// total, and the list is here, in the tree, rather than in the workflow that
// runs it.
//
// WHAT A BAR IS FOR. A statement no test reaches on this surface is a security
// property nobody has checked. `docs/decisions/admission.md` names the five
// things a join turns on, and each of them is a branch: the signature, the
// validity window, the room binding, single use, and the closed set of
// permitted claims. A test suite that reaches four of the five leaves the fifth
// as an assertion in a document.
//
// THE LIST IS CLOSED IN BOTH DIRECTIONS, which is the property that keeps it
// from decaying. Every package under internal/ carries an entry: either a bar,
// or a written reason it is not a surface this bar covers. A package that
// appears in the profile and on no entry is refused, so the authorisation code
// that issue #46 will add cannot arrive uncovered and unnoticed. What no reading
// of a profile can decide is whether a reason is a good one, and a package
// classified as carrying no decision when it carries one passes here. That is
// the review's, and it is the whole of what this check hands over.
//
// The reading is of the profile the toolchain wrote rather than of a percentage
// somebody printed. That keeps the verdict offline, identical on a pull request
// from a fork, and reproducible from the same file by anybody holding it.
//
// It fails closed on the shapes where a green result would mean the reading
// never happened: a file with no mode line, a mode this reading does not know,
// a block line it cannot parse, a profile carrying no block under internal/, and
// a listed surface the profile does not mention at all.
//
// Nothing here imports anything else in this repository.
package coverage

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
)

// Root is the import path prefix of the packages this bar judges. The surfaces
// issue #89 names all sit under it, because docs/architecture.md puts every
// package that does work there and leaves cmd/ as wiring and test/ as the
// suites that read the tree.
//
// Blocks outside it are counted and reported rather than judged, so a profile
// produced over a wider set than the workflow produces still gives an answer
// about the surfaces instead of refusing every suite in test/ as unclassified.
const Root = "github.com/iderex/relais/internal/"

// A Surface is one package under Root and what this project says about it.
//
// Bar is a percentage of statements, and a Bar of zero says this package is not
// one of the surfaces the bar covers. The two cases are one type rather than
// two lists because the property that matters is that every package has an
// entry, and a reader checking that is better served by one table than by
// remembering to read both halves of a pair.
type Surface struct {
	// Package is the full import path, as the profile writes it.
	Package string
	// Bar is the percentage of statements that has to be covered, or zero
	// where this package carries none.
	Bar float64
	// Why is the sentence that puts this package on the list at this bar,
	// or the reason it carries no bar. It is written out in full so a
	// failure reads as the argument rather than as an identifier a reader
	// has to go and look up.
	Why string
}

// Surfaces is the list. It is data rather than a pattern over directory names,
// because "which packages decide admission" is a judgement somebody made and
// wrote down, and a pattern would make it look derived.
var Surfaces = []Surface{
	{
		Package: "github.com/iderex/relais/internal/orchestration/credential",
		Bar:     88.9,
		Why: "This is the admission decision. A credential arrives from a service above the seam, " +
			"is parsed out of bytes this project did not write, and the join turns on the signature, " +
			"the validity window, the room binding and single use. docs/decisions/admission.md states " +
			"that admission fails closed against a clock it cannot trust, and every one of those is a " +
			"branch that a suite either reaches or leaves as a sentence in a record.",
	},
	{
		Package: "github.com/iderex/relais/internal/orchestration/config",
		Bar:     98.4,
		Why: "This is what decides whether the service starts and in what state. Every value here arrives from " +
			"outside, and each of the three fall backs issue #79 names turns a refusal into a service running as " +
			"nobody intended: a missing credential that admits everyone, an unwritable data path that becomes " +
			"memory, a cap that does not parse and becomes no cap. Each of those is one branch, and a branch a " +
			"suite does not reach is a sentence in a record rather than a rule.",
	},
	{
		Package: "github.com/iderex/relais/internal/mediaplane",
		Bar:     0,
		Why: "The vocabulary both sides of the port share, and it decides nothing about who may do what. " +
			"docs/decisions/media-plane-port.md puts it under both sides and lets it depend on neither, " +
			"so nothing here admits a participant, verifies anything or reads bytes from a stranger. " +
			"A bar on it would be a whole-tree percentage wearing a list's clothes.",
	},
}

// The rule ids, and the argument behind each one. A failing check prints the
// argument beside the refusal, so somebody meeting it for the first time reads
// what the rule is for rather than only that it fired.
const (
	// RuleClassified refuses a package that is under Root, in the profile,
	// and on no entry of the list.
	RuleClassified = "every-package-under-internal-is-on-the-list"
	// RulePresent refuses a listed surface carrying a bar that the profile
	// does not mention.
	RulePresent = "a-surface-with-a-bar-is-in-the-profile"
	// RuleBar refuses a listed surface below its bar.
	RuleBar = "a-surface-is-at-or-above-its-bar"
)

var reasons = map[string]string{
	RuleClassified: "Every package under " + Root + " carries an entry in Surfaces: a bar, or a written reason it is not a surface this bar covers. " +
		"This is what stops the list from decaying into a snapshot of the day it was written. A package added later arrives here as a red check rather than as a silent gap, " +
		"and somebody has to say in one sentence which of the two it is. What this cannot decide is whether that sentence is true.",
	RulePresent: "A surface carrying a bar is measured rather than assumed. A profile that does not mention it is a profile produced over the wrong set of packages, " +
		"or a package that was renamed or removed while its entry stayed, and both of those pass a check that only compares the numbers it was given. " +
		"A bar nothing was measured against is worse than no bar, because the check reports success either way.",
	RuleBar: "The bar holds the line the code is already at rather than creating a backlog, and it only ever moves up. " +
		"It bites on the loss of one covered statement and on the arrival of one uncovered statement, which is deliberate: " +
		"a new branch on the admission surface is exactly the thing that should not land untested. " +
		"Where a change genuinely cannot reach a statement, the repair is the reason written beside a lowered bar and argued in review, never a quiet edit to the number.",
}

// Reason returns the argument behind a rule id, so a failure can print both
// without the caller holding the table.
func Reason(id string) string { return reasons[id] }

// A Verdict is one measured surface: what the bar asked for and what the
// profile carried.
type Verdict struct {
	Package    string
	Bar        float64
	Statements int
	Covered    int
	Percent    float64
}

// A Refusal is one thing this reading will not pass, located well enough that
// somebody meeting the failure does not have to go and find it.
type Refusal struct {
	RuleID  string
	Package string
	Detail  string
}

// A Report is what one reading of a profile produced. The counts are part of
// the result rather than a debugging aid: a reading that judged nothing exits
// exactly like one that judged everything and was happy, and the counts are what
// tell a reader which of the two happened.
type Report struct {
	// Mode is the profile's own mode line.
	Mode string
	// Blocks is how many distinct coverage blocks were read, after the
	// blocks a merged profile repeats have been folded together.
	Blocks int
	// Outside is how many of those sit outside Root and were therefore
	// counted rather than judged.
	Outside int
	// Judged is every surface carrying a bar, with what the profile said
	// about it, in list order.
	Judged []Verdict
	// Unbarred names the packages on the list that carry no bar and were
	// therefore not judged, so a package that passed by not being asked is
	// not read as one that passed.
	Unbarred []string
	// Refusals is everything that failed, rule by rule.
	Refusals []Refusal
}

// block is one coverage block, keyed so that a merged profile repeating the
// same range does not count its statements twice.
type block struct {
	pkg     string
	stmts   int
	covered bool
}

// knownModes are the three the toolchain writes. An unknown one is refused
// rather than guessed at, because the count column means different things in
// each and a reading that guessed wrong would report a number rather than fail.
var knownModes = map[string]bool{"set": true, "count": true, "atomic": true}

// Judge reads one Go coverage profile and reports what the list makes of it.
func Judge(profile []byte, surfaces []Surface) (Report, error) {
	blocks, mode, err := parse(profile)
	if err != nil {
		return Report{}, err
	}

	judged := Report{Mode: mode}

	measured := map[string]*Verdict{}
	for _, b := range blocks {
		judged.Blocks++
		if !strings.HasPrefix(b.pkg, Root) {
			judged.Outside++
			continue
		}
		at, ok := measured[b.pkg]
		if !ok {
			at = &Verdict{Package: b.pkg}
			measured[b.pkg] = at
		}
		at.Statements += b.stmts
		if b.covered {
			at.Covered += b.stmts
		}
	}

	if len(measured) == 0 {
		return Report{}, fmt.Errorf("the profile carries no coverage block under %s, "+
			"so nothing this list is about was measured", Root)
	}

	listed := map[string]Surface{}
	for _, s := range surfaces {
		listed[s.Package] = s
	}

	for _, name := range sorted(measured) {
		if _, ok := listed[name]; ok {
			continue
		}
		judged.Refusals = append(judged.Refusals, Refusal{
			RuleID:  RuleClassified,
			Package: name,
			Detail: fmt.Sprintf("%d statement(s) measured and no entry in test/coverage.Surfaces; "+
				"add one carrying a bar, or one saying in a sentence why this package decides nothing about admission",
				measured[name].Statements),
		})
	}

	for _, s := range surfaces {
		if s.Bar == 0 {
			judged.Unbarred = append(judged.Unbarred, s.Package)
			continue
		}
		at, ok := measured[s.Package]
		if !ok {
			judged.Refusals = append(judged.Refusals, Refusal{
				RuleID:  RulePresent,
				Package: s.Package,
				Detail: fmt.Sprintf("carries a bar of %.1f%% and the profile does not mention it, "+
					"so nothing was measured against that bar", s.Bar),
			})
			continue
		}
		at.Bar = s.Bar
		at.Percent = percent(at.Covered, at.Statements)
		judged.Judged = append(judged.Judged, *at)
		if at.Percent < s.Bar {
			judged.Refusals = append(judged.Refusals, Refusal{
				RuleID:  RuleBar,
				Package: s.Package,
				Detail: fmt.Sprintf("%.4f%% of statements covered (%d of %d) against a bar of %.1f%%",
					at.Percent, at.Covered, at.Statements, s.Bar),
			})
		}
	}

	return judged, nil
}

// percent is the one place the figure is computed, so the number in a failure
// message and the number the comparison was made on cannot differ.
func percent(covered, statements int) float64 {
	if statements == 0 {
		return 0
	}
	return 100 * float64(covered) / float64(statements)
}

// parse reads the profile into distinct blocks and returns its mode.
//
// Blocks are folded by file and range rather than appended, because a profile
// merged across several test binaries repeats a range once per binary that
// instrumented it, and counting those twice would inflate a percentage in the
// direction that makes the bar easier to meet.
func parse(profile []byte) (map[string]block, string, error) {
	lines := strings.Split(strings.ReplaceAll(string(profile), "\r\n", "\n"), "\n")

	mode := ""
	blocks := map[string]block{}
	for n, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if mode == "" {
			named, ok := strings.CutPrefix(line, "mode: ")
			if !ok {
				return nil, "", fmt.Errorf("line %d is %q and a coverage profile opens with its mode line, "+
					"so this file is not one and nothing was judged", n+1, short(line))
			}
			if !knownModes[named] {
				return nil, "", fmt.Errorf("the profile is in mode %q, which this reading does not know; "+
					"the count column means something different in each mode and guessing it would report a figure rather than fail", named)
			}
			mode = named
			continue
		}
		key, b, err := parseBlock(line)
		if err != nil {
			return nil, "", fmt.Errorf("line %d: %w", n+1, err)
		}
		was, seen := blocks[key]
		if seen {
			b.covered = b.covered || was.covered
		}
		blocks[key] = b
	}

	if mode == "" {
		return nil, "", fmt.Errorf("the profile is empty, and an empty profile is a run that measured nothing rather than a tree that is covered")
	}
	if len(blocks) == 0 {
		return nil, "", fmt.Errorf("the profile carries a %q mode line and no coverage block, "+
			"which is a run that instrumented nothing rather than one that found everything covered", mode)
	}
	return blocks, mode, nil
}

// parseBlock reads one block line, which the toolchain writes as an import path
// and file name, a colon, the range, the statement count and the execution
// count.
//
// The file is split at the last colon rather than the first, because an import
// path may carry one and a Windows path certainly does.
func parseBlock(line string) (string, block, error) {
	at := strings.LastIndex(line, ":")
	if at < 0 {
		return "", block{}, fmt.Errorf("%q carries no colon, so there is no file to attribute it to", short(line))
	}
	file := line[:at]
	fields := strings.Fields(line[at+1:])
	if len(fields) != 3 {
		return "", block{}, fmt.Errorf("%q has %d field(s) after the file name and a coverage block has three, "+
			"which are the range, the statement count and the execution count", short(line), len(fields))
	}

	stmts, err := strconv.Atoi(fields[1])
	if err != nil {
		return "", block{}, fmt.Errorf("%q carries %q as its statement count, which is not a number", short(line), fields[1])
	}
	count, err := strconv.Atoi(fields[2])
	if err != nil {
		return "", block{}, fmt.Errorf("%q carries %q as its execution count, which is not a number", short(line), fields[2])
	}

	slash := strings.LastIndex(file, "/")
	if slash < 0 {
		return "", block{}, fmt.Errorf("%q names %q, which carries no directory, so there is no package to attribute it to", short(line), file)
	}

	return file + ":" + fields[0], block{pkg: file[:slash], stmts: stmts, covered: count > 0}, nil
}

// sorted returns the package names in a fixed order, so a failure message is the
// same on two machines.
func sorted(measured map[string]*Verdict) []string {
	names := make([]string, 0, len(measured))
	for name := range measured {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// short keeps a malformed line readable in a failure message. A profile line is
// short, but a file that is not a profile at all can carry anything, and the
// message is the thing a reader meets first.
func short(line string) string {
	const most = 80
	if len(line) <= most {
		return line
	}
	return line[:most] + "..."
}
