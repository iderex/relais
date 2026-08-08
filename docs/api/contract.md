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

Checked against [the media formats record](../decisions/media-formats.md): this
contract carries the room's format set outward, carries the format of each track,
and names the refusal a format outside a room's set produces. It asks this project
to convert nothing, and nothing below offers a subscriber a stream in a format its
publisher did not send. That record's list of formats is not restated here, because
the list grows by a change to that record and a copy of it in this document would
drift from the day it was written.

## The resources

Four, and every one of them exists only while the media it describes exists. None
of them is a record of anything that happened.

A room. The container everything else hangs on. Its identifier is minted by the
caller when the room is opened, so the caller can address the room before this
project has answered. It exists from the moment it is opened until it is closed,
and closing it is the only way it disappears. It holds no memory of who was in it,
and a second room opened under the same identifier after the first was closed is a
different room that happens to share a name. It carries a format set, described
below, which is part of its description from the moment it is opened and is
readable before anybody has joined.

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
whether it is audio or video, the format it is in, and the layers currently
available.

The format is in the description because the kind does not carry it and nothing
else in the description does. Two video tracks in one room can be in two different
formats, both inside the room's set, and the layers each one reports are structured
by its format rather than by a shape common to all of them, so a consumer reading a
track's layers without knowing its format is reading a structure whose meaning it
has not fixed. It is also what a consumer needs to say why a particular track is
not playing, which is the diagnostic issue #62 owns, and it is what shows that a
publisher stayed inside the set the room was opened with.

A subscription. One subscriber receiving one track at one target. Its identifier is
minted by the caller for the same reason a room's is. It exists from the moment it
is created until it is ended, until the track ends, or until either side leaves.
Whether media is flowing under it is a separate question from whether it exists,
and the events say which.

A layer is not a resource. It is part of a track's description, named by this
project, and a caller addresses it only inside a subscription target.

### The room's format set

The set of media formats a room carries. A publisher may only publish inside it, a
participant that cannot handle it is refused at join rather than left in a room
that stays quiet, and nothing is converted to bring anything into it, which is the
media formats record's position and not this document's to soften.

The caller supplies the set when it opens the room, and this decides the question
issue #141 left open rather than assuming an answer. Three reasons, in the order
they carry weight.

A room exists before any participant does. A set derived from the first join would
mean the room has no set until somebody arrives, so the earliest joiner would fix
what every later one is judged against, by accident and by being first. The caller
that opened the room would have no way to say what it wanted and no way to read
what it got until the outcome was already decided.

The caller is the only party that knows what its participants can do. It holds the
identity and the client population; this project holds neither and, under the seam
record, may not learn either. Deriving the set here would mean this project
reasoning about the software a person is running, which is a client concern the
seam refuses.

A consuming service has to be able to tell somebody what a room needs before they
join. That is a read on the room, answered from the moment it is opened, and it is
only possible if the set was fixed there.

The set is a non-empty subset of the formats in the media formats record. A set
naming a format that is not in that record is refused when the room is opened, with
`format-unsupported`, because a room whose set this project cannot carry is a room
in which the first publisher would be refused for a reason the caller could have
been told at the start.

The set does not change over a room's lifetime. There is no operation that widens
or narrows it, and this is a decision rather than an omission: narrowing it would
strand publishers already inside the room, and widening it would admit a publisher
whose format the participants admitted earlier were never judged against. A caller
that wants a different set closes the room and opens another, which the resource
description above already says is a different room.

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

Creates the room, with the format set the caller supplies. Idempotent by room
identifier: a repeat naming a room that is already open and was opened by the same
caller succeeds and creates nothing. A repeat naming a room identifier that is open
under different parameters fails with `already-exists`, because silently returning
success would tell the caller its parameters took effect when they did not. The
format set is one of those parameters, so a repeat naming the same room with a
different set fails the same way rather than replacing the set a live room is
running under.

A set naming a format this project does not carry fails with `format-unsupported`,
and the failure names the formats that were not carried. A retry after a failure of
class `transient` is safe and is the intended response to it.

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

A retry is safe. A retry after `credential-invalid` is not useful, because a
signature does not start holding by being sent twice. A retry after a long pause
fails with `clock-skew` rather than with the reason the first attempt gave,
because the short validity window in the admission record has closed by then, and
what that reason asks of a caller is set out with the error set below.

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

