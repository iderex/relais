# How the one command is built

The deployment contract says what the one command promises and what it may assume.
This says how it is built, and it comes before the parts so that they are pieces
of one design rather than a collection that ends up in the same directory.

Recorded for issue #54. Where this and the deployment contract disagree, the
contract is the one that decides.

## The mechanism

One static binary that configures itself, published also as a container image, and
the image is that binary in a container rather than a second thing.

The operator's command is the binary started with the name the service answers on,
or the composition that starts the image with the same value. Both forms run the
same program deriving the same values, so a defect in one is a defect in both and
there is no configuration path that exists only in the packaged form.

## Why not the alternatives

A script that fetches things is the usual answer and it is refused. It needs a
shell, a fetcher and a trust decision at the moment of installation, which is the
moment an operator is least equipped to make one. It fails differently on every
host, its failures are reported by whichever tool happened to fail rather than by
this project, and the widespread form of it asks an operator to pipe a network
response into a shell, which this repository is not going to teach.

A container composition alone is refused because it makes a container runtime
mandatory. The contract allows either a runtime or one static binary, and the
operator running the smallest always-on machine is frequently the one who has no
runtime. It is also the wrong direction of derivation: an image is a binary in a
container, so the binary is the artefact and the image follows from it. Building
the composition first would mean the binary was whatever fell out of the image.

A binary that configures itself is what is left, and its cost is stated rather
than passed over. Everything the deployment needs to do has to be inside the
program: obtaining a certificate, installing itself under the host's service
manager, checking its own assumptions, and removing itself. That is more surface
in one artefact than a script would have, and it is surface this project tests
rather than surface the operator's shell provides.

## State

One directory, and nothing outside it.

What lives there: the certificate and the account key used to obtain it, the key
material the operator supplied for verifying credentials, and the identifier this
host keeps across restarts. That is the whole list.

What does not: any record of a room, a participant, a session, or anything that
was said. Rooms are not durable. A restart loses every live session and nothing
else, which is the property that keeps backup and restore small enough to describe
in a sentence, and it is why issue #64 can say what is allowed to be lost without
a long argument.

Where the directory is, is one of the four configurable categories, because only
the operator knows their storage. It is derived to a sensible default and the
default is printed at startup rather than assumed, which is issue #58.

## Privileges

The service runs as an unprivileged user and holds no capability it does not need.
It never runs as root, and nothing in the deployment asks an operator to make it.

The media port has no privileged default, which the network shape record fixes, so
that half needs nothing.

The service port defaults to 443 and that is below the line where an ordinary
process may bind, so this is where the design has a real cost and it is written
down rather than smoothed over. In the container form the runtime publishes the
low port and the process inside binds an unprivileged one, so no elevated step
exists. In the binary form there is exactly one elevated action, performed once at
installation: granting the binary permission to bind a low port, or installing the
service unit that does the same thing. It is one action, it is named, it happens
at installation and never at runtime, and the running service holds nothing beyond
that one permission.

An operator who will not take that step can put the service on a high port and
redirect to it themselves. The contract refuses to assume they have done so, which
is why this is an alternative rather than the design.

## Supervision

The host's own service manager in the binary form, and the container runtime's
restart policy in the image form. Neither is a supervisor this project writes.

The policy is restart on failure with a backoff, and the reason it is safe to
state so briefly is the state section above: a restart costs the live sessions and
nothing else. There is no recovery procedure, no replay and no consistency
question, because there is nothing durable to be inconsistent with.

What a restart must not do is loop silently. A service that has failed to start
for the same reason ten times running is a service whose operator needs to be
told, and the diagnostic that says which layer failed is issue #62.

## Removal

One command, and it says what it did.

It stops the service, removes the service unit or the container, and leaves the
state directory in place unless it is explicitly told to remove that too. Then it
prints the path it left and what is in it.

Leaving the state is deliberate. An uninstall that silently deletes an operator's
certificate material and credential keys is worse than one that leaves them, and
an operator evaluating this software will remove it at least once. How easily they
can remove it decides whether they try it at all, so this is part of the product
rather than an afterthought.

## The unhappy paths

These are the whole difficulty, so each has a designed response written before it
is built. Every one of them is a response the operator can act on rather than a
stack trace.

The name does not resolve yet. Checked before the command commits to anything.
It reports the name it was given and what it resolved to, which is nothing, and
stops there. It does not go on to obtain a certificate, because that attempt would
fail and would report a certificate problem to an operator whose problem is a
missing record.

The port is not forwarded. This one cannot be proven from the host, and the
network shape record says why: a host with a public address and a router nobody
configured looks identical from the inside to one that is set up correctly.
What the command does is report the addresses the host holds, say when one of them
is a shared address that cannot be forwarded, and state plainly that reachability
was not verified. It does not claim success it cannot check. The verification
arrives when a participant tries, through the diagnostic in issue #62.

The certificate cannot be obtained. The service does not start serving without
one, and it does not substitute a self-signed certificate and carry on as though
nothing happened. It reports which step of the issuance failed, keeps retrying
with a backoff so that an outage at the issuing service resolves without the
operator doing anything, and names the path where an operator may put their own
certificate instead. Issue #57 builds it.

Another service already holds the port. Detected at the moment of binding, before
anything else is attempted. It reports the port and, where the host will tell it,
what is holding it. It does not pick a different port. A service listening
somewhere the operator did not open is a service that starts and then fails
silently, which is the failure this whole contract exists to prevent.

## What checks this rather than asserts it

Issue #55 runs the one command against a clean host and asserts that a participant
is accepted and media flows afterwards. Issue #56 turns the contract's assumption
list into checks that run before the command does work. Issue #59 proves the
default configuration is complete. Issue #62 is the diagnostic named twice above.

None of them has run. Everything in this record is a design and no execution of
anything has confirmed it.
