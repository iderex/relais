package credential

import (
	"crypto/ed25519"
	"encoding/base64"
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/iderex/relais/internal/mediaplane"
)

// Every test below is a failure this mechanism is known to be got wrong by, and
// each one is written so that it passes on an implementation missing exactly one
// check. Deleting the check named in the comment is what turns that test red, and
// nothing else in this file goes with it.
//
// The keys are built from fixed seeds rather than generated. Nothing here asserts
// anything about a key's value, and a suite that produces different bytes on every
// run is a suite whose failures cannot be reproduced from what it printed.

var (
	issuerSeed   = bytes32(1)
	strangerSeed = bytes32(2)
)

// bytes32 fills a seed with one repeated byte.
func bytes32(b byte) []byte {
	seed := make([]byte, ed25519.SeedSize)
	for i := range seed {
		seed[i] = b
	}
	return seed
}

// theMoment is the instant every test's clock is set relative to. Any instant
// would do; a fixed one keeps a failure readable.
var theMoment = time.Date(2026, time.March, 1, 12, 0, 0, 0, time.UTC)

// at returns a clock stopped at t.
func at(t time.Time) Clock {
	return func() time.Time { return t }
}

// issuer builds the minting side.
func issuer(t *testing.T) *Minter {
	t.Helper()
	m, err := NewMinter(ed25519.NewKeyFromSeed(issuerSeed), at(theMoment))
	if err != nil {
		t.Fatalf("building a minter: %v", err)
	}
	return m
}

// verifierAt builds the verifying side against the issuer's public key, with its
// clock stopped at now and no tolerance for a disagreeing clock.
func verifierAt(t *testing.T, now time.Time) *Verifier {
	t.Helper()
	v, err := NewVerifier(issuer(t).VerifyingKey(), at(now), 0)
	if err != nil {
		t.Fatalf("building a verifier: %v", err)
	}
	return v
}

// aJoin is the credential the rest of the file starts from: valid from theMoment
// for a minute, for one participant in one room.
func aJoin() Claims {
	return Claims{
		Room:        "room-1",
		Participant: "opaque-participant",
		Powers:      []string{"publish", "subscribe"},
		NotBefore:   theMoment,
		ExpiresAt:   theMoment.Add(time.Minute),
	}
}

// mint signs aJoin, or whatever claims are handed in.
func mint(t *testing.T, c Claims) string {
	t.Helper()
	token, err := issuer(t).Mint(c)
	if err != nil {
		t.Fatalf("minting: %v", err)
	}
	return token
}

// refusalFor runs a verification that is expected to fail and returns the reason.
func refusalFor(t *testing.T, v *Verifier, token string, room mediaplane.RoomID) Reason {
	t.Helper()
	if _, err := v.Verify(token, room); err != nil {
		var r *Refusal
		if !errors.As(err, &r) {
			t.Fatalf("refused with %v, which is not a Refusal", err)
		}
		return r.Reason
	}
	t.Fatal("the credential was accepted and the test exists because it must not be")
	return ""
}

// forge assembles a token out of raw header and payload JSON, signed with the key
// given. It is what lets a test present bytes no minter would produce.
func forge(headerJSON, payloadJSON string, key ed25519.PrivateKey) string {
	h := base64.RawURLEncoding.EncodeToString([]byte(headerJSON))
	p := base64.RawURLEncoding.EncodeToString([]byte(payloadJSON))
	sig := ed25519.Sign(key, signingInput(h, p))
	return h + "." + p + "." + base64.RawURLEncoding.EncodeToString(sig)
}

// TestAMintedCredentialVerifies is the case every other test is a departure from.
// It also fixes that the powers are carried through untouched, because a package
// that started reading them would be deciding authorisation here.
func TestAMintedCredentialVerifies(t *testing.T) {
	v := verifierAt(t, theMoment)

	grant, err := v.Verify(mint(t, aJoin()), "room-1")
	if err != nil {
		t.Fatalf("a credential minted for this room and this moment was refused: %v", err)
	}
	if grant.Room != "room-1" || grant.Participant != "opaque-participant" {
		t.Errorf("the grant names %q in %q, want opaque-participant in room-1", grant.Participant, grant.Room)
	}
	if !reflect.DeepEqual(grant.Powers, []string{"publish", "subscribe"}) {
		t.Errorf("the grant carries powers %v, want the two that were minted", grant.Powers)
	}
	if grant.ID == "" {
		t.Error("the grant carries no identifier, so nothing can refuse a second use of it")
	}
}

