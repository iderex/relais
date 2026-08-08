// Package credential mints and verifies the join credential a service above this
// project hands to a participant.
//
// What it decides is whether a credential is genuine, current and applicable, and
// nothing else. It does not resolve the participant identifier, it does not read
// the powers, and it does not know what a person is. Those are the seam record's
// boundaries and docs/decisions/admission.md is the decision this package
// implements: the three things a credential names, the short window that covers a
// join rather than a session, public key verification keeping the power to mint
// apart from the power to verify, and admission failing closed when the clocks do
// not line up.
//
// It sits under internal/orchestration because admission is orchestration's,
// which docs/architecture.md fixes, and because a credential has nothing to do
// with a wire format: internal/api translates what arrives on the wire, and a
// verifier living there would change when the framing did.
//
// This is not a JSON Web Token and the difference is the point. That format lets
// the token choose the algorithm its own signature is checked with, which is the
// oldest failure of this exact mechanism. A token here names its algorithm only so
// that naming the wrong one can be refused; the verifier never selects on it.
package credential

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/iderex/relais/internal/mediaplane"
)

// Algorithm is the one signature algorithm this project accepts. It is a constant
// rather than a setting: an algorithm that can be configured is an algorithm that
// can be configured downward, and there is no deployment that needs a second one.
const Algorithm = "Ed25519"

// maxTokenBytes bounds a token before anything decodes it. Everything reaching a
// verifier comes from a stranger, so the length is checked against a number rather
// than against available memory. The limits an operator can tune at the edge of the
// API are issue #50; this one is not tunable, because no legitimate credential is
// near it.
const maxTokenBytes = 4096

// Clock is the only way this package learns the time.
//
// It is a parameter rather than a call to time.Now, because an expiry compared
// against a clock nobody injected cannot be driven across its own boundary by a
// test, and an expiry check that has never been driven across its boundary is an
// expiry check nobody has seen work.
type Clock func() time.Time

// Claims is what a caller asks to have minted. It carries the three things the
// admission record says a credential names, and the two moments that bound it.
//
// It does not carry an identifier for the credential. That is minted below, so
// that no caller can hand out two credentials this package cannot tell apart, and
// it is what single use is decided on.
type Claims struct {
	Room        mediaplane.RoomID
	Participant mediaplane.ParticipantID
	// Powers is carried through and never read here. What a power means and
	// which operations it reaches is issue #46, and a package that started
	// interpreting them would be deciding authorisation in the place that
	// decides authenticity.
	Powers    []string
	NotBefore time.Time
	ExpiresAt time.Time
}

// Grant is what a verified credential yields. It is the claims that were signed,
// plus the identifier the single-use check refused a second presentation of.
type Grant struct {
	ID          string
	Room        mediaplane.RoomID
	Participant mediaplane.ParticipantID
	Powers      []string
	ExpiresAt   time.Time
}

// Reason names why a credential was refused. The set is closed and each entry is
// a different thing to tell an operator.
//
// Which reason from docs/api/contract.md's closed error set each of these maps to
// is not decided here. That contract distinguishes credential-invalid from
// clock-skew, the admission record requires a window failure to say the times did
// not line up, and the two do not obviously agree. Issue #145 holds that, and this
// package keeps the distinctions so the mapping has something to map.
type Reason string

// The reasons. Each one is produced in exactly one place below.
const (
	// ReasonMalformed is a token that is not a credential: wrong shape, too
	// long, not decodable, or carrying a field this package does not define.
	ReasonMalformed Reason = "malformed"
	// ReasonAlgorithm is a token naming an algorithm other than [Algorithm].
	ReasonAlgorithm Reason = "algorithm"
	// ReasonSignature is a token whose signature does not hold under the public
	// key this verifier was built with.
	ReasonSignature Reason = "signature"
	// ReasonNotYetValid is a credential whose window has not opened.
	ReasonNotYetValid Reason = "not-yet-valid"
	// ReasonExpired is a credential whose window has closed.
	ReasonExpired Reason = "expired"
	// ReasonWrongRoom is a credential presented at a room it does not name.
	ReasonWrongRoom Reason = "wrong-room"
	// ReasonAlreadyUsed is a credential presented a second time.
	ReasonAlreadyUsed Reason = "already-used"
)

// Refusal is the error every refusal below is reported as. It carries the reason
// as a field rather than only in a sentence, so a caller routing on it never has
// to read prose.
type Refusal struct {
	Reason Reason
	Detail string
}

// Error implements error.
func (r *Refusal) Error() string {
	return fmt.Sprintf("credential refused: %s: %s", r.Reason, r.Detail)
}

// refuse builds a refusal. It is the one place a [Refusal] is constructed, so a
// reason that reaches a caller is one of the constants above.
func refuse(reason Reason, format string, args ...any) *Refusal {
	return &Refusal{Reason: reason, Detail: fmt.Sprintf(format, args...)}
}

// header is the first segment. It names the algorithm and nothing else: a header
// with somewhere to put a key identifier or a critical-extension list is a header
// with somewhere to put an attack.
type header struct {
	Algorithm string `json:"alg"`
}

// payload is the second segment, and it is the whole of what a credential says.
//
// The times are seconds since the epoch. A credential crossing between two hosts
// carries no time zone and no calendar, because neither is a thing two hosts have
// to agree on and both are things they can disagree about.
type payload struct {
	ID          string   `json:"id"`
	Room        string   `json:"room"`
	Participant string   `json:"participant"`
	Powers      []string `json:"powers"`
	NotBefore   int64    `json:"notBefore"`
	ExpiresAt   int64    `json:"expiresAt"`
}

// decodeSegment decodes one base64url segment without padding.
func decodeSegment(name, segment string) ([]byte, error) {
	raw, err := base64.RawURLEncoding.DecodeString(segment)
	if err != nil {
		return nil, refuse(ReasonMalformed, "the %s segment is not base64url", name)
	}
	return raw, nil
}

// decodeJSON decodes one segment into v, refusing any field v does not define.
//
// Refusing an unknown field is what stops a credential carrying a display name
// from parsing at all. It is not the closed set of permitted claims the admission
// record owes and issue #124 holds: that one is a declared set with its own
// refusal and its own evidence, and what happens here is only that a field nobody
// defined does not decode.
func decodeJSON(name, segment string, v any) error {
	raw, err := decodeSegment(name, segment)
	if err != nil {
		return err
	}
	dec := json.NewDecoder(strings.NewReader(string(raw)))
	dec.DisallowUnknownFields()
	if err := dec.Decode(v); err != nil {
		return refuse(ReasonMalformed, "the %s segment is not a credential: %v", name, err)
	}
	return nil
}

// signingInput is the bytes a signature covers: the two encoded segments and the
// separator between them.
//
// The encoded forms are signed rather than the values they decode to, so that two
// hosts never have to agree on how a structure is serialised. A signature over a
// re-serialised structure is a signature over whatever the verifier's own encoder
// produced, which is not what the minter signed.
func signingInput(headerSegment, payloadSegment string) []byte {
	return []byte(headerSegment + "." + payloadSegment)
}
