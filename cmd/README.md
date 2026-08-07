# cmd

Every executable this repository produces, one directory per binary, and nothing
else. A package here wires the parts under `internal/` together, reads the
configuration, starts the service and stops it. It holds no rule of its own:
anything worth a test belongs in a package under `internal/`, because a test that
has to build a binary and run it is a test nobody writes twice. Nothing imports
a package under `cmd/`, which is what lets it depend on everything else without
creating a cycle. The layout and the dependency directions are argued in
[docs/architecture.md](../docs/architecture.md).
