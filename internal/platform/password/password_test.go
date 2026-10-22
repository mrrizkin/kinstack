package password

import (
	"errors"
	"strings"
	"testing"
)

// testCost keeps the KDF cheap enough to run in unit tests.
var testCost = Cost{Memory: 8 * 1024, Iterations: 1, Parallelism: 1}

func TestHashAndCompareRoundTrip(t *testing.T) {
	hash, err := HashWithCost("correct horse battery staple", testCost)
	if err != nil {
		t.Fatalf("hash: %v", err)
	}
	if err := Compare(hash, "correct horse battery staple"); err != nil {
		t.Fatalf("compare: %v", err)
	}
	if err := Compare(hash, "wrong password"); !errors.Is(err, ErrInvalidPasswordHash) {
		t.Fatalf("wrong password: %v", err)
	}
}

func TestHashUsesAFreshSalt(t *testing.T) {
	first, err := HashWithCost("same secret", testCost)
	if err != nil {
		t.Fatalf("hash: %v", err)
	}
	second, err := HashWithCost("same secret", testCost)
	if err != nil {
		t.Fatalf("hash: %v", err)
	}
	if first == second {
		t.Fatal("two hashes of one secret must differ")
	}
	if err := Compare(second, "same secret"); err != nil {
		t.Fatalf("compare: %v", err)
	}
}

func TestCompareVerifiesTheEncodedCost(t *testing.T) {
	hash, err := HashWithCost("one time code", OneTimeCost)
	if err != nil {
		t.Fatalf("hash: %v", err)
	}
	if !strings.Contains(hash, "m=32768,t=2,p=1") {
		t.Fatalf("hash does not carry its cost: %q", hash)
	}
	if err := Compare(hash, "one time code"); err != nil {
		t.Fatalf("compare: %v", err)
	}
}

func TestHashWithCostRejectsIncompleteCost(t *testing.T) {
	for name, cost := range map[string]Cost{
		"no memory":      {Iterations: 1, Parallelism: 1},
		"no iterations":  {Memory: 8 * 1024, Parallelism: 1},
		"no parallelism": {Memory: 8 * 1024, Iterations: 1},
	} {
		if _, err := HashWithCost("secret", cost); err == nil {
			t.Fatalf("%s: expected an error", name)
		}
	}
}

func TestCompareRejectsMalformedHashes(t *testing.T) {
	valid, err := HashWithCost("secret", testCost)
	if err != nil {
		t.Fatalf("hash: %v", err)
	}
	parts := strings.Split(valid, "$")
	cases := map[string]string{
		"empty":         "",
		"truncated":     "$argon2id$v=19$m=8192,t=1,p=1",
		"wrong scheme":  strings.Replace(valid, "argon2id", "argon2i", 1),
		"wrong version": strings.Replace(valid, "v=19", "v=16", 1),
		"zero memory":   strings.Replace(valid, "m=8192", "m=0", 1),
		"zero time":     strings.Replace(valid, "t=1", "t=0", 1),
		"zero parallel": strings.Replace(valid, "p=1", "p=0", 1),
		"bad base64":    parts[0] + "$" + parts[1] + "$" + parts[2] + "$" + parts[3] + "$!!!$" + parts[5],
		"short salt":    parts[0] + "$" + parts[1] + "$" + parts[2] + "$" + parts[3] + "$AAAA$" + parts[5],
		"empty key":     parts[0] + "$" + parts[1] + "$" + parts[2] + "$" + parts[3] + "$" + parts[4] + "$",
		"extra field":   valid + "$tail",
		"huge memory":   strings.Replace(valid, "m=8192", "m=1048576", 1),
		"slow rounds":   strings.Replace(valid, "t=1", "t=99", 1),
		"many lanes":    strings.Replace(valid, "p=1", "p=64", 1),
	}
	for name, encoded := range cases {
		if err := Compare(encoded, "secret"); !errors.Is(err, ErrInvalidPasswordHash) {
			t.Fatalf("%s: expected ErrInvalidPasswordHash, got %v", name, err)
		}
	}
}

func TestDummyHashStaysStableAndRejectsRealPasswords(t *testing.T) {
	first := DummyHash()
	if first != DummyHash() {
		t.Fatal("dummy hash must stay stable, otherwise unknown-user timing varies")
	}
	for _, candidate := range []string{"", "password", "correct horse battery staple"} {
		if err := Compare(first, candidate); !errors.Is(err, ErrInvalidPasswordHash) {
			t.Fatalf("dummy hash accepted %q: %v", candidate, err)
		}
	}
}
