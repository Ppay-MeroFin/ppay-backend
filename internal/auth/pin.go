package auth

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"

	"golang.org/x/crypto/argon2"
)

const (
	argonTime    uint32 = 2
	argonMemory  uint32 = 19 * 1024
	argonThreads uint8  = 1
	argonKeyLen  uint32 = 32
)

var ErrInvalidPINHash = errors.New("invalid PIN hash format")

func ValidatePIN(pin string) bool {
	if len(pin) != 4 {
		return false
	}

	for _, r := range pin {
		if r < '0' || r > '9' {
			return false
		}
	}

	return true
}

func HashPIN(pin string) (string, error) {
	if !ValidatePIN(pin) {
		return "", errors.New("PIN must contain exactly four digits")
	}

	salt := make([]byte, 16)
	if _, err := rand.Read(salt); err != nil {
		return "", fmt.Errorf("generate PIN salt: %w", err)
	}

	hash := argon2.IDKey(
		[]byte(pin),
		salt,
		argonTime,
		argonMemory,
		argonThreads,
		argonKeyLen,
	)

	return fmt.Sprintf(
		"$argon2id$v=19$m=%d,t=%d,p=%d$%s$%s",
		argonMemory,
		argonTime,
		argonThreads,
		base64.RawStdEncoding.EncodeToString(salt),
		base64.RawStdEncoding.EncodeToString(hash),
	), nil
}

func VerifyPIN(encodedHash, pin string) (bool, error) {
	if !ValidatePIN(pin) {
		return false, nil
	}

	parts := strings.Split(encodedHash, "$")
	if len(parts) != 6 || parts[1] != "argon2id" || parts[2] != "v=19" {
		return false, ErrInvalidPINHash
	}

	var memory, iterations uint32
	var threads uint8

	if _, err := fmt.Sscanf(
		parts[3],
		"m=%d,t=%d,p=%d",
		&memory,
		&iterations,
		&threads,
	); err != nil {
		return false, ErrInvalidPINHash
	}

	salt, err := base64.RawStdEncoding.DecodeString(parts[4])
	if err != nil {
		return false, ErrInvalidPINHash
	}

	expectedHash, err := base64.RawStdEncoding.DecodeString(parts[5])
	if err != nil {
		return false, ErrInvalidPINHash
	}

	actualHash := argon2.IDKey(
		[]byte(pin),
		salt,
		iterations,
		memory,
		threads,
		uint32(len(expectedHash)),
	)

	return subtle.ConstantTimeCompare(actualHash, expectedHash) == 1, nil
}
