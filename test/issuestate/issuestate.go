// relais, a realtime SFU backend for community self-hosters.
// Copyright (C) 2026 Nils Lehnen
//
// Licensed under the GNU Affero General Public License, version 3. See LICENSE
// for the full terms, including the warranty and liability disclaimer.

// Package issuestate holds the property issue #179 fixes about this
// repository's prose: a document may not tell a reader that a question is
// unresolved and name an issue that is closed.
//
// A document that reports tracker state reports it as of the day it was
// written, and nothing re-reads it. Issue #178 found four such sentences by
// hand, in four different files, and none of them was reachable by any check in
// this tree: the documentation gate is declared file-only in its own words, and
// a file cannot say whether an issue is open.
//
// So this is the shape the neighbouring hygiene gate already uses. One fetch
// happens in the job before anything is judged, its answer is written to a
// file, and everything below is a function of that file and the bytes of the
// documents. Nothing here reads a clock, reaches a network, or asks anything
// that could answer differently on a second run.
//
// WHAT IT REFUSES is deliberately narrower than the failure it is named for.
// The failure is a sentence whose meaning has stopped being true, and meaning is
// not decidable here. What is decidable is a shape: a mention of a closed issue
// standing in the same sentence as a phrase asserting, in the present tense,
// that the matter is unresolved. Fifty-seven mentions of closed issues stood in
// nineteen files when #178 measured them and four were wrong, so a rule refusing
// every mention of a closed issue would have been ninety-three per cent noise.
// The vocabulary below is what makes the difference, and it is a vocabulary
// rather than a pattern for the same reason the documentation gate's command
// list is one: "asserts something is unresolved" has no shape.
//
// WHAT IT CANNOT DECIDE, stated here rather than left to be discovered.
//
// A sentence that is wrong about anything other than an issue's state passes.
// The second of the four sites #178 found is exactly that: it rests on an entry
// inside an issue having been answered, the issue itself is open, and no reading
// of tracker state separates a stale premise from a live one. That site is
// outside this subject rather than missed by it, and it is why the suite carries
// three site fixtures rather than four.
//
// A sentence naming both a closed issue and an open one, where the assertion
// belongs to the open one, is refused. That is the false positive this rule has,
// it is measured rather than supposed, and the near-miss suite carries an
// instance of it.
//
// A mention inside a code block is not judged at all. Those are pasted readings,
// dated by the block they sit in, and #178 names two of them and asks that they
// stay. So a stale claim written inside an evidence block passes every rule here.
//
// A number naming a pull request rather than an issue is not judged, and neither
// is a number the tracker does not carry. Both are counted and printed, because
// a run that resolved nothing exits exactly like a run that resolved everything
// and was happy.
//
// The rules are data rather than code, the same as in the neighbouring suites,
// so a fixture can be put through exactly the decision the tree is put through.
// Nothing here imports anything else in this repository: the prose reader below
// answers a different question from the documentation gate's parser, which wants
// the contents of a block, where this wants the lines that are not in one.
package issuestate

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"unicode"
)

// A Tracked is one issue as the fetch reported it. Three fields, because three
// are what a rule can use: everything else in a tracker response is prose, and a
// verdict resting on it would depend on somebody's wording.
type Tracked struct {
	Number int
	// Closed is the state, reduced to the one distinction that matters
	// here. A tracker that grows a third state reaches this as not closed,
	// which is the direction that refuses nothing new.
	Closed bool
	// PullRequest marks the numbers that are pull requests. They share the
	// numbering with issues, so a mention of one would otherwise be read as
	// a mention of a closed issue the moment it merged.
	PullRequest bool
}

// A Mention is one reference written in the prose of one document, with the
// sentence it stands in. The sentence travels with it because that is the window
// every rule judges, and cutting it once keeps two rules from disagreeing about
// where it ended.
type Mention struct {
	File     string
	Line     int
	Number   int
	Sentence string
}

// A Finding is one refusal: which rule, where, and what was wrong.
type Finding struct {
	RuleID string
	File   string
	Line   int
	Detail string
}

