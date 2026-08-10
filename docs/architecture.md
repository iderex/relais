# The layout, and the direction its dependencies run

A layout is an architecture decision that gets made by accident if nobody makes
it on purpose. This one has a specific job: the boundary between orchestration
and media is visible in the directory names, because that boundary is what the
whole plan rests on and it is the thing a hurried change erodes first.

Recorded for issue #16. It is short because a long one does not get read.

## The tree

    cmd/relais/              the executable, and the only place the parts are wired together
    internal/mediaplane/     the port: the interface, and the values that cross it
    internal/forwarding/     the only package that holds a packet
    internal/orchestration/  rooms, admission, events, capacity, the pool
    internal/api/            the operator API and the signalling surface
    bench/                   the measuring instrument, outside the build of the service
    deploy/                  the artefacts an operator runs
    docs/decisions/          one decision per file
    test/                    suites whose subject is the tree rather than one package
    testdata/                fixtures the tests assert against, byte for byte

Every directory above carries a readme saying what belongs in it and what does
not, except `testdata/`, which holds bytes a test asserts against and no prose.
This document is the part that is about the relationships between them.

## The dependency directions

Stated in the form a test is written against, which is issue #95, and each one is
quoted into that test word for word, so a failure reads as this document rather
than as an identifier a reader has to go and look up.

`internal/mediaplane` may not import `internal/forwarding`, and may not import
`internal/orchestration` or `internal/api`. It sits under both sides and depends
on neither.

`internal/mediaplane` may not import the transport library. Neither may
`internal/orchestration`, `internal/api`, or `cmd/relais`.

`internal/forwarding` may not import `internal/orchestration` or `internal/api`.
The forwarding core does not reach upward.

`internal/orchestration` and `internal/api` may not import
`internal/forwarding`. They reach the media plane through the port and through
nothing else.

`internal/api` may import `internal/orchestration`. Not the other way round: an
event stream that knows its own wire format is an event stream that changes when
the wire format does.

`bench/` may not import `cmd/` or `internal/`, and nothing under `cmd/` or
`internal/` may import `bench/`. The bench measures a target it is pointed at.

`cmd/relais` may import anything under `internal/`. Nothing may import
`cmd/relais`.

## What refuses a violation of them

`test/architecture` holds the seven above as a table and puts every import in
every Go file in this tree through it, test files included. An orchestration test
reaching past the fake to the real forwarding core is the failure the section
below names in its own words, so a reader that skipped test files would miss the
case the directions exist for.

It reads the tree with the language's own parser in import-only mode rather than
through the build system, so a direction is measurable in a tree that does not
compile and the answer needs no module download and no network.

The bound on it, because a green suite says less here than it looks like it says.
The check decides which package may import which and nothing else. A direction
over a package holding no Go file has nothing to refuse: `internal/forwarding`
carries a readme and no code today, so the two directions about it pass by having
no subject rather than by being obeyed. What separates that from a run that found
nothing is printed on every run, as the number of imports and the number of
packages they were read from.

## What these directions buy

Two properties, and neither of them survives a single exception.

The orchestration suite runs against a fake that carries no media. That is what
makes the whole of it executable on a stock runner with no display, no audio
device and no network path, which is the constraint issue #18 holds. It only
works while nothing above the port can reach the implementation below it: one
import of `internal/forwarding` from an orchestration test and the suite starts
needing the real thing.

Replacing the forwarding core changes one package. The media plane port record
estimates that cost and marks the estimate unverified, because there is no
implementation to replace yet. The direction above is what the estimate rests on,
and the estimate is checked again when the first implementation lands.

## The example that belongs in two places

`cmd/relais/attributes_test.go` is a test about the repository rather than about
the program. It asserts that the bytes of a fixture survive a checkout, which is
a property of `.gitattributes` and of nothing that compiles.

It lives in the entry point package. A test needs a package to live in, this is
the package with no reason to exclude it, and putting it under `internal/` would
mean inventing a package whose only purpose is to hold it. The fixture it reads
stays at the repository root and is reached by a relative path, because the
`testdata/` line in `.gitattributes` is anchored at the root: a pattern with a
slash in it matches from the directory the file sits in, so a fixture moved into
a package directory would fall out of the exemption and be normalised on the next
checkout, which is the exact failure the test exists to catch.

That is the known limit of this layout rather than a detail. A package that needs
its own fixtures needs the `.gitattributes` pattern widened first, in its own
change, with the test that proves the widened pattern still holds. Doing it the
other way round produces a package whose fixtures are silently rewritten and a
test that passes against the wrong bytes.

## What this document does not decide

Which packages exist inside `internal/forwarding` or `internal/orchestration`.
Those are shaped by the domain model in issue #31 and the API contract in issue
#43, and a layout that named them now would be guessing.

Whether the other two structural rules issue #95 names are enforced. They are
not. That the forwarding core never learns who a participant is, and that the
degradation decision lives in one place, are both decidable from code that has
not been written, so a test for either would count to zero and report success.
Two guards going green because there is nothing there are worse than two absent
ones, because the suite then says the rules are held. Neither is in the table
above, and this document does not hold them.
