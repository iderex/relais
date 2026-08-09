// relais, a realtime SFU backend for community self-hosters.
// Copyright (C) 2026 Nils Lehnen
//
// Licensed under the GNU Affero General Public License, version 3. See LICENSE
// for the full terms, including the warranty and liability disclaimer.

// Package doclint holds the properties issue #94 fixes about this repository's
// written surfaces, in the form a machine can refuse a violation of.
//
// The documentation here carries measurements, commands and promises to
// operators, which puts it closer to code than to prose. The code is formatted,
// vetted and linted by a gate; the documents were read by whoever wrote them.
// This package is the other half of that.
//
// Four things are decidable by reading the files and nothing else, and each is a
// rule below. A link resolves to something that exists. A fragment on such a link
// names a heading in the file it points at. A repository path written in the
// prose is a path in the repository. And a block presenting output presents the
// command that produced it, which is this project's own rule about evidence and
// the most useful thing a check over prose can hold.
//
// WHAT IT CANNOT DECIDE, stated here rather than left to be discovered. Nothing
// here re-runs a command to see whether the output beside it is still what that
// command produces. A document whose figures were true a year ago passes every
// rule in this file, and that is the failure most likely to produce a
// confidently wrong page. It is recorded in docs/gate-parity.md as a gap rather
// than papered over, because the repair is a route that executes documentation
// rather than a stricter reader of it.
//
// The command rule is a lexer and not a shell. It decides whether a line opens a
// quote it never closes; it does not know whether the flags are real, whether the
// tool exists, or whether the pipeline means anything.
//
// The rules are data rather than code, for the same reason as in the neighbouring
// suites: the table reads as prose beside the property it holds, and a fixture
// can be put through exactly the decision the tree is put through. Nothing here
// imports anything else in this repository.
package doclint

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"
	"unicode"
)

// A Doc is one document, parsed into the four things the rules judge. Parsing
// once and judging the result keeps every rule reading the same view of a file,
// so two rules cannot disagree about where a block ended.
type Doc struct {
	// Path is the repository-relative path of the file, with forward
	// slashes on every operating system.
	Path string
	// Blocks are the code blocks, fenced and indented alike.
	Blocks []Block
	// Links are the inline markdown links.
	Links []Link
	// Paths are the repository paths written in backticks in the prose.
	Paths []PathMention
	// Trailing holds the line numbers ending in a space or a tab.
	Trailing []int
	// UnclosedFence is where a fence was opened that nothing closed, and
	// zero when every fence in the document is closed.
	UnclosedFence int
}

// A Block is one code block. Both markdown spellings produce one: a fenced block
// opened with three backticks, and an indented block of four spaces, which is
// what the documents under docs/ use for evidence.
type Block struct {
	// Line is where the first line inside the block is, so a failure sends
	// a reader to the content rather than to the fence above it.
	Line int
	// Language is the info string of a fenced block, empty for an untagged
	// fence and for every indented block.
	Language string
	// Lines are the lines inside the block, with the indent removed.
	Lines []string
}

// A Link is one inline markdown link, split at the fragment because the two
// halves are judged by different rules and a link can fail either one.
type Link struct {
	Line int
	// Target is the part before any '#', as written. Empty for a link into
	// the same document.
	Target string
	// Fragment is the part after the '#', without it. Empty when there is
	// none.
	Fragment string
}

// A PathMention is a repository path written in backticks in the prose, which is
// how this tree names a file it is talking about when it is not linking to it.
type PathMention struct {
	Line int
	Path string
}

// A Finding is one refusal: which rule, where, and what was wrong with it. The
// detail is written into the failure message beside the rule's reason, so a
// reader gets both the argument and the instance.
type Finding struct {
	RuleID string
	File   string
	Line   int
	Detail string
}

// A Tree answers the two questions a rule cannot answer from the document it is
// reading. It is an interface so that a fixture can be judged against a stated
// tree rather than against whatever happens to be on the disk, which is what
// lets the near-miss tests below be exact.
type Tree interface {
	// Exists reports whether a repository-relative path is in the tree.
	Exists(rel string) bool
	// Headings returns the heading slugs of a markdown file in the tree,
	// and whether the file could be read at all.
	Headings(rel string) ([]string, bool)
}

