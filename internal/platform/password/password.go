package password

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"golang.org/x/crypto/argon2"
)

const (
	argon2Memory      uint32 = 64 * 1024
	argon2Iterations  uint32 = 3
	argon2Parallelism uint8  = 2
)

var ErrInvalidPasswordHash = errors.New("invalid argon2id password hash")

var dummyHash = func() string {
	salt := []byte("kinstack-dummy-salt")
	key := argon2.IDKey(
		[]byte("kinstack-invalid-credential"),
		salt,
		argon2Iterations,
		argon2Memory,
		argon2Parallelism,
		32,
	)
	return fmt.Sprintf(
		"$argon2id$v=19$m=%d,t=%d,p=%d$%s$%s",
		argon2Memory,
		argon2Iterations,
		argon2Parallelism,
		base64.RawStdEncoding.EncodeToString(salt),
		base64.RawStdEncoding.EncodeToString(key),
	)
}()

// Hash generates an Argon2id password hash.
func Hash(password string) (string, error) {
	salt := make([]byte, 16)
	if _, err := rand.Read(salt); err != nil {
		return "", err
	}
	key := argon2.IDKey(
		[]byte(password),
		salt,
		argon2Iterations,
		argon2Memory,
		argon2Parallelism,
		32,
	)
	return fmt.Sprintf(
		"$argon2id$v=19$m=%d,t=%d,p=%d$%s$%s",
		argon2Memory,
		argon2Iterations,
		argon2Parallelism,
		base64.RawStdEncoding.EncodeToString(salt),
		base64.RawStdEncoding.EncodeToString(key),
	), nil
}

// Compare verifies a password against an Argon2id password hash.
func Compare(encoded, password string) error {
	parts := strings.Split(encoded, "$")
	if len(parts) != 6 || parts[1] != "argon2id" || parts[2] != "v=19" {
		return ErrInvalidPasswordHash
	}
	params := make(map[string]string, 3)
	for _, item := range strings.Split(parts[3], ",") {
		key, value, ok := strings.Cut(item, "=")
		if !ok {
			return ErrInvalidPasswordHash
		}
		params[key] = value
	}
	memory, err := strconv.ParseUint(params["m"], 10, 32)
	if err != nil || memory == 0 {
		return ErrInvalidPasswordHash
	}
	iterations, err := strconv.ParseUint(params["t"], 10, 32)
	if err != nil || iterations == 0 {
		return ErrInvalidPasswordHash
	}
	parallelism, err := strconv.ParseUint(params["p"], 10, 8)
	if err != nil || parallelism == 0 {
		return ErrInvalidPasswordHash
	}
	salt, err := base64.RawStdEncoding.DecodeString(parts[4])
	if err != nil || len(salt) < 8 {
		return ErrInvalidPasswordHash
	}
	expected, err := base64.RawStdEncoding.DecodeString(parts[5])
	if err != nil || len(expected) == 0 {
		return ErrInvalidPasswordHash
	}
	actual := argon2.IDKey(
		[]byte(password),
		salt,
		uint32(iterations),
		uint32(memory),
		uint8(parallelism),
		uint32(len(expected)),
	)
	if subtle.ConstantTimeCompare(actual, expected) != 1 {
		return ErrInvalidPasswordHash
	}
	return nil
}

// DummyHash returns a stable hash used to keep unknown-user authentication timing consistent.
func DummyHash() string {
	return dummyHash
}