// TestACredentialSignedByAStrangerIsRefused covers the signature check. Without
// it every token verifies, including one anybody can write.
func TestACredentialSignedByAStrangerIsRefused(t *testing.T) {
	stranger, err := NewMinter(ed25519.NewKeyFromSeed(strangerSeed), at(theMoment))
	if err != nil {
		t.Fatalf("building the stranger's minter: %v", err)
	}
	token, err := stranger.Mint(aJoin())
	if err != nil {
		t.Fatalf("minting: %v", err)
	}

	if got := refusalFor(t, verifierAt(t, theMoment), token, "room-1"); got != ReasonSignature {
		t.Errorf("a credential from another key was refused as %q, want %q", got, ReasonSignature)
	}
}

// TestATamperedPayloadIsRefused is the same check reached from the other side: the
// signature covers the encoded segments, so editing one after signing breaks it.
func TestATamperedPayloadIsRefused(t *testing.T) {
	token := mint(t, aJoin())
	parts := strings.Split(token, ".")
	forged := forge(`{"alg":"Ed25519"}`, `{"id":"x","room":"room-1","participant":"someone-else","powers":[],"notBefore":0,"expiresAt":4102444800}`, ed25519.NewKeyFromSeed(strangerSeed))
	parts[1] = strings.Split(forged, ".")[1]

	if got := refusalFor(t, verifierAt(t, theMoment), strings.Join(parts, "."), "room-1"); got != ReasonSignature {
		t.Errorf("a credential whose payload was swapped was refused as %q, want %q", got, ReasonSignature)
	}
}

// TestATokenNamingAnotherAlgorithmIsRefused covers the check that the algorithm is
// fixed by this service. The token below carries a signature that holds under the
// issuer's key, so an implementation that only checked the signature would accept
// it and an implementation that chose its verifier from the token would be told
// which one to use by a stranger.
func TestATokenNamingAnotherAlgorithmIsRefused(t *testing.T) {
	payloadJSON := `{"id":"x","room":"room-1","participant":"p","powers":[],"notBefore":1772366400,"expiresAt":1772366460}`

	for _, alg := range []string{"HS256", "none", "ED25519", ""} {
		token := forge(`{"alg":"`+alg+`"}`, payloadJSON, ed25519.NewKeyFromSeed(issuerSeed))
		if got := refusalFor(t, verifierAt(t, theMoment), token, "room-1"); got != ReasonAlgorithm {
			t.Errorf("a validly signed token naming %q was refused as %q, want %q", alg, got, ReasonAlgorithm)
		}
	}
}

// TestACredentialForAnotherRoomIsRefused covers the room binding. Without it a
// credential minted for any room admits its holder to every room, which is the
// failure that turns one leaked credential into access to the whole host.
func TestACredentialForAnotherRoomIsRefused(t *testing.T) {
	token := mint(t, aJoin())

	if got := refusalFor(t, verifierAt(t, theMoment), token, "room-2"); got != ReasonWrongRoom {
		t.Errorf("a credential for room-1 presented at room-2 was refused as %q, want %q", got, ReasonWrongRoom)
	}
}

// TestACredentialAdmitsOneSessionAndNoMore covers single use. The session outlives
// the credential deliberately, so without this a credential replayed after its
// session ended opens a second one, inside its own validity window, with no
// signature to break and nothing to notice.
func TestACredentialAdmitsOneSessionAndNoMore(t *testing.T) {
	v := verifierAt(t, theMoment)
	token := mint(t, aJoin())

	if _, err := v.Verify(token, "room-1"); err != nil {
		t.Fatalf("the first presentation was refused: %v", err)
	}
	if got := refusalFor(t, v, token, "room-1"); got != ReasonAlreadyUsed {
		t.Errorf("a second presentation was refused as %q, want %q", got, ReasonAlreadyUsed)
	}
}

// TestTwoCredentialsForOneParticipantAreBothAccepted is the near miss beside the
// test above. Keying single use on anything the caller controls, the participant
// or the room, would refuse the second credential of a participant who legitimately
// rejoins, and the suite would still look green without this.
func TestTwoCredentialsForOneParticipantAreBothAccepted(t *testing.T) {
	v := verifierAt(t, theMoment)

	first, second := mint(t, aJoin()), mint(t, aJoin())
	if first == second {
		t.Fatal("two mints produced the same token, so single use cannot tell them apart")
	}
	if _, err := v.Verify(first, "room-1"); err != nil {
		t.Fatalf("the first credential was refused: %v", err)
	}
	if _, err := v.Verify(second, "room-1"); err != nil {
		t.Fatalf("a second credential for the same participant was refused: %v", err)
	}
}

