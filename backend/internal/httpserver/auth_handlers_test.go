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
	"github.com/vbdondarenko-cell/qweqwe/backend/internal/password"
	"github.com/vbdondarenko-cell/qweqwe/backend/internal/ratelimit"
)

type authTestStore struct {
	user account.UserWithPassword
	session account.Session
	revoked bool
	reset account.PasswordReset
}

func (s *authTestStore) Register(_ context.Context, u account.User, hash string, sess account.Session) error {
	if s.user.ID != "" { return account.ErrConflict }
	s.user = account.UserWithPassword{User: u, PasswordHash: hash}
	s.session = sess
	return nil
}
func (s *authTestStore) FindByLogin(_ context.Context, identifier string) (account.UserWithPassword, error) {
	if identifier != s.user.Email && identifier != s.user.Username { return account.UserWithPassword{}, account.ErrNotFound }
	return s.user, nil
}
func (s *authTestStore) CreateSession(_ context.Context, sess account.Session) error { s.session=sess; s.revoked=false; return nil }
func (s *authTestStore) Authenticate(_ context.Context, digest []byte, now time.Time) (account.User,string,error) {
	if s.revoked || string(digest)!=string(s.session.TokenHash) || !s.session.ExpiresAt.After(now) { return account.User{},"",account.ErrUnauthorized }
	return s.user.User,s.session.ID,nil
}
func (s *authTestStore) RevokeSession(_ context.Context, digest []byte, _ time.Time) error {
	if string(digest)!=string(s.session.TokenHash) { return account.ErrNotFound }
	s.revoked=true
	return nil
}
func (s *authTestStore) UpdateProfile(_ context.Context, id string, patch account.ProfilePatch, now time.Time) (account.User,error) {
	if id!=s.user.ID { return account.User{},account.ErrNotFound }
	if patch.DisplayName!=nil { s.user.DisplayName=*patch.DisplayName }
	if patch.AvatarURL!=nil { if *patch.AvatarURL=="" { s.user.AvatarURL=nil } else { s.user.AvatarURL=patch.AvatarURL } }
	if patch.ProfileVisibility!=nil { s.user.ProfileVisibility=*patch.ProfileVisibility }
	if patch.Language!=nil { s.user.Language=*patch.Language }
	s.user.UpdatedAt=now
	return s.user.User,nil
}
func (s *authTestStore) CreatePasswordReset(_ context.Context, reset account.PasswordReset, _ time.Time) error { s.reset=reset; return nil }
func (s *authTestStore) ResetPassword(_ context.Context, digest []byte, hash string, now time.Time) error {
	if string(digest)!=string(s.reset.TokenHash) || !s.reset.ExpiresAt.After(now) { return account.ErrUnauthorized }
	s.user.PasswordHash=hash
	s.revoked=true
	return nil
}

func TestRegisterMeLogoutFlow(t *testing.T) {
	store:=&authTestStore{}
	accounts,err:=account.NewService(store,password.OWASPMinimum(),time.Hour)
	if err!=nil { t.Fatal(err) }
	server:=New(Dependencies{Accounts:accounts,Ready:func(context.Context)error{return nil}})

	body:=[]byte(`{"email":"a@example.com","username":"alice","displayName":"Alice","password":"correct horse battery staple","language":"uk"}`)
	reg:=httptest.NewRecorder()
	server.Handler().ServeHTTP(reg,httptest.NewRequest(http.MethodPost,"/v1/auth/register",bytes.NewReader(body)))
	if reg.Code!=http.StatusCreated { t.Fatalf("register status=%d body=%s",reg.Code,reg.Body.String()) }
	var auth account.AuthResult
	if err:=json.NewDecoder(reg.Body).Decode(&auth);err!=nil{t.Fatal(err)}
	if auth.Token==""{t.Fatal("missing bearer token")}

	meReq:=httptest.NewRequest(http.MethodGet,"/v1/me",nil)
	meReq.Header.Set("Authorization","Bearer "+auth.Token)
	me:=httptest.NewRecorder(); server.Handler().ServeHTTP(me,meReq)
	if me.Code!=http.StatusOK{t.Fatalf("me status=%d body=%s",me.Code,me.Body.String())}

	logoutReq:=httptest.NewRequest(http.MethodPost,"/v1/auth/logout",nil)
	logoutReq.Header.Set("Authorization","Bearer "+auth.Token)
	logout:=httptest.NewRecorder(); server.Handler().ServeHTTP(logout,logoutReq)
	if logout.Code!=http.StatusNoContent{t.Fatalf("logout status=%d",logout.Code)}

	afterReq:=httptest.NewRequest(http.MethodGet,"/v1/me",nil)
	afterReq.Header.Set("Authorization","Bearer "+auth.Token)
	after:=httptest.NewRecorder(); server.Handler().ServeHTTP(after,afterReq)
	if after.Code!=http.StatusUnauthorized{t.Fatalf("revoked token status=%d",after.Code)}
}

func TestAuthRateLimitReturns429AndRetryAfter(t *testing.T) {
	limiter,err:=ratelimit.New(ratelimit.Config{Limit:1,Window:time.Minute,IdleTTL:time.Minute,MaxEntries:100})
	if err!=nil{t.Fatal(err)}
	server:=New(Dependencies{AuthLimiter:limiter})
	body:=[]byte(`{"identifier":"nobody","password":"password"}`)
	first:=httptest.NewRecorder(); req1:=httptest.NewRequest(http.MethodPost,"/v1/auth/login",bytes.NewReader(body)); server.Handler().ServeHTTP(first,req1)
	second:=httptest.NewRecorder(); req2:=httptest.NewRequest(http.MethodPost,"/v1/auth/login",bytes.NewReader(body)); server.Handler().ServeHTTP(second,req2)
	if second.Code!=http.StatusTooManyRequests{t.Fatalf("status=%d body=%s",second.Code,second.Body.String())}
	if second.Header().Get("Retry-After")==""{t.Fatal("missing Retry-After")}
}

func TestRecoveryFailsClosedWhenDeliveryIsNotConfigured(t *testing.T) {
	store:=&authTestStore{}
	accounts,err:=account.NewService(store,password.OWASPMinimum(),time.Hour);if err!=nil{t.Fatal(err)}
	server:=New(Dependencies{Accounts:accounts})
	rec:=httptest.NewRecorder(); req:=httptest.NewRequest(http.MethodPost,"/v1/auth/recovery/request",bytes.NewBufferString(`{"email":"a@example.com"}`)); server.Handler().ServeHTTP(rec,req)
	if rec.Code!=http.StatusServiceUnavailable{t.Fatalf("status=%d body=%s",rec.Code,rec.Body.String())}
}
