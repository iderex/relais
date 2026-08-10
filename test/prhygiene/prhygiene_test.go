// relais, a realtime SFU backend for community self-hosters.
// Copyright (C) 2026 Nils Lehnen
//
// Licensed under the GNU Affero General Public License, version 3. See LICENSE
// for the full terms, including the warranty and liability disclaimer.

package prhygiene_test

import (
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"

	"github.com/iderex/relais/test/doclint"
	"github.com/iderex/relais/test/prhygiene"
)

// fetchedPullRequest is where the hygiene workflow puts the response of the one
// query it makes, so this suite judges the real pull request in the job that
// fetched it. It is empty everywhere else, and the test that reads it says so
// rather than passing quietly, because a gate that judged nothing and a gate
// that judged a whole change and was happy exit the same way.
const fetchedPullRequest = "RELAIS_PULL_REQUEST"

// templatePath is the tracked template the means rule compares a body against.
// It is a path in this tree rather than a copy of the text, so the rule follows
// the template when the template is reworded.
const templatePath = ".github/PULL_REQUEST_TEMPLATE.md"

// TestThePullRequestKeepsItsShape is the check itself. Everything below it
// exists to show that this one could have failed.
func TestThePullRequestKeepsItsShape(t *testing.T) {
	fetched := os.Getenv(fetchedPullRequest)
	if fetched == "" {
		t.Skipf("%s names no file, so no pull request was judged here; the hygiene job sets it "+
			"to what the fetch wrote and this test is the gate there", fetchedPullRequest)
	}

	response, err := os.ReadFile(fetched)
	if err != nil {
		// The working directory is in the message because a test binary
		// runs in its own package directory rather than where the job
		// that set the variable stood, so a path that was right on the
		// command line resolves somewhere else here.
		where, _ := os.Getwd()
		t.Fatalf("reading the fetched pull request at %s from %s: %v", fetched, where, err)
	}

	root, err := doclint.ModuleRoot()
	if err != nil {
		t.Fatalf("finding the module root: %v", err)
	}
	template, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(templatePath)))
	if err != nil {
		t.Fatalf("reading %s: %v", templatePath, err)
	}

	change, err := prhygiene.Read(response, string(template))
	if err != nil {
		t.Fatalf("judging %s: %v", fetched, err)
	}

	judged := prhygiene.Judge(change)
	t.Logf("pull request #%d: %d file(s), %d commit(s), %d closing issue(s), declared scope %q",
		judged.Number, judged.Files, judged.Commits, judged.ClosingIssues,
		strings.Join(judged.DeclaredScope, " "))

	for _, note := range judged.NotDecided {
		t.Logf("not decided: %s", note)
	}

	for _, f := range judged.Findings {
		t.Errorf("%s at %s: %s\n  %s", f.RuleID, f.Where, f.Detail, prhygiene.Reason(f.RuleID))
	}
}

// TestTheTemplateCarriesTheSectionTheRuleReads holds the one thing the means
// rule depends on outside its own package. A template reworded so that no
// heading names the means turns that rule into one that refuses every pull
// request, which is a failure worth meeting here rather than on somebody's
// change.
func TestTheTemplateCarriesTheSectionTheRuleReads(t *testing.T) {
	root, err := doclint.ModuleRoot()
	if err != nil {
		t.Fatalf("finding the module root: %v", err)
	}
	template, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(templatePath)))
	if err != nil {
		t.Fatalf("reading %s: %v", templatePath, err)
	}

	filled := clean()
	filled.Template = string(template)
	filled.Body = strings.Replace(filled.Body, meansHeading, "## The means, and why it fits", 1)

	if got := ids(prhygiene.Judge(filled).Findings); len(got) > 0 {
		t.Errorf("a body answering the tracked template's means section is refused by %v, "+
			"so either the template's heading no longer names the means or the section rule stopped finding it", got)
	}
}

// The fixture below is a pull request the table permits, and every case after it
// is one change away from it. The template is stated here rather than read from
// the tree so that a fixture proves the rule rather than the state of a file
// somebody may reword tomorrow; the test above is what holds the two together.
const meansHeading = "## The means, and why it fits"

const fixtureTemplate = `Say what changed and what failure it prevents.

## The issue this closes

Closes #

## The means, and why it fits

One sentence naming what this is made of and why that suits the job.

## The evidence

Paste the command and its output.
`

