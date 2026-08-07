# The seam between this project and a service built on it

This project is a media plane. A service that has users sits above it and keeps
everything that has a social meaning.

Recorded for issue #5. This record is the one every later issue that touches the
API is checked against.

## What this project owns

Terminating media connections. Rooms, as containers for media and nothing else.
Tracks, subscriptions, and the policy that decides which stream a subscriber
receives. Admission of a participant against a credential that was handed to it.
Capacity, resource behaviour, and the numbers that describe them. Deployment,
configuration, upgrade and observation of the service itself. An API narrow enough
that a second implementation of it is conceivable.

## What this project refuses, and why each one

User accounts, registration and password handling. These need a durable record of
a person, and a service that already has one is the only place that can hold it
without this project acquiring users of its own.

Channels, servers, and any structure with a social meaning. A room here is a
lifetime and a set of tracks. It carries no idea of membership over time, so a
structure built out of rooms would be a structure built out of the wrong
primitive.

Presence, participant lists shown before joining, and typing indicators. Each is
a projection over a service's own state rather than over this one's. This project
knows who is connected right now and nothing about who is expected, invited or
absent, so any list it produced would be the wrong list.

Chat, in every form. It is the most requested instance of the entry below.

Application data between participants, of any kind. The connection this project
terminates can carry a data stream beside audio and video, and this one does not
carry one. A service that needs a message path between its own users holds the
identity, membership and policy that every question about such a path turns on,
and it already has a connection to those users to carry it over. The reasoning is
set out in full in the application data record, because the entry is the one most
likely to be argued.

Moderation policy. The mechanical ability to stop a track belongs here, and issue
#38 builds it without deciding who may use it. Who may be silenced is a question
about people and rules, and this project holds neither.

Bots and application protocols. They are the vocabulary of the service above, and
carrying them would mean this project holding state it cannot name.

Clients and user interfaces. A client is where identity, policy and presentation
meet, and none of the three is here.

Identity federation. It is identity, which is refused above, with a second
operator's trust decisions attached.

## The direction of the arrow

A service on top holds identity and policy, mints or requests a credential, tells
this project to open a room, and receives an event stream.

This project never calls back into a service's own vocabulary and never asks who
a participant is. A participant identifier that arrives in a credential is opaque
here, as the admission record sets out.

That direction is the only real test of whether a seam is a seam, because it is
what keeps the API implementable by something that is not this project.

## The known consumers

The community service at github.com/iderex/stammtisch and the large-room
conferencing project at github.com/iderex/hoersaal both need a media plane, and
this project is one.

Each still builds for itself everything on the refuse list above. For a community
service that means accounts, the channel or server structure its users navigate,
presence, chat, and the moderation rules that decide who may be stopped. For a
large-room conferencing project it means the same accounts and identity, the
scheduling and invitation structure a room is created from, the participant list
before anyone joins, and the policy that says which participant is allowed to
speak. Both mint or request the credential this project verifies, and both consume
the event stream rather than reaching into this project's state.

Neither is a dependency of this repository, and nothing here is built to fit
either one in particular. If an interface is only implementable by one of them,
the seam has been drawn wrong.

## What moves an entry from refuse to owns

Three things have to be true at once, and any request that cannot show all three
is answered by this record rather than reconsidered.

The thing cannot be built by the service above out of what this project already
gives it. If a consumer can construct it from the event stream and its own state,
then it is a projection over the consumer's state and belongs there.

Holding it here does not require this project to learn a vocabulary it refuses.
Anything that needs a person, an account, a membership that outlives a room, or a
rule about who may do what fails this immediately.

The resource cost is stated as a number before the work starts, and it fits the
budget in issue #10. An entry that moves in without a measured cost moves the
budget instead, and the budget is the thing being sold.

The application data record is the first entry to be tested against this, and it
does not move.
