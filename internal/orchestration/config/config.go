// relais, a realtime SFU backend for community self-hosters.
// Copyright (C) 2026 Nils Lehnen
//
// Licensed under the GNU Affero General Public License, version 3. See LICENSE
// for the full terms, including the warranty and liability disclaimer.

// Package config reads what an operator supplied, validates the whole of it, and
// either returns a configuration or refuses to produce one.
//
// The surface is small because docs/decisions/configuration.md makes derivation
// the default and configuration the exception, and it closes the exception over
// four categories: the name the service answers on, the credential material,
// where data lives, and what the service is allowed to spend. Every setting here
// names the category that admits it, so a fifth category arrives as a change to
// that record rather than as a field somebody added.
//
// Small is not the same as safe, which is the whole argument of issue #79. Each
// of those four can be wrong in a way that either stops the service or, worse,
// starts it in a state nobody intended, so the reading is all-or-nothing: every
// setting is checked, every problem is collected, and a single refusal carries
// all of them. Nothing is applied while the reading is under way, so there is no
// half-configured state for a caller to find.
//
// WHAT FAILING CLOSED MEANS HERE, and it is the part worth reading twice. A
// setting that is present and unusable is a refusal. It is never a fall back to
// the value the service would have derived had the operator said nothing,
// because those two are opposite statements: saying nothing leaves a decision to
// the service, and saying something unusable is a decision that did not arrive.
// A missing credential does not admit everyone, an unwritable data path does not
// become memory, and a cap that does not parse does not become no cap at all.
//
// ABSENT AND UNUSABLE ARE DIFFERENT INPUTS AND THEY GET DIFFERENT ANSWERS. Two
// of the four settings are derivable and are derived when the operator says
// nothing about them; the same two are refused when the operator says something
// that cannot be used. A reader who takes the first for the second reads this
// package as one that falls back, and it does not.
//
// It reads a slice of environment entries handed to it rather than the process
// environment, so a test drives it without touching the machine it runs on and
// two tests cannot reach each other.
package config

import (
	"crypto/ed25519"
	"encoding/base64"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
)

// Prefix is what marks an environment entry as addressed to this service.
//
// It is also the reason an unknown setting can be refused at all: without a
// prefix the reading could not tell a setting nobody defines from a variable
// belonging to something else on the host, and it would have to ignore both.
//
// The cost is that the prefix belongs to the service and to nothing else. A tool
// that sets a prefixed variable in a process which goes on to start this service
// stops it, and this repository already carries one such variable in the
// coverage workflow, read by a suite rather than by the service. That is a near
// miss rather than a collision today, and it is written here because the next
// one will be written by somebody who did not know the prefix was claimed.
const Prefix = "RELAIS_"

// A Category is one of the four kinds of value the configuration record leaves
// to the operator. It is carried on every setting rather than checked once,
// because the test in issue #59 reds on an entry that fits none of them and a
// category held only in prose is one that test cannot read.
type Category string

// The four, in the record's own words.
const (
	CategoryName       Category = "the name the service answers on"
	CategoryCredential Category = "the credential material"
	CategoryData       Category = "where its data lives"
	CategorySpend      Category = "what it is allowed to spend"
)

// A Setting is one entry on the configuration surface.
//
// Expects is written as what a good value looks like rather than as a pattern,
// because it is printed back to an operator who has just got one wrong and a
// regular expression is not an instruction.
type Setting struct {
	Variable string
	Category Category
	Required bool
	Expects  string
	Why      string
}

