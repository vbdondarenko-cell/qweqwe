package slot

import (
    "context"
    "errors"
    "testing"
)

func TestV1CapacityCeiling(t *testing.T) {
    svc, err := NewService(&memoryStore{})
    if err != nil {
        t.Fatal(err)
    }

    if _, err := svc.Create(context.Background(), "host", CreateInput{
        Title: "Too large",
        Activity: "social",
        PlaceText: "Kyiv",
        Capacity: MaxV1Capacity + 1,
    }, "capacity-create-0001"); !errors.Is(err, ErrInvalidInput) {
        t.Fatalf("create capacity %d accepted: %v", MaxV1Capacity+1, err)
    }

    store := &memoryStore{created: Slot{
        ID: "slot-id",
        Organizer: Organizer{ID: "host"},
        Title: "Current",
        Activity: "social",
        PlaceText: "Kyiv",
        Capacity: MaxV1Capacity,
        Version: 1,
        State: StateFilling,
    }}
    svc, err = NewService(store)
    if err != nil {
        t.Fatal(err)
    }
    tooLarge := MaxV1Capacity + 1
    if _, err := svc.Edit(context.Background(), "host", "slot-id", EditInput{
        ExpectedVersion: 1,
        Capacity: &tooLarge,
    }, "capacity-edit-00001"); !errors.Is(err, ErrInvalidInput) {
        t.Fatalf("edit capacity %d accepted: %v", tooLarge, err)
    }
}

func TestV1CapacityCeilingAllowsBoundary(t *testing.T) {
    svc, err := NewService(&memoryStore{})
    if err != nil {
        t.Fatal(err)
    }
    if _, err := svc.Create(context.Background(), "host", CreateInput{
        Title: "Community table",
        Activity: "social",
        PlaceText: "Kyiv",
        Capacity: MaxV1Capacity,
    }, "capacity-boundary-01"); err != nil {
        t.Fatalf("boundary capacity rejected: %v", err)
    }
}
