# The media plane port

Everything above this interface orchestrates and holds no packet. Everything
below it holds packets and knows nothing about a participant beyond an opaque
identifier.

Recorded for issue #4. It is written before an implementation exists, because a
port written afterwards is a description of that implementation rather than a
constraint on it. The first implementation is judged against this record, and
where the two disagree the record is what has to be argued with.

## What sits on each side

Above the port: room lifetime, admission against a credential, the event stream a
consuming service reads, the API surface, the pool view and placement. None of it
touches a packet.

Below the port: connectivity establishment, the handshake, encryption, congestion
control, retransmission, and the forwarding loop that decides which bytes a
subscriber receives. None of it knows what a participant is called, who invited
them, or what the room is for.

The forwarding core record fixes what is taken from the transport library and
what this project owns. This record fixes the seam between that work and
everything above it.

## The three properties the interface owes

Each of these is a constraint that rejects otherwise reasonable designs, so each
is stated as a property rather than a preference.

It is expressible by an implementation that carries no media at all. The fake in
issue #30 is the thing the whole orchestration suite runs against, so an
operation that can only be given meaning by a real socket has been specified
wrongly. Reachability, encryption state and packet loss are reported through the
port rather than assumed by the caller, because a fake has to be able to report
them too.

No type from the transport library crosses the boundary. Not a peer connection,
not a track, not a session description, not a candidate, not an error value
declared by that library. A port that leaks one of those is decoration: the
orchestration layer would then compile against the library, and replacing the
implementation would mean changing every caller. There are no exceptions today.
An exception added later carries its reason in this record, and adding one is a
change to this file rather than a decision made in a function signature.

It is asynchronous everywhere the real thing is asynchronous. An operation that
depends on a remote peer answering does not return the result of that answer. It
returns once the intent is recorded, and the outcome arrives as an event. This
costs the caller a state machine it would not need against a synchronous fake,
and that is the point: a fake that answers instantly must not be able to hide a
race that the real implementation has.

## The identifiers that cross

Five, all of them opaque strings minted on one side and never parsed on the
other.

A room identifier, minted above the port when a room is opened and carried
downward from then on.

A participant identifier, minted above the port. The admission record already
fixes that it is opaque here and is not resolved, parsed, or assumed to mean
anything outside the service that issued it.

A track identifier, minted below the port when a publisher's track appears, and
carried upward. It is the only identifier that originates on the media side,
because it names something only the media side can observe.

A layer identifier, minted below the port as part of the track's reported
structure. It names one of the alternatives a subscriber can be given for a
track, and its meaning is fixed by the media formats record rather than by this
one.

A subscription identifier, minted above the port when a subscription is created,
so that the caller can address it before the media side has confirmed anything.

## The other values that cross

A track description, which is the identifier, which participant published it,
whether it is audio or video, and the list of layers currently available with the
attributes the selection policy acts on. The list of attributes is fixed by the
media formats record and by issue #34, and this record does not restate it.

A subscription target, which is a track identifier and either a named layer or
the instruction to let the allocation policy choose. Issue #36 owns the policy.
The port's job is only to carry which of the two the caller asked for.

A capacity report, described in its own section below.

A reason, which is the closed set of error cases below.

Nothing else. In particular no session description, no candidate, no address, no
statistics structure from the library, and no encryption material. Where the
orchestration layer needs to know something the transport library observed, this
record gains a value of this project's own with a stated meaning, and the
translation happens below the port.

## The operations

Every operation is listed with what it means, whether it may block, and what
happens when it fails. The rule is uniform and stated once: an operation returns
once the intent has been recorded on the media side, never after waiting on a
remote peer, and never after waiting on a timer. Where an outcome depends on a
peer, the outcome is an event.

Open a room. Creates the container the media side will attach sessions to. It may
block only for as long as it takes to record local state, which means it may
allocate memory and take a lock, and it may not wait on a socket. It fails when
the host is at capacity, when the identifier is already in use, or when the media
side is shutting down. On failure nothing is created, and the caller may retry
with the same identifier once the reason has changed.

Close a room. Ends every session in the room, releases everything the room held,
and is the only route by which a room disappears. It may not block on the
sessions ending. It returns once the room is marked closed and refuses every
later operation naming it, and each session end arrives as its own event. It
fails only when the room is not known, and that failure is safe to ignore,
because the state the caller wanted is the state that already holds.

Admit a participant. Records that a participant may attach a session to a room,
against a credential the layer above has already verified. The port receives the
decision rather than the credential: verification happens above, because it is
about a signature and a clock rather than about packets, and the admission record
owns it. It may not block. It fails when the room is not known or is closing,
when the participant is already admitted, or when the room or host is at
capacity. On failure nothing is admitted.

Revoke a participant. Detaches a participant's session, ends every track it
published and every subscription it held, and refuses a later attachment under
the same identifier. It may not block on the detachment completing, and the
completion arrives as an event. Revoking a participant who is not present
succeeds, for the same reason closing an unknown room is not worth an error: the
caller asked for a state that already holds.

Stop and resume a publisher's track. Issue #38 owns what this means to a
consuming service and deliberately does not decide who may ask for it. Here it is
one operation with a direction. It may not block, and the effect arrives as a
track description whose availability has changed.

Subscribe. Creates a subscription from one participant to one track, with a
target. It may not block, and it returns the subscription identifier the caller
minted. It fails when the track is not known, when the subscriber is not
admitted, when the subscriber is already subscribed to that track, or when the
room is at its per-subscriber ceiling. Whether media then flows is not part of
this answer. It arrives as an event.

