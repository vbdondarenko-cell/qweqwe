package httpserver

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/vbdondarenko-cell/qweqwe/backend/internal/account"
	"github.com/vbdondarenko-cell/qweqwe/backend/internal/monetization"
)

type fakeMonetizationStore struct {
	status      monetization.StoreStatus
	bindErr     error
	rewardedErr error
	purchaseErr error
}

func (f *fakeMonetizationStore) Status(context.Context, string) (monetization.StoreStatus, error) {
	return f.status, nil
}
func (f *fakeMonetizationStore) EnsureReferralCode(_ context.Context, _ string, code string) (string, error) {
	return code, nil
}
func (f *fakeMonetizationStore) BindReferral(context.Context, string, string, int) error {
	return f.bindErr
}
func (f *fakeMonetizationStore) RecordRewardedView(context.Context, string, string, []byte, time.Time, time.Duration, time.Duration, time.Duration, time.Duration) error {
	return f.rewardedErr
}
func (f *fakeMonetizationStore) RecordPurchase(context.Context, string, string, string, []byte, time.Time, time.Time, string, time.Time, []monetization.ReferralMilestone) error {
	return f.purchaseErr
}

type fakeMonetizationRewardedVerifier struct {
	provider, viewID string
	err              error
}

func (f *fakeMonetizationRewardedVerifier) VerifyRewardedView(context.Context, string, string) (string, string, error) {
	return f.provider, f.viewID, f.err
}

type fakeMonetizationPurchaseVerifier struct {
	result monetization.VerifiedPurchase
	err    error
}

func (f *fakeMonetizationPurchaseVerifier) VerifyPurchase(context.Context, string, string) (monetization.VerifiedPurchase, error) {
	return f.result, f.err
}

func requestWithMonetizationAuth(method, target, body string) *http.Request {
	var reader *bytes.Reader
	if body != "" {
		reader = bytes.NewReader([]byte(body))
	} else {
		reader = bytes.NewReader(nil)
	}
	request := httptest.NewRequest(method, target, reader)
	ctx := context.WithValue(request.Context(), authKey, authContext{User: account.User{ID: "11111111-1111-1111-1111-111111111111"}})
	return request.WithContext(ctx)
}

func TestGetMonetizationReturnsSnapshot(t *testing.T) {
	svc := monetization.NewService(&fakeMonetizationStore{})
	server := &Server{deps: Dependencies{Monetization: svc}}
	recorder := httptest.NewRecorder()
	server.getMonetization(recorder, requestWithMonetizationAuth(http.MethodGet, "/v1/me/monetization", ""))
	if recorder.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", recorder.Code, recorder.Body.String())
	}
	var got monetization.Snapshot
	if err := json.NewDecoder(recorder.Body).Decode(&got); err != nil {
		t.Fatal(err)
	}
	if len(got.Catalog.Plans) != 4 {
		t.Fatalf("plans=%d", len(got.Catalog.Plans))
	}
}

func TestGetMonetizationFailsClosedWhenServiceUnavailable(t *testing.T) {
	server := &Server{}
	recorder := httptest.NewRecorder()
	server.getMonetization(recorder, requestWithMonetizationAuth(http.MethodGet, "/v1/me/monetization", ""))
	if recorder.Code != http.StatusServiceUnavailable {
		t.Fatalf("status=%d body=%s", recorder.Code, recorder.Body.String())
	}
}

func TestSubmitRewardedViewHandlerRejectsWhenCapabilityDisabled(t *testing.T) {
	svc := monetization.NewService(&fakeMonetizationStore{})
	server := &Server{deps: Dependencies{Monetization: svc}}
	recorder := httptest.NewRecorder()
	server.submitRewardedView(recorder, requestWithMonetizationAuth(http.MethodPost, "/v1/me/monetization/rewarded-views", `{"receipt":"raw"}`))
	if recorder.Code != http.StatusServiceUnavailable {
		t.Fatalf("status=%d body=%s", recorder.Code, recorder.Body.String())
	}
}

func TestSubmitRewardedViewHandlerReturnsSnapshotOnSuccess(t *testing.T) {
	svc := monetization.NewService(&fakeMonetizationStore{})
	svc.ConfigureRewarded(&fakeMonetizationRewardedVerifier{provider: "admob", viewID: "view-1"})
	server := &Server{deps: Dependencies{Monetization: svc}}
	recorder := httptest.NewRecorder()
	server.submitRewardedView(recorder, requestWithMonetizationAuth(http.MethodPost, "/v1/me/monetization/rewarded-views", `{"receipt":"raw"}`))
	if recorder.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", recorder.Code, recorder.Body.String())
	}
}

