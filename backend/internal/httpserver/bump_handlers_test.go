package httpserver

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/vbdondarenko-cell/qweqwe/backend/internal/account"
	"github.com/vbdondarenko-cell/qweqwe/backend/internal/bump"
)

type fakeBumpStore struct {
	challenge       bump.Challenge
	challengeErr    error
	confirmResult   bump.ConfirmResult
	confirmErr      error
	reliability     bump.Reliability
	reliabilityErr  error
	vault           []bump.VaultEntry
	vaultErr        error
	publicBand      bump.Band
	publicBandErr   error
	lastConfirmArgs [4]string
	lastBandArgs    [2]string
}

func (f *fakeBumpStore) IssueChallenge(_ context.Context, _, _ string, _ time.Duration) (bump.Challenge, error) {
	return f.challenge, f.challengeErr
}

func (f *fakeBumpStore) Confirm(_ context.Context, actorID, slotID, counterpartID, nonce string) (bump.ConfirmResult, error) {
	f.lastConfirmArgs = [4]string{actorID, slotID, counterpartID, nonce}
	return f.confirmResult, f.confirmErr
}

func (f *fakeBumpStore) Reliability(_ context.Context, _ string) (bump.Reliability, error) {
	return f.reliability, f.reliabilityErr
}

func (f *fakeBumpStore) Vault(_ context.Context, _ string, _ int) ([]bump.VaultEntry, error) {
	return f.vault, f.vaultErr
}

func (f *fakeBumpStore) PublicBand(_ context.Context, viewerID, targetUserID string) (bump.Band, error) {
	f.lastBandArgs = [2]string{viewerID, targetUserID}
	return f.publicBand, f.publicBandErr
}

func requestWithBumpAuth(method, target, body string) *http.Request {
	var reader *bytes.Reader
	if body != "" {
		reader = bytes.NewReader([]byte(body))
	} else {
		reader = bytes.NewReader(nil)
	}
	request := httptest.NewRequest(method, target, reader)
	request.SetPathValue("slotID", "slot-1")
	ctx := context.WithValue(request.Context(), authKey, authContext{User: account.User{ID: "11111111-1111-1111-1111-111111111111"}})
	return request.WithContext(ctx)
}

func TestIssueBumpChallengeReturnsNonce(t *testing.T) {
	store := &fakeBumpStore{challenge: bump.Challenge{Nonce: "nonce-1", SlotID: "slot-1", ExpiresAt: time.Unix(1000, 0)}}
	svc, err := bump.NewService(store, time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	server := &Server{deps: Dependencies{Bump: svc}}
	recorder := httptest.NewRecorder()
	server.issueBumpChallenge(recorder, requestWithBumpAuth(http.MethodPost, "/v1/slots/slot-1/bump/challenge", ""))
	if recorder.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", recorder.Code, recorder.Body.String())
	}
	var got issueBumpChallengeResponse
	if err := json.NewDecoder(recorder.Body).Decode(&got); err != nil {
		t.Fatal(err)
	}
	if got.Nonce != "nonce-1" || got.SlotID != "slot-1" {
		t.Fatalf("unexpected response: %#v", got)
	}
}

func TestIssueBumpChallengeFailsClosedWhenServiceUnavailable(t *testing.T) {
	server := &Server{}
	recorder := httptest.NewRecorder()
	server.issueBumpChallenge(recorder, requestWithBumpAuth(http.MethodPost, "/v1/slots/slot-1/bump/challenge", ""))
	if recorder.Code != http.StatusServiceUnavailable {
		t.Fatalf("status=%d body=%s", recorder.Code, recorder.Body.String())
	}
}

