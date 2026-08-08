# What this software processes, for the operator who has to answer for it

Somebody running this for a community has obligations this project cannot
discharge for them. What it can do is describe its own behaviour accurately
enough that the operator, or somebody advising them, can reason from it. That is
what this document is.

Recorded for issue #102.

## This is a description, not advice and not a claim

Nothing here is legal advice, and nothing here is a compliance claim. It says what
the software does with data. Whether that is lawful in a particular deployment
depends on where the operator is, who the participants are and what the service
above this one is doing, and none of those is knowable from here.

It is also a description of designed behaviour rather than of observed behaviour.
There is no running deployment to observe: the tree at this commit builds a
program that reports its own version. Every category below names the issue that
builds the thing it describes, so a reader can check whether that part exists yet
rather than assuming it does.

## What is processed, why, and for how long

Media, meaning the audio and video packets participants send. Processed because
forwarding them is what this service is for. Held only as long as forwarding
requires, which is a buffer measured in fractions of a second, and written to no
disk. It is not recorded, and there is no setting that records it. Whether this
project ever writes media to disk is entry 2 of issue #1, which is open and is not
answered here; if it is ever answered in a way that adds recording, this paragraph
is what has to change first.

The content of a conversation is therefore never stored. It is also not readable
from the outside while it is in transit, because the media path is encrypted
between each participant and this service. It is readable by this service, which
is what a forwarding unit is; whether that changes is entry 5 of issue #1 and is
likewise open.

Network addresses, meaning the addresses participants connect from and the
candidate addresses their software offers while establishing a connection.
Processed because a packet cannot be delivered without one. This is the most
identifying thing this service handles, and it is unavoidable rather than a
choice. Held for the lifetime of the session and dropped when the session ends.
Whether any of it is written to a log, and for how long that log lives, is decided
by issues #82 and #84 and is not built yet.

Opaque participant identifiers, meaning the label the service above this one
attaches to a participant. Processed so that events can be correlated and a
participant can be addressed. Held for the lifetime of the session, and in the
admission audit trail for as long as the operator keeps that trail. It is not
resolved, not parsed, and not assumed to mean anything here.

Room identifiers, meaning the name this project gave the service above when a room
was opened. Processed as the container everything else hangs on. Held for the
lifetime of the room.

Credentials, meaning the token the service above hands over for a join. Processed
by verifying its signature and its validity window. Held for the moment of the
join and not afterwards. What it is permitted to carry is a closed question: the
admission record forbids anything about a human being, and verification refuses a
credential carrying any claim outside the set that record permits, naming the claim
it refused. What that check cannot do is judge a permitted claim's contents. An
account identifier placed in the participant field is the same bytes as an opaque
label, so for that case the prohibition is still a rule a person enforces by
reading a change, and an operator should read this paragraph as weaker than the
sentence above sounds.

Admission decisions, meaning a record that a join was accepted or refused, with
the reason. Processed so that an operator can answer what happened. Held for a
period the operator sets. Issue #84 builds it and its own rule is that it holds
admission decisions and nothing more.

Operational measurements, meaning counts, durations and resource figures.
Processed so an operator can see whether the service is healthy. They carry no
conversation content, which is the rule in issue #82, and no address or identifier
that would attach a figure to a participant.

Crash output, meaning what the program prints when it fails. Held on the
operator's host. It goes nowhere: there is no crash reporting and nothing is sent
anywhere, which the federation record fixes and issue #103 is meant to prove.

## What is not processed

There is no account, no registration, no password and no profile. This service
has no users of its own.

There is no name, display name, email address, or account identifier from the
service above. The admission record forbids a credential from carrying one.

There is no chat and no application data between participants, in any form. The
application data record argues that at length; the short version is that a
connection of this kind can carry a message stream and this one does not.

There is no presence, no participant list for people who have not joined, and no
history of who was in a room. This service knows who is connected right now and
nothing about who was, was expected, or was invited.

There is no usage telemetry at any level, and no setting that turns some on.

## Deletion, and why an identifier cannot be resolved here

Most of what is listed above is transient. When a session ends, the addresses and
the session state are gone, because they were only ever held to carry packets.
When a room closes, the room is gone. There is nothing to delete afterwards
because nothing was kept.

What outlives a session is the admission audit trail, and whatever the operator
has configured logs to keep. Those are where a deletion request has anything to
act on.

The part that needs explaining rather than assuming is that this service cannot
find a person's records on its own. It holds an opaque identifier issued by a
different system, and by design it does not resolve it, does not parse it and
holds no mapping from it to anybody. Asked to delete the data of a named person,
this service has no way to work out which identifier that is.

That is a property worth having rather than a defect. It means a compromise of
this service discloses no identity, and it means the operator's identity data does
not accumulate a second copy here.

It also means the deletion route runs through the service above. That service
holds identity, so it can turn a person into the identifier this one knows, and
the deletion is then performed here against that identifier. An operator planning
for such a request should know that the route has two steps and that the first one
is not in this repository.

## Who is responsible for what

This project is responsible for what the software does: processing only what is
listed above, keeping conversation content out of logs and metrics, opening no
outbound connection other than the one named exception, and describing all of it
accurately here.

The operator is responsible for the lawfulness of running the service in their
situation. For informing participants about what happens to their data, because
this project never speaks to a participant. For the host, its security, its
backups and who can reach it. For how long they keep the audit trail and the logs.
And for the certificate exception in the federation record, which discloses the
hostname they chose to a public issuing service and to public transparency logs
where anyone can read it, a hostname that for a self-hosted deployment frequently
names the community or the person running it.

The service above this one is responsible for identity in every form: accounts,
what a credential is issued for, who is allowed into which room, and what it does
with what it learns. The seam record is where that division is drawn, and this
document does not restate it.

## What an operator should not conclude from this document

That the software has been audited. It has not.

That the behaviour described has been observed. Most of it is not built yet, and
the paragraphs above name the issue for each part so that can be checked rather
than taken on trust.

That any of this constitutes a lawful basis, a data protection impact assessment,
or a record of processing activities. Those are the operator's documents. This one
exists so they can be written from something accurate.
