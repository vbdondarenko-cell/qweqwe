package realtime

import (
	"encoding/json"
	"testing"
	"time"
)

func TestEventAndOutcomeValidation(t *testing.T) {
	event := Event{
		Sequence: 1,
		ID: "event-id",
		Type: "slot.created",
		AggregateType: "slot",
		AggregateID: "slot-id",
		Payload: json.RawMessage(`{"state":"FILLING"}`),
		OccurredAt: time.Now().UTC(),
	}
	if !event.Valid() {
		t.Fatal("valid canonical event rejected")
	}
	broken := event
	broken.Payload = json.RawMessage(`{"state":`)
	if broken.Valid() {
		t.Fatal("invalid JSON payload accepted")
	}
	if !OutcomeDelivered.Valid() || !OutcomeSkipped.Valid() {
		t.Fatal("canonical delivery outcomes rejected")
	}
	if DeliveryOutcome("FAILED").Valid() {
		t.Fatal("non-terminal connector outcome accepted")
	}
}
