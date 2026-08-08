# The API contract

The surface a service above this project talks to. It is written before there is
an implementation to describe, because a contract derived from an implementation
is a changelog of that implementation.

Recorded for issue #43.

## What this is, and what it is not

[The media plane port record](../decisions/media-plane-port.md) draws a boundary
inside one process, between everything that orchestrates and everything that holds
a packet. This document draws the other boundary, between this project and the
service that runs above it, and the two are not the same line. Several things the
port carries are absent here, and several things here never reach the port.

The constraint that shapes it is that it has to be implementable by something that
is not this project. That is what stops it from becoming a description of internal
structures, and it is why the conformance suite in issue #51 is written against
this document rather than against the code.

Checked against [the seam record](../decisions/seam.md): this contract carries
nothing from that record's refuse list. There is no account, no registration, no
password, no channel or server structure, no presence, no participant list for
anybody who has not joined, no chat, no application data path between
participants, no moderation policy, no bot vocabulary, no client concern and no
identity federation. The one place the boundary is close is stopping a track,
which is a mechanical operation here and carries no rule about who may ask for it.

## The resources

Four, and every one of them exists only while the media it describes exists. None
of them is a record of anything that happened.

A room. The container everything else hangs on. Its identifier is minted by the
caller when the room is opened, so the caller can address the room before this
project has answered. It exists from the moment it is opened until it is closed,
and closing it is the only way it disappears. It holds no memory of who was in it,
and a second room opened under the same identifier after the first was closed is a
different room that happens to share a name.

A participant. One connected session. Its identifier is minted by the caller and
is opaque here, which [the admission record](../decisions/admission.md) fixes and
this contract does not restate. It exists from admission until it is revoked, until
its session detaches, or until the room closes, whichever comes first. It carries
no history: a participant that detaches and is admitted again under the same
identifier is a new participant, and nothing here says the two were the same
person, because nothing here knows what a person is.

A track. One media stream from one participant. Its identifier is minted by this
project, not by the caller, because a track names something only the media side can
observe. It exists from the moment the publisher's stream appears until it ends or
its publisher leaves. It carries a description: which participant published it,
whether it is audio or video, and the layers currently available.

A subscription. One subscriber receiving one track at one target. Its identifier is
minted by the caller for the same reason a room's is. It exists from the moment it
is created until it is ended, until the track ends, or until either side leaves.
Whether media is flowing under it is a separate question from whether it exists,
and the events say which.

A layer is not a resource. It is part of a track's description, named by this
project, and a caller addresses it only inside a subscription target.

## The operations

Every operation states what it means, whether it is idempotent, and what a retry
does. Two rules hold across all of them.

No operation waits on a participant's software to answer. Each returns once this
project has recorded the intent, and the outcome arrives on the event stream. A
caller that treats a successful return as "the media is flowing" has misread every
operation in this list.

Every identifier a caller mints is the caller's idempotency key. A retry of a
create operation with the same identifier is the operation the contract describes
below, not a second request that happens to look similar.

### Open a room

Creates the room. Idempotent by room identifier: a repeat naming a room that is
already open and was opened by the same caller succeeds and creates nothing. A
repeat naming a room identifier that is open under different parameters fails with
`already-exists`, because silently returning success would tell the caller its
parameters took effect when they did not. A retry after a failure of class
`transient` is safe and is the intended response to it.

### Close a room

Ends every session in the room and releases everything it held. Idempotent:
closing a room that is not known succeeds, because the caller asked for a state
that already holds. A retry is always safe. It returns once the room is marked
closed and refuses every later operation naming it; each session ending arrives as
its own event.

### Admit a participant

Records that a participant may attach a session to the room, against a credential
this project has verified. Idempotent by participant identifier within a room: a
repeat for a participant already admitted under the same credential succeeds and
admits nothing further. A repeat under a different credential fails with
`already-exists` rather than replacing the powers of a live participant, because a
caller that wants different powers is describing a different admission and should
say so by revoking first.

A retry is safe. A retry after `credential-invalid` is not useful, because the
credential does not become valid by being sent twice, and the short validity
window in the admission record means a retry after a long pause fails for a
different reason than the first attempt did.

### Revoke a participant

