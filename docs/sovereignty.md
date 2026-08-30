# Sovereignty: what stays on the operator's host

The claim this project rests on is that an operator's conversations, and the
personal data around them, stay on the operator's host. It is written out below
precisely enough that somebody could go and find it false.

Recorded for issue #101.

## How to read the backing of every claim below

Each statement here is one of three things, and the document says which one every
time.

A statement backed by a mechanism names the code or the configuration that makes
it true, and a reader can go and look at it.

A statement backed by a test names the test that would go red if it stopped being
true, and a reader can run it.

A statement backed by neither is a claim. It says what this project intends the
software to do, and nothing has confirmed it. Most of this document is currently
in that third category, because the tree at this commit builds a program that
reports its own version and nothing else. Where a claim is of that kind, the issue
that would build the thing is named, so a reader can check whether it has landed
rather than assuming it has.

    git ls-files internal/ | grep -v README

That command returning nothing is the honest state of the implementation, and it
is the reason a sovereignty document written today is mostly a specification.

## What data exists, where it lives, and for how long

The purposes are in [the data protection document](data-protection.md) and are not
repeated. What matters here is location and lifetime, because those are what the
claim is about.

Media, meaning the audio and video packets participants send. Location: the memory
of the process on the operator's host, in a buffer measured in fractions of a
second. It reaches no disk on that host and no second host. Lifetime: the time it
takes to forward. Claim, not mechanism. There is no code yet, and the forwarding
core is issue #32 onwards.

Network addresses, meaning the address a participant connects from and the
candidate addresses their software offers while a connection is established.
Location: process memory for the session, and whatever a log holds. Lifetime: the
session, plus the log retention the operator sets. Claim. What a log is allowed to
carry is issue #82 and is not built.

Opaque participant identifiers, meaning the label the service above attaches to a
participant. Location: process memory for the session, and the admission audit
trail. Lifetime: the session, and then the retention the operator sets on that
trail. Claim, with one part of it recorded: the identifier is not resolved and not
parsed here, which is in [the admission record](decisions/admission.md), and
verification refuses a credential carrying any claim outside the set that record
permits. That check reads the names of claims and not their contents, so it does
not reach an account identifier placed in the participant field.

Room identifiers, meaning the name this project issued when a room was opened.
Location: process memory. Lifetime: the room. Claim.

Credentials, meaning the token handed over for a join. Location: process memory
for the moment of the join. Lifetime: the join. Claim, with the rules recorded in
the admission record and issue #45 building them.

Admission decisions, meaning that a join was accepted or refused, with the reason.
Location: wherever the operator puts that trail, on their host. Lifetime: the
operator's retention. Claim. Issue #84 builds it.

Operational measurements, meaning counts, durations and resource figures.
Location: wherever the operator exposed them, which is a place they chose. There
is no address they appear on because the service started. Lifetime: whatever
consumes them. Recorded in [the configuration record](decisions/configuration.md)
and [the federation record](decisions/federation.md); built by issue #80.

Crash output, meaning what the program prints when it fails. Location: the
operator's host. Lifetime: the operator's. Claim that it goes nowhere, and the
route that would carry it away is one of the routes closed by name in the
federation record.

No category above has a copy anywhere except the operator's host. That sentence is
the whole of the sovereignty claim, and everything below is about the ways it
could stop being true.

## What leaves the host

One thing, and it is stated in full below rather than summarised.

Everything else that software of this kind commonly sends outward is closed by
name in [the federation record](decisions/federation.md): a public connectivity
server, a public relay, update checks, crash reporting, usage telemetry at every
level, and a metrics endpoint that appears without the operator asking for one.
That record is the authority for the list and this document does not restate it,
because a second copy of a list drifts against the first.

Federation itself, meaning two hosts running this service carrying media or
signalling between each other, is not built. If it is ever built it is off unless
the operator configures a named peer, and the same record says what would cross
the boundary.

## The certificate exception

Obtaining a certificate for the name the service answers on means talking to a
public issuing service. That service learns the hostname being certified, and the
certificate is published to public transparency logs, where anyone can read it.
For a self-hosted deployment a hostname is frequently identifying: it can name the
community, the organisation or the person running it, and that is disclosed to the
issuing service, to the transparency logs, and thereafter to anyone who watches
them.

Nothing about any participant is disclosed by this. That is a real limit on the
exposure and it is not a reason to describe the disclosure as small.

An operator who will not accept it supplies their own certificate. Issue #57
builds the certificate handling and that path exists so the exception is avoidable
rather than mandatory.

The wording above is the federation record's own wording, kept rather than
shortened, because a shorter version of a disclosure is a weaker one.

## What a third party learns by watching

Self-hosting moves data off somebody else's machine. It does not make the traffic
invisible, and a document that implied otherwise would be worse than none.

Whoever carries the operator's traffic sees that a host is running a realtime
media service, sees which addresses connect to it, sees when and for how long, and
sees how much data flows in each direction. That is enough to know who was in a
conversation with whom and for how long, without any of the content.

The content itself is not readable in transit. The media path is encrypted between
each participant and this service. That is a property of the transport this
project uses rather than a thing this project adds, and it is a claim until there
is a session to observe.

Traffic analysis of the kind above is not defended against here. [The threat
model](threat-model.md) is where the defended and undefended sets are drawn, and
this document does not draw a second one.

## The limits, stated without softening

This service can see the media it forwards. A forwarding unit reads enough of each
packet to route it, and today it can read all of it. End to end encrypted media
would reduce that and would not remove it, because the frame headers have to stay
readable for forwarding to work. Whether end to end encryption is a first release
promise is entry 5 of issue #1, and the answer there is that it is not: it is
deferred with the architecture kept open, ruled out neither now nor later. This
document is where that answer is read rather than where it is made, and what it
means here is the plain version. The first release does not carry it, so the
sentence above is the whole of what this project can claim about what it reads,
and it is not a state that ends on its own.

The operator's host is a machine somebody has to secure. Everything above says
data stays on that host. None of it says the host is safe. An operator who runs
this on a machine with a weak password has moved their participants' data from a
company they did not choose to a machine that is easier to break into, and this
project cannot tell them which of the two is worse for them.

Nothing here has been audited, and nothing here has been observed, because there
is nothing running to observe.

## What would make this document trustworthy

Issue #103 is the test. It brings up a default deployment, observes every outbound
connection attempt through a full session, and reds on any attempt that is not the
certificate exception above. It observes attempts rather than successful
connections, because a deployment on a host with no route out would otherwise pass
while doing the thing this document forbids.

Until that test exists and runs, the central claim of this document is a claim.
When it exists, this section names it as the mechanism and the sentence changes
from an intention to a measurement. That change is the only thing that should ever
upgrade the language in this document.
