# Threat model

This service accepts connections from strangers, carries conversation, and runs on
a machine the operator also uses for other things. This document turns that from a
feeling into a list somebody can argue with.

Recorded for issue #104.

Every mitigation named below points at a mechanism that exists or at an open issue
that builds it. Where a threat has no mitigation, it is listed as having none
rather than being given a control that does not exist. Almost nothing here is
built yet, and the issue numbers are what let a reader tell which is which.

## What is being protected

The confidentiality of conversations. Nobody who is not in a room should be able
to hear or see what is said in it.

The availability of conversations. A room that is running should keep running, and
one that can be started should be startable. For the audience this is aimed at,
an outage is the failure that actually happens.

The host. This runs on a machine that belongs to a person or a small community and
usually does other things. A compromise of this service must not become a
compromise of that machine.

The operator's ability to know what happened. An operator who cannot answer what
the service did cannot answer for it, to their participants or to anybody else.

## The adversaries, and what each can reach

An unauthenticated stranger on the network. They can reach every port a default
deployment opens, which the network shape record lists: the service port, the
media port, and the certificate challenge port where that route is used. They can
send arbitrary bytes to all of them, at any rate, from any number of sources.
They hold no credential. They can also watch the certificate transparency logs and
learn the hostname of every deployment that obtained a certificate, which the
federation record names as a real disclosure.

A participant holding a valid credential. They are inside a room. They can publish
media, subscribe to what is published, and ask this service for things at whatever
rate they like. They can send malformed media, media that is not what they said it
was, and volumes that are wrong for what the room needs. They see the addresses of
other participants where a direct connection is established, which is a property
of how this class of software works and not a defect this project introduced. They
can record everything they receive, which is their own screen and their own
speakers.

The operator of the service above. They mint credentials, so they decide who is in
which room and with which powers. They are trusted to that extent by construction:
this project verifies a signature and does not second-guess who it names. What
they cannot do is reach into this service's state, read a conversation they are
not in, or learn anything about a participant this service does not already know,
which is nothing beyond an opaque identifier.

A hostile dependency. Something this project links or an action a workflow runs.
It executes with the privileges of the process or the job it is inside. This is
the adversary that most cheaply reaches everything, and it is the one a reader
usually skips.

Whoever carries the traffic. A network operator, a transit provider, or anybody
between a participant and the host. They see that a connection exists, its
addresses, its timing and its volume. They do not see the media, which is
encrypted between each participant and this service.

The operator of the host is not on this list. They own the machine, they can read
the memory of the process, and defending against them is not a coherent goal.
What is a coherent goal is that they can find out what the service did, which is
why the last asset above is an asset.

## The threats, and what stands against each

Malformed or hostile media reaching the forwarding path. This is the largest
surface and the one adversaries reach without a credential. Against it: the
forwarding core holds no decoder, which the media formats record fixes as a
property and which removes the most attacked parsing surface in this class of
software. Refusing malformed and hostile input at the media boundary is #39,
which is open, and nothing in the tree refuses it. #91 is closed and its fuzzing
landed in `test/fuzz`, pointed at the credential parser rather than at this
surface, which the parity document already says in its own words. So the media
boundary has neither the refusal nor a fuzzer, and it has no bytes reaching it
either, because the forwarding core is not built. This paragraph said `Neither
exists yet` after #91 had closed, which is #178.

Exhaustion of the host by volume, from a stranger or from a participant. Against
it: rate limits and size limits at the edge, #50. Failing closed when the pool
cannot grow, #77, so that a host under pressure refuses rather than degrading
everything it is already carrying. Measuring what happens past the ceiling so the
failure is predictable rather than discovered, #70. None exists yet, and until
they do a stranger who can reach the media port can spend the host's resources.
That is stated rather than mitigated.

A forged or replayed credential. Against it: signatures verified against a public
key this deployment holds, with the minting key elsewhere, and a validity window
short enough that a leaked credential expires rather than being revoked. Both are
in the admission record, and the verifier that holds them is
`internal/orchestration/credential`, landed by #45. A credential is a key to a door
rather than a session, so a forged one that arrives after the window has passed
opens nothing.

A participant doing something their credential does not permit. Against it:
authorisation as a distinct question from admission, #46. Until it lands, a
credential admits and nothing narrows what the holder may then do, which is a real
gap and not a theoretical one.