A session attached. A session detached, with the reason. `format-refused` is one of
the reasons that arrives here rather than in a reply, because the offer it refuses
is made by the participant's software and not by the caller, and it is how a
consuming service learns at join that somebody cannot take part instead of learning
it from a room that never carries anything.

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
| `credential-invalid` | `caller` | the credential's signature did not hold |
| `unauthorised` | `caller` | the credential is good and does not carry the power this operation needs |
| `too-large` | `caller` | the request exceeded a size limit |
| `at-capacity` | `transient` | well formed, and this host cannot carry it |
| `unavailable` | `transient` | this project cannot answer right now, and the answer is absent rather than negative |
| `rate-limited` | `transient` | the caller is sending faster than the limit allows |
| `closing` | `refused` | the room, or this project, is shutting down |
| `format-unsupported` | `caller` | a format named in the room's set is not one this project carries |
| `refused-by-media` | `refused` | the request contradicts what the media side observes, a named layer the track does not report being the ordinary instance |
| `format-refused` | `refused` | the offer at negotiation is outside the room's format set, which the refusal names |
| `clock-skew` | `refused` | the credential's validity window did not hold against this host's clock, in either direction |
| `unsupported-version` | `refused` | the caller asked for a version this project does not serve |

Five notes on the set, each of which is a decision rather than a detail.

The two format reasons are separate from each other and from `refused-by-media`.
`format-unsupported` answers a caller that named a format this project does not
carry when it opened a room, and the repair is in the request, which is what makes
it class `caller`. `format-refused` answers an offer at negotiation that fell
outside a room's set, and the room's set was well formed and accepted. Collapsing
the two would leave a caller unable to tell a set it should not have asked for from
a participant that turned up with the wrong software.

Neither is `refused-by-media`, which is about what the media side observes, a
subscription naming a layer the track does not report being its instance. A format
outside a room's set contradicts a set the caller itself supplied rather than
anything this project observed, and it is refused before a packet arrives.

`format-refused` is class `refused` rather than `caller`, which is the reading that
takes the most care. Something does have to change, but it is not in the caller's
request and no version of that request fixes it: the media formats record rules out
converting, so this project will not carry the offer under any wording. The repair
is upstream of the caller, in what the participant's software offers, and the
refusal names the accepted set so the caller can pass that on. A caller that reads
the class and stops is doing the right thing; a caller that reads `caller` and
retries would be sending the same offer forever.

`clock-skew` is the reason for every validity window failure, in both directions,
and `credential-invalid` covers a signature that did not hold and nothing about
time. The two rows described one condition between them before this was settled,
and a class is a field a caller routes on, so two reasons answering to one
condition made the routing a coin toss.

What a verifier holds when a window fails is the credential's two times, its own
clock, and the tolerance the admission record makes part of the join contract. It
does not hold the issuing service's clock. So it can decide that the window did
not hold once the tolerance is applied, and it cannot decide which of the two
sides moved. A credential that sat too long and a host whose clock runs fast
produce the same observation; a credential presented before its window opens and
a host whose clock runs slow produce the other same observation. Nothing in a
credential separates them, which is why one reason covers both directions and why
that reason names what was checked rather than what is suspected.

The class follows from that. `refused` tells a caller to stop rather than to
re-mint, and re-minting is what a service must not do against a host whose clock
has drifted, because every credential it issues will fail the same way and the
loop hides the one fact the operator needs. The cost is real and is paid rather
than hidden: a service whose credential simply aged out is told to stop, when
minting a fresh one for the next attempt would have worked, and it has to reach
that conclusion itself instead of reading it off the class. It is paid because the
other mistake is the one nobody can see from outside the host, and because a
host's own skew is meant to be caught at startup by issue #81 rather than by a
caller inspecting error classes.

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

Nothing. No implementation of this contract exists, and no consumer of it exists.

The command that used to stand under that sentence listed everything under
`internal/`, and it stopped agreeing with the sentence the moment a package landed
that this document does not describe. Two have. One names the identifiers these
operations carry; the other decides whether a join credential is genuine, current
and applicable. Neither is a room, a track, a negotiation or an operation on this
contract, so both are real code and neither is evidence for or against anything
written above. A reader who ran that command saw output where the sentence told
them to expect none, and the honest reading of that is that the evidence had
stopped supporting the claim.

What an implementation of this contract needs first is code in the package that
carries it to the wire, and that package holds nothing but the note saying what
belongs in it:

    git ls-files internal/api
    internal/api/README.md

The bound on that command is that it is a necessary condition rather than the whole
of one. `internal/api` holding code would not by itself mean this contract was
implemented. This contract being implemented would mean `internal/api` holds code,
and it does not. The output moves when that changes and at no other time, which is
the property the previous command did not have.

The consumer half is a claim rather than a result, and it is written as one. A
consumer of this contract is a service built on top of this project, which lives
outside this repository, so no command run here could show its absence.

This document is a specification, and issue #51 is the conformance suite that
turns it into something a second implementation can be judged against. Until that
exists, a claim that an implementation follows this contract is a claim.
