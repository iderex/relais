// relais, a realtime SFU backend for community self-hosters.
// Copyright (C) 2026 Nils Lehnen
//
// Licensed under the GNU Affero General Public License, version 3. See LICENSE
// for the full terms, including the warranty and liability disclaimer.

package determinism_test

import (
	"strconv"
	"strings"
	"testing"

	"github.com/iderex/relais/test/determinism"
)

// TestTheTreeIsDeterministic is the guard itself. Every use of an imported
// identifier in every Go file in this repository is put through the table, and a
// refusal is reported with the file and line it is at and the sentence that
// refuses it, so a failure sends the reader to the argument rather than to a
// rule with no author.
func TestTheTreeIsDeterministic(t *testing.T) {
	root, err := determinism.ModuleRoot()
	if err != nil {
		t.Fatalf("finding the module root: %v", err)
	}

	uses, err := determinism.ReadUses(root)
	if err != nil {
		t.Fatalf("reading the tree at %s: %v", root, err)
	}

	// A pass over nothing is not a pass. Two of the three properties here
	// are about test files, and a walk that had silently stopped finding
	// them would report a clean tree in exactly the same words as a clean
	// one. So the scan is reported and an anchor is asserted.
	var production, test int
	for _, u := range uses {
		if u.Test {
			test++
		} else {
			production++
		}
	}
	t.Logf("%d uses under %s: %d in production files, %d in test files", len(uses), root, production, test)

	const anchor = "internal/orchestration/credential/"
	found := false
	for _, u := range uses {
		if strings.HasPrefix(u.File, anchor) && u.Package == "time" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("no use of the time package was read from %s, so this run judged a tree it did not find; "+
			"if that package has legitimately stopped using time, move this anchor to one that has not rather than deleting it", anchor)
	}

	for _, u := range uses {
		for _, rule := range determinism.Refusals(u) {
			t.Errorf("%s:%d %s, which %s refuses:\n\t%s",
				u.File, u.Line, describe(u), rule.ID, rule.Reason)
		}
	}
}

// describe says what a use is in the words somebody would use to read it back.
// A dot import selects no identifier, so the plain form would report it as a use
// of a package's empty name and read as a typo rather than as the thing refused.
func describe(u determinism.Use) string {
	if u.Name == determinism.DotImport {
		return "dot-imports " + strconv.Quote(u.Package)
	}
	return "uses " + u.Package + "." + u.Name
}

// A fixture is one file somebody could plausibly write. Realistic is the point:
// each of these is written while doing something else, not invented to trip a
// check.
type fixture struct {
	why    string
	file   string
	source string
	rule   string
}

// violations carries the mistakes the table has to refuse. Every one of them
// passes the analyser set in .golangci.yml today, which is why this package
// exists.
var violations = []fixture{
	{
		why:  "a verifier falling back to the machine's clock when nobody injected one, which is the line that makes every expiry test probable again",
		file: "internal/orchestration/credential/verify.go",
		rule: "production-reads-no-clock",
		source: `package credential

import "time"

func (v *Verifier) at() time.Time {
	if v.now == nil {
		return time.Now()
	}
	return v.now()
}
`,
	},
	{
		why:  "the same read behind an alias, which is what a check written against the local name would miss",
		file: "internal/orchestration/session.go",
		rule: "production-reads-no-clock",
		source: `package orchestration

import clock "time"

func (s *session) touch() {
	s.lastSeen = clock.Now()
}
`,
	},
	{
		why:  "a forwarding loop keeping its own pace off real time, which is a clock read that never says the word",
		file: "internal/forwarding/loop.go",
		rule: "production-reads-no-clock",
		source: `package forwarding

import "time"

func (l *loop) run() {
	tick := time.NewTicker(20 * time.Millisecond)
	defer tick.Stop()
	for range tick.C {
		l.step()
	}
}
`,
	},
	{
		why:  "a test asking the operating system for a free port, which is the most reasonable-looking way a suite starts depending on the machine under it",
		file: "internal/api/server_test.go",
		rule: "unit-test-opens-no-socket",
		source: `package api_test

import (
	"net"
	"testing"
)

func TestTheServerAnswers(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("no free port: %v", err)
	}
	defer listener.Close()
}
`,
	},
	{
		why:  "a test server from the standard library's own testing package, which binds a real port while reading as though it does not",
		file: "internal/api/handler_test.go",
		rule: "unit-test-opens-no-socket",
		source: `package api_test

import (
	"net/http/httptest"
	"testing"
)

func TestTheHandlerRefusesAnUnknownRoom(t *testing.T) {
	server := httptest.NewServer(handler())
	defer server.Close()
}
`,
	},
	{
		why:  "the sleep that lets a goroutine catch up, written once and copied everywhere afterwards",
		file: "internal/orchestration/room_test.go",
		rule: "test-does-not-wait-on-real-time",
		source: `package orchestration

import (
	"testing"
	"time"
)

func TestTheEventReachesTheSubscriber(t *testing.T) {
	go publish()
	time.Sleep(50 * time.Millisecond)
}
`,
	},
	{
		why:  "the same wait as a timeout arm in a select, which is what somebody reaches for once a sleep has been refused",
		file: "internal/orchestration/events_test.go",
		rule: "test-does-not-wait-on-real-time",
		source: `package orchestration

import (
	"testing"
	"time"
)

func TestTheStreamCloses(t *testing.T) {
	select {
	case <-closed:
	case <-time.After(time.Second):
		t.Fatal("the stream did not close")
	}
}
`,
	},
	{
		why:  "a dot import, which leaves Sleep in the file's own scope and no selector for any other rule in this table to read",
		file: "internal/forwarding/pacing_test.go",
		rule: "a-watched-package-is-not-dot-imported",
		source: `package forwarding

import (
	"testing"

	. "time"
)

func TestThePacerHoldsItsRate(t *testing.T) {
	Sleep(Millisecond)
}
`,
	},
}

