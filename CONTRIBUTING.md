# Contributing

## Before you push

```
golangci-lint run
```

That is the command. It formats, it runs the language's own correctness vetting,
and it runs the linter set, all three from `.golangci.yml`, which is a tracked
file rather than a runner default. The style gate on the server runs the same
command with the same file, so a green run here is a green run there.

Install the version the gate pins, which is the one in
`.github/workflows/lint.yml`:

```
go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.12.2
```

It also refuses a tree that does not compile, which is worth knowing because of
how it says so. A compilation failure is reported as a typechecking error and the
summary line still reads `0 issues`, while the exit code is non-zero. Read the exit
code, not the last line.

What it does not do is run the tests, and what runs on the server is more than
this. The pull request page is the authority for what ran and what it said; this
document does not list the checks, because a list in prose drifts against the
thing it describes from the day it is written.

## Every change starts as an issue

An issue says what is wrong, what the evidence is, and what done means.

Where the evidence is a number, it carries the command that produced it, run
against the reference a reader will have rather than against your working tree.

Where a claim cannot be backed by a command, write it as a claim. "Verified", "not
measured" and "not evaluated" are three different statements and the sentence
should say which one it is. A negative disclosure stays negative: if a passage
admits something was not done, that admission survives every later edit.

Nothing here refuses a change for skipping any of this. It is what a reader checks.

Who decides what merges, how a decision is recorded, what you can expect back
when you propose something, and what happens to this project if the maintainer
stops is in [GOVERNANCE.md](GOVERNANCE.md).

## Sign your work

Every commit carries a `Signed-off-by` trailer matching its author, which is how
you assert the [Developer Certificate of Origin](DCO). Read that file first; it is
short, and it is the thing you are certifying.

Add the trailer as you commit:

```
git commit -s
```

If you have already committed without it, add it to everything since the base:

```
git rebase --signoff origin/main
```

This one is refused by a machine. A commit without a matching trailer reds the
sign-off gate and the change does not merge.

## Sign your commits

The trailer above is a sentence you wrote into the commit. A signature is a
cryptographic claim about who produced it, checkable by anybody holding the public
half of the key. Two mechanisms, two failure modes, and they sit next to each
other here because the names are close enough to be read as one thing. Adding the
trailer does nothing about the signature and signing does nothing about the
trailer.

Configure it once per clone. An SSH key needs its format named as well as the key
itself:

```
git config commit.gpgsign true
git config gpg.format ssh
git config user.signingkey ~/.ssh/id_ed25519_signing.pub
```

Read what you produced rather than the setting you asked for:

```
git log --format='%h %G? %s' -1
```

`G` is a good signature and `N` is none at all. `E` means the signature could not
be checked in your clone because the key is not there, which is what a merge made
on the server looks like from here. The server keeps its own answer, and it is the
one a requirement would read:

```
gh api repos/iderex/relais/commits/9ff2f29 --jq '.commit.verification | {verified, reason}'
{"reason":"valid","verified":true}
```

That answer is `verified: false` until the signing key is registered on the account
that pushes, so the local `G` above is necessary and not sufficient.

Where the requirement is in force, one unsigned commit anywhere in a branch's
history refuses the merge, not only the last one. So the repair is to sign the
history again, and signing a commit changes it, which means the branch is rewritten:

```
git rebase --exec 'git commit --amend --no-edit -S' origin/main
```

The repair is never to turn signing off for the commit that failed. Both spellings
of that are written here so that neither is mistaken for a fix:

```
git commit --no-gpg-sign
git -c commit.gpgsign=false commit
```

A commit made either way compiles, tests and reads exactly like a signed one, and
nothing between here and the merge says otherwise. The refusal arrives at the end,
on a commit made days earlier, and it costs the whole branch.

Nothing refuses an unsigned commit here today. The ruleset carries no signature
requirement, read at this commit rather than assumed:

```
gh api repos/iderex/relais/rulesets/20487474 --jq '{enforcement, bypass: .bypass_actors, types: [.rules[].type]}'
{"bypass":[],"enforcement":"active","types":["deletion","non_fast_forward","pull_request"]}
```

Issue #98 is the request for it and it has not been granted. Until it is, this
section describes a requirement that is asked for and not in force, and signing is
a courtesy to whoever grants it rather than something the tree makes you do.

## Line endings

The tree is LF everywhere, in the repository and in your working copy, and
`.gitattributes` is what makes that true on all three operating systems.

You should not need to set anything locally. If your `core.autocrlf` is `true`,
which is the default on Windows, `.gitattributes` overrides it for this
repository and you can leave it alone. The reason to know it exists is that
without the rule, `golangci-lint run` reports every file in the tree as
unformatted on Windows and passes on the server, for a reason that has nothing to
do with your change.

Files under `testdata/` are exempt and keep exactly the bytes they were committed
with, because they are inputs a test asserts against. A fixture whose line endings
were rewritten on the way through a checkout is a test passing against the wrong
input.

This is not the Unicode guard, and the two are easy to confuse because both are
about bytes in text. The guard refuses bidirectional and invisible control
characters, which is text that renders differently from how it parses. This rule
is about bytes changing under you between one checkout and the next. Neither
covers the other.

## Suppressing an analyser finding

A suppression names the analyser and gives the reason on the same line:

```go
//nolint:errcheck // the write is to a buffer that cannot fail
```

A bare `//nolint` suppresses everything, including findings that did not exist
when it was written, so it is not used here. Neither is a suppression whose reason
is that the finding is wrong: if it is wrong, the linter's configuration is what
changes, in `.golangci.yml`, where the next person can see it.

`nolintlint` refuses that. A suppression naming no analyser reds the style gate,
and so does one that names an analyser and gives no reason. It also reports a
suppression that no longer silences anything, which is the failure the convention
above cannot catch by reading: the finding was fixed, the comment stayed, and it
now covers whatever lands on that line next.

## How large a change should be

Around 400 lines, and the number matters less than the reason.

A change that will not fit without losing quality is usually an issue that was
scoped wrong, and the first response is to split the issue rather than the
finished diff. Two pull requests carved out of one change only make sense together
and neither is reviewable alone, which satisfies the number and defeats its
purpose. Dividing the issue gives each piece its own reason to exist and its own
definition of done.

Some large changes are genuinely one thing to read, because a single property
holds across every changed byte and a reader checks the property instead of the
diff. Where that is the case, the pull request body says what the property is.

Nothing measures this. It is a judgement, and it is made in review.

## One topic per commit and per pull request

A commit message says what changed and what failure it prevents. Where it is a
correction, it says what was wrong and how it was found.

A commit carrying two unrelated changes ends up with a message describing one of
them.
