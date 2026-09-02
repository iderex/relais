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
    {"bypass":[],"enforcement":"active","types":["deletion","non_fast_forward","pull_request","required_status_checks"]}

    gh api repos/iderex/relais/rulesets/20487474 --jq '[.rules[] | select(.type=="required_status_checks") | .parameters.required_status_checks[].context]'
    ["Audit workflows (zizmor)","Build","DCO sign-off","Lint","Reject Trojan Source Unicode","Test","dependency-review"]

Seven contexts have to be green before a merge. This section said none did, which
was true when it was written and stopped being true without anything here
noticing, and correcting it is issue #162.

That is seven of the eight issue #29 asked for. The eighth is `CodeQL`, which
runs on every pull request and is not required, and no reason for leaving it out
is recorded where this document can read one.

Three of the checks that run here are outside the requirement. Read off the head
of a pull request rather than from the workflow files, and restricted to the app
that runs the jobs in this tree:

    gh api repos/iderex/relais/commits/54a4a73dfa53f1af7812f5f422575bf4df1b4a91/check-runs --jq '[.check_runs[] | select(.app.slug=="github-actions") | .name] | unique | .[]'
    Audit workflows (zizmor)
    Build
    CodeQL
    DCO sign-off
    Dependencies
    Documentation
    Lint
    Reject Trojan Source Unicode
    Test
    dependency-review

`CodeQL`, `Dependencies` and `Documentation` are the three in that output and not
in the requirement. The distance from the reference is therefore one granted
requirement short of what was asked for, plus the elements below that are not
built yet, rather than the whole list.

The strict policy is off, so a branch does not have to be up to date with the
default branch before it merges:

    gh api repos/iderex/relais/rulesets/20487474 --jq '[.rules[] | select(.type=="required_status_checks") | .parameters.strict_required_status_checks_policy]'
    [false]

## Per element of the reference gate

`build`. Adopted. A tree that does not compile makes every other result
meaningless. It runs here and is required, as `Build`.

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

What that analysis is pointed at, and how, is in the workflow rather than
described here:

    git grep -n 'languages:\|build-mode:\|queries:' -- .github/workflows/codeql.yml
    .github/workflows/codeql.yml:77:          languages: go
    .github/workflows/codeql.yml:81:          build-mode: autobuild
    .github/workflows/codeql.yml:87:          queries: security-extended

What it does not cover is recorded for issue #87, so a green result is not read
as coverage it does not provide. None of the following is a measurement. Each is
a claim about the kind of question this analyser answers, and the reason it is
worth writing down is that every one of them is a class this particular service
is exposed to.

Concurrency. A race between a publisher's write and a subscriber's read, or a
lock held across a network wait, is not the shape a taint analysis is looking
for. The race detector in the test suite is the tool for that class, and it sees
only what a test actually executes.

Resource cost. An allocation per packet, a buffer that grows with a stranger's
input, or a loop whose cost is quadratic in the number of participants are
correctness-neutral to the analyser and are most of what the resource budget in
this project rests on.

Whether a decision is the right decision. A path from a credential to an
authorisation check is visible; whether the powers that credential carries are
the powers actually enforced is a question about meaning, and no analyser reads
`docs/decisions/admission.md`.

Key handling away from the call site. A signature verified correctly in the code
and a key distributed to the wrong place read identically from the source.

Code the build does not compile. The mode is autobuild, so what is analysed is
what the ordinary build produces on the runner. Anything behind a build
constraint that the runner does not select is not read at all.

The severity at which a finding here stops a merge is in the tree, which is the
second thing issue #87 asks for. This section said there was none, and that was
true until the number below landed:

    git grep -n 'const Threshold' -- test/scanseverity/scanseverity.go
    test/scanseverity/scanseverity.go:71:const Threshold = 7.0

The number stands above as the output of the command that produces it rather than
as a sentence stating it, so a reader who doubts this page re-runs one line. The
argument for that value against a lower one is in the package beside it and is
not repeated here.

