package blocklist

import (
	"context"
	"errors"
	"testing"
	"time"
)

type memoryStore struct { items map[string]UserSummary }
func (m *memoryStore) Block(_ context.Context,_ string,blockedID string,_ time.Time)error{if m.items==nil{m.items=map[string]UserSummary{}};m.items[blockedID]=UserSummary{ID:blockedID,Username:"user",DisplayName:"User"};return nil}
func (m *memoryStore) Unblock(_ context.Context,_ string,blockedID string)error{delete(m.items,blockedID);return nil}
func (m *memoryStore) List(_ context.Context,_ string,_ int)([]UserSummary,error){out:=make([]UserSummary,0,len(m.items));for _,v:=range m.items{out=append(out,v)};return out,nil}

func TestBlockListAndUnblock(t *testing.T){
	store:=&memoryStore{};svc,err:=NewService(store);if err!=nil{t.Fatal(err)}
	if err:=svc.Block(context.Background(),"me","other");err!=nil{t.Fatal(err)}
	items,err:=svc.List(context.Background(),"me");if err!=nil{t.Fatal(err)};if len(items)!=1||items[0].ID!="other"{t.Fatalf("items=%#v",items)}
	if err:=svc.Unblock(context.Background(),"me","other");err!=nil{t.Fatal(err)}
	items,err=svc.List(context.Background(),"me");if err!=nil{t.Fatal(err)};if len(items)!=0{t.Fatalf("items=%#v",items)}
}

func TestCannotBlockSelf(t *testing.T){
	svc,_:=NewService(&memoryStore{})
	if err:=svc.Block(context.Background(),"same","same");!errors.Is(err,ErrInvalidTarget){t.Fatalf("got %v",err)}
}
