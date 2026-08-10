// relais, a realtime SFU backend for community self-hosters.
// Copyright (C) 2026 Nils Lehnen
//
// Licensed under the GNU Affero General Public License, version 3. See LICENSE
// for the full terms, including the warranty and liability disclaimer.

package fuzz_test

import (
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/iderex/relais/internal/mediaplane"
	"github.com/iderex/relais/internal/orchestration/credential"
)

// theRoom is the door the fuzzed credential is presented at. One room, because
// the interesting variable is the token and a second room would only halve the
// inputs that reach the checks past the room binding.
const theRoom = mediaplane.RoomID("room-1")

// theSigningSeed fixes the key pair. Thirty-two bytes of text rather than
// something drawn from the machine, so a crash found on one machine reproduces on
// every other one from the same corpus entry. A random key would make the
// signature check pass or fail for a reason no corpus file records.
const theSigningSeed = "relais fuzz signing seed, 32 by."

// theMoment fixes the clock, for the same reason. A credential's window is
// judged against it, so an input that reached the window checks yesterday still
// reaches them today.
var theMoment = time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC)

// theTolerance is the clock skew the verifier is built with. A minute rather than
// zero, so the inputs that exercise the two edges of the window are not all
// refused a step earlier.
const theTolerance = time.Minute

// declared is the closed set of reasons a refusal may carry. A refusal outside it
// is an error path somebody added without a reason a caller can route on, which
// is the shape this fuzz target is looking for as much as a panic.
var declared = map[credential.Reason]bool{
	credential.ReasonMalformed:        true,
	credential.ReasonTooLarge:         true,
	credential.ReasonAlgorithm:        true,
	credential.ReasonSignature:        true,
	credential.ReasonNotYetValid:      true,
	credential.ReasonExpired:          true,
	credential.ReasonWrongRoom:        true,
	credential.ReasonAlreadyUsed:      true,
	credential.ReasonUnpermittedClaim: true,
}

// permitted is the closed set of claims a credential may carry, restated here
// because the package does not export it. A second copy of a closed set is
// usually drift waiting to happen; this one is a guard rather than a document,
// and the drift it produces is the right one. A claim added to the package
// without being added here reds this target, which is what makes widening the
// set a deliberate act somebody sees rather than a field that appears.
//
// docs/decisions/admission.md is where the set is argued and why it is a set of
// names to permit rather than a list of names to refuse.
var permitted = map[string]bool{
	"id":          true,
	"room":        true,
	"participant": true,
	"powers":      true,
	"notBefore":   true,
	"expiresAt":   true,
}

// FuzzACredentialFromAStranger drives the admission decision with bytes nobody
// here wrote.
//
// This is the surface reachable by whoever can reach the service, and it is the
// one that runs before this project knows anything about who is asking. A panic
// in it is a denial of service that needs no credential at all, which is why the
// toolchain's own crash detection is the first of the four properties below and
// the one that needs no assertion.
//
// The other three are properties a parsing defect breaks without crashing:
//
// A refusal carries a declared reason. An error that is not a [credential.Refusal],
// or one carrying a reason outside the set, is a path a caller cannot route on and
// an operator cannot be told about.
//
// A refusal grants nothing. A partially filled grant returned beside an error is
// how a caller that checks the value before the error admits somebody who was
// refused.
//
// An admission is to the room it was presented at, once. The room binding and
// single use are two of the five things the admission record says a join turns
// on, and both are decided after everything a stranger controls has been read.
func FuzzACredentialFromAStranger(f *testing.F) {
	if len(theSigningSeed) != ed25519.SeedSize {
		f.Fatalf("the signing seed is %d bytes and Ed25519 takes %d", len(theSigningSeed), ed25519.SeedSize)
	}
	key := ed25519.NewKeyFromSeed([]byte(theSigningSeed))
	public, ok := key.Public().(ed25519.PublicKey)
	if !ok {
		f.Fatalf("the public half of an Ed25519 key is not an ed25519.PublicKey")
	}

	f.Fuzz(func(t *testing.T, token string) {
		// A verifier per input. The single-use record is state that
		// survives a call, so a shared verifier would make the answer
		// depend on what the fuzzer happened to try first, and a crash
		// found that way would not reproduce from its corpus entry.
		verifier, err := credential.NewVerifier(public, func() time.Time { return theMoment }, theTolerance)
		if err != nil {
			t.Fatalf("building the verifier: %v", err)
		}

		grant, err := verifier.Verify(token, theRoom)
		if err != nil {
			var refusal *credential.Refusal
			if !errors.As(err, &refusal) {
				t.Fatalf("refused with %v, which is not a credential.Refusal, so a caller has nothing to route on", err)
			}
			if !declared[refusal.Reason] {
				t.Fatalf("refused with the reason %q, which is outside the set this package declares", refusal.Reason)
			}
			// Field by field rather than against the zero value,
			// because a grant carries a slice and is therefore not
			// comparable. Every field is named so a grant that grows
			// one leaves this assertion visibly incomplete rather
			// than silently so.
			if grant.ID != "" || grant.Room != "" || grant.Participant != "" ||
				len(grant.Powers) != 0 || !grant.ExpiresAt.IsZero() {
				t.Fatalf("refused as %q and returned the grant %+v, so a caller reading the value before the error admits a stranger",
					refusal.Reason, grant)
			}
			return
		}

		if grant.Room != theRoom {
			t.Fatalf("admitted to %q, and the credential was presented at %q", grant.Room, theRoom)
		}
		if grant.ID == "" || grant.Participant == "" {
			t.Fatalf("admitted with the grant %+v, which names no credential or no participant", grant)
		}

		if _, err := verifier.Verify(token, theRoom); err == nil {
			t.Fatalf("the same token admitted a second session, and a credential admits one")
		}
	})
}