// A Rule refuses one thing. Reason is written out in full, so a failing test
// reads as the argument rather than as an identifier a reader has to look up.
type Rule struct {
	ID     string
	Reason string
	Refuse func(mentions []Mention, state map[int]Tracked) []Finding
}

// Rules is the whole table.
var Rules = []Rule{
	{
		ID: "closed-issue-not-called-unresolved",
		Reason: "A sentence naming an issue that is closed does not also assert, in the present tense, that the matter is unresolved. " +
			"This is the one thing about a document's report of tracker state that a machine can settle: the state is fetched, the sentence is bytes, and the two either disagree or they do not. " +
			"It is not the failure itself, which is a sentence whose meaning has stopped being true and which no reading decides. " +
			"What it catches is the shape that failure takes when it is written down, and the repair is the sentence rather than the rule: say what the issue settled, or move the claim to the issue that is still open.",
		Refuse: func(mentions []Mention, state map[int]Tracked) []Finding {
			var found []Finding
			for _, m := range mentions {
				tracked, known := state[m.Number]
				if !known || tracked.PullRequest || !tracked.Closed {
					continue
				}
				phrase, asserted := unresolvedPhrase(m.Sentence)
				if !asserted {
					continue
				}
				found = append(found, Finding{
					RuleID: "closed-issue-not-called-unresolved",
					File:   m.File,
					Line:   m.Line,
					Detail: fmt.Sprintf("issue #%d is closed and this sentence says %q: %s",
						m.Number, phrase, m.Sentence),
				})
			}
			return found
		},
	},
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

// unresolved is the vocabulary: the phrases that assert, in the present tense,
// that something has not been settled, built or decided.
//
// A vocabulary rather than a pattern, and phrases rather than words, because the
// bare word carrying most of this meaning is "open" and this tree opens ports,
// opens rooms and opens windows. Every entry below is matched as a run of words
// with a boundary at each end, so the "open" inside "opened" is not one.
//
// A phrase this tree uses and this list is missing is a sentence this rule lets
// through, which is the direction the miss goes in: the list bounds what is
// caught rather than what is refused, and it is extended in this one place.
var unresolved = []string{
	"is open",
	"are open",
	"stays open",
	"stay open",
	"remains open",
	"remain open",
	"still open",
	"holds it open",
	"is not settled",
	"are not settled",
	"is not decided",
	"are not decided",
	"is undecided",
	"are undecided",
	"is unresolved",
	"are unresolved",
	"is not built",
	"are not built",
	"is not yet built",
	"does not exist",
	"do not exist",
	"exists yet",
	"exist yet",
	"has not been taken",
	"have not been taken",
	"has not been decided",
	"have not been decided",
	"nobody has taken",
}

// unresolvedPhrase reports the first phrase from the vocabulary the sentence
// carries, and whether it carries one at all.
//
// The comparison is over the sentence lowered and with everything that is not a
// letter or a digit reduced to a single space, so a phrase broken across a line
// break or split by a comma is still one phrase. Which of two phrases in one
// sentence is reported follows the order of the vocabulary, and nothing depends
// on it beyond the failure message.
//
// An inline code span is removed before the comparison, and that is a rule
// rather than a convenience. This tree corrects a document by quoting the
// sentence it is replacing, in backticks, in the paragraph that replaces it, so
// a phrase inside a span is text being talked about rather than text asserting
// anything. The cost is stated where the bounds are: a live claim somebody wrote
// inside backticks is not judged.
func unresolvedPhrase(sentence string) (string, bool) {
	haystack := " " + words(withoutCodeSpans(sentence)) + " "
	for _, phrase := range unresolved {
		if strings.Contains(haystack, " "+phrase+" ") {
			return phrase, true
		}
	}
	return "", false
}

// withoutCodeSpans removes every run between two backticks. An unpaired backtick
// closes nothing and removes nothing, so a stray one cannot swallow the rest of
// a sentence.
func withoutCodeSpans(sentence string) string {
	var b strings.Builder
	rest := sentence
	for {
		open := strings.Index(rest, "`")
		if open < 0 {
			b.WriteString(rest)
			return b.String()
		}
		closing := strings.Index(rest[open+1:], "`")
		if closing < 0 {
			b.WriteString(rest)
			return b.String()
		}
		b.WriteString(rest[:open])
		b.WriteString(" ")
		rest = rest[open+1+closing+1:]
	}
}

// words lowers a string and reduces every run of bytes that are not letters or
// digits to one space, which is what makes a phrase match survive markdown.
func words(s string) string {
	var b strings.Builder
	space := false
	for _, r := range strings.ToLower(s) {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			if space && b.Len() > 0 {
				b.WriteByte(' ')
			}
			space = false
			b.WriteRune(r)
			continue
		}
		space = true
	}
	return b.String()
}

