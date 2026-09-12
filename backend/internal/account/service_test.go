package account

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/vbdondarenko-cell/qweqwe/backend/internal/password"
)

type memoryStore struct {
	user                        UserWithPassword
	session                     Session
	revoked                     bool
	reset                       PasswordReset
	resetUsed                   bool
	rotatePasswordBeforeSession bool
}

func (m *memoryStore) Register(_ context.Context, u User, h string, s Session) error {
	if m.user.ID != "" {
		return ErrConflict
	}
	m.user = UserWithPassword{User: u, PasswordHash: h}
	m.session = s
	return nil
}

func (m *memoryStore) FindByLogin(_ context.Context, id string) (UserWithPassword, error) {
	if m.user.ID == "" || (id != m.user.Email && id != m.user.Username) {
		return UserWithPassword{}, ErrNotFound
	}
	return m.user, nil
}

func (m *memoryStore) CreateSession(_ context.Context, s Session, expectedPasswordHash string) error {
	if m.rotatePasswordBeforeSession {
		m.user.PasswordHash = "rotated-password-hash"
		m.rotatePasswordBeforeSession = false
	}
	if m.user.PasswordHash != expectedPasswordHash {
		return ErrUnauthorized
	}
	m.session = s
	m.revoked = false
	return nil
}

func (m *memoryStore) Authenticate(_ context.Context, h []byte, now time.Time) (User, string, error) {
	if m.revoked || !m.session.ExpiresAt.After(now) || string(h) != string(m.session.TokenHash) {
		return User{}, "", ErrUnauthorized
	}
	return m.user.User, m.session.ID, nil
}

func (m *memoryStore) RevokeSession(_ context.Context, h []byte, _ time.Time) error {
	if string(h) != string(m.session.TokenHash) {
		return ErrNotFound
	}
	m.revoked = true
	return nil
}

func (m *memoryStore) UpdateProfile(_ context.Context, id string, p ProfilePatch, now time.Time) (User, error) {
	if id != m.user.ID {
		return User{}, ErrNotFound
	}
	if p.DisplayName != nil {
		m.user.DisplayName = *p.DisplayName
	}
	if p.ProfileVisibility != nil {
		m.user.ProfileVisibility = *p.ProfileVisibility
	}
	if p.Language != nil {
		m.user.Language = *p.Language
	}
	if p.Interests != nil {
		m.user.Interests = *p.Interests
	}
	m.user.UpdatedAt = now
	return m.user.User, nil
}

func (m *memoryStore) CreatePasswordReset(_ context.Context, reset PasswordReset, _ time.Time) error {
	m.reset = reset
	m.resetUsed = false
	return nil
}

func (m *memoryStore) ResetPassword(_ context.Context, h []byte, newHash string, now time.Time) error {
	if m.resetUsed || string(h) != string(m.reset.TokenHash) || !m.reset.ExpiresAt.After(now) {
		return ErrUnauthorized
	}
	m.resetUsed = true
	m.user.PasswordHash = newHash
	m.revoked = true
	return nil
}

type captureNotifier struct {
	email     string
	token     string
	expiresAt time.Time
}

func (n *captureNotifier) SendPasswordReset(_ context.Context, email, token string, expiresAt time.Time) error {
	n.email = email
	n.token = token
	n.expiresAt = expiresAt
	return nil
}