func TestSubmitRewardedViewHandlerMapsCooldownConflict(t *testing.T) {
	svc := monetization.NewService(&fakeMonetizationStore{rewardedErr: monetization.ErrRewardedCooldownActive})
	svc.ConfigureRewarded(&fakeMonetizationRewardedVerifier{provider: "admob", viewID: "view-1"})
	server := &Server{deps: Dependencies{Monetization: svc}}
	recorder := httptest.NewRecorder()
	server.submitRewardedView(recorder, requestWithMonetizationAuth(http.MethodPost, "/v1/me/monetization/rewarded-views", `{"receipt":"raw"}`))
	if recorder.Code != http.StatusConflict {
		t.Fatalf("status=%d body=%s", recorder.Code, recorder.Body.String())
	}
}

func TestSubmitRewardedViewHandlerRejectsInvalidJSON(t *testing.T) {
	svc := monetization.NewService(&fakeMonetizationStore{})
	server := &Server{deps: Dependencies{Monetization: svc}}
	recorder := httptest.NewRecorder()
	server.submitRewardedView(recorder, requestWithMonetizationAuth(http.MethodPost, "/v1/me/monetization/rewarded-views", `not-json`))
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("status=%d body=%s", recorder.Code, recorder.Body.String())
	}
}

func TestVerifyPurchaseHandlerRejectsWhenCapabilityDisabled(t *testing.T) {
	svc := monetization.NewService(&fakeMonetizationStore{})
	server := &Server{deps: Dependencies{Monetization: svc}}
	recorder := httptest.NewRecorder()
	server.verifyPurchase(recorder, requestWithMonetizationAuth(http.MethodPost, "/v1/me/monetization/purchases", `{"purchaseToken":"raw"}`))
	if recorder.Code != http.StatusServiceUnavailable {
		t.Fatalf("status=%d body=%s", recorder.Code, recorder.Body.String())
	}
}

func TestVerifyPurchaseHandlerReturnsSnapshotOnSuccess(t *testing.T) {
	svc := monetization.NewService(&fakeMonetizationStore{})
	svc.ConfigurePurchases(&fakeMonetizationPurchaseVerifier{result: monetization.VerifiedPurchase{
		Provider: "play", ProductID: "linkup_plus_monthly", PurchaseID: "purchase-1",
		State: "ACTIVE", PeriodStart: time.Now(), PeriodEnd: time.Now().Add(30 * 24 * time.Hour),
	}})
	server := &Server{deps: Dependencies{Monetization: svc}}
	recorder := httptest.NewRecorder()
	server.verifyPurchase(recorder, requestWithMonetizationAuth(http.MethodPost, "/v1/me/monetization/purchases", `{"purchaseToken":"raw"}`))
	if recorder.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", recorder.Code, recorder.Body.String())
	}
}

func TestVerifyPurchaseHandlerMapsInvalidPurchase(t *testing.T) {
	svc := monetization.NewService(&fakeMonetizationStore{})
	svc.ConfigurePurchases(&fakeMonetizationPurchaseVerifier{result: monetization.VerifiedPurchase{
		Provider: "play", ProductID: "p", PurchaseID: "id", State: "NOT_A_REAL_STATE",
		PeriodStart: time.Now(), PeriodEnd: time.Now().Add(time.Hour),
	}})
	server := &Server{deps: Dependencies{Monetization: svc}}
	recorder := httptest.NewRecorder()
	server.verifyPurchase(recorder, requestWithMonetizationAuth(http.MethodPost, "/v1/me/monetization/purchases", `{"purchaseToken":"raw"}`))
	if recorder.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status=%d body=%s", recorder.Code, recorder.Body.String())
	}
}

func TestBindReferralRequiresAuth(t *testing.T) {
	svc := monetization.NewService(&fakeMonetizationStore{})
	server := &Server{deps: Dependencies{Monetization: svc}}
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPut, "/v1/me/referral", bytes.NewReader([]byte(`{"code":"AB12CD"}`)))
	server.bindReferral(recorder, request)
	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("status=%d body=%s", recorder.Code, recorder.Body.String())
	}
}