// backReferences are the sentence openers that carry no subject of their own. A
// sentence beginning with one of them is about whatever the sentence before it
// named, so the two are judged as one window.
//
// This is not tidiness. The fourth site #178 found reads "Fuzzing that surface
// is #91. Neither exists yet.", where the number and the assertion sit in
// adjacent sentences and a rule reading one sentence at a time sees neither
// half.
var backReferences = []string{
	"neither", "both", "either", "none of", "all three", "all four",
	"the two", "the three", "the four", "those", "these",
}

// Refusals returns every finding the table has, in table order.
func Refusals(mentions []Mention, state map[int]Tracked) []Finding {
	var found []Finding
	for _, r := range Rules {
		found = append(found, r.Refuse(mentions, state)...)
	}
	return found
}

// MentionsIn reads one document and returns every reference written in its
// prose.
//
// Prose is everything that is not inside a code block, in either markdown
// spelling: a fence opened with three backticks, and the four-space indent this
// tree writes its evidence in. A blank line does not end an indented block, for
// the same reason it does not end one in the documentation gate: an evidence
// block with a gap between two runs is one block.
//
// The line reported is the line the number is written on, which is where a
// reader has to go, even where the sentence around it spans several.
func MentionsIn(docPath, source string) []Mention {
	var mentions []Mention
	for _, block := range paragraphs(proseOf(source)) {
		for _, sentence := range sentences(block) {
			for _, ref := range referencesIn(sentence.text) {
				mentions = append(mentions, Mention{
					File:     docPath,
					Line:     sentence.lineOf(ref.offset),
					Number:   ref.number,
					Sentence: collapse(sentence.text),
				})
			}
		}
	}
	return mentions
}

// proseOf returns the document's lines with every line inside a code block
// replaced by a blank, so the shape of the file survives and a block cannot be
// pulled into the sentence around it.
func proseOf(source string) []string {
	lines := strings.Split(strings.ReplaceAll(source, "\r\n", "\n"), "\n")
	prose := make([]string, len(lines))

	inFence := false
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		switch {
		case inFence:
			if strings.HasPrefix(trimmed, "```") {
				inFence = false
			}
		case strings.HasPrefix(trimmed, "```"):
			inFence = true
		case strings.HasPrefix(line, "    "):
			// An indented block. Its lines leave a blank entry, so
			// the paragraph before it and the paragraph after it
			// are two windows rather than one with the block's
			// bytes in the middle.
		case trimmed == "":
		default:
			prose[i] = line
		}
	}
	return prose
}

// a paragraph is a run of prose lines with the line number each byte came from,
// so a sentence assembled out of several lines can still say which line a
// number was written on.
type paragraph struct {
	text  string
	lines []int
}

// paragraphs cuts the prose into runs separated by blank lines and by the blocks
// that were removed. A paragraph is the largest window a sentence may be
// assembled from.
func paragraphs(prose []string) []paragraph {
	var out []paragraph
	var current paragraph

	flush := func() {
		if strings.TrimSpace(current.text) != "" {
			out = append(out, current)
		}
		current = paragraph{}
	}
	for i, line := range prose {
		if line == "" {
			flush()
			continue
		}
		if current.text != "" {
			current.text += " "
			current.lines = append(current.lines, i+1)
		}
		current.text += line
		for range len(line) {
			current.lines = append(current.lines, i+1)
		}
	}
	flush()
	return out
}

