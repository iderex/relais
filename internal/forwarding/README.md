# internal/forwarding

The forwarding core, and the only place in this repository that holds a packet.
It implements the port declared in `internal/mediaplane`, it is the only package
permitted to import the transport library, and it decides which stream a
subscriber receives, which layer, when a keyframe is asked for and what is given
up first when there is not enough bandwidth. It knows nothing about who a
participant is beyond the opaque identifier the port hands it, and it imports
nothing from `internal/orchestration` or `internal/api`, because a forwarding
loop that can reach upward is one that will. What it takes from the transport
library and what it owns is argued in
[the forwarding core record](../../docs/decisions/forwarding-core.md).
