# Admission and what a credential may carry

This project does not know who anybody is. Admission works off a credential the
service above it hands over, and that credential describes a room, a set of
powers, and nothing about a person.

Recorded for issue #11. Every later issue that touches admission or authorisation
is checked against this record.

## What a credential names

Three things.

The room it admits to, named by the identifier this project gave the service above
when the room was opened.

A participant identifier that is opaque here. It is a label this project carries
so that events can be correlated and a participant can be addressed. It is not
resolved, not parsed, and not assumed to mean anything outside the service that
issued it.

A set of powers, which is the subject of issue #46 and is named here only as the
third field so that authorisation has somewhere to live from the start.

## What a credential may never carry

Anything about a human being. No name, no display name, no address, no email, no
account identifier that means something outside the issuing service.

This is where admission meets the sovereignty position. A credential carrying a
display name puts personal data into this project's logs, metrics, event stream
and crash output by default, on the first join, before anybody has decided that it
should be there. No later redaction gets it back out of what has already been
written.

There is no mechanism for this today. It is a rule that a person enforces by
reading a change, and nothing in the tree refuses a credential with a name in it.
Issue #124 is open for the mechanism and is named here so the gap is visible
rather than assumed closed. Until it lands, this paragraph is the whole of the
enforcement, and it should be read as weaker than the sentence above it sounds.

The shape that issue takes is a closed set of permitted claims with everything
else refused, rather than a list of forbidden fields, because the prohibition is
about meaning and not about shape: an opaque participant identifier and an
account identifier are the same bytes.

## How long it is valid

A credential is valid for a short window that covers a join and not a session. It
carries a moment before which it is not valid and a moment after which it is not,
and both are checked.

The forwarding session outlives the credential deliberately. A participant who has
joined stays joined because the session says so, not because the credential is
still good. That keeps the credential a key to a door rather than a thing this
project has to keep re-reading.

Short windows limit what a leaked credential is worth, and they put a clock
dependency in the join path. That is a real cost and it is paid rather than
avoided, because the alternative is a long-lived token whose leak has no bound.

## Who signs

Public key. The service above holds a private key and mints. This project holds
the corresponding public key and verifies.

The reason is that it separates two powers that a shared secret joins. With a
shared secret, every host that can check a credential can also issue one, so a
compromised media host becomes an admission authority for every room it can name.
The cost of separating them is key distribution and rotation, which is work this
project has to do and would not otherwise have.

This project can also mint, and that is a separate power rather than the same one.
Issue #45 builds both sides. A deployment that only verifies must be able to run
without any minting key present, so that the separation above is a property of the
deployment and not a habit.

## Revocation

A signed credential cannot be unsigned, so the honest answer has three parts and
none of them is a revocation list.

The short validity window is the primary control. A credential that has leaked
stops being useful quickly, and how quickly is the number the window sets.

A participant who is already in a room is removed by acting on the session, not on
the credential. That path is the same one issue #38 builds for stopping a track,
and it works because the session is this project's own state.

A key that has been compromised is rotated, which invalidates everything signed
by it at once. It is blunt, it interrupts every pending join issued under that
key, and it is the only tool that reaches credentials already handed out. Rotation
being available is why key distribution is accepted as a cost above.

## The clock

Admission assumes this host's clock and the issuing service's clock agree within
a stated tolerance, and that tolerance is part of the join contract rather than an
implementation detail.

When the assumption is violated, admission fails closed. A credential that appears
not yet valid or already expired is refused, and the refusal says that the times
did not line up rather than reporting a bad credential, because an operator whose
clock has drifted needs to be told that and not sent looking at their token
minting.

Skew is a property of the host, so it belongs in the startup self-check in issue
#81 rather than being discovered at the first failed join. The audit trail in
issue #84 records the refusal along with every other admission decision.
