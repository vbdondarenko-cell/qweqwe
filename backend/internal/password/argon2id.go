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

const (
	minMemoryKiB  uint32 = 19 * 1024
	maxMemoryKiB  uint32 = 256 * 1024
	minIterations uint32 = 2
	maxIterations uint32 = 10
	minParallel   uint8  = 1
	maxParallel   uint8  = 8
	minSaltBytes  uint32 = 16
	maxSaltBytes  uint32 = 64
	minKeyBytes   uint32 = 32
	maxKeyBytes   uint32 = 64
)

type Params struct {
	MemoryKiB  uint32
	Iterations uint32
	Parallel   uint8
	SaltBytes  uint32
	KeyBytes   uint32
}

func OWASPMinimum() Params {
	return Params{MemoryKiB: minMemoryKiB, Iterations: minIterations, Parallel: minParallel, SaltBytes: minSaltBytes, KeyBytes: minKeyBytes}
}

func (p Params) Validate() error {
	if p.MemoryKiB < minMemoryKiB || p.Iterations < minIterations || p.Parallel < minParallel || p.SaltBytes < minSaltBytes || p.KeyBytes < minKeyBytes {
		return errors.New("argon2id parameters are below the project security floor")
	}
	if p.MemoryKiB > maxMemoryKiB || p.Iterations > maxIterations || p.Parallel > maxParallel || p.SaltBytes > maxSaltBytes || p.KeyBytes > maxKeyBytes {
		return errors.New("argon2id parameters exceed the project safety ceiling")
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
	if len(raw) < 8 || len(raw) > 1024 {
		return false, ErrInvalidPassword
	}
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
	if err != nil || len(salt) < int(minSaltBytes) || len(salt) > int(maxSaltBytes) {
		return false, ErrInvalidHash
	}
	expected, err := base64.RawStdEncoding.DecodeString(parts[5])
	if err != nil || len(expected) < int(minKeyBytes) || len(expected) > int(maxKeyBytes) {
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