// a segment is one sentence inside a paragraph, carrying enough of the
// paragraph's shape to report a line number.
type segment struct {
	text   string
	start  int
	parent *paragraph
}

// lineOf reports the line the byte at an offset inside this sentence was written
// on.
func (s segment) lineOf(offset int) int {
	at := s.start + offset
	if s.parent == nil || at < 0 || at >= len(s.parent.lines) {
		return 0
	}
	return s.parent.lines[at]
}

// sentences cuts a paragraph at sentence ends and then joins back the ones that
// begin with a back-reference, which have no subject of their own.
//
// The cut is at a full stop followed by a space and something a sentence can
// open with, which separates a sentence from a version number, an abbreviation
// and a file extension without a list of exceptions. A capital is the ordinary
// opener; a hash, a backtick and a bracket are the three this tree also opens
// with, naming an issue, a path and a link. A cut this misses widens the window
// and refuses more, so the near-miss suite is where that direction is held.
func sentences(p paragraph) []segment {
	text := p.text

	var cuts []int
	for i := 0; i+2 < len(text); i++ {
		if text[i] != '.' || text[i+1] != ' ' {
			continue
		}
		if opensASentence(text[i+2]) {
			cuts = append(cuts, i+1)
		}
	}

	var raw []segment
	from := 0
	for _, cut := range append(cuts, len(text)) {
		raw = append(raw, segment{text: text[from:cut], start: from, parent: &p})
		from = cut
	}

	var out []segment
	for _, s := range raw {
		if len(out) > 0 && opensWithABackReference(s.text) {
			out[len(out)-1].text += s.text
			continue
		}
		out = append(out, s)
	}
	return out
}

// opensASentence reports whether a byte after a full stop and a space is one a
// sentence in this tree begins with.
func opensASentence(b byte) bool {
	return unicode.IsUpper(rune(b)) || b == '#' || b == '`' || b == '['
}

// opensWithABackReference reports whether a sentence begins with a word that
// refers to what the sentence before it named.
func opensWithABackReference(sentence string) bool {
	opening := words(sentence)
	for _, ref := range backReferences {
		if opening == ref || strings.HasPrefix(opening, ref+" ") {
			return true
		}
	}
	return false
}

// a reference is one issue number and where it stands inside the sentence.
type reference struct {
	number int
	offset int
}

// referencesIn reads every issue number out of a sentence.
//
// A hash immediately after a word byte is not one: a fragment on a link and an
// anchor in a URL are written that way and neither names an issue. A markdown
// heading is never seen here either, because a heading opens with a hash
// followed by a space rather than by a digit.
func referencesIn(sentence string) []reference {
	var out []reference
	for i := 0; i < len(sentence); i++ {
		if sentence[i] != '#' {
			continue
		}
		if i > 0 && isWordByte(sentence[i-1]) {
			continue
		}
		j := i + 1
		for j < len(sentence) && sentence[j] >= '0' && sentence[j] <= '9' {
			j++
		}
		if j == i+1 {
			continue
		}
		if j < len(sentence) && isWordByte(sentence[j]) {
			continue
		}
		number, err := strconv.Atoi(sentence[i+1 : j])
		if err != nil {
			continue
		}
		out = append(out, reference{number: number, offset: i})
		i = j - 1
	}
	return out
}

// isWordByte reports whether a byte is one a word can be made of.
func isWordByte(b byte) bool {
	return b == '_' || b >= 'a' && b <= 'z' || b >= 'A' && b <= 'Z' || b >= '0' && b <= '9'
}

// collapse reduces the whitespace in a sentence so a failure message reads as
// one line.
func collapse(s string) string {
	return strings.Join(strings.Fields(s), " ")
}

