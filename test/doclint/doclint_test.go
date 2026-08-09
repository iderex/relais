// relais, a realtime SFU backend for community self-hosters.
// Copyright (C) 2026 Nils Lehnen
//
// Licensed under the GNU Affero General Public License, version 3. See LICENSE
// for the full terms, including the warranty and liability disclaimer.

package doclint_test

import (
	"strings"
	"testing"

	"github.com/iderex/relais/test/doclint"
)

// TestTheTreeSatisfiesEveryRule is the check itself. Everything below it exists
// to show that this one could have failed.
func TestTheTreeSatisfiesEveryRule(t *testing.T) {
	root, err := doclint.ModuleRoot()
	if err != nil {
		t.Fatalf("finding the module root: %v", err)
	}

	docs, err := doclint.ReadDocs(root)
	if err != nil {
		t.Fatalf("reading the documents under %s: %v", root, err)
	}

	// A pass over nothing is not a pass. The scan is reported and the anchor
	// below is asserted, so a walk that silently stopped finding documents
	// fails here rather than going green with no subject.
	var links, blocks, paths int
	seen := map[string]bool{}
	for _, d := range docs {
		seen[d.Path] = true
		links += len(d.Links)
		blocks += len(d.Blocks)
		paths += len(d.Paths)
	}
	t.Logf("%d documents, %d links, %d blocks, %d documented paths under %s",
		len(docs), links, blocks, paths, root)

	const anchor = "docs/gate-parity.md"
	if !seen[anchor] {
		t.Fatalf("no document was read at %s, so this run judged a tree it did not find; "+
			"if that file has legitimately gone, move this anchor to one that exists rather than deleting it", anchor)
	}

	tree := doclint.OnDisk{Root: root}
	for _, d := range docs {
		for _, f := range doclint.Refusals(d, tree) {
			t.Errorf("%s:%d refused under %s: %s\n\t%s",
				f.File, f.Line, f.RuleID, f.Detail, doclint.Reason(f.RuleID))
		}
	}
}

// fakeTree is a stated tree rather than the disk, so a near miss is judged
// against exactly the files it names and a fixture cannot pass because something
// unrelated happens to exist.
type fakeTree struct {
	files    map[string]bool
	headings map[string][]string
}

func (f fakeTree) Exists(rel string) bool { return f.files[rel] }

func (f fakeTree) Headings(rel string) ([]string, bool) {
	slugs, ok := f.headings[rel]
	return slugs, ok
}

var tree = fakeTree{
	files: map[string]bool{
		"docs":                          true,
		"docs/architecture.md":          true,
		"docs/decisions":                true,
		"docs/decisions/seam.md":        true,
		"internal":                      true,
		"internal/mediaplane":           true,
		"internal/mediaplane/domain.go": true,
		"README.md":                     true,
	},
	headings: map[string][]string{
		"docs/architecture.md":   {"the-layout", "what-imports-what"},
		"docs/decisions/seam.md": {"the-seam"},
	},
}

// A nearMiss is the one-character mistake somebody will actually make, beside
// the version of the same document that is correct. Both halves are asserted:
// a rule that refused the neighbour too would be refusing the shape rather than
// the mistake, and nothing in the pair would say so.
type nearMiss struct {
	rule      string
	why       string
	path      string
	refused   string
	permitted string
}

var nearMisses = []nearMiss{
	{
		rule:      "link-resolves",
		why:       "A file was renamed and one link was left behind. Nothing in the sentence is wrong, which is why nobody reads it again.",
		path:      "docs/threat-model.md",
		refused:   "See [the layout](architecture.mb) for the directions.",
		permitted: "See [the layout](architecture.md) for the directions.",
	},
	{
		rule:      "link-resolves",
		why:       "A link written from the wrong depth. One extra step up resolves outside the tree and the target is real, so the mistake is invisible in the sentence.",
		path:      "docs/decisions/seam.md",
		refused:   "The [architecture note](../../../docs/architecture.md) holds the directions.",
		permitted: "The [architecture note](../architecture.md) holds the directions.",
	},
	{
		rule:      "link-fragment-resolves",
		why:       "The section was retitled. The file is still there and the link still works, so the reader lands at the top of the page instead of at the paragraph.",
		path:      "docs/threat-model.md",
		refused:   "Read [the layout](architecture.md#the-layouts) first.",
		permitted: "Read [the layout](architecture.md#the-layout) first.",
	},
	{
		rule:      "documented-path-resolves",
		why:       "A package named in prose with the wrong spelling. It is the form most of this tree's statements about itself take, and no link checker would look at it.",
		path:      "docs/architecture.md",
		refused:   "The port is declared in `internal/mediaplain/domain.go`.",
		permitted: "The port is declared in `internal/mediaplane/domain.go`.",
	},
	{
		rule:      "evidence-carries-its-command",
		why:       "A figure pasted without the command that produced it, which is the rule this project states about its own evidence and the shape a reader is most likely to believe.",
		path:      "docs/gate-parity.md",
		refused:   "The suite is fast:\n\n    unit suite wall clock: 7s\n",
		permitted: "The suite is fast:\n\n    go test ./... -count=1\n    unit suite wall clock: 7s\n",
	},
	{
		rule:      "line-ends-in-no-whitespace",
		why:       "Two spaces left at the end of a line, which the renderer reads as a line break and no diff shows.",
		path:      "docs/architecture.md",
		refused:   "The port is the boundary.  \nEverything above it is orchestration.\n",
		permitted: "The port is the boundary.\nEverything above it is orchestration.\n",
	},
	{
		rule:      "fence-is-closed",
		why:       "A fence opened and never closed, which swallows every section after it into one block.",
		path:      "CONTRIBUTING.md",
		refused:   "Run it:\n\n```\ngo test ./...\n\n## The next section\n",
		permitted: "Run it:\n\n```\ngo test ./...\n```\n\n## The next section\n",
	},
	{
		rule:      "command-closes-its-quotes",
		why:       "One quote dropped from a --jq expression. The line reads correctly and a reader who copies it gets a shell waiting for the rest of it.",
		path:      "docs/gate-parity.md",
		refused:   "    gh api repos/iderex/relais --jq '.name\n",
		permitted: "    gh api repos/iderex/relais --jq '.name'\n",
	},
}

