package postgres

import (
	"testing"

	"github.com/vbdondarenko-cell/qweqwe/backend/internal/realtime"
)

func TestFinalizeViewerBatchAdvancesAcrossInvisibleTail(t *testing.T) {
	events := []realtime.Event{{Sequence: 11}, {Sequence: 17}}
	batch := finalizeViewerBatch(25, events, 10)
	if batch.Cursor != 25 || len(batch.Events) != 2 {
		t.Fatalf("cursor=%d events=%d", batch.Cursor, len(batch.Events))
	}
}

func TestFinalizeViewerBatchStopsBeforeUnreturnedVisibleEvent(t *testing.T) {
	events := []realtime.Event{
		{Sequence: 11},
		{Sequence: 17},
		{Sequence: 23},
	}
	batch := finalizeViewerBatch(40, events, 2)
	if batch.Cursor != 17 {
		t.Fatalf("cursor=%d want=17", batch.Cursor)
	}
	if len(batch.Events) != 2 || batch.Events[1].Sequence != 17 {
		t.Fatalf("unexpected events: %#v", batch.Events)
	}
}

func TestFinalizeViewerBatchNoVisibleEvents(t *testing.T) {
	batch := finalizeViewerBatch(55, nil, 10)
	if batch.Cursor != 55 || len(batch.Events) != 0 {
		t.Fatalf("cursor=%d events=%d", batch.Cursor, len(batch.Events))
	}
}
