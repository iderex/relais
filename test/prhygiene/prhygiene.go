// relais, a realtime SFU backend for community self-hosters.
// Copyright (C) 2026 Nils Lehnen
//
// Licensed under the GNU Affero General Public License, version 3. See LICENSE
// for the full terms, including the warranty and liability disclaimer.

// Package prhygiene holds the properties issue #93 fixes about the shape of a
// pull request here, in the form a machine can refuse a violation of.
//
// It judges the shape and never the content. The body names an issue, the means
// section was filled in rather than left as the template wrote it, a block of
// figures carries the command that produced it, every commit message has a
// subject and breaks correctly into a body, and the files touched are inside the
// scope the issue being closed declares. Each of those is decidable by reading a
// fetched view of one pull request and nothing else.
//
// WHAT IT CANNOT DECIDE, said here rather than left to be discovered by somebody
// reading a green result as more than it is. It refuses an empty box; it cannot
// refuse a false statement. A means sentence naming the wrong means passes. An
// evidence block whose command was never run passes, and so does one whose
// output was true a year ago, which is the same gap the documentation gate
// carries and for the same reason. A commit message stating neither what changed
// nor what failure it prevents passes, because that is what CONTRIBUTING.md asks
// for and no reading of the message decides it. Two unrelated topics inside one
// declared scope pass, because the scope rule is the weak test and the strong
// one is about meaning.
//
// The template's other two sections, the evidence and what was not checked, are
// not judged. Issue #93 names five mechanical items and neither is among them,
// and widening a gate past the issue that asked for it is how a check acquires
// rules nobody argued for.
//
// DETERMINISTIC MEANS THE SAME INPUT GIVES THE SAME VERDICT. Everything here is
// a function of the bytes handed to it. Nothing reads a clock, nothing reaches a
// network, nothing consults a model, and the one fetch this check performs
// happens in the job before the judging starts, so the same fetched file judged
// twice on two machines gives the same answer. That is what makes it a gate
// rather than an opinion, and it is the property being copied from the reference
// gate rather than the rules themselves.
//
// It fails closed on a truncated view. A fetch that returned the first page of
// files and stopped would let the scope rule pass a change whose foreign paths
// were on the second page, so a view carrying another page is an error rather
// than a clean verdict.
//
// This package imports test/doclint, which is the one place the convention that
// these suites import nothing else in this tree is departed from. The rule about
// a block of output carrying its command already exists there, with its
// vocabulary and its near misses, and a second copy of it here would drift
// against the first from the day it was written. A pull request body is markdown
// judged by a markdown rule, so the reuse is the rule rather than a shortcut
// past it.
package prhygiene

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/iderex/relais/test/doclint"
)

// A Change is the view of one pull request the rules judge, and the whole of
// what they may read. Anything not in this struct is not available to a rule,
// which is what keeps the verdict a function of one fetched file.
type Change struct {
	// Number is the pull request, for the failure message rather than for
	// any rule.
	Number int
	// Body is the pull request body as written.
	Body string
	// Commits are the commits on the branch, in the order the fetch
	// returned them.
	Commits []Commit
	// Files are the repository-relative paths the change touches.
	Files []string
	// Closes are the issues GitHub itself resolved this pull request as
	// closing, rather than the ones a reader would guess from the body.
	Closes []Issue
	// Template is the tracked pull request template, read from the tree
	// rather than from the fetch, because the comparison is against what
	// the template says today and not against what it said when the
	// author opened the page.
	Template string
}

// A Commit is one commit message. The subject and the body are not split here:
// where the break is wrong is exactly what one of the rules decides, and a
// splitter that had already made that decision would leave nothing to refuse.
type Commit struct {
	// SHA locates the commit in the failure message. Abbreviated by the
	// caller rather than here.
	SHA     string
	Message string
}

// An Issue is a closing issue, carried for the one line of it any rule reads.
type Issue struct {
	Number int
	Body   string
}

