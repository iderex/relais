// relais, a realtime SFU backend for community self-hosters.
// Copyright (C) 2026 Nils Lehnen
//
// Licensed under the GNU Affero General Public License, version 3. See LICENSE
// for the full terms, including the warranty and liability disclaimer.

package issuestate_test

import (
	"fmt"
	"os"
	"reflect"
	"sort"
	"strings"
	"testing"

	"github.com/iderex/relais/test/issuestate"
)

// fetchedState is where the job puts the response of the one query it makes, so
// this suite judges the real tracker in the job that fetched it. It is empty
// everywhere else, and the test that reads it says so rather than passing
// quietly, because a gate that judged nothing and a gate that judged the whole
// tree and was happy exit the same way.
const fetchedState = "RELAIS_TRACKER_STATE"

// TestNoDocumentCallsAClosedIssueUnresolved is the check itself. Everything
// below it exists to show that this one could have failed.
func TestNoDocumentCallsAClosedIssueUnresolved(t *testing.T) {
	fetched := os.Getenv(fetchedState)
	if fetched == "" {
		t.Skipf("%s names no file, so no tracker state was read here; the job sets it to "+
			"what the fetch wrote and this test is the gate there", fetchedState)
	}

	response, err := os.ReadFile(fetched)
	if err != nil {
		// The working directory is in the message because a test binary
		// runs in its own package directory rather than where the job
		// that set the variable stood, so a path that was right on the
		// command line resolves somewhere else here.
		where, _ := os.Getwd()
		t.Fatalf("reading the fetched tracker state at %s from %s: %v", fetched, where, err)
	}

	state, err := issuestate.ReadState(response)
	if err != nil {
		t.Fatalf("reading %s: %v", fetched, err)
	}

	root, err := issuestate.ModuleRoot()
	if err != nil {
		t.Fatalf("finding the module root: %v", err)
	}
	mentions, documents, err := issuestate.ReadMentions(root)
	if err != nil {
		t.Fatalf("reading the documents under %s: %v", root, err)
	}

	judged := issuestate.Judge(mentions, state, documents)
	t.Logf("%d document(s), %d mention(s), of which %d name a closed issue, %d a pull request, "+
		"and %d a number the tracker does not carry",
		judged.Documents, judged.Mentions, judged.Closed, judged.PullRequests, len(judged.Unknown))
	if len(judged.Unknown) > 0 {
		t.Logf("not resolved, and therefore not judged: %v", judged.Unknown)
	}
	if judged.Documents == 0 {
		t.Fatal("no document was read, so this run judged nothing and would have exited green")
	}

	for _, f := range judged.Findings {
		t.Errorf("%s at %s:%d: %s\n  %s", f.RuleID, f.File, f.Line, f.Detail, issuestate.Reason(f.RuleID))
	}
}

// theTracker is the state every fixture below is judged against. It is the five
// numbers the hand pass in issue #178 turned on, with the states they carry, so
// a fixture is judged by exactly the distinction the tree is judged by.
var theTracker = map[int]issuestate.Tracked{
	4:   {Number: 4, Closed: true},
	14:  {Number: 14, Closed: true},
	39:  {Number: 39, Closed: false},
	45:  {Number: 45, Closed: true},
	91:  {Number: 91, Closed: true},
	124: {Number: 124, Closed: false},
	181: {Number: 181, Closed: true, PullRequest: true},
}

