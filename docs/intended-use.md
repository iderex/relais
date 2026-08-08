# What this project is for, and what it is not for

[NOTICE.md](../NOTICE.md) states the general form: this software is developed for
lawful use and the operator answers for their deployment. This document is the
specific form, and it exists because a realtime media backend is a thing people
will try to use for surveillance, for covert recording, and for services that
harvest what passes through them.

Recorded for issue #107.

## Who this is for

Somebody running a conversation service for a community they are part of. A
club, a school, a practice, a small organisation, a group of friends, a public
body that wants its own machine rather than somebody else's. The operator is
usually a participant too, or answerable to the participants directly, and that is
the deployment every decision in this repository is measured against.

The gap this fills is the retreat of free hosted capacity for exactly those
groups. That is the whole of the intent, and the rest of this document is what
follows from it.

## What this project will not support

Covert monitoring of participants. A deployment where people are recorded,
transcribed or profiled without being told is not a deployment this project builds
for, and no feature request that assumes one will be accepted on its merits.

Interception facilities. This project does not build a route for a third party to
attach to a conversation, and it does not build the silent-observer participant
that would be the obvious way to provide one.

Harvesting what passes through. There is no analytics of conversation content and
no route for it. A service built above this one holds identity and can do what it
likes with what its own users tell it; what it cannot do is ask this layer for the
content of a conversation, because this layer keeps none.

These are positions about what gets built here. They are not claims about what is
technically impossible, and the last section says why that distinction matters.

## The design choices that support the position

Each one below is named with the place it is decided and the place it is
implemented. Most of the second column is an open issue, because the tree at this
commit holds no implementation:

    git ls-files internal/ | grep -v README

The service never learns who a participant is. Decided in [the seam
record](decisions/seam.md), which refuses accounts, identity, membership over time
and moderation policy, and in [the admission
record](decisions/admission.md), which allows a credential to name a room, an
opaque participant label and a set of powers, and nothing about a human being.
Implemented by issue #45, which is not built. The mechanism that refuses a
credential carrying a claim outside that set is built and lives in
`internal/orchestration/credential`, and it judges the names of claims and never
their contents, so a permitted claim carrying something it should not is not
reached by it.

Nothing is recorded. There is no recording in the plan and no setting that turns
some on, which [the data protection document](data-protection.md) states as the
current position. Whether that ever changes is entry 2 of issue #1, which is open,
and this paragraph is what would have to be rewritten first. Not built, because
there is nothing to record yet.

The audit trail cannot identify a person. It holds admission decisions and the
opaque identifier, which by design resolves to nobody here. Issue #84 builds it.
Not built.

Logs and metrics carry no conversation content. Issue #82 holds that rule and
issue #80 builds the metrics. Not built.

Nothing is sent anywhere. Update checks, crash reporting and usage telemetry at
every level are closed by name in [the federation
record](decisions/federation.md), and issue #103 is the test that would red on any
outbound connection other than certificate issuance. Not built.

There is no client and no interface, so this project never speaks to a participant
and has no surface on which to ask one for anything. That is a consequence of the
seam. Whether a diagnostic page is ever added is entry 7 of issue #1, which is
open.

## What remains possible for whoever controls the host

An operator of any media server can, in principle, capture what passes through it.
That is true here and a document claiming otherwise would be false.

Whoever controls the host can read the memory of this process, capture the packets
at the network interface, and run a modified build that does whatever they want
with the media it forwards. None of the choices above prevents that, and none of
them is meant to. [The threat model](threat-model.md) says the same thing in its
own words and does not defend against a compromised host.

The reduction this project actually offers is narrower and it is worth stating
exactly. There is no button. Somebody who wants to monitor a community using this
software has to modify it or instrument the machine, which is a deliberate act
they have to take, rather than a setting they can switch on and later describe as
having been on by default. It also means this project does not learn who anybody
is, so a modified build captures conversations attached to opaque labels and has
to go to the service above to find out whose they are.

Participants are told nothing by this project, because it never speaks to them.
Whether they know what their operator is doing is the operator's obligation, and
[the data protection document](data-protection.md) says so in the section on who
is responsible for what.

## What this document does not do

It does not restrict use. The licence decides what anybody may do with this
software, and there is no licence file yet:

    git ls-files LICENSE | wc -l
    0

That is entry 1 of issue #1 and it is open. Until it is answered this document
states an intent and nothing more, and it should not be read as a term.