What belongs in this document is the shape of the decision. The threshold is
applied after the findings are uploaded, so it decides what stops a merge and
never what is reported: a finding under it is still raised, still on the code
scanning surface and still somebody's.

It judges the file the analyser wrote rather than asking the code scanning
surface what it made of it. That keeps the verdict offline and identical on a
pull request from a fork, where that surface is not readable, and it means the
same file judged twice gives the same answer. The reading fails closed on the
three shapes in which a pass would mean it had read nothing, and those are
fixtures in the suite rather than an assurance in this paragraph:

    go test ./test/scanseverity/ -run TestTheReadingFailsClosed -count=1

What is still not measured is whether the code scanning surface would have failed
this check on its own. That is a repository setting rather than something a
reader of this repository can see, and the threshold above neither depends on it
nor replaces it.

`DCO sign-off`. Adopted unchanged, and already running. The certificate is the
same certificate whatever the artefact is.

`Deterministic PR-hygiene checks`. Adopted, and already running as `PR hygiene`,
under issue #93. The reference's version reads a pull request body for the fields
its process requires; the fields here are this project's own, and the determinism
is the part being copied. Five properties are refused: an issue is resolved as
closed by the change, the means section carries something the template did not
already write, a block of figures in the body carries the command that produced
it, every commit message has a subject and a blank line under it, and every file
touched is inside the `Scope:` the closing issue declares. What each rule is and
why is in `test/prhygiene` beside the property it holds.

Determinism is what the job is arranged around rather than a claim about it. One
query runs, its response is written to a file, and the verdict is a function of
that file, so a reader can re-run the query and hold the input the gate saw. The
reading fails closed on a response carrying no pull request, one carrying no
commit, and one carrying another page of anything, and those are fixtures in the
suite rather than an assurance in this paragraph:

    go test ./test/prhygiene/ -run TestTheReadingFailsClosed -count=1

What it cannot decide is the larger half, and it is recorded here so a green
result is not read as a reviewed body. It refuses an empty box and never a false
statement. A means sentence naming the wrong means passes, and whether the answer
is right is the judgement the contribution guide already says review makes. An
evidence block whose command was never run passes, and so does one whose output
stopped being true a year ago, which is the same gap the documentation rules
carry two sections above and for the same reason: nothing here executes a
documented command to see what it produces now.

A commit message stating neither what changed nor what failure it prevents
passes. That is what `CONTRIBUTING.md` actually asks of a message, and no reading
of the message decides it; what the rule holds is the shape every git view
assumes, which is a subject and a break.

The scope rule is the weak test of one topic and it refuses, which is what issue
#93 asks for. It compares paths and cannot tell work that belongs to an issue
from work that merely lands in the same directory, so two unrelated topics inside
one declared scope pass. In the other direction the consequence is worth stating
because it falls on the author rather than on the gate: where a change lands
outside the declared scope the repair is the issue and not the diff. The scope
was written before the means was chosen, the means put the work somewhere else,
and the `Scope:` line is corrected before the change lands. That is the
contribution guide's own answer to a change that does not fit, one rung earlier,
and it is a heavier procedure than a check that merely reported the crossing
would have been.

The template's other two sections, the evidence and what was not checked, are not
judged at all. Issue #93 names five mechanical items and neither is among them.

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
the same posture it has there, and this repository has two reasons of its own for
it. It is slow, so it does not belong beside the checks somebody waits on before
merging. And a surviving mutant is something for a person to read rather than a
verdict, because a mutant can be equivalent to the code it replaced and then
nothing can ever kill it. What is done with a survivor is a rule instead of a
threshold, and it is under mutation evidence below.

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
is not a question anybody asks; for this it is most of the question. It is four
pieces of evidence rather than one, and the section under this list is where they
are set out.

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

## Load and soak evidence

