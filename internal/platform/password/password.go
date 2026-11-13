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

type Cost struct {
	Memory      uint32
	Iterations  uint32
	Parallelism uint8
}

var DefaultCost = Cost{Memory: 64 * 1024, Iterations: 3, Parallelism: 2}

// OneTimeCost is a lighter work factor for short-lived one-time secrets such
// as recovery codes, where verification may have to try several stored hashes
// per attempt.
var OneTimeCost = Cost{Memory: 32 * 1024, Iterations: 2, Parallelism: 1}

func (c Cost) valid() bool {
	return c.Memory > 0 && c.Iterations > 0 && c.Parallelism > 0
}

var ErrInvalidPasswordHash = errors.New("invalid argon2id password hash")

// Compare accepts work factors only within these bounds: a corrupted or
// tampered stored hash must not turn one login attempt into an unbounded
// allocation. The defaults and the one-time cost both fit well inside them.
const (
	memoryKiBMax   = 256 * 1024
	iterationsMax  = 10
	parallelismMax = 8
)

var dummyHash = func() string {
	salt := []byte("kinstack-dummy-salt")
	key := argon2.IDKey(
		[]byte("kinstack-invalid-credential"),
		salt,
		DefaultCost.Iterations,
		DefaultCost.Memory,
		DefaultCost.Parallelism,
		32,
	)
	return fmt.Sprintf(
		"$argon2id$v=19$m=%d,t=%d,p=%d$%s$%s",
		DefaultCost.Memory,
		DefaultCost.Iterations,
		DefaultCost.Parallelism,
		base64.RawStdEncoding.EncodeToString(salt),
		base64.RawStdEncoding.EncodeToString(key),
	)
}()

func Hash(password string) (string, error) {
	return HashWithCost(password, DefaultCost)
}

func HashWithCost(secret string, cost Cost) (string, error) {
	if !cost.valid() {
		return "", errors.New("invalid argon2id cost")
	}
	salt := make([]byte, 16)
	if _, err := rand.Read(salt); err != nil {
		return "", err
	}
	key := argon2.IDKey(
		[]byte(secret),
		salt,
		cost.Iterations,
		cost.Memory,
		cost.Parallelism,
		32,
	)
	return fmt.Sprintf(
		"$argon2id$v=19$m=%d,t=%d,p=%d$%s$%s",
		cost.Memory,
		cost.Iterations,
		cost.Parallelism,
		base64.RawStdEncoding.EncodeToString(salt),
		base64.RawStdEncoding.EncodeToString(key),
	), nil
}

func Compare(encoded, password string) error {
	parsed, err := parseArgon2id(encoded)
	if err != nil {
		return err
	}
	actual := argon2.IDKey(
		[]byte(password),
		parsed.salt,
		uint32(parsed.iterations),
		uint32(parsed.memory),
		uint8(parsed.parallelism),
		uint32(len(parsed.expected)),
	)
	if subtle.ConstantTimeCompare(actual, parsed.expected) != 1 {
		return ErrInvalidPasswordHash
	}
	return nil
}

// parsedHash has bounded every field, so a hostile hash cannot ask for unbounded work.
type parsedHash struct {
	memory      uint64
	iterations  uint64
	parallelism uint64
	salt        []byte
	expected    []byte
}

func parseArgon2id(encoded string) (parsedHash, error) {
	parts := strings.Split(encoded, "$")
	if len(parts) != 6 || parts[1] != "argon2id" || parts[2] != "v=19" {
		return parsedHash{}, ErrInvalidPasswordHash
	}
	parsed, err := parseArgon2Params(parts[3])
	if err != nil {
		return parsedHash{}, err
	}
	if parsed.salt, err = parseArgon2Field(parts[4], 8); err != nil {
		return parsedHash{}, err
	}
	if parsed.expected, err = parseArgon2Field(parts[5], 1); err != nil {
		return parsedHash{}, err
	}
	return parsed, nil
}

// parseArgon2Params rejects values outside the ceilings, so a stored hash cannot become a
// CPU or memory attack.
func parseArgon2Params(raw string) (parsedHash, error) {
	values := make(map[string]string, 3)
	for _, item := range strings.Split(raw, ",") {
		key, value, ok := strings.Cut(item, "=")
		if !ok {
			return parsedHash{}, ErrInvalidPasswordHash
		}
		values[key] = value
	}
	memory, err := boundedParam(values["m"], memoryKiBMax)
	if err != nil {
		return parsedHash{}, err
	}
	iterations, err := boundedParam(values["t"], iterationsMax)
	if err != nil {
		return parsedHash{}, err
	}
	parallelism, err := boundedParam(values["p"], parallelismMax)
	if err != nil {
		return parsedHash{}, err
	}
	return parsedHash{memory: memory, iterations: iterations, parallelism: parallelism}, nil
}

func boundedParam(raw string, ceiling uint64) (uint64, error) {
	value, err := strconv.ParseUint(raw, 10, 32)
	if err != nil || value == 0 || value > ceiling {
		return 0, ErrInvalidPasswordHash
	}
	return value, nil
}

func parseArgon2Field(raw string, minimum int) ([]byte, error) {
	decoded, err := base64.RawStdEncoding.DecodeString(raw)
	if err != nil || len(decoded) < minimum {
		return nil, ErrInvalidPasswordHash
	}
	return decoded, nil
}

// DummyHash keeps unknown-user timing consistent; callers burn the comparison and discard
// the result.
func DummyHash() string {
	return dummyHash
}