func TestRegisterAuthenticateLogout(t *testing.T) {
	store := &memoryStore{}
	svc, err := NewService(store, password.OWASPMinimum(), time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	out, err := svc.Register(context.Background(), Registration{Email: "A@Example.com", Username: "Alice_1", DisplayName: "Alice", Password: "correct horse battery staple", Language: "uk"})
	if err != nil {
		t.Fatal(err)
	}
	if out.User.Email != "a@example.com" || out.User.Username != "alice_1" {
		t.Fatalf("normalization failed: %#v", out.User)
	}
	if _, _, err := svc.Authenticate(context.Background(), out.Token); err != nil {
		t.Fatal(err)
	}
	if err := svc.Logout(context.Background(), out.Token); err != nil {
		t.Fatal(err)
	}
	if _, _, err := svc.Authenticate(context.Background(), out.Token); !errors.Is(err, ErrUnauthorized) {
		t.Fatalf("expected revoked session, got %v", err)
	}
}

func TestLoginRejectsWrongPassword(t *testing.T) {
	store := &memoryStore{}
	svc, _ := NewService(store, password.OWASPMinimum(), time.Hour)
	_, err := svc.Register(context.Background(), Registration{Email: "a@example.com", Username: "alice", DisplayName: "Alice", Password: "correct horse battery staple"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Login(context.Background(), Login{Identifier: "alice", Password: "wrong password"}); !errors.Is(err, ErrUnauthorized) {
		t.Fatalf("expected unauthorized, got %v", err)
	}
}

func TestLoginRejectsSessionWhenPasswordChangesAfterVerification(t *testing.T) {
	store := &memoryStore{}
	svc, _ := NewService(store, password.OWASPMinimum(), time.Hour)
	_, err := svc.Register(context.Background(), Registration{Email: "a@example.com", Username: "alice", DisplayName: "Alice", Password: "correct horse battery staple"})
	if err != nil {
		t.Fatal(err)
	}
	store.rotatePasswordBeforeSession = true
	if _, err := svc.Login(context.Background(), Login{Identifier: "alice", Password: "correct horse battery staple"}); !errors.Is(err, ErrUnauthorized) {
		t.Fatalf("stale verified password must not create a session, got %v", err)
	}
}

func TestPasswordResetRevokesSessionsAndChangesPassword(t *testing.T) {
	store := &memoryStore{}
	svc, err := NewService(store, password.OWASPMinimum(), time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	notifier := &captureNotifier{}
	if err := svc.ConfigureRecovery(notifier, 15*time.Minute); err != nil {
		t.Fatal(err)
	}
	registered, err := svc.Register(context.Background(), Registration{Email: "a@example.com", Username: "alice", DisplayName: "Alice", Password: "old correct horse battery staple"})
	if err != nil {
		t.Fatal(err)
	}
	if err := svc.BeginPasswordReset(context.Background(), "A@example.com"); err != nil {
		t.Fatal(err)
	}
	if notifier.email != "a@example.com" || notifier.token == "" {
		t.Fatalf("recovery notification not captured: %#v", notifier)
	}
	if err := svc.ResetPassword(context.Background(), notifier.token, "new correct horse battery staple"); err != nil {
		t.Fatal(err)
	}
	if _, _, err := svc.Authenticate(context.Background(), registered.Token); !errors.Is(err, ErrUnauthorized) {
		t.Fatalf("old session should be revoked, got %v", err)
	}
	if _, err := svc.Login(context.Background(), Login{Identifier: "alice", Password: "old correct horse battery staple"}); !errors.Is(err, ErrUnauthorized) {
		t.Fatalf("old password should fail, got %v", err)
	}
	if _, err := svc.Login(context.Background(), Login{Identifier: "alice", Password: "new correct horse battery staple"}); err != nil {
		t.Fatalf("new password should work: %v", err)
	}
	if err := svc.ResetPassword(context.Background(), notifier.token, "another correct horse battery staple"); !errors.Is(err, ErrUnauthorized) {
		t.Fatalf("reset token must be one-time, got %v", err)
	}
}

func TestUpdateProfileNormalizesInterests(t *testing.T) {
	store := &memoryStore{}
	svc, err := NewService(store, password.OWASPMinimum(), time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	registered, err := svc.Register(context.Background(), Registration{Email: "a@example.com", Username: "alice", DisplayName: "Alice", Password: "correct horse battery staple"})
	if err != nil {
		t.Fatal(err)
	}
	interests := []string{"Hiking", " board games ", "hiking"}
	out, err := svc.UpdateProfile(context.Background(), registered.User.ID, ProfilePatch{Interests: &interests})
	if err != nil {
		t.Fatal(err)
	}
	if got := out.Interests; len(got) != 2 || got[0] != "board games" || got[1] != "hiking" {
		t.Fatalf("interests = %v", got)
	}

	// A non-nil empty slice clears interests; nil leaves them untouched.
	empty := []string{}
	out, err = svc.UpdateProfile(context.Background(), registered.User.ID, ProfilePatch{Interests: &empty})
	if err != nil {
		t.Fatal(err)
	}
	if len(out.Interests) != 0 {
		t.Fatalf("expected cleared interests, got %v", out.Interests)
	}
}

func TestUpdateProfileRejectsBlankInterestTag(t *testing.T) {
	store := &memoryStore{}
	svc, err := NewService(store, password.OWASPMinimum(), time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	registered, err := svc.Register(context.Background(), Registration{Email: "a@example.com", Username: "alice", DisplayName: "Alice", Password: "correct horse battery staple"})
	if err != nil {
		t.Fatal(err)
	}
	blank := []string{"  "}
	if _, err := svc.UpdateProfile(context.Background(), registered.User.ID, ProfilePatch{Interests: &blank}); !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got %v", err)
	}
}

func TestUpdateProfileRejectsTooManyInterests(t *testing.T) {
	store := &memoryStore{}
	svc, err := NewService(store, password.OWASPMinimum(), time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	registered, err := svc.Register(context.Background(), Registration{Email: "a@example.com", Username: "alice", DisplayName: "Alice", Password: "correct horse battery staple"})
	if err != nil {
		t.Fatal(err)
	}
	many := make([]string, maxInterests+1)
	for i := range many {
		many[i] = string(rune('a' + i))
	}
	if _, err := svc.UpdateProfile(context.Background(), registered.User.ID, ProfilePatch{Interests: &many}); !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got %v", err)
	}
}

func TestPasswordResetRequestDoesNotRevealUnknownAccount(t *testing.T) {
	store := &memoryStore{}
	svc, err := NewService(store, password.OWASPMinimum(), time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	notifier := &captureNotifier{}
	if err := svc.ConfigureRecovery(notifier, 15*time.Minute); err != nil {
		t.Fatal(err)
	}
	if err := svc.BeginPasswordReset(context.Background(), "nobody@example.com"); err != nil {
		t.Fatal(err)
	}
	if notifier.token != "" {
		t.Fatal("unknown account must not send a reset token")
	}
}
