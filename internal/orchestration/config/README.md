# internal/orchestration/config

Reading what an operator supplied, validating the whole of it, and either
returning a configuration or refusing to produce one. The decision it implements
is [the configuration record](../../../docs/decisions/configuration.md), which
makes derivation the default and configuration the exception and closes the
exception over four categories; every setting here names the category that admits
it, so a fifth category is a change to that record rather than a field somebody
added. It sits under `internal/orchestration` rather than beside the entry point
because it is above the port and is not the API surface, and because a reading
that lived beside the wiring would be re-implemented by the second thing that
needed it.

## The sources, and which one wins

There are two, and only one of them is the operator.

The **process environment** is the whole of what an operator supplies. Nothing
here reads a file, and that is the deployment contract's clause rather than a
preference: the promise is that no file is edited by hand before the one command
runs, and a reading that accepted a configuration file would make that promise
false the first time somebody wrote one. A second source is a change to
[the deployment contract](../../../docs/decisions/deployment-contract.md) and to
this paragraph, in that order.

**Derivation** is the other, and it is the service working a value out rather
than an operator stating it. Two of the four settings have one; the other two
have none, because no reading of a machine produces the name somebody pointed at
it or the key a service above the seam is holding.

The precedence between them is one sentence. **A value the operator supplied is
used as supplied and is never replaced by a derivation.** A value the operator
did not supply is derived where the record permits a derivation and refused where
it does not.

## What a supplied value that cannot be used does

It is refused. It does not fall back to the derivation, and this is the half of
the precedence rule that costs something, so it is written separately from the
sentence above rather than as a clause inside it.

Saying nothing leaves a decision to the service. Saying something unusable is a
decision that did not arrive, and treating the second as the first is how a
service starts in a state nobody intended: a missing credential that admits
everyone, an unwritable data path that becomes memory, a cap that does not parse
and becomes no cap at all. Those three are the fall backs issue #79 names, and
each one has a test beside a near miss where the same setting is absent instead
of unusable and the answer is the derivation.

## What it refuses beyond a bad value

A variable carrying the service's prefix that names no setting stops startup. A
name that is silently ignored is how an operator comes to believe a limit is in
force that nothing ever read, so an unknown setting is not a warning here.

The cost of that rule is that the prefix belongs to the service and to nothing
else. A tool that sets a prefixed variable in a process which then starts this
service stops it. This repository already carries one such variable, in
`.github/workflows/coverage.yml`, where a suite rather than the service reads it,
so it is a near miss rather than a collision — and it is written down because the
next one will be written by somebody who did not know the prefix was claimed.

Every problem in one reading is collected and reported together. A configuration
read one problem at a time is a configuration an operator repairs one restart at
a time, and the reading knows all of them already.

## What it does not do, and what nothing here has confirmed

Nothing in this tree calls it yet. `cmd/relais` reports what the build knows
about itself and exits, so no route reaches this package outside its own suite,
and the sentence about validating "before anything starts" is a property of the
reading rather than an observation of a service starting. What turns it into one
is the wiring in issue #55 and the derived-value printing in issue #58.

It settles no number for the cap beyond the range a share of one machine can be.
What a cap of sixty percent means in workers, queues and ceilings is derived from
it elsewhere, and the resource budget those derivations rest on is issue #10.

It does not enumerate itself against a recorded list. `Surface` is that list, and
the test that reds when it changes without the record changing is issue #59.

The clock tolerance the credential verifier is built with is named as
configuration by `internal/orchestration/credential` and is not on this surface,
because it fits none of the four categories the configuration record closes over.
That is issue #185.
