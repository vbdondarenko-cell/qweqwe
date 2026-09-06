package password

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"

	"golang.org/x/crypto/argon2"
)

var (
	ErrInvalidHash     = errors.New("invalid password hash")
	ErrInvalidPassword = errors.New("password must be 8 to 1024 bytes")
)

type Params struct {
	MemoryKiB  uint32
	Iterations uint32
	Parallel   uint8
	SaltBytes  uint32
	KeyBytes   uint32
}

func OWASPMinimum() Params {
	return Params{MemoryKiB: 19 * 1024, Iterations: 2, Parallel: 1, SaltBytes: 16, KeyBytes: 32}
}

func (p Params) Validate() error {
	if p.MemoryKiB < 19*1024 || p.Iterations < 2 || p.Parallel < 1 || p.SaltBytes < 16 || p.KeyBytes < 32 {
		return errors.New("argon2id parameters are below the project security floor")
	}
	return nil
}

func Hash(raw string, p Params) (string, error) {
	if len(raw) < 8 || len(raw) > 1024 {
		return "", ErrInvalidPassword
	}
	if err := p.Validate(); err != nil {
		return "", err
	}
	salt := make([]byte, p.SaltBytes)
	if _, err := rand.Read(salt); err != nil {
		return "", fmt.Errorf("password salt entropy: %w", err)
	}
	key := argon2.IDKey([]byte(raw), salt, p.Iterations, p.MemoryKiB, p.Parallel, p.KeyBytes)
	return fmt.Sprintf("$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s", argon2.Version, p.MemoryKiB, p.Iterations, p.Parallel, base64.RawStdEncoding.EncodeToString(salt), base64.RawStdEncoding.EncodeToString(key)), nil
}

func Verify(encoded, raw string) (bool, error) {
	parts := strings.Split(encoded, "$")
	if len(parts) != 6 || parts[1] != "argon2id" {
		return false, ErrInvalidHash
	}
	var version int
	if _, err := fmt.Sscanf(parts[2], "v=%d", &version); err != nil || version != argon2.Version {
		return false, ErrInvalidHash
	}
	var p Params
	if _, err := fmt.Sscanf(parts[3], "m=%d,t=%d,p=%d", &p.MemoryKiB, &p.Iterations, &p.Parallel); err != nil {
		return false, ErrInvalidHash
	}
	salt, err := base64.RawStdEncoding.DecodeString(parts[4])
	if err != nil || len(salt) < 16 {
		return false, ErrInvalidHash
	}
	expected, err := base64.RawStdEncoding.DecodeString(parts[5])
	if err != nil || len(expected) < 32 {
		return false, ErrInvalidHash
	}
	p.SaltBytes = uint32(len(salt))
	p.KeyBytes = uint32(len(expected))
	if err := p.Validate(); err != nil {
		return false, ErrInvalidHash
	}
	actual := argon2.IDKey([]byte(raw), salt, p.Iterations, p.MemoryKiB, p.Parallel, p.KeyBytes)
	return subtle.ConstantTimeCompare(actual, expected) == 1, nil
}