// Surface is the whole configuration surface. A value that is not on this list
// is not asked of an operator, and a variable carrying [Prefix] that is not on
// it stops startup.
var Surface = []Setting{
	{
		Variable: Prefix + "NAME",
		Category: CategoryName,
		Required: true,
		Expects:  "a name in DNS that points at this host, such as relais.example.org, with no scheme, no port and no path",
		Why:      "Only the operator knows what their network looks like, and a certificate needs a name.",
	},
	{
		Variable: Prefix + "CREDENTIAL_PUBLIC_KEY",
		Category: CategoryCredential,
		Required: true,
		Expects:  "the Ed25519 public key of the service that mints credentials, base64url without padding",
		Why:      "Only the operator knows what the service above them is holding, and the private half stays there.",
	},
	{
		Variable: Prefix + "DATA_DIRECTORY",
		Category: CategoryData,
		Required: false,
		Expects:  "an absolute path to a directory that exists and can be written",
		Why:      "Only the operator knows their storage. Where they say nothing, the packaged location is used.",
	},
	{
		Variable: Prefix + "CAP",
		Category: CategorySpend,
		Required: false,
		Expects:  "a whole number of percent between 1 and 100, with or without a trailing sign, such as 60 or 60%",
		Why:      "How much of a machine this service may consume is a policy about the operator's hardware, and no derivation substitutes for it.",
	},
}

// DefaultDataDirectory is where data lives when the operator says nothing about
// it. It is a derivation rather than a preference: the value comes from the
// deployment shape, and an operator who says something else is not overridden by
// it.
const DefaultDataDirectory = "/var/lib/relais"

// A Config is a configuration that passed the whole reading. There is no way to
// build one that did not, which is what stops a partially validated value from
// reaching the rest of the service.
type Config struct {
	// Name is the name the service answers on, lowercased.
	Name string
	// VerifyingKey is the public half the credential verifier is built from.
	VerifyingKey ed25519.PublicKey
	// DataDirectory is an absolute path to a directory that existed and
	// accepted a write at the moment it was read.
	DataDirectory string
	// Cap is the share of the host the service may consume.
	Cap Cap
	// Derived names the variables the operator did not supply and the
	// service worked out, in surface order. Issue #58 prints these with the
	// input each came from; this is the list rather than the printing.
	Derived []string
}

// A Cap is what the operator allows the service to spend.
//
// Stated is carried rather than inferred from Percent, because "the operator
// allowed the whole machine" and "the operator said nothing and the whole
// machine is what that means" are the same number and different facts, and only
// the second one is a derivation to print.
type Cap struct {
	Percent int
	Stated  bool
}

// A Refusal is one thing wrong with the configuration.
//
// It names the setting, what was wrong with it, and what was expected, which is
// the shape issue #79 asks for. The three are separate fields rather than one
// sentence, so a caller reporting them somewhere other than a terminal does not
// have to take a sentence apart again.
type Refusal struct {
	Variable string
	Problem  string
	Expected string
}

func (r Refusal) Error() string {
	return fmt.Sprintf("%s: %s. Expected %s.", r.Variable, r.Problem, r.Expected)
}

// Refused is every refusal from one reading.
//
// The whole set is returned rather than the first, because a configuration read
// one problem at a time is a configuration an operator repairs one restart at a
// time, and the reading knows all of them already.
type Refused []Refusal

func (r Refused) Error() string {
	lines := make([]string, 0, len(r)+1)
	lines = append(lines, "the configuration was refused, "+counted(len(r))+":")
	for _, one := range r {
		lines = append(lines, "  "+one.Error())
	}
	return strings.Join(lines, "\n")
}

// Names returns the variables the refusals are about, in the order they were
// found. A caller that reports refusals somewhere structured uses this rather
// than reading them back out of the rendered sentence.
func (r Refused) Names() []string {
	all := make([]string, 0, len(r))
	for _, one := range r {
		all = append(all, one.Variable)
	}
	return all
}

// counted renders the number of refusals so the first line does not read
// "1 problems". The plural is the only thing it decides.
func counted(n int) string {
	if n == 1 {
		return "1 problem"
	}
	return strconv.Itoa(n) + " problems"
}

// Load reads a configuration out of environment entries in the form the process
// environment uses, and refuses rather than returning a partial one.
//
// probe decides whether a directory can be written. It is a parameter so the
// reading is drivable without a filesystem, and [WritableDirectory] is what a
// caller with one passes.
func Load(environment []string, probe func(dir string) error) (Config, error) {
	if probe == nil {
		return Config{}, errors.New("config: reading a configuration needs a way to check that a directory can be written")
	}

	values, refused := split(environment)

	cfg := Config{}
	for _, setting := range Surface {
		value, present := values[setting.Variable]
		if !present {
			if setting.Required {
				refused = append(refused, Refusal{
					Variable: setting.Variable,
					Problem:  "no value was supplied and this one cannot be worked out",
					Expected: setting.Expects,
				})
				continue
			}
			cfg.Derived = append(cfg.Derived, setting.Variable)
			derive(&cfg, setting)
			continue
		}
		if bad := apply(&cfg, setting, value, probe); bad != nil {
			refused = append(refused, *bad)
		}
	}

	if len(refused) > 0 {
		return Config{}, Refused(refused)
	}
	return cfg, nil
}

