# Parity with the reference gate

The target for this repository's gate is the one already running on the single
sign-on plugin at github.com/iderex/jellyfin-plugin-sso. It is public, its ruleset
is readable, and it is a standard this repository is measured against rather than
one invented here.

Recorded for issue #86. Parity is not copying: that gate is for a plugin loaded
into somebody else's process, in another language, shipped through a package
manifest. This one is for a network service that terminates connections from
strangers and carries live conversation. Every place the two differ is a deviation
and each owes a reason.

## What the reference requires, at this commit

    gh api repos/iderex/jellyfin-plugin-sso/rulesets --jq '.[] | select(.name=="Protect main and 5.0") | .id'
    18802863
    gh api repos/iderex/jellyfin-plugin-sso/rulesets/18802863 --jq '{enforcement, bypass: .bypass_actors, required: [.rules[] | select(.type=="required_status_checks") | .parameters.required_status_checks[].context]}'
    {"bypass":[],"enforcement":"active","required":["build","ABI floor build","Package (JPRM) / Build package","Package (JPRM) / Generate SBOM","CodeQL","Analyze (csharp)","DCO sign-off","Deterministic PR-hygiene checks","Enforce greppable invariants","Reject Trojan Source Unicode","Audit workflows (zizmor)","prettier","dependency-review"]}

## What this repository requires, at the same commit

    gh api repos/iderex/relais/rulesets/20487474 --jq '{enforcement, bypass: .bypass_actors, types: [.rules[].type]}'
    {"bypass":[],"enforcement":"active","types":["deletion","non_fast_forward","pull_request"]}

No required status check exists here at all, so the distance is the whole list.
Several of the checks below already run on every pull request; none of them is
required to be green before a merge, which is a different statement and is what
issue #29 changes.

## Per element of the reference gate

`build`. Adopted. A tree that does not compile makes every other result
meaningless. It runs here already and #29 makes it required.

`ABI floor build`. Dropped. It exists because a plugin is loaded into a host
process whose interface version it must not exceed. Nothing loads this service
into anything, so the check has no analogue rather than a weaker form.

`Package (JPRM) / Build package`. Adapted, into a reproducible container image in
issue #60. The reference builds a package for one application's package manager;
what an operator of this service receives is an image or a binary.

`Package (JPRM) / Generate SBOM`. Adopted, and already running: the build gate
produces a bill of materials from the binary it just compiled rather than from a
separate scan of the source, which landed under issue #21. Publishing it with a
release is issue #113.

`CodeQL` and `Analyze (csharp)`. Adapted, into one analyser for the language this
project chose, in issue #22. Two check names there are one analyser configured for
one language; here the second lens is a different tool entirely and is issue #88,
which does not pretend to be the same element.

`DCO sign-off`. Adopted unchanged, and already running. The certificate is the
same certificate whatever the artefact is.

`Deterministic PR-hygiene checks`. Adopted, in issue #93. The reference's version
reads a pull request body for the fields its process requires; the fields here are
this project's own, and the determinism is the part being copied.

`Enforce greppable invariants`. Adopted, in issue #92. The invariants themselves
are this project's, and the shape of the check is what carries over.

`Reject Trojan Source Unicode`. Adopted unchanged, and already running. Text that
renders differently from how it parses is a defect in any language.

`Audit workflows (zizmor)`. Adopted unchanged, and already running. The workflows
are the same kind of attack surface in both repositories.

`prettier`. Adapted, and split in two, because one tool there covers both jobs.
The code half is the language's own formatter inside the style gate, which landed
under issue #19. The documentation half landed under issue #94 as its own gate,
and it is a deviation rather than a copy: the reference formats its written
surfaces, and this one judges what they claim. A link resolves, a fragment names a
heading that exists, a repository path written in the prose is a path in the
repository, a block of output carries the command that produced it, and a command
closes the quotes it opens.

