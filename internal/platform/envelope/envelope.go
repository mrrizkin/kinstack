// Package envelope encrypts small secrets at rest with AES-256-GCM. Keys are
// derived from one configured secret material with HKDF so the same material
// can serve several purposes without sharing a key.
package envelope

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/hkdf"
	"crypto/rand"
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"strings"
)

const (
	// KeySize is the AES-256 key length.
	KeySize   = 32
	nonceSize = 12
)

// ErrMalformed is returned when a ciphertext cannot be opened.
var ErrMalformed = errors.New("invalid encrypted payload")

// Box seals and opens secrets with one derived key.
type Box struct {
	aead    cipher.AEAD
	version int
}

// New derives an encryption key from material using HKDF-SHA256. The version
// is stored next to every ciphertext so keys can be rotated later.
func New(material, info string, version int) (*Box, error) {
	if len(strings.TrimSpace(material)) < KeySize {
		return nil, fmt.Errorf("envelope key material must be at least %d bytes", KeySize)
	}
	if strings.TrimSpace(info) == "" {
		return nil, errors.New("envelope key info is required")
	}
	if version <= 0 {
		return nil, errors.New("envelope key version must be positive")
	}
	key, err := hkdf.Key(sha256.New, []byte(material), nil, info, KeySize)
	if err != nil {
		return nil, fmt.Errorf("derive envelope key: %w", err)
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("create envelope cipher: %w", err)
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("create envelope aead: %w", err)
	}
	return &Box{aead: aead, version: version}, nil
}

// Version reports the key version this box writes.
func (b *Box) Version() int {
	if b == nil {
		return 0
	}
	return b.version
}

// Seal encrypts plaintext and returns nonce||ciphertext.
func (b *Box) Seal(plaintext []byte) ([]byte, error) {
	if b == nil {
		return nil, errors.New("envelope box is nil")
	}
	nonce := make([]byte, nonceSize)
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, fmt.Errorf("read envelope nonce: %w", err)
	}
	return b.aead.Seal(nonce, nonce, plaintext, []byte(b.additionalData())), nil
}

// Open decrypts a payload produced by Seal.
func (b *Box) Open(payload []byte) ([]byte, error) {
	if b == nil {
		return nil, errors.New("envelope box is nil")
	}
	if len(payload) <= nonceSize {
		return nil, ErrMalformed
	}
	nonce, ciphertext := payload[:nonceSize], payload[nonceSize:]
	plaintext, err := b.aead.Open(nil, nonce, ciphertext, []byte(b.additionalData()))
	if err != nil {
		return nil, ErrMalformed
	}
	return plaintext, nil
}

// additionalData binds a ciphertext to its key version.
func (b *Box) additionalData() string {
	return fmt.Sprintf("kinstack/envelope/v%d", b.version)
}
