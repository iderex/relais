# Governance

How decisions are made here, who makes them, and what a contributor gets back
when they propose something. Written before the first disagreement, because a
project that improvises this answers its first hard question badly.

Recorded for issue #109.

## Who decides

One person. This repository has a single maintainer, and every question about
scope, architecture, what merges and what is refused ends there.

That is a fact about the repository, and it is readable from the outside:

    gh api repos/iderex/relais/contributors --jq '[.[].login] | unique | .[]'
    iderex

There is no steering group, no vote and no tie to break, so a document
describing one would describe a project this is not. The honest short version is
better than a borrowed structure, and it is also the version that has to change
first if a second maintainer ever appears.

What the maintainer does not hold is the licence. Contributions arrive under the
Developer Certificate of Origin rather than a contributor licence agreement:

    git ls-files | grep -ciE 'cla|contributor.license'
    0

So nobody here has been given the power to relicense somebody else's
contribution or to sell an exception to it. Relicensing this repository would
need the agreement of everyone who has contributed by then, and that constraint
applies to the maintainer exactly as it applies to anybody else.

## How a decision is recorded

Every change starts as an issue saying what is wrong, what the evidence is and
what done means. That rule and the conventions around it are in
[the contribution guide](CONTRIBUTING.md).

A decision that shapes the architecture is written down before the code that
depends on it, as one file under `docs/decisions/`. What those records are for,
and the rule that a record disagreeing with the tree is corrected first and in
its own change, is in [their own readme](docs/decisions/README.md).

A change then lands as a pull request against the protected branch. Direct
pushes to it are refused, and that refusal reaches the maintainer too:

    gh api repos/iderex/relais/rulesets/20487474 --jq '{enforcement, bypass: .bypass_actors, types: [.rules[].type]}'
    {"bypass":[],"enforcement":"active","types":["deletion","non_fast_forward","pull_request","required_status_checks"]}

The order matters more than the paperwork. An argument that lives only in a
merged diff is an argument nobody can find later, and the reason this project
writes the record first is that a decision defended after the fact is defended
from the code that already exists, and the reasons have stopped carrying it.

## What a contributor can expect

An answer in the issue or on the pull request, saying yes, no, or what would
change the answer.

A reason where the answer is no. A refusal here is written in the pull request
body, and it says which rule or which record it rests on, so a contributor has
something to argue with.

An argument about a rule taken as an argument about the record behind it. Most
disagreements about a convention here turn out to be disagreements with a
decision record, and that record is the shorter thing to argue with.

Credit in the history. The commit keeps its author, and the sign-off trailer
naming that author is a gate rather than a courtesy.

No promise about how quickly any of that happens. This is one person's project
and a waiting contributor deserves to know that before sending work.

## What happens if the maintainer stops

The answer an operator adopting this will want is that the project survives, and
the truthful version is narrower than that.

There is no second maintainer today and no arrangement that hands the repository
to anybody. If the maintainer stops, this repository stops receiving changes,
and it stays readable at whatever commit it stopped on.

What that leaves an operator is set by the licence rather than by goodwill.
Everything needed to continue is in the tree and under
[AGPL-3.0](LICENSE), so anybody may fork it, run it, change it and publish the
result, on the same terms and without asking. There is no dependency on a
service this project runs, no key only the maintainer holds, and nothing an
operator would have to be granted. A fork keeps the licence and the history; the
name and the tracker do not travel with it.

That is the whole of the continuity arrangement, and it is deliberately a
property of the licence rather than a promise from a person, because a promise
from a person is exactly what stops being true in the case this section is
about.

## The code of conduct

There is none in this repository today.

Issue #109 asks for one, and it says the part that is holding it up: a code of
conduct is a standard text plus a contact route, and the route matters more than
the text. A reporting address that goes nowhere invites a report that is never
received, which is worse than having no policy at all. Choosing where such a
report arrives is the maintainer's to make, and that choice has not been made.
The gap is written down here; a text with a dead address in it would be the
worse answer.

Until then, behaviour here is not covered by a published policy. That sentence
is the disclosure, and it should stay accurate rather than be softened.