// A Finding is one refusal: which rule, where in the change, and what was wrong
// with it. Where is a commit, a file or the body rather than a line number,
// because a pull request is not a file and a line number in a body sends a
// reader to a textarea.
type Finding struct {
	RuleID string
	// Where names the part of the change the finding is about: a commit,
	// a path, or the body.
	Where  string
	Detail string
}

// A Rule is one refusable property of a pull request's shape.
type Rule struct {
	ID     string
	Reason string
	Refuse func(c Change) []Finding
}

// Rules is the whole table, in the order issue #93 lists the mechanical items.
var Rules = []Rule{
	{
		ID: "body-names-a-closing-issue",
		Reason: "A pull request closes an issue, and the linkage GitHub resolved is what says so. This is read from the resolved references rather than by matching text in the body, which is the difference the pull request template already explains: " +
			"a closing keyword inside a code block reads as a link to a person and does nothing on merge, so a finished issue stays open under a merged change pointing straight at it. Matching the text would pass exactly that case. " +
			"A change that closes nothing is also a change nobody agreed to first, which is the rule the contribution guide opens with.",
		Refuse: func(c Change) []Finding {
			if len(c.Closes) > 0 {
				return nil
			}
			return []Finding{{
				RuleID: "body-names-a-closing-issue",
				Where:  "the body",
				Detail: "no issue is resolved as closed by this pull request, so either the body carries no closing keyword or the one it carries is inside a code block",
			}}
		},
	},
	{
		ID: "means-section-is-filled-in",
		Reason: "The section naming what the change is made of and why that suits the job carries something the template did not already say. The contribution rule behind it is that the means is chosen per artefact rather than carried over from the last one, and what is checkable about that is only that the question was asked. " +
			"It is decided by comparison rather than by judgement: the template writes its own instruction text into the section, so a section holding nothing but that text, or nothing at all, is one nobody filled in. Whether the answer is right is a judgement made in review and this rule makes no claim about it.",
		Refuse: func(c Change) []Finding {
			heading, ok := meansHeading(c.Template)
			if !ok {
				return []Finding{{
					RuleID: "means-section-is-filled-in",
					Where:  ".github/PULL_REQUEST_TEMPLATE.md",
					Detail: "the template carries no means section, so this rule has nothing to compare a body against and refuses rather than passing every pull request quietly",
				}}
			}
			written, present := section(c.Body, heading)
			if !present {
				return []Finding{{
					RuleID: "means-section-is-filled-in",
					Where:  "the body",
					Detail: fmt.Sprintf("the body carries no %q section, which the template asks for", heading),
				}}
			}
			boilerplate, _ := section(c.Template, heading)
			if len(added(written, boilerplate)) > 0 {
				return nil
			}
			return []Finding{{
				RuleID: "means-section-is-filled-in",
				Where:  "the body",
				Detail: fmt.Sprintf("the %q section carries no line the template did not already write, so the question it asks was not answered", heading),
			}}
		},
	},
	{
		ID: "evidence-carries-its-command",
		Reason: "A block presenting output presents the command that produced it. The rule, its vocabulary of program names and its near misses are in test/doclint, where they already judge every document in the tree, and this applies the same rule to a pull request body rather than keeping a second copy of it here. " +
			"A body is the one surface where a figure is most likely to be pasted from a run nobody can repeat, and it is the surface no reviewer can re-derive from the diff.",
		Refuse: func(c Change) []Finding {
			var found []Finding
			for _, f := range doclintFindings(c.Body) {
				found = append(found, Finding{
					RuleID: "evidence-carries-its-command",
					Where:  fmt.Sprintf("the body, line %d", f.Line),
					Detail: f.Detail,
				})
			}
			return found
		},
	},
	{
		ID: "commit-message-has-a-subject-and-a-break",
		Reason: "A commit message opens with a non-empty subject, and where it says more than that, a blank line separates the subject from the body. Both halves are readable from the message and nothing else. " +
			"Without the break, every tool that shows a subject shows the whole first paragraph, and the message stops being greppable in the one view most people read history through. " +
			"What this rule cannot decide is what CONTRIBUTING.md actually asks of a message, which is that it says what changed and what failure it prevents. That is a judgement about meaning and it is left to review rather than approximated here.",
		Refuse: func(c Change) []Finding {
			var found []Finding
			for _, commit := range c.Commits {
				lines := strings.Split(strings.ReplaceAll(commit.Message, "\r\n", "\n"), "\n")
				switch {
				case strings.TrimSpace(lines[0]) == "":
					found = append(found, Finding{
						RuleID: "commit-message-has-a-subject-and-a-break",
						Where:  at(commit.SHA),
						Detail: "the message opens with an empty line, so it has no subject",
					})
				case len(lines) > 1 && strings.TrimSpace(lines[1]) != "":
					found = append(found, Finding{
						RuleID: "commit-message-has-a-subject-and-a-break",
						Where:  at(commit.SHA),
						Detail: fmt.Sprintf("the line under the subject is %q rather than blank, so the subject and the body run together", short(lines[1])),
					})
				}
			}
			return found
		},
	},
	{
		ID: "change-is-inside-the-scope-its-issue-declares",
		Reason: "Every path the change touches is inside the `Scope:` its closing issue declares. This is the weak test of one topic and it is the only one a machine makes: it compares paths, and it cannot tell work that belongs to an issue from work that merely lands in the same directory. " +
			"It refuses rather than reports, which is what issue #93 asks for, and the consequence is worth knowing before meeting it. Where a change lands outside the declared scope the repair is the issue rather than the diff: the scope was written before the means was chosen, the means put the work elsewhere, and the `Scope:` line is corrected before the change lands. " +
			"That is the contribution guide's own answer to a change that does not fit, one rung earlier.",
		Refuse: func(c Change) []Finding {
			declared := scopes(c.Closes)
			if len(declared) == 0 {
				return nil
			}
			var found []Finding
			for _, path := range c.Files {
				if insideAny(path, declared) {
					continue
				}
				found = append(found, Finding{
					RuleID: "change-is-inside-the-scope-its-issue-declares",
					Where:  path,
					Detail: fmt.Sprintf("the declared scope is %s and this path is outside all of it", strings.Join(declared, " ")),
				})
			}
			return found
		},
	},
}

