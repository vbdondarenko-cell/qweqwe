package push

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"

	"github.com/vbdondarenko-cell/qweqwe/backend/internal/identifier"
)

var (
	ErrInvalidRegistration = errors.New("invalid push registration")
	ErrInvalidInstallation = errors.New("invalid installation id")
)

type EncryptedDevice struct {
	ID             string
	UserID         string
	SessionID      string
	InstallationID string
	TokenHash      []byte
	Ciphertext     []byte
	Nonce          []byte
	KeyID          string
	AppVersion     string
}

type StoredToken struct {
	Ciphertext []byte
	Nonce      []byte
	KeyID      string
}

type Store interface {
	UpsertAndroid(context.Context, EncryptedDevice) error
	RevokeInstallation(context.Context, string, string) error
	RevokeSession(context.Context, string) error
	ActiveTokens(context.Context, string) ([]StoredToken, error)
}

type Sender interface {
	Send(context.Context, string, Message) error
}

type Message struct {
	Title string
	Body  string
	Data  map[string]string
}

type Service struct {
	store  Store
	keyID  string
	aead   cipher.AEAD
	sender Sender
}

func NewService(store Store, keyID, keyBase64 string, sender Sender) (*Service, error) {
	keyID = strings.TrimSpace(keyID)
	if store == nil || keyID == "" || len(keyID) > 64 {
		return nil, errors.New("push store and key id are required")
	}
	key, err := base64.StdEncoding.DecodeString(strings.TrimSpace(keyBase64))
	if err != nil || len(key) != 32 {
		return nil, errors.New("push token key must be 32-byte base64")
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	return &Service{store: store, keyID: keyID, aead: aead, sender: sender}, nil
}

func (s *Service) RegisterAndroid(ctx context.Context, userID, sessionID, installationID, token, appVersion string) error {
	installationID = strings.ToLower(strings.TrimSpace(installationID))
	token = strings.TrimSpace(token)
	appVersion = strings.TrimSpace(appVersion)
	if !validUUID(installationID) {
		return ErrInvalidInstallation
	}
	if len(token) < 16 || len(token) > 4096 || len(appVersion) > 64 {
		return ErrInvalidRegistration
	}
	id, err := identifier.NewUUID()
	if err != nil {
		return err
	}
	nonce := make([]byte, s.aead.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return fmt.Errorf("push nonce entropy: %w", err)
	}
	ciphertext := s.aead.Seal(nil, nonce, []byte(token), []byte(userID+"|"+installationID+"|"+s.keyID))
	hash := sha256.Sum256([]byte(token))
	return s.store.UpsertAndroid(ctx, EncryptedDevice{
		ID: id, UserID: userID, SessionID: sessionID, InstallationID: installationID,
		TokenHash: hash[:], Ciphertext: ciphertext, Nonce: nonce, KeyID: s.keyID, AppVersion: appVersion,
	})
}

func (s *Service) RevokeInstallation(ctx context.Context, userID, installationID string) error {
	installationID = strings.ToLower(strings.TrimSpace(installationID))
	if !validUUID(installationID) {
		return ErrInvalidInstallation
	}
	return s.store.RevokeInstallation(ctx, userID, installationID)
}

func (s *Service) RevokeSession(ctx context.Context, sessionID string) error {
	if strings.TrimSpace(sessionID) == "" {
		return nil
	}
	return s.store.RevokeSession(ctx, sessionID)
}

func (s *Service) NotifyUser(ctx context.Context, userID string, message Message) error {
	if s.sender == nil {
		return nil
	}
	tokens, err := s.store.ActiveTokens(ctx, userID)
	if err != nil {
		return err
	}
	var firstErr error
	for _, stored := range tokens {
		if stored.KeyID != s.keyID || len(stored.Nonce) != s.aead.NonceSize() {
			if firstErr == nil { firstErr = errors.New("push token key mismatch") }
			continue
		}
		plain, err := s.aead.Open(nil, stored.Nonce, stored.Ciphertext, []byte(userID+"|"+""+"|"+s.keyID))
		if err != nil {
			// Legacy rows cannot be decrypted without their installation id in AAD.
			if firstErr == nil { firstErr = err }
			continue
		}
		if err := s.sender.Send(ctx, string(plain), message); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	return firstErr
}

func validUUID(v string) bool {
	if len(v) != 36 || v[8] != '-' || v[13] != '-' || v[18] != '-' || v[23] != '-' {
		return false
	}
	for i, r := range v {
		if i == 8 || i == 13 || i == 18 || i == 23 { continue }
		if !((r >= '0' && r <= '9') || (r >= 'a' && r <= 'f')) { return false }
	}
	return true
}
