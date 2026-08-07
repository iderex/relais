# internal/api

The operator API and the signalling surface: the transport, the framing, the
versioning and the translation between what arrives on the wire and what
`internal/orchestration` understands. It is the only package that knows the wire
format exists, so a change to the framing reaches no further than this directory.
It depends on `internal/orchestration` and on `internal/mediaplane` for the
identifiers it carries, and never on `internal/forwarding`, because an API that
can reach the forwarding core will eventually expose it. The contract it
implements is written before the code in issue #43 and is checked against
[the seam record](../../docs/decisions/seam.md).