Personal data entering this service through a credential. Against it: the
admission record forbids it, and verification refuses a credential carrying any
claim outside the set that record permits, naming the claim it refused. That
covers a claim nobody agreed to and nothing else. A permitted claim carrying
something it should not, an account identifier in the participant field, is the
same bytes as an opaque label and no check reaches it, so for that case this is
still a rule enforced by a person reading a change. An operator should read that
as weaker than the record's sentence sounds.

Conversation content leaking into logs, metrics or a bug report. Against it: logs
that carry no conversation content, #82; secrets handled so they cannot reach a
log or a report, #83; an audit trail that holds admission decisions and nothing
more, #84. None exists yet.

Data leaving the host to a third party. Against it: the federation record closes
each route by name, and #103 builds the test that observes outbound connection
attempts and reds on any that is not the certificate exception. The exception
itself is not mitigated and is not being argued away: obtaining a certificate
discloses the hostname to a public issuing service and to public transparency
logs. An operator who will not accept that supplies their own certificate, which
is why #57 keeps that path.

A hostile or compromised dependency. Against it: a locked dependency graph, #20;
the supply chain around it, #97; dependency review, which runs today; static
analysis of the program, which runs today; a second analyser with a different
lens, #88; workflow auditing and pinned actions, which run today. The residual is
that none of this detects a dependency that is malicious and has not been reported
as such, and no amount of pinning does.

A tampered release. Against it: signed release artefacts, #113, and a release
reproducible from a tag, #112, so that an operator can check the artefact they are
running is the one that was published. Verified signatures on the protected
branch, #98. None exists yet, and there is no release.

Reading the media in transit. Against it: the media path is encrypted between each
participant and this service. The limit is exactly stated: this service decrypts,
forwards and re-encrypts, so it can read what it carries. Whether that changes is
entry 5 of issue #1, and the answer there defers end to end encrypted media with
the architecture kept open rather than ruling it out, so it is not in the first
release and this limit is the one to plan against. A deployment's confidentiality
claim today therefore rests on the host and on the operator, and any stronger
claim would be false.

Discovery of who is talking to whom by watching the network. Not mitigated. See
the out-of-scope list below.

## What is out of scope, and why

This is the part most threat models skip, and it is the part that makes the rest
honest.

A compromised host is not defended against. Somebody who can run code on the
machine can read the memory of this process, and no design here changes that.

Traffic analysis is not prevented. Somebody who can watch the network learns that
a connection exists, its endpoints, its timing and its volume. Media servers are
particularly legible this way, and padding or cover traffic would cost the
resource budget this project exists to defend. That is a trade made deliberately
and stated rather than hidden.

A participant recording what they receive is not prevented and cannot be. Their
screen and their speakers are theirs.

Denial of service by an adversary with substantially more capacity than the host
is not defended against. A machine in a home or a small office cannot absorb it,
and pretending otherwise would mean recommending a third party in front of the
service, which contradicts the position the federation record takes.

Legal compulsion of the operator is not addressed here. It is not a technical
threat and this document would be the wrong place to say anything useful about it.

The operator of the service above is trusted to decide who joins. A model in which
they are not is a different product.

## What has no mitigation at all today

Listed separately because a reader skimming the sections above should not have to
assemble it.

Nothing refuses personal data placed in a claim a credential is permitted to
carry. A claim outside the permitted set is refused; an account identifier in the
participant field is the same bytes as an opaque label and is not.

Nothing limits the rate or size of anything arriving at the edge. #50.

Nothing keeps conversation content out of a log, because the logging that would
have to obey the rule does not exist yet. #82.

Nothing narrows what a credential holder may do once admitted. #46.

Nothing observes what this service connects to. #103.

Every one of those is an open issue rather than an oversight, and the list is the
honest state of the tree at this commit rather than a plan.

## What this document is for

[SECURITY.md](../SECURITY.md) is the security policy and the disclosure process.
It is where a reporter is told what is in scope for a report, and it sends them
here for the answer rather than quoting one, so the two cannot drift apart. Issue
#105 is where that document is argued, and what is still open about it is
recorded there rather than restated here.

The line this document draws is between what is claimed and what is built. Almost
every mitigation above is an open issue, and reading this document as a
description of what runs today would be reading it backwards.