// TestAFailedVerificationDoesNotSpendTheCredential is the other near miss. A
// verifier that recorded the credential before the rest of the checks held would
// turn one mistyped room into a credential nobody can use again.
func TestAFailedVerificationDoesNotSpendTheCredential(t *testing.T) {
	v := verifierAt(t, theMoment)
	token := mint(t, aJoin())

	if got := refusalFor(t, v, token, "room-2"); got != ReasonWrongRoom {
		t.Fatalf("presenting at the wrong room was refused as %q, want %q", got, ReasonWrongRoom)
	}
	if _, err := v.Verify(token, "room-1"); err != nil {
		t.Errorf("the credential was refused at its own room after a failed attempt elsewhere: %v", err)
	}
}

// TestTheWindowIsDrivenAcrossBothOfItsBoundaries covers expiry against the
// injected clock, in both directions. Without the clock being a parameter none of
// these four moments can be reached at all, and an expiry check nobody has driven
// across its boundary is one nobody has seen work.
func TestTheWindowIsDrivenAcrossBothOfItsBoundaries(t *testing.T) {
	opens := theMoment
	closes := theMoment.Add(time.Minute)

	cases := []struct {
		what string
		now  time.Time
		want Reason // empty means the credential is accepted
	}{
		{"a second before the window opens", opens.Add(-time.Second), ReasonNotYetValid},
		{"the moment the window opens", opens, ""},
		{"a second before the window closes", closes.Add(-time.Second), ""},
		{"the moment the window closes", closes, ReasonExpired},
		{"an hour after the window closes", closes.Add(time.Hour), ReasonExpired},
	}

	for _, c := range cases {
		t.Run(c.what, func(t *testing.T) {
			v := verifierAt(t, c.now)
			token := mint(t, aJoin())

			if c.want == "" {
				if _, err := v.Verify(token, "room-1"); err != nil {
					t.Fatalf("refused inside its own window: %v", err)
				}
				return
			}
			if got := refusalFor(t, v, token, "room-1"); got != c.want {
				t.Errorf("refused as %q, want %q", got, c.want)
			}
		})
	}
}

// TestTheToleranceWidensTheWindowInBothDirections covers the clock tolerance the
// admission record makes part of the join contract. Without it a host whose clock
// is a second out refuses every credential, and the refusal says the times did not
// line up rather than reporting a bad credential either way.
func TestTheToleranceWidensTheWindowInBothDirections(t *testing.T) {
	opens := theMoment
	closes := theMoment.Add(time.Minute)

	for _, now := range []time.Time{opens.Add(-5 * time.Second), closes.Add(5 * time.Second)} {
		v, err := NewVerifier(issuer(t).VerifyingKey(), at(now), 10*time.Second)
		if err != nil {
			t.Fatalf("building a verifier: %v", err)
		}
		if _, err := v.Verify(mint(t, aJoin()), "room-1"); err != nil {
			t.Errorf("a clock %v out of step with a 10s tolerance refused the credential: %v", now.Sub(opens), err)
		}
	}
}

// TestAVerifyOnlyConfigurationCannotMint covers the separation of the two powers.
// A host that only verifies holds the public key and nothing else, and this asserts
// both halves of what that has to mean: the public key cannot be used to build a
// minter, and the verifier itself carries no way to produce a credential.
func TestAVerifyOnlyConfigurationCannotMint(t *testing.T) {
	public := issuer(t).VerifyingKey()

	if _, err := NewMinter(ed25519.PrivateKey(public), at(theMoment)); err == nil {
		t.Error("a minter was built out of a public key")
	}
	if _, err := NewMinter(nil, at(theMoment)); err == nil {
		t.Error("a minter was built out of no key at all")
	}

	verifierType := reflect.TypeOf(verifierAt(t, theMoment))
	for i := range verifierType.NumMethod() {
		if name := verifierType.Method(i).Name; strings.Contains(strings.ToLower(name), "mint") {
			t.Errorf("a verifier carries the method %q, so verifying reaches minting", name)
		}
	}
}

// TestACredentialCarryingAFieldNobodyDefinedDoesNotParse is the floor under the
// prohibition on personal data. It is not the closed set of permitted claims that
// issue #124 owes: this only says an unlisted field fails to decode.
func TestACredentialCarryingAFieldNobodyDefinedDoesNotParse(t *testing.T) {
	token := forge(
		`{"alg":"Ed25519"}`,
		`{"id":"x","room":"room-1","participant":"p","powers":[],"notBefore":1772366400,"expiresAt":1772366460,"displayName":"a person"}`,
		ed25519.NewKeyFromSeed(issuerSeed),
	)

	if got := refusalFor(t, verifierAt(t, theMoment), token, "room-1"); got != ReasonMalformed {
		t.Errorf("a credential carrying a display name was refused as %q, want %q", got, ReasonMalformed)
	}
}