That last pair is where the adaptation earns its cost, and the gap underneath it
is the reason this paragraph is longer than the others. Nothing in that check runs
a documented command to see whether the output printed beside it is still what
that command produces. Every figure in this document was true when it was pasted
and no route re-reads any of them, so a page whose numbers went stale a year ago
passes the gate exactly like one written this morning, and it passes while
carrying the one thing this project's own rule about evidence exists to prevent: a
number a reader believes because a command is sitting above it. The check makes
the absence of a command refusable and says nothing about the truth of an output.

Closing it means a route that executes documentation rather than a stricter reader
of it, which is a different kind of artefact with its own failure modes: a
documented command that reaches the network, costs money, or changes state is one
a gate must not run on every pull request. Nothing on the board asks for it today,
and that is a reading of the board rather than a command's answer. It is recorded
here because the gap is larger than what was closed, and a parity document that
recorded only the half that landed would be the same defect one level up.

`dependency-review`. Adopted unchanged, and already running.

## What the reference carries without gating on it

A coverage bar pinned to the code that decides security outcomes rather than to
the whole tree:

    gh api repos/iderex/jellyfin-plugin-sso/contents/scripts/check-coverage.py --jq .content | base64 -d | grep -n 'SECURITY_LINE_BAR'
    68:SECURITY_LINE_BAR = 92.0

Adapted. The idea carries over exactly and the subject changes, from authentication
decision code to the admission, authorisation and placement surfaces, which is
issue #89.

Mutation testing. Adopted as reporting rather than gating, in issue #90, which is
the same posture it has there.

Fuzzing. Adopted and pointed at a different surface, in issue #91. There it is
input to an authentication flow; here it is the media boundary, which takes bytes
from strangers by design.

An end-to-end login harness. Adapted into the media integration harness, issue
#40. Both exist to run the real path rather than a mocked one, and what the real
path is differs completely.

## What this repository adds

Each of these is a deviation upward, and each exists because this is a service
carrying live media rather than a plugin.

Load and soak evidence, issue #99. A plugin's behaviour after six hours under load
is not a question anybody asks; for this it is most of the question.

A resource budget guarded against regression on a schedule, issue #68. The
reference has no counterpart because it is not selling a resource figure. This
repository is, and a number nothing defends stops being true quietly.

Proof that every test runs with no display, no elevation and no special hardware,
issues #18 and #96. The reference's suite is a unit suite by nature. This one has
every temptation to need a device, and the constraint is held by where the suite
runs rather than by a rule.

The architecture rules as tests, issue #95. The port between orchestration and
media is the property this whole project rests on, and an import that breaks it
compiles perfectly.

A lock file that cannot drift, issue #20, and the supply chain around it, issue
#97. The reference gates on dependency review, which reads what changed; this adds
the requirement that what is restored is exactly what was recorded.

Verified signatures on the protected branch, issue #98. The reference ruleset does
not require them:

    gh api repos/iderex/jellyfin-plugin-sso/rulesets/18802863 --jq '[.rules[].type]'
    ["deletion","non_fast_forward","required_status_checks","pull_request"]

The cost falls on somebody sending a first change, and it is taken knowingly
rather than absorbed. A signature is a key generated, named in git, and registered
on the account that pushes, which is three steps standing between a one-line fix
and the branch it is on, none of them about the fix. The failure then arrives late
and reads as arbitrary: every check green, and the merge refusing on a commit made
days earlier. The repair is a rewrite of the branch rather than an addition to it,
which is the operation somebody new to git is least willing to run, and the reason
[the contribution guide](../CONTRIBUTING.md) carries the command that performs it
beside the two spellings of the bypass that is not the repair. What is bought for
that is the reason it is paid: an operator deciding to run a release built from
this history is trusting the history, and without the requirement a commit's
stated author is a field anybody can write.

A reproducible release from a tag, issue #112. An operator who cannot rebuild the
artefact they are running cannot check that it is the artefact they were given.

## What this document is not

It is not a claim that any of the adopted elements is required here today. The
second command above is the authority for that, and it says none of them is.

It is not a schedule, and it names no order. Which of these lands first is decided
by the milestones, not here.
