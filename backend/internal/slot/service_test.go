package slot

import (
	"context"
	"errors"
	"testing"
	"time"
)

type memoryStore struct {
	created     Slot
	createKey   string
	createHash  []byte
	lastEdit    EditInput
	lastEditKey string
	pending     map[string]PendingRequest
	members     map[string]bool
}

func (m *memoryStore) ensure() {
	if m.pending == nil {
		m.pending = map[string]PendingRequest{}
	}
	if m.members == nil {
		m.members = map[string]bool{}
	}
}

func (m *memoryStore) Create(_ context.Context, actorID string, candidate Slot, key string, requestHash []byte) (Slot, error) {
	m.ensure()
	m.created = candidate
	m.created.Organizer = Organizer{ID: actorID, Username: "host", DisplayName: "Host"}
	m.created.ViewerState = ViewerHost
	m.createKey = key
	m.createHash = append([]byte(nil), requestHash...)
	return m.created, nil
}

func (m *memoryStore) Get(_ context.Context, actorID, slotID string) (Slot, error) {
	m.ensure()
	if m.created.ID != slotID {
		return Slot{}, ErrNotFound
	}
	out := m.created
	out.ViewerState = m.viewer(actorID)
	return out, nil
}

func (m *memoryStore) ListPulse(_ context.Context, actorID string, _ int) ([]Slot, error) {
	m.ensure()
	if m.created.ID == "" || m.created.State == StateCancelled || m.created.State == StateCompleted || m.created.State == StateActive {
		return []Slot{}, nil
	}
	out := m.created
	out.ViewerState = m.viewer(actorID)
	return []Slot{out}, nil
}

func (m *memoryStore) Edit(_ context.Context, actorID, slotID string, patch EditInput, key string, _ []byte, now time.Time) (Slot, error) {
	if m.created.ID != slotID {
		return Slot{}, ErrNotFound
	}
	if actorID != m.created.Organizer.ID {
		return Slot{}, ErrForbidden
	}
	m.lastEdit = patch
	m.lastEditKey = key
	if patch.ExpectedVersion != m.created.Version {
		return Slot{}, ErrConflict
	}
	if patch.Title != nil {
		m.created.Title = *patch.Title
	}
	if patch.Capacity != nil {
		if *patch.Capacity < m.created.AcceptedCount {
			return Slot{}, ErrInvalidInput
		}
		m.created.Capacity = *patch.Capacity
	}
	m.created.Version++
	m.created.UpdatedAt = now
	m.created.ViewerState = ViewerHost
	return m.created, nil
}

func (m *memoryStore) Cancel(_ context.Context, actorID, slotID string, expectedVersion int64, _ string, _ []byte, now time.Time) (Slot, error) {
	if m.created.ID != slotID {
		return Slot{}, ErrNotFound
	}
	if actorID != m.created.Organizer.ID {
		return Slot{}, ErrForbidden
	}
	if expectedVersion != m.created.Version {
		return Slot{}, ErrConflict
	}
	m.pending = map[string]PendingRequest{}
	m.created.State = StateCancelled
	m.created.Version++
	m.created.UpdatedAt = now
	m.created.ViewerState = ViewerHost
	return m.created, nil
}

func (m *memoryStore) Request(_ context.Context, actorID, slotID, _ string, _ []byte, now time.Time) (Slot, error) {
	m.ensure()
	if m.created.ID != slotID {
		return Slot{}, ErrNotFound
	}
	if actorID == m.created.Organizer.ID {
		return Slot{}, ErrForbidden
	}
	if m.members[actorID] {
		return Slot{}, ErrAlreadyMember
	}
	if _, ok := m.pending[actorID]; ok {
		return Slot{}, ErrDuplicateRequest
	}
	if m.created.AcceptedCount >= m.created.Capacity {
		return Slot{}, ErrCapacityFull
	}
	m.pending[actorID] = PendingRequest{User: Organizer{ID: actorID, Username: actorID, DisplayName: actorID}, RequestedAt: now}
	m.created.Version++
	out := m.created
	out.ViewerState = ViewerPending
	return out, nil
}

func (m *memoryStore) Leave(_ context.Context, actorID, slotID, _ string, _ []byte, now time.Time) (Slot, error) {
	m.ensure()
	if m.created.ID != slotID {
		return Slot{}, ErrNotFound
	}
	if _, ok := m.pending[actorID]; ok {
		delete(m.pending, actorID)
		m.created.Version++
		m.created.UpdatedAt = now
		out := m.created
		out.ViewerState = ViewerNone
		return out, nil
	}
	if !m.members[actorID] {
		return Slot{}, ErrNotFound
	}
	delete(m.members, actorID)
	m.created.AcceptedCount--
	if m.created.State == StateFull {
		m.created.State = StateFilling
	}
	m.created.Version++
	m.created.UpdatedAt = now
	out := m.created
	out.ViewerState = ViewerNone
	return out, nil
}