func clean() prhygiene.Change {
	return prhygiene.Change{
		Number: 1,
		Body: `A guard over the shape of a pull request.

## The issue this closes

Closes #93

## The means, and why it fits

One sentence naming what this is made of and why that suits the job.
Go, because the rule reads a fetched file offline and the suite already runs here.

## The evidence

    go test ./test/prhygiene/ -count=1
    ok  	github.com/iderex/relais/test/prhygiene	0.004s
`,
		Commits: []prhygiene.Commit{{
			SHA:     "0123456789abcdef",
			Message: "Refuse a pull request whose shape nobody filled in\n\nThe body of the message, after a blank line.",
		}},
		Files:    []string{".github/workflows/pr-hygiene.yml"},
		Closes:   []prhygiene.Issue{{Number: 93, Body: "Scope: .github/workflows/\n\nThe body of the issue."}},
		Template: fixtureTemplate,
	}
}

// TestTheCleanFixtureIsPermitted is the baseline every case below is measured
// against. Without it a rule that refused everything would look like a rule that
// bites.
func TestTheCleanFixtureIsPermitted(t *testing.T) {
	judged := prhygiene.Judge(clean())
	if got := ids(judged.Findings); len(got) > 0 {
		t.Errorf("the clean fixture is refused by %v and it is meant to pass", got)
	}
	if len(judged.NotDecided) != 0 {
		t.Errorf("the clean fixture left %v undecided, so a rule passed by having no subject", judged.NotDecided)
	}
}

// TestEachRuleBites puts a fixture through the table that breaks exactly one
// property, and asserts the exact set of rule ids refused. Comparing the set
// rather than asserting that something failed is what stops a case from passing
// for the wrong rule's reason.
func TestEachRuleBites(t *testing.T) {
	for _, c := range []struct {
		name    string
		change  func(prhygiene.Change) prhygiene.Change
		refused []string
	}{
		{
			name: "a body that closes nothing",
			change: func(c prhygiene.Change) prhygiene.Change {
				c.Closes = nil
				return c
			},
			refused: []string{"body-names-a-closing-issue"},
		},
		{
			name: "a means section left exactly as the template wrote it",
			change: func(c prhygiene.Change) prhygiene.Change {
				c.Body = strings.Replace(c.Body,
					"Go, because the rule reads a fetched file offline and the suite already runs here.\n", "", 1)
				return c
			},
			refused: []string{"means-section-is-filled-in"},
		},
		{
			name: "a body carrying no means section at all",
			change: func(c prhygiene.Change) prhygiene.Change {
				c.Body = "Closes #93\n\n## The evidence\n\n    go test ./test/prhygiene/ -count=1\n    ok\n"
				return c
			},
			refused: []string{"means-section-is-filled-in"},
		},
		{
			name: "a block of figures with no command above it",
			change: func(c prhygiene.Change) prhygiene.Change {
				c.Body = strings.Replace(c.Body,
					"    go test ./test/prhygiene/ -count=1\n", "", 1)
				return c
			},
			refused: []string{"evidence-carries-its-command"},
		},
		{
			name: "a commit message whose body runs into its subject",
			change: func(c prhygiene.Change) prhygiene.Change {
				c.Commits[0].Message = "Refuse a pull request whose shape nobody filled in\nThe body, with no blank line."
				return c
			},
			refused: []string{"commit-message-has-a-subject-and-a-break"},
		},
		{
			name: "a commit message with no subject",
			change: func(c prhygiene.Change) prhygiene.Change {
				c.Commits[0].Message = "\nEverything is in the body."
				return c
			},
			refused: []string{"commit-message-has-a-subject-and-a-break"},
		},
		{
			name: "one file outside the scope the closing issue declares",
			change: func(c prhygiene.Change) prhygiene.Change {
				c.Files = append(c.Files, "internal/api/router.go")
				return c
			},
			refused: []string{"change-is-inside-the-scope-its-issue-declares"},
		},
	} {
		t.Run(c.name, func(t *testing.T) {
			// The fixture is rebuilt per case rather than shared,
			// because two of these mutate a slice element and a
			// shared fixture would carry the previous case's
			// damage into the next one.
			got := ids(prhygiene.Judge(c.change(clean())).Findings)
			want := append([]string(nil), c.refused...)
			sort.Strings(want)
			if !reflect.DeepEqual(got, want) {
				t.Errorf("refused %v, expected exactly %v", got, want)
			}
		})
	}
}

