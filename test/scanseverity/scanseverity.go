// relais, a realtime SFU backend for community self-hosters.
// Copyright (C) 2026 Nils Lehnen
//
// Licensed under the GNU Affero General Public License, version 3. See LICENSE
// for the full terms, including the warranty and liability disclaimer.

// Package scanseverity holds the severity at which a finding from the program
// analyser stops a merge, and the reading of the analyser's own output that
// refuses one.
//
// Issue #87 asks for the threshold to be stated with its reason. Stating it in
// prose alone was not available: the conclusion of a code scanning check is
// otherwise decided by a repository setting, which nothing in this tree can read
// and no reader of this repository can see, so a sentence here would have been a
// claim about a switch nobody can check. The threshold below is the whole of the
// rule instead, and the analysis output is judged against it in the same job
// that produced it.
//
// WHAT THE THRESHOLD DECIDES AND WHAT IT DOES NOT. It decides what stops a
// merge. It does not decide what is reported: every finding the analyser raises
// still reaches the code scanning surface, whatever its severity, because the
// upload happens before this reads anything. A finding under the threshold is
// therefore a finding somebody has to deal with rather than one that was thrown
// away, and this package never sees the difference.
//
// The rule reads the analyser's own output rather than asking the code scanning
// API what it made of it. That keeps the decision offline, identical on a fork's
// pull request where the API answer is not available, and reproducible from the
// same file by anybody holding it.
//
// It fails closed in the direction that matters. A file that cannot be parsed, a
// file carrying no run, and a file in which no rule carries a severity at all are
// each an error rather than a clean verdict, because all three are shapes in
// which a green result would mean the join never happened rather than that
// nothing was found.
//
// Nothing here imports anything else in this repository.
package scanseverity

import (
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
)

// Threshold is the security severity at or above which a finding stops a merge.
//
// The analyser scores a finding on the 0.0 to 10.0 scale it publishes as the
// `security-severity` property of the rule that raised it, and 7.0 is the floor
// of the band that scale calls high. That is the value, and this is the reason
// it rather than a lower one:
//
// A threshold set too low produces a backlog and a habit of merging red, which
// is issue #87's own sentence and the failure this number is chosen against. The
// analyser runs the `security-extended` query set on a service that will take
// bytes from strangers, and that set is wider than the default precisely because
// it admits queries that are less sure of themselves. Stopping a merge on every
// one of them, in a tree this early, would teach the first person to meet a
// medium finding that the way past this gate is to merge anyway.
//
// What is bought for that is narrow and worth saying plainly: a high or critical
// finding is one this project has decided is worth a person's attention before a
// merge rather than after. Everything below it is still raised, still visible and
// still somebody's, and the only thing the threshold withholds from it is the
// power to block.
//
// At rather than above. A finding scored exactly 7.0 is in the high band, and a
// boundary that let it through would refuse 7.1 and admit the same class of
// defect one hundredth below it.
const Threshold = 7.0

// A Finding is one result the threshold refuses, located well enough that a
// reader meeting the failure does not have to go and find it.
type Finding struct {
	RuleID   string
	Severity float64
	// Location is the file and line the analyser put the finding at,
	// empty where the result carries no physical location.
	Location string
	Message  string
}

// A Report is what one reading of an analysis produced. The counts are part of
// the result rather than a debugging aid: a run that judged nothing exits the
// same way as a run that judged everything and was happy, and the counts are what
// tell a reader which of the two happened.
type Report struct {
	// Rules is every rule the analyser declared in the file.
	Rules int
	// Scored is how many of those carry a security severity. A rule
	// without one is a query this threshold has no opinion about.
	Scored int
	// Results is every finding in the file, at any severity.
	Results int
	// Unscored names the rules that raised a finding and carry no
	// severity, so a result this threshold could not judge is reported
	// rather than counted as passing.
	Unscored []string
	// Highest is the severity of the worst finding, and zero where there
	// are none.
	Highest float64
	// Refused is every finding at or above the threshold.
	Refused []Finding
}