Recorded for issue #99. Four pieces of evidence stand where the reference gate has
nothing at all, and they are gathered here rather than left one to an issue apiece
because the argument only holds when they are read together. Separately each is a
measurement. Together they are the answer to whether a deployment is still working
after a week nobody watched it.

None of the four exists. All of them run on the bench, and the bench today is one
file saying where the instrument will be made:

    git ls-files bench/
    bench/README.md

So everything below is what this document asks of each piece, with the issue that
delivers it. None of it describes something that runs.

The budget guard is issue #68. It measures the recorded budget lines against the
mainline on a machine whose shape is stated, and reports a figure that has moved
beyond a derived tolerance as a finding naming the range of commits it appeared
in. It is the only one of the four that runs on a clock, and what this document
asks for is at least once a week. The week is paid for twice: once in the time a
resource claim is wrong while nothing has noticed, and once in the commits that
have to be bisected afterwards to say which change did it. Weekly rather than
nightly because a shared runner is not a laboratory and issue #68 derives its
tolerance from measured run-to-run variation rather than from a guess, and a
tighter cadence buys attribution with noise. Weekly rather than monthly because a
month of commits is where attribution stops being worth the run at all.

The soak is issue #69. It holds a stated load for stated hours with participants
joining and leaving throughout, samples memory, handles, thread counts, open
sessions and internal delay, and judges drift against a stated rule instead of
against a chart. It runs on no clock. What this document asks for is one run per
release, and one before any release whose notes carry a resource figure, because
what it finds is what an operator meets after a month rather than after a merge.
Its detection delay is therefore the gap between two releases, and that gap has no
end yet:

    gh api repos/iderex/relais/releases --jq 'length'
    0

The past-the-ceiling measurement is issue #70. It pushes a host beyond its limit
deliberately and records what that costs the sessions already on it, whether new
work is refused cleanly rather than accepted and starved, and how far ahead of the
participants the health and metrics surfaces saw it coming. What this document
asks for is one run per release, and one more whenever the degradation policy or
the placement rule changes, since those two are what it is checking. Its detection
delay is the soak's, for the soak's reason.

The network characterisation is issue #42. It runs a stated matrix of loss, delay,
jitter and bandwidth conditions and records what happens to the picture, to the
audio, to the delay and to recovery once conditions improve. What this document
asks for is one run per release, and one more whenever layer selection or
bandwidth allocation changes. Its detection delay is the soak's again.

What the four have in common is the part that has to be said plainly rather than
left for a reader to work out. Not one of them runs on a pull request, so not one
of them stops a bad merge. They detect, at the delays above, and a detection is a
finding written after the change is already on the mainline and possibly already
in somebody's deployment. The two commands under what this repository requires are
the authority, and none of the four is among what they name:

    gh api repos/iderex/relais/rulesets/20487474 --jq '[.rules[] | select(.type=="required_status_checks") | .parameters.required_status_checks[].context]'
    ["Audit workflows (zizmor)","Build","DCO sign-off","Lint","Reject Trojan Source Unicode","Test","dependency-review"]

What that leaves open is a regression that lands and is not found until the next
run. For the budget guard the window is a week of commits. For the other three it
is a release, which is unbounded today because nothing has been released. A
resource figure in a document, or in an operator's decision to adopt this, can be
wrong for that whole window without anything here noticing, and what stands
against it in between is review, which reads a diff and cannot see five per cent.

Two conditions would make any of them a required context, and both are numbers
rather than intentions.

The first is cost. One run has to finish inside five minutes on the runner the
gate already uses. Five minutes is a choice; the measurement it was chosen against
is not. The seven required contexts together span sixty-one seconds on a pull
request head, from the first one starting to the last one finishing:

    gh api repos/iderex/relais/commits/c71aaedffafdc2c551ac3f0f4f6b3aeee903272f/check-runs --jq '[.check_runs[] | select(.app.slug=="github-actions") | select(.name=="Audit workflows (zizmor)" or .name=="Build" or .name=="DCO sign-off" or .name=="Lint" or .name=="Reject Trojan Source Unicode" or .name=="Test" or .name=="dependency-review")] | "\([.[].started_at] | min) to \([.[].completed_at] | max)"'
    "2026-08-10T07:49:25Z to 2026-08-10T07:50:26Z"