// TestBytesThatAreNotACredentialAreRefused covers the parse boundary. Everything
// reaching a verifier comes from a stranger, so each of these is a shape somebody
// will send.
func TestBytesThatAreNotACredentialAreRefused(t *testing.T) {
	valid := mint(t, aJoin())

	cases := map[string]string{
		"nothing at all":                 "",
		"two segments":                   strings.Join(strings.Split(valid, ".")[:2], "."),
		"four segments":                  valid + ".extra",
		"a header that is not base64url": "not!base64." + strings.Join(strings.Split(valid, ".")[1:], "."),
		"a header that is not JSON":      forge(`{`, `{"id":"x","room":"room-1","participant":"p","powers":[],"notBefore":0,"expiresAt":4102444800}`, ed25519.NewKeyFromSeed(issuerSeed)),
	}

	for what, token := range cases {
		if got := refusalFor(t, verifierAt(t, theMoment), token, "room-1"); got != ReasonMalformed {
			t.Errorf("%s was refused as %q, want %q", what, got, ReasonMalformed)
		}
	}
}

// TestATokenPastTheSizeThisProjectReadsIsRefusedOnItsLength covers the bound in
// front of every decode. Deleting the bound leaves the token refused anyway, by
// whichever check it fails next, so what this asserts is the reason: a guard whose
// removal changes nothing a test can see is a guard that can be removed.
func TestATokenPastTheSizeThisProjectReadsIsRefusedOnItsLength(t *testing.T) {
	token := strings.Repeat("a", maxTokenBytes+1)

	if got := refusalFor(t, verifierAt(t, theMoment), token, "room-1"); got != ReasonTooLarge {
		t.Errorf("a token of %d bytes was refused as %q, want %q", len(token), got, ReasonTooLarge)
	}
}

// TestAVerifierRefusesToBeBuiltWrong covers the constructor. A verifier holding a
// key of the wrong length, no clock, or a tolerance that runs backwards is one
// whose refusals mean nothing, and it fails closed at the moment it is built rather
// than at the first join.
func TestAVerifierRefusesToBeBuiltWrong(t *testing.T) {
	good := issuer(t).VerifyingKey()

	if _, err := NewVerifier(nil, at(theMoment), 0); err == nil {
		t.Error("a verifier was built with no key")
	}
	if _, err := NewVerifier(good[:16], at(theMoment), 0); err == nil {
		t.Error("a verifier was built with half a key")
	}
	if _, err := NewVerifier(good, nil, 0); err == nil {
		t.Error("a verifier was built with no clock")
	}
	if _, err := NewVerifier(good, at(theMoment), -time.Second); err == nil {
		t.Error("a verifier was built with a negative clock tolerance")
	}
}

// TestAMinterRefusesClaimsItCannotSign covers the minting side's own floor.
func TestAMinterRefusesClaimsItCannotSign(t *testing.T) {
	m := issuer(t)

	cases := map[string]Claims{
		"no room":            {Participant: "p", NotBefore: theMoment, ExpiresAt: theMoment.Add(time.Minute)},
		"no participant":     {Room: "room-1", NotBefore: theMoment, ExpiresAt: theMoment.Add(time.Minute)},
		"a window of zero":   {Room: "room-1", Participant: "p", NotBefore: theMoment, ExpiresAt: theMoment},
		"a backwards window": {Room: "room-1", Participant: "p", NotBefore: theMoment, ExpiresAt: theMoment.Add(-time.Minute)},
	}

	for what, c := range cases {
		if _, err := m.Mint(c); err == nil {
			t.Errorf("%s was minted", what)
		}
	}
}

// TestSpentCredentialsAreDroppedOnceTheyCouldNoLongerBeUsed covers the bound on
// what the single-use record holds. Without it the set grows with the age of the
// process, which is the shape of every slow memory failure in a service that runs
// for months.
func TestSpentCredentialsAreDroppedOnceTheyCouldNoLongerBeUsed(t *testing.T) {
	now := theMoment
	v, err := NewVerifier(issuer(t).VerifyingKey(), func() time.Time { return now }, 0)
	if err != nil {
		t.Fatalf("building a verifier: %v", err)
	}

	if _, err := v.Verify(mint(t, aJoin()), "room-1"); err != nil {
		t.Fatalf("the first credential was refused: %v", err)
	}
	if len(v.spent) != 1 {
		t.Fatalf("the verifier holds %d spent credentials after one join, want 1", len(v.spent))
	}

	now = theMoment.Add(time.Hour)
	later := aJoin()
	later.NotBefore, later.ExpiresAt = now, now.Add(time.Minute)
	if _, err := v.Verify(mint(t, later), "room-1"); err != nil {
		t.Fatalf("a later credential was refused: %v", err)
	}
	if len(v.spent) != 1 {
		t.Errorf("the verifier holds %d spent credentials, want 1: the first is an hour past being usable", len(v.spent))
	}
}