// A Report is what one reading of a pull request produced. The counts are part
// of the result rather than a debugging aid: a check that judged an empty view
// exits exactly like one that judged a whole change and was happy, and the
// counts in the log are what tell a reader which of the two happened.
type Report struct {
	// Number is the pull request judged.
	Number int
	// Files, Commits and ClosingIssues are how much there was to judge.
	Files         int
	Commits       int
	ClosingIssues int
	// DeclaredScope is the union of the `Scope:` lines the closing issues
	// carry, empty where none of them declares one.
	DeclaredScope []string
	// NotDecided names the rules that had nothing to decide on this
	// change, so a rule that passed by having no subject is not read as a
	// rule that passed.
	NotDecided []string
	// Findings is every refusal, in table order.
	Findings []Finding
}

// Judge reports what the table makes of one pull request.
func Judge(c Change) Report {
	judged := Report{
		Number:        c.Number,
		Files:         len(c.Files),
		Commits:       len(c.Commits),
		ClosingIssues: len(c.Closes),
		DeclaredScope: scopes(c.Closes),
	}
	if len(judged.DeclaredScope) == 0 {
		judged.NotDecided = append(judged.NotDecided,
			"change-is-inside-the-scope-its-issue-declares: no closing issue declares a `Scope:` line, so no path comparison was made")
	}
	for _, r := range Rules {
		judged.Findings = append(judged.Findings, r.Refuse(c)...)
	}
	return judged
}