// TestEveryRuleRefusesARealisticViolation is the proof that the table bites.
// Each fixture goes through the same parse and the same decision the tree goes
// through, and the rule that has to catch it is named rather than left to
// whichever one happens to fire.
func TestEveryRuleRefusesARealisticViolation(t *testing.T) {
	for _, v := range violations {
		t.Run(v.rule+"/"+v.file, func(t *testing.T) {
			uses, err := determinism.UsesInFile(v.file, v.source)
			if err != nil {
				t.Fatalf("parsing the fixture: %v", err)
			}

			var refused []string
			for _, u := range uses {
				for _, rule := range determinism.Refusals(u) {
					refused = append(refused, rule.ID)
				}
			}
			if len(refused) == 0 {
				t.Fatalf("%s is permitted and must not be: %s", v.file, v.why)
			}
			if !contains(refused, v.rule) {
				t.Errorf("%s is refused, but not by %s: %s\n\t%s",
					v.file, v.rule, strings.Join(refused, ", "), v.why)
			}
		})
	}
}

// TestEveryRuleIsExercisedByAViolation closes the gap the table above cannot
// see. A rule with no fixture is a rule nothing has ever run, and it would sit
// in the table looking enforced.
func TestEveryRuleIsExercisedByAViolation(t *testing.T) {
	exercised := map[string]bool{}
	for _, v := range violations {
		exercised[v.rule] = true
	}
	for _, rule := range determinism.Rules {
		if !exercised[rule.ID] {
			t.Errorf("no fixture exercises %s, so nothing has shown it refuses anything:\n\t%s",
				rule.ID, rule.Reason)
		}
	}
	for id := range exercised {
		if !inTable(id) {
			t.Errorf("a fixture names %s, which is not a rule; the case is testing nothing", id)
		}
	}
}

