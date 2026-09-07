package push

import (
	"bytes"
	"context"
	"encoding/base64"
	"testing"
)

type memoryStore struct {
	device  EncryptedDevice
	revokedSession string
}

func (m *memoryStore) UpsertAndroid(_ context.Context, d EncryptedDevice) error { m.device = d; return nil }
func (m *memoryStore) RevokeInstallation(context.Context, string, string) error { return nil }
func (m *memoryStore) RevokeSession(_ context.Context, sessionID string) error { m.revokedSession = sessionID; return nil }
func (m *memoryStore) ActiveTokens(_ context.Context, userID string) ([]StoredToken, error) {
	if m.device.UserID != userID { return nil, nil }
	return []StoredToken{{InstallationID:m.device.InstallationID, Ciphertext:m.device.Ciphertext, Nonce:m.device.Nonce, KeyID:m.device.KeyID}}, nil
}

type memorySender struct { token string; message Message }
func (m *memorySender) Send(_ context.Context, token string, message Message) error { m.token = token; m.message = message; return nil }

func testKey() string { return base64.StdEncoding.EncodeToString(bytes.Repeat([]byte{7}, 32)) }

func TestRegisterEncryptsTokenAndNotifyDecryptsIt(t *testing.T) {
	store := &memoryStore{}
	sender := &memorySender{}
	svc, err := NewService(store, "production-v1", testKey(), sender)
	if err != nil { t.Fatal(err) }
	const token = "fcm-registration-token-that-is-not-plaintext"
	const installation = "11111111-2222-4333-8444-555555555555"
	if err := svc.RegisterAndroid(context.Background(), "user-1", "session-1", installation, token, "1.0.0"); err != nil { t.Fatal(err) }
	if string(store.device.Ciphertext) == token { t.Fatal("token stored as plaintext") }
	if bytes.Contains(store.device.Ciphertext, []byte(token)) { t.Fatal("ciphertext contains plaintext token") }
	if len(store.device.TokenHash) != 32 || len(store.device.Nonce) != 12 { t.Fatalf("unexpected encrypted material: hash=%d nonce=%d", len(store.device.TokenHash), len(store.device.Nonce)) }
	if err := svc.NotifyUser(context.Background(), "user-1", Message{Title:"Approved", Body:"Your request was approved"}); err != nil { t.Fatal(err) }
	if sender.token != token { t.Fatalf("sender token mismatch: %q", sender.token) }
}

func TestRegisterRejectsBadInstallationAndToken(t *testing.T) {
	svc, err := NewService(&memoryStore{}, "production-v1", testKey(), nil)
	if err != nil { t.Fatal(err) }
	if err := svc.RegisterAndroid(context.Background(), "u", "s", "not-a-uuid", "long-enough-registration-token", "1"); err != ErrInvalidInstallation { t.Fatalf("got %v", err) }
	if err := svc.RegisterAndroid(context.Background(), "u", "s", "11111111-2222-4333-8444-555555555555", "short", "1"); err != ErrInvalidRegistration { t.Fatalf("got %v", err) }
}

func TestRevokeSession(t *testing.T) {
	store := &memoryStore{}
	svc, err := NewService(store, "production-v1", testKey(), nil)
	if err != nil { t.Fatal(err) }
	if err := svc.RevokeSession(context.Background(), "session-1"); err != nil { t.Fatal(err) }
	if store.revokedSession != "session-1" { t.Fatalf("session not revoked: %q", store.revokedSession) }
}
