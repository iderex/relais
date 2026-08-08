// relais, a realtime SFU backend for community self-hosters.
// Copyright (C) 2026 Nils Lehnen
//
// Licensed under the GNU Affero General Public License, version 3. See LICENSE
// for the full terms, including the warranty and liability disclaimer.

package main

import (
	"os"
	"testing"
)

// This test is about the repository rather than about the program. It lives in
// the entry point package because a test needs a package to live in and this is
// the one that has no other reason to exclude it; docs/architecture.md is where
// that placement is argued. The fixture it reads stays at the repository root,
// because the `testdata/` line in .gitattributes is anchored there and a fixture
// moved out from under it would be normalised on the next checkout.
//
// It exists because .gitattributes is a rule nothing else would notice breaking:
// change that line and every fixture in the tree quietly changes shape on the
// next checkout, with no compilation failure and no lint finding to show for it.
//
// The bytes below are the ones the fixture was committed with. They hold both
// line endings and a lone carriage return, which is the byte a normalising
// checkout deletes first and the one a protocol frame is most likely to depend
// on.
func TestFixtureBytesSurviveTheCheckout(t *testing.T) {
	const path = "../../testdata/line-endings.golden"

	want := []byte("line ending fixture\r\nsecond line ends LF\nlone CR follows:\rend\r\n")

	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading %s: %v", path, err)
	}

	if string(got) != string(want) {
		t.Errorf("%s was rewritten by the checkout.\n got %q\nwant %q\n"+
			"The `testdata/** -text` line in .gitattributes is what keeps these "+
			"bytes intact; check it before changing the fixture.", path, got, want)
	}
}
