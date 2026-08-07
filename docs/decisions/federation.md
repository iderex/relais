# Federation, and nothing leaving the host by default

Personal data does not leave the operator's host unless the operator deliberately
federates. A default deployment opens no outbound connection except the one named
below.

Recorded for issue #13.

## Why this needs a record rather than a sentence in the readme

Software of this kind sends data outward by default, and it does so without
anybody choosing it. The defaults are helpful, they are copied from examples, and
each one is individually reasonable. The result is a service that quietly tells
third parties who is talking to whom. Naming each of those routes and closing it
is the only way the claim survives contact with the code.

## The routes out, each closed by name

A public connectivity server used as a default tells a third party the network
address of every participant, which is the most identifying thing this service
handles. None is configured out of the box. A deployment that has not been given
one has none.

A public relay used as a fallback sends the media itself through somebody else's
machine. None is configured out of the box either. Operators whose participants
sit behind restrictive networks need a relay, and that relay runs on the
operator's own host as part of the deployment, which is issue #120.

Update checks are absent. The service does not ask anything whether a newer
version exists.

Crash reporting is absent. A crash produces output on the operator's host and
goes nowhere.

Usage telemetry is absent, at every level. There is none that is off by default
and can be switched on, because a switch is a thing that gets switched, by an
operator who was told it would help or by a default that changed in a later
release.

A metrics endpoint is not reachable unless the operator asked for it. Metrics are
served where the operator put them, which is one of the four configurable
categories in the configuration record, and there is no address they appear on
merely because the service started. Issue #80 builds them and issue #82 holds the
separate rule that neither logs nor metrics carry conversation content.

## The certificate exception

There is one outbound connection in a default deployment and it is not being
argued away.

Obtaining a certificate for the name the service answers on means talking to a
public issuing service. That service learns the hostname being certified, and the
certificate is published to public transparency logs, where anyone can read it.
For a self-hosted deployment a hostname is frequently identifying: it can name the
community, the organisation or the person running it, and that is disclosed to
the issuing service, to the transparency logs, and thereafter to anyone who
watches them.

Nothing about any participant is disclosed by this. That is a real limit on the
exposure and it is not a reason to describe the disclosure as small.

The exception is written into the sovereignty statement in issue #101 with the
same wording rather than a shorter one, and issue #57 builds the certificate
handling. An operator who will not accept it supplies their own certificate, and
that path exists so that the exception is avoidable rather than mandatory.

## What federation would mean here

Federation is two hosts running this service carrying media or signalling between
each other so that participants on both are in one room.

Nothing in the plan builds it today. This record defines it so that a later
request is answered against a definition rather than against whatever the request
happens to mean.

If it is ever built, four things hold. It is off, and a deployment that has not
been configured for it does not federate. It takes a configured peer, named by the
operator, rather than discovering peers. It is visible while it is on, in the
service's own state and in what an operator can see, so that federating is not a
condition somebody has to infer. And the operator's own data protection
documentation states what crosses the boundary, which under this seam is media,
the opaque participant identifiers, and network addresses.

Nothing enables it as a side effect of another setting. A setting that turns on
federation does one thing and says so.

## What proves this rather than asserts it

Issue #103 builds the test. It brings up a default deployment, observes every
outbound connection attempt it makes, and reds on any that is not the certificate
exception above.

The test has to observe attempts rather than successful connections, because a
deployment on a host with no route out would otherwise pass while doing exactly
the thing this record forbids.

Until that test exists, everything above is a description of what the code is
meant to do, and no run of anything has confirmed it.