// split separates the environment entries addressed to this service from the
// refusals for the ones nothing defines.
//
// An entry carrying the prefix and no separator is refused rather than ignored:
// it is how a shell exports a name with no value, and reading it as absent would
// make a typed setting disappear silently, which is the failure the
// unknown-setting rule exists for one step further along.
func split(environment []string) (map[string]string, []Refusal) {
	known := make(map[string]struct{}, len(Surface))
	for _, setting := range Surface {
		known[setting.Variable] = struct{}{}
	}

	values := make(map[string]string)
	var refused []Refusal
	for _, entry := range environment {
		name, value, ok := strings.Cut(entry, "=")
		if !strings.HasPrefix(name, Prefix) {
			continue
		}
		if !ok {
			refused = append(refused, Refusal{
				Variable: name,
				Problem:  "the entry carries no value at all",
				Expected: "a name and a value separated by an equals sign",
			})
			continue
		}
		if _, isKnown := known[name]; !isKnown {
			refused = append(refused, Refusal{
				Variable: name,
				Problem:  "no setting of this project has that name, and a name that is silently ignored is how an operator comes to believe a limit is in force when it is not",
				Expected: "one of " + variables(),
			})
			continue
		}
		values[name] = value
	}
	return values, refused
}

// variables renders the surface for a refusal that has to say what was expected.
func variables() string {
	all := make([]string, 0, len(Surface))
	for _, setting := range Surface {
		all = append(all, setting.Variable)
	}
	return strings.Join(all, ", ")
}

// derive fills in a setting the operator said nothing about.
//
// Only the two optional settings reach this, and each derivation is the one the
// configuration record permits. A required setting that is absent is a refusal
// before this is called, because a derivation for it would be the service
// inventing an answer only the operator could know.
func derive(cfg *Config, setting Setting) {
	switch setting.Category {
	case CategoryData:
		cfg.DataDirectory = DefaultDataDirectory
	case CategorySpend:
		cfg.Cap = Cap{Percent: 100, Stated: false}
	case CategoryName, CategoryCredential:
	}
}

// apply validates one supplied value and puts it into cfg, or reports what was
// wrong with it. It writes no derived value on failure, which is the whole of
// the fail-closed rule at the one place it could be broken.
func apply(cfg *Config, setting Setting, value string, probe func(dir string) error) *Refusal {
	problem := ""
	switch setting.Category {
	case CategoryName:
		name, bad := readName(value)
		problem = bad
		cfg.Name = name
	case CategoryCredential:
		key, bad := readKey(value)
		problem = bad
		cfg.VerifyingKey = key
	case CategoryData:
		dir, bad := readDirectory(value, probe)
		problem = bad
		cfg.DataDirectory = dir
	case CategorySpend:
		allowed, bad := readCap(value)
		problem = bad
		cfg.Cap = allowed
	}
	if problem == "" {
		return nil
	}
	return &Refusal{Variable: setting.Variable, Problem: problem, Expected: setting.Expects}
}

// readName reads the name the service answers on.
//
// The first three refusals are the mistakes an operator actually makes rather
// than the ones a grammar would find first: a URL pasted out of a browser, a
// host and port copied out of a proxy configuration, and a name that is a single
// label because it works on the operator's own network and nowhere else.
func readName(value string) (string, string) {
	name := strings.ToLower(strings.TrimSpace(value))
	switch {
	case name == "":
		return "", "the value is empty"
	case strings.Contains(name, "://"):
		return "", "the value is a URL rather than a name"
	case strings.Contains(name, "/"):
		return "", "the value carries a path"
	case strings.Contains(name, ":"):
		return "", "the value carries a port, and the ports belong to the network shape rather than to the name"
	case len(name) > 253:
		return "", "the value is longer than a name in DNS may be"
	case !strings.Contains(name, "."):
		return "", "the value is a single label, which is a name on one network rather than a name in DNS"
	}
	for _, label := range strings.Split(name, ".") {
		if bad := readLabel(label); bad != "" {
			return "", bad
		}
	}
	return name, ""
}

