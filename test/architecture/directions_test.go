package architecture_test

import (
	"strings"
	"testing"

	"github.com/iderex/relais/test/architecture"
)

// TestTheTreeObeysTheDependencyDirections is the guard itself. Every import in
// every Go file in this repository is put through the table, and a refusal is
// reported with the file it is in and the sentence from docs/architecture.md
// that refuses it, so a failure sends the reader to the argument rather than to
// a rule with no author.
func TestTheTreeObeysTheDependencyDirections(t *testing.T) {
	root, err := architecture.ModuleRoot()
	if err != nil {
		t.Fatalf("finding the module root: %v", err)
	}

	imports, err := architecture.ReadImports(root)
	if err != nil {
		t.Fatalf("reading the tree at %s: %v", root, err)
	}

	// A pass over nothing is not a pass. The scan is reported and the anchor
	// below is asserted, so a walk that silently stopped finding Go files
	// fails here rather than going green with no subject.
	packages := map[string]int{}
	for _, imp := range imports {
		packages[imp.Package]++
	}
	t.Logf("%d imports across %d packages under %s", len(imports), len(packages), root)

	const anchor = "internal/mediaplane"
	if packages[anchor] == 0 {
		t.Fatalf("no import was read from %s, so this run judged a tree it did not find; "+
			"if that package has legitimately gone, move this anchor to one that exists rather than deleting it", anchor)
	}

	for _, imp := range imports {
		for _, rule := range architecture.Refusals(imp.Package, imp.Path) {
			t.Errorf("%s imports %q, which docs/architecture.md refuses under %s:\n\t%s",
				imp.File, imp.Path, rule.ID, rule.Reason)
		}
	}
}

// violation is one import somebody could plausibly write, and the rule that has
// to refuse it.
type violation struct {
	why      string
	importer string
	imported string
	rule     string
}

// violations carries one realistic mistake per rule. Realistic is the point: each
// of these is an import a person writes while doing something else, not one
// invented to trip a check.
var violations = []violation{
	{
		why:      "the port reaching for the implementation it exists to hide",
		importer: "internal/mediaplane",
		imported: "github.com/iderex/relais/internal/forwarding",
		rule:     "media-plane-depends-on-neither-side",
	},
	{
		why:      "the port borrowing a type from the layer above to describe a value crossing it",
		importer: "internal/mediaplane",
		imported: "github.com/iderex/relais/internal/orchestration",
		rule:     "media-plane-depends-on-neither-side",
	},
	{
		why:      "the port naming a transport type in the values that cross it",
		importer: "internal/mediaplane",
		imported: "github.com/pion/webrtc/v4",
		rule:     "transport-library-only-below-the-port",
	},
	{
		why:      "the entry point wiring the transport up itself instead of leaving it below the port",
		importer: "cmd/relais",
		imported: "github.com/pion/webrtc/v4",
		rule:     "transport-library-only-below-the-port",
	},
	{
		why:      "the forwarding core emitting an event in the orchestration layer's own vocabulary",
		importer: "internal/forwarding",
		imported: "github.com/iderex/relais/internal/orchestration",
		rule:     "forwarding-core-does-not-reach-upward",
	},
	{
		why:      "an orchestration test reaching past the fake to the real forwarding core",
		importer: "internal/orchestration",
		imported: "github.com/iderex/relais/internal/forwarding",
		rule:     "above-the-port-reaches-the-port-and-nothing-below-it",
	},
	{
		why:      "the API layer calling the forwarding core directly because the port did not carry what it wanted yet",
		importer: "internal/api",
		imported: "github.com/iderex/relais/internal/forwarding",
		rule:     "above-the-port-reaches-the-port-and-nothing-below-it",
	},
	{
		why:      "orchestration reaching for the wire encoding of the events it produces",
		importer: "internal/orchestration",
		imported: "github.com/iderex/relais/internal/api",
		rule:     "orchestration-does-not-know-its-wire-format",
	},
	{
		why:      "the bench importing the thing it is supposed to be measuring from outside",
		importer: "bench",
		imported: "github.com/iderex/relais/internal/orchestration",
		rule:     "bench-measures-a-target-it-is-pointed-at",
	},
	{
		why:      "the service importing a helper that happened to be written in the bench",
		importer: "internal/orchestration",
		imported: "github.com/iderex/relais/bench",
		rule:     "bench-measures-a-target-it-is-pointed-at",
	},
	{
		why:      "a package reusing something the entry point had already wired together",
		importer: "internal/api",
		imported: "github.com/iderex/relais/cmd/relais",
		rule:     "nothing-imports-the-entry-point",
	},
}

