package account

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/vbdondarenko-cell/qweqwe/backend/internal/password"
)

type memoryStore struct { user UserWithPassword; session Session; revoked bool }
func (m *memoryStore) Register(_ context.Context,u User,h string,s Session)error{ if m.user.ID!=""{return ErrConflict}; m.user=UserWithPassword{User:u,PasswordHash:h};m.session=s;return nil }
func (m *memoryStore) FindByLogin(_ context.Context,id string)(UserWithPassword,error){ if m.user.ID=="" || (id!=m.user.Email && id!=m.user.Username){return UserWithPassword{},ErrNotFound};return m.user,nil }
func (m *memoryStore) CreateSession(_ context.Context,s Session)error{m.session=s;m.revoked=false;return nil}
func (m *memoryStore) Authenticate(_ context.Context,h []byte,now time.Time)(User,string,error){ if m.revoked || !m.session.ExpiresAt.After(now) || string(h)!=string(m.session.TokenHash){return User,"",ErrUnauthorized};return m.user.User,m.session.ID,nil }
func (m *memoryStore) RevokeSession(_ context.Context,h []byte,_ time.Time)error{if string(h)!=string(m.session.TokenHash){return ErrNotFound};m.revoked=true;return nil}
func (m *memoryStore) UpdateProfile(_ context.Context,id string,p ProfilePatch,now time.Time)(User,error){if id!=m.user.ID{return User{},ErrNotFound};if p.DisplayName!=nil{m.user.DisplayName=*p.DisplayName};if p.ProfileVisibility!=nil{m.user.ProfileVisibility=*p.ProfileVisibility};if p.Language!=nil{m.user.Language=*p.Language};m.user.UpdatedAt=now;return m.user.User,nil}

func TestRegisterAuthenticateLogout(t *testing.T){
	store:=&memoryStore{}; svc,err:=NewService(store,password.OWASPMinimum(),time.Hour);if err!=nil{t.Fatal(err)}
	out,err:=svc.Register(context.Background(),Registration{Email:"A@Example.com",Username:"Alice_1",DisplayName:"Alice",Password:"correct horse battery staple",Language:"uk"});if err!=nil{t.Fatal(err)}
	if out.User.Email!="a@example.com" || out.User.Username!="alice_1"{t.Fatalf("normalization failed: %#v",out.User)}
	if _,_,err:=svc.Authenticate(context.Background(),out.Token);err!=nil{t.Fatal(err)}
	if err:=svc.Logout(context.Background(),out.Token);err!=nil{t.Fatal(err)}
	if _,_,err:=svc.Authenticate(context.Background(),out.Token);!errors.Is(err,ErrUnauthorized){t.Fatalf("expected revoked session, got %v",err)}
}

func TestLoginRejectsWrongPassword(t *testing.T){
	store:=&memoryStore{};svc,_:=NewService(store,password.OWASPMinimum(),time.Hour);_,err:=svc.Register(context.Background(),Registration{Email:"a@example.com",Username:"alice",DisplayName:"Alice",Password:"correct horse battery staple"});if err!=nil{t.Fatal(err)}
	if _,err:=svc.Login(context.Background(),Login{Identifier:"alice",Password:"wrong password"});!errors.Is(err,ErrUnauthorized){t.Fatalf("expected unauthorized, got %v",err)}
}