// The three sites, quoted from the documents the hand pass in #178 found them
// in. They are fixtures rather than a reading of the tree: every one of them was
// repaired before this check existed, so a suite that judged the tree for them
// would be judging text nobody can put back.
//
// #178 found four. The fourth is not of this shape and is not here, and the
// difference is the whole reason this rule is narrow. That site rests on an
// entry inside issue #1 having been answered while the issue itself is open, so
// there is no closed issue to find and no reading of tracker state that would
// have caught it. It is named in the package documentation as outside the
// subject rather than left to look like an oversight.
const (
	theListeningSocket = `The media plane listening socket is an operator input rather than a derivation,
because it has to match what the operator opened. Its shape is fixed by issue #4
and the network shape by issue #14, and both are open.
`

	theVerifier = `The set of claims a credential may carry is closed.

Implemented by issue #45, which is not built. The mechanism that refuses a
credential carrying a claim outside that set is built and lives in
` + "`internal/orchestration/credential`" + `.
`

	theMediaBoundary = `The parsing surface is the part of this project most likely to be attacked.
Refusing malformed and hostile input at the media boundary is #39. Fuzzing that
surface is #91. Neither exists yet.
`
)

// TestEverySiteTheHandPassFoundIsRefused is the proof that the rule bites, and
// it bites on the text that was actually written rather than on text invented to
// be caught. Emptying the vocabulary or dropping the back-reference join turns
// this red and nothing else in the file goes with it.
func TestEverySiteTheHandPassFoundIsRefused(t *testing.T) {
	cases := map[string]struct {
		source string
		want   []int
	}{
		// Two numbers in one sentence and one assertion covering both,
		// which is why the rule reports a finding per mention rather
		// than per sentence.
		"the listening socket": {theListeningSocket, []int{4, 14}},
		// The contradiction inside one paragraph: the sentence saying
		// it is not built, immediately before the sentence saying it
		// is.
		"the verifier": {theVerifier, []int{45}},
		// The number and the assertion in adjacent sentences, joined
		// by a back-reference that has no subject of its own. The open
		// issue in the same paragraph is not refused, which is the
		// half that shows the state is read rather than the wording.
		"the media boundary": {theMediaBoundary, []int{91}},
	}

	for what, c := range cases {
		mentions := issuestate.MentionsIn("docs/fixture.md", c.source)
		var refused []int
		for _, f := range issuestate.Refusals(mentions, theTracker) {
			if f.RuleID != "closed-issue-not-called-unresolved" {
				t.Errorf("%s was refused by %q, and this suite knows one rule", what, f.RuleID)
			}
			refused = append(refused, numberIn(t, f.Detail))
		}
		sort.Ints(refused)
		if !reflect.DeepEqual(refused, c.want) {
			t.Errorf("%s refused %v, want %v", what, refused, c.want)
		}
	}
}

// TestTheNearMissesArePermitted is the other half of the bite, and it is the
// half that decides whether this rule is worth having. Every case below names a
// closed issue, which is the population #178 measured at fifty-seven mentions in
// nineteen files, of which four were wrong. A rule that refused these would be
// the naive one the issue rules out.
func TestTheNearMissesArePermitted(t *testing.T) {
	permitted := map[string]string{
		// The commonest shape in this tree, and right precisely
		// because the issue closed.
		"a record naming the issue it answers": "Recorded for issue #45.\n",
		// Past tense about a closed issue is a statement about
		// history rather than about today.
		"a past-tense mention": "The verifier landed by #45 and the package it named is in the tree.\n",
		// The word inside another word. This tree opens ports, opens
		// rooms and opens windows.
		"open inside a longer word": "Issue #4 fixed the port the operator has already opened.\n",
		// The assertion belongs to the sentence after the one naming
		// the issue, and nothing joins them.
		"an assertion in the next sentence": "Issue #4 settled the shape of the socket. The network shape is open.\n",
		// An open issue said to be open is the sentence this rule
		// exists to leave alone.
		"an open issue called open": "The mechanism is #124, which is open.\n",
		// A number that is a pull request rather than an issue. They
		// share the numbering, so a merged one would otherwise read as
		// a closed issue.
		"a pull request number": "The change is #181 and the question it closed is not settled.\n",
	}

	for what, source := range permitted {
		mentions := issuestate.MentionsIn("docs/fixture.md", source)
		if len(mentions) == 0 {
			t.Errorf("%s produced no mention at all, so it passes by not being read", what)
		}
		if found := issuestate.Refusals(mentions, theTracker); len(found) > 0 {
			t.Errorf("%s was refused: %s", what, found[0].Detail)
		}
	}
}

