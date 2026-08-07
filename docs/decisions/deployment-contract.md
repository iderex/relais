# The deployment contract

The deployment story is not a feature of this project. It is the product. A
selective forwarding unit that works and takes a weekend to stand up has not
filled the gap this repository is aimed at.

Recorded for issue #6. It is written as a contract, with a promise, an assumption
list and a never list, before the thing it describes is built, because a
deployment story written afterwards describes what the scripts happened to need.

## The promise

From a host with nothing on it, one command produces a service that accepts a
participant and forwards media.

No file is edited by hand first. No value is chosen by the operator. No second
command is needed to make the service reachable. The operator's total input is the
name the service answers on.

That is the whole promise, and each clause is there because it is a place a
deployment story usually breaks. A configuration file to edit first is a file
whose example is out of date. A value to choose is a value the operator cannot
know. A second command is the one that gets left out of the instructions.

## What the command is allowed to assume

This list is the whole of it. An issue that needs something not on it is changing
this record, not making an exception.

A current 64-bit Linux host.

Either a container runtime, or the ability to run one static binary.

A name in DNS pointing at the host.

Inbound reachability on the ports the network shape record settles, which the
operator opens once.

Nothing else.

In particular it may not assume a reverse proxy, a database, a message broker, a
certificate the operator obtained themselves, a tuned kernel, or a second machine.

## What each assumption costs

Stated because each one excludes somebody, and an excluded operator is a real
person rather than a rounding error.

A current 64-bit Linux host excludes 32-bit hardware, which is where some of the
cheapest always-on machines a self-hoster already owns still sit. It also excludes
the other operating systems entirely at first release. The cost is accepted
because supporting a second kernel means a second set of resource measurements,
and the resource budget is one of the two things this project is offering.

A container runtime or a static binary excludes distribution packages at first
release. An operator who installs everything through their distribution's package
manager has to make an exception for this one, and the operators who most want
that are frequently the ones running the smallest machines. The cost is accepted
because a distribution package is a promise to track that distribution's release
cycle, and it is added later by somebody who wants it rather than assumed now.

A name in DNS excludes the operator who has a host and no name. It is the one
assumption with no workaround inside this project, because a certificate needs a
name and a browser needs a certificate.

Inbound reachability excludes the operator who cannot forward a port at all. The
network shape record says what they are told and when, and the relay in issue #120
is what recovers the case where the participants rather than the host are the
problem. An operator whose own host is unreachable from outside is not served by
this project, and that is written down rather than discovered.

Automatic certificates cost one outbound connection in a deployment whose whole
argument is that nothing leaves the host. The issuing service learns the hostname,
and the certificate is published to public transparency logs where anyone can read
it. For a self-hosted deployment a hostname frequently identifies the community,
the organisation or the person running it. That is the federation record's
exception, stated there in full, and it is carried into the sovereignty statement
in issue #101 with the same wording rather than a shorter one. The operator who
will not accept it supplies their own certificate, which is why the assumption
list says the command may not assume a certificate the operator obtained
themselves rather than saying it must obtain one.

## What the command must never do

Ask a question whose answer this project could derive. The configuration record
holds the rule and the four categories that survive it.

Print a value for the operator to copy into a file. A value that has to be moved
by hand is a second command wearing a disguise.

Succeed while leaving the service unreachable. An exit code of zero on a
deployment that cannot accept a participant is the failure this contract exists to
prevent, because it moves the discovery from the install to the first call.

Leave the operator to discover from a media failure what a check could have told
them at startup. Issue #56 checks the assumption list before the command commits
to anything, issue #81 is the startup self-check, and issue #62 is what speaks when
media does not flow.

## How the contract is tested rather than asserted

Every clause above is a claim, and a claim about a deployment is worth what its
test is worth.

Issue #55 runs the command against a clean host and asserts that a participant is
accepted and media is forwarded afterwards, with no file edited and no second
command. That is the promise, tested end to end, and it is the issue that decides
whether the rest of this record is true.

Issue #56 turns the assumption list into checks that run before the command does
work, so that a missing assumption is reported as a missing assumption rather than
as a failure somewhere later.

Issue #59 proves the default configuration is complete, which is the never list's
first clause tested from the other direction: a service that starts with only the
values this record permits is a service that asked no question it could have
derived.

Issue #58 prints every derived value with the input it came from, which is what
makes a wrong derivation visible rather than invisible.

None of those four has run. Until they have, this record describes what the
deployment is meant to do and no run of anything has confirmed it.

## The cost of the contract itself

A contract this tight gets harder to keep with every feature. Each new capability
arrives with a value somebody wants to configure and a dependency somebody wants
to assume, and each individual request is reasonable.

That is why the never list is short enough to remember and why the assumption list
is closed rather than illustrative. The next argument about a setting is an
argument against this record, in one place, rather than a decision made inside a
pull request that was about something else.