Five times that is the point past which somebody waiting on the gate stops waiting
and starts merging on the assumption it will come back green, which is the failure
a longer gate actually produces.

The second is noise. The run-to-run variation of a figure has to be smaller than
half the smallest regression that figure's line is meant to refuse, so that one
crossing decides it. Above that only a sustained series decides, and a check that
needs a series before it can refuse anything is a report whatever it is called.

The soak fails the first condition by construction and will go on failing it. It
is hours long because hours is what it measures, and a version of it that fits in
five minutes measures something else. It is permanently evidence rather than a
gate, and that is a property of the question rather than a gap in the plan. The
other three could meet both conditions one day on a reduced form of themselves,
and the reduced form would be a different measurement under the same name, which
is worth noticing before somebody asks for it.

The cadences here are how often a piece of evidence runs. They are not an order in
which the four are built, which is the milestones' business and is not decided
here.

## Mutation evidence

Recorded for issue #90. A coverage bar says a statement was reached. This says an
assertion would have noticed if the statement were wrong, and the two disagree
most sharply on the surface the bar is highest on, because a suite written to
reach a branch and a suite written to decide it look the same in a profile.

What runs is `.github/workflows/mutation.yml`, weekly and on request, over every
package the toolchain reports under `internal/`:

    go list ./internal/...
    github.com/iderex/relais/internal/mediaplane
    github.com/iderex/relais/internal/orchestration/credential

That set is not written into the workflow. It is the set the coverage bar is
closed over, since `test/coverage` refuses a package under `internal/` carrying no
entry, so every package this run reaches is one that list has already given a bar
or a written reason not to.

### What happens to a surviving mutant

The rule is by surface, and the surfaces are read off the same list rather than
off a second one kept here.

A survivor on a surface carrying a bar becomes an issue. The argument is the bar's
own: that list is where somebody wrote down which packages decide a security
outcome, and a mutant that lives on one of them is a branch deciding something
with nothing watching it.

A survivor on a surface the list carries at a bar of zero is recorded in this
section and nothing else follows. `internal/mediaplane` is that case. It is the
vocabulary both sides of the port share and it decides nothing about who may do
what, so a mutant living there says something about the suite rather than about a
decision.

Neither half is a threshold, and that is deliberate. An equivalent mutant can
never be killed, so a percentage this job refused below would be a number
somebody lowers rather than a rule anybody keeps.

### One taken through it

A run at the commit this section landed on produced one survivor, on the surface
that carries a bar:

    gremlins unleash --timeout-coefficient 10 ./internal/orchestration/credential
           LIVED CONDITIONALS_BOUNDARY at verify.go:68:16
    Mutation testing completed in 47 seconds 407 milliseconds
    Killed: 32, Lived: 1, Not covered: 0
    Timed out: 9, Not viable: 0, Skipped: 0
    Test efficacy: 96.97%
    Mutator coverage: 100.00%

That line is where a token is bounded before anything decodes it, and the mutant
moves the comparison by one position, so a token of exactly the permitted length
is refused as oversized instead of read. It reproduces without the tool, which is
what makes it a finding rather than a reading of a report:

    sed -i '68s/len(token) > maxTokenBytes/len(token) >= maxTokenBytes/' internal/orchestration/credential/verify.go
    go test ./internal/orchestration/credential/ -count=1
    ok  	github.com/iderex/relais/internal/orchestration/credential	0.737s

Read that as dated 2026-08-11, the commit this section landed on. The suite was
green with the bound moved, which is what the rule exists to catch: a bound that
is checked and never approached. It went to issue #173.