Retarget a subscription. Changes which layer a live subscription receives,
without tearing it down. It exists as its own operation rather than as an
unsubscribe followed by a subscribe, because those two produce a visible gap and
this must not. It may not block. It fails when the subscription is not known or
when the named layer is not among the ones the track currently reports, and on
failure the subscription keeps the target it had.

Unsubscribe. Ends one subscription. It may not block, it succeeds when the
subscription is not known, and the media side stops sending under it before the
event that reports it ended.

Report capacity. Returns what the host can still carry, as a value rather than a
promise. It may not block and it has no failure case: a media side that cannot
compute a figure reports that the figure is unavailable, which is a value the
caller can act on, rather than an error the caller has to interpret. Issue #73
owns what the signal is derived from and issue #74 owns the pool that reads it.

Drain. Puts the media side into a state where it accepts no new room and no new
admission, and it does not end what is already running. It may not block. There
is no undrain: a host that has been drained is retired, which is issue #76, and a
reversible drain would be a second state with its own failure modes for the
benefit of an operator action nobody has asked for.

Subscribe to the event stream. Returns the stream described below. It may not
block. It fails only when the media side is already shut down.

Shut down. Ends everything and releases the media side's resources. It is the one
operation that is allowed to block, because a caller that has decided to stop
needs to know when it is safe to exit, and the alternative is a process that
exits while a socket is still open. It takes a deadline from the caller and
reports whether it finished within it. What happens when it does not is that the
caller is told, and the decision to exit anyway belongs to the caller.

## The events

The stream carries what the media side observed. It never carries a request, and
nothing above the port answers it.

A session attached, and a session detached with the reason it detached.

A track appeared, with its description.

A track's layers changed, with the new description. This is one event rather than
a layer appearing and a layer disappearing, because a subscriber's allocation is
decided against the whole set and two events would let a policy run against half
of one.

A track ended.

A subscription is receiving media, and a subscription stopped receiving media
with the reason. A subscription changed the layer it is receiving, which is
raised whether the change came from a retarget or from the allocation policy, so
that a consuming service reading the stream sees the same thing in both cases.

Capacity changed.

The media side is failing, with the reason. This is the event that says the
implementation cannot do its job, as distinct from a particular session going
wrong, and it exists so that a startup self-check and a readiness signal have
something to read.

Two properties of the stream itself. Events are ordered per room, and not across
rooms, because a total order over every room is a serialisation point in the
place this project can least afford one. And the stream is lossy in one stated
way: a consumer that falls behind is told that it fell behind and how many events
it missed, rather than being silently trimmed or being allowed to grow the media
side's memory without bound. Issue #48 carries that property outward to a
consuming service, and it may not weaken it.

## The error cases

A closed set. An implementation that needs a reason outside it adds one to this
record first.

Not known, meaning the room, participant, track or subscription named does not
exist here. Retrying does not help until something else changes.

Already exists, meaning the identifier is in use. It is a caller defect rather
than a condition, and it is separate from not known so that a caller does not
have to distinguish them by inspection.

At capacity, meaning the request is well formed and the host cannot carry it.
Retrying is meaningful, and this is the only reason for which that is true
without another operation happening first. The pool acts on it.

Closing, meaning the room or the media side is shutting down. Retrying does not
help, and the caller is expected to stop rather than to look for another route.

Refused by the media side, meaning the request contradicts what the media side
observes. A layer that the track does not report is the ordinary instance. It is
distinct from not known because the thing named exists and the request about it
does not hold.

Unavailable, meaning the media side cannot answer right now and the caller should
treat the answer as absent rather than as negative. Capacity uses it. Nothing
else does today, and an operation that wants it has to say why its absent answer
is not a failure.

There is no timeout in the set, and that is a consequence of the blocking rule
rather than an omission. No operation waits on a peer, so no operation can time
out. What times out is a subscription that never starts receiving media, and that
is an event carrying its own reason.

## What replacing the implementation would cost

The estimate below is what the port is being built to buy, so it is written down
now and checked against reality later rather than being claimed once and never
tested.

Changed: the package implementing the port, and its own tests. Nothing else,
if this record holds.

Unchanged: the package declaring the port, the orchestration layer above it, the
API surface, the admission path, the event stream, the pool, and the fake. The
fake is unchanged by construction, because it implements the same interface and
knows nothing about either implementation.

Also changed, and this is the part an estimate usually forgets: the dependency
lock file, the bill of materials, the third-party notices, and the startup
self-check where it names what the transport layer requires of the host. A second
implementation behind the port is a second dependency graph, and none of those
four artefacts is inside the boundary this record draws.

The package names are not written here because the layout is issue #16 and a
record that restates a layout drifts against it. The number of packages is the
claim, not their names.

This estimate is unverified. It is a design intention, and nothing has tested it,
because there is no implementation to replace. It is checked again when the first
implementation lands, in issue #33, by listing the packages that actually had to
change. If the list is longer than the one above, this record is wrong and is
corrected here rather than explained away in the pull request that found it.

## What this record does not decide

The wire framing between a consuming service and this project, which is issue
#47. This record is about a boundary inside one process.

The powers a credential carries, which is issue #46. The port receives an
admission decision that has already been made.

Which layer a subscriber gets, which is issue #36. The port carries the target
and the outcome, and holds no policy.

What the forwarding core understands about a payload format, which is the media
formats record.
