# What an operator is never asked to tune

A value this service can work out from the machine, the network or the load is
worked out. A value the operator cannot possibly know is not asked for.

Recorded for issue #7.

## The rule

Derivation is the default and configuration is the exception. Adding a setting is
a change to a recorded list and has to survive a reviewer, rather than something
that accumulates because a derivation was hard on the day.

The readme says there is nothing to hand-tune. Without this rule that sentence
decays into a set of defaults nobody has checked, which is the usual state of a
media server after a year of accepting one reasonable request at a time.

## The categories that remain configurable

Four, and nothing else.

The name the service answers on, together with the network shape that name
implies, because only the operator knows what their network looks like.

The credential material, because only the operator knows what the service above
them is holding.

Where its data lives, because only the operator knows their storage.

What it is allowed to spend, which is the cap described below.

A proposed setting that fits none of these four is refused, and issue #59 makes
that refusal a red test rather than a matter of who reviews it.

## The cap, which is the interesting case

An operator may say how much of their machine this service is allowed to consume.
That is a policy about their hardware, and no derivation can substitute for it,
because the service cannot know what else the machine is for.

An operator may not be asked how many workers to run in order to stay inside that
cap. That is the service's arithmetic, and asking for it hands back the exact
problem the cap exists to avoid.

The line is the difference between stating a limit and implementing one. Every
future setting will be argued at this line, so it is written here rather than
rediscovered each time.

## What is derived, and from what

This is the design as planned. Where a value depends on a decision that is not yet
made, the issue holding it is named rather than an answer being assumed.

The number of forwarding workers is derived from the processors available to the
process, bounded by the operator's cap.

Queue and buffer sizes on the forwarding path are derived from the number of
workers and the per-stream memory the resource budget in issue #10 fixes.

The per-stream and per-host ceilings that admission checks against are derived
from the cap and the same budget, so a host refuses a room it cannot carry rather
than accepting it and degrading everything already running.

The initial bandwidth estimate and the constants the estimator moves from are
derived from the transport library rather than restated here, since the forwarding
core record leaves congestion control to it.

The simulcast layer offered to a subscriber is derived from that subscriber's
measured bandwidth and the allocation policy in issue #36, never from a
configured preference.

The keyframe interval requested from a publisher is derived from the join and
layer-switch behaviour in issue #35.

The number of audio streams a subscriber receives is derived from the room size
and the policy in issue #37.

The media plane listening socket is an operator input rather than a derivation,
because it has to match what the operator opened. Its shape was fixed by issue #4
and the network shape by issue #14, both of which are closed, and what they settled
is in `docs/decisions/media-plane-port.md` and `docs/decisions/network-shape.md`.
This sentence said both were open. That is #178: a document reports tracker state
as of the day it was written, and nothing here re-reads it.

Where data lives, the service name and the credential material are operator inputs
by the categories above and are not derived.

## How this stays true

Every derived value is printed at startup with the input it came from, which is
issue #58. A wrong derivation is then visible in a form an operator can paste into
a report, instead of being a number nobody can see.

The configuration surface is enumerated by a test that reds when the surface
changes without the recorded list changing, which is issue #59. That test also
requires each entry to name one of the four categories above and reds on an entry
that fits none.

A setting that exists only because a derivation was hard is recorded as a debt
carrying the condition that retires it, rather than as a feature.

## When a derivation is wrong on a particular machine

The answer is not to add a setting, because that answer given twice is the end of
the rule.

A derivation that produces a bad value on a real machine is a defect in the
derivation. The machine shape is added to the derivation tests in issue #58 and
the derivation is changed to handle it. The operator's cap is the only lever in
the meantime, and it lowers what the service spends rather than tuning how it
spends it.

Where a derivation genuinely cannot be made to work from what the machine exposes,
the value becomes a setting under the debt rule above, naming the input that was
missing and what would let the derivation come back. That is a deliberate act with
a reviewer and a recorded reason, which is the difference between this and the
thing the rule exists to prevent.
