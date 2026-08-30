# internal/orchestration

Everything above the port that is not the API surface: room lifetime, admission
against a credential that has already been verified, the event stream, capacity,
the pool, and the reading of the configuration the whole of that runs under. It depends on `internal/mediaplane` and never on
`internal/forwarding`, so the whole of it runs against a fake that carries no
media, which is what makes the orchestration suite something a stock runner can
execute. It holds no packet and no type from the transport library. What it may
and may not own is drawn by [the seam record](../../docs/decisions/seam.md), and
what it may not learn about a person by
[the admission record](../../docs/decisions/admission.md).
