Say what changed and what failure it prevents. Where this is a correction, say
what was wrong and how it was found.

## The issue this closes

    Closes #

One topic per pull request. A change that closes nothing is a change nobody
agreed to first, and the contribution guide asks for the issue before the work.

## The means, and why it fits

One sentence naming what this is made of and why that suits the job. The
language, the format, the tool, the runtime, whatever the thing is built from.

It is asked for every time rather than carried over, because a means that was
right for the last change is an assumption about this one. Whether the answer is
right is a judgement made in review. What this section makes checkable is that
the question was asked.

## The evidence

Every number here carries the command that produced it, run at the commit being
pushed and against the reference a reader will have rather than against your
working tree. Paste the command and its output.

Where a claim cannot be backed by a command, write it as a claim rather than as a
result.

## What was not checked

What this change does not cover, what was not measured, and what was skipped and
why. A pull request that says nothing here is claiming it covered everything.