// readLabel judges one dot-separated part of a name.
func readLabel(label string) string {
	switch {
	case label == "":
		return "the value has an empty label between two dots"
	case len(label) > 63:
		return "the value has a label longer than a label in DNS may be"
	case strings.HasPrefix(label, "-"), strings.HasSuffix(label, "-"):
		return "the value has a label that starts or ends with a hyphen"
	}
	for _, r := range label {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' {
			continue
		}
		return fmt.Sprintf("the value carries %q, which a name in DNS may not", r)
	}
	return ""
}

// readKey reads the public half of the key the service above the seam signs
// with. The encoding is the credential's own, base64url without padding, so an
// operator moves one string between the two and never learns a second spelling.
func readKey(value string) (ed25519.PublicKey, string) {
	if strings.TrimSpace(value) == "" {
		return nil, "the value is empty, and an absent key admits nobody rather than everybody"
	}
	raw, err := base64.RawURLEncoding.DecodeString(value)
	if err != nil {
		return nil, "the value is not base64url without padding"
	}
	if len(raw) != ed25519.PublicKeySize {
		return nil, fmt.Sprintf("the value decodes to %d bytes and an Ed25519 public key is %d", len(raw), ed25519.PublicKeySize)
	}
	return ed25519.PublicKey(raw), ""
}

// readDirectory reads where data lives and proves the claim rather than parsing
// it. A path that looks like a directory and cannot be written is the case this
// exists for, and it is not decidable from the string.
//
// Absolute means absolute on the host this service runs on, which the deployment
// contract fixes as a current 64-bit Linux host, so the test is the leading
// separator rather than the running machine's opinion of what an absolute path
// looks like. The reading is a configuration for a deployment and not for the
// developer machine it is being read on, and asking the toolchain would give two
// different answers to one question about somebody else's host.
func readDirectory(value string, probe func(dir string) error) (string, string) {
	dir := strings.TrimSpace(value)
	switch {
	case dir == "":
		return "", "the value is empty"
	case !strings.HasPrefix(dir, "/"):
		return "", "the value is not an absolute path, and a relative one means a different directory depending on where the service was started"
	}
	if err := probe(dir); err != nil {
		return "", fmt.Sprintf("the directory could not be written: %v", err)
	}
	return dir, ""
}

// readCap reads what the service is allowed to spend.
//
// Zero is refused rather than read as "nothing", because a service allowed
// nothing is a service that cannot start, and an operator who typed it meant
// something else.
func readCap(value string) (Cap, string) {
	text := strings.TrimSuffix(strings.TrimSpace(value), "%")
	if text == "" {
		return Cap{}, "the value is empty"
	}
	percent, err := strconv.Atoi(text)
	if err != nil {
		return Cap{}, "the value is not a whole number of percent, and an unreadable cap is not the same as no cap"
	}
	if percent < 1 || percent > 100 {
		return Cap{}, fmt.Sprintf("the value is %d, which is outside the range a share of one machine can be", percent)
	}
	return Cap{Percent: percent, Stated: true}, ""
}

// WritableDirectory is the probe a caller with a real filesystem passes to
// [Load]. It writes rather than asks: a permission bit says what the kernel
// would decide for the calling user, and a full disk, a read-only mount and a
// path that is a file say nothing through it at all.
func WritableDirectory(dir string) error {
	info, err := os.Stat(dir)
	if err != nil {
		return err
	}
	if !info.IsDir() {
		return errors.New("the path is not a directory")
	}
	probe, err := os.CreateTemp(dir, ".relais-writable-*")
	if err != nil {
		return err
	}
	name := probe.Name()
	if err := probe.Close(); err != nil {
		return err
	}
	return os.Remove(name)
}
