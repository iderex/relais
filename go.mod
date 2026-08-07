module github.com/iderex/relais

// The floor is the transport library's own requirement rather than a preference.
// pion/webrtc v4 declares `go 1.24.0`, and a floor below that could not build
// against the library the forwarding decision names.
go 1.24.0