func (m *memoryStore) ListPending(_ context.Context, actorID, slotID string) ([]PendingRequest, error) {
	m.ensure()
	if m.created.ID != slotID {
		return nil, ErrNotFound
	}
	if actorID != m.created.Organizer.ID {
		return nil, ErrForbidden
	}
	out := make([]PendingRequest, 0, len(m.pending))
	for _, request := range m.pending {
		out = append(out, request)
	}
	return out, nil
}

func (m *memoryStore) Approve(_ context.Context, actorID, slotID, requesterID, _ string, _ []byte, now time.Time) (Slot, error) {
	m.ensure()
	if m.created.ID != slotID {
		return Slot{}, ErrNotFound
	}
	if actorID != m.created.Organizer.ID {
		return Slot{}, ErrForbidden
	}
	if m.created.AcceptedCount >= m.created.Capacity {
		return Slot{}, ErrCapacityFull
	}
	if _, ok := m.pending[requesterID]; !ok {
		return Slot{}, ErrRequestNotFound
	}
	delete(m.pending, requesterID)
	m.members[requesterID] = true
	m.created.AcceptedCount++
	m.created.State = StateFilling
	if m.created.AcceptedCount >= m.created.Capacity {
		m.created.State = StateFull
	}
	m.created.Version++
	m.created.UpdatedAt = now
	out := m.created
	out.ViewerState = ViewerHost
	return out, nil
}

func (m *memoryStore) Reject(_ context.Context, actorID, slotID, requesterID, _ string, _ []byte, now time.Time) (Slot, error) {
	m.ensure()
	if m.created.ID != slotID {
		return Slot{}, ErrNotFound
	}
	if actorID != m.created.Organizer.ID {
		return Slot{}, ErrForbidden
	}
	if _, ok := m.pending[requesterID]; !ok {
		return Slot{}, ErrRequestNotFound
	}
	delete(m.pending, requesterID)
	m.created.Version++
	m.created.UpdatedAt = now
	out := m.created
	out.ViewerState = ViewerHost
	return out, nil
}

func (m *memoryStore) Start(_ context.Context, actorID, slotID, _ string, _ []byte, now time.Time) (Slot, error) {
	if m.created.ID != slotID {
		return Slot{}, ErrNotFound
	}
	if actorID != m.created.Organizer.ID {
		return Slot{}, ErrForbidden
	}
	if m.created.AcceptedCount < 1 || (m.created.State != StateFilling && m.created.State != StateFull) {
		return Slot{}, ErrInvalidState
	}
	m.pending = map[string]PendingRequest{}
	m.created.State = StateActive
	m.created.Version++
	m.created.UpdatedAt = now
	out := m.created
	out.ViewerState = ViewerHost
	return out, nil
}

func (m *memoryStore) Complete(_ context.Context, actorID, slotID, _ string, _ []byte, now time.Time) (Slot, error) {
	if m.created.ID != slotID {
		return Slot{}, ErrNotFound
	}
	if actorID != m.created.Organizer.ID {
		return Slot{}, ErrForbidden
	}
	if m.created.State != StateActive {
		return Slot{}, ErrInvalidState
	}
	m.created.State = StateCompleted
	m.created.Version++
	m.created.UpdatedAt = now
	out := m.created
	out.ViewerState = ViewerHost
	return out, nil
}

func (m *memoryStore) viewer(actorID string) ViewerState {
	if actorID == m.created.Organizer.ID {
		return ViewerHost
	}
	if m.members[actorID] {
		return ViewerAccepted
	}
	if _, ok := m.pending[actorID]; ok {
		return ViewerPending
	}
	return ViewerNone
}

func TestCreateUsesFoundationPublicApprovalContract(t *testing.T) {
	store := &memoryStore{}
	svc, err := NewService(store)
	if err != nil {
		t.Fatal(err)
	}
	out, err := svc.Create(context.Background(), "host-id", CreateInput{
		Title: "  Morning Coffee  ", Activity: " COFFEE ", PlaceText: "  Podil  ", Capacity: 2,
	}, "create-slot-000001")
	if err != nil {
		t.Fatal(err)
	}
	if out.Title != "Morning Coffee" || out.Activity != "coffee" || out.PlaceText != "Podil" {
		t.Fatalf("normalization failed: %#v", out)
	}
	if out.State != StateFilling || out.AccessMode != AccessApproval || out.Visibility != VisibilityPublic || out.ViewerState != ViewerHost || out.Version != 1 {
		t.Fatalf("foundation contract mismatch: %#v", out)
	}
	if store.createKey != "create-slot-000001" || len(store.createHash) != 32 {
		t.Fatalf("idempotency data not forwarded: key=%q hashLen=%d", store.createKey, len(store.createHash))
	}
}