func TestIssueBumpChallengeMapsNotEligible(t *testing.T) {
	store := &fakeBumpStore{challengeErr: bump.ErrNotEligible}
	svc, err := bump.NewService(store, time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	server := &Server{deps: Dependencies{Bump: svc}}
	recorder := httptest.NewRecorder()
	server.issueBumpChallenge(recorder, requestWithBumpAuth(http.MethodPost, "/v1/slots/slot-1/bump/challenge", ""))
	if recorder.Code != http.StatusConflict {
		t.Fatalf("status=%d body=%s", recorder.Code, recorder.Body.String())
	}
	var body problem
	if err := json.NewDecoder(recorder.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	if body.Code != "not_eligible" {
		t.Fatalf("code=%q", body.Code)
	}
}

func TestConfirmBumpReturnsVerifiedResult(t *testing.T) {
	confirmedAt := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	store := &fakeBumpStore{confirmResult: bump.ConfirmResult{Verified: true, ConfirmedAt: confirmedAt}}
	svc, err := bump.NewService(store, time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	server := &Server{deps: Dependencies{Bump: svc}}
	body := `{"counterpartUserId":"22222222-2222-2222-2222-222222222222","nonce":"nonce-1"}`
	recorder := httptest.NewRecorder()
	server.confirmBump(recorder, requestWithBumpAuth(http.MethodPost, "/v1/slots/slot-1/bump/confirm", body))
	if recorder.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", recorder.Code, recorder.Body.String())
	}
	var got confirmBumpResponse
	if err := json.NewDecoder(recorder.Body).Decode(&got); err != nil {
		t.Fatal(err)
	}
	if !got.Verified || got.ConfirmedAt == "" {
		t.Fatalf("unexpected response: %#v", got)
	}
	want := [4]string{"11111111-1111-1111-1111-111111111111", "slot-1", "22222222-2222-2222-2222-222222222222", "nonce-1"}
	if store.lastConfirmArgs != want {
		t.Fatalf("args=%#v, want %#v", store.lastConfirmArgs, want)
	}
}

func TestConfirmBumpUnverifiedOmitsConfirmedAt(t *testing.T) {
	store := &fakeBumpStore{confirmResult: bump.ConfirmResult{Verified: false}}
	svc, err := bump.NewService(store, time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	server := &Server{deps: Dependencies{Bump: svc}}
	body := `{"counterpartUserId":"22222222-2222-2222-2222-222222222222","nonce":"nonce-1"}`
	recorder := httptest.NewRecorder()
	server.confirmBump(recorder, requestWithBumpAuth(http.MethodPost, "/v1/slots/slot-1/bump/confirm", body))
	if recorder.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", recorder.Code, recorder.Body.String())
	}
	var raw map[string]any
	if err := json.NewDecoder(recorder.Body).Decode(&raw); err != nil {
		t.Fatal(err)
	}
	if raw["verified"] != false {
		t.Fatalf("expected verified=false, got %#v", raw)
	}
	if _, present := raw["confirmedAt"]; present {
		t.Fatalf("confirmedAt must be omitted when not verified, got %#v", raw)
	}
}

func TestConfirmBumpRejectsInvalidJSON(t *testing.T) {
	store := &fakeBumpStore{}
	svc, err := bump.NewService(store, time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	server := &Server{deps: Dependencies{Bump: svc}}
	recorder := httptest.NewRecorder()
	server.confirmBump(recorder, requestWithBumpAuth(http.MethodPost, "/v1/slots/slot-1/bump/confirm", "{not json"))
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("status=%d body=%s", recorder.Code, recorder.Body.String())
	}
}

func TestConfirmBumpMapsInvalidChallenge(t *testing.T) {
	store := &fakeBumpStore{confirmErr: bump.ErrInvalidChallenge}
	svc, err := bump.NewService(store, time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	server := &Server{deps: Dependencies{Bump: svc}}
	body := `{"counterpartUserId":"22222222-2222-2222-2222-222222222222","nonce":"nonce-1"}`
	recorder := httptest.NewRecorder()
	server.confirmBump(recorder, requestWithBumpAuth(http.MethodPost, "/v1/slots/slot-1/bump/confirm", body))
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("status=%d body=%s", recorder.Code, recorder.Body.String())
	}
	var got problem
	if err := json.NewDecoder(recorder.Body).Decode(&got); err != nil {
		t.Fatal(err)
	}
	if got.Code != "invalid_challenge" {
		t.Fatalf("code=%q", got.Code)
	}
}

func TestConfirmBumpMapsForbidden(t *testing.T) {
	store := &fakeBumpStore{confirmErr: bump.ErrForbidden}
	svc, err := bump.NewService(store, time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	server := &Server{deps: Dependencies{Bump: svc}}
	body := `{"counterpartUserId":"22222222-2222-2222-2222-222222222222","nonce":"nonce-1"}`
	recorder := httptest.NewRecorder()
	server.confirmBump(recorder, requestWithBumpAuth(http.MethodPost, "/v1/slots/slot-1/bump/confirm", body))
	if recorder.Code != http.StatusForbidden {
		t.Fatalf("status=%d body=%s", recorder.Code, recorder.Body.String())
	}
}

func TestGetReliabilityReturnsOwnBand(t *testing.T) {
	store := &fakeBumpStore{reliability: bump.Reliability{UserID: "11111111-1111-1111-1111-111111111111", VerifiedBumpCount: 3, Band: bump.BandReliable}}
	svc, err := bump.NewService(store, time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	server := &Server{deps: Dependencies{Bump: svc}}
	recorder := httptest.NewRecorder()
	server.getReliability(recorder, requestWithBumpAuth(http.MethodGet, "/v1/me/reliability", ""))
	if recorder.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", recorder.Code, recorder.Body.String())
	}
	var got bump.Reliability
	if err := json.NewDecoder(recorder.Body).Decode(&got); err != nil {
		t.Fatal(err)
	}
	if got.VerifiedBumpCount != 3 || got.Band != bump.BandReliable {
		t.Fatalf("unexpected response: %#v", got)
	}
}

func TestGetReliabilityFailsClosedWhenServiceUnavailable(t *testing.T) {
	server := &Server{}
	recorder := httptest.NewRecorder()
	server.getReliability(recorder, requestWithBumpAuth(http.MethodGet, "/v1/me/reliability", ""))
	if recorder.Code != http.StatusServiceUnavailable {
		t.Fatalf("status=%d body=%s", recorder.Code, recorder.Body.String())
	}
}

func TestGetBumpVaultReturnsEntries(t *testing.T) {
	confirmedAt := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	store := &fakeBumpStore{vault: []bump.VaultEntry{
		{SlotID: "slot-1", SlotTitle: "Coffee", CounterpartID: "user-2", ConfirmedAt: confirmedAt},
	}}
	svc, err := bump.NewService(store, time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	server := &Server{deps: Dependencies{Bump: svc}}
	recorder := httptest.NewRecorder()
	server.getBumpVault(recorder, requestWithBumpAuth(http.MethodGet, "/v1/me/bump-vault", ""))
	if recorder.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", recorder.Code, recorder.Body.String())
	}
	var got []bumpVaultEntryResponse
	if err := json.NewDecoder(recorder.Body).Decode(&got); err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].SlotID != "slot-1" || got[0].CounterpartID != "user-2" {
		t.Fatalf("unexpected response: %#v", got)
	}
}

func TestGetBumpVaultPropagatesStoreError(t *testing.T) {
	store := &fakeBumpStore{vaultErr: errors.New("db down")}
	svc, err := bump.NewService(store, time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	server := &Server{deps: Dependencies{Bump: svc}}
	recorder := httptest.NewRecorder()
	server.getBumpVault(recorder, requestWithBumpAuth(http.MethodGet, "/v1/me/bump-vault", ""))
	if recorder.Code != http.StatusInternalServerError {
		t.Fatalf("status=%d body=%s", recorder.Code, recorder.Body.String())
	}
}

func requestWithBandAuth(targetUserID string) *http.Request {
	request := httptest.NewRequest(http.MethodGet, "/v1/users/"+targetUserID+"/reliability-band", nil)
	request.SetPathValue("userID", targetUserID)
	ctx := context.WithValue(request.Context(), authKey, authContext{User: account.User{ID: "11111111-1111-1111-1111-111111111111"}})
	return request.WithContext(ctx)
}

// TestGetUserReliabilityBandReturnsOnlyBand closes README §6.9's
// "private/public reliability bands" at the API surface: unlike
// getReliability (self-only, exact VerifiedBumpCount included), this
// endpoint answers for ANY user and must never leak the exact count.
func TestGetUserReliabilityBandReturnsOnlyBand(t *testing.T) {
	store := &fakeBumpStore{publicBand: bump.BandTrusted}
	svc, err := bump.NewService(store, time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	server := &Server{deps: Dependencies{Bump: svc}}
	recorder := httptest.NewRecorder()
	server.getUserReliabilityBand(recorder, requestWithBandAuth("22222222-2222-2222-2222-222222222222"))
	if recorder.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", recorder.Code, recorder.Body.String())
	}
	if store.lastBandArgs[0] != "11111111-1111-1111-1111-111111111111" || store.lastBandArgs[1] != "22222222-2222-2222-2222-222222222222" {
		t.Fatalf("expected (viewer,target) forwarded to the store, got %#v", store.lastBandArgs)
	}
	body := recorder.Body.String()
	if !bytes.Contains([]byte(body), []byte(`"band":"TRUSTED"`)) {
		t.Fatalf("expected band in response, got %s", body)
	}
	if bytes.Contains([]byte(body), []byte("verifiedBumpCount")) {
		t.Fatalf("public band response must never include the exact count: %s", body)
	}
}

func TestGetUserReliabilityBandRequiresAuth(t *testing.T) {
	server := &Server{}
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/v1/users/target/reliability-band", nil)
	request.SetPathValue("userID", "target")
	server.getUserReliabilityBand(recorder, request)
	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("status=%d body=%s", recorder.Code, recorder.Body.String())
	}
}

func TestGetUserReliabilityBandFailsClosedWhenServiceUnavailable(t *testing.T) {
	server := &Server{}
	recorder := httptest.NewRecorder()
	server.getUserReliabilityBand(recorder, requestWithBandAuth("target"))
	if recorder.Code != http.StatusServiceUnavailable {
		t.Fatalf("status=%d body=%s", recorder.Code, recorder.Body.String())
	}
}

func TestGetUserReliabilityBandPropagatesForbiddenForBlockedPair(t *testing.T) {
	store := &fakeBumpStore{publicBandErr: bump.ErrForbidden}
	svc, err := bump.NewService(store, time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	server := &Server{deps: Dependencies{Bump: svc}}
	recorder := httptest.NewRecorder()
	server.getUserReliabilityBand(recorder, requestWithBandAuth("blocked-user"))
	if recorder.Code != http.StatusForbidden {
		t.Fatalf("status=%d body=%s", recorder.Code, recorder.Body.String())
	}
}