#173 closed with a test that kills it, so the same one-character edit reds the
suite now. Run at `dcaf14bf48757578b65783dc9f92c35fb2b64741`:

    sed -i '68s/len(token) > maxTokenBytes/len(token) >= maxTokenBytes/' internal/orchestration/credential/verify.go
    go test ./internal/orchestration/credential/ -count=1
    --- FAIL: TestTheLengthBoundIsDrivenAcrossTheByteItTakesEffectOn (0.05s)
        credential_test.go:490: a credential of exactly 4096 bytes was refused: credential refused: too-large: the token is 4096 bytes, over the 4096 this project reads
    FAIL
    FAIL	github.com/iderex/relais/internal/orchestration/credential	0.545s

    git checkout -- internal/orchestration/credential/verify.go

What kills it is a credential built to exactly the permitted length and read, then
one byte appended and the refusal required to name the length, so the pair differs
by the single byte the comparison is about:

    git log --format='%h %ad %s' --date=short -1 -S 'TestTheLengthBoundIsDrivenAcrossTheByteItTakesEffectOn' -- internal/orchestration/credential/credential_test.go
    009557e 2026-08-27 Drive the token length bound at the byte it takes effect on

So the sequence is readable here rather than only its first half: the mutant lived,
it reproduced without the tool, it became an issue, and the issue closed with a
test that kills it. The tool has not been re-run since the run above, and nothing
here is a claim about what a fresh one would report.

The other surface produced nothing to place:

    gremlins unleash --timeout-coefficient 10 ./internal/mediaplane
    Mutation testing completed in 18 seconds 372 milliseconds
    Killed: 9, Lived: 0, Not covered: 0
    Timed out: 0, Not viable: 0, Skipped: 0
    Test efficacy: 100.00%
    Mutator coverage: 100.00%

### A timed-out mutant is a mutant nobody judged

The word "a run" above is doing work. This instrument does not give the same
answer twice on an unchanged tree, and what moves is not the survivor but how many
mutants finish at all. The same command, three times in a row, on the tree this
section landed on:

    gremlins unleash --timeout-coefficient 10 ./internal/orchestration/credential
    Mutation testing completed in 52 seconds 760 milliseconds
    Killed: 27, Lived: 0, Not covered: 0
    Timed out: 15, Not viable: 0, Skipped: 0
    Test efficacy: 100.00%

    gremlins unleash --timeout-coefficient 10 ./internal/orchestration/credential
    Mutation testing completed in 54 seconds 705 milliseconds
    Killed: 18, Lived: 0, Not covered: 0
    Timed out: 24, Not viable: 0, Skipped: 0
    Test efficacy: 100.00%

    gremlins unleash --timeout-coefficient 10 ./internal/orchestration/credential
    Mutation testing completed in 45 seconds 603 milliseconds
    Killed: 16, Lived: 0, Not covered: 0
    Timed out: 26, Not viable: 0, Skipped: 0
    Test efficacy: 100.00%

Fifteen, twenty-four and twenty-six of the same forty-two mutants exceeded the
timeout, the survivor was among them every time, and the efficacy line read a
hundred per cent on all three. That is the whole argument for the check the
workflow makes. A timeout is not a kill: it is a mutant the run never decided, and
the score is computed over the ones that finished, so an instrument having a bad
minute publishes a perfect result. The finding above would have been reported by
none of these three runs.

So the job refuses a report carrying a timed-out mutant rather than publishing it,
which is the same shape as the fuzzing gate refusing a pass over no targets and
the coverage gate refusing an empty profile. It refuses nothing else: a survivor
is reported and never fails the job, and this one is about the instrument rather
than about the tree.