// Judge reads one SARIF file from the analyser and reports what the threshold
// makes of it.
//
// The severity lives on the rule rather than on the finding, so every result has
// to be joined back to the rule that raised it. The rules arrive in two places in
// the same file, on the driver and on each of its extensions, and this reads both
// rather than the one that happens to be populated today: the query pack that
// carries them is the analyser's own arrangement and is not this project's to
// depend on.
func Judge(sarif []byte, threshold float64) (Report, error) {
	var file report
	if err := json.Unmarshal(sarif, &file); err != nil {
		return Report{}, fmt.Errorf("reading the analysis output: %w", err)
	}
	if len(file.Runs) == 0 {
		return Report{}, fmt.Errorf("the analysis output carries no run, so there is nothing in it to judge")
	}

	var judged Report
	severities := map[string]float64{}
	unscored := map[string]bool{}

	for _, run := range file.Runs {
		components := append([]component{run.Tool.Driver}, run.Tool.Extensions...)
		for _, c := range components {
			for _, rule := range c.Rules {
				judged.Rules++
				if rule.Properties.SecuritySeverity == "" {
					continue
				}
				severity, err := strconv.ParseFloat(rule.Properties.SecuritySeverity, 64)
				if err != nil {
					return Report{}, fmt.Errorf("rule %q carries the severity %q, which is not a number: %w",
						rule.ID, rule.Properties.SecuritySeverity, err)
				}
				judged.Scored++
				severities[rule.ID] = severity
			}
		}
	}

	if judged.Scored == 0 {
		return Report{}, fmt.Errorf(
			"none of the %d rule(s) in the analysis output carries a security severity, "+
				"so no finding could ever be judged against the threshold and a pass here would mean the join failed "+
				"rather than that the tree is clean", judged.Rules)
	}

	for _, run := range file.Runs {
		for _, result := range run.Results {
			judged.Results++
			severity, ok := severities[result.RuleID]
			if !ok {
				unscored[result.RuleID] = true
				continue
			}
			if severity > judged.Highest {
				judged.Highest = severity
			}
			if severity < threshold {
				continue
			}
			judged.Refused = append(judged.Refused, Finding{
				RuleID:   result.RuleID,
				Severity: severity,
				Location: locate(result),
				Message:  result.Message.Text,
			})
		}
	}

	for id := range unscored {
		judged.Unscored = append(judged.Unscored, id)
	}
	sort.Strings(judged.Unscored)
	sort.Slice(judged.Refused, func(i, j int) bool {
		if judged.Refused[i].Severity != judged.Refused[j].Severity {
			return judged.Refused[i].Severity > judged.Refused[j].Severity
		}
		return judged.Refused[i].RuleID < judged.Refused[j].RuleID
	})

	return judged, nil
}

// locate turns the first physical location on a result into the file and line a
// reader would open. A result with none is not an error: some queries answer
// about the build rather than about a place in it.
func locate(r result) string {
	for _, l := range r.Locations {
		uri := l.PhysicalLocation.ArtifactLocation.URI
		if uri == "" {
			continue
		}
		if l.PhysicalLocation.Region.StartLine == 0 {
			return uri
		}
		return fmt.Sprintf("%s:%d", uri, l.PhysicalLocation.Region.StartLine)
	}
	return ""
}

// The shape below is the part of the analysis output this rule reads, and
// nothing else in it. A partial type is deliberate: the file carries the whole
// help text of every query it declares, and a reader that unmarshalled all of it
// would break on the day one of those fields changed shape for a reason that has
// nothing to do with a severity.
type report struct {
	Runs []run `json:"runs"`
}

type run struct {
	Tool    tool     `json:"tool"`
	Results []result `json:"results"`
}

type tool struct {
	Driver     component   `json:"driver"`
	Extensions []component `json:"extensions"`
}

type component struct {
	Name  string `json:"name"`
	Rules []rule `json:"rules"`
}

type rule struct {
	ID         string `json:"id"`
	Properties struct {
		// A string in the file rather than a number, which is the
		// analyser's own encoding and not a mistake to correct here.
		SecuritySeverity string `json:"security-severity"`
	} `json:"properties"`
}

type result struct {
	RuleID  string `json:"ruleId"`
	Message struct {
		Text string `json:"text"`
	} `json:"message"`
	Locations []location `json:"locations"`
}

type location struct {
	PhysicalLocation struct {
		ArtifactLocation struct {
			URI string `json:"uri"`
		} `json:"artifactLocation"`
		Region struct {
			StartLine int `json:"startLine"`
		} `json:"region"`
	} `json:"physicalLocation"`
}