func TestEveryRuleRefusesARealisticMistake(t *testing.T) {
	for _, m := range nearMisses {
		t.Run(m.rule+": "+m.why, func(t *testing.T) {
			refused := doclint.Refusals(doclint.ParseDoc(m.path, m.refused), tree)
			if !refusedUnder(refused, m.rule) {
				t.Errorf("the mistake was permitted, so %s does not refuse what it is written for.\n\tdocument: %q\n\tfindings: %v",
					m.rule, m.refused, refused)
			}

			permitted := doclint.Refusals(doclint.ParseDoc(m.path, m.permitted), tree)
			if len(permitted) != 0 {
				t.Errorf("the corrected neighbour was refused as well, so the rule is judging the shape rather than the mistake.\n\tdocument: %q\n\tfindings: %v",
					m.permitted, permitted)
			}
		})
	}
}

func refusedUnder(findings []doclint.Finding, rule string) bool {
	for _, f := range findings {
		if f.RuleID == rule {
			return true
		}
	}
	return false
}

// TestAPathSomewhereElseIsNotJudged is the bound on the path rule, asserted
// rather than described. A module path and a repository on the hosting site both
// look exactly like a path in this tree and neither is a claim about it.
func TestAPathSomewhereElseIsNotJudged(t *testing.T) {
	source := "The transport library is `github.com/pion/webrtc/v4` and the reference gate is at `iderex/jellyfin-plugin-sso`."
	if findings := doclint.Refusals(doclint.ParseDoc("docs/architecture.md", source), tree); len(findings) != 0 {
		t.Errorf("a path outside this tree was judged as one inside it: %v", findings)
	}
}

// TestATaggedBlockIsSourceRatherThanEvidence holds the other bound the evidence
// rule needs. A Go snippet is full of digits and opens with none of the words in
// the command vocabulary.
func TestATaggedBlockIsSourceRatherThanEvidence(t *testing.T) {
	source := "An example:\n\n```go\nconst budget = 60\n```\n"
	if findings := doclint.Refusals(doclint.ParseDoc("CONTRIBUTING.md", source), tree); len(findings) != 0 {
		t.Errorf("a tagged block was judged as evidence: %v", findings)
	}
}

// TestAHeadingInsideAFenceIsNotAHeading is the case a reader of lines alone gets
// wrong: a shell comment at the start of a line inside a block is not a section
// of the document, and a link pointing at it should not resolve.
func TestAHeadingInsideAFenceIsNotAHeading(t *testing.T) {
	source := "# The real heading\n\n```\n# not a heading\n```\n"
	slugs := doclint.HeadingsIn(source)
	if len(slugs) != 1 || slugs[0] != "the-real-heading" {
		t.Errorf("headings read as %v, want exactly the one outside the block", slugs)
	}
}

// TestSlugMatchesTheHostingSite fixes the derivation the fragment rule depends
// on. A slug derived differently from the one a reader clicks would make the rule
// refuse working links and accept broken ones, in both directions at once.
func TestSlugMatchesTheHostingSite(t *testing.T) {
	for heading, want := range map[string]string{
		"What is out of scope, and why":   "what-is-out-of-scope-and-why",
		"The three rules":                 "the-three-rules",
		"`ops verify` and what it prints": "ops-verify-and-what-it-prints",
		"Growing past one host":           "growing-past-one-host",
	} {
		if got := doclint.Slug(heading); got != want {
			t.Errorf("Slug(%q) = %q, want %q", heading, got, want)
		}
	}
}

// TestAnUnclosedQuoteIsFoundAndAnEscapedOneIsNot separates the two cases the
// lexer exists to tell apart, because a check that reported an escaped quote as
// unclosed would be refused by whoever hit it first and then switched off.
func TestAnUnclosedQuoteIsFoundAndAnEscapedOneIsNot(t *testing.T) {
	source := "    echo \"a \\\" inside a string\"\n    gh api repos/iderex/relais --jq '\"\\(.name)\"'\n"
	if findings := doclint.Refusals(doclint.ParseDoc("docs/architecture.md", source), tree); len(findings) != 0 {
		t.Errorf("an escaped quote was reported as unclosed: %v", findings)
	}
	if !strings.Contains(source, "\\\"") {
		t.Fatal("the fixture lost its escaped quote, so this test proves nothing")
	}
}