// permitted is the near-miss set: the file one word or one direction away from
// each violation above, which has to stay legal. A table that refused these
// would pass every case above and stop the work instead of the mistake.
var permitted = []fixture{
	{
		why:  "fixing a moment from parts is what a deterministic test does, and it is the opposite of reading the machine's clock",
		file: "internal/orchestration/credential/credential_test.go",
		source: `package credential

import (
	"testing"
	"time"
)

var theMoment = time.Date(2026, time.March, 1, 12, 0, 0, 0, time.UTC)

func TestTheWindowCloses(t *testing.T) {
	_ = theMoment.Add(time.Minute)
}
`,
	},
	{
		why:  "a duration is a number and a month is a constant; neither waits for anything",
		file: "internal/forwarding/pacing.go",
		source: `package forwarding

import "time"

const interval = 20 * time.Millisecond

func deadline(from time.Time) time.Time {
	return from.Add(interval)
}
`,
	},
	{
		why:  "reconstructing a moment the credential already carried is not a clock read",
		file: "internal/orchestration/credential/verify.go",
		source: `package credential

import "time"

func expiry(seconds int64) time.Time {
	return time.Unix(seconds, 0)
}
`,
	},
	{
		why:  "the service has to listen, and this rule's subject is a unit test rather than every file in the tree",
		file: "internal/api/listener.go",
		source: `package api

import "net"

func listen(address string) (net.Listener, error) {
	return net.Listen("tcp", address)
}
`,
	},
	{
		why:  "parsing an address and joining a host to a port reach no network, and refusing them would stop a table test about addresses",
		file: "internal/api/address_test.go",
		source: `package api

import (
	"net"
	"testing"
)

func TestTheAddressParses(t *testing.T) {
	if net.ParseIP("192.0.2.1") == nil {
		t.Fatal("a documentation address did not parse")
	}
	_ = net.JoinHostPort("192.0.2.1", "7880")
}
`,
	},
	{
		why:  "a package whose import path begins with a watched one is a different package, which is the mistake a prefix test makes",
		file: "internal/api/candidate_test.go",
		source: `package api

import (
	"net/netip"
	"testing"
)

func TestTheCandidateParses(t *testing.T) {
	if _, err := netip.ParseAddr("192.0.2.1"); err != nil {
		t.Fatalf("a documentation address did not parse: %v", err)
	}
}
`,
	},
	{
		why:  "an alias can point the other way too: this file's time is not the standard library's, and judging it by the local name would refuse a rate limiter for being called time",
		file: "internal/api/limits.go",
		source: `package api

import time "golang.org/x/time/rate"

func limiter() *time.Limiter {
	return time.NewLimiter(10, 1)
}
`,
	},
	{
		why:  "a dot import of a package no rule here reads hides nothing from any of them",
		file: "internal/mediaplane/domain_test.go",
		source: `package mediaplane_test

import (
	"testing"

	. "github.com/iderex/relais/internal/mediaplane"
)

func TestTheRoomIdentifierIsOpaque(t *testing.T) {
	_ = RoomID("r-1")
}
`,
	},
	{
		why:  "a blank import binds no name, so nothing in the file can reach the package and no rule has a subject",
		file: "cmd/relais/main.go",
		source: `package main

import (
	_ "net/http/pprof"
)

func main() {}
`,
	},
}

func TestThePermittedFilesStayPermitted(t *testing.T) {
	for _, p := range permitted {
		t.Run(p.file, func(t *testing.T) {
			uses, err := determinism.UsesInFile(p.file, p.source)
			if err != nil {
				t.Fatalf("parsing the fixture: %v", err)
			}
			for _, u := range uses {
				for _, rule := range determinism.Refusals(u) {
					t.Errorf("%s:%d %s and is refused by %s, which it must not be: %s",
						p.file, u.Line, describe(u), rule.ID, p.why)
				}
			}
		})
	}
}

// TestEveryRuleStatesItsReason holds the condition that a failure sends the
// reader to the argument. A rule whose reason is empty produces a message
// nobody can act on, and a rule with no identifier cannot be named by a fixture.
func TestEveryRuleStatesItsReason(t *testing.T) {
	for i, rule := range determinism.Rules {
		if rule.ID == "" {
			t.Errorf("rule %d has no identifier", i)
		}
		if strings.TrimSpace(rule.Reason) == "" {
			t.Errorf("rule %s states no reason", rule.ID)
		}
	}
}

// TestAFileThatDoesNotParseIsReportedRatherThanSkipped holds the failure mode
// this reader would otherwise have. A walk that swallowed a parse error would
// report the tree as clean when it had not read it, which is the shape every
// guard in this repository is asked not to have.
func TestAFileThatDoesNotParseIsReportedRatherThanSkipped(t *testing.T) {
	_, err := determinism.UsesInFile("internal/api/broken.go", "package api\n\nfunc (\n")
	if err == nil {
		t.Fatal("a file that does not parse was read as a file with no uses in it")
	}
	if !strings.Contains(err.Error(), "internal/api/broken.go") {
		t.Errorf("the error does not name the file it could not read: %v", err)
	}
}

func contains(ids []string, id string) bool {
	for _, got := range ids {
		if got == id {
			return true
		}
	}
	return false
}

func inTable(id string) bool {
	for _, rule := range determinism.Rules {
		if rule.ID == id {
			return true
		}
	}
	return false
}