// Reason returns the argument behind a rule id, so a failure can print both
// without the caller holding the table.
func Reason(id string) string {
	for _, r := range Rules {
		if r.ID == id {
			return r.Reason
		}
	}
	return ""
}

// Read turns the response of the one query this check makes into the view the
// rules judge. The template is read from the tree by the caller and passed in,
// because it is a fact about the commit being judged rather than about the
// fetch.
//
// It fails closed on three shapes in which a pass would mean the fetch never
// happened rather than that the change is clean: a response carrying no pull
// request, a response carrying no commit, and a response with another page of
// anything. The last one is the one that matters. A first page of files and a
// silent stop is how a scope rule passes a change whose foreign paths were on
// the second page, and that is a green verdict on a view nobody assembled.
func Read(response []byte, template string) (Change, error) {
	var fetched struct {
		Data struct {
			Repository struct {
				PullRequest *struct {
					Number                  int    `json:"number"`
					Body                    string `json:"body"`
					ClosingIssuesReferences struct {
						Nodes    []Issue  `json:"nodes"`
						PageInfo pageInfo `json:"pageInfo"`
					} `json:"closingIssuesReferences"`
					Commits struct {
						Nodes []struct {
							Commit struct {
								OID     string `json:"oid"`
								Message string `json:"message"`
							} `json:"commit"`
						} `json:"nodes"`
						PageInfo pageInfo `json:"pageInfo"`
					} `json:"commits"`
					Files struct {
						Nodes []struct {
							Path string `json:"path"`
						} `json:"nodes"`
						PageInfo pageInfo `json:"pageInfo"`
					} `json:"files"`
				} `json:"pullRequest"`
			} `json:"repository"`
		} `json:"data"`
	}
	if err := json.Unmarshal(response, &fetched); err != nil {
		return Change{}, fmt.Errorf("reading the fetched pull request: %w", err)
	}

	pr := fetched.Data.Repository.PullRequest
	if pr == nil {
		return Change{}, fmt.Errorf("the fetched response carries no pull request, so there is nothing in it to judge")
	}

	truncated := map[string]bool{
		"closing issues": pr.ClosingIssuesReferences.PageInfo.HasNextPage,
		"commits":        pr.Commits.PageInfo.HasNextPage,
		"files":          pr.Files.PageInfo.HasNextPage,
	}
	var partial []string
	for what, more := range truncated {
		if more {
			partial = append(partial, what)
		}
	}
	sort.Strings(partial)
	if len(partial) > 0 {
		return Change{}, fmt.Errorf(
			"the fetch returned another page of %s, so this is part of a pull request rather than one; "+
				"a rule reading a partial view passes what was not fetched", strings.Join(partial, " and "))
	}

	change := Change{
		Number:   pr.Number,
		Body:     pr.Body,
		Closes:   pr.ClosingIssuesReferences.Nodes,
		Template: template,
	}
	for _, node := range pr.Commits.Nodes {
		change.Commits = append(change.Commits, Commit{SHA: node.Commit.OID, Message: node.Commit.Message})
	}
	for _, node := range pr.Files.Nodes {
		change.Files = append(change.Files, node.Path)
	}

	if len(change.Commits) == 0 {
		return Change{}, fmt.Errorf("the fetched pull request carries no commit, so the message rule would pass by having nothing to read")
	}
	return change, nil
}

type pageInfo struct {
	HasNextPage bool `json:"hasNextPage"`
}

// doclintFindings puts a pull request body through the one documentation rule
// that applies to it. The path handed over is what a failure names, and the tree
// is one that holds nothing, because this rule asks the tree no questions and a
// tree that answered them would make the verdict depend on a checkout.
func doclintFindings(body string) []doclint.Finding {
	const evidence = "evidence-carries-its-command"
	parsed := doclint.ParseDoc("the pull request body", body)
	for _, r := range doclint.Rules {
		if r.ID == evidence {
			return r.Refuse(parsed, emptyTree{})
		}
	}
	return nil
}

