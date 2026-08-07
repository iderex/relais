# The pool, and what it does when its view is wrong

A pool is a distributed system, and this record does not pretend otherwise. Split
views, stale state and a host that is unreachable but alive are the normal
conditions rather than the exceptional ones, so each of them has a stated
behaviour here, before any of the parts are built.

Recorded for issue #72. It turns the line the growth record drew into a design.
Where the two disagree, the growth record is the one that decides.

## What a host is

A running instance of this service, reachable at an address, holding zero or more
rooms.

It has an identifier it chooses once and keeps, so that a host which restarts is
recognisable as the same host rather than arriving as a new one whose predecessor
never left.

It has a state, and there are four. Joining, which is a host that has announced
itself and has not yet reported capacity. Ready, which is a host that will accept
a room. Draining, which is a host that will accept no new room and is still
serving what it has. Gone, which is a host that has reported its own departure or
has been marked so by an operator.

There is no fifth state for a host that has stopped answering. That is a property
of the view rather than of the host, and it is described below, because the two
are different things and collapsing them is how a live host gets treated as a
dead one.

## How a host joins and leaves

It joins by announcing itself to the pool with its identifier, its address and
its first capacity report. Nothing else admits it, and in particular the view
does not discover hosts, because discovery means a mechanism that can be wrong in
the direction of admitting a host nobody meant to add.

It leaves by draining. Draining is a state the host enters and reports; the pool
stops placing on it; the rooms it holds end when their sessions end; and the host
reports itself gone when the last one has. Issue #76 builds it and issue #78
proves the loop with no media and no second machine.

A host can also stop, without draining, by failing. What happens then is that its
rooms are lost and the participants in them are disconnected. Nothing in this
design moves a live room from one host to another, because one room spanning
several hosts is out of scope and moving a room is that problem wearing a
different name.

## Where the authoritative view lives

Nowhere, in the sense the question usually means.

The pool's view is derived state. It is what the hosts have reported, held by
whichever host currently holds the pool role, and it is rebuilt from those reports
rather than being a record that outlives them. A view that has just been rebuilt
from scratch and one that has been running for a week hold the same thing.

What is authoritative is narrower and lives in one place each. Whether a room
exists, and who is in it, is answered by the host holding that room and by nothing
else. What a host's capacity is, is answered by that host. The view answers one
question only: which hosts exist and what state each is in.

That split is the whole design. It is what keeps the pool from becoming a database
this project would then have to make durable, and it is what makes the failure
behaviours below possible to state rather than merely to hope for.

## What happens when the view is unreachable

Nothing that is already running stops.

Media keeps flowing, because forwarding needs no pool. A participant joining a
room that already exists is admitted, because that is answered by the host holding
the room. A host that is already serving keeps serving.

What stops is placement. No new room is placed while no view is reachable, and a
request to open one is refused with the reason, rather than being placed somewhere
optimistically and reconciled later. Optimistic placement is how two hosts end up
believing they hold the same room, and this design would rather refuse.

That is fail-static rather than fail-closed, and the difference is deliberate:
what is running is not interrupted by the failure of a thing it does not need.

## Split view

Two views can disagree about which hosts exist. They cannot disagree about whether
a room exists, because neither of them is asked.

So a split view costs placement quality rather than correctness. A room may be
placed on a host another view believed was draining, and the outcome is a room on
a draining host, which serves it until it ends. It cannot produce two hosts
holding one room, because a room is created by the host that holds it and a second
creation under the same identifier is refused there.

Only one host holds the pool role at a time, and losing that role is not a failure
that needs a resolution step. The view is derived, so a new holder rebuilds it
from the reports.

## Stale state

Every capacity report carries the moment the reporting host produced it, by that
host's own clock. The view carries that moment rather than the moment it received
the report.

An entry whose newest report is older than a stated multiple of the reporting
interval is marked stale. A stale entry is not placed on, and it is not deleted.
Deleting it is the tempting repair and it is refused here: a host that has fallen
out of the view is a host nobody drains, nobody retires, and nobody notices is
still running and still holding rooms.

The multiple is a number that has to be chosen against measurement rather than
taste, and it is not chosen here. What is fixed is that it is a multiple of the
reporting interval rather than an absolute time, so the two cannot drift apart.

## Unreachable but alive

The most common of the three, and the one where the wrong response is worst.

A host that stops answering the view is marked unreachable. The view stops placing
on it. That is all it does. It does not declare the host's rooms ended, does not
tell anybody the participants have left, and does not permit a room of the same
name to be created elsewhere.

The reason is that unreachable from the view and unreachable from the participants
are different facts, and the view is not in a position to tell them apart. A host
whose link to the pool has failed may be forwarding media perfectly to everybody
in the room. Ending those sessions on the strength of the view's own ignorance
would be this project breaking a working call in order to keep its bookkeeping
tidy.

Nothing kills a host from the view. Retirement is a drain, which the host itself
performs, or an operator action against the host. Both act on the host rather than
on the record of it.

## What a consuming service uses to place rooms itself

Four things, all of them on the operator API rather than internal state, because a
replaceable default has to be replaceable from outside the process.

The capacity signal, per host, which issue #73 derives from measurement rather
than from configuration.

The pool view, which is the host list with the state and the freshness of each,
including the entries marked stale and unreachable. A consumer that cannot see
those is a consumer that will place on them.

The placement operation, which opens a room on a named host. Issue #75 builds it.

The drain operation, so that a service which manages its own capacity can retire a
host as well as fill one.

Issue #74 builds the pool and issue #43 is where these appear in the API contract.

## The default policy, and turning it off

The default exists for the operator who wants none of this. It places a room on
the ready host with the most room to spare, and it refuses to place when no ready
host has enough, which is issue #77.

Growing the pool needs a machine, and this project does not provision machines.
Where the operator has configured a driver that can create one, the default policy
asks it; where they have not, which is the default, the policy has nothing to grow
with and refusal is the whole of its behaviour at the ceiling. That is a real
limit and it is written here rather than discovered by an operator whose room was
refused.

Turning it off is not a setting. A room opened with an explicit host is placed
there, and a consuming service that places every room explicitly has replaced the
default without switching anything. That is deliberate: a policy that has to be
disabled before it can be replaced is a policy with a second failure mode, which
is being disabled while nothing has replaced it.

## What is out of scope at first release

Restated here so that no issue in this milestone quietly assumes otherwise.

One room spanning several hosts. Geographic distribution. Provisioning of machines
by this project, which is a driver interface at most.
