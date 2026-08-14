# The network shape an operator has to open

One inbound port number for media, multiplexed over every session, plus the port
the API and signalling answer on. Not a range. An operator forwards a small,
fixed, documented set once and never revisits it.

Recorded for issue #14.

## Why this is a decision and not an implementation detail

The audience is behind a home router or a small firewall. Media servers habitually
ask for a wide inbound range, and that is where a self-hosting attempt usually
dies. It dies quietly, because the service starts, the page loads, the room opens,
and only the media fails. The operator has nothing to read that says which of the
four layers between them and the other participant did not work.

Multiplexing every session over one port constrains the transport library choice,
the way sessions are told apart on arrival, and what happens to a single socket
under load. It also interacts with the relay question, because an operator whose
participants sit behind restrictive address translation needs a relay, and under
the federation record that relay runs on the operator's own host. Deciding those
three in three places produces three answers that do not fit together, so they are
settled here at once.

## The ports

A default deployment needs these and nothing else.

The service port, TCP, inbound, required. It carries the API, the signalling
described in issue #47, and the TLS that protects both. It defaults to 443
because that is the port a browser reaches without being told to, and because a
network that blocks it has blocked the web rather than this service.

The media port, UDP, inbound, required. One port number, one socket, every
session. It carries the encrypted media and the connectivity checks that precede
it. It has no privileged default, because binding below 1024 would mean the
service starts with a capability the deployment contract does not ask for and the
headless test job in issue #18 refuses.

The same media port number, TCP, inbound, optional. Some networks pass no UDP at
all, and for those the media rides a TCP connection instead, at a real cost in
delay and in behaviour under loss that a participant hears. It is the same
number on purpose, so that the instruction to the operator stays one number. An
operator who does not open it has a working service for every participant whose
network passes UDP.

The certificate challenge port, TCP, inbound, conditionally required. Obtaining a
certificate over a challenge that runs on port 80 requires port 80 to be
reachable. Which challenge a default deployment uses is issue #57, and this
record does not settle it, so the port is listed as conditional and the open
question stays where a reader meets it. An operator who supplies their own
certificate needs none of it.

Outbound, a default deployment opens name resolution and the certificate
exception named in the federation record. Nothing else. That record is the
authority for the list and this one does not restate it.

There is no separate port for metrics, health or administration in a default
deployment. Where an operator asks for metrics, they say where those are served,
which is one of the four configurable categories in the configuration record.

## Address families

A default deployment listens on both IPv4 and IPv6, and gathers candidates on
both.

This is the part usually left to whatever the first implementation happened to do,
and it fails the same way a wide port range fails. Some of this audience sits
behind a shared public address that is not theirs to forward a port on. Some sits
where the only path that works is IPv6. A service that gathers on one family
serves one of those and fails the other, and it fails after the room has opened.

Both families are treated as ordinary rather than one being a fallback. A
deployment on a host with no IPv6 address gathers no IPv6 candidates and says so
at startup, and that line is a fact about the host.

## What an operator on a shared public address is told, and when

At startup, on every start and not only the first, and before any participant
has tried to join.

The self-check in issue #81 reads the addresses the host actually holds. Where the
only address in a family is one that cannot be reached from outside, the operator
is told that, in that family, by name: an address inside the shared address space
that carriers use for this, or a private address with no public address anywhere on
the host. Both are local observations. Neither requires asking anything outside the
host, which is what keeps this check inside the federation record and out of a
second exception to it.

What the operator is told is what that means for them: participants outside their
network will not reach this host on that family, the other family or the relay
below is what has to carry it, and the port they forwarded will not help. What they
are not told is that the deployment is broken, because it may not be. A host on a
shared IPv4 address with working IPv6 serves every participant who has IPv6.

The honest limit is that this check cannot prove reachability. A host with a public
address and a router that was never configured looks identical from the inside to
one that is correctly forwarded. Proving the difference means asking something
outside the host, and this project does not do that by default. So the check
catches the case it can catch, says nothing about the case it cannot, and the
diagnostic below is what covers the rest.

## The relay, for networks that pass nothing else

Two participants both behind address translation that refuses to be traversed have
no direct path, and the only remaining answer is a relay that both can reach.

That relay runs on the operator's own host, as part of the same deployment, on the
media port already listed above. It is not a separate service to stand up, not a
public one, and not an outbound connection to somebody else's machine. Issue #120
builds it, and the federation record is where the refusal of a public relay is
argued.

Its cost is stated rather than hidden: media that is relayed passes through the
operator's host in both directions, so a relayed participant costs upload as well
as download on a connection whose upload is usually the scarce half. That cost
lands on the operator, which is the right place for it and is still a cost.

## The cost of one media port

Every session shares one socket and one receive path. That concentrates load where
a range would spread it, and the failure mode is not gradual: a receive path that
falls behind loses packets for every session at once rather than for one.

This is a performance question, and it is answered by measurement. The bench in
issue #8 is the instrument. Issue #67 draws the cost per stream as concurrency
rises, which is where a per-socket ceiling would appear as a bend in the curve,
and issue #70 measures what happens past the ceiling. Until those have run, the
position in this record is a design choice whose cost has not been measured, and
it should be read that way.

If the measurement shows the single socket is the limit before the budget in issue
#10 is reached, what changes is the receive path, and the operator's instruction
stays as it is. Widening the port range is the answer this record refuses, because
it moves the cost onto the person least equipped to pay it.

## What an operator gets when media does not flow

A diagnostic that names the layer that failed, rather than a service that started
and a room that is silent.

Issue #62 builds it. It reports, in order, whether the name resolves to this host,
whether the service port accepted a connection, whether the media port received
anything at all from the participant, whether connectivity checks completed, and
whether media arrived after they did. The first of those that did not happen is the
answer, and it is the sentence an operator can act on or paste into a report.

It is a separate issue because a document cannot observe what happened, and an
install guide is a document. Issues #56 and #81 hold the parts
that run before a participant exists, and #62 holds the part that runs after one
has tried.

## What is out of scope here

Which ports a second host would need in a pool, which belongs to the growth record
and its milestone.

Whether the media port number is configurable, which the configuration record
already answers: the media listening socket is an operator input, because it has to
match what the operator opened.

Anything about federation, which opens no port in a default deployment because it
is off.