// TestTheFalsePositiveThisRuleHasIsRefusedRatherThanHidden holds the cost of the
// window being a sentence rather than a meaning. A sentence naming a closed
// issue and an open one, where the assertion belongs to the open one, is refused
// and the rule cannot tell. It is asserted here so that the day somebody widens
// the rule they meet this case rather than discovering it on a document.
func TestTheFalsePositiveThisRuleHasIsRefusedRatherThanHidden(t *testing.T) {
	const source = "The verifier landed by #45 and the powers it carries are #124, which is open.\n"

	found := issuestate.Refusals(issuestate.MentionsIn("docs/fixture.md", source), theTracker)
	if len(found) != 1 {
		t.Fatalf("the known false positive produced %d finding(s), want 1", len(found))
	}
	if got := numberIn(t, found[0].Detail); got != 45 {
		t.Errorf("the finding names #%d, and the false positive is the closed issue #45", got)
	}
}

// TestAMentionInsideABlockIsNotJudged holds the two sites #178 named and asked
// to be left alone: a pasted command output carrying a state that has since
// moved. Those are dated by the block they sit in, and a rule that refused them
// would ask for a repair that deletes the date.
func TestAMentionInsideABlockIsNotJudged(t *testing.T) {
	sources := map[string]string{
		"an indented block": "The board at the commit this record landed on:\n\n" +
			"    gh issue list --repo iderex/relais\n" +
			"    #45 [OPEN] Mint and verify the join credential\n",
		"a fenced block": "The board at the commit this record landed on:\n\n" +
			"```\n#45 is open\n```\n",
	}

	for what, source := range sources {
		if mentions := issuestate.MentionsIn("docs/fixture.md", source); len(mentions) > 0 {
			t.Errorf("%s produced %d mention(s), and a block is not prose", what, len(mentions))
		}
	}
}

// TestAMentionCarriesTheLineItIsWrittenOn holds the part a reader needs. A
// sentence here runs across line breaks, so the line reported has to be the one
// the number is on rather than the one the sentence started on.
func TestAMentionCarriesTheLineItIsWrittenOn(t *testing.T) {
	const source = "First line.\n\nThe shape is fixed by issue #4 and\nthe network shape by issue #14, and both are open.\n"

	mentions := issuestate.MentionsIn("docs/fixture.md", source)
	if len(mentions) != 2 {
		t.Fatalf("read %d mention(s), want 2", len(mentions))
	}
	if mentions[0].Line != 3 || mentions[1].Line != 4 {
		t.Errorf("the two mentions are on lines %d and %d, want 3 and 4", mentions[0].Line, mentions[1].Line)
	}
}

// TestAFragmentAndAnAnchorAreNotIssueNumbers holds the shape that would
// otherwise turn every link into a mention.
func TestAFragmentAndAnAnchorAreNotIssueNumbers(t *testing.T) {
	const source = "See [the gap](docs/threat-model.md#what-has-no-mitigation) and issue #45.\n"

	mentions := issuestate.MentionsIn("docs/fixture.md", source)
	if len(mentions) != 1 || mentions[0].Number != 45 {
		t.Fatalf("read %v, want one mention of #45", mentions)
	}
}

