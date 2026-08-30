// relais, a realtime SFU backend for community self-hosters.
// Copyright (C) 2026 Nils Lehnen
//
// Licensed under the GNU Affero General Public License, version 3. See LICENSE
// for the full terms, including the warranty and liability disclaimer.

package fuzz_test

import (
	"crypto/ed25519"
	"errors"
	"strings"
	"testing"

	"github.com/iderex/relais/internal/orchestration/config"
)

// alwaysWritable is the probe the fuzzed reading is given. It accepts every
// directory, so what the fuzzer is driving is the reading rather than the
// filesystem of whichever machine the target happens to run on, and a crash
// found on one machine reproduces on another from the same corpus entry.
func alwaysWritable(string) error { return nil }

// FuzzTheConfigurationAnOperatorSupplies drives the configuration reading with
// environment entries nobody here wrote.
//
// The bytes are split on newlines into entries in the form the process
// environment uses, which gives the fuzzer the names as well as the values. That
// matters: the unknown-setting rule is a decision about a NAME, and a target that
// only varied values would report the surface as fuzzed while never reaching it.
//
// This is a decoder by the derivation in this package's own suite, and it decodes
// something an operator pasted rather than something a stranger sent, so the
// argument for fuzzing it is not the one the credential target rests on. It is
// that the reading is what decides whether the service starts and in what state,
// and its refusals are the only thing standing between a mistyped value and a
// service running as nobody intended.
//
// Four properties, the first of which needs no assertion:
//
// It does not panic. A configuration reading that crashes on a value is a service
// that cannot be told what is wrong with it.
//
// A refusal is one a caller can act on: every refusal names a variable, what was
// wrong and what was expected. An empty field there is a refusal an operator
// cannot repair.
//
// A refusal returns nothing. A partially filled configuration returned beside an
// error is how a caller that reads the value before the error starts a service on
// settings that were refused, which is the fall back issue #79 is about, arriving
// through the caller rather than through the reading.
//
// An accepted configuration is one nothing downstream has to check again. The
// name is lowercase and has more than one label, the key is exactly an Ed25519
// public key, the directory is absolute, and the cap is inside the range a share
// of one machine can be. Every one of those is a promise the rest of the service
// is entitled to make without re-reading the string it came from.
func FuzzTheConfigurationAnOperatorSupplies(f *testing.F) {
	f.Fuzz(func(t *testing.T, environment string) {
		cfg, err := config.Load(strings.Split(environment, "\n"), alwaysWritable)
		if err != nil {
			var refused config.Refused
			if !errors.As(err, &refused) {
				t.Fatalf("refused with %v, which is not a config.Refused, so a caller has nothing to route on", err)
			}
			if len(refused) == 0 {
				t.Fatal("refused and named nothing")
			}
			for _, one := range refused {
				if one.Variable == "" || one.Problem == "" || one.Expected == "" {
					t.Fatalf("refused with %+v, which does not name the setting, the problem and the expectation", one)
				}
			}
			// Field by field rather than against the zero value,
			// because a configuration carries slices and is
			// therefore not comparable. Every field is named, so a
			// configuration that grows one leaves this assertion
			// visibly incomplete rather than silently so.
			if cfg.Name != "" || len(cfg.VerifyingKey) != 0 || cfg.DataDirectory != "" ||
				cfg.Cap != (config.Cap{}) || len(cfg.Derived) != 0 {
				t.Fatalf("refused and returned %+v, so a caller reading the value before the error starts on settings that were refused", cfg)
			}
			return
		}

		if cfg.Name != strings.ToLower(cfg.Name) || !strings.Contains(cfg.Name, ".") {
			t.Fatalf("accepted the name %q, which is not a lowercased name in DNS", cfg.Name)
		}
		if strings.ContainsAny(cfg.Name, ":/ ") {
			t.Fatalf("accepted the name %q, which carries a port, a path or a space", cfg.Name)
		}
		if len(cfg.VerifyingKey) != ed25519.PublicKeySize {
			t.Fatalf("accepted a key of %d bytes, and an Ed25519 public key is %d", len(cfg.VerifyingKey), ed25519.PublicKeySize)
		}
		if !strings.HasPrefix(cfg.DataDirectory, "/") {
			t.Fatalf("accepted the data directory %q, which is not an absolute path", cfg.DataDirectory)
		}
		if cfg.Cap.Percent < 1 || cfg.Cap.Percent > 100 {
			t.Fatalf("accepted the cap %+v, which is outside the range a share of one machine can be", cfg.Cap)
		}
	})
}
