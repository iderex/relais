// relais, a realtime SFU backend for community self-hosters.
// Copyright (C) 2026 Nils Lehnen
//
// Licensed under the GNU Affero General Public License, version 3. See LICENSE
// for the full terms, including the warranty and liability disclaimer.

package scanseverity_test

import (
	"os"
	"strings"
	"testing"

	"github.com/iderex/relais/test/scanseverity"
)

// analysisOutput is where the code scanning workflow puts the file the analyser
// wrote, so this suite can judge the real analysis in the job that produced it.
// It is empty everywhere else, and the test that reads it says so rather than
// passing quietly, because a gate that judged nothing and a gate that judged
// everything and was happy exit the same way.
const analysisOutput = "RELAIS_ANALYSIS_SARIF"

// TestTheAnalysisCarriesNothingAtOrAboveTheThreshold is the check itself.
// Everything below it exists to show that this one could have failed.
func TestTheAnalysisCarriesNothingAtOrAboveTheThreshold(t *testing.T) {
	path := os.Getenv(analysisOutput)
	if path == "" {
		t.Skipf("%s names no file, so no analysis was judged here; the code scanning job sets it "+
			"to what the analyser wrote and this test is the gate there", analysisOutput)
	}

	sarif, err := os.ReadFile(path)
	if err != nil {
		// The working directory is in the message because a test binary
		// runs in its own package directory rather than where the job
		// that set the variable stood, so a path that was right on the
		// command line resolves somewhere else here. That is how the
		// first run of this gate failed, and the message is what turns
		// it from a puzzle into one line.
		where, _ := os.Getwd()
		t.Fatalf("reading the analysis output at %s from %s: %v", path, where, err)
	}

	judged, err := scanseverity.Judge(sarif, scanseverity.Threshold)
	if err != nil {
		t.Fatalf("judging %s: %v", path, err)
	}

	t.Logf("%d rule(s), %d of them scored, %d result(s), highest severity %.1f, threshold %.1f",
		judged.Rules, judged.Scored, judged.Results, judged.Highest, scanseverity.Threshold)

	for _, id := range judged.Unscored {
		t.Logf("rule %s raised a finding and carries no security severity, "+
			"so this threshold has no opinion about it and it is reported here instead", id)
	}

	for _, f := range judged.Refused {
		where := f.Location
		if where == "" {
			where = "no location in the analysis output"
		}
		t.Errorf("%s scored %.1f at %s, which is at or above the threshold of %.1f: %s",
			f.RuleID, f.Severity, where, scanseverity.Threshold, f.Message)
	}
}

// The fixtures below carry the shape the analyser actually writes rather than a
// shape that would be convenient to read. It was taken from a real analysis of
// this repository:
//
//	gh api -H "Accept: application/sarif+json" repos/iderex/relais/code-scanning/analyses/1591715013
//
// Three things in that file decide the shape here. The rules sit on an extension
// of the tool rather than on its driver, the severity is a string rather than a
// number, and a result names its rule by id rather than by index.
const atTheThreshold = `{
  "runs": [{
    "tool": {
      "driver": {"name": "CodeQL"},
      "extensions": [{
        "name": "codeql/go-queries",
        "rules": [{"id": "go/clear-text-logging", "properties": {"security-severity": "7.0"}}]
      }]
    },
    "results": [{
      "ruleId": "go/clear-text-logging",
      "message": {"text": "This logs sensitive data."},
      "locations": [{"physicalLocation": {
        "artifactLocation": {"uri": "internal/api/log.go"},
        "region": {"startLine": 41}
      }}]
    }]
  }]
}`

// One hundredth below, which is the near miss worth spending the fixture on. A
// threshold that admitted this and refused 7.0 would be drawing the line
// somewhere the severity scale does not.
const belowTheThreshold = `{
  "runs": [{
    "tool": {
      "driver": {"name": "CodeQL"},
      "extensions": [{
        "name": "codeql/go-queries",
        "rules": [{"id": "go/bad-redirect-check", "properties": {"security-severity": "6.9"}}]
      }]
    },
    "results": [{"ruleId": "go/bad-redirect-check", "message": {"text": "A bad redirect check."}}]
  }]
}`

func TestAFindingAtTheThresholdIsRefused(t *testing.T) {
	judged, err := scanseverity.Judge([]byte(atTheThreshold), scanseverity.Threshold)
	if err != nil {
		t.Fatalf("judging the fixture: %v", err)
	}
	if len(judged.Refused) != 1 {
		t.Fatalf("a finding scored exactly at the threshold produced %d refusal(s), want 1", len(judged.Refused))
	}
	if judged.Refused[0].Location != "internal/api/log.go:41" {
		t.Errorf("the refusal is located at %q, want internal/api/log.go:41; a reader meeting this failure "+
			"has to be able to open the file it is about", judged.Refused[0].Location)
	}
	if judged.Refused[0].Message == "" {
		t.Error("the refusal carries no message from the analyser, which is the half of it that says what is wrong")
	}
}

func TestAFindingBelowTheThresholdIsNotRefused(t *testing.T) {
	judged, err := scanseverity.Judge([]byte(belowTheThreshold), scanseverity.Threshold)
	if err != nil {
		t.Fatalf("judging the fixture: %v", err)
	}
	if len(judged.Refused) != 0 {
		t.Fatalf("a finding one hundredth below the threshold produced %d refusal(s), want 0", len(judged.Refused))
	}
	if judged.Results != 1 {
		t.Errorf("the finding was not read at all: %d result(s) counted, want 1. A rule that refuses nothing "+
			"because it saw nothing is not the same as one that looked and was satisfied", judged.Results)
	}
	if judged.Highest != 6.9 {
		t.Errorf("the highest severity read is %.1f, want 6.9", judged.Highest)
	}
}

