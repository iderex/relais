# gate

The verb a contributor runs before pushing, and nothing else.

```
go run ./gate
```

It runs the legs whose result is the same on every machine, in order, stopping at
the first failure: the build, the suite and the style gate. Then it names the
gates it did not run, derived from `.github/workflows/` rather than from a list
kept in its source, so a workflow added tomorrow turns up in that section without
anybody remembering this directory.

What belongs here is a rule about the repository that a contributor needs before
a pull request exists. What does not is anything the service does: nothing under
`cmd/` or `internal/` may be reached from here, and nothing there may import
this. The gate is not part of the program it judges, which is the same position
`bench/` holds for the same reason.

It is a Go program because the three rules this project is held to need one. A
refusable property needs somewhere for a check to live, a guard needs a test that
shows it biting, and a claim needs a command behind it; a shell script or a make
target would carry the same logic in a language this tree has no suite and no
linter for. The accounting and the two version readings are ordinary functions
with ordinary tests beside them, which is the whole argument for the means.

What it does not do is imitate the runner. The headless assertions in
`.github/workflows/test.yml` read the display variables, the audio device nodes
and `/proc` of a Linux runner; running them on a machine somebody works at would
refuse that machine for having a display. They are named as not run rather than
approximated, and the pull request page stays the authority for what ran on the
server.

The layout and the directions its dependencies run are argued in
[docs/architecture.md](../docs/architecture.md).
