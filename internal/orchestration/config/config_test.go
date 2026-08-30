// relais, a realtime SFU backend for community self-hosters.
// Copyright (C) 2026 Nils Lehnen
//
// Licensed under the GNU Affero General Public License, version 3. See LICENSE
// for the full terms, including the warranty and liability disclaimer.

// The suite for the configuration reading.
//
// Every test below is a way this is known to be got wrong rather than a way it
// might be, and each is written so that it passes on a reading that is missing
// exactly one rule. The three that matter most are the three fall backs issue
// #79 names: a missing credential that admits everyone, an unwritable data path
// that becomes memory, and a cap that does not parse and becomes no cap. Each of
// those has a near miss beside it, where the same setting is ABSENT rather than
// unusable and the answer is a derivation instead of a refusal, because a
// reading that collapsed the two would pass every test that only tried one.
//
// Nothing here reads the process environment, needs a display, needs elevation
// or reaches a network. The one test that touches a filesystem uses the
// directory the toolchain hands it and removes nothing of its own.
package config

import (
	"crypto/ed25519"
	"encoding/base64"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// aKey is the public half an operator would paste in, in the encoding the
// credential package reads. Built from a fixed seed rather than generated: no
// test asserts anything about its value, and a suite producing different bytes
// on every run is a suite whose failures cannot be reproduced from what it
// printed.
func aKey() string {
	seed := make([]byte, ed25519.SeedSize)
	for i := range seed {
		seed[i] = 0x2b
	}
	return base64.RawURLEncoding.EncodeToString(ed25519.NewKeyFromSeed(seed).Public().(ed25519.PublicKey))
}

// writable is the probe for the tests that are not about the filesystem. It
// accepts every directory, so a refusal in those tests is about the value rather
// than about the machine the suite is running on.
func writable(string) error { return nil }

// unwritable is the probe for the tests that are. It refuses every directory the
// way a read-only mount or a full disk does, which is a condition no permission
// bit on the path would have reported.
func unwritable(string) error { return errors.New("read-only file system") }

// complete is an environment with every setting supplied and every value good.
// Each test below is one departure from it.
func complete() []string {
	return []string{
		"RELAIS_NAME=relais.example.org",
		"RELAIS_CREDENTIAL_PUBLIC_KEY=" + aKey(),
		"RELAIS_DATA_DIRECTORY=/srv/relais",
		"RELAIS_CAP=60",
	}
}

// without returns the environment with one setting removed.
func without(environment []string, variable string) []string {
	kept := make([]string, 0, len(environment))
	for _, entry := range environment {
		if strings.HasPrefix(entry, variable+"=") {
			continue
		}
		kept = append(kept, entry)
	}
	return kept
}

// with returns the environment with one setting replaced or added.
func with(environment []string, variable, value string) []string {
	return append(without(environment, variable), variable+"="+value)
}

// refusalFor runs a reading that is expected to fail and returns what it refused.
func refusalFor(t *testing.T, environment []string, probe func(string) error) Refused {
	t.Helper()
	cfg, err := Load(environment, probe)
	if err == nil {
		t.Fatalf("the configuration was accepted and should have been refused: %+v", cfg)
	}
	var refused Refused
	if !errors.As(err, &refused) {
		t.Fatalf("the refusal is %T rather than Refused: %v", err, err)
	}
	if len(refused) == 0 {
		t.Fatal("the reading refused and named nothing")
	}
	return refused
}

// about returns the refusal for one variable, and fails when there is none.
func about(t *testing.T, refused Refused, variable string) Refusal {
	t.Helper()
	for _, one := range refused {
		if one.Variable == variable {
			return one
		}
	}
	t.Fatalf("nothing was refused about %s, only %v", variable, refused.Names())
	return Refusal{}
}

// names fails unless one refusal is about the variable. It is beside [about]
// rather than a call to it because a Refusal is an error, and a test that
// discarded one would be discarding an error value, which the style gate refuses
// for the good reason that everywhere else it is a mistake.
func names(t *testing.T, refused Refused, variable string) {
	t.Helper()
	for _, one := range refused {
		if one.Variable == variable {
			return
		}
	}
	t.Fatalf("nothing was refused about %s, only %v", variable, refused.Names())
}

// TestACompleteConfigurationIsRead is the case every other test is a departure
// from. It also fixes that a supplied value arrives unchanged, since a reading
// that quietly rewrote one would pass every refusal test in this file.
func TestACompleteConfigurationIsRead(t *testing.T) {
	cfg, err := Load(complete(), writable)
	if err != nil {
		t.Fatalf("a complete configuration was refused: %v", err)
	}
	if cfg.Name != "relais.example.org" {
		t.Errorf("the name is %q", cfg.Name)
	}
	if got := base64.RawURLEncoding.EncodeToString(cfg.VerifyingKey); got != aKey() {
		t.Errorf("the key is %q", got)
	}
	if cfg.DataDirectory != "/srv/relais" {
		t.Errorf("the data directory is %q", cfg.DataDirectory)
	}
	if cfg.Cap != (Cap{Percent: 60, Stated: true}) {
		t.Errorf("the cap is %+v", cfg.Cap)
	}
	if len(cfg.Derived) != 0 {
		t.Errorf("nothing was left to derive and %v was derived", cfg.Derived)
	}
}

// TestAnUnknownSettingStopsStartup covers the rule that a typed name is never
// silently ignored. Without it an operator who mistypes a cap believes a limit
// is in force that the service never read.
func TestAnUnknownSettingStopsStartup(t *testing.T) {
	refused := refusalFor(t, append(complete(), "RELAIS_CAPP=60"), writable)
	one := about(t, refused, "RELAIS_CAPP")
	if !strings.Contains(one.Expected, "RELAIS_CAP") {
		t.Errorf("the refusal does not say what was expected: %q", one.Expected)
	}
}

// TestAVariableOutsideThePrefixIsNotJudged is the other side of the same rule.
// A reading that refused everything it did not recognise would refuse every
// process environment it was ever handed.
func TestAVariableOutsideThePrefixIsNotJudged(t *testing.T) {
	if _, err := Load(append(complete(), "PATH=/usr/bin", "HOME=/root"), writable); err != nil {
		t.Fatalf("a variable belonging to something else was judged: %v", err)
	}
}

// TestAPrefixedEntryWithNoValueIsRefused covers the exported name with nothing
// after it. Read as absent, it would take a typed setting out of the reading
// altogether, which is the unknown-setting failure one step further along.
func TestAPrefixedEntryWithNoValueIsRefused(t *testing.T) {
	refused := refusalFor(t, append(complete(), "RELAIS_CAP"), writable)
	names(t, refused, "RELAIS_CAP")
}

// TestEveryProblemIsReportedInOneRefusal covers validation as a whole. A reading
// that stopped at the first problem is a configuration an operator repairs one
// restart at a time.
func TestEveryProblemIsReportedInOneRefusal(t *testing.T) {
	environment := with(with(without(complete(), "RELAIS_CREDENTIAL_PUBLIC_KEY"),
		"RELAIS_NAME", "https://relais.example.org"),
		"RELAIS_CAP", "sixty")
	refused := refusalFor(t, environment, writable)
	for _, variable := range []string{"RELAIS_NAME", "RELAIS_CREDENTIAL_PUBLIC_KEY", "RELAIS_CAP"} {
		names(t, refused, variable)
	}
	if len(refused) != 3 {
		t.Errorf("three settings were wrong and %d were refused: %v", len(refused), refused.Names())
	}
	if !strings.Contains(refused.Error(), "3 problems") {
		t.Errorf("the rendered refusal does not count them: %q", refused.Error())
	}
}

// TestARefusalNamesTheSettingTheProblemAndTheExpectation is the shape issue #79
// asks a refusal to carry. A refusal saying only that the configuration is wrong
// sends the operator to read the source.
func TestARefusalNamesTheSettingTheProblemAndTheExpectation(t *testing.T) {
	one := about(t, refusalFor(t, with(complete(), "RELAIS_NAME", "relais.example.org:8443"), writable), "RELAIS_NAME")
	if one.Problem == "" || one.Expected == "" {
		t.Fatalf("the refusal is incomplete: %+v", one)
	}
	rendered := one.Error()
	for _, part := range []string{one.Variable, one.Problem, one.Expected} {
		if !strings.Contains(rendered, part) {
			t.Errorf("the rendered refusal drops %q: %q", part, rendered)
		}
	}
}

// TestAMissingCredentialDoesNotAdmitEveryone is the first of the three fall
// backs. Deleting the required check for the credential is what turns it red.
func TestAMissingCredentialDoesNotAdmitEveryone(t *testing.T) {
	refused := refusalFor(t, without(complete(), "RELAIS_CREDENTIAL_PUBLIC_KEY"), writable)
	names(t, refused, "RELAIS_CREDENTIAL_PUBLIC_KEY")
}

// TestAnUnusableCredentialIsRefusedRatherThanIgnored is the same fall back
// reached from the other side, where the operator supplied something that is not
// a key. A reading that dropped what it could not decode would start with no key
// at all.
func TestAnUnusableCredentialIsRefusedRatherThanIgnored(t *testing.T) {
	for _, value := range []string{"", "not base64url!", base64.RawURLEncoding.EncodeToString([]byte("too short"))} {
		refused := refusalFor(t, with(complete(), "RELAIS_CREDENTIAL_PUBLIC_KEY", value), writable)
		if cfg, err := Load(with(complete(), "RELAIS_CREDENTIAL_PUBLIC_KEY", value), writable); err == nil {
			t.Fatalf("%q was accepted as a key: %+v", value, cfg)
		}
		names(t, refused, "RELAIS_CREDENTIAL_PUBLIC_KEY")
	}
}

// TestAnUnwritableDataPathDoesNotBecomeMemory is the second fall back. The path
// is well formed and absolute; only the write fails, which is the case a check
// on the string cannot reach.
func TestAnUnwritableDataPathDoesNotBecomeMemory(t *testing.T) {
	refused := refusalFor(t, complete(), unwritable)
	one := about(t, refused, "RELAIS_DATA_DIRECTORY")
	if !strings.Contains(one.Problem, "could not be written") {
		t.Errorf("the refusal is about something other than the write: %q", one.Problem)
	}
}

// TestAnUnusableDataPathDoesNotFallBackToItsDerivation is the precedence rule at
// the place it costs something. The derivation exists and is good, and it is
// still not used, because the operator said something and what they said did not
// work.
func TestAnUnusableDataPathDoesNotFallBackToItsDerivation(t *testing.T) {
	cfg, err := Load(with(complete(), "RELAIS_DATA_DIRECTORY", "relais-data"), writable)
	if err == nil {
		t.Fatalf("a relative data path was accepted: %+v", cfg)
	}
	if cfg.DataDirectory == DefaultDataDirectory {
		t.Fatal("the refused reading returned the derived directory")
	}
	if cfg.DataDirectory != "" {
		t.Fatalf("the refused reading returned a directory at all: %q", cfg.DataDirectory)
	}
}

// TestAnAbsentDataPathIsDerived is the near miss beside the two tests above.
// Saying nothing leaves the decision to the service; saying something unusable
// does not, and a reading that collapsed them would pass whichever of the two
// was written first.
func TestAnAbsentDataPathIsDerived(t *testing.T) {
	cfg, err := Load(without(complete(), "RELAIS_DATA_DIRECTORY"), writable)
	if err != nil {
		t.Fatalf("an absent optional setting was refused: %v", err)
	}
	if cfg.DataDirectory != DefaultDataDirectory {
		t.Errorf("the data directory is %q rather than the derived one", cfg.DataDirectory)
	}
	if len(cfg.Derived) != 1 || cfg.Derived[0] != "RELAIS_DATA_DIRECTORY" {
		t.Errorf("the derivation is not reported: %v", cfg.Derived)
	}
}

// TestACapThatDoesNotParseDoesNotBecomeNoCap is the third fall back. Each value
// below is a way of writing a cap that a lenient reading turns into the whole
// machine.
func TestACapThatDoesNotParseDoesNotBecomeNoCap(t *testing.T) {
	for _, value := range []string{"sixty", "60.5", "", "-5", "0", "101"} {
		cfg, err := Load(with(complete(), "RELAIS_CAP", value), writable)
		if err == nil {
			t.Fatalf("%q was accepted as a cap: %+v", value, cfg)
		}
		if cfg.Cap.Percent == 100 {
			t.Fatalf("%q was refused and the whole machine was returned anyway", value)
		}
		names(t, refusalFor(t, with(complete(), "RELAIS_CAP", value), writable), "RELAIS_CAP")
	}
}

// TestAnAbsentCapIsTheWholeMachineAndSaysSo is the near miss beside it. The
// number is the same as an unlimited fall back and the fact is different, which
// is what Stated carries and what issue #58 has to print.
func TestAnAbsentCapIsTheWholeMachineAndSaysSo(t *testing.T) {
	cfg, err := Load(without(complete(), "RELAIS_CAP"), writable)
	if err != nil {
		t.Fatalf("an absent cap was refused: %v", err)
	}
	if cfg.Cap != (Cap{Percent: 100, Stated: false}) {
		t.Errorf("the derived cap is %+v", cfg.Cap)
	}
}

// TestACapWithATrailingSignIsRead covers the spelling an operator types. It is
// beside the test above rather than inside it because a reading that refused it
// would be refusing a correct answer, which is the more expensive mistake.
func TestACapWithATrailingSignIsRead(t *testing.T) {
	cfg, err := Load(with(complete(), "RELAIS_CAP", "60%"), writable)
	if err != nil {
		t.Fatalf("a cap written with a sign was refused: %v", err)
	}
	if cfg.Cap != (Cap{Percent: 60, Stated: true}) {
		t.Errorf("the cap is %+v", cfg.Cap)
	}
}

// TestASuppliedValueIsNotOverriddenByADerivation is the other half of the
// precedence rule. Both derivable settings are supplied and neither moves.
func TestASuppliedValueIsNotOverriddenByADerivation(t *testing.T) {
	cfg, err := Load(with(with(complete(), "RELAIS_DATA_DIRECTORY", "/mnt/tank/relais"), "RELAIS_CAP", "25"), writable)
	if err != nil {
		t.Fatalf("a supplied configuration was refused: %v", err)
	}
	if cfg.DataDirectory != "/mnt/tank/relais" {
		t.Errorf("the supplied directory was replaced by %q", cfg.DataDirectory)
	}
	if cfg.Cap != (Cap{Percent: 25, Stated: true}) {
		t.Errorf("the supplied cap was replaced by %+v", cfg.Cap)
	}
}

// TestASettingSetToNothingIsRefused covers the value an operator supplies as an
// empty string, which is what a template with an unfilled placeholder produces.
// An optional setting read as absent here would be a fall back arriving through
// the shell rather than through the reading.
func TestASettingSetToNothingIsRefused(t *testing.T) {
	for _, variable := range []string{
		"RELAIS_NAME",
		"RELAIS_CREDENTIAL_PUBLIC_KEY",
		"RELAIS_DATA_DIRECTORY",
		"RELAIS_CAP",
	} {
		refused := refusalFor(t, with(complete(), variable, ""), writable)
		one := about(t, refused, variable)
		if !strings.Contains(one.Problem, "empty") {
			t.Errorf("%s was refused for something other than being empty: %q", variable, one.Problem)
		}
		if names := refused.Names(); len(names) != 1 || names[0] != variable {
			t.Errorf("the refusal names %v", names)
		}
		if !strings.Contains(refused.Error(), "1 problem:") {
			t.Errorf("one problem is rendered as %q", refused.Error())
		}
	}
}

// TestAMissingNameIsRefused covers the one value the deployment contract says an
// operator always supplies. Nothing derives it, because no reading of a machine
// produces the name somebody pointed at it.
func TestAMissingNameIsRefused(t *testing.T) {
	names(t, refusalFor(t, without(complete(), "RELAIS_NAME"), writable), "RELAIS_NAME")
}

// TestTheNameIsRefusedForWhatAnOperatorActuallyTypes walks the near misses one
// by one. Each value below is a name somebody would paste rather than a string
// chosen to break a parser.
func TestTheNameIsRefusedForWhatAnOperatorActuallyTypes(t *testing.T) {
	for _, value := range []string{
		"https://relais.example.org",
		"relais.example.org/join",
		"relais.example.org:8443",
		"relais",
		"relais..example.org",
		"-relais.example.org",
		"relais.example.org-",
		"relais_1.example.org",
		strings.Repeat("a", 64) + ".example.org",
		strings.Repeat("a.", 130) + "example.org",
	} {
		if cfg, err := Load(with(complete(), "RELAIS_NAME", value), writable); err == nil {
			t.Errorf("%q was accepted as a name: %q", value, cfg.Name)
		}
	}
}

// TestANameIsLowercasedAndTrimmedRatherThanRefused keeps the reading from being
// strict in the direction that costs an operator a working deployment for a
// capital letter, which a name in DNS does not distinguish, or for a space a
// shell carried in from the end of a line. Both are repaired rather than
// refused; everything in the test above is refused rather than repaired, and the
// line between the two is that a repair has exactly one sensible result.
func TestANameIsLowercasedAndTrimmedRatherThanRefused(t *testing.T) {
	for _, value := range []string{"Relais.Example.ORG", "  relais.example.org	"} {
		cfg, err := Load(with(complete(), "RELAIS_NAME", value), writable)
		if err != nil {
			t.Fatalf("%q was refused: %v", value, err)
		}
		if cfg.Name != "relais.example.org" {
			t.Errorf("%q was read as %q", value, cfg.Name)
		}
	}
}

// TestAReadingWithNoProbeIsRefused covers the caller that hands in nothing to
// check a directory with. A reading that took the absence for permission would
// accept any path at all, which is the fall back this whole package is against,
// arriving through the caller instead of through the value.
func TestAReadingWithNoProbeIsRefused(t *testing.T) {
	if cfg, err := Load(complete(), nil); err == nil {
		t.Fatalf("a reading with no probe was accepted: %+v", cfg)
	}
}

// TestTheProbeWritesRatherThanAsks exercises the real probe against the
// filesystem, in the directory the toolchain hands the test. It is the one test
// here that touches a disk, and it needs no display, no elevation and no
// network.
func TestTheProbeWritesRatherThanAsks(t *testing.T) {
	dir := t.TempDir()
	if err := WritableDirectory(dir); err != nil {
		t.Fatalf("a directory the suite owns was reported unwritable: %v", err)
	}

	file := filepath.Join(dir, "not-a-directory")
	if err := os.WriteFile(file, []byte("x"), 0o600); err != nil {
		t.Fatalf("the fixture could not be written: %v", err)
	}
	if err := WritableDirectory(file); err == nil {
		t.Error("a path that is a file was reported writable")
	}
	if err := WritableDirectory(filepath.Join(dir, "absent")); err == nil {
		t.Error("a path that is not there was reported writable")
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("the directory could not be read back: %v", err)
	}
	if len(entries) != 1 {
		t.Errorf("the probe left %d entries behind rather than only the fixture", len(entries))
	}
}

// TestEverySettingNamesOneOfTheFourCategories is the rule the configuration
// record states and the test issue #59 will read. A fifth category is a change
// to that record rather than a field somebody added here.
func TestEverySettingNamesOneOfTheFourCategories(t *testing.T) {
	four := map[Category]struct{}{
		CategoryName:       {},
		CategoryCredential: {},
		CategoryData:       {},
		CategorySpend:      {},
	}
	seen := map[string]struct{}{}
	for _, setting := range Surface {
		if _, ok := four[setting.Category]; !ok {
			t.Errorf("%s names %q, which is not one of the four", setting.Variable, setting.Category)
		}
		if !strings.HasPrefix(setting.Variable, Prefix) {
			t.Errorf("%s does not carry the prefix, so nothing would read it", setting.Variable)
		}
		if setting.Expects == "" || setting.Why == "" {
			t.Errorf("%s says nothing about what it expects or why it exists", setting.Variable)
		}
		if _, twice := seen[setting.Variable]; twice {
			t.Errorf("%s is on the surface twice", setting.Variable)
		}
		seen[setting.Variable] = struct{}{}
	}
}