// TestTheNearMissesArePermitted is the other half of the proof. Each of these is
// the one-character neighbour of a case above, and a rule that refused any of
// them would be a rule refusing the shape rather than the defect.
func TestTheNearMissesArePermitted(t *testing.T) {
	for _, c := range []struct {
		name   string
		change func(prhygiene.Change) prhygiene.Change
	}{
		{
			name: "a commit message that is a subject and nothing else",
			change: func(c prhygiene.Change) prhygiene.Change {
				c.Commits[0].Message = "Refuse a pull request whose shape nobody filled in"
				return c
			},
		},
		{
			name: "a block with no figure in it and no command either",
			change: func(c prhygiene.Change) prhygiene.Change {
				c.Body = strings.Replace(c.Body,
					"    go test ./test/prhygiene/ -count=1\n    ok  	github.com/iderex/relais/test/prhygiene	0.004s\n",
					"    a list of names\n    and another one\n", 1)
				return c
			},
		},
		{
			name: "a file that is the scope entry itself rather than under it",
			change: func(c prhygiene.Change) prhygiene.Change {
				c.Closes[0].Body = "Scope: README.md"
				c.Files = []string{"README.md"}
				return c
			},
		},
		{
			name: "a scope of the whole repository",
			change: func(c prhygiene.Change) prhygiene.Change {
				c.Closes[0].Body = "Scope: ."
				c.Files = []string{"anything/at/all.go", "README.md"}
				return c
			},
		},
		{
			name: "two scope entries on one line, one of which admits the path",
			change: func(c prhygiene.Change) prhygiene.Change {
				c.Closes[0].Body = "Scope: .github/workflows/, test/"
				c.Files = []string{".github/workflows/pr-hygiene.yml", "test/prhygiene/prhygiene.go"}
				return c
			},
		},
		{
			name: "a means section carrying the template text and an answer beside it",
			change: func(c prhygiene.Change) prhygiene.Change {
				return c
			},
		},
	} {
		t.Run(c.name, func(t *testing.T) {
			if got := ids(prhygiene.Judge(c.change(clean())).Findings); len(got) > 0 {
				t.Errorf("refused %v, and this case is meant to pass", got)
			}
		})
	}
}

// TestAPrefixIsNotAParent is the near miss the scope rule is most likely to get
// wrong, and it is worth its own test because both directions of it are silent
// failures: a rule comparing raw prefixes admits testdata/ under a scope of
// test, and one comparing only whole segments refuses a file that is the scope
// entry itself.
func TestAPrefixIsNotAParent(t *testing.T) {
	c := clean()
	c.Closes[0].Body = "Scope: test"
	c.Files = []string{"testdata/line-endings.golden"}

	got := ids(prhygiene.Judge(c).Findings)
	want := []string{"change-is-inside-the-scope-its-issue-declares"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("refused %v, expected exactly %v: testdata/ is not under a scope of test", got, want)
	}
}

// TestTheScopeComparisonSaysWhenItWasNotMade holds the difference between a rule
// that passed and a rule that had nothing to read. An issue writing its scope as
// prose declares none, and a report that recorded that as a pass would be the
// failure this project's own rule about negative disclosures is against.
func TestTheScopeComparisonSaysWhenItWasNotMade(t *testing.T) {
	c := clean()
	c.Closes[0].Body = "The scope of this issue is everything under the workflows directory."
	c.Files = []string{"internal/api/router.go"}

	judged := prhygiene.Judge(c)
	if got := ids(judged.Findings); len(got) > 0 {
		t.Errorf("refused %v, and an issue declaring no scope is not a change outside one", got)
	}
	if len(judged.NotDecided) != 1 || !strings.Contains(judged.NotDecided[0], "no path comparison was made") {
		t.Errorf("the report says %v, and it has to say that the comparison was not made", judged.NotDecided)
	}
}

