// relais, a realtime SFU backend for community self-hosters.
// Copyright (C) 2026 Nils Lehnen
//
// Licensed under the GNU Affero General Public License, version 3. See LICENSE
// for the full terms, including the warranty and liability disclaimer.

package credential

import (
	"crypto/ed25519"
	"errors"
	"strings"
	"sync"
	"time"

	"github.com/iderex/relais/internal/mediaplane"
)

// Verifier decides whether a credential is genuine, current and applicable.
//
// It holds a public key and no private key, and it has no method that produces a
// credential. That is the separation the admission record asks for, expressed so
// that a deployment which only verifies cannot mint by mistake: there is nothing
// on this type to call.
type Verifier struct {
	key   ed25519.PublicKey
	now   Clock
	skew  time.Duration
	mu    sync.Mutex
	spent map[string]time.Time
}

// NewVerifier builds a verifier from the public key of the service that mints.
//
// skew is the tolerance between this host's clock and the issuing service's, which
// the admission record makes part of the join contract. No number is chosen here.
// Zero is legal and means the two clocks are required to agree exactly, and what
// an operator runs is configuration, which is issue #79. Whether this host's clock
// is actually inside the tolerance is the startup self-check in issue #81, and
// nothing in this package can tell.
func NewVerifier(key ed25519.PublicKey, now Clock, skew time.Duration) (*Verifier, error) {
	if len(key) != ed25519.PublicKeySize {
		return nil, errors.New("credential: a verifier needs an Ed25519 public key")
	}
	if now == nil {
		return nil, errors.New("credential: a verifier needs a clock")
	}
	if skew < 0 {
		return nil, errors.New("credential: a negative clock tolerance is not a tolerance")
	}
	return &Verifier{
		key:   key,
		now:   now,
		skew:  skew,
		spent: make(map[string]time.Time),
	}, nil
}

// Verify checks a credential against the room it was presented at and returns what
// it grants.
//
// Every check below refuses on its own, and each one is a way this mechanism is
// known to be got wrong rather than a way it might be. The order matters in one
// place only: nothing is recorded as spent until everything else has held, so a
// token that fails any check can be presented again once whatever was wrong has
// been repaired.
func (v *Verifier) Verify(token string, room mediaplane.RoomID) (Grant, error) {
	if len(token) > maxTokenBytes {
		return Grant{}, refuse(ReasonTooLarge, "the token is %d bytes, over the %d this project reads", len(token), maxTokenBytes)
	}

	headerSegment, payloadSegment, signatureSegment, err := split(token)
	if err != nil {
		return Grant{}, err
	}

	var h header
	if err := decodeJSON("header", headerSegment, &h); err != nil {
		return Grant{}, err
	}
	// The algorithm is fixed by this service. The token names one so that naming
	// the wrong one can be refused, and the check below is the whole of what the
	// name is used for: the signature is verified with Ed25519 either way, so a
	// token claiming none, or claiming a symmetric algorithm whose key is the
	// public one, never reaches a verifier chosen by its own bytes.
	if h.Algorithm != Algorithm {
		return Grant{}, refuse(ReasonAlgorithm, "the token names %q and this project verifies %q", h.Algorithm, Algorithm)
	}

	signature, err := decodeSegment("signature", signatureSegment)
	if err != nil {
		return Grant{}, err
	}
	if !ed25519.Verify(v.key, signingInput(headerSegment, payloadSegment), signature) {
		return Grant{}, refuse(ReasonSignature, "the signature does not hold under this host's verifying key")
	}

	// The claim set is read after the signature holds, so what it judges is what
	// the issuing service actually put in a credential rather than what a stranger
	// asked this host to look at.
	p, err := decodePayload(payloadSegment)
	if err != nil {
		return Grant{}, err
	}
	if p.ID == "" || p.Room == "" || p.Participant == "" {
		return Grant{}, refuse(ReasonMalformed, "the credential names no identifier, no room or no participant")
	}

	// The room binding. A credential is a key to one door, and a verifier that
	// reads the room out of the token and reports it is a verifier that admits a
	// participant to whichever room it names.
	if mediaplane.RoomID(p.Room) != room {
		return Grant{}, refuse(ReasonWrongRoom, "the credential names a room other than the one it was presented at")
	}

	now := v.now()
	notBefore := time.Unix(p.NotBefore, 0)
	expiresAt := time.Unix(p.ExpiresAt, 0)
	if now.Before(notBefore.Add(-v.skew)) {
		return Grant{}, refuse(ReasonNotYetValid, "this host's clock is before the credential's window opens")
	}
	if !now.Before(expiresAt.Add(v.skew)) {
		return Grant{}, refuse(ReasonExpired, "this host's clock is at or past the moment the credential's window closes")
	}

	if err := v.spend(p.ID, expiresAt.Add(v.skew), now); err != nil {
		return Grant{}, err
	}

	return Grant{
		ID:          p.ID,
		Room:        mediaplane.RoomID(p.Room),
		Participant: mediaplane.ParticipantID(p.Participant),
		Powers:      append([]string(nil), p.Powers...),
		ExpiresAt:   expiresAt,
	}, nil
}

// spend records a credential as used and refuses a second presentation of it.
//
// A forwarding session outlives the credential that opened it, which the admission
// record settles, so a credential replayed after the session it was minted for
// ended would otherwise open a second one. The record is what makes it single use.
//
// What is held is bounded by the credential's own window: an entry is dropped once
// this host's clock is past the moment the credential could no longer be accepted
// anyway, so the set holds at most the credentials minted inside one window and
// cannot grow with the age of the process.
func (v *Verifier) spend(id string, until, now time.Time) error {
	v.mu.Lock()
	defer v.mu.Unlock()

	for spentID, deadline := range v.spent {
		if !deadline.After(now) {
			delete(v.spent, spentID)
		}
	}
	if _, used := v.spent[id]; used {
		return refuse(ReasonAlreadyUsed, "this credential has already admitted a session")
	}
	v.spent[id] = until
	return nil
}

// split cuts a token into its three segments.
//
// It refuses anything that is not exactly three, rather than reading the first
// three of more, because a token with a fourth segment is a token somebody built
// against a different reader.
func split(token string) (headerSegment, payloadSegment, signatureSegment string, err error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return "", "", "", refuse(ReasonMalformed, "the token has %d segments and a credential has 3", len(parts))
	}
	return parts[0], parts[1], parts[2], nil
}