// TestEveryDirectionRefusesARealisticViolation is the proof that the table
// bites. Each case is put through the same decision the tree is put through,
// and the rule that has to catch it is named rather than left to whichever one
// happens to fire.
func TestEveryDirectionRefusesARealisticViolation(t *testing.T) {
	for _, v := range violations {
		t.Run(v.rule+"/"+v.importer, func(t *testing.T) {
			refusals := architecture.Refusals(v.importer, v.imported)
			if len(refusals) == 0 {
				t.Fatalf("%s importing %q is permitted and must not be: %s",
					v.importer, v.imported, v.why)
			}
			if !named(refusals, v.rule) {
				t.Errorf("%s importing %q is refused, but not by %s: %s",
					v.importer, v.imported, v.rule, ids(refusals))
			}
		})
	}
}

// TestEveryRuleIsExercisedByAViolation closes the gap the table above cannot
// see. A rule with no case is a rule nothing has ever run, and it would sit in
// the table looking enforced.
func TestEveryRuleIsExercisedByAViolation(t *testing.T) {
	exercised := map[string]bool{}
	for _, v := range violations {
		exercised[v.rule] = true
	}
	for _, rule := range architecture.Rules {
		if !exercised[rule.ID] {
			t.Errorf("no violation exercises %s, so nothing has shown it refuses anything:\n\t%s",
				rule.ID, rule.Reason)
		}
	}
	for id := range exercised {
		if !inTable(id) {
			t.Errorf("a violation names %s, which is not a rule; the case is testing nothing", id)
		}
	}
}

// permitted is the near-miss set: the import one character or one direction away
// from each violation above, which has to stay legal. A table that refused these
// would pass every case above and stop the work instead of the mistake.
var permitted = []violation{
	{
		why:      "the direction the record permits, and the reason the other one is refused",
		importer: "internal/api",
		imported: "github.com/iderex/relais/internal/orchestration",
	},
	{
		why:      "the forwarding core is the one package the transport library belongs in",
		importer: "internal/forwarding",
		imported: "github.com/pion/webrtc/v4",
	},
	{
		why:      "reaching the media plane through the port is what the port is for",
		importer: "internal/orchestration",
		imported: "github.com/iderex/relais/internal/mediaplane",
	},
	{
		why:      "the entry point may import anything under internal/, which is what wiring is",
		importer: "cmd/relais",
		imported: "github.com/iderex/relais/internal/forwarding",
	},
	{
		why:      "no rule here judges the standard library",
		importer: "internal/mediaplane",
		imported: "slices",
	},
	{
		why:      "a package whose name begins with a refused one is a different package, which is the mistake a prefix test without a separator makes",
		importer: "internal/mediaplaneutil",
		imported: "github.com/iderex/relais/internal/api",
	},
	{
		why:      "a library whose path begins with the transport library's is a different library, for the same reason",
		importer: "internal/api",
		imported: "github.com/pion/webrtcutil",
	},
}

func TestThePermittedImportsStayPermitted(t *testing.T) {
	for _, p := range permitted {
		t.Run(p.importer, func(t *testing.T) {
			if refusals := architecture.Refusals(p.importer, p.imported); len(refusals) > 0 {
				t.Errorf("%s importing %q is refused by %s and must not be: %s",
					p.importer, p.imported, ids(refusals), p.why)
			}
		})
	}
}

// TestEveryRuleQuotesTheRecordItEnforces holds the condition that a failing test
// sends the reader to the argument. A rule whose reason is empty or whose
// identifier is missing produces a failure message nobody can act on.
func TestEveryRuleQuotesTheRecordItEnforces(t *testing.T) {
	for i, rule := range architecture.Rules {
		if rule.ID == "" {
			t.Errorf("rule %d has no identifier", i)
		}
		if strings.TrimSpace(rule.Reason) == "" {
			t.Errorf("rule %s quotes nothing from docs/architecture.md", rule.ID)
		}
	}
}

func named(refusals []architecture.Rule, id string) bool {
	for _, r := range refusals {
		if r.ID == id {
			return true
		}
	}
	return false
}

func inTable(id string) bool {
	for _, r := range architecture.Rules {
		if r.ID == id {
			return true
		}
	}
	return false
}

func ids(refusals []architecture.Rule) string {
	names := make([]string, 0, len(refusals))
	for _, r := range refusals {
		names = append(names, r.ID)
	}
	return strings.Join(names, ", ")
}