// A Rule refuses one thing. Reason is written out in full, so a failing test
// reads as the argument rather than as an identifier a reader has to go and look
// up.
type Rule struct {
	ID     string
	Reason string
	Refuse func(d Doc, t Tree) []Finding
}

// Rules is the whole table.
var Rules = []Rule{
	{
		ID: "link-resolves",
		Reason: "A link to a path in this repository points at something that exists. A document naming a file that is not there is the cheapest kind of wrong: it costs a reader the assumption that the rest of the page was checked, " +
			"and it happens by renaming a file rather than by writing anything incorrect, so the author of the break is never the author of the sentence. " +
			"Links to somewhere else on the internet are not judged here, because deciding those means asking a network and the suite asks nothing.",
		Refuse: func(d Doc, t Tree) []Finding {
			var found []Finding
			for _, l := range d.Links {
				if l.Target == "" || isExternal(l.Target) {
					continue
				}
				target := resolve(d.Path, l.Target)
				if !t.Exists(target) {
					found = append(found, Finding{
						RuleID: "link-resolves",
						File:   d.Path,
						Line:   l.Line,
						Detail: fmt.Sprintf("the link to %q resolves to %s, which is not in the tree", l.Target, target),
					})
				}
			}
			return found
		},
	},
	{
		ID: "link-fragment-resolves",
		Reason: "A fragment on such a link names a heading in the file it points at. This is the half that survives a rename: the file is still there, the section it was pointing into has been retitled, and the link now drops the reader at the top of a long document " +
			"with no sign that anything went wrong. The slug is derived the way the hosting site derives it, so what is checked is what a reader would click.",
		Refuse: func(d Doc, t Tree) []Finding {
			var found []Finding
			for _, l := range d.Links {
				if l.Fragment == "" || isExternal(l.Target) {
					continue
				}
				target := d.Path
				if l.Target != "" {
					target = resolve(d.Path, l.Target)
				}
				if !strings.HasSuffix(target, ".md") {
					continue
				}
				headings, ok := t.Headings(target)
				if !ok {
					// The file itself is missing, which the rule
					// above already refuses. Reporting it twice
					// would send a reader to two findings with
					// one repair.
					continue
				}
				if !contains(headings, l.Fragment) {
					found = append(found, Finding{
						RuleID: "link-fragment-resolves",
						File:   d.Path,
						Line:   l.Line,
						Detail: fmt.Sprintf("%s has no heading with the slug %q", target, l.Fragment),
					})
				}
			}
			return found
		},
	},
	{
		ID: "documented-path-resolves",
		Reason: "A repository path written in backticks in the prose is a path in the repository. Most of what these documents say about the tree is said this way rather than as a link, so a rule that judged only links would leave the larger half unread. " +
			"Only a mention whose first segment is a directory or file at the root of this tree is judged, so a module path, a hostname or a path in somebody else's repository is not mistaken for a claim about this one.",
		Refuse: func(d Doc, t Tree) []Finding {
			var found []Finding
			for _, m := range d.Paths {
				root, _, _ := strings.Cut(m.Path, "/")
				if !t.Exists(root) {
					continue
				}
				if t.Exists(strings.TrimSuffix(m.Path, "/")) {
					continue
				}
				found = append(found, Finding{
					RuleID: "documented-path-resolves",
					File:   d.Path,
					Line:   m.Line,
					Detail: fmt.Sprintf("%q names %s, which is not in the tree", m.Path, root),
				})
			}
			return found
		},
	},
	{
		ID: "evidence-carries-its-command",
		Reason: "A block presenting output presents the command that produced it. This project's rule is that where the evidence is a number it carries the command that produced it, and a block of figures with nothing above them is that rule broken in the one place it is most convincing: " +
			"a reader takes indented output for a measurement whoever wrote it must have run. The block is refused when it holds a digit and does not open with a command, so a list of files, a fragment of a name and a block of prose are left alone, " +
			"and a block tagged with a language is source rather than evidence and is not judged at all. Where the command is genuinely elsewhere, it moves into the block: two blocks are cheaper to keep true than a sentence linking them.",
		Refuse: func(d Doc, t Tree) []Finding {
			var found []Finding
			for _, b := range d.Blocks {
				if b.Language != "" || !holdsDigit(b.Lines) {
					continue
				}
				first, at := firstContentLine(b)
				if first == "" || looksLikeCommand(first) {
					continue
				}
				found = append(found, Finding{
					RuleID: "evidence-carries-its-command",
					File:   d.Path,
					Line:   at,
					Detail: fmt.Sprintf("the block opens with %q, which is not a command, and carries a figure", trim(first)),
				})
			}
			return found
		},
	},
	{
		ID: "line-ends-in-no-whitespace",
		Reason: "A line ends in the last character somebody typed. Two spaces at the end of a markdown line are a line break in the rendered page, which is a formatting decision made invisibly: it does not show in a diff, it does not show in an editor, and it survives every later edit of the paragraph. " +
			"The rest of the class is the same shape. Whitespace nobody can see is whitespace nobody reviewed, and this is the one formatting rule that pays for itself, because the tree is already free of it and a violation therefore always arrives with a change.",
		Refuse: func(d Doc, t Tree) []Finding {
			var found []Finding
			for _, line := range d.Trailing {
				found = append(found, Finding{
					RuleID: "line-ends-in-no-whitespace",
					File:   d.Path,
					Line:   line,
					Detail: "the line ends in a space or a tab",
				})
			}
			return found
		},
	},
	{
		ID: "fence-is-closed",
		Reason: "Every fence a document opens is closed. An unclosed one swallows the rest of the page into a code block, so a reader gets the remaining sections as monospaced text and every link in them stops being a link. " +
			"It is invisible in the source, where the missing thing is the thing that is not there, and obvious in the rendered page, which is the surface nobody looks at before merging a documentation change.",
		Refuse: func(d Doc, t Tree) []Finding {
			if d.UnclosedFence == 0 {
				return nil
			}
			return []Finding{{
				RuleID: "fence-is-closed",
				File:   d.Path,
				Line:   d.UnclosedFence,
				Detail: "this fence is never closed, so the rest of the document is inside it",
			}}
		},
	},
	{
		ID: "command-closes-its-quotes",
		Reason: "A command in a document closes every quote it opens. It is a lexical check and not a shell: it decides whether a reader who copies the line gets a prompt waiting for the rest of it, which is the failure a --jq expression full of single quotes actually produces. " +
			"What it cannot decide is whether the flags exist, whether the tool is installed, or whether the pipeline means anything, and it makes no claim to.",
		Refuse: func(d Doc, t Tree) []Finding {
			var found []Finding
			for _, b := range d.Blocks {
				if b.Language != "" {
					continue
				}
				for i, line := range b.Lines {
					if !looksLikeCommand(line) {
						continue
					}
					if quote, ok := unclosedQuote(line); !ok {
						found = append(found, Finding{
							RuleID: "command-closes-its-quotes",
							File:   d.Path,
							Line:   b.Line + i,
							Detail: fmt.Sprintf("the command opens a %s quote it never closes", quote),
						})
					}
				}
			}
			return found
		},
	},
}