// emptyTree answers no to everything. The evidence rule asks it nothing; it
// exists because the rule signature takes a tree and handing over the real one
// would let a rule reach the disk from here.
type emptyTree struct{}

func (emptyTree) Exists(string) bool               { return false }
func (emptyTree) Headings(string) ([]string, bool) { return nil, false }

// meansHeading finds the template's means section by its heading, so the rule
// names the heading the template actually carries rather than a copy of it that
// drifts the day the template is reworded. It is the second-level heading whose
// text names the means.
func meansHeading(template string) (string, bool) {
	for _, line := range strings.Split(strings.ReplaceAll(template, "\r\n", "\n"), "\n") {
		if !strings.HasPrefix(line, "## ") {
			continue
		}
		heading := strings.TrimSpace(strings.TrimPrefix(line, "## "))
		if strings.Contains(strings.ToLower(heading), "means") {
			return heading, true
		}
	}
	return "", false
}

// section returns the lines under a second-level heading, and whether the
// heading was there at all. The two answers are different findings: a missing
// section and an unfilled one are different mistakes and get different
// sentences.
func section(source, heading string) ([]string, bool) {
	lines := strings.Split(strings.ReplaceAll(source, "\r\n", "\n"), "\n")
	var body []string
	found := false
	for _, line := range lines {
		if strings.HasPrefix(line, "## ") {
			if found {
				break
			}
			found = strings.EqualFold(strings.TrimSpace(strings.TrimPrefix(line, "## ")), heading)
			continue
		}
		if found {
			body = append(body, line)
		}
	}
	return body, found
}

// added returns the content lines of a section that the template did not
// already write. Comparison is on the trimmed line, so re-indenting the
// instruction text does not read as an answer, and a line the author left in
// place beside their own sentence does not hide it either.
func added(written, boilerplate []string) []string {
	template := map[string]bool{}
	for _, line := range boilerplate {
		if trimmed := strings.TrimSpace(line); trimmed != "" {
			template[trimmed] = true
		}
	}
	var own []string
	for _, line := range written {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || template[trimmed] {
			continue
		}
		own = append(own, trimmed)
	}
	return own
}

// scopes reads the `Scope:` line out of every closing issue and returns the
// union, sorted so that a failure message is the same on two machines.
//
// The line is read at column zero and split on whitespace and commas, which is
// the shape both issue templates ask for. An issue writing its scope as prose
// declares none as far as this is concerned, and the report says the comparison
// was not made rather than that it passed.
func scopes(issues []Issue) []string {
	seen := map[string]bool{}
	for _, issue := range issues {
		for _, line := range strings.Split(strings.ReplaceAll(issue.Body, "\r\n", "\n"), "\n") {
			if !strings.HasPrefix(line, "Scope:") {
				continue
			}
			for _, entry := range strings.FieldsFunc(strings.TrimPrefix(line, "Scope:"), func(r rune) bool {
				return r == ' ' || r == '\t' || r == ','
			}) {
				seen[entry] = true
			}
			break
		}
	}
	var declared []string
	for entry := range seen {
		declared = append(declared, entry)
	}
	sort.Strings(declared)
	return declared
}

// insideAny reports whether a path is under any declared scope entry. A scope of
// "." is the whole repository and admits everything, which is what the issue
// template means by it.
func insideAny(path string, declared []string) bool {
	for _, entry := range declared {
		if entry == "." {
			return true
		}
		if path == entry {
			return true
		}
		if strings.HasPrefix(path, strings.TrimSuffix(entry, "/")+"/") {
			return true
		}
	}
	return false
}

// at abbreviates a commit for a failure message. Seven characters is what git
// prints and what a reader will paste back.
func at(sha string) string {
	if len(sha) > 7 {
		return "commit " + sha[:7]
	}
	return "commit " + sha
}

func short(line string) string {
	line = strings.TrimSpace(line)
	if len(line) > 60 {
		return line[:60] + "..."
	}
	return line
}
