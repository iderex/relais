# internal/orchestration/credential

Minting and verifying the join credential. It decides whether a credential is
genuine, current and applicable, and stops there: the participant identifier is
carried and never resolved, and the powers are carried and never read, because
what a power reaches is issue #46 and reading one here would decide authorisation
in the place that decides authenticity. The decision it implements is
[the admission record](../../../docs/decisions/admission.md), and the reason it
sits under `internal/orchestration` rather than under `internal/api` is that a
credential has nothing to do with a wire format: a verifier living beside the
framing would change when the framing did. The public key and the private key
never meet in one type, so a deployment that only verifies has nothing to call
that would issue a credential. It also holds the closed set of claims a credential
may carry and refuses one carrying anything outside it, naming the claim it
refused. What it cannot do is judge the contents of a claim that is permitted,
which is why the record's prohibition on personal data is only half enforced here
and says so.