func TestCreateRejectsMissingIdempotencyKey(t *testing.T) {
	svc, _ := NewService(&memoryStore{})
	_, err := svc.Create(context.Background(), "host-id", CreateInput{Title: "Coffee", Activity: "coffee", PlaceText: "Podil", Capacity: 4}, "")
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected invalid input, got %v", err)
	}
}

func TestApprovalLastSeatAndLeaveReopens(t *testing.T) {
	store := &memoryStore{}
	svc, _ := NewService(store)
	created, err := svc.Create(context.Background(), "host", CreateInput{Title: "Coffee", Activity: "coffee", PlaceText: "Podil", Capacity: 2}, "approval-create-001")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Request(context.Background(), "u1", created.ID, "approval-request-u1"); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Request(context.Background(), "u2", created.ID, "approval-request-u2"); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Approve(context.Background(), "host", created.ID, "u1", "approval-approve-u1"); err != nil {
		t.Fatal(err)
	}
	full, err := svc.Approve(context.Background(), "host", created.ID, "u2", "approval-approve-u2")
	if err != nil {
		t.Fatal(err)
	}
	if full.AcceptedCount != 2 || full.State != StateFull {
		t.Fatalf("expected FULL 2/2, got %#v", full)
	}
	if _, err := svc.Request(context.Background(), "u3", created.ID, "approval-request-u3"); !errors.Is(err, ErrCapacityFull) {
		t.Fatalf("expected full capacity, got %v", err)
	}
	reopened, err := svc.Leave(context.Background(), "u2", created.ID, "approval-leave-u2")
	if err != nil {
		t.Fatal(err)
	}
	if reopened.AcceptedCount != 1 || reopened.State != StateFilling || reopened.ViewerState != ViewerNone {
		t.Fatalf("expected reopen after leave, got %#v", reopened)
	}
}

func TestPendingWithdrawAndHostLifecycle(t *testing.T) {
	store := &memoryStore{}
	svc, _ := NewService(store)
	created, err := svc.Create(context.Background(), "host", CreateInput{Title: "Run", Activity: "running", PlaceText: "Park", Capacity: 3}, "lifecycle-create-01")
	if err != nil {
		t.Fatal(err)
	}
	pending, err := svc.Request(context.Background(), "u1", created.ID, "lifecycle-request1")
	if err != nil || pending.ViewerState != ViewerPending {
		t.Fatalf("pending=%#v err=%v", pending, err)
	}
	withdrawn, err := svc.Leave(context.Background(), "u1", created.ID, "lifecycle-leave-01")
	if err != nil || withdrawn.ViewerState != ViewerNone {
		t.Fatalf("withdrawn=%#v err=%v", withdrawn, err)
	}
	if _, err := svc.Request(context.Background(), "u1", created.ID, "lifecycle-request2"); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Approve(context.Background(), "host", created.ID, "u1", "lifecycle-approve1"); err != nil {
		t.Fatal(err)
	}
	active, err := svc.Start(context.Background(), "host", created.ID, "lifecycle-start-01")
	if err != nil || active.State != StateActive {
		t.Fatalf("active=%#v err=%v", active, err)
	}
	completed, err := svc.Complete(context.Background(), "host", created.ID, "lifecycle-complete")
	if err != nil || completed.State != StateCompleted {
		t.Fatalf("completed=%#v err=%v", completed, err)
	}
}

func TestEditRequiresExpectedVersion(t *testing.T) {
	store := &memoryStore{created: Slot{ID: "slot-id", Organizer: Organizer{ID: "host-id"}, Version: 3, Capacity: 6}}
	svc, _ := NewService(store)
	title := "  Updated title  "
	capacity := 8
	out, err := svc.Edit(context.Background(), "host-id", "slot-id", EditInput{ExpectedVersion: 3, Title: &title, Capacity: &capacity}, "edit-slot-0000001")
	if err != nil {
		t.Fatal(err)
	}
	if out.Title != "Updated title" || out.Capacity != 8 || out.Version != 4 {
		t.Fatalf("unexpected edit result: %#v", out)
	}
	_, err = svc.Edit(context.Background(), "host-id", "slot-id", EditInput{Title: &title}, "edit-slot-0000002")
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected version validation, got %v", err)
	}
}

func (m *memoryStore) ListMine(_ context.Context, actorID, view string, _ int) ([]Slot, error) {
    m.ensure()
    out := m.created
    out.ViewerState = m.viewer(actorID)
    if out.ID == "" || (out.State != StateFilling && out.State != StateFull && out.State != StatePublished && out.State != StateActive) { return []Slot{}, nil }
    matches := view == "HOSTING" && out.ViewerState == ViewerHost || view == "JOINED" && out.ViewerState == ViewerAccepted || view == "REQUESTED" && out.ViewerState == ViewerPending && out.State != StateActive
    if !matches { return []Slot{}, nil }
    return []Slot{out}, nil
}

