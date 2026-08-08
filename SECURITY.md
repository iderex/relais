# Security policy

Somebody will find a vulnerability in this project. What happens next is written
down here rather than improvised, because a project that decides its disclosure
process while handling its first report handles that report badly.

Recorded for issue #105.

## Reporting

Use the private reporting route on this repository:

    https://github.com/iderex/relais/security/advisories/new

That route is enabled, which is a fact about the repository rather than a claim
about this document:

    gh api repos/iderex/relais/private-vulnerability-reporting
    {"enabled":true}

It opens a draft advisory that only the maintainers of this repository and you
can read. Do not open a public issue for a vulnerability. An issue is world
readable from the moment it is created, and asking for a vulnerability there is
asking for it to be published before anybody can act on it.

A report is most useful when it carries the version or commit it was found
against, what an attacker gets out of it, and the smallest sequence that
reproduces it. None of that is required. A report with less in it is still worth
sending, and a request for more detail is not a rejection.

## What a reporter can expect

An acknowledgement within three days, saying the report was received and read.

An assessment within fourteen days, saying whether it is accepted, what the
severity is judged to be, and what the intended fix is. Where the assessment says
this is not a vulnerability, it says why, and a reporter who disagrees is welcome
to say so.

A named date for publication, set at the assessment and not moved later without
telling you.

Credit in the published advisory under whatever name you ask for, or none. This
is asked rather than assumed, because a reporter who wants to stay anonymous
should not have to request it after the fact.

These are the intervals this project commits to. They are not a service level
agreement and nothing enforces them, which is a fact worth knowing before a
report is sent rather than after.

## Disclosure

A finding is published ninety days after the report is received, whether or not a
fix exists.

Ninety days is long enough for a fix to be written, reviewed and released, and it
is the period most of the industry already works to, so it needs no explaining to
a reporter and no negotiating over. It is the position taken here rather than
inherited: an operator running this on their own hardware is the person the
deadline is for. They cannot mitigate a problem they have not been told about,
and a deadline that slips whenever a fix is late protects the project's
appearance at the expense of the people running it.

Publication earlier than ninety days happens where a fix is ready and released
sooner, which is the ordinary case, or where the finding is already public or
being exploited, in which case waiting protects nobody.

Publication later than ninety days happens only where the reporter asks for it.
The extension belongs to the person who found the problem and not to the project
that has it.

Where the deadline arrives without a fix, the advisory says so and says what an
operator can do instead, even where that is switching a feature off or taking the
service off a public network. An operator told a workaround is better placed than
an operator told nothing.

## Where advisories appear

Published advisories go to this repository's security advisories, which feeds the
GitHub Advisory Database. That is the route a dependency scanner and a
notification both come from, and it reaches an operator who is not reading the
issue tracker.

The release notes for the release carrying the fix name the advisory, so an
operator deciding whether to upgrade can see the reason without leaving the
release page.

The issue tracker is not an advisory route. A fix lands as an ordinary change
with an issue behind it like everything else here, and that issue is not where an
operator learns they are exposed.

## Which versions receive fixes

Nothing has been released yet:

    gh api repos/iderex/relais/releases --jq 'length'
    0
    gh api repos/iderex/relais/tags --jq 'length'
    0

So today there is exactly one thing to fix and it is the default branch. An
operator running this is running a commit rather than a version, and the fix
reaches them when they take a newer one.

Once the first release is cut, which is issue #117, this section states the
supported set: the current release always, and the one before it for a stated
period. Issue #115 decides the upgrade path this rests on, and this section is
rewritten from its answer rather than guessed at now. A supported-versions table
written before any version exists would be a promise made about releases nobody
has planned.

## What is in scope

The threats this project accepts as its own are listed in
[the threat model](docs/threat-model.md), along with the adversaries each one
belongs to and what stands against it today. A report about anything in that
document is in scope, including a report that a mitigation named there does not
work, or does not exist where the document says it does.

That document also lists
[what has no mitigation at all today](docs/threat-model.md#what-has-no-mitigation-at-all-today).
Those are known
gaps rather than accepted risks, and a report about one is still worth sending:
knowing that somebody hit a gap in practice is different from knowing it is
there.

## What is out of scope

[The threat model's own out-of-scope section](docs/threat-model.md#what-is-out-of-scope-and-why)
is the authority, and it carries the reason for each entry. It is not restated here,
because a second copy would drift against it and a reporter would then have two
answers to choose from.

Beyond that document, three things are out of scope here and are named so a
report about one gets a fast answer rather than silence:

A finding in a dependency, rather than in how this project uses it. Report it to
the dependency. Where this project uses a dependency in a way that creates the
problem, that is in scope and is this project's to fix.

A scanner result with no reachable path behind it. A configuration that a tool
flags, in code nothing calls, with no input that reaches it, is worth an issue
rather than an advisory.

An attack that assumes the operator's host is already compromised. Everything on
that host is reachable from there, and this project does not defend against it,
which the threat model says in its own words.

A report that turns out to be out of scope gets that answer with its reason,
inside the same fourteen days as any other. A fast honest no is what the
out-of-scope list is for.
