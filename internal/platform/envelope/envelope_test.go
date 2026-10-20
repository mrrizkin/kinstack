package envelope

import (
	"bytes"
	"errors"
	"strings"
	"testing"
)

func TestSealAndOpenRoundTrip(t *testing.T) {
	box, err := New(strings.Repeat("k", 32), "kinstack/test/v1", 1)
	if err != nil {
		t.Fatalf("new box: %v", err)
	}
	if box.Version() != 1 {
		t.Fatalf("expected version 1, got %d", box.Version())
	}

	payload, err := box.Seal([]byte("JBSWY3DPEHPK3PXP"))
	if err != nil {
		t.Fatalf("seal: %v", err)
	}
	if bytes.Contains(payload, []byte("JBSWY3DPEHPK3PXP")) {
		t.Fatal("ciphertext must not contain the plaintext")
	}

	plaintext, err := box.Open(payload)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	if string(plaintext) != "JBSWY3DPEHPK3PXP" {
		t.Fatalf("unexpected plaintext %q", plaintext)
	}
}

func TestSealUsesFreshNonces(t *testing.T) {
	box, err := New(strings.Repeat("k", 32), "kinstack/test/v1", 1)
	if err != nil {
		t.Fatalf("new box: %v", err)
	}
	first, err := box.Seal([]byte("secret"))
	if err != nil {
		t.Fatalf("seal: %v", err)
	}
	second, err := box.Seal([]byte("secret"))
	if err != nil {
		t.Fatalf("seal: %v", err)
	}
	if bytes.Equal(first, second) {
		t.Fatal("expected a fresh nonce per seal")
	}
}

func TestOpenRejectsTampering(t *testing.T) {
	box, err := New(strings.Repeat("k", 32), "kinstack/test/v1", 1)
	if err != nil {
		t.Fatalf("new box: %v", err)
	}
	payload, err := box.Seal([]byte("secret"))
	if err != nil {
		t.Fatalf("seal: %v", err)
	}
	payload[len(payload)-1] ^= 0xff
	if _, err := box.Open(payload); !errors.Is(err, ErrMalformed) {
		t.Fatalf("expected ErrMalformed, got %v", err)
	}
	if _, err := box.Open(payload[:4]); !errors.Is(err, ErrMalformed) {
		t.Fatalf("expected ErrMalformed for a short payload, got %v", err)
	}
}

func TestOpenRejectsOtherKeyVersion(t *testing.T) {
	first, err := New(strings.Repeat("k", 32), "kinstack/test/v1", 1)
	if err != nil {
		t.Fatalf("new box: %v", err)
	}
	second, err := New(strings.Repeat("k", 32), "kinstack/test/v1", 2)
	if err != nil {
		t.Fatalf("new box: %v", err)
	}
	payload, err := first.Seal([]byte("secret"))
	if err != nil {
		t.Fatalf("seal: %v", err)
	}
	if _, err := second.Open(payload); !errors.Is(err, ErrMalformed) {
		t.Fatalf("expected ErrMalformed across versions, got %v", err)
	}
}

func TestNewValidatesConfiguration(t *testing.T) {
	if _, err := New("short", "kinstack/test/v1", 1); err == nil {
		t.Fatal("expected short material to be rejected")
	}
	if _, err := New(strings.Repeat("k", 32), "", 1); err == nil {
		t.Fatal("expected empty info to be rejected")
	}
	if _, err := New(strings.Repeat("k", 32), "kinstack/test/v1", 0); err == nil {
		t.Fatal("expected a zero version to be rejected")
	}
}