func TestListMineIncludesActiveRelationships(t *testing.T) {
    store := &memoryStore{created: Slot{ID:"slot", Organizer:Organizer{ID:"host"}, State:StateActive}, members:map[string]bool{"member":true}}
    svc, err := NewService(store)
    if err != nil { t.Fatal(err) }
    for _, tc := range []struct{ actor, view string; count int }{
        {"host","HOSTING",1}, {"member","JOINED",1}, {"stranger","JOINED",0},
        {"member","HOSTING",0}, {"host","JOINED",0},
    } {
        got, err := svc.ListMine(context.Background(),tc.actor,tc.view)
        if err != nil || len(got)!=tc.count { t.Fatalf("%s/%s count=%d err=%v",tc.actor,tc.view,len(got),err) }
    }
    if _, err := svc.ListMine(context.Background(),"host","ALL_USERS"); !errors.Is(err,ErrInvalidInput) { t.Fatal("invalid scope accepted") }
    if _, err := svc.ListMine(context.Background(),"","HOSTING"); !errors.Is(err,ErrInvalidInput) { t.Fatal("empty actor accepted") }
}

func (m *memoryStore) ListAccepted(_ context.Context, actorID, slotID string) ([]Organizer, error) {
    if m.created.ID != slotID || m.created.Organizer.ID != actorID { return nil, ErrNotFound }
    switch m.created.State {
    case StatePublished, StateFilling, StateFull, StateActive:
    default: return nil, ErrNotFound
    }
    out := make([]Organizer, 0)
    for id := range m.members { if id != actorID { out = append(out, Organizer{ID:id}) } }
    return out, nil
}

func TestAcceptedRosterRequiresCurrentHost(t *testing.T) {
    store := &memoryStore{created: Slot{ID:"slot", Organizer:Organizer{ID:"host"}, State:StateActive}, members:map[string]bool{"member":true}}
    svc, _ := NewService(store)
    items, err := svc.ListAccepted(context.Background(), "host", "slot")
    if err != nil || len(items) != 1 || items[0].ID != "member" { t.Fatalf("items=%v err=%v", items, err) }
    for _, actor := range []string{"member", "pending", "stranger"} {
        if _, err := svc.ListAccepted(context.Background(), actor, "slot"); !errors.Is(err, ErrNotFound) { t.Fatalf("actor=%s err=%v", actor, err) }
    }
    store.created.State = StateCompleted
    if _, err := svc.ListAccepted(context.Background(), "host", "slot"); !errors.Is(err, ErrNotFound) { t.Fatal("terminal roster exposed") }
    if _, err := svc.ListAccepted(context.Background(), "", "slot"); !errors.Is(err, ErrInvalidInput) { t.Fatal("empty actor accepted") }
}

func (m *memoryStore) RemoveMember(_ context.Context, actorID, slotID, memberID string, version int64, _ string, _ []byte, _ time.Time) (Slot, error) {
    if m.created.ID != slotID { return Slot{}, ErrNotFound }
    if m.created.Organizer.ID != actorID { return Slot{}, ErrForbidden }
    if m.created.Version != version { return Slot{}, ErrConflict }
    if !m.members[memberID] { return Slot{}, ErrNotFound }
    delete(m.members, memberID)
    m.created.AcceptedCount--
    m.created.Version++
    if m.created.State == StateFull { m.created.State = StateFilling }
    return m.created, nil
}

func TestRemoveMemberValidatesVersionAndIdentity(t *testing.T) {
    store := &memoryStore{created:Slot{ID:"link", Organizer:Organizer{ID:"host"}, Version:3, State:StateFull, AcceptedCount:1}, members:map[string]bool{"member":true}}
    svc, _ := NewService(store)
    for _, tc := range []struct { actor, member, key string; version int64 }{
        {"host","host","remove-member-001",3}, {"host","member","",3}, {"host","member","remove-member-001",0},
    } {
        if _, err := svc.RemoveMember(context.Background(),tc.actor,"link",tc.member,tc.version,tc.key); !errors.Is(err,ErrInvalidInput) { t.Fatalf("invalid input: %v",err) }
    }
    if _, err := svc.RemoveMember(context.Background(),"host","link","member",2,"remove-member-001"); !errors.Is(err,ErrConflict) { t.Fatal("stale version accepted") }
    out, err := svc.RemoveMember(context.Background(),"host","link","member",3,"remove-member-001")
    if err != nil || out.AcceptedCount != 0 || out.State != StateFilling || out.Version != 4 { t.Fatalf("out=%v err=%v",out,err) }
}