Detaches the participant's session, ends every track it published and every
subscription it held, and refuses a later attachment under the same identifier
within the room. Idempotent: revoking a participant who is not present succeeds. A
retry is always safe. The detachment completing arrives as an event.

### Stop a published track, and resume it

One operation with a direction. Idempotent in each direction: stopping a stopped
track succeeds and changes nothing, and the same holds for resuming. A retry is
safe. The effect arrives as a changed track description rather than in the reply.

This operation decides nothing about who may ask for it. Issue #38 owns what it
means to a consuming service, and issue #46 owns whether a given credential
carries the power to use it.

### Subscribe

Creates a subscription from one participant to one track with a target, which is
either a named layer or the instruction to let this project choose. Idempotent by
subscription identifier: a repeat with the same identifier and the same target
succeeds and creates nothing. A repeat with the same identifier and a different
target fails with `already-exists`; changing a target is the next operation and not
this one.

A repeat under a different subscription identifier for a subscriber that is already
subscribed to that track fails with `already-exists` as well, and it names the
existing subscription so the caller can find what it forgot.

A retry is safe. Whether media then flows is not part of the reply and arrives as
an event.

### Retarget a subscription

Changes which layer a live subscription receives without tearing it down. It exists
as its own operation rather than as an unsubscribe followed by a subscribe, because
those two produce a visible gap and this must not. Idempotent by target: retargeting
to the layer already targeted succeeds and changes nothing. A retry is safe. On
failure the subscription keeps the target it had, so a failed retarget never leaves
a subscription in a state the caller did not ask for.

### Unsubscribe

Ends one subscription. Idempotent: unsubscribing an unknown subscription succeeds.
A retry is always safe. This project stops sending under the subscription before
the event that reports it ended, so a caller that acts on the event is acting on
something already true.

### Read a room, a participant, a track or a subscription

Returns the current description. Safe and idempotent by construction, and a retry
is always safe. These exist because the event stream is allowed to lose events in
one stated way, and a consumer that has been told it fell behind needs a route back
to the truth that is not replay.

A read reflects what this project holds now. It is not a snapshot a caller can
compare against an event to detect a gap, because the two are answered at different
moments.

### Read capacity

Returns what the host can still carry. Safe, idempotent, and a retry is always
safe. It has no failure of class `refused`: a host that cannot compute a figure
answers `unavailable`, which is a value the caller can act on rather than an error
it has to interpret. Issue #73 owns what the signal is derived from and issue #74
owns the pool that reads it.

### Drain

Puts this host into a state where it accepts no new room and no new admission,
without ending what is already running. Idempotent: draining a drained host
succeeds. A retry is safe. There is no undrain, and that is the port record's
decision rather than this one's: a drained host is retired, which is issue #76.

### Read the event stream

Opens the stream described below. Not idempotent in the sense the others are,
because each call produces a stream rather than a state change, and two calls
produce two streams. A retry after a dropped connection is the ordinary case and is
safe. What a caller is guaranteed to see after reconnecting is in the delivery
section and is deliberately weaker than "everything".

## The events

The stream carries what this project observed. It never carries a request, and
nothing a caller does answers an event.

A session attached. A session detached, with the reason.

A track appeared, with its description. A track's layers changed, with the new
description, which is one event rather than a layer appearing and a layer
disappearing, because an allocation is decided against the whole set and two events
would let a consumer act on half of one. A track ended.

A subscription is receiving media. A subscription stopped receiving media, with the
reason. A subscription changed the layer it is receiving, raised whether the change
came from a retarget or from this project's own allocation, so a consumer sees the
same thing in both cases.

Capacity changed.

This project is failing, with the reason. This is the event that says the service
cannot do its job, as distinct from one session going wrong, and it exists so that
a readiness signal and a startup self-check have something to read.

Nothing in that list names a person, a display name or an address. A consumer that
wants to show a name holds the name itself and joins it to the opaque participant
identifier, which is the direction the seam record fixes.

## What a caller is guaranteed to observe, and what it may miss

Three rules, and they are the part of this contract most likely to be assumed
wrongly.

Events are ordered per room and not across rooms. A consumer that infers an
ordering between two rooms from the order it received two events has read something
this contract does not say. A total order across every room would be a
serialisation point in the place this project can least afford one.

