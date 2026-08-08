package credential

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"time"
)

// idBytes is how much randomness a credential's identifier carries. It is not a
// secret and it is not guessed at: it only has to be unique among the credentials
// alive inside one validity window, and this is far past what that needs.
const idBytes = 16

// Minter produces credentials. It is a separate type from [Verifier], holding a
// separate key, reached through a separate constructor, and that separation is the
// point rather than a structure: a host that can check a credential must not be
// able to issue one, or a compromised media host becomes an admission authority
// for every room it can name.
type Minter struct {
	key ed25519.PrivateKey
	now Clock
}

// NewMinter builds a minter from the private key of the service that issues.
//
// A deployment that only verifies has no private key to give it and cannot build
// one of these, which is what makes the separation a property of the deployment
// rather than a habit. Handing it a public key does not work either: an Ed25519
// public key is not the size of a private one and this refuses it.
func NewMinter(key ed25519.PrivateKey, now Clock) (*Minter, error) {
	if len(key) != ed25519.PrivateKeySize {
		return nil, errors.New("credential: a minter needs an Ed25519 private key")
	}
	if now == nil {
		return nil, errors.New("credential: a minter needs a clock")
	}
	return &Minter{key: key, now: now}, nil
}

// Mint signs a credential for the claims given and returns it as a token.
//
// The identifier is minted here rather than taken from the caller. It is what the
// single-use check refuses a second presentation of, so a caller able to choose it
// would be a caller able to hand out two credentials a verifier cannot tell apart,
// or to reuse one deliberately.
func (m *Minter) Mint(c Claims) (string, error) {
	if c.Room == "" || c.Participant == "" {
		return "", errors.New("credential: a credential names a room and a participant")
	}
	if !c.ExpiresAt.After(c.NotBefore) {
		return "", errors.New("credential: the window closes at or before it opens")
	}

	id, err := newID()
	if err != nil {
		return "", err
	}

	headerSegment, err := encodeSegment(header{Algorithm: Algorithm})
	if err != nil {
		return "", err
	}
	payloadSegment, err := encodeSegment(payload{
		ID:          id,
		Room:        string(c.Room),
		Participant: string(c.Participant),
		Powers:      c.Powers,
		NotBefore:   c.NotBefore.Unix(),
		ExpiresAt:   c.ExpiresAt.Unix(),
	})
	if err != nil {
		return "", err
	}

	signature := ed25519.Sign(m.key, signingInput(headerSegment, payloadSegment))
	return headerSegment + "." + payloadSegment + "." + base64.RawURLEncoding.EncodeToString(signature), nil
}

// VerifyingKey is the public half of this minter's key, which is what a host that
// only verifies is given.
func (m *Minter) VerifyingKey() ed25519.PublicKey {
	return m.key.Public().(ed25519.PublicKey)
}

// newID produces a credential identifier.
func newID() (string, error) {
	raw := make([]byte, idBytes)
	if _, err := rand.Read(raw); err != nil {
		return "", fmt.Errorf("credential: no randomness for an identifier: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(raw), nil
}

// encodeSegment serialises one segment.
func encodeSegment(v any) (string, error) {
	raw, err := json.Marshal(v)
	if err != nil {
		return "", fmt.Errorf("credential: a segment did not serialise: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(raw), nil
}

// Now returns this minter's view of the time, so a caller building claims uses the
// same clock the minter was given rather than reaching for its own.
func (m *Minter) Now() time.Time {
	return m.now()
}