// TestTheReadingFailsClosed covers the three shapes in which a pass would mean
// the fetch never happened rather than that the tree is clean. Every rule here
// leaves a number it has no state for unjudged, so an empty or broken fetch
// would otherwise be a green run over nothing.
func TestTheReadingFailsClosed(t *testing.T) {
	broken := map[string]string{
		"an empty response":            "",
		"a response of blank lines":    "\n\n\n",
		"a line that is not an object": `{"number": 4, "state": "closed"}` + "\nnot json\n",
		"a line naming no number":      `{"state": "closed"}` + "\n",
	}

	for what, response := range broken {
		if _, err := issuestate.ReadState([]byte(response)); err == nil {
			t.Errorf("%s was read as a clean fetch", what)
		}
	}

	state, err := issuestate.ReadState([]byte(`{"number":4,"state":"closed"}` + "\n" + `{"number":39,"state":"open"}` + "\n"))
	if err != nil {
		t.Fatalf("a well-formed response was refused: %v", err)
	}
	if !state[4].Closed || state[39].Closed {
		t.Errorf("the states read as %v, want #4 closed and #39 open", state)
	}
}

// TestAPullRequestIsMarkedAsOne holds the field that keeps a merged pull request
// from reading as a closed issue. The tracker endpoint returns both under one
// numbering and marks the difference with a member rather than a type.
func TestAPullRequestIsMarkedAsOne(t *testing.T) {
	state, err := issuestate.ReadState([]byte(`{"number":181,"state":"closed","pull_request":true}` + "\n"))
	if err != nil {
		t.Fatalf("reading: %v", err)
	}
	if !state[181].PullRequest {
		t.Error("a pull request was read as an issue, so its number judges prose")
	}
}

// TestTheSameInputGivesTheSameVerdict is the property being copied from the
// hygiene gate rather than the rule itself. Everything here is a function of the
// bytes handed to it, and a check whose answer moves between two runs on one
// input is an opinion rather than a gate.
func TestTheSameInputGivesTheSameVerdict(t *testing.T) {
	source := theListeningSocket + "\n" + theVerifier + "\n" + theMediaBoundary

	first := issuestate.Judge(issuestate.MentionsIn("docs/fixture.md", source), theTracker, 1)
	for range 10 {
		again := issuestate.Judge(issuestate.MentionsIn("docs/fixture.md", source), theTracker, 1)
		if !reflect.DeepEqual(first, again) {
			t.Fatalf("two readings of one input disagree:\n%+v\n%+v", first, again)
		}
	}
}

// TestTheReportSaysWhatItCouldNotResolve holds the counts. A run that resolved
// nothing exits exactly like a run that resolved everything and was happy, so
// the numbers in the log are what tell a reader which of the two happened.
func TestTheReportSaysWhatItCouldNotResolve(t *testing.T) {
	const source = "The shape is #4, the powers are #124, the change is #181 and the plan is #9999.\n"

	judged := issuestate.Judge(issuestate.MentionsIn("docs/fixture.md", source), theTracker, 1)
	if judged.Mentions != 4 {
		t.Errorf("read %d mention(s), want 4", judged.Mentions)
	}
	if judged.Closed != 1 || judged.PullRequests != 1 {
		t.Errorf("counted %d closed and %d pull request(s), want 1 and 1", judged.Closed, judged.PullRequests)
	}
	if !reflect.DeepEqual(judged.Unknown, []int{9999}) {
		t.Errorf("the unresolved numbers are %v, want [9999]", judged.Unknown)
	}
}

// TestEveryRuleCarriesItsReason holds the convention the neighbouring suites
// keep. A rule whose reason is empty fails as an identifier rather than as an
// argument, and the reader of that failure has nowhere to go.
func TestEveryRuleCarriesItsReason(t *testing.T) {
	for _, r := range issuestate.Rules {
		if strings.TrimSpace(r.ID) == "" {
			t.Error("a rule carries no id")
		}
		if len(strings.Fields(issuestate.Reason(r.ID))) < 20 {
			t.Errorf("the reason behind %q is too short to be an argument", r.ID)
		}
	}
}

// numberIn pulls the issue number out of a finding's detail, which is where the
// rule writes it.
func numberIn(t *testing.T, detail string) int {
	t.Helper()
	var number int
	if _, err := fmt.Sscanf(detail, "issue #%d ", &number); err != nil {
		t.Fatalf("the detail %q does not open with an issue number: %v", detail, err)
	}
	return number
}
