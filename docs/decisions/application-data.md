# Application data between participants

This project carries no application data between participants.

Recorded for issue #119. It is an entry on the seam record's refuse list and is
argued here at length because the silence around it was the problem.

## Why the silence was the problem

A connection of the kind this project terminates can carry an application data
stream beside audio and video. Nothing on this board said whether this one does.
The domain model in issue #31 names room, participant, track, layer and
subscription and no data primitive. The seam record refuses chat and every other
application protocol. The operator API in issue #43 describes an event stream that
runs between a consuming service and this one rather than between participants.

All three are consistent with a decision that nobody had written down, which means
the first person to want a message path would have found nothing refusing them and
a domain model that merely had not got around to it yet.

## The position and its reasons

A service on top that needs a message path between its own users already holds
identity, membership and policy. That is where a message path belongs, because
every question a message path raises is a question about people. Who may send to
whom. What happens to a message for somebody who is not connected. Who may delete
one.

That service also already has a connection to those users, which is how it told
them about the room in the first place. So the path exists and putting a second
one here adds a route without removing a requirement.

Carrying it here would mean this project ordering and buffering application state
it has no vocabulary for. The request that follows is always history, because a
message path without history is a message path people complain about. History is
storage, and storage of what participants said to each other is the thing the
sovereignty position works hardest to avoid.

## The argument against this position

There is one class of message where the case is real, and it is worth stating
before it is answered rather than after somebody raises it.

Data the media plane itself produces or consumes is not application data. A
subscriber saying what it wants at a moment when its other connection is the slow
one is the clearest example: the information is about the media, the timing
matters, and routing it over a path that is congested for a different reason
defeats it.

That case is answered by the next section rather than by opening a general
channel, because a general channel opened for a specific need is how every one of
these ends up carrying everything else.

## Where media-plane control messages ride

The signalling path, which is issue #47, is the default and carries everything
that is not on the critical timing path of the media itself.

A stream on the media connection carries media-plane control messages only where
the signalling path is demonstrably the wrong one for that message, and what may
travel there is a closed list rather than a shape. Today that list holds
subscriber-side requests about what the subscriber is receiving: the layer it
wants, a request for a keyframe, and its own view of what it is getting. Each
exists because it is about the media, it is time-sensitive, and it is meaningless
outside this project's own vocabulary.

Nothing is added to that list without an entry in this record saying what it is
and why the signalling path cannot carry it. An addition that cannot name that
reason is refused. The list being closed is the whole mechanism, and it is a rule
that a reviewer holds rather than one anything refuses today.

## What would reverse this

The same three conditions the seam record applies to every entry on its refuse
list, all three at once.

A named use that the consuming service's own connection cannot serve, with the
reason it cannot. Not a use that would be more convenient here, and not a use that
is slower there. A reason it cannot.

An answer to what this project would then owe about ordering, delivery guarantees
and size limits, written before the work starts rather than discovered by the
first consumer that depends on the answer.

A stated resource cost that fits the budget in issue #10, because buffering
application state for every participant is memory that the per-stream numbers do
not currently account for.

## The board at the commit this record lands on

The search this record owes returned a rate limit error rather than a result, at
this commit, in the form it was written in:

    gh issue list --repo iderex/relais --state all --search '"data channel" in:body,title' --json number --jq 'length'
    GraphQL: API rate limit already exceeded for user ID 30603423

That is recorded rather than worked around silently. The same query run directly
against the same budget did return, and it is the substitute this record stands
on:

    gh api graphql -f query='query { search(query: "repo:iderex/relais \"data channel\" in:body,title", type: ISSUE, first: 20) { issueCount nodes { ... on Issue { number title state } } } }'
    count=1
      #119 [OPEN] Decide whether the media plane carries application data between participants

The count is one rather than zero, and the one is this record's own issue, which
matches because it quotes the phrase it is searching for. So no issue on this
board asks this project to carry application data, and the only issue that
mentions the subject is the one deciding against it.

The count will move as soon as this sentence lands, for the same reason. A later
reader who re-runs it should read the matches rather than the number.
