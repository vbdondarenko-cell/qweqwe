package realtime

import (
	"encoding/json"
	"errors"
	"time"
)

const MaxBatch = 500

type DeliveryOutcome string

const (
	OutcomeDelivered DeliveryOutcome = "DELIVERED"
	OutcomeSkipped   DeliveryOutcome = "SKIPPED"
)

var (
	ErrInvalidInput       = errors.New("invalid realtime input")
	ErrCursorOutOfOrder   = errors.New("realtime cursor out of order")
	ErrReceiptConflict    = errors.New("realtime delivery receipt conflict")
)

type Event struct {
	Sequence      int64           `json:"sequence"`
	ID            string          `json:"eventId"`
	Type          string          `json:"eventType"`
	AggregateType string          `json:"aggregateType"`
	AggregateID   string          `json:"aggregateId"`
	SubjectUserID *string         `json:"subjectUserId,omitempty"`
	SlotID        *string         `json:"slotId,omitempty"`
	Payload       json.RawMessage `json:"payload"`
	OccurredAt    time.Time       `json:"occurredAt"`
}

func (e Event) Valid() bool {
	return e.Sequence > 0 && e.ID != "" && e.Type != "" && e.AggregateType != "" &&
		e.AggregateID != "" && !e.OccurredAt.IsZero() && json.Valid(e.Payload)
}

func (o DeliveryOutcome) Valid() bool {
	return o == OutcomeDelivered || o == OutcomeSkipped
}