// Refusals returns every finding the table has against one document, in table
// order. An empty result is a document the rules permit.
func Refusals(d Doc, t Tree) []Finding {
	var found []Finding
	for _, r := range Rules {
		found = append(found, r.Refuse(d, t)...)
	}
	return found
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

// commands is the set of program names a documented line may open with. It is a
// vocabulary rather than a pattern because "looks like a command" has no shape:
// a bare word with arguments is also a sentence with words after it, and the
// tree's evidence blocks contain both.
//
// A tool used in a document and missing from here reds the evidence rule, and
// the repair is to add it in this one place. That is the cost of the vocabulary
// and it is the same cost the fingerprint and invariant tables carry.
var commands = map[string]bool{
	"awk": true, "base64": true, "cat": true, "curl": true, "docker": true,
	"echo": true, "find": true, "gh": true, "git": true, "go": true,
	"gofmt": true, "golangci-lint": true, "grep": true, "head": true,
	"ls": true, "mkdir": true, "printf": true, "sed": true, "sort": true,
	"tail": true, "tar": true, "uniq": true, "uvx": true, "wc": true,
}

// keywords are the shell words a command line may open with instead of a program
// name. A loop over repositories is how this tree gathers a table of them, and it
// opens with "for".
var keywords = map[string]bool{
	"for": true, "if": true, "while": true, "case": true,
}

// looksLikeCommand reports whether a line inside a block is something a reader
// would run. A leading prompt is accepted and stripped, and so is a run of
// NAME=value assignments, which is how an environment is set for one command.
func looksLikeCommand(line string) bool {
	line = strings.TrimSpace(line)
	line = strings.TrimPrefix(line, "$ ")
	for {
		word, rest, _ := strings.Cut(line, " ")
		if word == "" {
			return false
		}
		if name, _, ok := strings.Cut(word, "="); ok && name != "" && isAssignmentName(name) {
			line = strings.TrimSpace(rest)
			continue
		}
		return commands[word] || keywords[word]
	}
}

// isAssignmentName reports whether a word is a shell variable name, so that
// GOPROXY=off in front of a command is read as an environment rather than as the
// command itself. A path with an equals sign in it is not one.
func isAssignmentName(word string) bool {
	for _, r := range word {
		if !unicode.IsLetter(r) && !unicode.IsDigit(r) && r != '_' {
			return false
		}
	}
	return word != ""
}

// unclosedQuote reports whether every quote on a command line is closed, and
// which one is not when it is. A backslash escapes the character after it, which
// is what keeps an escaped quote inside a double-quoted string from being read
// as the closing one.
func unclosedQuote(line string) (string, bool) {
	var open rune
	escaped := false
	for _, r := range line {
		switch {
		case escaped:
			escaped = false
		case r == '\\' && open != '\'':
			escaped = true
		case open == 0 && (r == '\'' || r == '"'):
			open = r
		case open == r:
			open = 0
		}
	}
	switch open {
	case '\'':
		return "single", false
	case '"':
		return "double", false
	}
	return "", true
}

// ParseDoc reads one document into the view the rules judge.
func ParseDoc(docPath, source string) Doc {
	d := Doc{Path: docPath}
	lines := strings.Split(strings.ReplaceAll(source, "\r\n", "\n"), "\n")

	var fence *Block
	var fenceOpenedAt int
	var indented *Block
	for i, line := range lines {
		number := i + 1

		if line != "" && strings.TrimRight(line, " \t") != line {
			d.Trailing = append(d.Trailing, number)
		}

		if fence != nil {
			if strings.HasPrefix(strings.TrimSpace(line), "```") {
				d.Blocks = append(d.Blocks, *fence)
				fence = nil
				continue
			}
			fence.Lines = append(fence.Lines, line)
			continue
		}
		if trimmed := strings.TrimSpace(line); strings.HasPrefix(trimmed, "```") {
			if indented != nil {
				d.Blocks = append(d.Blocks, *indented)
				indented = nil
			}
			fence = &Block{Line: number + 1, Language: strings.TrimSpace(strings.TrimPrefix(trimmed, "```"))}
			fenceOpenedAt = number
			continue
		}

		switch {
		case strings.HasPrefix(line, "    "):
			if indented == nil {
				indented = &Block{Line: number}
			}
			indented.Lines = append(indented.Lines, strings.TrimPrefix(line, "    "))
			continue
		case strings.TrimSpace(line) == "" && indented != nil:
			// A blank line does not end an indented block: an
			// evidence block with a gap between two runs is one
			// block, and splitting it would put the output of the
			// second run into a block of its own with no command.
			indented.Lines = append(indented.Lines, "")
			continue
		case indented != nil:
			d.Blocks = append(d.Blocks, *indented)
			indented = nil
		}

		d.Links = append(d.Links, linksInLine(number, line)...)
		d.Paths = append(d.Paths, pathsInLine(number, line)...)
	}
	if fence != nil {
		d.Blocks = append(d.Blocks, *fence)
		d.UnclosedFence = fenceOpenedAt
	}
	if indented != nil {
		d.Blocks = append(d.Blocks, *indented)
	}
	return d
}

// linksInLine reads the inline markdown links out of one line. A reference-style
// link is not read, because this tree writes none and a reader that guessed at
// one would report a finding nobody could act on.
func linksInLine(number int, line string) []Link {
	var found []Link
	rest := line
	for {
		open := strings.Index(rest, "](")
		if open < 0 {
			return found
		}
		rest = rest[open+2:]
		end := strings.IndexByte(rest, ')')
		if end < 0 {
			return found
		}
		destination := strings.TrimSpace(rest[:end])
		rest = rest[end+1:]
		if destination == "" {
			continue
		}
		target, fragment, _ := strings.Cut(destination, "#")
		found = append(found, Link{Line: number, Target: target, Fragment: fragment})
	}
}

// pathsInLine reads the backticked tokens that could be a path in this tree: one
// word, no spaces, at least one slash, and no placeholder in it. Which of them is
// actually a claim about this repository is decided by the rule, which has the
// tree to ask.
func pathsInLine(number int, line string) []PathMention {
	var found []PathMention
	parts := strings.Split(line, "`")
	for i := 1; i < len(parts); i += 2 {
		token := parts[i]
		if token == "" || strings.ContainsAny(token, " \t<>*?") || !strings.Contains(token, "/") {
			continue
		}
		if isExternal(token) {
			continue
		}
		found = append(found, PathMention{Line: number, Path: token})
	}
	return found
}

// Slug turns a heading into the fragment a link would use for it, the way the
// hosting site does: the text lowercased, punctuation dropped, spaces turned into
// hyphens.
func Slug(heading string) string {
	var b strings.Builder
	for _, r := range strings.ToLower(strings.TrimSpace(heading)) {
		switch {
		case unicode.IsLetter(r) || unicode.IsDigit(r) || r == '-':
			b.WriteRune(r)
		case r == ' ':
			b.WriteRune('-')
		}
	}
	return b.String()
}

// HeadingsIn returns the slug of every ATX heading in a document. A heading
// inside a fenced block is not one, which is the case a reader of lines alone
// gets wrong.
func HeadingsIn(source string) []string {
	var slugs []string
	inFence := false
	for _, line := range strings.Split(strings.ReplaceAll(source, "\r\n", "\n"), "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "```") {
			inFence = !inFence
			continue
		}
		if inFence || !strings.HasPrefix(line, "#") {
			continue
		}
		slugs = append(slugs, Slug(strings.TrimLeft(line, "# ")))
	}
	return slugs
}

// isExternal reports whether a link target is somewhere other than this tree.
func isExternal(target string) bool {
	lower := strings.ToLower(target)
	return strings.HasPrefix(lower, "http://") ||
		strings.HasPrefix(lower, "https://") ||
		strings.HasPrefix(lower, "mailto:") ||
		strings.HasPrefix(lower, "//")
}

// resolve turns a link written in one document into a repository-relative path.
func resolve(from, target string) string {
	return path.Clean(path.Join(path.Dir(from), target))
}

func contains(slugs []string, want string) bool {
	for _, s := range slugs {
		if s == want {
			return true
		}
	}
	return false
}

func holdsDigit(lines []string) bool {
	for _, line := range lines {
		for _, r := range line {
			if unicode.IsDigit(r) {
				return true
			}
		}
	}
	return false
}

func firstContentLine(b Block) (string, int) {
	for i, line := range b.Lines {
		if strings.TrimSpace(line) != "" {
			return line, b.Line + i
		}
	}
	return "", b.Line
}

func trim(line string) string {
	line = strings.TrimSpace(line)
	if len(line) > 60 {
		return line[:60] + "..."
	}
	return line
}

// OnDisk is the Tree the suite judges the repository with, rooted at the
// directory holding go.mod.
type OnDisk struct{ Root string }

func (o OnDisk) Exists(rel string) bool {
	if rel == "" || rel == "." || strings.HasPrefix(rel, "..") {
		return false
	}
	_, err := os.Stat(filepath.Join(o.Root, filepath.FromSlash(rel)))
	return err == nil
}

func (o OnDisk) Headings(rel string) ([]string, bool) {
	source, err := os.ReadFile(filepath.Join(o.Root, filepath.FromSlash(rel)))
	if err != nil {
		return nil, false
	}
	return HeadingsIn(string(source)), true
}

// ReadDocs walks the tree at root and returns every markdown document in it.
func ReadDocs(root string) ([]Doc, error) {
	var docs []Doc

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
		docs = append(docs, ParseDoc(filepath.ToSlash(relative), string(source)))
		return nil
	})
	if err != nil {
		return nil, err
	}
	return docs, nil
}

// skippedDir names the directories whose contents are not this project's prose.
// testdata holds bytes a test asserts against, and rewriting one of those to
// satisfy a documentation rule would be changing a fixture to please a linter.
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