// FuzzTheClaimsAServiceAboveTheSeamSigns drives the same decision with a payload
// that is signed and therefore reaches the checks the signature stands in front
// of.
//
// It exists because the target above cannot reach them. A verifier holds a public
// key, `ed25519.Verify` refuses everything a mutation produces, and no amount of
// fuzzing gets a made-up token past it. So the whole of the claim set, the room
// binding, the window and single use sit behind a wall the first target never
// climbs, and reporting that surface as fuzzed would be reporting the wall.
//
// Signing the fuzzer's bytes with the fixed key is what removes the wall, and the
// case it models is real rather than convenient. docs/decisions/admission.md puts
// identity in the service above the seam and lets this project verify what that
// service signed. A signed credential carrying a person's name is exactly what
// the closed claim set is for, and the issuer is who would put it there, by a
// defect or by a compromise. The bytes here are the issuer's, and this project
// still refuses what it will not carry.
func FuzzTheClaimsAServiceAboveTheSeamSigns(f *testing.F) {
	if len(theSigningSeed) != ed25519.SeedSize {
		f.Fatalf("the signing seed is %d bytes and Ed25519 takes %d", len(theSigningSeed), ed25519.SeedSize)
	}
	key := ed25519.NewKeyFromSeed([]byte(theSigningSeed))
	public, ok := key.Public().(ed25519.PublicKey)
	if !ok {
		f.Fatalf("the public half of an Ed25519 key is not an ed25519.PublicKey")
	}
	// The header segment is taken from a credential this package minted rather
	// than written out here. Its encoding is the package's own, and a second
	// copy of it in this file would drift the day the header gains a field,
	// leaving every input refused before it reached anything and the target
	// still green.
	header := headerSegmentOf(f, key)

	f.Fuzz(func(t *testing.T, claims []byte) {
		payload := base64.RawURLEncoding.EncodeToString(claims)
		signature := ed25519.Sign(key, []byte(header+"."+payload))
		token := header + "." + payload + "." + base64.RawURLEncoding.EncodeToString(signature)

		verifier, err := credential.NewVerifier(public, func() time.Time { return theMoment }, theTolerance)
		if err != nil {
			t.Fatalf("building the verifier: %v", err)
		}

		grant, err := verifier.Verify(token, theRoom)
		if err != nil {
			var refusal *credential.Refusal
			if !errors.As(err, &refusal) {
				t.Fatalf("refused with %v, which is not a credential.Refusal, so a caller has nothing to route on", err)
			}
			if !declared[refusal.Reason] {
				t.Fatalf("refused with the reason %q, which is outside the set this package declares", refusal.Reason)
			}
			if refusal.Reason == credential.ReasonSignature {
				t.Fatalf("refused the signature this target produced, so nothing past it was reached and the target proves nothing")
			}
			if grant.ID != "" || grant.Room != "" || grant.Participant != "" ||
				len(grant.Powers) != 0 || !grant.ExpiresAt.IsZero() {
				t.Fatalf("refused as %q and returned the grant %+v, so a caller reading the value before the error admits a stranger",
					refusal.Reason, grant)
			}
			return
		}

		if grant.Room != theRoom {
			t.Fatalf("admitted to %q, and the credential was presented at %q", grant.Room, theRoom)
		}
		if grant.ID == "" || grant.Participant == "" {
			t.Fatalf("admitted with the grant %+v, which names no credential or no participant", grant)
		}
		if !grant.ExpiresAt.After(theMoment.Add(-theTolerance)) {
			t.Fatalf("admitted on a window that closed at %s, and this host's clock is %s", grant.ExpiresAt, theMoment)
		}

		// The closed claim set, which is the property docs/decisions/admission.md
		// cares most about and the only one this target can check: the bytes
		// that were signed are here, so what the credential carried is
		// readable rather than inferred from what came back. A credential
		// carrying a person's name is admitted or it is not, and the grant
		// says nothing about the claims that were dropped on the way.
		var carried map[string]json.RawMessage
		if err := json.Unmarshal(claims, &carried); err != nil {
			t.Fatalf("admitted claims that are not a JSON object: %v", err)
		}
		for name := range carried {
			if !permitted[name] {
				t.Fatalf("admitted a credential carrying %q, which is outside the set this project permits a credential to carry", name)
			}
		}

		if _, err := verifier.Verify(token, theRoom); err == nil {
			t.Fatalf("the same token admitted a second session, and a credential admits one")
		}
	})
}

// headerSegmentOf mints one credential and returns its header segment, so this
// suite signs what the package itself would have signed.
//
// The claims minted here are thrown away. What is kept is the first of the three
// segments, which is the same for every credential this package produces because
// the header names only the algorithm.
func headerSegmentOf(f *testing.F, key ed25519.PrivateKey) string {
	f.Helper()

	minter, err := credential.NewMinter(key, func() time.Time { return theMoment })
	if err != nil {
		f.Fatalf("building the minter: %v", err)
	}
	token, err := minter.Mint(credential.Claims{
		Room:        theRoom,
		Participant: mediaplane.ParticipantID("participant-1"),
		NotBefore:   theMoment.Add(-time.Hour),
		ExpiresAt:   theMoment.Add(time.Hour),
	})
	if err != nil {
		f.Fatalf("minting the credential this target takes its header from: %v", err)
	}
	header, _, found := strings.Cut(token, ".")
	if !found || header == "" {
		f.Fatalf("the minted credential %q carries no header segment", token)
	}
	return header
}
