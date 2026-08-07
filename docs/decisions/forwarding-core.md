# The forwarding core

This project takes the commodity parts of a media server from a mature transport
library and owns the routing and allocation policy above them.

Recorded for issue #3.

## What is taken from the library

Connectivity establishment through home routers, the handshake, encryption of the
media path, congestion control, retransmission and jitter handling. These are
years of work, their failure modes appear under load rather than in a unit test,
and rewriting them serves nobody who wants to run this software.

## What this project owns

Which of the available streams a given subscriber receives. Which simulcast layer
it receives. When a keyframe is requested. How a fixed amount of bandwidth and
memory is divided when there is not enough of either, and what is given up first.
These are the decisions that set the resource behaviour, and the resource budget
is one of the two things this project exists to offer.

## The library

pion/webrtc. It is a library rather than a server, it is MIT licensed, and it is
the transport underneath one of the complete units below, so choosing it does not
bet on any single product's roadmap.

The candidates, read at the commit this record lands on:

    for r in pion/webrtc versatica/mediasoup meetecho/janus-gateway livekit/livekit jech/galene; do gh api "repos/$r" --jq '"\(.full_name) \(.license.spdx_id) \(.language) \(.pushed_at)"'; done
    pion/webrtc MIT Go 2026-08-06T21:48:58Z
    versatica/mediasoup ISC C++ 2026-08-06T11:42:05Z
    meetecho/janus-gateway GPL-3.0 C 2026-07-27T13:27:11Z
    livekit/livekit Apache-2.0 Go 2026-08-07T09:59:44Z
    jech/galene MIT Go 2026-07-28T13:52:55Z

Why each of the others was not taken.

versatica/mediasoup is a library, and it is driven from a second runtime. Taking
it means this repository carries two languages and two dependency graphs before it
carries a line of forwarding policy, and every later piece of work that locks,
scans or reproduces the dependency graph pays for that twice.

meetecho/janus-gateway is GPL-3.0. Entry 1 of issue #1 is open, so the licence of
this repository is not settled, and a dependency whose licence would settle it by
implication is not a choice this plan is allowed to make. This record does not
choose it and does not rule it out on its merits. If entry 1 is answered in a way
that permits it, the comparison is worth running again.

livekit/livekit and jech/galene are complete selective forwarding units. A
complete unit used as a library is not a library, because its routing and
allocation decisions are the product. Taking either would mean wrapping those
decisions rather than owning them, which is the position refused below.

## Why not wrap a complete unit

Wrapping buys the commodity parts and the opinionated parts in one move. It also
means that what this repository offers, one command to a working backend and a
resource budget defended with numbers, is a repackaging of a server whose
footprint is not ours to move. When the budget does not hold, the only action
available is to report it upstream and wait. Both of the nearest complete units
are already deployable as a single binary, so a wrapper would have to argue what
it added to them.

## Why not own the whole byte path

Owning connectivity, the handshake, encryption and congestion control buys total
control and costs the commodity list above. The audience is self-hosters behind
home routers. They need connectivity establishment that works through those
routers far more than they need a transport nobody outside this repository has
run.

## Residual risk

This project takes on layer selection, keyframe policy and bandwidth allocation.
They are subtle, they behave differently under load than in a test, and they are
the exact places a media server is usually wrong. Two things hold that risk and
neither of them is a promise. The port in issue #4 is meant to keep an adapter for
a complete unit a bounded project rather than a rewrite. The bench in issue #8 is
meant to turn a regression into a red number rather than a report from somebody
who is already running this. Whether either holds is not known until they exist
and have caught something.

## What reopens this

If this project cannot hold a stated cost per forwarded stream within a stated
margin of a packaged existing unit at a stated concurrency, wrapping is revisited.

The three numbers are not fixed here, because none of them has been measured yet.
The cost per stream and the margin come from the resource budget in issue #10, and
the concurrency comes with them. The bench in issue #8 produces the comparison
against a packaged existing unit, and issue #9 is where the packaged units are
first measured on it. The first bench run that produces all three is where they
are written into this record.
