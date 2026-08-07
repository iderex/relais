# How far this project grows past one host

One host has a ceiling. This project owns the parts of what happens at that
ceiling that only a media plane can supply, ships a default that copes without
anybody being asked to think about it, and owns no placement policy for a service
that has its own.

Recorded for issue #12. It reaches into another project's plan, so it is settled
here rather than discovered later.

## What this project owns

A capacity signal derived from measurement rather than configured. A number an
operator typed is a number about the machine they imagined, and the whole argument
of this repository is that the resource behaviour is measured. Issue #73 builds it
and has to show that it leads quality rather than trailing it, because a signal
that goes red after the media is already bad is a report and not a signal.

A pool with an authoritative view of which hosts exist and what state each is in.
Something has to be able to answer the question, and every consumer answering it
separately from an event stream produces as many views as consumers. Issue #74
builds it.

Placing a new room onto a host. This is the mechanism rather than the policy: the
ability to say that this room goes on that host and have it be true. Issue #75.

Draining a host and retiring it without dropping a live session. Only the media
plane knows what a live session is and when it has ended, so nothing above the
port can do this correctly. Issue #76.

A default policy that grows and shrinks the pool. It exists so that a single
operator who wants nothing to do with any of this gets a service that copes, which
is the deployment contract's promise applied to the second host.

## What this project does not own

Placement policy for a service that has its own.

The conferencing project at github.com/iderex/hoersaal plans placement, cascading
and an autoscaling loop as its own work. If this project also owned those, two
plans would build the same thing and the seam record would become untrue on the
day the second one landed.

Under this arrangement that project still builds for itself the policy that
decides which host a scheduled room belongs on, together with everything on the
seam record's refuse list that it already owns.

There is a live cross-project question underneath this, which is whether that
project delegates its pool to this one or keeps its own placer. It belongs to the
maintainer and this record does not answer it. The plan works either way, because
the default policy is replaceable rather than mandatory.

## What makes the default replaceable

The capacity signal and the pool view are part of the operator API rather than
internal state, and placement is an operation on that API rather than something
that only happens as a side effect of a room being opened.

A consuming service that wants to place rooms itself reads the signal, reads the
pool view, and calls the placement operation. It does not have to switch the
default off first, because a room placed explicitly is placed where it was told
and the default only acts on a room that arrived without an instruction.

That is the whole of the mechanism, and it is the reason the pool view is API
surface rather than an implementation detail. Issue #43 writes the API contract and
this record is one of the things it is checked against.

## What is out of scope at first release

One room spanning several hosts. It is a different and much harder problem than
one room per host: it needs media to cross between hosts, a policy for which
participant is on which, and a failure story for the link between them. Stating it
as out of scope here is what stops it being assumed by an issue that only needed
one room per host.

Geographic distribution. It is the previous entry with latency and law attached.

Provisioning machines. This project can drive an interface that creates a host, and
it does not become the thing that talks to a hosting provider's API, holds that
credential, or knows what a machine costs.

## The ceiling this is all about

The number of streams one host carries before quality degrades is not known. It
comes from the resource budget in issue #10, which comes from the bench in issue
#8, and neither has produced a figure yet.

That matters to this record in one specific way: a pool that grows is only worth
building if the ceiling it grows past is real and measured. Until the budget
exists, every issue in the growth milestone is building a mechanism against a
ceiling nobody has seen. The mechanisms are still right, and the trigger points of
the default policy are not written here because they would be invented rather than
measured.