Leaving the timeout at the tool's default is worse than a bad minute. On the same
tree it decided nothing at all:

    gremlins unleash ./internal/orchestration/credential
    Mutation testing completed in 32 seconds 883 milliseconds
    Killed: 0, Lived: 0, Not covered: 0
    Timed out: 42, Not viable: 0, Skipped: 0
    Test efficacy: 0.00%
    Mutator coverage: 0.00%

### The cost, and what the cadence is chosen from

Around a minute of mutation across the two surfaces, from the runs above, on a
machine with thirty-two logical processors running the toolchain the tree pins,
on an operating system the gate never uses:

    go version
    go version go1.26.5 windows/amd64

Weekly, and on a morning none of the other scheduled runs uses. The cadence is
chosen against that cost rather than against thoroughness: a week of commits over
two packages is not enough to outgrow a run this size, and a report arriving
nightly is a report nobody opens.

The runner is a different machine and it turns out to be both quicker and
steadier. Asked for by hand, it decided every mutant on both surfaces and found
the same survivor:

    gh run view 31478089166 --log | grep -oE '(Mutation testing completed in .*|Killed: [0-9]+, Lived: [0-9]+, Not covered: [0-9]+|Timed out: [0-9]+, Not viable: [0-9]+, Skipped: [0-9]+|[0-9]+ surviving mutant\(s\) and [0-9]+ undecided mutant\(s\) across [0-9]+ surface\(s\)\.)'
    Mutation testing completed in 1 second 404 milliseconds
    Killed: 9, Lived: 0, Not covered: 0
    Timed out: 0, Not viable: 0, Skipped: 0
    Mutation testing completed in 6 seconds 35 milliseconds
    Killed: 41, Lived: 1, Not covered: 0
    Timed out: 0, Not viable: 0, Skipped: 0
    1 surviving mutant(s) and 0 undecided mutant(s) across 2 surface(s).

Under eight seconds against the minute the same work takes above, which says the
timeouts on that machine were the machine and not the surfaces. The weekly cadence
is chosen against the local figure rather than this one, because a cadence set
from the best number a run ever produced is a cadence that fails the first time
the runner has a bad morning.

That is ONE RUN, ASKED FOR BY HAND. No scheduled run has fired yet, and one run
says nothing about the variation the rule above exists for. Whether the runner
ever times a mutant out is unanswered, and it is answered by this job going red
rather than by anything written here. If it does, the verdict is about the
instrument rather than about the suite, and what settles it is a coefficient or a
worker count measured against the runner rather than guessed at.

### What it does not cover

None of this stops a bad merge. The job runs on a clock and on request, never on a
pull request, so it is absent from the required contexts:

    gh api repos/iderex/relais/rulesets/20487474 --jq '[.rules[] | select(.type=="required_status_checks") | .parameters.required_status_checks[].context]'
    ["Audit workflows (zizmor)","Build","DCO sign-off","Lint","Reject Trojan Source Unicode","Test","dependency-review"]

It is also not a candidate for that list, because a required context has to be
produced by a pull request and nothing here produces one. The detection delay is a
week of commits, and what stands in between is review, which reads a diff.

The reach is smaller than the rule reads, too. The paragraph above about the
coverage bar names admission, authorisation and placement as its subjects, and the
list in the tree holds two entries, one of which carries no bar:

    git grep -n 'Package: "github.com/iderex/relais' -- test/coverage/coverage.go
    test/coverage/coverage.go:89:		Package: "github.com/iderex/relais/internal/orchestration/credential",
    test/coverage/coverage.go:98:		Package: "github.com/iderex/relais/internal/mediaplane",

So the rule about a survivor on a barred surface has exactly one surface to be
about today. Authorisation is issue #46 and placement is issue #75, and neither has
code for a mutant to live in. Everything above is measured on the surface that
exists rather than on the three the argument is written for.

## What this document is not

It is not a claim that an adopted element is required here merely because it runs.
The two commands under what this repository requires are the authority for that,
and they name seven contexts and no others.

It is not a schedule, and it names no order. Which of these lands first is decided
by the milestones, not here.