func TestAFindingAboveTheThresholdIsRefused(t *testing.T) {
	above := strings.ReplaceAll(atTheThreshold, `"7.0"`, `"9.8"`)
	judged, err := scanseverity.Judge([]byte(above), scanseverity.Threshold)
	if err != nil {
		t.Fatalf("judging the fixture: %v", err)
	}
	if len(judged.Refused) != 1 || judged.Refused[0].Severity != 9.8 {
		t.Fatalf("a finding in the critical band produced %+v, want one refusal scored 9.8", judged.Refused)
	}
}

func TestAnAnalysisWithNoFindingIsNotRefused(t *testing.T) {
	clean := `{
  "runs": [{
    "tool": {
      "driver": {"name": "CodeQL"},
      "extensions": [{
        "name": "codeql/go-queries",
        "rules": [{"id": "go/clear-text-logging", "properties": {"security-severity": "7.5"}}]
      }]
    },
    "results": []
  }]
}`
	judged, err := scanseverity.Judge([]byte(clean), scanseverity.Threshold)
	if err != nil {
		t.Fatalf("judging the fixture: %v", err)
	}
	if len(judged.Refused) != 0 || judged.Results != 0 {
		t.Fatalf("an analysis carrying no finding produced %d refusal(s) over %d result(s), want 0 over 0",
			len(judged.Refused), judged.Results)
	}
	if judged.Scored != 1 {
		t.Errorf("%d scored rule(s) read, want 1; the threshold has to have something to judge against "+
			"before an empty result set means anything", judged.Scored)
	}
}

// A rule on the driver rather than on an extension. The analyser puts them on an
// extension today and this reads both, because which of the two carries them is
// the analyser's arrangement of its own query packs rather than a promise to
// this repository.
func TestARuleOnTheDriverIsRead(t *testing.T) {
	onTheDriver := `{
  "runs": [{
    "tool": {"driver": {
      "name": "CodeQL",
      "rules": [{"id": "go/command-injection", "properties": {"security-severity": "9.8"}}]
    }},
    "results": [{"ruleId": "go/command-injection", "message": {"text": "Injected."}}]
  }]
}`
	judged, err := scanseverity.Judge([]byte(onTheDriver), scanseverity.Threshold)
	if err != nil {
		t.Fatalf("judging the fixture: %v", err)
	}
	if len(judged.Refused) != 1 {
		t.Fatalf("a rule declared on the driver produced %d refusal(s), want 1", len(judged.Refused))
	}
}

// A finding whose rule carries no severity cannot be judged by a severity
// threshold, and it is not silently a pass either. It is named in the report so
// that the one thing this rule cannot decide is visible in the run rather than
// only in this comment.
func TestAFindingWithNoSeverityIsReportedRatherThanPassed(t *testing.T) {
	mixed := `{
  "runs": [{
    "tool": {"driver": {
      "name": "CodeQL",
      "rules": [
        {"id": "go/command-injection", "properties": {"security-severity": "9.8"}},
        {"id": "go/unused-something", "properties": {}}
      ]
    }},
    "results": [{"ruleId": "go/unused-something", "message": {"text": "Not a security query."}}]
  }]
}`
	judged, err := scanseverity.Judge([]byte(mixed), scanseverity.Threshold)
	if err != nil {
		t.Fatalf("judging the fixture: %v", err)
	}
	if len(judged.Refused) != 0 {
		t.Fatalf("a finding with no severity produced %d refusal(s), want 0", len(judged.Refused))
	}
	if len(judged.Unscored) != 1 || judged.Unscored[0] != "go/unused-something" {
		t.Fatalf("the unjudged finding is reported as %v, want [go/unused-something]", judged.Unscored)
	}
}

// The three shapes in which a green verdict would mean the reading failed rather
// than that the tree is clean. Each is an error, and each is the failure this
// rule is most likely to arrive at years from now, when the analyser changes
// where it writes something.
func TestTheReadingFailsClosed(t *testing.T) {
	cases := []struct {
		name  string
		sarif string
		says  string
	}{
		{
			name:  "not the output of anything",
			sarif: "this is not a SARIF file",
			says:  "reading the analysis output",
		},
		{
			name:  "an output carrying no run",
			sarif: `{"runs": []}`,
			says:  "carries no run",
		},
		{
			name: "an output in which no rule carries a severity",
			sarif: `{"runs": [{"tool": {"driver": {"name": "CodeQL",
				"rules": [{"id": "go/one", "properties": {}}, {"id": "go/two", "properties": {}}]}},
				"results": []}]}`,
			says: "carries a security severity",
		},
		{
			name: "a severity that is not a number",
			sarif: `{"runs": [{"tool": {"driver": {"name": "CodeQL",
				"rules": [{"id": "go/one", "properties": {"security-severity": "high"}}]}},
				"results": []}]}`,
			says: "which is not a number",
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			_, err := scanseverity.Judge([]byte(c.sarif), scanseverity.Threshold)
			if err == nil {
				t.Fatalf("judging %s returned no error, so this shape would pass as a clean analysis", c.name)
			}
			if !strings.Contains(err.Error(), c.says) {
				t.Errorf("the error says %q, which does not name %q; the message is what sends "+
					"whoever meets this to the right repair", err, c.says)
			}
		})
	}
}
