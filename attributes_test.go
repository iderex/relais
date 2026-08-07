package main

import (
	"os"
	"testing"
)

// This test is about the repository rather than about the program, and it lives
// here because this is the only package. It exists because .gitattributes is a
// rule nothing else would notice breaking: change the line beginning `testdata/`
// and every fixture in the tree quietly changes shape on the next checkout, with
// no compilation failure and no lint finding to show for it.
//
// The bytes below are the ones the fixture was committed with. They hold both
// line endings and a lone carriage return, which is the byte a normalising
// checkout deletes first and the one a protocol frame is most likely to depend
// on.
func TestFixtureBytesSurviveTheCheckout(t *testing.T) {
	const path = "testdata/line-endings.golden"

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