// TestTheReadingFailsClosed covers the three shapes in which a pass would mean
// the fetch never happened rather than that the change is clean. Each is an
// error rather than an empty verdict.
func TestTheReadingFailsClosed(t *testing.T) {
	for _, c := range []struct {
		name     string
		response string
		says     string
	}{
		{
			name:     "a response carrying no pull request",
			response: `{"data":{"repository":{"pullRequest":null}}}`,
			says:     "carries no pull request",
		},
		{
			name: "a response carrying no commit",
			response: `{"data":{"repository":{"pullRequest":{"number":1,"body":"b",
				"closingIssuesReferences":{"nodes":[],"pageInfo":{"hasNextPage":false}},
				"commits":{"nodes":[],"pageInfo":{"hasNextPage":false}},
				"files":{"nodes":[{"path":"a.go"}],"pageInfo":{"hasNextPage":false}}}}}}`,
			says: "carries no commit",
		},
		{
			name: "a response with another page of files",
			response: `{"data":{"repository":{"pullRequest":{"number":1,"body":"b",
				"closingIssuesReferences":{"nodes":[],"pageInfo":{"hasNextPage":false}},
				"commits":{"nodes":[{"commit":{"oid":"abc","message":"m"}}],"pageInfo":{"hasNextPage":false}},
				"files":{"nodes":[{"path":"a.go"}],"pageInfo":{"hasNextPage":true}}}}}}`,
			says: "another page of files",
		},
		{
			name:     "a response that is not the shape of a response",
			response: `not json at all`,
			says:     "reading the fetched pull request",
		},
	} {
		t.Run(c.name, func(t *testing.T) {
			_, err := prhygiene.Read([]byte(c.response), fixtureTemplate)
			if err == nil {
				t.Fatalf("read this without an error, and a verdict on it would be a verdict on nothing")
			}
			if !strings.Contains(err.Error(), c.says) {
				t.Errorf("the error is %q and it has to say %q", err, c.says)
			}
		})
	}
}

// TestAWholeResponseIsRead is the other direction of the test above: a fetch
// that returned everything is read into the view the rules judge, so a reading
// that failed closed on everything would not pass this.
func TestAWholeResponseIsRead(t *testing.T) {
	const response = `{"data":{"repository":{"pullRequest":{"number":167,"body":"Closes #93",
		"closingIssuesReferences":{"nodes":[{"number":93,"body":"Scope: ."}],"pageInfo":{"hasNextPage":false}},
		"commits":{"nodes":[{"commit":{"oid":"0123456789abcdef","message":"A subject"}}],"pageInfo":{"hasNextPage":false}},
		"files":{"nodes":[{"path":".github/workflows/pr-hygiene.yml"}],"pageInfo":{"hasNextPage":false}}}}}}`

	change, err := prhygiene.Read([]byte(response), fixtureTemplate)
	if err != nil {
		t.Fatalf("reading a whole response: %v", err)
	}
	if change.Number != 167 || len(change.Commits) != 1 || len(change.Files) != 1 || len(change.Closes) != 1 {
		t.Fatalf("read #%d with %d commit(s), %d file(s) and %d closing issue(s)",
			change.Number, len(change.Commits), len(change.Files), len(change.Closes))
	}
	if change.Commits[0].SHA != "0123456789abcdef" {
		t.Errorf("the commit is %q", change.Commits[0].SHA)
	}
}

// TestTheSameInputGivesTheSameVerdict is the determinism the issue asks for,
// held rather than asserted. The failure it is against is a rule whose output
// order comes from a map, which is stable within one run and different between
// two, so a gate would red on a rerun of the same change and nobody would know
// why.
func TestTheSameInputGivesTheSameVerdict(t *testing.T) {
	broken := clean()
	broken.Closes = nil
	broken.Files = []string{"z/last.go", "a/first.go", "m/middle.go"}
	broken.Commits[0].Message = "A subject\nand no break"
	broken.Body = "## The means, and why it fits\n\n    41\n    42\n"

	first := prhygiene.Judge(broken)
	for i := 0; i < 32; i++ {
		again := prhygiene.Judge(broken)
		if !reflect.DeepEqual(first, again) {
			t.Fatalf("run %d differs from the first: %+v against %+v", i, again, first)
		}
	}
	if len(first.Findings) == 0 {
		t.Fatal("this fixture refuses nothing, so the comparison above compared two empty verdicts")
	}
}

// TestEveryRuleCarriesItsReason holds the property that makes a failure useful.
// A rule id with no argument behind it sends a reader to a gate with no author.
func TestEveryRuleCarriesItsReason(t *testing.T) {
	for _, r := range prhygiene.Rules {
		if strings.TrimSpace(r.Reason) == "" {
			t.Errorf("rule %s carries no reason", r.ID)
		}
		if prhygiene.Reason(r.ID) != r.Reason {
			t.Errorf("rule %s is not reachable by its id", r.ID)
		}
	}
}

// ids is the set of rule ids a verdict refused, sorted and deduplicated, which
// is what makes an assertion about a verdict an assertion about which rules bit
// rather than about how many findings one of them produced.
func ids(found []prhygiene.Finding) []string {
	seen := map[string]bool{}
	for _, f := range found {
		seen[f.RuleID] = true
	}
	var out []string
	for id := range seen {
		out = append(out, id)
	}
	sort.Strings(out)
	return out
}
