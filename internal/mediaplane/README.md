# internal/mediaplane

The port. This package declares the interface between everything that
orchestrates and everything that holds a packet, together with the identifiers
and values that cross it, and it contains no implementation of either side. It
depends on nothing in this repository and nothing from the transport library: a
type from that library appearing here is the failure this whole layout exists to
prevent, because the moment one crosses, replacing the forwarding core means
changing every caller instead of one package. What the interface says is fixed by
[the media plane port record](../../docs/decisions/media-plane-port.md) rather
than by whatever the first implementation needed, and the dependency directions
that keep it honest are in [docs/architecture.md](../../docs/architecture.md).