The stream is lossy in exactly one way, and a consumer is always told. A consumer
that falls behind is sent a notice saying it fell behind and how many events it
missed, rather than being trimmed silently or being allowed to grow this project's
memory without bound. That property comes from the port record and issue #48
carries it outward; neither may weaken it.

Every state reachable through the event stream is also reachable through a read.
That is what makes the loss recoverable: a consumer that has been told it missed
events re-reads the resources it cares about instead of asking for replay, and this
contract promises no replay.

What follows from the three together is that the event stream is a way to learn
quickly and the read operations are the authority. A consumer built the other way
round is correct only while nothing falls behind.

## The errors

Every error carries three things: a machine-readable reason from the closed set
below, a class from the three below, and a human sentence. The class is a field the
caller reads, not something inferred from the reason or from the prose, so a caller
handling an error it has never seen still knows what to do with it.

The three classes.

`caller` means the request was wrong. Sending it again unchanged produces the same
error. Something in the request has to change.

`transient` means the request was well formed and this project cannot carry it
right now. Sending it again later is meaningful and is the intended response.

`refused` means the request was well formed and this project will not carry it.
Sending it again does not help and neither does changing the request. The condition
is on this side, and the caller is expected to stop rather than to look for another
route.

The closed set of reasons. An implementation that needs a reason outside it adds
one to this document first.

| reason | class | what it means |
| --- | --- | --- |
| `malformed` | `caller` | the request could not be understood |
| `unknown` | `caller` | the room, participant, track or subscription named does not exist here |
| `already-exists` | `caller` | the identifier is in use and the request is not the idempotent repeat described above |
| `credential-invalid` | `caller` | the credential's signature or validity window did not hold |
| `unauthorised` | `caller` | the credential is good and does not carry the power this operation needs |
| `too-large` | `caller` | the request exceeded a size limit |
| `at-capacity` | `transient` | well formed, and this host cannot carry it |
| `unavailable` | `transient` | this project cannot answer right now, and the answer is absent rather than negative |
| `rate-limited` | `transient` | the caller is sending faster than the limit allows |
| `closing` | `refused` | the room, or this project, is shutting down |
| `refused-by-media` | `refused` | the request contradicts what the media side observes, a named layer the track does not report being the ordinary instance |
| `clock-skew` | `refused` | the credential's times and this host's clock did not line up |
| `unsupported-version` | `refused` | the caller asked for a version this project does not serve |

Three notes on the set, each of which is a decision rather than a detail.

`clock-skew` is separate from `credential-invalid` and is class `refused` rather
than `caller`, which looks wrong until the repair is considered. The caller's
request is fine and its credential may be fine; what is out of step is a clock, on
one host or the other. A caller that retried or re-minted would be working on the
wrong thing. The admission record requires the refusal to say the times did not
line up rather than reporting a bad credential, and the class is how that survives
a caller that reads no prose.

`unknown` is class `caller` rather than `refused` because the repair is in the
request: the caller named something that is not here. It is separate from
`already-exists` so a caller never has to tell the two apart by inspecting state.

`at-capacity` is the only reason for which retrying the identical request is
meaningful without anything else happening first. `unavailable` and `rate-limited`
are also `transient`, and both mean the caller should wait; the difference is that
`rate-limited` says the caller caused it.

There is no timeout in the set. No operation waits on a participant's software, so
no operation can time out. What times out is a subscription that never starts
receiving media, and that is an event carrying its own reason.

## What this contract does not decide

The transport and the framing this contract is carried over, which is issue #47.
Nothing above depends on a particular one, and if something here can only be
expressed over one framing, this document has been written wrongly.

The delivery semantics of the event stream beyond the three rules above, which is
issue #48.

Versioning and how a change to this document is announced and retired, which is
issue #49. `unsupported-version` is in the set above so the shape exists from the
start, and what a version is remains that issue's to decide.

The rate limits and size limits behind `rate-limited` and `too-large`, which is
issue #50.

The powers a credential carries, which is issue #46. This document names the
`unauthorised` reason and the operations it can apply to, and holds no list of
powers.

Which layer a subscriber gets when the target is left to this project, which is
issue #36.

## What has confirmed any of this

Nothing. There is no implementation and no consumer:

    git ls-files internal/ | grep -v README

This document is a specification, and issue #51 is the conformance suite that
turns it into something a second implementation can be judged against. Until that
exists, a claim that an implementation follows this contract is a claim.