// A Report is what one reading produced. The counts are part of the result
// rather than a debugging aid: a run that resolved nothing exits exactly like a
// run that resolved everything and was happy, and the counts are what tell a
// reader which of the two happened.
type Report struct {
	Documents int
	Mentions  int
	// Closed is how many mentions named an issue the fetch reported closed,
	// which is the population this rule judges.
	Closed int
	// PullRequests is how many named a pull request, and Unknown which
	// numbers the fetch did not carry. Neither is judged, and both are
	// printed so a green run is not read as one that resolved every number
	// in the tree.
	PullRequests int
	Unknown      []int
	Findings     []Finding
}

// Judge reports what the table makes of one set of mentions.
func Judge(mentions []Mention, state map[int]Tracked, documents int) Report {
	judged := Report{Documents: documents, Mentions: len(mentions)}

	unknown := map[int]bool{}
	for _, m := range mentions {
		tracked, known := state[m.Number]
		switch {
		case !known:
			unknown[m.Number] = true
		case tracked.PullRequest:
			judged.PullRequests++
		case tracked.Closed:
			judged.Closed++
		}
	}
	for number := range unknown {
		judged.Unknown = append(judged.Unknown, number)
	}
	sort.Ints(judged.Unknown)

	judged.Findings = Refusals(mentions, state)
	return judged
}

// ReadState turns what the fetch wrote into the state the rules read.
//
// The file is one JSON object per line, which is what the tracker query writes
// when it is paged, so no second tool has to stitch the pages together between
// the fetch and the judging.
//
// It fails closed on the shapes in which a pass would mean the fetch never
// happened rather than that the tree is clean: a line that is not an object, a
// line naming no number, and a file carrying no object at all. A rule that finds
// no state for a number leaves it unjudged, so an empty fetch would otherwise be
// a green run over nothing.
func ReadState(response []byte) (map[int]Tracked, error) {
	state := map[int]Tracked{}

	scanner := bufio.NewScanner(bytes.NewReader(response))
	scanner.Buffer(make([]byte, 0, 64*1024), 4*1024*1024)
	for line := 1; scanner.Scan(); line++ {
		raw := strings.TrimSpace(scanner.Text())
		if raw == "" {
			continue
		}
		var row struct {
			Number      int    `json:"number"`
			State       string `json:"state"`
			PullRequest bool   `json:"pull_request"`
		}
		if err := json.Unmarshal([]byte(raw), &row); err != nil {
			return nil, fmt.Errorf("line %d of the fetched tracker state is not an object: %w", line, err)
		}
		if row.Number == 0 {
			return nil, fmt.Errorf("line %d of the fetched tracker state names no number", line)
		}
		state[row.Number] = Tracked{
			Number:      row.Number,
			Closed:      strings.EqualFold(row.State, "closed"),
			PullRequest: row.PullRequest,
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("reading the fetched tracker state: %w", err)
	}
	if len(state) == 0 {
		return nil, fmt.Errorf("the fetched tracker state carries no issue, so nothing could be judged")
	}
	return state, nil
}

// ReadMentions walks the tree at root and returns every mention in every tracked
// markdown document, with how many documents were read.
func ReadMentions(root string) ([]Mention, int, error) {
	var mentions []Mention
	documents := 0

	err := filepath.WalkDir(root, func(p string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			if skippedDir(entry.Name()) && p != root {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(entry.Name(), ".md") {
			return nil
		}
		relative, err := filepath.Rel(root, p)
		if err != nil {
			return err
		}
		source, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		documents++
		mentions = append(mentions, MentionsIn(filepath.ToSlash(relative), string(source))...)
		return nil
	})
	if err != nil {
		return nil, 0, err
	}
	return mentions, documents, nil
}

// skippedDir names the directories whose contents are not this project's prose.
// The same three the documentation gate skips, and for the same reasons.
func skippedDir(name string) bool {
	return name == ".git" || name == "testdata" || name == "vendor"
}

// ModuleRoot walks up from the working directory to the directory holding
// go.mod, so the tree being judged is found rather than assumed from a relative
// path that stops being true when this package moves.
func ModuleRoot() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("no go.mod above the working directory")
		}
		dir = parent
	}
}
